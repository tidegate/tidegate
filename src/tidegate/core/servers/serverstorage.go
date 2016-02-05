package servers

import (
	"errors"
	"fmt"
)

type ServerStorage struct {
	servers   map[ServerId]*Server
	observers    []*ServerChangeObserver
	//observers []chan *ServerChangeEvent
}

func NewServerStorage() (res *ServerStorage) {
	res = &ServerStorage{}
	res.servers = make(map[ServerId]*Server)
	//res.observers = mapset.NewSet()
	//res.observers = make([]chan *ServerChangeEvent)
	return
}
func (self *ServerStorage) AddServer(domain string, externalPort int64) (res *Server, err error) {
	serverId := NewServerId(domain,externalPort)
	if !self.Contains(serverId) {
		res = NewServer(domain, externalPort)
		self.servers[*serverId] = res
		for _,obs := range self.observers {
		  res.AddObserver(obs)
		}
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' already exists", serverId))
	}
	return
}

func (self *ServerStorage) RemoveServer(id *ServerId) (err error) {
	_,err = self.GetServer(id)
	if err == nil {
		delete(self.servers, *id)
		logger.Debugf("Server '%s' deleted", id)
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' does not exist", id))
	}
	return
}

func (self *ServerStorage) Contains(serverId *ServerId) (res bool) {
	_, res = self.servers[*serverId]
	return
}

func (self *ServerStorage) GetServer(serverId *ServerId) (res *Server, err error) {
	if self.Contains(serverId) {
		res = self.servers[*serverId]
	} else {
		err = errors.New(fmt.Sprintf("Server '%s' does not exist", serverId))
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

func (self *ServerStorage) AddServerObserver(obs *ServerChangeObserver) {
  logger.Debugf("Added observer")
  self.observers = append(self.observers,obs)
  for server := range self.Iter() {
    server.(*Server).AddObserver(obs)
  }
}









