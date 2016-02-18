package core

import (
	"errors"
	"fmt"
	"github.com/aacebedo/tidegate/src/monitors"
	"reflect"
	"net/url"
	"regexp"
	"net"
)

type ServerManager struct {
	//servers                map[servers.ServerId]*servers.Server

	//observable             patterns.Observable
	//serverMonitors mapset.Set //  []backends.EndpointProcessingBackend

	servers map[string]*TidegateServer
}

func NewServerManager() (res *ServerManager) {
	res = &ServerManager{}
	//res.observer = patterns.NewBasicObserver(res)
	//res.observable = patterns.NewBasicObservable()
	//res.servers = make(map[servers.ServerId]*servers.Server)
	//res.serverMonitors = mapset.NewSet()
	res.servers = make(map[string]*TidegateServer)
	return
}

func (self *ServerManager) AddServer(hostPort string) (res *TidegateServer) {
	if server, ok := self.servers[hostPort]; !ok {
		res = NewTidegateServer(hostPort)
		self.servers[hostPort] = res
		go res.ListenAndServe()
		logger.Debugf("Add new server: %v",hostPort)
	} else {
		res = server
	}
	return

	//	serverId := servers.NewServerId(domain, externalPort)
	//	if !self.Contains(serverId) {
	//		res = servers.NewServer(domain, externalPort)
	//		self.servers[*serverId] = res
	//		for monitor := range self.serverMonitors.Iter() {
	//		  res.AddMonitor(monitor.(servers.ServerMonitor))
	//		}
	//		self.observable.NotifyObservers(&ServerAdditionEvent{res})
	//
	//	} else {
	//		err = errors.New(fmt.Sprintf("Server '%s' already exists", serverId))
	//	}
	//	return
}

//func (self *ServerManager) RemoveServer(id *servers.ServerId) (err error) {
//	server, err := self.GetServer(id)
//	if err == nil {
//		delete(self.servers, *id)
//		self.observable.NotifyObservers(&ServerRemovalEvent{server})
//
//
//
//		//		for backend := range self.serverBackends.Iter() {
//		//		  server.RemoveObserver(backend.(patterns.Observer))
//		//		  //backend.(patterns.Observer).Unobserve(server)
//		//		}
//		logger.Debugf("Server '%s' deleted", id)
//	} else {
//		err = errors.New(fmt.Sprintf("Server '%s' does not exist", id))
//	}
//	return
//}

func (self *ServerManager) GetServer(hostPort string) (res *TidegateServer, err error) {
	if server, ok := self.servers[hostPort]; ok {
		res = server
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' does not exist", hostPort))
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

//func (self *ServerManager) Unobserve(obs patterns.Observable) {
//	obs.RemoveObserver(self)
//self.observer.Unobserve(obs)
//}

//func (self *ServerManager) Update(value interface{}) {
//  logger.Debugf("toto")
//	self.observer.Update(value)
//}

func (self *ServerManager) HandleEvent(value interface{}) {
	switch value.(type) {
	case *monitors.ContainerEndpointAdditionEvent:
		event := value.(*monitors.ContainerEndpointAdditionEvent)
		//		serverId := servers.NewServerId(event.Endpoint.Domain,
		//		    event.Endpoint.ExternalPort)
		
		host,_,err := net.SplitHostPort(event.Endpoint.InternalHostPort)
		if err != nil {
		  logger.Fatalf("Unable to parse host and port")
		}
		server := self.AddServer(fmt.Sprintf("%v:%v",host,443))
		  if !server.Contains(event.Endpoint.Domain) {
		  re,err := regexp.Compile("/.well-known/acme-challenge/.*")
		  if err == nil {
		    proxy := server.AddReverseProxy(event.Endpoint.Domain, re)
		    proxy.AddEndpoint(&url.URL{
                    	    Scheme: "http",
                    		  Host:   "0.0.0.0:444",
                   	})
		    }
		 }
		
		server = self.AddServer(fmt.Sprintf("%v:%v",host,80))
		if !server.Contains(event.Endpoint.Domain) {
		  re,err := regexp.Compile("/.well-known/acme-challenge/.*")
		  if err == nil {
		    proxy := server.AddReverseProxy(event.Endpoint.Domain, re)
		    proxy.AddEndpoint(&url.URL{
                    	    Scheme: "http",
                    		  Host:   "0.0.0.0:81",
                   	})
		    }
		 }
		
		server = self.AddServer(event.Endpoint.InternalHostPort)
		re,err := regexp.Compile(".*")  
		proxy := server.AddReverseProxy(event.Endpoint.Domain, re)
		proxy.AddEndpoint(&url.URL{
                    	    Scheme: "http",
                    		  Host:   event.Endpoint.ExternalHostPort,
                    	})
		  
		
	case *monitors.ContainerEndpointRemovalEvent:
		//		event := value.(*monitors.ContainerEndpointRemovalEvent)
		//		serverId := servers.NewServerId(event.Endpoint.Domain,
		//		    event.Endpoint.ExternalPort)
		//
		//
		//		server, err := self.GetServer(serverId)
		//		if err == nil {
		//		  serverId := servers.NewServerId(event.Endpoint.Domain,
		//		    event.Endpoint.ExternalPort)
		//		  logger.Debugf("Server '%v' been found, removing endpoint from it", serverId)
		//			server.RemoveEndpoint(servers.NewEndpointId(event.Endpoint.IP, event.Endpoint.InternalPort))
		//		  if len(server.Endpoints) == 0 {
		//		    self.RemoveServer(serverId)
		//		  }
		//		}

	default:
		logger.Debugf("Event of type '%s' ignored", reflect.TypeOf(value))
	}
}

//func (self *ServerManager) AddMonitor(monitor ServerManagerMonitor) {
//	logger.Debugf("New endpoint event monitor added")
//	self.observable.AddObserver(monitor.(patterns.Observer))
//}
//
//func (self *ServerManager) AddServerMonitor(monitor servers.ServerMonitor) {
//	logger.Debugf("New server event monitor added")
//	self.serverMonitors.Add(monitor)
//	for _, server := range self.servers {
//	  server.AddMonitor(monitor)
//	}
//}
