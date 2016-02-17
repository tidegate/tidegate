package servers

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"github.com/aacebedo/tidegate/src/patterns"
)

type Endpoint struct {
	IP    net.IP
	Port  int64
	IsSSL bool
}

func (self *Endpoint) String() string {
	return string(*NewEndpointId(self.IP, self.Port))
}

type EndpointId string

func NewEndpointId(ip net.IP, port int64) (res *EndpointId) {
	tmp := EndpointId(fmt.Sprintf("%v_%v", string(ip), port))
	res = &tmp
	return
}

type ServerId string

func NewServerId(domain string, port int64) (res *ServerId) {
	tmp := ServerId(fmt.Sprintf("%v_%v", domain, port))
	res = &tmp
	return
}

type Server struct {
	Domain       string
	ExternalPort int64
	Endpoints    map[EndpointId]Endpoint
	observable   patterns.Observable
}

func NewServer(domain string, externalPort int64) (res *Server) {
	res = &Server{Domain: domain, ExternalPort: externalPort}
	res.Endpoints = make(map[EndpointId]Endpoint)
	res.observable = patterns.NewBasicObservable()
	return
}

func (self *Server) GetId() ServerId {
	return *NewServerId(self.Domain, self.ExternalPort)
}
func (self *Server) AddEndpoint(ip net.IP, port int64, isSSL bool) {
	endpoint := Endpoint{IP: ip, Port: port, IsSSL: isSSL}
	endpointId := NewEndpointId(ip, port)
	self.Endpoints[*endpointId] = endpoint
	logger.Debugf("New endpoint '%v' added to server", endpointId)
	self.NotifyObservers(&EndpointAdditionEvent{self})
}

func (self *Server) RemoveEndpoint(endpointId *EndpointId) {
	delete(self.Endpoints, *endpointId)
	self.NotifyObservers(&EndpointRemovalEvent{self})
}

func (self *Server) NotifyObservers(value interface{}) {
	self.observable.NotifyObservers(value)
}

func (self *Server) AddMonitor(monitor ServerMonitor) {
	logger.Debugf("Monitor added")
	self.observable.AddObserver(monitor)
}

func (self *Server) RemoveMonitor(monitor ServerMonitor) {
	logger.Debugf("Monitor removed")
	self.observable.RemoveObserver(monitor)
}

func (self *Server) GetRootDomain() (res string, err error) {
	r := regexp.MustCompile(`^(?P<subdomain>[A-Za-z0-9]{1,2}|[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]\.)??(?P<domain>[A-Za-z0-9][A-Za-z0-9]+|[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]\.)(?P<ext>[A-Za-z]{2,6})$`)
	matches := r.FindStringSubmatch(self.Domain)
	if len(matches) == 4 ||  len(matches) == 3 {
		res = fmt.Sprintf("%s%s",matches[len(matches)-2],matches[len(matches)-1])
	} else {
		err = errors.New("Unable to parse domain")
	}
  return
}

func (self *Server) IsSSL() (res bool, err error) {
	if len(self.Endpoints) > 0 {
		counter := 0
		for _, port := range self.Endpoints {
			if port.IsSSL {
				counter += 1
			}
		}
		if counter == 0 || counter == len(self.Endpoints) {
			res = (counter == len(self.Endpoints))
		} else {
		  logger.Errorf("Unable to know if endpoint is SSL or not")
			err = errors.New("Unable to know if endpoint is SSL or not")
		}

	} else {
		err = errors.New("No endpoints in server")
	}
	return
}
