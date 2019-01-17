package service

import (
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

	observedResources     map[string]map[int64]observedResource // [deviceID][instanceID]
	observedResourcesLock sync.Mutex
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
		observedResources: make(map[string]map[int64]observedResource),
	}
}

func (session *Session) observeResource(res resources.Resource) error {
	session.observedResourcesLock.Lock()
	defer session.observedResourcesLock.Unlock()
	if _, ok := session.observedResources[res.DeviceId]; !ok {
		session.observedResources[res.DeviceId] = make(map[int64]observedResource)
	}
	if _, ok := session.observedResources[res.DeviceId][res.InstanceId]; ok {
		log.Warnf("Resource ocf://%v/%v are already published", res.DeviceId, res.Href)
		return nil
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
	session.observedResources[res.DeviceId][res.InstanceId] = observedResource{res: res, observation: observation}
	return nil
}

func (session *Session) getObservedResources(deviceID string, instanceIDs []int64, matches []resources.Resource) []resources.Resource {
	session.observedResourcesLock.Lock()
	defer session.observedResourcesLock.Unlock()

	getAllDeviceIDMatches := len(instanceIDs) == 0

	if deviceResourcesMap, ok := session.observedResources[deviceID]; ok {
		for _, instanceID := range instanceIDs {
			if getAllDeviceIDMatches {
				matches = append(matches, deviceResourcesMap[instanceID].res)
			} else if resource, ok := deviceResourcesMap[instanceID]; ok {
				matches = append(matches, resource.res)
			}
		}
	}

	return matches
}

func (session *Session) unobserveResourceLocked(deviceID string, instanceID int64, deleteResource bool) error {
	log.Infof("remove published resource ocf://%v/%v", deviceID, instanceID)

	obs := session.observedResources[deviceID][instanceID].observation
	if obs != nil {
		log.Infof("cancel observation of ocf://%v/%v", deviceID, instanceID)
		err := obs.Cancel()
		if err != nil {
			log.Errorf("Cannot cancel observation ocf//%v/%v", deviceID, instanceID)
		}
	}

	if deleteResource {
		delete(session.observedResources[deviceID], instanceID)
		if len(session.observedResources[deviceID]) == 0 {
			delete(session.observedResources, deviceID)
		}
	}

	return nil
}

func (session *Session) unobserveResources(rscs []resources.Resource, rscsUnpublished map[string]bool) error {
	session.observedResourcesLock.Lock()
	defer session.observedResourcesLock.Unlock()

	for _, resource := range rscs {
		if _, ok := session.observedResources[resource.DeviceId]; ok {
			session.unobserveResourceLocked(resource.DeviceId, resource.InstanceId, rscsUnpublished[resource.Id])
		} else {
			log.Errorf("Cannot unobserve resource %v: resource not found", resource.Id)
		}
	}

	return nil
}

func (session *Session) close() {
	log.Infof("Close session %v", session.client.RemoteAddr())
	session.keepalive.Done()
	session.observedResourcesLock.Lock()
	defer session.observedResourcesLock.Unlock()
	for deviceID, instanceIDs := range session.observedResources {
		for instanceID := range instanceIDs {
			err := session.unobserveResourceLocked(deviceID, instanceID, true)
			if err != nil {
				log.Errorf("Cannot remove observed resource ocf//%v/%v", deviceID, instanceID)
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
