# P2 实施规格：会话亲和 + cache 命中反馈 + HRW 防抖

> 上游设计：`docs/scale-lb-roadmap-design.md` P2 章节
> 状态：P2 众包亲和已实施；§4B Direct 粘性待实施 | 依赖：P0（缓存分让亲和校验零 DB 成本；冷却检查复用） | 回滚：`AFFINITY_ENABLED=false`、`LB_HRW_SUBSET_SIZE=0`、`DIRECT_AFFINITY_ENABLED=false`
> 核心原则：路由决策用**确定性预测命中**（亲和键→原 client/后端），实测 `cached_tokens` 只做反馈校验与 tiebreak；CacheHitEMA **不进 perfScore 主权重**（防正反馈马太效应）。
> **Direct 粘性（改动四B）**：Direct 后端是 prefix-cache 最佳候选，多后端部署需会话粘性（一致性哈希 + 亲和写回），独立开关 `DIRECT_AFFINITY_ENABLED`。

## 0. 现状代码事实（已核实）

| 事实 | 位置 |
|---|---|
| `extractConversationKey(cr *public.CanonicalRequest) string` 已存在（读 `Extra.metadata.prompt_cache_key`，无则返回空） | `internal/service/format_adapter.go` L641 |
| chat 路径入口 `handleChatWithRetry`；成功分支在 `case public.MESSAGE, public.MESSAGE_STREAM:`（chat.go L302–306），此处调 `ResetFailures` 后进入 `handleChatResponseWithFirst` | `internal/service/chat.go` |
| 计费函数 `recordTokenUsage(..., cachedTokens int, clientID string, ...)`（L677）、`recordCanonicalUsage(...)`（format_adapter.go L698）——cached/input 都在这两处可得 | 已核实 |
| `LoadBalanceExcluding` 流程：Predicate 过滤 → `s.pick(model, eligible)`；HRW 子集插在两者之间 | `internal/models/server.go` |
| `pickSmart` 阶段1按 `perfScore×(1±jitter)` 降序排序取 Top-N | `server.go` |
| `Server.reasoning` 已有按会话 key 存储的先例（分片不重要，量小） | `server.go` L60 |
| 软限流需要 `effectiveMaxConnections(c)`（已存在，Server 方法） | `server.go` |
| `clientHealthy` 是包级函数，service 包不可见 ⇒ 亲和校验需 Server 暴露方法 | 设计约束 |
| gin Context 传值先例：`c.Set("reasoning_conv_key")` | format_adapter.go |

## 1. 改动一：config（config/config.go）

```go
AffinityEnabled       bool    // AFFINITY_ENABLED，默认 false
AffinityTTLMin        int     // AFFINITY_TTL_MIN，默认 20（分钟，滑动）
AffinityLeaseSec      int     // AFFINITY_LEASE_SEC，默认 60
AffinitySoftLimitRate float64 // AFFINITY_SOFT_LIMIT_RATE，默认 0.9
AffinityMinHitRate    float64 // AFFINITY_MIN_HIT_RATE，默认 0.2
AffinityMissK         int     // AFFINITY_MISS_K，默认 3
AffinityMaxEntries    int     // AFFINITY_MAX_ENTRIES，默认 100000（LRU 上限）
LBHRWSubsetSize       int     // LB_HRW_SUBSET_SIZE，默认 0（关闭）
```

## 2. 改动二：AffinityStore（新文件 internal/models/affinity.go）

```go
type AffinityEntry struct {
    ClientID   string // 众包 clientID 或 DirectBackend 的裸 backendID
    IsDirect   bool
    ExpireAt   int64 // unix ns，滑动 TTL
    LeaseUntil int64 // unix ns，0=无租约；粘住的目标临时不可用时保留等待回迁
    MissCount  int32 // 连续实测低命中次数
}

type AffinityStore interface {
    Get(key string) (AffinityEntry, bool)      // 过期即 miss（惰性删除）
    Touch(key, clientID string, isDirect bool) // 写入/续期；换目标时 MissCount/Lease 清零
    MarkLease(key string)                      // 目标临时不可用：设置 LeaseUntil=now+lease，不删
    LeaseExpired(key string) bool              // 无记录或 LeaseUntil 过期 → true
    RecordMiss(key string) int32               // MissCount++，返回新值（达到 K 由调用方 Delete）
    Delete(key string)
    Len() int                                  // 监控/测试
}

func NewMemoryAffinityStore(ttl, lease time.Duration, maxEntries int) AffinityStore
```

内存实现 `memoryAffinityStore`：

