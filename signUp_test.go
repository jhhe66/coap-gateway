package main

import (
	"net"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/go-ocf/authorization/protobuf/auth"
	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/http"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

func testSignUp(t *testing.T, ctx *fasthttp.RequestCtx) {
	var signUpResponse auth.SignUpResponse

	ctx.Response.Header.SetContentType(http.ProtobufContentType(&signUpResponse))
	signUpResponse.AccessToken = "abc"
	signUpResponse.UserId = "0"
	out := make([]byte, 1024)
	var err error
	if len(out) < signUpResponse.Size() {
		out, err = signUpResponse.Marshal()
	} else {
		var l int
		l, err = signUpResponse.MarshalTo(out)
		out = out[:l]
	}

	if err != nil {
		t.Fatalf("Cannot marshal response: %v", err)
	}

	ctx.Response.SetBody(out)
}

func testSignIn(t *testing.T, ctx *fasthttp.RequestCtx) {
	var signInResponse auth.SignInResponse
	ctx.SetContentType(http.ProtobufContentType(&signInResponse))
	signInResponse.ExpiresIn = 1
	out := make([]byte, 1024)
	var err error
	if len(out) < signInResponse.Size() {
		out, err = signInResponse.Marshal()
	} else {
		var l int
		l, err = signInResponse.MarshalTo(out)
		out = out[:l]
	}

	if err != nil {
		t.Fatalf("Cannot marshal response: %v", err)
	}

	ctx.SetBody(out)
}

func testCreateAuthServer(t *testing.T) (*fasthttp.Server, string, chan error) {
	router := fasthttprouter.New()
	router.POST("/signup", func(ctx *fasthttp.RequestCtx) {
		testSignUp(t, ctx)
	})
	router.POST("/signin", func(ctx *fasthttp.RequestCtx) {
		testSignIn(t, ctx)
	})

	s := fasthttp.Server{
		Handler: router.Handler,
	}
	ln, err := net.Listen("tcp", ":")
	if err != nil {
		t.Fatalf("Cannot listen on server: %v", err)
	}

	fin := make(chan error)
	var wk sync.WaitGroup
	wk.Add(1)
	go func() {
		wk.Done()
		fin <- s.Serve(ln)
	}()
	wk.Wait()

	return &s, "127.0.0.1:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port), fin
}

func TestSignUpPostHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coap.POST, `{}`, nil}, output{coap.BadRequest, ``, nil}},
		{"BadRequest1", input{coap.POST, `{"di": "abc", "accesstoken": 123}`, nil}, output{coap.BadRequest, ``, nil}},
		{"Changed0", input{coap.POST, `{"di": "abc", "accesstoken": "123"}`, nil}, output{coap.Changed, `{"accesstoken":"abc","expiresin":0,"redirecturi":"","refreshtoken":"","uid":"0"}`, nil}},
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
			testPostHandler(t, signUp, test, co)
		}
		t.Run(test.name, tf)
	}
}
