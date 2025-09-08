package client

import (
	"context"

	"github.com/s84662355/simple-message/connection"
)

type Action interface {
	DialContext(context.Context) (connection.Conn, error)                // 拨号函数
	DialErr(ctx context.Context, err error)                              // 拨号失败回调函数  拨号错误回调 - 返回新的拨号函数用于重连
	ConnErr(ctx context.Context, conn *connection.Connection, err error) // 连接断连回调函数
	ConnectedBegin(ctx context.Context, conn *connection.Connection)     // 拨号成功后开始处理回调函数
}
