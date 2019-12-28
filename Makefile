
dev:
	go run -v ttt.go

patch:
	patch -N -p0 vendor/github.com/abbot/go-http-auth/digest.go -i digest.go.patch

patch-reverse:
	patch -N -p0 vendor/github.com/abbot/go-http-auth/digest.go -i digest.go.patch
