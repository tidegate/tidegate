package servers

import (
	"errors"
	"fmt"
	"github.com/deckarep/golang-set"
	"net"
)

type Endpoint struct {
	IP          net.IP
	Port        int64
	IsSSL       bool
	ContainerId string
}

func (self *Endpoint) String() string {
	return string(*NewEndpointId(self.ContainerId, self.IP, self.Port))
}

type EndpointId string

func NewEndpointId(containerId string, ip net.IP, port int64) (res *EndpointId) {
	tmp := EndpointId(fmt.Sprintf("%v:%v:%v", containerId, string(ip), port))
	res = &tmp
	return
}

type ServerId string

func NewServerId(domain string, port int64) (res *ServerId) {
	tmp := ServerId(fmt.Sprintf("%v:%v", domain, port))
	res = &tmp
	return
}

type Server struct {
	Domain       string
	ExternalPort int64
	Endpoints    map[EndpointId]Endpoint
	observers    mapset.Set
}

func NewServer(domain string, externalPort int64) (res *Server) {
	res = &Server{Domain: domain, ExternalPort: externalPort}
	res.observers = mapset.NewSet()
	res.Endpoints = make(map[EndpointId]Endpoint)
	return
}

func (self *Server) GetID() string {
	return fmt.Sprintf("%v:%v", self.Domain, self.ExternalPort)
}
func (self *Server) AddEndpoint(containerId string, ip net.IP, port int64, isSSL bool) {
	endpoint := Endpoint{IP: ip, Port: port, IsSSL: isSSL}
	endpointId := NewEndpointId(containerId, ip, port)
	self.Endpoints[*endpointId] = endpoint
	logger.Debugf("New endpoint '%v' added to server", endpointId)
		logger.Debugf("Notifying observers %v",self.observers.Cardinality())
	for obs := range self.observers.Iter() {
		logger.Debugf("Notified observer")
		obs.(*ServerChangeObserver).HandleEndpointCreation(self, endpointId)
	}
}

func (self *Server) RemoveEndpoint(endpointId *EndpointId) {
	delete(self.Endpoints, *endpointId)
	for obs := range self.observers.Iter() {
		obs.(*ServerChangeObserver).HandleEndpointDeletion(self, endpointId)
	}
	return
}

func (self *Server) AddObserver(obs *ServerChangeObserver) {
	if self.observers.Add(obs) {
	  logger.Warningf("Observer has bee added")
		for endpointId, _ := range self.Endpoints {
			obs.HandleEndpointCreation(self, &endpointId)
		}
	} else {
	  logger.Warningf("Unable to add observer")
	}
	
}

func (self *Server) RemoveObserver(obs *ServerChangeObserver) {
	for endpointId, _ := range self.Endpoints {
		obs.HandleEndpointDeletion(self, &endpointId)
	}
	self.observers.Remove(obs)
}

func (self *Server) IsSSL() (res bool, err error) {
	var finalRes *bool
	if len(self.Endpoints) > 0 {
		for _, port := range self.Endpoints {
			if finalRes != nil && (*finalRes) != port.IsSSL {
				err = errors.New("Endpoints do not have a consistent SSL state")
				return
			}
			*finalRes = port.IsSSL
		}
		res = *finalRes
	} else {
		err = errors.New("No endpoints in server")
	}
	return
}
