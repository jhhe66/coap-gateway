package main

import (
	"github.com/go-ocf/coap-gateway/service"
)

func main() {
	service.New().Serve()
}
