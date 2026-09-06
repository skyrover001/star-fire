# P3 实施规格：ControlConn 单一 Writer Goroutine（根治并发写 panic）

> 上游分析：`docs/architecture-analysis.md` §5 并发风险 #2、§8 建议 #4
> 状态：待实施 | 依赖：无（独立于 P0/P1/P2） | 回滚：`CONTROL_WRITER_ENABLED=false`（保留旧 mutex 路径）
> 核心原则：gorilla/websocket 的 `Conn` **不允许并发写**。当前 `ControlConnMutex` 靠"每个写点记得加锁"来串行化，但写点分散在 5 个文件、至少 7 处，漏一处就 panic。本方案改为**每个控制连接一个专用 writer goroutine + 有缓冲 channel**，所有写都投递到 channel，由唯一 goroutine 串行执行，从机制上杜绝并发写。

## 0. 现状代码事实（已核实）

| 事实 | 位置 |
|---|---|
| `Client.ControlConnMutex sync.Mutex` 定义 | `internal/models/client.go` L44 |
| keepalive ping 写（持锁） | `internal/service/client.go` L52–68 |
| `notifyLatencyExceeded` 写（持锁） | `internal/models/server.go` L315–335 |
| `MODEL_PRICE_UPDATE` 写（持锁） | `internal/models/server.go` L1665–1680 |
| chat 请求下发 MESSAGE（**未持锁**） | `internal/service/chat.go` L310 |
| abort CLOSE（持锁） | `internal/service/chat.go` L544 |
| INCOME 收益通知 goroutine（**未持锁**，panic 点） | `internal/service/chat.go` L872 |
| embedding 请求下发（**未持锁**） | `internal/service/embedding.go` L75 |
| 多格式 MESSAGE 下发（**未持锁**） | `internal/service/format_adapter.go` L307 |

**panic 根因**：`recordTokenUsage`（chat.go L855–857）在锁内拷贝 `conn := chatClient.ControlConn` 后**释放锁**，随后 goroutine（L865–887）在**无锁**状态下 `conn.WriteJSON(INCOME)`。该 goroutine 每请求 spawn 一个，与 keepalive ping、chat 下发等并发写同一 `Conn` → gorilla/websocket `panic: concurrent write to websocket connection`（`messageWriter.flushFrame` conn.go:617）。

## 1. 设计目标

1. **根治**：任何写点都不再需要手动加锁，从机制上杜绝并发写。
2. **收敛**：所有 `ControlConn` 写统一走一个入口，新增写点不会漏。
3. **兼容**：保留 `ControlConnMutex` 字段（外部引用多），但写路径改为 writer goroutine。
4. **可回滚**：开关控制，出问题可切回旧 mutex 路径。

## 2. 改动一：config（config/config.go）

```go
ControlWriterEnabled bool // CONTROL_WRITER_ENABLED，默认 true
ControlWriterBufSize int  // CONTROL_WRITER_BUF_SIZE，默认 64（channel 缓冲）
```

## 3. 改动二：Client 结构（internal/models/client.go）

在 `Client` 上新增 writer 字段（不删 `ControlConnMutex`，保持向后兼容）：

```go
type Client struct {
    // ... 现有字段 ...

    ControlConnMutex sync.Mutex // 保留：兼容旧路径 + 保护 ControlConn 指针读写

    // P3: 单一 writer goroutine
    writeCh   chan public.WSMessage // 控制通道写队列（nil = 未启用 writer）
    writeDone chan struct{}         // writer goroutine 退出信号
    writeOnce sync.Once             // 保证只启动一个 writer
}
```

> 说明：`ControlConn` 指针本身的读写仍需 `ControlConnMutex` 保护（断线时 `handleClientMessages` 会置 nil），但**实际的 `WriteJSON` 不再在调用方持锁**，而是投递到 `writeCh`。

## 4. 改动三：writer goroutine 生命周期（internal/models/client.go）

### 4.1 启动

