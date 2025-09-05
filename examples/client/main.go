package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/s84662355/simple-tcp-message/client"
	"github.com/s84662355/simple-tcp-message/connection"
)

type Handler1 struct{}

func (h *Handler1) Handle(request connection.IRequest) {
	fmt.Println(request.GetMsgID(), string(request.GetData()))
	// request.GetData()
	//	request.GetMsgID()
}

func main() {
	handler := map[uint32]connection.Handler{
		1: &Handler1{},
	}

	c := client.NewClient(
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
	defer func() {
		<-c.Stop()
	}()
	// 2. 设置信号监听通道
	signalChan := make(chan os.Signal, 1) // 缓冲大小为1的信号通道

	// 注册要监听的信号：
	// - SIGINT (Ctrl+C中断)
	// - SIGTERM (终止信号)
	// - Kill信号 (强制终止)
	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Kill,
	)
	<-signalChan // 当接收到上述任意信号时继续执行
}
