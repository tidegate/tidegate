package monitors

import (
	
	"encoding/json"
	//"errors"
	
	"github.com/samalba/dockerclient"
	"net"
	
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"github.com/aacebedo/tidegate/src/patterns"
	"time"
)

type TideGatePortDescriptor struct {
	Port uint64
	IsSSL  bool
}

type TideGateDescriptor struct {
	Domain string
	Ports  []TideGatePortDescriptor
}

type DockerManager struct {
	daemonAddr    string
	client        *dockerclient.DockerClient
	eventStopChan chan struct{}
	observable    patterns.Observable
}

//func (self *DockerManager) waitForInterrupt() {
//	sigChan := make(chan os.Signal, 1)
//	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
//	for _ = range sigChan {
//		self.client.StopAllMonitorEvents()
//		os.Exit(0)
//	}
//}

func (self *DockerManager) EventCallback(e *dockerclient.Event) {

	container, err := self.client.InspectContainer(e.ID)
	if err == nil {
		switch e.Status {
		case "kill":
			{
				logger.Debugf("Container '%s' has been stopped", container.Name)
				self.HandleContainerStop(container)
				break
			}
		case "start":
			{
				logger.Debugf("Container '%s' has been started", container.Name)
				self.HandleContainerStart(container)
				break
			}
		default:
			logger.Warningf("Status '%s' of container '%s' ignored", e.Status, container.Name)
		}

	} else {
		logger.Warningf("Unable to process container '%s': %s", e.ID, err.Error())
	}
}

