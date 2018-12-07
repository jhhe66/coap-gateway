package main

import (
	"os"
	"testing"

	coap "github.com/go-ocf/go-coap"
)

func TestSignInPostHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coap.POST, `{}`, nil}, output{coap.BadRequest, ``, nil}},
		{"BadRequest1", input{coap.POST, `{"di": "abc", "accesstoken": 123}`, nil}, output{coap.BadRequest, ``, nil}},
		{"BadRequest2", input{coap.POST, `{"di": "abc", "accesstoken": "123"}`, nil}, output{coap.BadRequest, ``, nil}},
		{"BadRequest3", input{coap.POST, `{"di": "abc", "uid": "0"}`, nil}, output{coap.BadRequest, ``, nil}},
		{"Changed1", input{coap.POST, `{"di": "abc", "uid":"0", "accesstoken":"123" }`, nil}, output{coap.Changed, `{"expiresin":1}`, nil}},
	}

	sauth, authAddrstr, authfin := testCreateAuthServer(t)
	os.Setenv("NETWORK", "tcp")
	os.Setenv("AUTH_HOST", authAddrstr)
	defer func() {
		sauth.Shutdown()
		if err := <-authfin; err != nil {
			t.Fatalf("server unexcpected shutdown: %v", err)
		}
	}()

	s, addrstr, fin, err := testCreateCoapGateway(t)
	if err != nil {
		t.Fatalf("unable to run test server: %v", err)
	}
	defer func() {
		s.Shutdown()
		err := <-fin
		if err != nil {
			t.Fatalf("server unexcpected shutdown: %v", err)
		}
	}()

	client := &coap.Client{Net: "tcp"}
	co, err := client.Dial(addrstr)
	if err != nil {
		t.Fatalf("unable to dialing: %v", err)
	}
	defer co.Close()

	for _, test := range tbl {
		tf := func(t *testing.T) {
			testPostHandler(t, signIn, test, co)
		}
		t.Run(test.name, tf)
	}
}
