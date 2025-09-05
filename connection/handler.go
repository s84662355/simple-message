package connection

import (
	"context"
	"io"
	"maps"
	"sync"

	"github.com/s84662355/simple-tcp-message/nqueue"
	"github.com/s84662355/simple-tcp-message/protocol"
)

type Handler interface {
	Handle(request IRequest)
}

type HandlerManager struct {
	readWriteCloser io.ReadWriteCloser
	queue           nqueue.Queue[*protocol.Message]
	conn            *Connection
	msgChan         chan *protocol.Message
	handler         map[uint32]Handler
	decoder         protocol.Decoder
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	err             error
	errOnce         sync.Once
}

func NewHandlerManager(
	readWriteCloser io.ReadWriteCloser,
	handler map[uint32]Handler,
	maxDataLen uint32,
) *HandlerManager {
	h := &HandlerManager{
		readWriteCloser: readWriteCloser,
		handler:         maps.Clone(handler),
		queue:           nqueue.NewNQueue[*protocol.Message](),
		decoder:         protocol.NewDecoder(maxDataLen),
	}
	h.conn, h.msgChan = NewConnection()
	h.ctx, h.cancel = context.WithCancel(context.Background())

	h.wg.Add(3)
	go func() {
		defer h.wg.Done()
		defer h.stop()
		h.read()
	}()
	go func() {
		defer h.wg.Done()
		defer h.stop()
		h.send()
	}()
	go func() {
		defer h.wg.Done()
		defer h.stop()
		h.queueConsumer()
	}()

	return h
}

func (h *HandlerManager) GetConnection() *Connection {
	return h.conn
}

func (h *HandlerManager) Stop() {
	h.stop()
	h.wg.Wait()
}

func (h *HandlerManager) Ctx() context.Context {
	return h.ctx
}

func (h *HandlerManager) Err() error {
	return h.err
}

func (h *HandlerManager) stop() {
	h.conn.Close()
	h.queue.Close()
	h.readWriteCloser.Close()
	h.cancel()
}

func (h *HandlerManager) merr(err error) {
	h.errOnce.Do(func() {
		h.err = err
	})
}

func (h *HandlerManager) read() {
	for {
		if message, err := h.decoder.Unmarshal(h.readWriteCloser); err != nil {
			h.merr(err)
			return
		} else {
			h.queue.Enqueue(message)
		}
	}
}

func (h *HandlerManager) send() {
	var err error
	for {
		select {
		case m, ok := <-h.msgChan:
			if !ok {
				return
			}
			m.AckMessage(func() error {
				message := m.GetMessage()
				err = h.decoder.Marshal(h.readWriteCloser, message.MsgID, message.Data)
				return
			})

			if err != nil {
				h.merr(err)
				return
			}

		case <-h.ctx.Done():
			return
		}
	}
}

func (h *HandlerManager) queueConsumer() {
	for {
		if t, ok, isClose := h.queue.DequeueWait(); isClose {
			return
		} else if ok {
			if h, ok := h.handler[t.MsgID]; ok {
				r := &Request{
					conn:  h.conn,
					data:  t.Data,
					msgID: t.MsgID,
				}
				h.Handle(r)
			}
		}
	}
}