- 32 个分片，`shard = fnv1a(key) & 31`（`hash/fnv` 标准库，不引新依赖），每片 `sync.Mutex + map[string]*entryNode + container/list`（LRU：Get/Touch 移到队头，超 `maxEntries/32` 逐出队尾）。
- 后台 janitor goroutine：每 60s 扫描删除过期项（`NewMemoryAffinityStore` 内启动；提供 `Close()` 停止，测试用）。
- 所有方法 O(1)；`-race` 干净是硬性要求。
- **P4 换 Redis 实现时接口不变**——因此接口里不出现任何内存实现细节。

Server 挂载：

```go
// server.go Server struct
Affinity AffinityStore
// NewServer():
if configs.Config.AffinityEnabled {
    server.Affinity = NewMemoryAffinityStore(
        time.Duration(configs.Config.AffinityTTLMin)*time.Minute,
        time.Duration(configs.Config.AffinityLeaseSec)*time.Second,
        configs.Config.AffinityMaxEntries)
}
```

## 3. 改动三：亲和键提取

### 3.1 chat 路径（internal/service/chat.go 新 helper）

```go
// affinityKeyFromChat 会话指纹：sha256(model + 前两条消息的 role+content 前 256B)[:16]，hex。
// 多轮对话 messages 只会追加，开头前缀稳定 ⇒ 同会话同 key。
func affinityKeyFromChat(req openai.ChatCompletionRequest) string {
    h := sha256.New()
    h.Write([]byte(req.Model))
    for i := 0; i < len(req.Messages) && i < 2; i++ {
        m := req.Messages[i]
        h.Write([]byte(m.Role))
        content := m.Content
        if len(content) > 256 { content = content[:256] }
        h.Write([]byte(content))
    }
    return hex.EncodeToString(h.Sum(nil)[:16])
}
```

空 messages（不应发生）→ 兜底 `userID+":"+model` 由调用方拼。

### 3.2 多格式路径（format_adapter.go）

```go
key := extractConversationKey(canonical)          // 1. prompt_cache_key（已有）
if key == "" { key = affinityKeyFromCanonical(canonical) } // 2. canonical 前两条消息指纹（同 3.1 逻辑）
if key == "" { key = userIDStr + ":" + canonical.Model }   // 3. 兜底
```

## 4. 改动四：Server 亲和校验方法（internal/models/server.go）

service 包无法调 `clientHealthy`（包级私有），Server 暴露：

```go
// ResolveAffinity 校验亲和目标（众包 client）是否当前可用。返回 nil 表示不可用（调用方 MarkLease）。
// 校验链：在线健康 + 非冷却 + 价格帽 + 软限流（Active < rate × effectiveMax）。
// 注意：entry.IsDirect 时本方法不适用，走 ResolveDirectAffinity（见改动四B）。
func (s *Server) ResolveAffinity(entry AffinityEntry, model, userID string) *Client {
    if entry.IsDirect { return nil } // 交给 ResolveDirectAffinity
    all := s.clients.Load().(map[string]map[string]*Client)
    c := all[model][entry.ClientID]
    if c == nil { return nil }
    if configs.Config.LBCooldownEnabled && c.InCooldown() { return nil }
    if !clientHealthy(c, model) { return nil }
    // 价格帽
    maxIPPM, maxOPPM := math.MaxFloat64, math.MaxFloat64
    if s.UserPriceCapDB != nil && userID != "" {
        maxIPPM, maxOPPM, _ = s.UserPriceCapDB.GetPriceCap(userID, model)
    }
    if !priceEligible(maxIPPM, maxOPPM)(c, model) { return nil }
    // 软限流：粘性请求不把 client 顶满
    rate := configs.Config.AffinitySoftLimitRate
    if rate <= 0 || rate > 1 { rate = 0.9 }
    limit := int32(float64(s.effectiveMaxConnections(c)) * rate)
    if limit < 1 { limit = 1 }
    if c.GetActiveConnections() >= limit { return nil }
    return c
}
```

## 4B. 改动四B：Direct 后端会话亲和（prefix-cache 粘性，待实施）

> **目标与边界**：Direct 后端（vLLM/SGLang 等）拥有进程内 KV cache；同一会话的完整历史请求持续到达同一**推理实例**，才可能获得 prefix-cache 命中。本节通过会话亲和减少网关侧漂移，但不能凭空创建缓存：上游必须启用 prefix caching、请求必须携带完整且 token 序列一致的历史上下文，且 `BaseURL` 必须直达固定实例，或其后的负载均衡器必须按本节传入的会话键粘到固定 replica。随机轮询的多 replica `BaseURL` 会抵消本设计的收益。
>
> 现状 `ResolveAffinity` 对 `IsDirect` 直接返回 nil，且 `PickDirect` 按 `Priority → 负载率` 轮转，因此多后端时同一会话会漂移。本节复用 `AffinityEntry.IsDirect` 与 `AffinityStore`，不新增存储；`AffinityEntry.ClientID` 对 Direct **始终保存裸 `b.ID`**，`"direct:"+b.ID` 只用于 `failedClients` 排除键和账单 `client_id`。

