package servers



type ServerStateChangeEvent struct {
  Server Server
  Action string
}

type ServerChangeObserver struct {
	EventChannel chan *ServerStateChangeEvent
}

func NewServerChangeObserver() (res *ServerChangeObserver) {
  res = &ServerChangeObserver{}
  res.EventChannel = make(chan *ServerStateChangeEvent)
  return
}

func (self *ServerChangeObserver) NotifyUpdate(event *ServerStateChangeEvent) {
  self.EventChannel <- event
}

func (self *ServerChangeObserver) WaitForEvent() (res *ServerStateChangeEvent) {
  res = <- self.EventChannel
  return
}

func (self *ServerChangeObserver) HandleEndpointCreation(server *Server, endpointId *EndpointId) {
  self.NotifyUpdate(&ServerStateChangeEvent{*server,"CREATE"})
}
func (self *ServerChangeObserver) HandleEndpointDeletion(server *Server, endpointId *EndpointId) {
  self.NotifyUpdate(&ServerStateChangeEvent{*server,"DELETE"})
  
}