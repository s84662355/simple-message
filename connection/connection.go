package connection

import (
	"context"
	"sync"

	"github.com/s84662355/simple-tcp-message/protocol"
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
	m := NewMessageBody(&protocol.Message{
		MsgID: MsgID,
		Data:  Data,
	})
	select {
	case <-C.ctx.Done():
		return ErrIsClose
	case <-ctx.Done():
		return ctx.Err()
	case C.msgChan <- m:
		<-m.ackChan
		return m.err
	}
}

func (C *Connection) Close() {
	C.cancel()
}

func (C *Connection) Ctx() context.Context {
	return C.ctx
}
