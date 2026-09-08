# P1 实施规格：DirectBackend 固定后端 + 偏好双池路由

> 上游设计：`docs/scale-lb-roadmap-design.md` P1 章节
> 状态：已实施 | 依赖：无（M2 的 cost 有效单价依赖 P2 的 CacheHitEMA，先用名义价） | 回滚：`DIRECT_BACKENDS_ENABLED=false`
>
> **里程碑拆分**：
> - **M1（先做，独立可上线）**：DirectBackend 数据模型 + 注册表 + 健康检查 + `stability` 默认路由（Direct 优先→众包溢出）+ HTTP 直连转发 + 计费 + CLI。
> - **M2（后做）**：`routing` 偏好（cost/balanced）+ 容忍度参数 + tiers 暴露。cost 的有效单价在 P2 落地前用名义 IPPM/OPPM。

## 0. 现状代码事实（已核实）

| 事实 | 位置 |
|---|---|
| `handleChatWithRetry(c, server, extendedRequest, userIDStr)`：attempt 循环内第 1 步 `server.LoadBalanceExcluding(request.Model, userIDStr, failedClients)` | `internal/service/chat.go` L186 起 |
| 计费入口 `recordTokenUsage(c, server, requestID, model, inputTokens, outputTokens, totalTokens, cachedTokens int, clientID string, ippm, oppm, cippm float64)` | `chat.go` L677 |
| 多格式路径 `handleMultiFormatWithRetry`（结构与 chat 相同，第 1 步同样是 LoadBalanceExcluding）；计费 `recordCanonicalUsage`（L698） | `internal/service/format_adapter.go` L121 |
| 4xx 判定 `isClientRequestError(content interface{})` | `chat.go` L383 |
| 价格帽查询：`s.UserPriceCapDB.GetPriceCap(userID, model)` 返回 `(maxIPPM, maxOPPM, error)`，MaxFloat64=不限 | `server.go` `LoadBalanceExcluding` |
| `Client` 的 ModelsJSON 序列化模式：`BeforeSave`/`AfterFind` + `gorm:"type:text"` （MySQL varchar(191) 截断教训） | `internal/models/client.go` |
| CLI 子命令模式：`main.go` 已有 set-bonus/set-membership/migrate-db 等 | `main.go` |
| Server 构造：`NewServer()` 集中 New 各 DB + 挂字段 | `internal/models/server.go` |
| SSE 转发参考实现（逐行读 + 原样写）| `client/internal/inference/openai/engine.go` |
| `public.Model` 结构（Name/IPPM/OPPM/CIPPM） | `pkg/public/` |

## 1. M1-改动一：数据模型（新文件 internal/models/direct_backend.go）

```go
package models

// DirectBackend 固定后端（平台直连的模型供给：自建 vLLM 集群 / 合作方 OpenAI 兼容 API）。
type DirectBackend struct {
    ID        string `json:"id" gorm:"primaryKey;size:64"`
    Name      string `json:"name" gorm:"size:128"`
    BaseURL   string `json:"base_url" gorm:"size:256"` // e.g. http://vllm-1:8000/v1（不含尾斜杠）
    APIKey    string `json:"-" gorm:"size:256"`        // 出站鉴权，日志必须脱敏
    Format    string `json:"format" gorm:"size:32"`    // 首期仅 "openai"
    Enabled   bool   `json:"enabled"`
    MaxConns  int    `json:"max_conns"`                // 硬并发上限，<=0 视为 1
    Priority  int    `json:"priority"`                 // 越小越优先
    ModelsJSON string `json:"-" gorm:"column:models;type:text"` // []public.Model
    CreatedAt time.Time
    UpdatedAt time.Time

    // 运行态（内存）
    Models         []*public.Model `json:"models" gorm:"-"`
    ActiveConns    int32           `json:"-" gorm:"-"` // atomic
    RecentFailures int32           `json:"-" gorm:"-"` // atomic
    CooldownUntil  int64           `json:"-" gorm:"-"` // atomic, unix ns
    LatencyEMA     uint64          `json:"-" gorm:"-"` // atomic float64bits（健康检查 RTT ms）
    Healthy        int32           `json:"-" gorm:"-"` // atomic bool（健康检查结果）
}
```