```go
// StartControlWriter 启动该 client 控制连接的单一 writer goroutine。
// 幂等：多次调用只启动一个。
func (c *Client) StartControlWriter() {
    if !configs.Config.ControlWriterEnabled {
        return
    }
    c.writeOnce.Do(func() {
        c.writeCh = make(chan public.WSMessage, configs.Config.ControlWriterBufSize)
        c.writeDone = make(chan struct{})
        go c.controlWriterLoop()
    })
}

func (c *Client) controlWriterLoop() {
    defer close(c.writeDone)
    for msg := range c.writeCh {
        c.ControlConnMutex.Lock()
        conn := c.ControlConn
        c.ControlConnMutex.Unlock()
        if conn == nil {
            continue // 连接已断，丢弃
        }
        if err := conn.WriteJSON(msg); err != nil {
            log.Printf("control writer: write to client %s failed: %v", c.ID, err)
        }
    }
}
```

### 4.2 投递（所有写点统一入口）

```go
// SendControl 向控制连接投递一条消息（非阻塞，缓冲满则丢弃并告警）。
// 所有 ControlConn 写都应改走此方法。
func (c *Client) SendControl(msg public.WSMessage) {
    if c.writeCh == nil {
        // 未启用 writer：回退旧 mutex 路径（兼容/回滚）
        c.ControlConnMutex.Lock()
        defer c.ControlConnMutex.Unlock()
        if c.ControlConn != nil {
            _ = c.ControlConn.WriteJSON(msg)
        }
        return
    }
    select {
    case c.writeCh <- msg:
    default:
        // 缓冲满：丢弃并告警，避免阻塞调用方（控制消息可丢，业务消息走响应连接）
        log.Printf("control writer: buffer full for client %s, dropping msg type=%s", c.ID, msg.Type)
    }
}
```

### 4.3 关闭

```go
// StopControlWriter 关闭 writer goroutine（连接断开时调用）。
func (c *Client) StopControlWriter() {
    if c.writeCh == nil {
        return
    }
    close(c.writeCh)
    <-c.writeDone
    c.writeCh = nil
}
```

> **注意**：`StopControlWriter` 必须在 `ControlConn` 置 nil / `Close()` **之后**调用，且要保证没有其它 goroutine 还在向 `writeCh` 投递（否则 `close` 后投递会 panic）。建议在 `handleClientMessages` 的断线分支统一处理（见 §6）。

## 5. 改动四：迁移所有写点

将下表所有 `ControlConn.WriteJSON(...)` 改为 `client.SendControl(...)`：

| 位置 | 原写法 | 改为 |
|---|---|---|
| `client.go` L52–68 keepalive | 持锁 `WriteJSON(KEEPALIVE)` | `client.SendControl(KEEPALIVE)` |
| `server.go` L315–335 notifyLatencyExceeded | 持锁 `WriteJSON(LATENCY_EXCEEDED)` | `c.SendControl(LATENCY_EXCEEDED)` |
| `server.go` L1665–1680 price update | 持锁 `WriteJSON(MODEL_PRICE_UPDATE)` | `client.SendControl(MODEL_PRICE_UPDATE)` |
| `chat.go` L310 MESSAGE 下发 | 无锁 `WriteJSON(MESSAGE)` | `client.SendControl(MESSAGE)` |
| `chat.go` L544 abort | 持锁 `WriteJSON(CLOSE)` | `client.SendControl(CLOSE)` |
| `chat.go` L872 INCOME goroutine | 无锁 `WriteJSON(INCOME)` | `chatClient.SendControl(INCOME)` |
| `embedding.go` L75 | 无锁 `WriteJSON(EMBEDDING_REQUEST)` | `client.SendControl(EMBEDDING_REQUEST)` |
| `format_adapter.go` L307 | 无锁 `WriteJSON(MESSAGE)` | `client.SendControl(MESSAGE)` |

**关键点**：
- `chat.go` L872 的 INCOME goroutine 改为 `chatClient.SendControl(INCOME)` 后，**不再需要**在 goroutine 里持锁，也无需先拷贝 `conn`。`recordTokenUsage` 里那段"锁内拷贝 conn"可以删掉，直接 `chatClient.SendControl(...)`。
- 所有写点统一收敛到 `SendControl`，新增写点只需调 `SendControl`，不会漏锁。

## 6. 改动五：断线清理（internal/service/client.go handleClientMessages）

当前断线分支（`ReadJSON` 出错时）只置 `ControlConn = nil`。需补充 writer 关闭：

