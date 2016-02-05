package core

import (
  "github.com/deckarep/golang-set"
)

type Observable interface {
  NotifyObservers(value *interface {})
  AddObserver(obs Observer)
  RemoveObserver(obs Observer)
}

type Observer interface {
  Observe(obs Observable)
  Unobserver(obs Observable)
  Update(value *interface {})
}

type ObservationHandler interface {
  Update(value *interface {})
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
  self.observers.Add(obs)
}

func (self *BasicObservable) RemoveObserver(obs Observer) {
  self.observers.Remove(obs)
}

func (self *BasicObservable) NotifyObservers(payload *interface {}) {
  for obs := range self.observers.Iter() {
    obs.(Observer).Update(payload)
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

func (self *BasicObserver) Observe(obs Observable) {
  obs.AddObserver(self)
}

func (self *BasicObserver) Unobserver(obs Observable) {
  obs.RemoveObserver(self)
}

func (self *BasicObserver)  Update(value *interface {}) {
  self.hdlr.Update(value)
}
