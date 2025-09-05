package main

import (
	"fmt"
	"net"

	"github.com/s84662355/simple-tcp-message/connection"
	"github.com/s84662355/simple-tcp-message/server"
)

type Handler1 struct{}

func (h *Handler1) Handle(request connection.IRequest) {
	// request.GetData()

	//	request.GetMsgID()
}

func main() {
	l, _ := net.Listen("tcp", ":2000")
	handler := map[uint32]connection.Handler{
		1: &Handler1{},
	}
	ss := server.NewServer(
		l,
		handler,
		1024*1024,
		func(conn *connection.Connection, err error) {
			fmt.Println(conn)
		},
		func(conn *connection.Connection) {
			fmt.Println(conn)
		},
	)

	ss.Start(16)
	select {}
}
