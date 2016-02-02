package core

import (
	"errors"
	"fmt"
	"github.com/deckarep/golang-set"
)

type ServerStorage struct {
	servers   map[string]*Server
	observers    mapset.Set
}

func NewServerStorage() (res *ServerStorage) {
	res = &ServerStorage{}
	res.servers = make(map[string]*Server)
	res.observers = mapset.NewSet()
	return
}
func (self ServerStorage) AddServer(domain string, externalPort int, sslEnabled bool) (res *Server, err error) {
	var id = fmt.Sprintf("%v:%v", domain, externalPort)
	if !self.Contains(domain, externalPort) {
		res = NewServer(domain, externalPort, sslEnabled)
		self.servers[id] = res
		RootLogger.Debugf("Added new server '%s'", id)
		for obs := range self.observers.Iter() {
		  res.AddObserver(obs.(ServerChangeObserver))
		}
	} else {
		err = errors.New(fmt.Sprintf("Server '%s:%s' already exists", id))
	}
	return
}

func (self ServerStorage) RemoveServer(domain string, externalPort int) (err error) {
	var id = fmt.Sprintf("%v:%v", domain, externalPort)
	res,err := self.GetServer(domain, externalPort)
	if err == nil {
		delete(self.servers, id)
		RootLogger.Debugf("Deleted server '%s'", id)
		for _,endpoint := range res.Endpoints {
		  res.RemoveEndpoint(endpoint.IP, endpoint.Port)
		}
	} else {
		err = errors.New(fmt.Sprintf("Server '%s:%s' does not exists", id))
	}
	return
}

func (self ServerStorage) Contains(domain string, port int) bool {
	var _, contains = self.servers[fmt.Sprintf("%s:%s", domain, port)]
	return contains
}

func (self ServerStorage) GetServer(domain string, port int) (res *Server, err error) {
	var id = fmt.Sprintf("%v:%v", domain, port)
	if self.Contains(domain, port) {
		res = self.servers[id]
	} else {
		err = errors.New(fmt.Sprintf("Server '%s:%s' does not exists", id))
	}
	return
}

func (self *ServerStorage) Iter() <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		for _,value := range self.servers {
			ch <- value
		}
		close(ch)
	}()

	return ch
}

func (self ServerStorage) AddObserver(obs ServerChangeObserver) {
  self.observers.Add(obs)
  for _, server :=range self.servers {
	  server.AddObserver(obs)
	}
}

func (self ServerStorage) RemoveObserver(obs ServerChangeObserver) {
	for _, server :=range self.servers {
	  server.RemoveObserver(obs)
	}
	self.observers.Remove(obs)
}








