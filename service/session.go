package service

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-ocf/authorization/protobuf/auth"
	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/resources/protobuf/resources"
	resourcesCommands "github.com/go-ocf/resources/protobuf/resources/commands"
)

type observedResource struct {
	res         resources.Resource
	observation *coap.Observation
}

//Session a setup of connection
type Session struct {
	server    *Server
	client    *coap.ClientCommander
	keepalive *Keepalive

	lockobservedResources sync.Mutex
	observedResources     map[string]map[string]observedResource
	authContext           resourcesCommands.AuthorizationContext
	authContextLock       sync.Mutex
}

//NewSession create and initialize session
func newSession(server *Server, client *coap.ClientCommander) *Session {
	log.Infof("Close session %v", client.RemoteAddr())
	return &Session{
		server:            server,
		client:            client,
		keepalive:         NewKeepalive(server, client),
		observedResources: make(map[string]map[string]observedResource),
	}
}

func (session *Session) observeResource(res resources.Resource) error {
	session.lockobservedResources.Lock()
	defer session.lockobservedResources.Unlock()
	if _, ok := session.observedResources[res.DeviceId]; !ok {
		session.observedResources[res.DeviceId] = make(map[string]observedResource)
	}
	if _, ok := session.observedResources[res.DeviceId][res.Href]; ok {
		return fmt.Errorf("Resource ocf://%v/%v are already published", res.DeviceId, res.Href)
	}
	return session.addObservedResourceLocked(res)
}

func (session *Session) addObservedResourceLocked(res resources.Resource) error {
	var observation *coap.Observation
	obs := isObservable(res)
	log.Infof("add published resource ocf://%v/%v, observable: %v", res.DeviceId, res.Href, obs)
	if obs {
		obs, err := session.client.Observe(res.Href, onObserveNotification)
		if err != nil {
			log.Errorf("Cannot observe ocf://%v/%v", res.DeviceId, res.Href)
		} else {
			observation = obs
		}
	} else {
		go func(client *coap.ClientCommander, deviceId string, href string) {
			resp, err := client.Get(href)
			if err != nil {
				log.Errorf("Cannot get ocf://%v/%v", deviceId, href)
				return
			}
			onGetResponse(&coap.Request{Client: client, Msg: resp})
		}(session.client, res.DeviceId, res.Href)
	}
	session.observedResources[res.DeviceId][res.Href] = observedResource{res: res, observation: observation}
	return nil
}

func (session *Session) removeObservedResourceLocked(deviceID, href string) error {
	log.Infof("remove published resource ocf://%v/%v", deviceID, href)
	obs := session.observedResources[deviceID][href].observation
	if obs != nil {
		log.Infof("cancel observation of ocf://%v/%v", deviceID, href)
		err := obs.Cancel()
		if err != nil {
			log.Errorf("Cannot cancel observation ocf//%v/%v", deviceID, href)
		}
	}

	delete(session.observedResources[deviceID], href)
	if len(session.observedResources[deviceID]) == 0 {
		delete(session.observedResources, deviceID)
	}
	return nil
}

func (session *Session) unobserveResource(deviceID string, observedResourcesIDs map[int64]bool) error {
	session.lockobservedResources.Lock()
	defer session.lockobservedResources.Unlock()

	if hrefs, ok := session.observedResources[deviceID]; ok {
		if len(observedResourcesIDs) == 0 {
			for href := range hrefs {
				session.removeObservedResourceLocked(deviceID, href)
			}
			return nil
		}
		for href, obsRes := range hrefs {
			if _, ok := observedResourcesIDs[obsRes.res.InstanceId]; ok {
				session.removeObservedResourceLocked(deviceID, href)
				delete(observedResourcesIDs, obsRes.res.InstanceId)
			}
		}
		if len(observedResourcesIDs) == 0 {
			return nil
		}
		out := make([]int64, 0, len(observedResourcesIDs))
		for _, val := range reflect.ValueOf(observedResourcesIDs).MapKeys() {
			out = append(out, val.Interface().(int64))
		}
		return fmt.Errorf("Cannot unobserve resources with %v: resource not found", out)
	}
	return fmt.Errorf("Cannot unobserve resource for %v: device not found", deviceID)

}

func (session *Session) close() {
	log.Infof("Close session %v", session.client.RemoteAddr())
	session.keepalive.Done()
	session.lockobservedResources.Lock()
	defer session.lockobservedResources.Unlock()
	for deviceID, hrefs := range session.observedResources {
		for href := range hrefs {
			err := session.removeObservedResourceLocked(deviceID, href)
			if err != nil {
				log.Errorf("Cannot remove observed resource ocf//%v/%v", deviceID, href)
			}
		}
	}
}

func (session *Session) storeAuthorizationContext(authContext resourcesCommands.AuthorizationContext) {
	log.Infof("Authorization context stored for client %v, device %v, user %v", session.client.RemoteAddr(), authContext.GetDeviceId(), authContext.GetUserId())
	session.authContextLock.Lock()
	defer session.authContextLock.Unlock()
	session.authContext = authContext
}

func (session *Session) loadAuthorizationContext() resourcesCommands.AuthorizationContext {
	session.authContextLock.Lock()
	defer session.authContextLock.Unlock()
	return session.authContext
}

func signInRequest2AuthorizationContext(signInRequest auth.SignInRequest) resourcesCommands.AuthorizationContext {
	return resourcesCommands.AuthorizationContext{
		AccessToken: signInRequest.AccessToken,
		DeviceId:    signInRequest.DeviceId,
		UserId:      signInRequest.UserId,
	}
}
