package client

import (
	"context"
	"maps"

	"github.com/s84662355/simple-tcp-message/connection"
)

type DialContext func(context.Context) (connection.Conn, error)

type Client struct {
	dialContext    DialContext
	handler        map[uint32]connection.Handler
	ctx            context.Context
	cancel         context.CancelFunc
	maxDataLen     uint32
	dialErr        func(ctx context.Context, err error) DialContext
	connErr        func(ctx context.Context, conn *connection.Connection, err error) DialContext
	connectedBegin func(ctx context.Context, conn *connection.Connection)
	done           chan struct{}
}

func NewClient(
	dialContext DialContext,
	handler map[uint32]connection.Handler,
	maxDataLen uint32,
	dialErr func(ctx context.Context, err error) DialContext,
	connErr func(ctx context.Context, conn *connection.Connection, err error) DialContext,
	connectedBegin func(ctx context.Context, conn *connection.Connection),
) *Client {
	c := &Client{
		handler:        maps.Clone(handler),
		dialErr:        dialErr,
		connErr:        connErr,
		connectedBegin: connectedBegin,
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.done = make(chan struct{})
	c.dialContext = dialContext
	go func() {
		defer close(c.done)
		c.start()
		c.cancel()
	}()

	return c
}

func (c *Client) Stop() <-chan struct{} {
	c.cancel()
	return c.done
}

func (c *Client) start() {
	for c.dialContext != nil {
		select {
		case <-c.ctx.Done():
			return
		default:

		}
		c.dial()
	}
}

func (c *Client) dial() {
	if conn, err := c.dialContext(c.ctx); err != nil {
		c.dialContext = c.dialErr(c.ctx, err)
		return
	} else {
		defer conn.Close()
		handlerManager := connection.NewHandlerManager(
			conn,
			c.handler,
			c.maxDataLen,
			c.connectedBegin,
		)
		defer func() {
			<-handlerManager.Stop()
			c.dialContext = c.connErr(c.ctx, handlerManager.GetConnection(), handlerManager.Err())
		}()

		select {
		case <-c.ctx.Done():
			return
		case <-handlerManager.Ctx().Done():
			return
		}
	}
}
