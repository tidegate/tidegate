package monitors

import (
)

type ContainerEndpoint struct {
	Domain       string
	InternalHostPort string
	ExternalHostPort string
	scheme        string
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