### 4B.1 新增 Server 方法：ResolveDirectAffinity（internal/models/server.go）

与 `ResolveAffinity` 同构，但校验对象是 `DirectBackend`（从 `s.directBackends[model]` 按裸 ID 定位），校验链对齐 `PickDirect` 的过滤条件：

```go
// ResolveDirectAffinity 校验亲和目标（Direct 后端）是否当前可用。返回 nil 表示不可用（调用方 MarkLease）。
// 校验链：Enabled(载入时已过滤) + IsHealthy + 非冷却 + Active<MaxConns + 有该模型价格 + 价格帽 + 软限流。
// 软限流：粘性请求不把后端顶满（Active < rate × MaxConns），与 ResolveAffinity 语义一致。
func (s *Server) ResolveDirectAffinity(entry AffinityEntry, model, userID string) *DirectBackend {
    if !entry.IsDirect { return nil }
    s.directBackendsMu.RLock()
    list := s.directBackends[model]
    s.directBackendsMu.RUnlock()
    for _, b := range list {
        if b.ID != entry.ClientID { continue } // entry.ClientID 存的是 "direct:"+ID 的裸 ID
        if !b.IsHealthy() || b.InCooldown() { return nil }
        maxc := b.MaxConns
        if maxc <= 0 { maxc = 1 }
        if int(b.GetActive()) >= maxc { return nil }
        ippm, oppm, _, ok := b.PriceFor(model)
        if !ok { return nil }
        // 价格帽
        maxIPPM, maxOPPM := math.MaxFloat64, math.MaxFloat64
        if s.UserPriceCapDB != nil && userID != "" {
            maxIPPM, maxOPPM, _ = s.UserPriceCapDB.GetPriceCap(userID, model)
        }
        if ippm > maxIPPM || oppm > maxOPPM { return nil }
        // 软限流
        rate := configs.Config.AffinitySoftLimitRate
        if rate <= 0 || rate > 1 { rate = 0.9 }
        limit := int32(float64(maxc) * rate)
        if limit < 1 { limit = 1 }
        if b.GetActive() >= limit { return nil }
        return b
    }
    return nil
}
```

> **ID 约定（强制）**：`AffinityEntry.ClientID` 对 Direct 存**裸 ID**（`b.ID`）；写回必须是 `Touch(key, b.ID, true)`。`PickDirect` 的 `exclude["direct:"+b.ID]` 与账单 `client_id="direct:"+b.ID` 是不同命名空间，禁止写入亲和表。

### 4B.2 新会话的确定性选择：Direct HRW 一致性哈希（无亲和条目时）

多后端场景下，新会话（无亲和条目）若走 `PickDirect` 的负载率轮转，同一用户的新会话仍会漂移。为让**同一用户的新会话也稳定落在同一后端**（prefix-cache 概率命中），新增一致性哈希选择，替代 `PickDirect` 作为 Direct 的默认选择：

```go
// pickDirectSticky 选择 Direct 后端：优先亲和命中（entry 非空且可用），否则按一致性哈希。
// 一致性哈希：fnv1a64(routeKey+model+backendID) 取最高者——同一会话稳定落在同一后端；
// 后端上下线只影响其哈希邻域（天然防抖，与 HRW 子集同理念）。
func (s *Server) pickDirectSticky(model, userID, routeKey string, exclude map[string]bool, entry *AffinityEntry) *DirectBackend {
    // 1. 亲和命中优先
    if entry != nil && entry.IsDirect {
        if b := s.ResolveDirectAffinity(*entry, model, userID); b != nil {
            return b
        }
    }
    // 2. 一致性哈希兜底（过滤条件与 PickDirect 一致）
    maxIPPM, maxOPPM := math.MaxFloat64, math.MaxFloat64
    if s.UserPriceCapDB != nil && userID != "" {
        maxIPPM, maxOPPM, _ = s.UserPriceCapDB.GetPriceCap(userID, model)
    }
    s.directBackendsMu.RLock()
    list := s.directBackends[model]
    s.directBackendsMu.RUnlock()
    var best *DirectBackend
    var bestH uint64
    for _, b := range list {
        if exclude["direct:"+b.ID] || !b.IsHealthy() || b.InCooldown() { continue }
        maxc := b.MaxConns
        if maxc <= 0 { maxc = 1 }
        if int(b.GetActive()) >= maxc { continue }
        ippm, oppm, _, ok := b.PriceFor(model)
        if !ok || ippm > maxIPPM || oppm > maxOPPM { continue }
        f := fnv.New64a()
        f.Write([]byte(routeKey)); f.Write([]byte(model)); f.Write([]byte(b.ID))
        h := f.Sum64()
        if best == nil || h > bestH { best, bestH = b, h }
    }
    return best
}
```

