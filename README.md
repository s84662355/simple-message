# simple-tcp-message 文档

## 项目概述

`simple-tcp-message` 是一个基于 Go 语言实现的轻量级 TCP 消息通信库，提供了客户端和服务器端的基础框架，支持自定义消息处理、连接管理、消息编解码等功能。该库通过模块化设计，简化了 TCP 通信中的消息处理流程，适用于构建简单的基于 TCP 的通信应用。

### 核心功能特点

1. **简单易用**：封装了 TCP 通信的底层细节，提供简洁的 API 接口
2. **协议灵活**：通过 `protocol` 包定义消息格式，支持自定义消息 ID 和数据
3. **并发安全**：使用读写锁、原子操作、条件变量等保证高并发场景下的稳定性
4. **连接管理**：支持连接属性存储、连接生命周期管理、最大连接数限制
5. **自动重连**：客户端断开连接后会自动尝试重连
6. **消息缓冲**：通过 `nqueue` 包的队列实现消息缓冲，避免消息处理阻塞

## 安装方法

使用 `go get` 命令安装：

```bash
go get github.com/s84662355/simple-tcp-message
```

## 快速开始

### 服务器端示例

```go
package main

import (
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"

    "github.com/s84662355/simple-tcp-message/connection"
    "github.com/s84662355/simple-tcp-message/server"
)

// 定义消息处理器（处理 MsgID=1 的消息）
type Handler1 struct{}

func (h *Handler1) Handle(request connection.IRequest) {
    fmt.Printf("收到消息 - ID: %d, 内容: %s
", request.GetMsgID(), string(request.GetData()))
    // 可以通过 request.GetConnection() 获取当前连接，进行回复等操作
}

func main() {
    // 创建 TCP 监听器
    listener, err := net.Listen("tcp", ":2000")
    if err != nil {
        fmt.Printf("监听失败: %v
", err)
        return
    }

    // 注册消息处理器（MsgID 与 Handler 映射）
    handlers := map[uint32]connection.Handler{
        1: &Handler1{}, // 处理 MsgID=1 的消息
    }

    // 创建服务器实例
    srv := server.NewServer(
        listener,
        handlers,
        1024*1024, // 最大数据长度（1MB）
        1024,      // 最大连接数
        // 连接错误回调
        func(conn *connection.Connection, err error) {
            fmt.Printf("连接错误: %v, 连接: %v
", err, conn)
        },
        // 连接建立回调
        func(conn *connection.Connection) {
            fmt.Println("新连接建立")
            // 向客户端发送欢迎消息（MsgID=1）
            conn.SendMsg(1, []byte("欢迎连接到服务器！"))
        },
    )

    // 启动服务器（16 个 accept 协程）
    srv.Start(16)
    defer srv.Stop()

    // 监听退出信号（Ctrl+C 等）
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, os.Kill)
    <-signalChan // 等待退出信号
    fmt.Println("服务器退出")
}
```

### 客户端示例

```go
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/s84662355/simple-tcp-message/client"
    "github.com/s84662355/simple-tcp-message/connection"
)

// 定义消息处理器（处理服务器发送的 MsgID=1 的消息）
type Handler1 struct{}

func (h *Handler1) Handle(request connection.IRequest) {
    fmt.Println("Received from server:", request.GetMsgID(), string(request.GetData()))
}

func main() {
    // 注册消息处理器（MsgID=1 对应 Handler1）
    handler := map[uint32]connection.Handler{1: &Handler1{}}
    // 创建客户端
    c := client.NewClient(
        "127.0.0.1:2000",  // 服务器地址
        handler,
        1024*1024,         // 最大数据长度
        func(err error) string {
            // 拨号错误回调（返回重连地址）
            fmt.Println("Dial error:", err)
            return "127.0.0.1:2000"
        },
        func(conn *connection.Connection, err error) string {
            // 连接错误回调（返回重连地址）
            fmt.Println("Connection error:", err)
            return "127.0.0.1:2000"
        },
        func(conn *connection.Connection) {
            // 连接建立回调
            fmt.Println("Connected to server")
        },
    )
    defer func() {
        <-c.Stop()
    }()

    // 等待退出信号
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan
}
```

## 核心模块

### 1. protocol 包（协议处理）

实现消息的编解码功能，定义了消息格式和协议规范。

#### 核心结构
- `Message`：消息结构体，包含 MsgID 和 Data 字段
- `Decoder`：消息编解码器

#### 主要方法
- `NewDecoder(maxDataLen uint32) *Decoder`：创建解码器实例
- `Unmarshal(conn io.Reader) (*Message, error)`：从连接读取并解析消息
- `Marshal(conn io.Writer, MsgID uint32, data []byte) error`：将消息编码并写入连接

### 2. connection 包（连接管理）

处理单个 TCP 连接的消息收发和生命周期管理。

#### 核心结构
- `Connection`：连接实例，管理连接属性和消息发送
- `Handler`：消息处理接口，需实现 Handle 方法
- `Request`：消息请求，包含连接、消息 ID 和数据

#### 主要方法
- `NewConnection() (*Connection, <-chan *MessageBody)`：创建连接实例
- `SetProperty(k string, v interface{})` / `GetProperty(k string) (interface{}, error)`：设置/获取连接属性
- `SendMsg(MsgID uint32, Data []byte) error`：发送消息

#### 消息处理流程
1. 从 TCP 连接读取数据并解析为 `Message`
2. 将消息存入缓冲队列（`nqueue`）
3. 消费队列消息，调用对应 `Handler` 处理

### 3. server 包（服务器端）

实现 TCP 服务器功能，负责监听端口、接受连接并管理。

#### 核心结构
- `Server`：服务器实例

#### 主要方法
- `NewServer(...) *Server`：创建服务器实例
- `Start(acceptAmount int)`：启动服务器（指定 accept 协程数量）
- `Stop()`：停止服务器

### 4. client 包（客户端）

实现 TCP 客户端功能，负责与服务器建立连接及重连。

#### 核心结构
- `Client`：客户端实例

#### 主要方法
- `NewClient(...) *Client`：创建客户端实例
- `Stop() <-chan struct{}`：停止客户端

### 5. nqueue 包（消息队列）

提供并发安全的消息队列，用于缓冲消息，避免处理阻塞。

#### 核心结构
- `NQueue[T]`：泛型队列实现

#### 主要方法
- `Enqueue(v T) error`：入队操作
- `Dequeue() (t T, ok bool, isClose bool)`：非阻塞出队
- `DequeueWait() (t T, ok bool, isClose bool)`：阻塞出队
- `Close()`：关闭队列

## 总结

`simple-tcp-message` 适合用于构建简单的 TCP 通信应用，通过模块化设计降低了 TCP 消息处理的复杂度。该库提供了消息编解码、连接管理、并发处理等基础功能，可根据实际需求扩展消息处理器和业务逻辑。