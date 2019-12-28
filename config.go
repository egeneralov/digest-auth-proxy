package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"crypto/md5"
	"encoding/hex"
)

type Config struct {
	Realm string  `json:"realm" yaml:"realm"`
	Users []*User `json:"users" yaml:"users"`
}

type User struct {
	Name     string `json:"name" yaml:"name"`
	Password string `json:"password" yaml:"password"`
}

/*
func main() {
  c, e := ParseConfig("config.yaml")
  if e != nil {
    panic(e)
  }
	b, e := json.Marshal(c)
	if e != nil {
  	panic(e)
	}
	fmt.Println(string(b))
}
*/

func ParseConfig(filename string) (*Config, error) {
	c, e := parseConfigFile(filename)
	if e != nil {
		return c, e
	}

	config := &Config{
		Realm: c.Realm,
	}

	for _, user := range c.Users {
		h := md5.New()
		h.Write([]byte(fmt.Sprintf("%s:%s:%s", user.Name, c.Realm, user.Password)))
		secret := hex.EncodeToString(h.Sum(nil))
		//   	fmt.Println(c.Realm, user.Name, secret)
		u := &User{
			Name:     user.Name,
			Password: secret,
		}
		config.Users = append(config.Users, u)
		// fmt.Sprintf("%s:%s:%s", user.Name, c.Realm, hex.EncodeToString(h.Sum(nil)))
	}
	return config, nil
}

func parseConfigFile(filename string) (*Config, error) {
	var config *Config

	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(dat, &config)
	if err != nil {
		err = json.Unmarshal(dat, &config)
		if err != nil {
			return config, err
		}
	}
	return config, nil
}
