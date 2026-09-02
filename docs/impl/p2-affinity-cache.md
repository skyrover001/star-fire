# P2 实施规格：会话亲和 + cache 命中反馈 + HRW 防抖

> 上游设计：`docs/scale-lb-roadmap-design.md` P2 章节
> 状态：待实施 | 依赖：P0（缓存分让亲和校验零 DB 成本；冷却检查复用） | 回滚：`AFFINITY_ENABLED=false`、`LB_HRW_SUBSET_SIZE=0`
> 核心原则：路由决策用**确定性预测命中**（亲和键→原 client），实测 `cached_tokens` 只做反馈校验与 tiebreak；CacheHitEMA **不进 perfScore 主权重**（防正反馈马太效应）。

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
    ClientID   string // 众包 clientID 或 "direct:"+backendID
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
// ResolveAffinity 校验亲和目标是否当前可用。返回 nil 表示不可用（调用方 MarkLease）。
// 校验链：在线健康 + 非冷却 + 价格帽 + 软限流（Active < rate × effectiveMax）。
func (s *Server) ResolveAffinity(entry AffinityEntry, model, userID string) *Client {
    if entry.IsDirect { return nil } // direct 亲和由 PickDirect 天然处理（无状态 HTTP，粘性收益小，M1 先跳过）
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

前置核对：`Server.GetClientByID` 已存在（membership 连接计数用过）；`"direct:"` 前缀 ID 查不到返回 nil，自动跳过（Direct 的 EMA 更新在 direct_chat.go 内直接持有 backend 引用做）。

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

① config + AffinityStore + 单测（-race） → ② 亲和键 helper + 单测 → ③ ResolveAffinity + 单测 → ④ chat.go L0 短路 + 成功写回 → ⑤ CacheHitEMA + 实测校验 → ⑥ format_adapter.go 同构接入 → ⑦ HRW + tiebreak + 单测 → ⑧ 仿真 + 集成。

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

**internal/service（追加）**：

| 测试 | 断言 |
|---|---|
| `TestAffinityKeyFromChat` | 同会话多轮（messages 追加）key 不变；不同 system/user 前缀 key 不同；content>256B 只取前缀 |
| `TestUpdateCacheHitEMA` | 初值直赋；序列 [1,0] → 0.2 alpha 精确值 |

全部 `go test -race ./internal/...`。

## 11. 集成测试

1. **回归**：默认 env（亲和关、HRW 关）全量测试 + build，行为不变。
2. **亲和命中链路**：`AFFINITY_ENABLED=true` + 2 个 client 挂同一模型：
   - 同一会话（相同 system+首条 user，追加多轮）连发 10 请求 → 日志确认 10 次同一 clientID；
   - 杀掉粘住的 client → 下一请求落另一 client（租约内不覆盖），原 client 恢复后（租约 60s 内）请求回迁；租约过期后粘新 client；
   - 打满粘住 client 至 ≥0.9×max（并发压测）→ 溢出走 LoadBalance。
3. **实测校验**：重启粘住 client 的推理引擎（cache 清空）→ 若后端 usage 返回 cached_tokens=0，连续 3 请求后日志确认亲和删除、重新选择。（后端不回 cached_tokens 时此路径静默不触发——验证不误删。）
4. **KV 收益度量**（上线前后各跑 1 天）：`SELECT model, SUM(cached_tokens)*1.0/SUM(input_tokens) FROM token_usages WHERE timestamp > ? GROUP BY model;` 记录对比表。
5. **HRW**：`LB_HRW_SUBSET_SIZE=8` + 仿真 `docs/lb_saturation_test.py` 加会话模型（每用户连续 10 请求），统计"会话内 client 切换次数"与 client 上下线时的全局重分配比例，对比开关前后。

## 12. 验收标准

- [ ] `-race` 全量通过；默认 env 行为不变。
- [ ] 集成 2/3 的三条链路（命中/回迁/软限流溢出）人工验证通过。
- [ ] 亲和命中/回迁/逐出/实测失效计数有每分钟聚合日志（单行，不刷屏）。
- [ ] cached_tokens 比率上线前后对比数据已记录（§11.4）。
- [ ] CacheHitEMA 未出现在 perfScore 主权重路径（code review 检查项）。
