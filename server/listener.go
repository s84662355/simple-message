package server

import (
	"github.com/s84662355/simple-tcp-message/connection"
)

type Listener interface {
	Accept() (connection.Conn, error)
	Close() error
}