- `BeforeSave`/`AfterFind`：Models ↔ ModelsJSON，照抄 Client 的实现。
- 原子方法（与 Client 同名同语义，独立实现）：`IncrActive/DecrActive/GetActive`、`IncrFailures/ResetFailures/GetFailures`、`TripCooldown/InCooldown/ClearCooldown`（冷却参数复用 P0 的 `LBCooldownBaseMs/MaxMs`，但**不受** `LB_COOLDOWN_ENABLED` 开关限制——Direct 冷却始终开启，因为无 P0 前置时也需要故障隔离）、`SetLatencyEMA(ms float64)`（alpha 复用 `LBEMAAplha`）、`SetHealthy(bool)/IsHealthy()`。
- `PriceFor(model string) (ippm, oppm, cippm float64, ok bool)`：遍历 Models 找价格。

`DirectBackendDB`：

```go
type DirectBackendDB struct{ db *gorm.DB }
func NewDirectBackendDB(db *gorm.DB) *DirectBackendDB // AutoMigrate
func (d *DirectBackendDB) List() ([]*DirectBackend, error)          // 全部（含 disabled，CLI 用）
func (d *DirectBackendDB) Save(b *DirectBackend) error
func (d *DirectBackendDB) SetEnabled(id string, enabled bool) error
func (d *DirectBackendDB) Delete(id string) error
```

## 2. M1-改动二：Server 注册表 + PickDirect + 健康检查（internal/models/server.go）

Server 增加字段：

```go
DirectBackendDB  *DirectBackendDB
directBackendsMu sync.RWMutex
directBackends   map[string][]*DirectBackend // model name → backends（同一 backend 可出现在多个 model 桶）
```

`NewServer()`：`NewDirectBackendDB(gormDB)` 挂上；当 `configs.Config.DirectBackendsEnabled` 时调用 `server.LoadDirectBackends()` + `go server.runDirectHealthCheck()`。

```go
// LoadDirectBackends 从 DB 载入 enabled 后端到内存注册表（启动时/CLI 变更后调用）。
func (s *Server) LoadDirectBackends() error {
    all, err := s.DirectBackendDB.List()
    if err != nil { return err }
    m := map[string][]*DirectBackend{}
    for _, b := range all {
        if !b.Enabled { continue }
        b.SetHealthy(true) // 初始乐观，健康检查会纠正
        for _, mod := range b.Models { m[mod.Name] = append(m[mod.Name], b) }
    }
    s.directBackendsMu.Lock(); s.directBackends = m; s.directBackendsMu.Unlock()
    return nil
}

// PickDirect 选择固定后端。exclude 的 key 格式与 failedClients 一致："direct:"+ID。
// 过滤：Enabled(已在载入时过滤)、IsHealthy、非冷却、Active<MaxConns、有该模型价格、价格帽。
// 选择：Priority 最小层 → 层内 Active/MaxConns 最低。
func (s *Server) PickDirect(model, userID string, exclude map[string]bool) *DirectBackend {
    maxIPPM, maxOPPM := math.MaxFloat64, math.MaxFloat64
    if s.UserPriceCapDB != nil && userID != "" {
        maxIPPM, maxOPPM, _ = s.UserPriceCapDB.GetPriceCap(userID, model)
    }
    s.directBackendsMu.RLock()
    list := s.directBackends[model]
    s.directBackendsMu.RUnlock()

    var best *DirectBackend
    bestPrio := int(^uint(0) >> 1) // maxint
    bestLoad := 2.0
    for _, b := range list {
        if exclude["direct:"+b.ID] || !b.IsHealthy() || b.InCooldown() { continue }
        maxc := b.MaxConns; if maxc <= 0 { maxc = 1 }
        if int(b.GetActive()) >= maxc { continue }
        ippm, oppm, _, ok := b.PriceFor(model)
        if !ok || ippm > maxIPPM || oppm > maxOPPM { continue }
        load := float64(b.GetActive()) / float64(maxc)
        if b.Priority < bestPrio || (b.Priority == bestPrio && load < bestLoad) {
            best, bestPrio, bestLoad = b, b.Priority, load
        }
    }
    return best
}
```