> `routeKey` 必须优先使用稳定的显式会话 ID（多格式的 `prompt_cache_key` / `thread_id` / `session_id`），否则使用 `affinityKeyFromChat` 或 `affinityKeyFromCanonical`。只有这些都不可用时，才可退化为非空 `userID+":"+model`；匿名空 userID 不得使用该退化值，必须使用请求前缀指纹。这样不会把同一用户的所有独立会话固定到一个后端而形成热点。
>
> **与 `PickDirect` 的关系**：`pickDirectSticky` 的哈希兜底不按 Priority/负载率，适用于同质 Direct 池；若存在主备或异构容量，采用“最小 Priority 可用层内 HRW”，而不是跨层哈希。`PickDirect` 仅用于 Direct 粘性关闭时的兼容路径。

### 4B.3 配置（config/config.go）

复用现有 `AFFINITY_ENABLED` 总开关（默认 false），新增一个独立开关控制 Direct 粘性，避免与众包亲和耦合：

```go
// config.go Config 新增
DirectAffinityEnabled bool // DIRECT_AFFINITY_ENABLED，默认 false：Direct 后端会话粘性（prefix-cache）
```

```go
// config.go 解析
directAffinityEnabled, _ := strconv.ParseBool(getEnv("DIRECT_AFFINITY_ENABLED", "false"))
```

> **开关语义**：`DIRECT_AFFINITY_ENABLED=true` 时，仅 `stability`/`balanced` 的 Direct 路径改用 `pickDirectSticky` 并做亲和写回/校验；`cost` 仍严格走 `PickCheapest`。`false` 时完全走现状 `PickDirect`/`PickCheapest`（行为不变，回滚路径）。**依赖 `AFFINITY_ENABLED=true`**（亲和表需存在）；若 `AFFINITY_ENABLED=false` 则 `DirectAffinityEnabled` 自动失效（`server.Affinity == nil` 分支天然短路）。

> 多 Star-Fire 网关部署时，内存 `AffinityStore` 仅可用于单实例或入口已按会话键粘住网关；否则必须提供 Redis 版 `AffinityStore` 并以 `routeKey` 作为 key。Direct 后端列表、健康状态与并发上限也必须具有集群一致性，或由上游网关执行全局限流。未满足这些条件不得宣称跨网关 KV 命中提升。

### 4B.4 集成点（chat.go / format_adapter.go）

两处调用点同构改造。以 `handleChatWithRetry` 为例：

**L0 亲和解析**（现有块之后追加 Direct 分支）：

```go
var affinityKey string
var affinityClient *models.Client
var affinityDirect *models.DirectBackend
var affinityHit bool
if server.Affinity != nil {
    affinityKey = affinityKeyFromChat(request)
    if affinityKey == "" { affinityKey = userIDStr + ":" + request.Model }
    if entry, ok := server.Affinity.Get(affinityKey); ok {
        if entry.IsDirect {
            if configs.Config.DirectAffinityEnabled {
                if b := server.ResolveDirectAffinity(entry, request.Model, userIDStr); b != nil {
                    affinityDirect = b
                } else {
                    server.Affinity.MarkLease(affinityKey)
                }
            }
        } else if ac := server.ResolveAffinity(entry, request.Model, userIDStr); ac != nil {
            affinityClient = ac
        } else {
            server.Affinity.MarkLease(affinityKey)
        }
    }
}
```

**Direct 优先块**（attempt 循环内第 0 步）改为：attempt 0 且 `affinityDirect != nil` 时直接用粘住后端，否则走 `pickDirectSticky`（一致性哈希兜底）：

```go
if configs.Config.DirectBackendsEnabled && routing != RoutingCost {
    var b *models.DirectBackend
    if attempt == 0 && affinityDirect != nil {
        b = affinityDirect
        affinityHit = true
    } else if configs.Config.DirectAffinityEnabled {
        b = server.PickDirectSticky(request.Model, userIDStr, affinityKey, failedClients, nil)
    } else {
        b = server.PickDirect(request.Model, userIDStr, failedClients)
    }
    if b != nil {
        failedClients["direct:"+b.ID] = true
        if handleDirectChat(c, server, b, extendedRequest, userIDStr) {
            return
        }
        time.Sleep(backoff(attempt))
        continue
    }
}
```

