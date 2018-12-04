package main

import (
	"bytes"
	"errors"

	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/resources/http"
	"github.com/go-ocf/resources/protobuf/auth"
	"github.com/ugorji/go/codec"
)

var (
	oicSecSession = "/oic/sec/session"
)

func validateSignIn(signIn auth.SignInRequest) error {
	if len(signIn.DeviceId) == 0 {
		return errors.New("Invalid DeviceId")
	}
	if len(signIn.AccessToken) == 0 {
		return errors.New("Invalid AuthorizationCode")
	}
	if len(signIn.UserId) == 0 {
		return errors.New("Invalid UserId")
	}
	return nil
}

func postSignInURI(server *Server) string {
	return server.AuthProtocol + "://" + server.AuthHost + "/signin"
}

func oicSecSessionPostHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	var signIn auth.SignInRequest
	var cborHandle codec.CborHandle
	err := codec.NewDecoder(bytes.NewBuffer(req.Msg.Payload()), &cborHandle).Decode(&signIn)
	if err != nil {
		log.Errorf("Cannot unmarshal request for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	if err = validateSignIn(signIn); err != nil {
		log.Errorf("Invalid request from client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	httpRequestCtx := http.AcquireRequestCtx()
	defer http.ReleaseRequestCtx(httpRequestCtx)

	var signInResponse auth.SignInResponse
	httpCode, err := httpRequestCtx.PostProto(server.httpClient, postSignInURI(server), &signIn, &signInResponse)
	if err != nil {
		log.Errorf("Cannot sign up to auth server for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}
	code := httpCode2CoapCode(httpCode, coap.POST)
	log.Infof("Auth server response with code %v for client %v", code, req.Client.RemoteAddr())
	if code != coap.Changed {
		sendResponse(s, req.Client, code, nil)
		return
	}

	out := bytes.NewBuffer(make([]byte, 0, 1024))
	err = codec.NewEncoder(out, &cborHandle).Encode(signInResponse)
	if err != nil {
		log.Errorf("Cannot marshal response for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}

	//TODO store access token, user_id, device_id

	sendResponse(s, req.Client, code, out.Bytes())
}

func oicSecSessionHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	switch req.Msg.Code() {
	case coap.POST:
		oicSecSessionPostHandler(s, req, server)
	default:
		log.Errorf("Forbidden request from %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.Forbidden, nil)
	}
}
