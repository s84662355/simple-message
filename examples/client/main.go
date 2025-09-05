package main

import (
	"fmt"

	"github.com/s84662355/simple-tcp-message/client"
	"github.com/s84662355/simple-tcp-message/connection"
)

type Handler1 struct{}

func (h *Handler1) Handle(request connection.IRequest) {
	// request.GetData()
	//	request.GetMsgID()
}

func main() {
	handler := map[uint32]connection.Handler{
		1: &Handler1{},
	}

	client.NewClient(
		"127.0.0.1:2000",
		handler,
		1024*1024,
		func(err error) string {
			return "127.0.0.1:2000"
		},
		func(conn *connection.Connection, err error) string {
			fmt.Println(conn)
			return "127.0.0.1:2000"
		},
		func(conn *connection.Connection) {
			fmt.Println(conn)
		},
	)

	select {}
}
