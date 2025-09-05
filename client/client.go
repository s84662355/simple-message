package client

import (
	"context"
	"io"
	"maps"
	"net"
	"sync"

	"github.com/s84662355/simple-tcp-message/connection"
)

type Client struct {
	address        string
	handler        map[uint32]connection.Handler
	ctx            context.Context
	cancel         context.CancelFunc
	startOnce      sync.Once
	maxDataLen     uint32
	dialErr        func(err error) string
	connErr        func(err error) string
	connectedBegin func(conn *connection.Connection)
	done           chan struct{}
}

func NewClient(
	address string,
	handler map[uint32]connection.Handler,
	maxDataLen uint32,
	dialErr func(err error) string,
	connErr func(err error) string,
	connectedBegin func(conn *connection.Connection),
) *Client {
	c := &Client{
		handler:        maps.Clone(handler),
		dialErr:        dialErr,
		connErr:        connErr,
		connectedBegin: connectedBegin,
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.startOnce.Do(func() {
		c.done = make(chan struct{})
		c.address = address
		go func() {
			defer close(c.done)
			c.start()
		}()
	})

	return c
}

func (c *Client) Stop() {
	c.cancel()
	for range c.done {
	}
}

func (c *Client) start() {
	for {
		d.dial()
		select {
		case <-c.ctx.Done():
			return
		default:

		}
	}
}

func (c *Client) dial() {
	var d net.Dialer
	if conn, err := d.DialContext(c.ctx, "tcp", c.address); err != nil {
		c.address = c.dialErr(err)
		return
	} else {
		handlerManager := connection.NewHandlerManager(
			conn,
			c.handler,
			c.maxDataLen,
		)
		done := make(chan struct{})
		defer func() {
			for range done {
				/* code */
			}
		}()
		go func() {
			defer close(done)
			conn := handlerManager.GetConnection()
			c.connectedBegin(conn)
		}()

		defer handlerManager.Stop()

		select {
		case <-c.ctx.Done():
			return nil
		case <-handlerManager.Ctx().Done():
			c.address = c.connErr(handlerManager.Err())
			return
		}
	}
}
