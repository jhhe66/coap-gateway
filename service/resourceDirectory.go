package service

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/http"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/resources/protobuf/resources"
	"github.com/go-ocf/resources/protobuf/resources/commands"
	"github.com/go-ocf/resources/uri"
	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
	"github.com/valyala/fasthttp"
)

const observable = 2

var resourceDirectory = "oic/rd"

type wkRd struct {
	DeviceID   string               `json:"di"`
	Links      []resources.Resource `json:"links"`
	TimeToLive int                  `json:"ttl"`
}

func parsePostPayload(msg coap.Message) (wkRd map[string]interface{}, err error) {
	err = codec.NewDecoderBytes(msg.Payload(), new(codec.CborHandle)).Decode(&wkRd)
	if err != nil {
		err = fmt.Errorf("Cannot decode CBOR: %v", err)
	}
	return
}

func sendResponse(s coap.ResponseWriter, client *coap.ClientCommander, code coap.COAPCode, payload []byte) {
	s.SetCode(code)
	if payload != nil {
		s.SetContentFormat(coap.AppCBOR)
	}
	_, err := s.Write(payload)
	if err != nil {
		log.Errorf("Cannot send reply to %v: %v", client.RemoteAddr(), err)
	}
}

func isObservable(res resources.Resource) bool {
	return res.Policies != nil && res.Policies.BitFlags&observable == observable
}

func resource2UUID(deviceID, href string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+href).String()
}

func postResourcePublishURI(server *Server) string {
	return server.ResourceProtocol + "://" + server.ResourceHost + uri.PublishResource
}

func resourceDirectoryPublishHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	var w wkRd
	var cborHandle codec.CborHandle
	err := codec.NewDecoder(bytes.NewBuffer(req.Msg.Payload()), &cborHandle).Decode(&w)
	if err != nil {
		log.Errorf("Cannot unmarshal request for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	if w.DeviceID == "" || len(w.Links) == 0 || w.TimeToLive <= 0 {
		log.Error("wkRd structure cannot contain empty fields")
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	httpRequestCtx := http.AcquireRequestCtx()
	defer http.ReleaseRequestCtx(httpRequestCtx)

	session := server.clientContainer.find(req.Client.RemoteAddr().String())
	if session == nil {
		log.Errorf("Could not find a valid session for client %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}
	authContext := session.loadAuthorizationContext()

	links := make([]resources.Resource, 0, len(w.Links))
	for _, resource := range w.Links {

		if resource.DeviceId == "" {
			log.Error("cannot publish a resource without device ID for client %v", req.Client.RemoteAddr())
			continue
		}

		if resource.Href == "" {
			log.Error("cannot publish a resource without a href for client %v", req.Client.RemoteAddr())
			continue
		}

		resource.Id = resource2UUID(resource.DeviceId, resource.Href)

		request := commands.PublishResourceRequest{
			Resource:             &resource,
			AuthorizationContext: &authContext,
			PublishedBy:          resource.DeviceId,
			TimeToLive:           int32(w.TimeToLive),
		}
		var response commands.PublishResourceResponse
		httpCode, err := httpRequestCtx.PostProto(server.httpClient, postResourcePublishURI(server), &request, &response)
		if err != nil {
			log.Errorf("cannot publish resource ID:%v for device ID:%v", resource.Id, resource.DeviceId)
		}

		if httpCode == fasthttp.StatusOK {
			resource.InstanceId = response.InstanceId
			links = append(links, resource)
			log.Info("resource successfull published for resource %v, device ID", resource.Id, resource.DeviceId)
		} else {
			log.Error("cannot publish resource ID:%v for device ID:%v", resource.Id, resource.DeviceId)
		}
	}

	if len(links) == 0 {
		log.Error("empty links for device %v", w.DeviceID)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	w.Links = links

	for _, res := range links {
		err := session.observeResource(res)
		if err != nil {
			log.Errorf("cannot observe published resource %v for device %v", res.Id, res.DeviceId)
		}
	}

	out := bytes.NewBuffer(make([]byte, 0, 1024))
	err = codec.NewEncoder(out, &cborHandle).Encode(w)
	if err != nil {
		log.Errorf("cannot marshal response for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}
	sendResponse(s, req.Client, coap.Changed, out.Bytes())
}

func resourceDirectoryUnpublishHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	session := server.clientContainer.find(req.Client.RemoteAddr().String())
	if session == nil {
		log.Errorf("Cannot find session for client %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}

	queries := req.Msg.Options(coap.URIQuery)
	var deviceID string
	inss := make(map[int64]bool)
	for _, query := range queries {
		q := strings.Split(query.(string), "=")
		if len(q) == 2 {
			switch q[0] {
			case "di":
				deviceID = q[1]
			case "ins":
				i, err := strconv.Atoi(q[1])
				if err != nil {
					log.Errorf("Cannot convert %v to number", q[1])
				}
				inss[int64(i)] = true
			}
		}
	}

	err := session.unobserveResource(deviceID, inss)
	if err != nil {
		log.Errorf("%v", err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	sendResponse(s, req.Client, coap.Deleted, nil)
}

func resourceDirectoryHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	switch req.Msg.Code() {
	case coap.POST:
		resourceDirectoryPublishHandler(s, req, server)
	case coap.DELETE:
		resourceDirectoryUnpublishHandler(s, req, server)
	default:
		log.Errorf("Forbidden request from %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.Forbidden, nil)
	}
}
