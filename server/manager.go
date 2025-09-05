package server

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"github.com/s84662355/simple-tcp-message/connection"
)

type Server struct {
	tcpListener    net.Listener
	isRun          atomic.Bool
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	startOnce      sync.Once
	statusMu       sync.Mutex
	connErr        func(conn *connection.Connection, err error)
	connectedBegin func(conn *connection.Connection)
	handler        map[uint32]connection.Handler
	maxDataLen     uint32
	maxConnCount   int32
	connCount      atomic.Int32
}

func NewServer(
	tcpListener net.Listener,
	handler map[uint32]connection.Handler,
	maxDataLen uint32,
	maxConnCount int32,
	connErr func(conn *connection.Connection, err error),
	connectedBegin func(conn *connection.Connection),
) *Server {
	m := &Server{
		tcpListener:    tcpListener,
		connErr:        connErr,
		connectedBegin: connectedBegin,
		handler:        handler,
		maxDataLen:     maxDataLen,
		maxConnCount:   maxConnCount,
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// Start 启动代理服务的各个组件
func (m *Server) Start(acceptAmount int) {
	m.startOnce.Do(func() {
		m.statusMu.Lock()
		defer m.statusMu.Unlock()
		m.isRun.Store(true)
		m.wg.Add(acceptAmount)
		for i := 0; i < acceptAmount; i++ {
			go func() {
				defer m.wg.Done()
				m.tcpAccept(m.ctx)
			}()
		}
	})
}

// Stop 停止所有服务组件
func (m *Server) Stop() {
	m.statusMu.Lock()
	if !m.isRun.Load() {
		m.statusMu.Unlock()
		return
	}
	m.isRun.Store(false)
	m.statusMu.Unlock()

	m.tcpListener.Close()
	m.cancel()
	m.wg.Wait()
}
