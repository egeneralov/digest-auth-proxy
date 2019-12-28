package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
  "os"
	log "github.com/sirupsen/logrus"
  auth "github.com/abbot/go-http-auth"
)


var (
  bind string
	host string
	configFilename string
	ProxyConfig *Config
	err error
	debug = false
)


func secret(user, realm string) string {
  if realm != ProxyConfig.Realm {
    return ""
  }
	for _, user := range ProxyConfig.Users {
  	return user.Password
	}
	return ""
}


func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func logRequest(code int, r *auth.AuthenticatedRequest, message string) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	log.Infoln(code, host, r.Method, r.URL, r.Referer(), r.UserAgent(), message)
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func getHost(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}

func reverseProxy(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	outReq := new(http.Request)
	outReq.Method = req.Method
	outReq.URL = &url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1
	outReq.Header = make(http.Header)
	outReq.Body = req.Body
	outReq.ContentLength = req.ContentLength
	outReq.Host = host

	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
	copyHeader(outReq.Header, req.Header)
	outReq.Header.Set("Host", host)
	outReq.Header.Set("Referer", host)
	outReq.Header.Set("Origin", host)

	resp, err := http.DefaultClient.Do(outReq)
	if err != nil {
  	logRequest(500, req, fmt.Sprintf("proxy error: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	reqHost := getHost(req.Host)
	for _, cookie := range cookies {
		cookie.Domain = reqHost
		resp.Header.Add("Set-Cookie", cookie.String())
	}

	for _, h := range hopHeaders {
		resp.Header.Del(h)
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		if u, err := url.Parse(loc); err == nil && (u.Host == host) {
			u.Scheme = "http"
			u.Host = req.Host
			resp.Header.Set("Location", u.String())
		}
	}
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
	logRequest(200, req, "")
}


func main() {
	flag.StringVar(&host, "host", "localhost:8545", "proxy to")
	flag.StringVar(&bind, "bind", "0.0.0.0:9001", "bind to")
	flag.BoolVar(&debug, "debug", false, "expose sensetive info")
	flag.StringVar(&configFilename, "config", "config.yaml", "config.[yaml|json]")
	flag.Parse()
	
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
  if debug {
  	log.SetLevel(log.DebugLevel)
  } else {
  	log.SetLevel(log.InfoLevel)
  }
  
  ProxyConfig, err = ParseConfig("config.yaml")
  if err != nil {
    log.Fatal(err)
  }
  if debug {
    log.WithFields(log.Fields{
      "config": ProxyConfig,
    }).Debug("loaded configuration")
  } else {
    log.Info("loaded configuration")
  }
  
	authenticator := auth.NewDigestAuthenticator(ProxyConfig.Realm, secret)
	http.HandleFunc("/", authenticator.Wrap(reverseProxy))
	log.Infof("listening on %s", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}

