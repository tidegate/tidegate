package servers

type EndpointAdditionEvent struct {
	Server *Server
}
type EndpointRemovalEvent struct {
	Server *Server
}

type ServerMonitor interface {
  //HandleEndpointAddition(server *Server) (err error)
  //HandleEndpointRemoval(server *Server) (err error)
  HandleEvent(value interface{})
}