健康检查：

```go
// runDirectHealthCheck 每 30s GET {BaseURL}/models（Bearer APIKey），5s 超时。
// 成功：SetHealthy(true) + SetLatencyEMA(rtt)；失败：SetHealthy(false)。
// 连续 3 次失败打 WARN 日志（一次性，不刷屏）。
```

周期常量 `DIRECT_HEALTH_INTERVAL`（env，默认 30 秒）可后置，先写死 30s。

## 3. M1-改动三：HTTP 直连转发（新文件 internal/service/direct_chat.go）

```go
// handleDirectChat 通过 HTTP 直连固定后端处理 chat completions。
// 返回值语义：
//   done=true  已向调用方写响应（成功或 4xx 直接回传），调用方 return；
//   done=false 后端失败（5xx/超时/断流且未写出任何字节），调用方记失败后继续重试循环。
func handleDirectChat(c *gin.Context, server *models.Server, b *models.DirectBackend,
    extendedRequest public.ExtendedChatRequest, userIDStr string) (done bool)
```

实现要点（全部有现成参照）：

1. **请求**：`json.Marshal(extendedRequest)`（含 RawBody 时优先用 RawBody，语义对齐 client 端 openai engine）POST `b.BaseURL + "/chat/completions"`；Header：`Authorization: Bearer <APIKey>`、`Content-Type: application/json`；`http.Client{Timeout: 0}` + `context.WithTimeout(c, configs.Config.ChatMaxTime秒)`（流式不能设 Client.Timeout，会截断 SSE）。
2. **计数**：进入即 `b.IncrActive()`，`defer b.DecrActive()`。
3. **状态码分派**：
   - `>=400 && <500`：读 body 原样回传（状态码+body 透传），`done=true`（对齐 isClientRequestError 语义：请求本身错误不重试、不记失败）。
   - `>=500` 或网络错误：`b.IncrFailures(); b.TripCooldown()`，`done=false`。
4. **非流式**：读全量 body → 解析 `openai.ChatCompletionResponse` 取 usage（`PromptTokens/CompletionTokens/TotalTokens` + `PromptTokensDetails.CachedTokens`）→ 原样写给用户（`c.Data(200, "application/json", body)`）→ `recordTokenUsage(c, server, requestID, model, in, out, total, cached, "direct:"+b.ID, ippm, oppm, cippm)`。requestID 用 `uuid.NewString()`；价格从 `b.PriceFor(model)`。
5. **流式**：`bufio.Scanner` 逐行读 SSE → 原样写 `c.Writer` + `Flush()`；解析 `data:` 行中含 `"usage"` 的尾块提取 usage；`[DONE]` 后 `recordTokenUsage`；**若已写出至少一个 chunk 后发生断流**：终止流（写 `data: [DONE]`），`done=true`（不可重试——用户已收到部分内容，与 WS 路径"首 token 后不重试"语义一致），失败仍记 `b.IncrFailures()`。
6. **成功**：`b.ResetFailures()`（含 ClearCooldown）。
7. 日志脱敏：任何日志不得输出 `b.APIKey`。

## 4. M1-改动四：路由整合（internal/service/chat.go）

`handleChatWithRetry` attempt 循环内、现第 1 步之前插入：

