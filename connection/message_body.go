package connection

import (
	"sync/atomic"

	"github.com/s84662355/simple-tcp-message/protocol"
)

type T func() error

type MessageBody struct {
	message *protocol.Message
	err     error
	ackChan chan struct{}
	status  atomic.Bool
}

func (m *MessageBody) GetMessage() *protocol.Message {
	return m.message
}

func (m *MessageBody) AckMessage(t T) {
	if m.status.CompareAndSwap(false, true) {
		m.err = t()
		close(m.ackChan)
	}
}
