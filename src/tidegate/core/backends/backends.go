package backends

import (
  "tidegate/core/servers"
)

type ReverseProxyBackend interface {
  Start() (err error)
  Stop() (err error)
  ProcessServerCreation(server *servers.Server) (err error)
  ProcessServerDeletion(server *servers.Server) (err error)
}

type SSLCertificateBackend interface {
  ProcessServerCreation(server *servers.Server) (err error)
  ProcessServerDeletion(server *servers.Server) (err error)
}