func (self *DockerManager) Start() (err error) {
	now := time.Now().Unix()
	containers, err := self.client.ListContainers(false, false, "")
	if err != nil {
		logger.Fatalf("Unable to connect to docker daemon on '%s'. Are you sure the daemon is running ?", self.daemonAddr)
	}
	for _, c := range containers {
		containerInfo, err := self.client.InspectContainer(c.Id)
		if err == nil {
			self.HandleContainerStart(containerInfo)
		} else {
			logger.Warningf("Unable to get container info for container '%s'", c.Id)
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
	//			logger.Warningf("Unable to handle server creation: %s", genErr.Error())
	//		}
	//	}

	//	self.daemon = &NGINXDaemon{configPath: "/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin", binPath: "/usr/sbin"}
	//	self.daemon.Start()
	//daemon.Stop()
	logger.Debugf("Docker monitor started")
	return
}

func (self *DockerManager) Stop() {
	//self.daemon.Stop()
}

func (self *DockerManager) Join() {
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

func (self *DockerManager) HandleContainerStop(container *dockerclient.ContainerInfo) (err error) {

	label, labelIsPresent := container.Config.Labels["tidegate_descriptor"]
	if labelIsPresent && len(container.NetworkSettings.Ports) != 0 {
		var descriptor TideGateDescriptor
		var err = json.Unmarshal([]byte(label), &descriptor)
		if err == nil {

			for k, v := range container.NetworkSettings.Ports {

				if len(v) != 0 {
					servicePort := strings.Split(k, "/")
					servicePort2, err := strconv.Atoi(servicePort[0])
					internalPort, err := strconv.Atoi(v[0].HostPort)
					isSSL := false
					isPublished := false

					for _, portDesc := range descriptor.Ports {
						if portDesc.Port == uint64(servicePort2) {
							isPublished = true
							isSSL = portDesc.IsSSL
							break
						}
					}
					if isPublished {
						var ip = net.ParseIP(v[0].HostIp)
						if ip != nil && err == nil {
							/*tr := &http.Transport{
								TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
							}
							client := &http.Client{Transport: tr}
							_, err = client.Get(fmt.Sprintf("https://%v:%v/", ip, internalPort))*/
							val := ContainerEndpointRemovalEvent{&ContainerEndpoint{descriptor.Domain, ip, int64(servicePort2), int64(internalPort), isSSL}}
							self.observable.NotifyObservers(&val)

						} else {
							logger.Warningf("Port binding ignored for container '%s': Invalid IPAddress", container.Name)
						}
					} else {
						logger.Warningf("Container '%s' expose port '%v' but is not declared as published", container.Name, servicePort2)
					}

				}
			}
			//			for k, v := range container.NetworkSettings.Ports {
			//
			//			//	servicePort := strings.Split(k, "/")
			//			//	servicePort2, err := strconv.Atoi(servicePort[0])
			//		//		internalPort, err := strconv.Atoi(v[0].HostPort)
			//				var ip = net.ParseIP(v[0].HostIp)
			//				if ip != nil && err == nil {
			////					server, err := self.servers.GetServer(servers.NewServerId(descriptor.Domain, int64(servicePort2)))
			////					if err == nil {
			////						server.RemoveEndpoint(servers.NewEndpointId(container.Name, ip, int64(internalPort)))
			////					} else {
			////						err = errors.New(fmt.Sprintf("Server '%v:%v' not been found, cannot remove endpoint from it", descriptor.Domain, servicePort2))
			////					}
			//
			//				} else {
			//					logger.Warningf("Port binding ignored for container '%s': Invalid IPAddress", container.Name)
			//				}
			//
			//			}
		} else {
			logger.Warningf("Container '%s' ignored: Invalid tidegate_descriptor (%s)", container.Name, err.Error())
		}
	} else {
		var reason string
		if labelIsPresent {
			reason = "tidegate_descriptor label not found"
		} else {
			reason = "No port bindings found"
		}
		logger.Debugf("Container '%s' ignored: %s", container.Name, reason)
	}
	return
}

func (self *DockerManager) HandleContainerStart(container *dockerclient.ContainerInfo) (err error) {
	label, labelIsPresent := container.Config.Labels["tidegate_descriptor"]
	if labelIsPresent && len(container.NetworkSettings.Ports) != 0 {
		var descriptor TideGateDescriptor
		var err = json.Unmarshal([]byte(label), &descriptor)
		if err == nil {
			for k, v := range container.NetworkSettings.Ports {
				servicePort := strings.Split(k, "/")
				if len(v) != 0 {
					servicePort2, err := strconv.Atoi(servicePort[0])
					internalPort, err := strconv.Atoi(v[0].HostPort)
					
					isSSL := false
					isPublished := false

					for _, portDesc := range descriptor.Ports {
						if portDesc.Port == uint64(servicePort2) {
							isPublished = true
							isSSL = portDesc.IsSSL
							break
						}
					}
					if isPublished {
						var ip = net.ParseIP(v[0].HostIp)
						if ip != nil && err == nil {
							/*tr := &http.Transport{
								TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
							}
							client := &http.Client{Transport: tr}
							_, err = client.Get(fmt.Sprintf("https://%v:%v/", ip, internalPort))*/
							val :=ContainerEndpointAdditionEvent{&ContainerEndpoint{descriptor.Domain, ip, int64(servicePort2), int64(internalPort), isSSL}}
							self.observable.NotifyObservers(&val)

						} else {
							logger.Warningf("Port binding ignored for container '%s': Invalid IPAddress", container.Name)
						}
					} else {
						logger.Warningf("Container '%s' expose port '%v' but is not declared as published", container.Name, servicePort2)
					}
				}
			}
		} else {
			logger.Warningf("Container '%s' ignored: Invalid tidegate_descriptor (%s)", container.Name, err.Error())
		}
	} else {
		var reason string
		if labelIsPresent {
			reason = "tidegate_descriptor label not found"
		} else {
			reason = "No port bindings found"
		}
		logger.Debugf("Container '%s' ignored: %s", container.Name, reason)
	}
	return
}

func NewDockerManager(dockerAddr string) (res *DockerManager, err error) {
	res = &DockerManager{}
	res.client, _ = dockerclient.NewDockerClient(dockerAddr, nil)
	info, _ := res.client.Info()
	logger.Errorf("%v", info)

	//res.servers = servers
	res.daemonAddr = dockerAddr
	res.observable = patterns.NewBasicObservable()

	//serverManager.Observe(res.Observable)

	return
}

func (self *DockerManager) AddMonitor(monitor DockerManagerMonitor) {
	self.observable.AddObserver(monitor)
}