> **注意**：`pickDirectSticky` 的一致性哈希兜底**不把 affinity 条目写回**（新会话首次无条目，哈希选择后由成功写回建立亲和）。`affinityHit` 仅在 attempt 0 命中既有亲和条目时为 true（供实测校验用）。

**成功写回**（`handleDirectChat` 返回 true 后，即 Direct 成功路径）：在 `handleDirectChat` 内部成功分支（`recordDirectUsage` 之后）追加：

```go
// direct_chat.go handleDirectNonStream / handleDirectStream 成功尾部
if server.Affinity != nil && configs.Config.DirectAffinityEnabled {
    if key, ok := c.Get("affinity_key"); ok {
        if hit, _ := c.Get("affinity_hit"); hit == true || server.Affinity.LeaseExpired(key.(string)) {
            server.Affinity.Touch(key.(string), b.ID, true) // 裸 ID + IsDirect=true
        }
    }
}
```

> **写回位置**：Direct 成功写回放在 `handleDirectChat` 内部（而非 chat.go 的 `case MESSAGE` 分支），因为 Direct 路径不经过 WS `respConn` 的 `case public.MESSAGE` 分支。`affinity_key`/`affinity_hit` 需在调用 `handleDirectChat` 前 `c.Set`（见下）。

**传给计费层**：在调用 `handleDirectChat` 前设置（与 WS 路径同款）：

```go
if server.Affinity != nil {
    c.Set("affinity_key", affinityKey)
    if affinityHit { c.Set("affinity_hit", true) }
}
```

**cost 路由的明确取舍**：首期保持“每请求严格最低价层”的语义，**不读取或覆盖 Direct 亲和条目，也不调用 `pickDirectSticky`**。这样价格保证与缓存收益边界清晰。若产品需要“首次按价、后续优先缓存”，必须新增独立 `cost_sticky` 路由模式，并定义最大允许溢价（例如 `X-SF-Max-Cache-Premium`）；不得在现有 `cost` 模式中静默绕过更低价格。

**format_adapter.go**：与 chat 路径同构，`routeKey` 优先取 §3.2 的显式会话键，再回退内容指纹；仅 stability/balanced 的 Direct 路径接入 `pickDirectSticky` 与写回。

### 4B.5 变体与取舍

| 变体 | 说明 | 取舍 |
|---|---|---|
| **A. 纯一致性哈希**（默认） | 哈希兜底不按 Priority/负载率 | 同质后端下稳定最优；主备容灾场景需人工保证同规格 |
| **B. Priority 分层 + 层内哈希** | 先按 Priority 分层，层内一致性哈希 | 保留主备语义；实现多一层循环，代码略复杂 |
| **C. 仅亲和、无哈希兜底** | 新会话仍走 `PickDirect` 负载率 | 改动最小，但新会话漂移问题未解，多后端收益打折 |

首期建议 **A**（实现最简、收益最大）；若现网 Direct 有明确主备分层需求再升级 B。

## 5. 改动五：L0 短路 + 成功写回（chat.go / format_adapter.go）

`handleChatWithRetry` 循环之前：

```go
var affinityKey string
var affinityHit bool // 本次请求是否粘性命中（供实测校验用）
if server.Affinity != nil {
    affinityKey = affinityKeyFromChat(request)
    if affinityKey == "" { affinityKey = userIDStr + ":" + request.Model }
    if entry, ok := server.Affinity.Get(affinityKey); ok {
        if ac := server.ResolveAffinity(entry, request.Model, userIDStr); ac != nil {
            // 作为 attempt 0 的选择：跳过 LoadBalance 直接用 ac
        } else {
            server.Affinity.MarkLease(affinityKey)
        }
    }
}
```

实现方式（最小侵入）：循环内第 1 步改为

```go
var client *models.Client
if attempt == 0 && affinityClient != nil {
    client = affinityClient
    affinityHit = true
} else {
    // 1a Direct（P1）→ 1b LoadBalanceExcluding（现有）
}
```

- **成功写回**：`case public.MESSAGE, public.MESSAGE_STREAM:` 分支（`client.ResetFailures()` 旁）：
  ```go
  if server.Affinity != nil {
      if affinityHit || server.Affinity.LeaseExpired(affinityKey) {
          server.Affinity.Touch(affinityKey, client.ID, false)
      } // 租约未过期且换了 client → 不覆盖（下次仍优先试原 client = 回迁）
  }
  ```
