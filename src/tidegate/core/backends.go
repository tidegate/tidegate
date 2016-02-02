package core

import (
)

type Backend interface {
  Start() (err error)
  Stop() (err error)
}
