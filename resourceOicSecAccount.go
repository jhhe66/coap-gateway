package main

import (
	"bytes"
	"errors"

	"github.com/go-ocf/authorization/protobuf/auth"
	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/http"
	"github.com/ugorji/go/codec"
)

var (
	oicSecAccount = "/oic/sec/account"
)

func validateSignUp(signUp auth.SignUpRequest) error {
	if len(signUp.DeviceId) == 0 {
		return errors.New("Invalid DeviceId")
	}
	if len(signUp.AuthorizationCode) == 0 {
		return errors.New("Invalid AuthorizationCode")
	}
	return nil
}

func postSignUpURI(server *Server) string {
	return server.AuthProtocol + "://" + server.AuthHost + "/signup"
}

// https://github.com/openconnectivityfoundation/security/blob/master/oic.r.account.raml#L27
func oicSecAccountPostHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	var signUp auth.SignUpRequest

	var cborHandle codec.CborHandle
	err := codec.NewDecoder(bytes.NewBuffer(req.Msg.Payload()), &cborHandle).Decode(&signUp)
	if err != nil {
		log.Errorf("Cannot unmarshal request for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	if err = validateSignUp(signUp); err != nil {
		log.Errorf("Invalid request from client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	httpRequestCtx := http.AcquireRequestCtx()
	defer http.ReleaseRequestCtx(httpRequestCtx)

	var signUpResponse auth.SignUpResponse
	httpCode, err := httpRequestCtx.PostProto(server.httpClient, postSignUpURI(server), &signUp, &signUpResponse)
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
	err = codec.NewEncoder(out, &cborHandle).Encode(signUpResponse)
	if err != nil {
		log.Errorf("Cannot marshal response for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}

	sendResponse(s, req.Client, coap.Changed, out.Bytes())
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/oic.r.account.raml
func oicSecAccountHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	switch req.Msg.Code() {
	case coap.POST:
		oicSecAccountPostHandler(s, req, server)
	default:
		log.Errorf("Forbidden request from %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.Forbidden, nil)
	}
}
