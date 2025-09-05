package connection

import (
	"context"
	"errors"
	"sync"

	"github.com/s84662355/simple-tcp-message/protocol"
)

var (
	ErrIsClose     = errors.New("已关闭")
	ErrKeyNotFound = errors.New("属性不存在")
)

type Connection struct {
	msgChan  chan *MessageBody
	ctx      context.Context
	cancel   context.CancelFunc
	property sync.Map
}

func NewConnection() (*Connection, <-chan *MessageBody) {
	C := &Connection{
		msgChan: make(chan *MessageBody),
	}
	C.ctx, C.cancel = context.WithCancel(context.Background())
	return C, C.msgChan
}

func (C *Connection) SetProperty(k string, v interface{}) {
	C.property.Store(k, v)
}

func (C *Connection) GetProperty(k string) (interface{}, error) {
	if v, ok := C.property.Load(k); ok {
		return v, nil
	}
	return nil, ErrKeyNotFound
}

func (C *Connection) RemoveProperty(key string) {
	C.property.Delete(key)
}

func (C *Connection) SendMsgContext(ctx context.Context, MsgID uint32, Data []byte) error {
	return C.sendMsg(ctx, MsgID, Data)
}

func (C *Connection) SendMsg(MsgID uint32, Data []byte) error {
	return C.sendMsg(context.TODO(), MsgID, Data)
}

func (C *Connection) sendMsg(ctx context.Context, MsgID uint32, Data []byte) error {
	m := &MessageBody{
		message: &protocol.Message{
			MsgID: MsgID,
			Data:  Data,
		},
		ackChan: make(chan struct{}),
	}
	select {
	case <-C.ctx.Done():
		return ErrIsClose
	case <-ctx.Done():
		return ctx.Err()
	case C.msgChan <- m:
		select {
		case <-C.ctx.Done():
			if !m.status.CompareAndSwap(false, true) {
				<-m.ackChan
				return m.err
			}
			return ErrIsClose
		case <-ctx.Done():
			if !m.status.CompareAndSwap(false, true) {
				<-m.ackChan
				return m.err
			}
			return ctx.Err()
		case <-m.ackChan:
			return m.err
		}
	}
}

func (C *Connection) Close() {
	C.cancel()
}

func (C *Connection) Ctx() context.Context {
	return C.ctx
}
