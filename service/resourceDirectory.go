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

func postResourceUnpublishURI(server *Server) string {
	return server.ResourceProtocol + "://" + server.ResourceHost + uri.UnpublishResource
}

func publishResource(resource resources.Resource, server *Server, req *coap.Request, httpRequestCtx *http.RequestCtx, authContext commands.AuthorizationContext, ttl int32, links []resources.Resource) []resources.Resource {
	if resource.DeviceId == "" {
		log.Error("cannot publish a resource without device ID for client %v", req.Client.RemoteAddr())
		return links
	}

	if resource.Href == "" {
		log.Error("cannot publish a resource without a href for client %v", req.Client.RemoteAddr())
		return links
	}

	resource.Id = resource2UUID(resource.DeviceId, resource.Href)

	request := commands.PublishResourceRequest{
		AuthorizationContext: &authContext,
		ResourceId:           resource.Id,
		DeviceId:             resource.DeviceId,
		Resource:             &resource,
		TimeToLive:           ttl,
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

	return links
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
		links = publishResource(resource, server, req, httpRequestCtx, authContext, int32(w.TimeToLive), links)
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

func parseUnpublishQueryString(queries []interface{}, deviceID *string, instanceIDs []int64) ([]int64, error) {
	deviceIDFound := false

	for _, query := range queries {
		q := strings.Split(query.(string), "=")
		if len(q) == 2 {
			switch q[0] {
			case "di":
				*deviceID = q[1]
				deviceIDFound = true
			case "ins":
				i, err := strconv.Atoi(q[1])
				if err != nil {
					log.Errorf("Cannot convert %v to number", q[1])
				}
				instanceIDs = append(instanceIDs, int64(i))
			}
		}
	}

	if !deviceIDFound {
		return nil, fmt.Errorf("DeviceID not found")
	}

	return instanceIDs, nil
}

func unpublishResource(resource resources.Resource, server *Server, httpRequestCtx *http.RequestCtx, authContext commands.AuthorizationContext, deviceID string, rscsUnpublished map[string]bool) map[string]bool {
	request := commands.UnpublishResourceRequest{
		AuthorizationContext: &authContext,
		ResourceId:           resource.Id,
		DeviceId:             deviceID,
	}
	var response commands.UnpublishResourceResponse
	httpCode, err := httpRequestCtx.PostProto(server.httpClient, postResourceUnpublishURI(server), &request, &response)
	if err != nil {
		log.Errorf("cannot unpublish resource ID:%v for device ID:%v", resource.Id, resource.DeviceId)
	}

	if httpCode == fasthttp.StatusOK {
		log.Info("resource %v successfully unpublished for device ID %v", resource.Id, resource.DeviceId)
		rscsUnpublished[resource.Id] = true
	} else {
		log.Error("cannot unpublish resource %v for device %v", resource.Id, resource.DeviceId)
		rscsUnpublished[resource.Id] = false
	}

	return rscsUnpublished
}

func resourceDirectoryUnpublishHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	httpRequestCtx := http.AcquireRequestCtx()
	defer http.ReleaseRequestCtx(httpRequestCtx)

	session := server.clientContainer.find(req.Client.RemoteAddr().String())
	if session == nil {
		log.Errorf("Cannot find session for client %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}
	authContext := session.loadAuthorizationContext()

	queries := req.Msg.Options(coap.URIQuery)
	var deviceID string
	inss := make([]int64, 0, 32)
	inss, err := parseUnpublishQueryString(queries, &deviceID, inss)
	if err != nil {
		log.Errorf("Incorrect Unpublish query string - %v", err)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	rscs := make([]resources.Resource, 0, 32)
	rscsUnpublished := make(map[string]bool, 32)

	rscs = session.getObservedResources(deviceID, inss, rscs)
	if len(rscs) == 0 {
		log.Errorf("no matching resources found for the DELETE request parameters - with device ID and instance IDs %v, ", queries)
		sendResponse(s, req.Client, coap.BadRequest, nil)
		return
	}

	for _, resource := range rscs {
		rscsUnpublished = unpublishResource(resource, server, httpRequestCtx, authContext, deviceID, rscsUnpublished)
	}

	session.unobserveResources(rscs, rscsUnpublished)

	sendResponse(s, req.Client, coap.Deleted, nil)
}

type resourceDirectorySelector struct {
	SelectionCriteria int `json:"sel"`
}

func resourceDirectoryGetSelector(s coap.ResponseWriter, req *coap.Request, server *Server) {
	session := server.clientContainer.find(req.Client.RemoteAddr().String())
	if session == nil {
		log.Errorf("Cannot find session for client %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}

	var rds resourceDirectorySelector //we want to use sel:0 to prefer cloud RD

	var cborHandle codec.CborHandle
	out := bytes.NewBuffer(make([]byte, 0, 1024))
	err := codec.NewEncoder(out, &cborHandle).Encode(rds)
	if err != nil {
		log.Errorf("cannot marshal response for client %v: %v", req.Client.RemoteAddr(), err)
		sendResponse(s, req.Client, coap.InternalServerError, nil)
		return
	}

	sendResponse(s, req.Client, coap.Content, out.Bytes())
}

func resourceDirectoryHandler(s coap.ResponseWriter, req *coap.Request, server *Server) {
	switch req.Msg.Code() {
	case coap.POST:
		resourceDirectoryPublishHandler(s, req, server)
	case coap.DELETE:
		resourceDirectoryUnpublishHandler(s, req, server)
	case coap.GET:
		resourceDirectoryGetSelector(s, req, server)
	default:
		log.Errorf("Forbidden request from %v", req.Client.RemoteAddr())
		sendResponse(s, req.Client, coap.Forbidden, nil)
	}
}
