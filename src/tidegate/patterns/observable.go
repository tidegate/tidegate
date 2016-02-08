package patterns

import (
	"github.com/deckarep/golang-set"
)

type Observable interface {
	NotifyObservers(value interface{})
	AddObserver(obs Observer)
	RemoveObserver(obs Observer)
}

type Observer interface {
//	Observe(obs Observable)
//	Unobserve(obs Observable)
	HandleEvent(payload interface{})
}

type ObservationHandler interface {
	HandleUpdate(value interface{})
}

type BasicObservable struct {
	observers mapset.Set
}

func NewBasicObservable() (res *BasicObservable) {
	res = &BasicObservable{}
	res.observers = mapset.NewSet()
	return
}

func (self *BasicObservable) AddObserver(obs Observer) {
	obsCh := make(chan interface{})
	go func() {
		for {
			val := <-obsCh
			obs.HandleEvent(val)
			obsCh <- nil
		}
	}()
	self.observers.Add(obsCh)
}

func (self *BasicObservable) RemoveObserver(obs Observer) {
	//self.observers.Remove(obs)
}

func (self *BasicObservable) NotifyObservers(payload interface{}) {
	for obs := range self.observers.Iter() {
		obs.(chan interface{}) <- payload
		<-obs.(chan interface{})
	}
}

type BasicObserver struct {
	hdlr ObservationHandler
}

func NewBasicObserver(hdlr ObservationHandler) (res *BasicObserver) {
	res = &BasicObserver{}
	res.hdlr = hdlr
	return
}

//func (self *BasicObserver) Observe(obs Observable) {
//	obs.AddObserver(self)
//}
//
//func (self *BasicObserver) Unobserve(obs Observable) {
//	obs.RemoveObserver(self)
//}

func (self *BasicObserver) HandleEvent(value interface{}) {
	self.hdlr.HandleUpdate(value)
}