```go
// 1a. Direct 主力池（默认 stability 偏好：Direct 优先，失败/无供给溢出到众包）
if configs.Config.DirectBackendsEnabled {
    if b := server.PickDirect(request.Model, userIDStr, failedClients); b != nil {
        failedClients["direct:"+b.ID] = true
        if handleDirectChat(c, server, b, extendedRequest, userIDStr) {
            return
        }
        time.Sleep(backoff(attempt))
        continue // 本次 attempt 消耗在 direct 上，下一轮先重试其他 direct，用尽后自然落 1b
    }
}
// 1b. 众包池（现有代码，不动）
client := server.LoadBalanceExcluding(request.Model, userIDStr, failedClients)
```

- key 前缀 `direct:` 保证与众包 clientID 不冲突（clientID 为 uuid）。
- 无 Direct 供给（注册表无该模型/全部排除）→ `PickDirect` 返回 nil → 直接走 1b，即现行为（冷启动退化，无特殊逻辑）。
- `handleMultiFormatWithRetry` 同位置同逻辑（M1 可只支持 Format=="openai" 的 backend：非 openai backend 载入时跳过并 WARN；anthropic/responses 后端 M2 再做）。

## 5. M1-改动五：config + CLI + 收益兜底

**config/config.go**：

```go
DirectBackendsEnabled bool // DIRECT_BACKENDS_ENABLED，默认 false
```

**main.go** 子命令（模式照抄 set-membership）：

```
starfire add-backend    -id=vllm1 -name="内部vLLM" -url=http://host:8000/v1 -key=$KEY -maxconns=32 -priority=10 -models='[{"name":"qwen3-32b","IPPM":2.0,"OPPM":6.0,"CIPPM":0.5}]'
starfire list-backends            # 表格输出，APIKey 打码（前4后4）
starfire enable-backend <id>
starfire disable-backend <id>
starfire del-backend <id>
```

CLI 直接操作 DB；运行中的 server 下次重启生效（M1 不做热载；如需热载，M2 加 SIGHUP 或管理 API）。

**收益兜底核对**：`recordTokenUsage` 里 `GetClientByModel`（或等价查找）查不到 `"direct:"+ID` 会走 nil 分支跳过 INCOME 推送——实施时**跑一次真实请求确认日志级别**，若每请求打 error 级则降为 debug。

## 6. M2：路由偏好 + 容忍度（依赖 M1，cost 有效单价先用名义价）

> 实施状态：**同价层可靠性竞争规则与 `ReliabilityEMA` 已实现**（`PickCheapest` 内强制规则 + `directQualityScore`/`communityQualityScore` + `DirectBackend.UpdateReliability`，`handleDirectChat` 成功采样 1、5xx/网络错误采样 0）。偏好解析、balanced、容忍度、tiers 仍待实施。
>
> **采样归属语义**：仅后端过错采样 0（网络错误、5xx、断流）；本地序列化、格式适配（converter）错误、用户侧断连不采样（仍计 `IncrFailures` 用于冷却）。`writeUserStreamEvent` 早退路径为适配层/用户侧错误，有意不采样也不 `ResetFailures`。

