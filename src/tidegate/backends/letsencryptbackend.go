package backends

import (
	"reflect"
	"tidegate/servers"
	"tidegate/core"
)

/**************************************************/
/**************************************************/
type LetsEncryptBackend struct {
	//Observer patterns.Observer
}

func NewLetsEncryptBackend() (res *LetsEncryptBackend) {
	res = &LetsEncryptBackend{}
	//res.Observer = patterns.NewBasicObserver(res)
	return
}

//func (self *LetsEncryptBackend) HandleEvent(value interface{}) {
//	self.Observer.Update(value)
//}

func (self *LetsEncryptBackend) HandleEvent(value interface{}) {
	switch value.(type) {
	case *core.ServerAdditionEvent:
		event := value.(*core.ServerAdditionEvent)
		logger.Debugf("Handling server '%s' creation", event.Server.GetId())
		self.HandleServerAddition(event.Server)
	case *core.ServerRemovalEvent:
		event := value.(*core.ServerRemovalEvent)
		logger.Debugf("Handling server '%s' removal", event.Server.GetId())
		self.HandleServerRemoval(event.Server)
	default:
		logger.Debugf("Event of type '%s' ignored", reflect.TypeOf(value))
	}
}

func (self *LetsEncryptBackend) HandleServerAddition(server *servers.Server) (err error) {
	logger.Debugf("Certificate for server '%s' has been generated", server.Domain)
	return
}

func (self *LetsEncryptBackend) HandleServerRemoval(server *servers.Server) (err error) {
	logger.Debugf("Certificate for server '%s' has been removed", server.Domain)
	return
}

