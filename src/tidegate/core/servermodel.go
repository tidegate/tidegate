package core

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
	return EndpointId(self.ContainerId, self.IP, self.Port)
}

func EndpointId(containerId string, ip net.IP, port int64) string {
	return fmt.Sprintf("%v:%v:%v", containerId, string(ip), port)
}

type Server struct {
	Domain       string
	ExternalPort int64
	Endpoints    map[string]Endpoint
	observers    mapset.Set
}

type ServerChangeObserver interface {
	HandleEndpointCreation(server *Server, endpointId string) (err error)
	HandleEndpointDeletion(server *Server, endpointId string) (err error)
}

func NewServer(domain string, externalPort int64) (res *Server) {
	res = &Server{Domain: domain, ExternalPort: externalPort}
	res.observers = mapset.NewSet()
	res.Endpoints = make(map[string]Endpoint)
	return
}

func (self *Server) GetID() string {
	return fmt.Sprintf("%v:%v", self.Domain, self.ExternalPort)
}
func (self *Server) AddEndpoint(containerId string, ip net.IP, port int64, isSSL bool) {
	endpoint := Endpoint{IP: ip, Port: port, IsSSL: isSSL}
	self.Endpoints[endpoint.String()] = endpoint
	RootLogger.Debugf("New endpoint '%v:%v' added to server", ip, port)
	for observer := range self.observers.Iter() {
		RootLogger.Debugf("Notifier observer")
		observer.(ServerChangeObserver).HandleEndpointCreation(self, endpoint.String())
	}
}

func (self *Server) RemoveEndpoint(containerId string, ip net.IP, port int64) (err error) {
	id := EndpointId(containerId, ip, port)
	delete(self.Endpoints, id)
	for observer := range self.observers.Iter() {
		observer.(ServerChangeObserver).HandleEndpointDeletion(self, id)
	}
	return
	err = errors.New("Unable to find endpoint")
	return
}

func (self *Server) AddObserver(obs ServerChangeObserver) {
	self.observers.Add(obs)
	for endpointId, _ := range self.Endpoints {
		err := obs.HandleEndpointCreation(self, endpointId)
		if err != nil {
			RootLogger.Errorf("Unable to handle endpoint deletion: %s", err.Error())
		}
	}
}

func (self *Server) RemoveObserver(obs ServerChangeObserver) {
	for endpointId, _ := range self.Endpoints {
		err := obs.HandleEndpointDeletion(self, endpointId)
		if err != nil {
			RootLogger.Errorf("Unable to handle endpoint deletion: %s", err.Error())
		}
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
