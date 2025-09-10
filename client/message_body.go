package client

import (
	"context"
	"sync/atomic"
)

type T func() error

type MessageBody struct {
	ctx     context.Context
	err     error
	ackChan chan struct{}
	status  atomic.Bool
	data    []byte
	id      uint32
}

func NewMessageBody(
	ctx context.Context,
	id uint32,
	data []byte,
) *MessageBody {
	return &MessageBody{
		ctx:     ctx,
		data:    data,
		id:      id,
		ackChan: make(chan struct{}),
	}
}

func (m *MessageBody) GetMessage() (context.Context, uint32, []byte) {
	return m.ctx, m.id, m.data
}

func (m *MessageBody) Done() <-chan struct{} {
	return m.ackChan
}

func (m *MessageBody) AckMessage(t T) {
	if m.status.CompareAndSwap(false, true) {
		m.err = t()
		close(m.ackChan)
	}
}
