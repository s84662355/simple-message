package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

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
		1024,
		func(conn *connection.Connection, err error) {
			fmt.Println(conn)
		},
		func(conn *connection.Connection) {
			conn.SendMsg(1, []byte("diuauiodhsauidsa"))
			fmt.Println(conn)
		},
	)

	ss.Start(16)
	defer ss.Stop()
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
