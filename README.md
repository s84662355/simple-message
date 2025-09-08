
# simple-message 文档

## 项目概述
`simple-message` 是一个基于 Go 语言的轻量级 TCP 消息通信库，提供了客户端与服务器端的完整实现，支持消息编解码、连接管理、并发消息处理等核心功能。通过模块化设计，简化了 TCP 通信中的底层细节，让开发者可以专注于业务逻辑的实现。

### 主要特点
- **简单易用**：封装底层 TCP 细节，提供简洁 API，快速实现通信功能
- **并发安全**：使用读写锁和原子操作，确保高并发场景下的稳定性
- **自动重连**：客户端内置自动重连机制，提升连接可靠性

## 安装方法
使用 `go get` 命令安装：
go get github.com/s84662355/simple-message
## 快速开始

### 服务器端示例package main
```go
import (
    "context"
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"

    "github.com/s84662355/simple-message/connection"
    "github.com/s84662355/simple-message/server"
)

// Handler1 消息处理器，用于处理MsgID=1的消息
type Handler1 struct{}

// Handle 实现消息处理接口
func (h *Handler1) Handle(request connection.IRequest) {
    // 处理消息逻辑
    fmt.Printf("收到消息 - ID: %d, 内容: %s\n", 
        request.GetMsgID(), 
        string(request.GetData()))
}

// Listener 自定义监听器实现，包装net.Listener
type Listener struct {
    listener net.Listener
}

// Accept 实现Accept接口，返回读写关闭器
func (l *Listener) Accept() (connection.Conn, error) {
    return l.listener.Accept()
}

// Close 实现关闭接口
func (l *Listener) Close() error {
    return l.listener.Close()
}

type Action struct{}

// 连接错误回调
func (l *Action) ConnErr(ctx context.Context, conn *connection.Connection, err error) {
    fmt.Printf("连接错误: %v, 连接信息: %v\n", err, conn)
}

// 连接建立回调
func (l *Action) ConnectedBegin(ctx context.Context, conn *connection.Connection) {
    // 向新连接发送欢迎消息
    if err := conn.SendMsg(1, []byte("欢迎连接到服务器")); err != nil {
        fmt.Printf("发送消息失败: %v\n", err)
    }
    fmt.Printf("新连接建立: %v\n", conn)
}

func main() {
    // 创建TCP监听器，监听2000端口
    listener, err := net.Listen("tcp", ":2000")
    if err != nil {
        fmt.Printf("创建监听器失败: %v\n", err)
        return
    }
    defer listener.Close()

    // 注册消息处理器，MsgID=1对应Handler1
    handlers := map[uint32]connection.Handler{
        1: &Handler1{},
    }

    // 包装自定义监听器
    customListener := &Listener{
        listener: listener,
    }

    // 创建服务器实例
    srv := server.NewServer(
        customListener,
        handlers,
        1024*1024, // 最大数据长度 (1MB)
        1024,      // 最大连接数
        new(Action),
    )

    // 启动服务器，使用16个accept协程
    done := srv.Start(16)
    defer func() {
        // 停止服务器并等待完成
        srv.Stop()
        <-done
        fmt.Println("服务器已完全停止")
    }()

    // 设置信号监听，处理程序退出
    signalChan := make(chan os.Signal, 1)
    signal.Notify(
        signalChan,
        syscall.SIGINT,  // Ctrl+C中断
        syscall.SIGTERM, // 终止信号
        os.Kill,         // 强制终止
    )

    fmt.Println("服务器已启动，监听端口: 2000")
    fmt.Println("按Ctrl+C停止服务器")

    // 等待退出信号或服务器完成信号
    select {
    case <-done:
        fmt.Println("服务器正常退出")
    case sig := <-signalChan:
        fmt.Printf("收到退出信号: %v，正在停止服务器...\n", sig)
    }
}
```

 
### 客户端示例package main
```go
import (
    "context"
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"

    "github.com/s84662355/simple-message/client"
    "github.com/s84662355/simple-message/connection"
)

// Handler1 消息处理器，用于处理MsgID=1的消息
type Handler1 struct{}

// Handle 实现消息处理接口，打印收到的消息ID和内容
func (h *Handler1) Handle(request connection.IRequest) {
    fmt.Printf("收到消息 - ID: %d, 内容: %s\n",
        request.GetMsgID(),
        string(request.GetData()))
}

type Action struct{}

func (a *Action) DialContext(ctx context.Context) (connection.Conn, error) {
    var d net.Dialer
    // 连接到本地2000端口的TCP服务器
    conn, err := d.DialContext(ctx, "tcp", "127.0.0.1:2000")
    if err != nil {
        return nil, fmt.Errorf("连接失败: %w", err)
    }

    // 可选：配置TCP连接属性（如心跳机制）
    if tcpConn, ok := conn.(*net.TCPConn); ok {
        // 启用TCP保活机制
        if err := tcpConn.SetKeepAlive(true); err != nil {
            fmt.Printf("设置TCP保活失败: %v\n", err)
        }
    }

    return conn, nil
}

// 拨号错误回调
func (a *Action) DialErr(ctx context.Context, err error) {
    fmt.Printf("拨号错误: %v, 准备重连...\n", err)
}

func (a *Action) ConnErr(ctx context.Context, conn *connection.Connection, err error) {
    fmt.Printf("连接错误: %v, 连接信息: %v, 准备重连...\n", err, conn)
}

func (a *Action) ConnectedBegin(ctx context.Context, conn *connection.Connection) {
    fmt.Printf("成功连接到服务器: %v\n", conn)
    // 发送初始化消息
    if err := conn.SendMsg(1, []byte("客户端已连接")); err != nil {
        fmt.Printf("发送消息失败: %v\n", err)
    }
}

func main() {
    // 注册消息处理器，MsgID=1对应Handler1
    handlers := map[uint32]connection.Handler{
        1: &Handler1{},
    }

    // 创建客户端实例
    c := client.NewClient(
        handlers,  // 消息处理器映射
        1024*1024, // 最大数据长度 (1MB)
        new(Action),
    )

    // 确保程序退出时正确停止客户端
    defer func() {
        fmt.Println("正在关闭客户端...")
        // 等待客户端完全停止
        <-c.Stop()
        fmt.Println("客户端已完全停止")
    }()

    // 设置信号监听，处理程序退出
    signalChan := make(chan os.Signal, 1)
    signal.Notify(
        signalChan,
        syscall.SIGINT,  // Ctrl+C中断
        syscall.SIGTERM, // 终止信号
        os.Kill,         // 强制终止
    )

    fmt.Println("客户端已启动，正在连接到服务器 127.0.0.1:2000...")
    fmt.Println("按Ctrl+C停止客户端")

    // 等待退出信号
    <-signalChan
    fmt.Println("收到退出信号，正在停止客户端...")
}
```
## 核心模块说明

