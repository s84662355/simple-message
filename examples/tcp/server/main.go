package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/s84662355/simple-message/connection"
	"github.com/s84662355/simple-message/server"
)

// Handler1 消息处理器，用于处理MsgID=1的消息
type Handler1 struct{}

// Handle 实现消息处理接口
func (h *Handler1) Handle(request connection.IRequest) {
	// 可以通过以下方法获取消息数据和ID
	// data := request.GetData()
	// msgID := request.GetMsgID()

	// 此处可添加消息处理逻辑
}

// Listener 自定义监听器实现，包装net.Listener
type Listener struct {
	listener net.Listener
}

// Accept 实现Accept接口，返回读写关闭器
func (l *Listener) Accept() (connection.Conn, error) {
	return l.listener.Accept()
}

// Close 实现关闭接口
func (l *Listener) Close() error {
	return l.listener.Close()
}

type Action struct{}

// 连接错误回调
func (l *Action) ConnErr(ctx context.Context, conn *connection.Connection, err error) {
	fmt.Printf("连接错误: %v, 连接信息: %v\n", err, conn)
}

// 连接建立回调
func (l *Action) ConnectedBegin(ctx context.Context, conn *connection.Connection) {
	fmt.Printf("新连接建立: %v\n", conn)
	// 向新连接发送欢迎消息
	if err := conn.SendMsg(1, []byte("欢迎连接到服务器")); err != nil {
		fmt.Printf("发送消息失败: %v\n", err)
	}
}

func main() {
	// 创建TCP监听器，监听2000端口
	listener, err := net.Listen("tcp", ":2000")
	if err != nil {
		fmt.Printf("创建监听器失败: %v\n", err)
		return
	}
	defer listener.Close()

	// 注册消息处理器，MsgID=1对应Handler1
	handlers := map[uint32]connection.Handler{
		1: &Handler1{},
	}

	// 包装自定义监听器
	customListener := &Listener{
		listener: listener,
	}

	// 创建服务器实例
	srv := server.NewServer(
		customListener,
		handlers,
		1024*1024, // 最大数据长度 (1MB)
		1024,      // 最大连接数
		new(Action),
	)

	// 启动服务器，使用16个accept协程
	done := srv.Start(16)
	defer func() {
		// 停止服务器并等待完成
		srv.Stop()
		<-done
		fmt.Println("服务器已完全停止")
	}()

	// 设置信号监听，处理程序退出
	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		syscall.SIGINT,  // Ctrl+C中断
		syscall.SIGTERM, // 终止信号
		os.Kill,         // 强制终止
	)

	fmt.Println("服务器已启动，监听端口: 2000")
	fmt.Println("按Ctrl+C停止服务器")

	// 等待退出信号或服务器完成信号
	select {
	case <-done:
		fmt.Println("服务器正常退出")
	case sig := <-signalChan:
		fmt.Printf("收到退出信号: %v，正在停止服务器...\n", sig)
	}
}