- **失败**：粘性 client 失败进 `failedClients`（现有逻辑天然覆盖），后续 attempt 走正常 LoadBalance。
- 把 `affinityKey`/`affinityHit` 通过 `c.Set("affinity_key", ...)`/`c.Set("affinity_hit", true)` 传给计费层（§6 实测校验用），沿用 `reasoning_conv_key` 的先例。
- `handleMultiFormatWithRetry` 完全同构（key 来源 §3.2）。

## 6. 改动六：CacheHitEMA + 实测命中校验

### 6.1 Client 字段（internal/models/client.go）

```go
// P2: 实测 cache 命中率 EMA（float64bits 原子；仅用于 tiebreak 与 cost 有效单价，不进 perfScore）
CacheHitEMA uint64 `json:"-" gorm:"-"`
```

```go
// UpdateCacheHit 以 EMA(alpha=0.2) 更新实测命中率，h ∈ [0,1]。
func (c *Client) UpdateCacheHit(h float64)
// GetCacheHitEMA 返回当前 EMA（未初始化返回 0）。
func (c *Client) GetCacheHitEMA() float64
```

`DirectBackend` 同样加（P1 文件）。

### 6.2 recordTokenUsage / recordCanonicalUsage 尾部追加

```go
// P2: 实测命中反馈（纯内存，零 DB）
if inputTokens > 0 {
    h := float64(cachedTokens) / float64(inputTokens)
    if cl := server.GetClientByID(clientID); cl != nil { cl.UpdateCacheHit(h) }
    // 亲和实测校验：粘性命中但实测低命中 → 连续 K 次删除亲和
    if server.Affinity != nil {
        if key, ok := c.Get("affinity_key"); ok {
            if hit, _ := c.Get("affinity_hit"); hit == true {
                if h < configs.Config.AffinityMinHitRate {
                    if server.Affinity.RecordMiss(key.(string)) >= int32(configs.Config.AffinityMissK) {
                        server.Affinity.Delete(key.(string))
                    }
                } // h 达标 → Touch 已在成功分支做过，MissCount 由 Touch 清零
            }
        }
    }
}
```

前置核对：`Server.GetClientByID` 已存在（membership 连接计数用过）；`"direct:"` 前缀 ID 查不到普通 Client 是预期行为。Direct 的反馈必须在 `recordDirectUsage` 中、调用 `recordTokenUsage` 前独立处理，因为后者找不到普通 Client 时会提前返回：

```go
if usage.PromptTokens > 0 {
    h := float64(cached) / float64(usage.PromptTokens)
    b.UpdateCacheHit(h)
    // 仅当本次确为 Direct 亲和命中时执行连续低命中失效；无 usage 不更新也不误删。
    recordAffinityCacheFeedback(c, server, h)
}
```

`recordAffinityCacheFeedback` 应抽出 `recordTokenUsage` 中现有的 `affinity_key` / `affinity_hit` / `RecordMiss` / `Delete` 逻辑，供普通 Client 与 Direct 共用；成功写回在反馈前执行，使达标命中能清零 `MissCount`。

## 7. 改动七：HRW 偏好子集 + tiebreak（internal/models/server.go）

### 7.1 HRW 子集

`LoadBalanceExcluding` 在 Predicate 循环之后、`s.pick(...)` 之前：

```go
if n := configs.Config.LBHRWSubsetSize; n > 0 && len(eligible) > n {
    eligible = hrwSubset(userID, eligible, n)
}
```

```go
// hrwSubset Rendezvous hashing：按 fnv1a64(userID+clientID) 取 top-n。
// 同一用户偏好子集稳定；client 上下线只影响其哈希邻域的用户（天然防抖）。
func hrwSubset(userID string, eligible []*Client, n int) []*Client {
    type hw struct { c *Client; h uint64 }
    hs := make([]hw, len(eligible))
    for i, c := range eligible {
        f := fnv.New64a()
        f.Write([]byte(userID)); f.Write([]byte(c.ID))
        hs[i] = hw{c, f.Sum64()}
    }
    sort.Slice(hs, func(i, j int) bool { return hs[i].h > hs[j].h })
    out := make([]*Client, n)
    for i := 0; i < n; i++ { out[i] = hs[i].c }
    return out
}
```

### 7.2 pickSmart tiebreak

阶段1排序比较器改为：

