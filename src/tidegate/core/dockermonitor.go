package core

import (
	"encoding/json"
	"github.com/samalba/dockerclient"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type TideGateDescriptor struct {
	Domain     string
	SSLEnabled bool
}

type DockerMonitor struct {
	daemonAddr string
	client     *dockerclient.DockerClient
	servers    *ServerStorage
	backends   []Backend
}

func (self DockerMonitor) waitForInterrupt() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	for _ = range sigChan {
		self.client.StopAllMonitorEvents()
		os.Exit(0)
	}
}

func (self DockerMonitor) EventCallback(e *dockerclient.Event, ec chan error, args ...interface{}) {
	RootLogger.Infof("%v", e)
}

func (self DockerMonitor) AddBackend(backend Backend) {
  self.backends = append(self.backends, backend)
  self.servers.AddObserver(backend.(ServerChangeObserver))
}
func (self DockerMonitor) Start() {
	self.client.StartMonitorEvents(self.EventCallback, nil)
	containers, err := self.client.ListContainers(false, false, "")
	if err != nil {
		RootLogger.Fatalf("Unable to connect to docker daemon on '%s'. Are you sure the daemon is running ?", self.daemonAddr)
	}
	for _, c := range containers {
		self.ProcessContainer(&c)
	}
	
//	for server := range self.servers.Iter() {
//	  
//		genErr := backend.HandleServerCreation(server.(*Server))
//		if genErr != nil {
//			RootLogger.Warningf("Unable to handle server creation: %s", genErr.Error())
//		}
//	}

	//daemon := NGINXDaemon{ConfigPath:"/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin",BinPath:"/usr/sbin"}
	//daemon.Start()
	//daemon.Stop()
}

func (self DockerMonitor) Join() {

	self.waitForInterrupt()
}

func (self DockerMonitor) ProcessContainer(container *dockerclient.Container) (err error) {
	label, labelIsPresent := container.Labels["tidegate_descriptor"]
	if labelIsPresent && len(container.Ports) != 0 {
		var descriptor TideGateDescriptor
		var err = json.Unmarshal([]byte(label), &descriptor)
		if err == nil {
			for _, portBinding := range container.Ports {
				var ip = net.ParseIP(portBinding.IP)
				if ip != nil {
					server, err := self.servers.GetServer(descriptor.Domain, portBinding.PrivatePort)
					if err != nil {
						server, _ = self.servers.AddServer(descriptor.Domain, portBinding.PrivatePort, false)
					}
					server.AddEndpoint(ip, portBinding.PublicPort)
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
		if labelIsPresent {
			reason = "tidegate_descriptor label not found"
		} else {
			reason = "No port bindings found"
		}
		RootLogger.Debugf("Container '%s' ignored: %s", container.Names, reason)
	}
	return

}

func NewDockerMonitor(dockerAddr string) (res *DockerMonitor, err error) {
	res = &DockerMonitor{}
	res.client, _ = dockerclient.NewDockerClient(dockerAddr, nil)
	res.servers = NewServerStorage()
	res.daemonAddr = dockerAddr
	return
}