### 1. protocol 包（消息协议）
负责消息的编解码，定义了消息格式和传输规则。

#### 核心结构
- `Message`：消息载体type Message struct {
    MsgID uint32 // 消息唯一标识
    Data  []byte // 消息内容
}
- `Decoder`：消息编解码器
  - 协议格式：`4字节MsgID + 4字节数据长度 + 数据内容`
  - 支持最大数据长度限制（防止超大消息攻击）

#### 主要方法
- `Unmarshal(conn io.Reader) (*Message, error)`：从连接读取并解析消息
- `Marshal(conn io.Writer, MsgID uint32, data []byte) error`：将消息编码并写入连接

### 2. connection 包（连接管理）
封装 TCP 连接的核心逻辑，包括消息收发、连接属性管理等。

#### 核心结构
- `Connection`：连接实例
  - 支持消息发送（`SendMsg`/`SendMsgContext`）
  - 连接属性存储（`StoreProperty` / `LoadProperty`/ `DeleteProperty` / `ClearProperty` / `CompareAndDeleteProperty` / `CompareAndSwapProperty` / `LoadAndDeleteProperty` / `LoadOrStoreProperty` / `RangeProperty` / `SwapProperty` ）
  - 连接关闭（`Close`）

- `Handler`：消息处理接口type Handler interface {
    Handle(request IRequest)
}
- `IRequest`：消息请求接口
  - `GetConnection()`：获取当前连接
  - `GetData()`：获取消息数据
  - `GetMsgID()`：获取消息ID

