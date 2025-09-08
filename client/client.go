package client

import (
	"context"
	"maps"

	"github.com/s84662355/simple-message/connection"
)

type Client struct {
	handler    map[uint32]connection.Handler
	ctx        context.Context
	cancel     context.CancelFunc
	maxDataLen uint32
	action     Action
	done       chan struct{}
}

func NewClient(
	handler map[uint32]connection.Handler,
	maxDataLen uint32,
	action Action,
) *Client {
	c := &Client{
		handler:    maps.Clone(handler),
		action:     action,
		maxDataLen: maxDataLen,
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.done = make(chan struct{})
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
	for c.action != nil {
		select {
		case <-c.ctx.Done():
			return
		default:

		}
		c.dial()
	}
}

func (c *Client) dial() {
	if conn, err := c.action.DialContext(c.ctx); err != nil {
		c.action.DialErr(c.ctx, err)
		return
	} else {
		defer conn.Close()
		handlerManager := connection.NewHandlerManager(
			conn,
			c.handler,
			c.maxDataLen,
			c.action.ConnectedBegin,
		)
		defer func() {
			<-handlerManager.Stop()
			c.action.ConnErr(c.ctx, handlerManager.GetConnection(), handlerManager.Err())
		}()

		select {
		case <-c.ctx.Done():
			return
		case <-handlerManager.Ctx().Done():
			return
		}
	}
}