```go
sort.Slice(scoredClients, func(i, j int) bool {
    di := scoredClients[i].score - scoredClients[j].score
    if math.Abs(di) < 0.01 { // ε：perf 接近时实测命中率高者优先
        return scoredClients[i].c.GetCacheHitEMA() > scoredClients[j].c.GetCacheHitEMA()
    }
    return di > 0
})
```

ε 固定 0.01（不做配置，避免过度参数化）；影响面仅限排序边界，不形成流量正反馈。

## 8. 亲和 TTL 与 client 引擎匹配（可后置项）

client 注册/心跳上报 `keep_alive`（`pkg/public/protocol.go` `PPMessage` 加 `KeepAliveSec int \`json:"keep_alive_sec,omitempty"\``，模式照抄 BandwidthMbps 链路：Python→TCP→Go client→PPMessage→`keepAliveClient` 解析→`Client.KeepAliveSec`）。`Touch` 时对该 client 的 TTL 取 `min(全局TTL, KeepAliveSec)`。**首期可跳过**：实测校验（§6）已兜底 cache 失效场景。

## 9. 编码顺序

① config + AffinityStore + 单测（-race） → ② 亲和键 helper + 单测 → ③ ResolveAffinity + 单测 → ④ chat.go L0 短路 + 成功写回 → ⑤ CacheHitEMA + 实测校验 → ⑥ format_adapter.go 同构接入 → ⑦ HRW + tiebreak + 单测 → ⑧ **Direct 粘性（改动四B）**：`ResolveDirectAffinity` + `pickDirectSticky` + config + 两处调用点 + 写回 + 单测 → ⑨ 仿真 + 集成。

> **Direct 粘性可独立交付**：⑧ 不依赖 ⑦（HRW 只影响众包新会话），可与 ⑦ 并行；`DIRECT_AFFINITY_ENABLED=false` 时完全走现状，可随时回滚。

## 10. 单元测试清单

**internal/models/affinity_test.go（新）**：

| 测试 | 断言 |
|---|---|
| `TestAffinityTouchGet` | Touch→Get 命中；TTL 过期后 Get miss；Get/Touch 滑动续期 |
| `TestAffinityLease` | MarkLease 后 LeaseExpired=false；lease 到期 true；Touch 清零 lease |
| `TestAffinityMiss` | RecordMiss 递增；Touch 同 client 清零 MissCount |
| `TestAffinityLRU` | maxEntries=64（2/片）写 128 条：Len()≤64，最旧被逐出 |
| `TestAffinityConcurrent` | 16 goroutine × 1000 次混合 Get/Touch/Delete，`-race` 干净 |
| `TestAffinityJanitor` | 短 TTL + janitor 周期缩短（构造函数测试钩子或直接调私有 sweep），过期项被清 |

**internal/models/server_smart_test.go 追加**：

| 测试 | 断言 |
|---|---|
| `TestHRWSubsetStable` | 同 userID 调 10 次子集一致；不同 userID 子集不同；删除子集外 client 子集不变，删除子集内 client 只替换 1 个 |
| `TestPickSmartCacheHitTiebreak` | 两 client perfScore 相同（同参数、jitter=0）、CacheHitEMA 0.8 vs 0：候选排序前者在前 |
| `TestResolveAffinity` | 5 分支：正常返回；冷却 nil；不健康 nil；超价格帽 nil；Active≥0.9×max nil |
| `TestResolveDirectAffinity` | 5 分支（同 ResolveAffinity 语义）：正常返回；不健康 nil；冷却 nil；Active≥MaxConns nil；超价格帽 nil；软限流 Active≥0.9×max nil；`entry.IsDirect=false` 直接 nil |
| `TestPickDirectStickyHashStable` | 同 routeKey+model 调 10 次选同一后端；删除选中后端只替换 1 个（邻域防抖） |
| `TestPickDirectStickyAffinityFirst` | 有可用亲和条目时优先返回该后端（即使哈希指向别的）；亲和条目不可用（冷却/不健康）时回退哈希 |
| `TestPickDirectStickyUsesRouteKey` | 同 routeKey 恒定；同一 userID 的不同 routeKey 可落到不同后端；空 userID 不会退化为全局相同哈希 |
| `TestDirectAffinityIDNamespace` | `Touch(key, b.ID, true)` 能被 `ResolveDirectAffinity` 找回；`"direct:"+b.ID` 不得写入亲和表 |

**internal/service（追加）**：