1. **偏好解析**（新 helper `resolveRouting(c *gin.Context, req) string`）：优先级 = body `routing` 字段（`ExtendedChatRequest` 加 `Routing string \`json:"routing,omitempty"\``，`pkg/public/chat.go`）> header `X-SF-Routing` > API Key 级（`APIKey` 表加 `Routing` 列，可再后置）> `configs.Config.RoutingDefault`。合法值 `stability|cost|balanced`，非法回退默认。
2. **stability**：即 M1 行为。
3. **cost**：新 `Server.PickCheapest(model, userID, exclude) (client *Client, backend *DirectBackend)`——两池候选先分别通过模型、健康、容量、冷却和用户价格帽过滤，再按单价升序分层（单价 = IPPM×0.7+OPPM×0.3，P2 后可换缓存感知有效单价；相等容差 1e-9）。

    **最低价层内的可靠性竞争（强制规则）**：先取 `bestDirect` 与 `bestCommunity`。若只有一类候选，直接选择该类；两类都有时，默认选择 `bestDirect`，**仅当** `communityQualityScore(bestCommunity) >= directQualityScore(bestDirect)` 时选择个人算力。等分时个人胜出，表示已有实测稳定性不低于 Direct 的个人节点可以承担请求；否则 Direct 优先，保证消费者默认获得更稳定、可控的固定供给。不得在比较前对个人候选使用 `pickSmart` 的会员加权随机阶段，以免较低分个人节点偶然越过 Direct。

    `communityQualityScore` 直接复用阶段 1 `perfScore`（容量、延迟、失败、在线稳定性、服务能力、带宽；不含会员权重）。新增 `directQualityScore`，使用同一组权重与归一化函数：容量=`capacityScore(MaxConns, Active)`，延迟=`latencyScore(LatencyEMA)`，失败=`failureScore(RecentFailures)`，稳定性=`ReliabilityEMA`，服务能力与带宽按 Direct 的受管基线取 1。`ReliabilityEMA` 是 DirectBackend 新增的原子 float64bits 字段：每次真实 Direct 请求成功采样 1、5xx/网络错误采样 0，健康检查仅更新可用状态，不得伪造成功样本；未初始化采用 0.5 中性值。这样 Direct 与个人的比较基于可解释、可观测的可靠性，而非仅凭 `Priority` 或“健康即满分”。

    个人算力没有合格候选时，合格 DirectBackend 必须作为兜底；反过来，Direct 都不合格而个人合格时选择个人；两池都无合格候选才返回 `503`。任何兜底目标仍必须通过用户价格帽，绝不能用超价格帽的 Direct 强行服务。失败时在同一价格层内排除已失败目标后重选；该层耗尽才溢出下一价格层，下一层同样执行上述竞争规则。
4. **balanced**：先 `LoadBalanceExcluding`，但 Predicate 后过滤 `perfScore >= LB_BALANCED_MIN_SCORE`（需要 Server 暴露该过滤，实现为 `LoadBalanceExcluding` 的变体参数）；无候选/失败溢出 Direct。
5. **容忍度**：`ExtendedChatRequest` 加 `MaxLatencyMs int`、`MinStability float64`（json omitempty）；透传到 LoadBalance 作为额外 Predicate（`LatencyEMA`、P0 缓存 stab 分）。
6. **tiers 暴露**：`/v1/models` 响应（或 market API）每个模型加 `tiers: ["direct","community"]`。
7. config：`RoutingDefault string`（默认 "stability"）、`LBBalancedMinScore float64`（默认 0.5）。

## 7. 编码顺序

M1：① direct_backend.go 模型+DB+单测 → ② Server 注册表+PickDirect+健康检查+单测 → ③ direct_chat.go+httptest 单测 → ④ chat.go/format_adapter.go 接入+config → ⑤ main.go CLI → ⑥ 集成测试。
M2：⑦ 偏好解析+stability/cost/balanced 分支+单测 → ⑧ 容忍度 Predicate → ⑨ tiers。

## 8. 单元测试清单

**internal/models/direct_backend_test.go（新）**，:memory: sqlite：

| 测试 | 断言 |
|---|---|
| `TestDirectBackendModelsRoundTrip` | Save→List 后 Models 从 ModelsJSON 还原 |
| `TestPickDirectFilters` | disabled 载入时排除；unhealthy/冷却/饱和（Active==MaxConns）/无该模型价格/超价格帽 各被过滤 |
| `TestPickDirectPriorityThenLoad` | 两层 Priority：低值层优先；同层选 Active/MaxConns 低者 |
| `TestPickDirectExclude` | exclude["direct:"+id] 生效；全排除返回 nil |
| `TestPickDirectNoSupply` | 注册表无该模型 → nil（退化路径） |
| `TestDirectCooldownAlwaysOn` | `LB_COOLDOWN_ENABLED=false` 时 DirectBackend 冷却仍生效 |
| `TestPickCheapestSamePriceDirectWins` | Direct 与个人名义单价相同，且个人 quality 低于 Direct → 选 Direct |
| `TestPickCheapestSamePriceCommunityWinsOnTie` | Direct 与个人同价且 quality 相等或个人更高 → 选个人 |
| `TestPickCheapestCommunityAbsentDirectFallback` | 无合格个人候选、Direct 通过价格帽与健康检查 → 选 Direct |
| `TestPickCheapestNeverBreaksPriceCap` | 个人无供给但所有 Direct 超价格帽 → 返回空候选，由调用方返回 503 |
| `TestDirectReliabilityEMA` | 成功采样提高、5xx/网络失败采样降低、健康检查不改变 EMA、未初始化为中性值 |

