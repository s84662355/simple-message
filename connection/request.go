package connection

type IRequest interface {
	GetConnection() *Connection
	GetData() []byte
	GetMsgID() uint32
}

type Request struct {
	conn  *Connection
	data  []byte
	msgID uint32
}

func (m *Request) GetConnection() *Connection {
	return m.conn
}

func (m *Request) GetData() []byte {
	return m.data
}

func (m *Request) GetMsgID() uint32 {
	return m.msgID
}
