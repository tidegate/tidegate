package core

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samalba/dockerclient"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type TideGateDescriptor struct {
	Domain     string
	SSLEnabled bool
}

type DockerMonitor struct {
	daemonAddr string

	client        *dockerclient.DockerClient
	servers       *ServerStorage
	eventStopChan chan struct{}
	daemon        *NGINXDaemon
}

//func (self *DockerMonitor) waitForInterrupt() {
//	sigChan := make(chan os.Signal, 1)
//	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
//	for _ = range sigChan {
//		self.client.StopAllMonitorEvents()
//		os.Exit(0)
//	}
//}

func (self *DockerMonitor) EventCallback(e *dockerclient.Event) {
	RootLogger.Infof("%v", e)
	container, err := self.client.InspectContainer(e.ID)

	if err == nil {
	  RootLogger.Debugf("%v",container)
		switch e.Status {
		case "kill":
			{
			  RootLogger.Debugf("Container '%s' has been stopped", container.Name)
				self.ProcessContainerStart(container)
				break
			}
		case "start":
			{
			  RootLogger.Debugf("Container '%s' has been started", container.Name)
				self.ProcessContainerStop(container)
				break
			}
		default:
			RootLogger.Warningf("Status '%s' of container '%s' ignored", e.Status, container.Name)
		}

	} else {
		RootLogger.Warningf("Unable to process container '%s': %s", e.ID, err.Error())
	}
}

func (self *DockerMonitor) Start() (err error) {

	now := time.Now().Unix()
	containers, err := self.client.ListContainers(false, false, "")
	if err != nil {
		RootLogger.Fatalf("Unable to connect to docker daemon on '%s'. Are you sure the daemon is running ?", self.daemonAddr)
	}
	for _, c := range containers {
		containerInfo, err := self.client.InspectContainer(c.Id)
		if err == nil {
			self.ProcessContainerStart(containerInfo)
		} else {
			RootLogger.Warningf("Unable to get container info for container '%s'", c.Id)
		}
	}

	self.eventStopChan = make(chan struct{})

	go func() {
		eventErrChan, err := self.client.MonitorEvents(&dockerclient.MonitorEventsOptions{int(now), 0, nil}, self.eventStopChan)
		if err != nil {
			return
		}

		for e := range eventErrChan {
			if e.Error != nil {
				err = e.Error
				return
			}
			self.EventCallback(&e.Event)
		}
	}()

	//self.client.StartMonitorEvents(self.EventCallback, nil)

	//	for server := range self.servers.Iter() {
	//
	//		genErr := backend.HandleServerCreation(server.(*Server))
	//		if genErr != nil {
	//			RootLogger.Warningf("Unable to handle server creation: %s", genErr.Error())
	//		}
	//	}

	self.daemon = &NGINXDaemon{configPath: "/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin", binPath: "/usr/sbin"}
	self.daemon.Start()
	//daemon.Stop()
	return
}

func (self *DockerMonitor) Stop() {
	self.daemon.Stop()
}

func (self *DockerMonitor) Join() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	for _ = range sigChan {
		if self.eventStopChan == nil {
			return
		}
		close(self.eventStopChan)
		os.Exit(0)
	}
}

func (self *DockerMonitor) ProcessContainerStop(container *dockerclient.ContainerInfo) (err error) {
	label, labelIsPresent := container.Config.Labels["tidegate_descriptor"]
	if labelIsPresent && len(container.NetworkSettings.Ports) != 0 {
		var descriptor TideGateDescriptor
		var err = json.Unmarshal([]byte(label), &descriptor)
		if err == nil {
			for k, v := range container.NetworkSettings.Ports {
				RootLogger.Errorf("%v %v", k, v)
				servicePort := strings.Split(k, "/")
				servicePort2, err := strconv.Atoi(servicePort[0])
				internalPort, err := strconv.Atoi(v[0].HostPort)
				var ip = net.ParseIP(v[0].HostIp)
				if ip != nil && err == nil {
					server, err := self.servers.GetServer(descriptor.Domain, int64(servicePort2))
					if err == nil {
						server.RemoveEndpoint(container.Name, ip, int64(internalPort))
						RootLogger.Debugf("Length of endpoint %v", len(server.Endpoints))
						RootLogger.Debugf("External port %v, internal port %v", internalPort, servicePort2)
					} else {
						err = errors.New(fmt.Sprintf("Server '%v:%v' not been found, cannot remove endpoint from it", descriptor.Domain, servicePort2))
					}

				} else {
					RootLogger.Warningf("Port binding ignored for container '%s': Invalid IPAddress", container.Name)
				}

			}
		} else {
			RootLogger.Warningf("Container '%s' ignored: Invalid tidegate_descriptor (%s)", container.Name, err.Error())
		}
	} else {
		var reason string
		if labelIsPresent {
			reason = "tidegate_descriptor label not found"
		} else {
			reason = "No port bindings found"
		}
		RootLogger.Debugf("Container '%s' ignored: %s", container.Name, reason)
	}
	return
}

func (self *DockerMonitor) ProcessContainerStart(container *dockerclient.ContainerInfo) (err error) {
	label, labelIsPresent := container.Config.Labels["tidegate_descriptor"]
	if labelIsPresent && len(container.NetworkSettings.Ports) != 0 {
		var descriptor TideGateDescriptor
		var err = json.Unmarshal([]byte(label), &descriptor)
		if err == nil {
			for k, v := range container.NetworkSettings.Ports {
				RootLogger.Errorf("%v %v", k, v)
				servicePort := strings.Split(k, "/")
				servicePort2, err := strconv.Atoi(servicePort[0])
				internalPort, err := strconv.Atoi(v[0].HostPort)
				var ip = net.ParseIP(v[0].HostIp)
				if ip != nil && err == nil {
					server, err := self.servers.GetServer(descriptor.Domain, int64(servicePort2))
					if err != nil {
						RootLogger.Debugf("Server '%v:%v' not been found, creating it", descriptor.Domain, servicePort2)
						server, _ = self.servers.AddServer(descriptor.Domain, int64(servicePort2))
					}
					tr := &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					}
					client := &http.Client{Transport: tr}
					_, err = client.Get(fmt.Sprintf("https://%v:%v/", ip, internalPort))
					isSSL := err == nil

					server.AddEndpoint(container.Name, ip, int64(internalPort), isSSL)
					RootLogger.Debugf("Length of endpoint %v", len(server.Endpoints))
					RootLogger.Debugf("External port %v, internal port %v", internalPort, servicePort2)
				} else {
					RootLogger.Warningf("Port binding ignored for container '%s': Invalid IPAddress", container.Name)
				}
			}
		} else {
			RootLogger.Warningf("Container '%s' ignored: Invalid tidegate_descriptor (%s)", container.Name, err.Error())
		}
	} else {
		var reason string
		if labelIsPresent {
			reason = "tidegate_descriptor label not found"
		} else {
			reason = "No port bindings found"
		}
		RootLogger.Debugf("Container '%s' ignored: %s", container.Name, reason)
	}
	return
}

func NewDockerMonitor(dockerAddr string, servers *ServerStorage) (res *DockerMonitor, err error) {
	res = &DockerMonitor{}
	res.client, _ = dockerclient.NewDockerClient(dockerAddr, nil)
	info, _ := res.client.Info()
	RootLogger.Errorf("%v", info)

	res.servers = servers
	res.daemonAddr = dockerAddr

	return
}