```go
// 断线分支（ReadJSON err）
client.ControlConnMutex.Lock()
client.ControlConn = nil
client.ControlConnMutex.Unlock()
client.StopControlWriter() // P3: 关闭 writer goroutine
client.Status = "offline"
return
```

> 顺序要求：先置 `ControlConn = nil`（writer loop 读到 nil 会丢弃后续消息），再 `StopControlWriter`（close channel 并等 goroutine 退出）。这样 writer loop 不会在关闭后还尝试写已断开的连接。

## 7. 改动六：启动时机

在 client 注册/上线时调用 `StartControlWriter`。找到 client 建立控制连接的位置（`RegisterModel` / 上线流程），在 `ControlConn` 赋值后调用：

```go
client.ControlConnMutex.Lock()
client.ControlConn = conn
client.ControlConnMutex.Unlock()
client.StartControlWriter() // P3
```

## 8. 边界与取舍

| 场景 | 处理 |
|---|---|
| 缓冲满 | 丢弃 + 告警。控制消息（KEEPALIVE/INCOME/LATENCY）可丢；业务消息（MESSAGE/EMBEDDING_REQUEST）走响应连接，不走此通道，不受影响 |
| 连接断开后投递 | writer loop 读到 `conn == nil` 直接丢弃，不 panic |
| 未启用 writer（回滚） | `SendControl` 回退旧 mutex 路径，行为与现状一致 |
| 多个 goroutine 同时投递 | channel 天然串行，无锁竞争 |
| `StopControlWriter` 与投递并发 | 需保证断线后不再投递（断线分支先置 nil 再 close）；若确有投递方，需在 close 前用 `sync.WaitGroup` 或加 `writeClosed` 标志保护 |

## 9. 单元测试（internal/models/client_test.go 或 internal/service/client_test.go）

1. **TestSendControlSerializesWrites**：并发 N 个 goroutine 各投递 M 条消息，writer goroutine 串行写出，无 `concurrent write` panic（用 `-race` 跑）。
2. **TestSendControlBufferFullDrops**：填满缓冲后继续投递，不阻塞、不 panic，返回后缓冲未增长。
3. **TestSendControlFallbackNoWriter**：`ControlWriterEnabled=false` 时走旧 mutex 路径，行为正确。
4. **TestStopControlWriterIdempotent**：重复调用 `StopControlWriter` 不 panic（`writeCh == nil` 短路）。
5. **TestControlWriterNilConnDrops**：`ControlConn = nil` 后投递，writer loop 丢弃不 panic。
6. **TestStartControlWriterOnce**：多次调用 `StartControlWriter` 只启动一个 goroutine（`writeOnce`）。

## 10. 验收标准

1. 并发压测（多请求 + keepalive 同时进行）下**不再出现** `panic: concurrent write to websocket connection`。
2. 所有 `ControlConn.WriteJSON` 调用点收敛到 `SendControl`（`grep ControlConn.WriteJSON` 仅剩 `SendControl` 内部一处）。
3. `go test -race ./...` 通过，无数据竞争。
4. `CONTROL_WRITER_ENABLED=false` 时行为与改动前一致（回滚路径）。
5. 断线后 writer goroutine 正常退出，无 goroutine 泄漏（`go test` 的 goroutine 计数检查）。

## 11. 与方案1（补锁）对比

| 维度 | 方案1：补锁 | 方案2：writer goroutine（本方案） |
|---|---|---|
| 改动量 | 小（4 处补锁） | 中（新增 writer + 迁移 8 处写点） |
| 根治性 | 否（靠人肉记得加锁，仍会漏） | 是（机制上杜绝并发写） |
| 新增写点风险 | 高（忘加锁即 panic） | 低（统一走 SendControl） |
| 锁粒度 | 调用方持锁，易出现"取锁/写锁分离" | 单一 goroutine 串行，无锁竞争 |
| 断线竞态 | 需小心 `ControlConn` 置 nil 与写并发 | writer loop 统一处理 nil conn |
| 上线风险 | 低 | 中（需处理 channel 生命周期） |

**建议**：若线上正在 panic，先用方案1止血（改动最小）；本方案（方案2）作为根治重构，在止血后落地。
