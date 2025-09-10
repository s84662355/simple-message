package client

import (
	"context"
	"errors"
	"maps"

	"github.com/s84662355/simple-message/connection"
)

var ErrIsClose = errors.New("已关闭")

type Client struct {
	handler    map[uint32]connection.Handler
	ctx        context.Context
	cancel     context.CancelFunc
	maxDataLen uint32
	action     Action
	done       chan struct{}
	msgChan    chan *MessageBody
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
	c.msgChan = make(chan *MessageBody)

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

func (c *Client) SendMsg(MsgID uint32, Data []byte) error {
	return c.sendMsgContext(context.Background(), MsgID, Data)
}

func (c *Client) SendMsgContext(ctx context.Context, MsgID uint32, Data []byte) error {
	return c.sendMsgContext(ctx, MsgID, Data)
}

func (c *Client) sendMsgContext(ctx context.Context, MsgID uint32, Data []byte) error {
	m := NewMessageBody(ctx, MsgID, Data)
	select {
	case <-c.ctx.Done():
		return ErrIsClose
	case <-ctx.Done():
		return ctx.Err()
	case c.msgChan <- m:
		<-m.ackChan
		return m.err
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

		for {
			select {
			case msg := <-c.msgChan:
				msg.AckMessage(func() error {
					ctx, id, data := msg.GetMessage()
					return handlerManager.GetConnection().SendMsgContext(ctx, id, data)
				})
			case <-c.ctx.Done():
				return
			case <-handlerManager.Ctx().Done():
				return
			}
		}

	}
}
