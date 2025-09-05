# simple-tcp-message 代码库说明

## 项目概述

`simple-tcp-message` 是一个基于 Go 语言实现的轻量级 TCP 消息通信库，提供了客户端和服务器端的基础框架，支持自定义消息处理、连接管理、消息编解码等功能。该库通过模块化设计，简化了 TCP 通信中的消息处理流程，适用于构建简单的基于 TCP 的通信应用。


## 核心包结构

### 1. `protocol` 包
负责消息的协议定义与编解码，定义了消息格式和数据传输规则。

- **`Message` 结构体**：表示一个完整的消息，包含消息 ID（`MsgID`）和消息数据（`Data`）。
  ```go
  type Message struct {
      MsgID uint32  // 消息唯一标识
      Data  []byte  // 消息内容
  }
  ```

- **`Decoder` 结构体**：处理消息的序列化（`Marshal`）和反序列化（`Unmarshal`），基于固定协议格式：
  - 协议头包含 4 字节 `MsgID` + 4 字节数据长度（`DataSize`）
  - 支持最大数据长度限制，避免超大消息导致的问题
  - 核心方法：
    - `Unmarshal(conn io.Reader) (*Message, error)`：从连接读取并解析消息
    - `Marshal(conn io.Writer, MsgID uint32, data []byte) error`：将消息编码并写入连接


### 2. `connection` 包
封装 TCP 连接的核心逻辑，包括连接管理、消息收发、消息处理等。

- **`Connection` 结构体**：管理单个 TCP 连接，提供：
  - 消息发送（`SendMsg`/`SendMsgContext`）：支持带上下文的消息发送，支持超时控制
  - 连接属性存储（`SetProperty`/`GetProperty`）：用于存储连接相关的附加信息
  - 连接关闭（`Close`）：通过 context 管理连接生命周期

- **`Handler` 接口**：定义消息处理规范，需实现 `Handle(request IRequest)` 方法，用于处理指定 `MsgID` 的消息。

- **`HandlerManager` 结构体**：协调连接的读写和消息处理，内部包含：
  - 读协程：从 TCP 连接读取数据并解析为 `Message`，存入队列
  - 写协程：从消息通道读取待发送消息，编码后写入 TCP 连接
  - 队列消费协程：从队列中获取消息，调用对应 `Handler` 处理


### 3. `client` 包
实现 TCP 客户端功能，负责与服务器建立连接、重连及连接管理。

- **`Client` 结构体**：
  - 通过 `NewClient` 创建客户端，需指定服务器地址、消息处理器、最大数据长度等参数
  - 自动重连机制：连接断开后会尝试重新连接
  - 生命周期管理：`Start` 启动客户端，`Stop` 停止客户端并释放资源


### 4. `server` 包
实现 TCP 服务器功能，负责监听端口、接受客户端连接及连接管理。

- **`Server` 结构体**：
  - 通过 `NewServer` 创建服务器，需指定监听地址、消息处理器、最大连接数等参数
  - 支持多协程接受连接（`Start(acceptAmount int)` 可指定 accept 协程数量）
  - 连接限制：通过 `maxConnCount` 控制最大并发连接数
  - 生命周期管理：`Stop` 停止服务并关闭所有连接


### 5. `nqueue` 包
提供一个并发安全的泛型队列，用于缓冲消息，支持阻塞/非阻塞读写，提升消息处理的并发性能。

- **`NQueue[T]` 结构体**：
  - 核心方法：`Enqueue`（入队）、`Dequeue`（非阻塞出队）、`DequeueWait`（阻塞出队）
  - 支持通过 `DequeueFunc` 持续消费队列元素
  - 线程安全：使用读写锁和条件变量保证并发安全


## 示例用法

### 服务器端示例
```go
// 定义消息处理器
type Handler1 struct{}
func (h *Handler1) Handle(request connection.IRequest) {
    // 处理 MsgID=1 的消息
    fmt.Println("Received message:", request.GetMsgID(), string(request.GetData()))
}

func main() {
    // 创建监听器
    l, _ := net.Listen("tcp", ":2000")
    // 注册消息处理器（MsgID=1 对应 Handler1）
    handler := map[uint32]connection.Handler{1: &Handler1{}}
    // 创建服务器
    s := server.NewServer(
        l,
        handler,
        1024*1024,  // 最大数据长度
        1024,       // 最大连接数
        func(conn *connection.Connection, err error) {
            // 连接错误回调
            fmt.Println("Connection error:", err)
        },
        func(conn *connection.Connection) {
            // 连接建立回调
            fmt.Println("New connection:", conn)
            // 向客户端发送欢迎消息
            conn.SendMsg(1, []byte("Welcome!"))
        },
    )
    // 启动服务器（16 个 accept 协程）
    s.Start(16)
    defer s.Stop()

    // 等待退出信号
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan
}
```

### 客户端示例
```go
// 定义消息处理器
type Handler1 struct{}
func (h *Handler1) Handle(request connection.IRequest) {
    // 处理服务器发送的 MsgID=1 的消息
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
    defer c.Stop()

    // 等待退出信号
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan
}
```


## 核心功能特点

1. **简单易用**：封装了 TCP 通信的底层细节，提供简洁的 API 接口
2. **协议灵活**：通过 `protocol` 包定义消息格式，支持自定义消息 ID 和数据
3. **并发安全**：使用读写锁、原子操作、条件变量等保证高并发场景下的稳定性
4. **连接管理**：支持连接属性存储、连接生命周期管理、最大连接数限制
5. **自动重连**：客户端断开连接后会自动尝试重连
6. **消息缓冲**：通过 `nqueue` 包的队列实现消息缓冲，避免消息处理阻塞


## 总结

`simple-tcp-message` 适合用于构建简单的 TCP 通信应用，通过模块化设计降低了 TCP 消息处理的复杂度。该库提供了消息编解码、连接管理、并发处理等基础功能，可根据实际需求扩展消息处理器和业务逻辑。