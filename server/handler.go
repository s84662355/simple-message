package server

import (
	"context"
	"net"
	"sync"

	"github.com/s84662355/simple-tcp-message/connection"
)

func (m *Server) tcpAccept(ctx context.Context) {
	m.tcpListenerAccept(ctx, m.tcpListener)
}

func (m *Server) tcpListenerAccept(ctx context.Context, tcpListener net.Listener) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	for m.isRun.Load() {
		// 接受客户端的连接
		conn, err := tcpListener.Accept()
		if err != nil {
			continue
		}

		if m.connCount.Add(1) > m.maxConnCount {
			m.connCount.Add(-1)
			conn.Close()
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer conn.Close()
			defer m.connCount.Add(-1)
			if c, ok := conn.(*net.TCPConn); ok {
				c.SetKeepAlive(true)
			}

			m.handlerTcpConn(ctx, conn)
		}()

	}
}

func (m *Server) handlerTcpConn(ctx context.Context, conn net.Conn) {
	handlerManager := connection.NewHandlerManager(
		conn,
		m.handler,
		m.maxDataLen,
		m.connectedBegin,
	)
	defer func() {
		<-handlerManager.Stop()
	}()

	select {
	case <-ctx.Done():
		return
	case <-handlerManager.Ctx().Done():
		m.connErr(handlerManager.GetConnection(), handlerManager.Err())
		return
	}
}