#### 消息处理流程
1. 从 TCP 连接读取数据并解析为 `Message`
2. 将消息存入缓冲队列（`nqueue`）
3. 消费队列消息，调用对应 `Handler` 处理

### 3. server 包（服务器端）
实现 TCP 服务器功能，负责监听端口、接受连接并管理。

#### 核心结构
- `Server`：服务器实例
- `Listener`：服务器监听接口
- `Action`：服务器回调接口

#### 主要方法
- `NewServer(...) *Server`：创建服务器实例
- `Start(acceptAmount int) <-chan struct{}`：启动服务器（指定 accept 协程数量）
- `Stop()`：停止服务器并释放资源

#### 关键特性
- 支持最大连接数限制
- 多协程处理连接接入
- 连接生命周期管理
- 通过`Action`接口提供回调扩展

### 4. client 包（客户端）
实现 TCP 客户端功能，负责与服务器建立连接及重连。

#### 核心结构
- `Client`：客户端实例
- `Action`：客户端行为接口，定义了拨号、错误处理等方法

#### 主要方法
- `NewClient(handlers map[uint32]connection.Handler, maxDataLen uint32, action Action) *Client`：创建客户端实例
- `Start()`：启动客户端
- `Stop() <-chan struct{}`：停止客户端并释放资源
- `GetConnection() *connection.Connection`：获取当前连接

#### 关键特性
- 自动重连机制（通过`Action`接口实现）
- 连接状态监听
- 与服务器端统一的消息处理接口
- 上下文感知的连接管理

### 5. nqueue 包（消息队列）
提供并发安全的消息队列，用于缓冲消息，避免处理阻塞。

#### 核心结构
- `NQueue[T]`：泛型队列实现，支持任意类型的消息存储

#### 主要作用
- 平衡消息生产与消费速度差异
- 提供异步消息处理能力
- 支持带等待的消息出队操作
- 线程安全的队列操作

## API 参考

### Connection 接口

| 方法 | 说明 | 参数 | 返回值 |
|------|------|------|--------|
| SendMsg | 发送消息到对端 | msgID uint32, data []byte | error |
| SendMsgContext | 带上下文的消息发送 | ctx context.Context, msgID uint32, data []byte | error |
| StoreProperty | 设置连接属性 | key string, value interface{} | - |
| LoadProperty | 获取连接属性 | key string | interface{}, bool |
| DeleteProperty | 删除连接属性 | key string | - |
| ClearProperty | 清空连接属性 | - | - |
| CompareAndDeleteProperty | 在当前值与指定的旧值匹配时才删除条目 | key, old any | deleted bool |
| CompareAndSwapProperty | 检查键是否存在以及当前值是否与旧值相等 只有当条件满足时，才会将值更新为新值并返回 true | key, old, new any | swapped bool |
| LoadOrStoreProperty | 如果键存在，直接返回已有的值和true（表示值已存在） 如果键不存在，获取写锁并进行双重检查 | key, value any | actual any, loaded bool|
| RangeProperty | 接收一个函数参数 f，该函数接收键和值作为参数，并返回一个布尔值  | f func(key, value any) bool | - |
| SwapProperty | 先获取该键当前的旧值（如果存在） 无论键是否已存在，都将新值存储到该键中 返回旧值和一个布尔值  | key, value any | previous any, loaded bool |
| Close | 关闭连接 | - | - |

### Server 接口

| 方法 | 说明 | 参数 | 返回值 |
|------|------|------|--------|
| server.NewServer | 创建服务器实例 | 监听器、处理器等配置参数 | *Server |
| Server.Start | 启动服务器 | acceptAmount int | <-chan struct{} |
| Server.Stop | 停止服务器 | - | - |

### Client 接口

| 方法 | 说明 | 参数 | 返回值 |
|------|------|------|--------|
| client.NewClient | 创建客户端实例 | 处理器、最大数据长度、Action接口 | *Client |
| Client.Start | 启动客户端 | - | - |
| Client.Stop | 停止客户端 | - | <-chan struct{} |
| Client.GetConnection | 获取当前连接 | - | *connection.Connection |
