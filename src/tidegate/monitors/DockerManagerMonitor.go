package monitors

import (
	"net"
)

type ContainerEndpoint struct {
	Domain       string
	IP           net.IP
	ExternalPort int64
	InternalPort int64
	IsSSL        bool
}

type ContainerEndpointAdditionEvent struct {
	Endpoint *ContainerEndpoint
}

type ContainerEndpointRemovalEvent struct {
	Endpoint *ContainerEndpoint
}

type DockerManagerMonitor interface {
	//HandleEndpointAddition(endpoint *ContainerEndpoint) (err error)
	//HandleEndpointRemoval(endpoint *ContainerEndpoint) (err error)
	//Update(value interface{})
	HandleEvent(value interface{})
	
}
