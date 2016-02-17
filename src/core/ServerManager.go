package core

import (
	"errors"
	"fmt"
	"github.com/deckarep/golang-set"
	"reflect"
	"github.com/aacebedo/tidegate/src/patterns"
	"github.com/aacebedo/tidegate/src/servers"
	"github.com/aacebedo/tidegate/src/monitors"
)

type ServerManager struct {
	servers                map[servers.ServerId]*servers.Server
	
	observable             patterns.Observable
	serverMonitors mapset.Set //  []backends.EndpointProcessingBackend

}

func NewServerManager() (res *ServerManager) {
	res = &ServerManager{}
	//res.observer = patterns.NewBasicObserver(res)
	res.observable = patterns.NewBasicObservable()
	res.servers = make(map[servers.ServerId]*servers.Server)
	res.serverMonitors = mapset.NewSet()
	return
}
func (self *ServerManager) AddServer(domain string, externalPort int64) (res *servers.Server, err error) {
	serverId := servers.NewServerId(domain, externalPort)
	if !self.Contains(serverId) {
		res = servers.NewServer(domain, externalPort)
		self.servers[*serverId] = res
		for monitor := range self.serverMonitors.Iter() {
		  res.AddMonitor(monitor.(servers.ServerMonitor))
		}
		self.observable.NotifyObservers(&ServerAdditionEvent{res})
		
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' already exists", serverId))
	}
	return
}

func (self *ServerManager) RemoveServer(id *servers.ServerId) (err error) {
	server, err := self.GetServer(id)
	if err == nil {
		delete(self.servers, *id)
		self.observable.NotifyObservers(&ServerRemovalEvent{server})
		
		
		
		//		for backend := range self.serverBackends.Iter() {
		//		  server.RemoveObserver(backend.(patterns.Observer))
		//		  //backend.(patterns.Observer).Unobserve(server)
		//		}
		logger.Debugf("Server '%s' deleted", id)
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' does not exist", id))
	}
	return
}

func (self *ServerManager) Contains(serverId *servers.ServerId) (res bool) {
	_, res = self.servers[*serverId]
	return
}

func (self *ServerManager) GetServer(serverId *servers.ServerId) (res *servers.Server, err error) {
	if self.Contains(serverId) {
		res = self.servers[*serverId]
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' does not exist", serverId))
	}
	return
}

//
//func (self *ServerStorage) Iter() <-chan interface{} {
//	ch := make(chan interface{})
//	go func() {
//		for _,value := range self.servers {
//			ch <- value
//		}
//		close(ch)
//	}()
//	return ch
//}
//
//func (self *ServerStorage) AddServerObserver(obs patterns.Observer) {
//  logger.Debugf("Added observer")
//  self.observers = append(self.observers,obs)
//  for server := range self.Iter() {
//    server.(*Server).AddObserver(obs)
//  }
//}


func (self *ServerManager) Unobserve(obs patterns.Observable) {
	obs.RemoveObserver(self)
	//self.observer.Unobserve(obs)
}

//func (self *ServerManager) Update(value interface{}) {
//  logger.Debugf("toto")
//	self.observer.Update(value)
//}

func (self *ServerManager) HandleEvent(value interface{}) {
	switch value.(type) {
	case *monitors.ContainerEndpointAdditionEvent:
		event := value.(*monitors.ContainerEndpointAdditionEvent)
		serverId := servers.NewServerId(event.Endpoint.Domain, 
		    event.Endpoint.ExternalPort)
		server, err := self.GetServer(serverId)
		if err != nil {
			logger.Debugf("Server '%v' not been found, creating it",serverId)
			server, _ = self.AddServer(event.Endpoint.Domain, 
			  event.Endpoint.ExternalPort)
		}
		server.AddEndpoint(event.Endpoint.IP, event.Endpoint.InternalPort, 
		  event.Endpoint.IsSSL)
	case *monitors.ContainerEndpointRemovalEvent:
		event := value.(*monitors.ContainerEndpointRemovalEvent)
		serverId := servers.NewServerId(event.Endpoint.Domain, 
		    event.Endpoint.ExternalPort)
		
		
		server, err := self.GetServer(serverId)
		if err == nil {
		  serverId := servers.NewServerId(event.Endpoint.Domain, 
		    event.Endpoint.ExternalPort)
		  logger.Debugf("Server '%v' been found, removing endpoint from it", serverId)
			server.RemoveEndpoint(servers.NewEndpointId(event.Endpoint.IP, event.Endpoint.InternalPort))
		  if len(server.Endpoints) == 0 {
		    self.RemoveServer(serverId)
		  }
		}
	
	default:
		logger.Debugf("Event of type '%s' ignored", reflect.TypeOf(value))
	}
}

func (self *ServerManager) AddMonitor(monitor ServerManagerMonitor) {
	logger.Debugf("New endpoint event monitor added")
	self.observable.AddObserver(monitor.(patterns.Observer))
}

func (self *ServerManager) AddServerMonitor(monitor servers.ServerMonitor) {
	logger.Debugf("New server event monitor added")
	self.serverMonitors.Add(monitor)
	for _, server := range self.servers {
	  server.AddMonitor(monitor)
	}
}