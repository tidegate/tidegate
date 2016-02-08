package core

import (
  "tidegate/servers"
  )

type ServerRemovalEvent struct {
	Server *servers.Server
}
type ServerAdditionEvent struct {
	Server *servers.Server
}

type ServerManagerMonitor interface {
//  HandleServerAddition(server *servers.Server) (err error)
//  HandleServerRemoval(server *servers.Server) (err error)
  HandleEvent(value interface {})
  
}