**internal/service/direct_chat_test.go（新）**，`httptest.Server` 模拟后端 + `gin.CreateTestContext` + :memory: sqlite 构造的 `models.Server`（只需 TokenUsageDB/UserDB/UserPriceCapDB 字段）：

| 测试 | 后端行为 | 断言 |
|---|---|---|
| `TestDirectChatNonStream` | 200 + 完整 JSON（含 usage.prompt_tokens_details.cached_tokens） | done=true；用户收到原样 body；token_usages 表 1 行 ClientID=="direct:xx"，in/out/cached 正确 |
| `TestDirectChatStream` | SSE 5 个 chunk + usage 尾块 + [DONE] | done=true；用户收到全部 chunk；计费落库 |
| `TestDirectChat4xx` | 400 + error json | done=true；状态码/呔 body 透传；**不**记 failures；无计费 |
| `TestDirectChat5xx` | 503 | done=false；failures+1；冷却触发；无输出写给用户 |
| `TestDirectChatTimeout` | 挂起 > 超时 | done=false；failures+1 |
| `TestDirectChatMidStreamCut` | 2 个 chunk 后断连 | done=true（已写出内容不可重试）；failures+1 |

**chat.go 接入测试**（internal/service 内）：`TestRetryLoopDirectThenFallback`——PickDirect 命中但 handleDirectChat 失败（5xx mock）→ 同一循环后续落到众包 mock（可用注册假 client 或断言 LoadBalanceExcluding 被到达，按现有 client_test.go 的测试基建取舍）。

## 9. 集成测试（手工脚本，写入 docs/impl/ 或 Makefile target）

前置：本地 Ollama（`http://127.0.0.1:11434/v1`）当固定后端。

1. `starfire add-backend -id=ollama-local -url=http://127.0.0.1:11434/v1 -key=x -maxconns=4 -priority=10 -models='[{"name":"qwen3:8b","IPPM":1.0,"OPPM":2.0,"CIPPM":0.2}]'` → `list-backends` 确认。
2. `$env:DIRECT_BACKENDS_ENABLED='true'; go run .`，日志确认载入 + 健康检查通过。
3. `curl /v1/chat/completions`（非流式 + 流式）：响应正常；日志显示走 direct 路径（无 WS fingerprint）；`token_usages` 落库 `client_id='direct:ollama-local'`；用户余额扣减正确。
4. **溢出验证**：`maxconns` 改 0（或并发 4+ 请求打满）→ 请求落众包 client（需一个真实 client 在线）；两池都无 → 503 "All clients failed"。
5. **故障隔离**：停掉 Ollama → 健康检查 30s 内标记 unhealthy，请求全部走众包；恢复后自动回来。
6. **回滚**：`DIRECT_BACKENDS_ENABLED=false` 重启 → 完全现路径（日志无 direct 字样）。

## 10. 验收标准

- [ ] 默认 env 全量回归通过（`go test -race ./internal/... && go build .`）。
- [ ] M1 集成测试 6 步全过；APIKey 不出现在任何日志。
- [ ] 计费公式与 WS 路径一致（同一模型同价格下 Cost 相同，SQL 抽查对比）。
- [ ] 无 Direct 供给的模型行为与现网 byte 级一致。
