package core

import (
	"net"
	"github.com/deckarep/golang-set"
	"reflect"
	"errors"
)

type Endpoint struct {
	IP   net.IP
	Port int
}

type Server struct {
	Domain       string
	ExternalPort int
	Endpoints    []Endpoint
	SSLEnabled   bool
	observers    mapset.Set
}

type ServerChangeObserver interface {
	HandleEndpointCreation(server *Server) (err error)
	HandleEndpointDeletion(server *Server) (err error)
}

func  NewServer(domain string, externalPort int, sslEnabled bool) (res *Server) {
  res = &Server{Domain: domain, ExternalPort: externalPort, SSLEnabled: sslEnabled}
  res.observers = mapset.NewSet()
  return
} 

func (self Server) AddEndpoint(ip net.IP, port int) {
	self.Endpoints = append(self.Endpoints, Endpoint{IP: ip, Port: port})
	RootLogger.Debugf("New endpoint '%v:%v' added to server", ip,port)
	for observer := range self.observers.Iter() {
	  RootLogger.Debugf("Notifier observer")
	  observer.(ServerChangeObserver).HandleEndpointCreation(&self)
	}
}

func (self Server) RemoveEndpoint(ip net.IP, port int) (err error) {
	for p, endpoint := range self.Endpoints {
	  if reflect.DeepEqual( endpoint.IP, ip) && endpoint.Port == port {
	    self.Endpoints = append(self.Endpoints[:p], self.Endpoints[p+1:]...)
	    for observer := range self.observers.Iter() {
	      observer.(ServerChangeObserver).HandleEndpointCreation(&self)
	    }
	    return
	  }
	}
	err = errors.New("Unable to find endpoint")
	return
}

func (self Server) AddObserver(obs ServerChangeObserver) {
	self.observers.Add(obs)
	obs.HandleEndpointCreation(&self)
}

func (self Server) RemoveObserver(obs ServerChangeObserver) {
  obs.HandleEndpointDeletion(&self)
	self.observers.Remove(obs)
}