| 测试 | 断言 |
|---|---|
| `TestAffinityKeyFromChat` | 同会话多轮（messages 追加）key 不变；不同 system/user 前缀 key 不同；content>256B 只取前缀 |
| `TestUpdateCacheHitEMA` | 初值直赋；序列 [1,0] → 0.2 alpha 精确值 |
| `TestDirectAffinityWriteBack` | mock Direct 后端成功响应 → `Affinity.Touch(key, b.ID, true)` 被调用；`affinity_hit` 时写回，非 hit 且租约未过期不覆盖 |
| `TestDirectAffinityMissDelete` | Direct 粘性命中但 cached_tokens=0 连续 K 次 → 亲和删除（复用 §6.2 逻辑，验证 Direct 分支同样生效） |
| `TestDirectUsageFeedback` | Direct 响应的 `cached_tokens` 更新该 DirectBackend 的 CacheHitEMA；无 usage 不误判为 cache miss |
| `TestCostRoutingDoesNotBypassCheapest` | 已有 Direct 亲和记录时，`cost` 仍选择最低价格层；只在新增 `cost_sticky` 模式后才允许带上限地偏离最低价 |

全部 `go test -race ./internal/...`。

## 11. 集成测试

1. **回归**：默认 env（亲和关、HRW 关、Direct 粘性关）全量测试 + build，行为不变。
2. **亲和命中链路**：`AFFINITY_ENABLED=true` + 2 个 client 挂同一模型：
   - 同一会话（相同 system+首条 user，追加多轮）连发 10 请求 → 日志确认 10 次同一 clientID；
   - 杀掉粘住的 client → 下一请求落另一 client（租约内不覆盖），原 client 恢复后（租约 60s 内）请求回迁；租约过期后粘新 client；
   - 打满粘住 client 至 ≥0.9×max（并发压测）→ 溢出走 LoadBalance。
3. **实测校验**：重启粘住 client 的推理引擎（cache 清空）→ 若后端 usage 返回 cached_tokens=0，连续 3 请求后日志确认亲和删除、重新选择。（后端不回 cached_tokens 时此路径静默不触发——验证不误删。）
4. **KV 收益度量**（上线前后各跑 1 天）：`SELECT model, SUM(cached_tokens)*1.0/SUM(input_tokens) FROM token_usages WHERE timestamp > ? GROUP BY model;` 记录对比表。
5. **HRW**：`LB_HRW_SUBSET_SIZE=8` + 仿真 `docs/lb_saturation_test.py` 加会话模型（每用户连续 10 请求），统计"会话内 client 切换次数"与 client 上下线时的全局重分配比例，对比开关前后。
6. **Direct 粘性**：`AFFINITY_ENABLED=true` + `DIRECT_AFFINITY_ENABLED=true` + 2 个 Direct 后端（httptest.Server mock，各自独立计数）挂同一模型：
   - 同一会话连发 10 请求 → 日志确认 10 次同一 backendID（一致性哈希 + 亲和写回）；
    - 两个不同会话键各连发 10 请求 → 各自稳定；验证同一 user 的不同会话键允许分散到不同后端；
   - 杀掉粘住的后端（mock 返回 5xx）→ 下一请求经 `pickDirectSticky` 哈希兜底落另一后端，原后端恢复后（租约内）回迁；
   - 打满粘住后端至 ≥0.9×MaxConns → 溢出走哈希兜底另一后端；
    - 在每个 Direct URL 后再置两个随机轮询 mock replica → 验证命中率下降；改为实例直连或按会话键粘性转发后恢复。这是上线环境的必要验证；
   - 关闭 `DIRECT_AFFINITY_ENABLED` → 行为回到 `PickDirect` 负载率轮转（回归对照）。

## 12. 验收标准

- [ ] `-race` 全量通过；默认 env 行为不变。
- [ ] 集成 2/3 的三条链路（命中/回迁/软限流溢出）人工验证通过。
- [ ] 集成 6 的 Direct 粘性链路（命中/哈希稳定/回迁/软限流溢出/开关回滚）人工验证通过。
- [ ] 亲和命中/回迁/逐出/实测失效计数有每分钟聚合日志（单行，不刷屏）。
- [ ] cached_tokens 比率、同会话同实例率、TTFT p50/p95、每请求 prefill tokens 上线前后按 model/DirectBackend 对比数据已记录；`cached_tokens=0` 仅作观测信号，不能单独断言路由失败。
- [ ] CacheHitEMA 未出现在 perfScore 主权重路径（code review 检查项）。
- [ ] `DIRECT_AFFINITY_ENABLED=false` 时 Direct 路径与现状逐字节一致（code review 检查项）。
- [ ] 每个 Direct `BaseURL` 已验证为单一推理实例，或其后负载均衡器已按 `routeKey` 粘性转发；多 Star-Fire 网关时 AffinityStore 与后端容量状态已共享，或入口已按 `routeKey` 粘住网关。
