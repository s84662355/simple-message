package server

import (
	"github.com/s84662355/simple-message/connection"
)

type Listener interface {
	Accept() (connection.Conn, error)
	Close() error
}
