package connection

import (
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
