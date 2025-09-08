package connection

import (
	"context"
	"errors"
	"io"
)

type Conn interface {
	io.ReadWriteCloser
}

var (
	ErrIsClose     = errors.New("已关闭")
	ErrKeyNotFound = errors.New("属性不存在")
)

type ConnectedBegin func(ctx context.Context, conn *Connection)
