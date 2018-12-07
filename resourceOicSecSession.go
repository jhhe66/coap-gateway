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

func storeSessionInformation(s coap.ResponseWriter, req *coap.Request, server *Server, signIn auth.SignInRequest) error {
	session := server.clientContainer.find(req.Client.RemoteAddr().String())
	if session == nil {
		return errors.New("Cannot find session")
	}

	session.storeAuthorizationContext(signInRequest2AuthorizationContext(signIn))
	return nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/oic.r.session.raml#L27
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

	err = storeSessionInformation(s, req, server, signIn)
	if err != nil {
		log.Errorf("Cannot store session information for client: %v", err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	sendResponse(s, req.Client, code, out.Bytes())
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/oic.r.session.raml
func oicSecSessionHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	switch req.Msg.Code() {
	case coap.POST:
		oicSecSessionPostHandler(s, req, server)
	default:
		log.Errorf("Forbidden request from %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.Forbidden, nil)
	}
}
