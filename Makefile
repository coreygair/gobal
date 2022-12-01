.PHONY: go-balancer server demo

balancer:
	go build -o bin/gobal ./cmd/balancer

server:
	go build -o bin/server ./cmd/server

demo:
	CGO_ENABLED=0 go build -o demo/balancer/bin/balancer -a -installsuffix cgo cmd/balancer/main.go
	docker build -t balancer-demo demo/balancer