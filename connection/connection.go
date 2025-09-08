package connection

import (
	"context"
	"sync"

	"github.com/s84662355/simple-message/protocol"
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

func (C *Connection) StoreProperty(k string, v interface{}) {
	C.property.Store(k, v)
}

func (C *Connection) LoadProperty(k string) (interface{}, bool) {
	return C.property.Load(k)
}

func (C *Connection) DeleteProperty(key string) {
	C.property.Delete(key)
}

func (C *Connection) ClearProperty() {
	C.property.Clear()
}

func (C *Connection) CompareAndDeleteProperty(key, old any) (deleted bool) {
	return C.property.CompareAndDelete(key, old)
}

func (C *Connection) CompareAndSwapProperty(key, old, new any) (swapped bool) {
	return C.property.CompareAndSwap(key, old, new)
}

func (C *Connection) LoadAndDeleteProperty(key any) (value any, loaded bool) {
	return C.property.LoadAndDelete(key)
}

func (C *Connection) LoadOrStoreProperty(key, value any) (actual any, loaded bool) {
	return C.property.LoadOrStore(key, value)
}

func (C *Connection) RangeProperty(f func(key, value any) bool) {
	C.property.Range(f)
}

func (C *Connection) SwapProperty(key, value any) (previous any, loaded bool) {
	return C.property.Swap(key, value)
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
