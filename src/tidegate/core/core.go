package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samalba/dockerclient"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/deckarep/golang-set"
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
}

func (self Server) AddEndpoint(ip net.IP, port int) {
	self.Endpoints = append(self.Endpoints, Endpoint{IP: ip, Port: port})
}

type TideGateDescriptor struct {
	Domain     string
	SSLEnabled bool
}

type ServerStorage struct {
	servers map[string]Server
	observers mapset.Set
}

func NewServerStorage() (res *ServerStorage) {
	res.servers = make(map[string]Server)
	res.observers = mapset.NewSet()
	return
}
func (self ServerStorage) AddServer(domain string, externalPort int, endpoints []Endpoint, sslEnabled bool) (err error) {
	var id = fmt.Sprintf("%v:%v", domain, externalPort)
	if !self.Contains(domain, externalPort) {
		self.servers[id] = Server{Domain: domain, ExternalPort: externalPort, Endpoints: endpoints, SSLEnabled: sslEnabled}
		RootLogger.Debugf("Added new server '%s'", id)
		for obs := range self.observers.Iter() {
		  obs.HandleServerDestruction()
		}
	} else {
		err = errors.New(fmt.Sprintf("Server '%s:%s' already exists", id))
	}
	return
}

func (self ServerStorage) RemoveServer(domain string, externalPort int) (err error) {
	var id = fmt.Sprintf("%v:%v", domain, externalPort)
	if self.Contains(domain, externalPort) {
		delete(self.servers,id)
		RootLogger.Debugf("Deleted server '%s'", id)
		for obs := range self.observers.Iter() {
		  obs.HandleServerDestruction()
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

func (self ServerStorage) GetServer(domain string, port int) (res Server, err error) {
	var id = fmt.Sprintf("%v:%v", domain, port)
	if self.Contains(domain, port) {
		res = self.servers[id]
	} else {
		err = errors.New(fmt.Sprintf("Server '%s:%s' does not exists", id))
	}
	return
}

func (self ServerStorage) GetServers() (*map[string]Server) {
  return &self.servers
}

type Observer interface {
  updateChange()
}

func (self ServerStorage) AddObserver(obs *Observer) {
  self.observers.Add(obs)
}

func (self ServerStorage) RemoveObserver(obs *Observer) {
  self.observers.Remove(obs)
}

var (
	client  *dockerclient.DockerClient
	servers ServerStorage = *NewServerStorage()
)

func waitForInterrupt() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	for _ = range sigChan {
		client.StopAllMonitorEvents()
		os.Exit(0)
	}
}

func eventCallback(e *dockerclient.Event, ec chan error, args ...interface{}) {
	RootLogger.Infof("%v", e)
}

func GenerateFile(dockerAddr string) {
	client, _ = dockerclient.NewDockerClient(dockerAddr, nil)

	//client.StartMonitorEvents(eventCallback, nil)

	var containers, err = client.ListContainers(false, false, "")
	if err != nil {
		RootLogger.Fatalf("Unable to connect to docker daemon on '%s'. Are you sure the daemon is running ?", dockerAddr)
	}
	for _, c := range containers {
		ProcessContainer(&c)
	}
	
	backend, err := NewNGINXBackend("/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin/","/usr/sbin")
	backend.Start()
  
  backend.configGenerator.GenerateConfigurations(&servers)
  
  //daemon := NGINXDaemon{ConfigPath:"/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin",BinPath:"/usr/sbin"}
	//daemon.Start()
	//daemon.Stop()
	
	//waitForInterrupt()
}

func ProcessContainer(container *dockerclient.Container) {
	var labelValue, present = container.Labels["tidegate_descriptor"]
	if present && len(container.Ports) != 0 {
		var descriptor TideGateDescriptor
		var err = json.Unmarshal([]byte(labelValue), &descriptor)
		if err == nil {
			for _, portBinding := range container.Ports {
				var ip = net.ParseIP(portBinding.IP)
				if ip != nil {
					var server, err = servers.GetServer(descriptor.Domain, portBinding.PrivatePort)
					if err == nil {
						server.AddEndpoint(ip, portBinding.PublicPort)
					} else {
						servers.AddServer(descriptor.Domain, portBinding.PrivatePort, []Endpoint{Endpoint{IP: ip, Port: portBinding.PublicPort}}, false)
					}
					RootLogger.Debugf("External port %s, internal port %s", portBinding.PrivatePort, portBinding.PublicPort)
				} else {
          RootLogger.Warningf("Port binding ignored for container '%s': Invalid IPAddress", container.Names)
				}
			}
		} else {
			RootLogger.Warningf("Container '%s' ignored: Invalid tidegate_descriptor (%s)", container.Names, err.Error())
		}
	} else {
		var reason string
		if present {
			reason = "tidegate_descriptor label not found"
		} else {
			reason = "No port bindings found"
		}
		RootLogger.Debugf("Container '%s' ignored: %s", container.Names, reason)
	}

}
