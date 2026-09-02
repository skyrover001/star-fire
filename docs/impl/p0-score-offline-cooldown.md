# P0 实施规格：打分离线化 + 失败熔断冷却

> 上游设计：`docs/scale-lb-roadmap-design.md` P0 章节
> 状态：待实施 | 依赖：无 | 回滚：`LB_SCORE_OFFLINE=false` + `LB_COOLDOWN_ENABLED=false`

## 0. 现状代码事实（已核实，2026-09-02）

| 事实 | 位置 |
|---|---|
| `perfScore(c *Client)` 每次调用做 2 次 DB 查询：`s.ClientStatsDB.GetStats(c.ID)` + `s.TokenUsageDB.GetIncomeTokenUsage([]string{c.ID}, time.Time{}, time.Now())` 全量行扫描求和 | `internal/models/server.go` `perfScore`（约 L545–576） |
| 打分公式：`wCap*capacityScore(maxConn, active) + wLat*latencyScore(lat) + wFail*failureScore(fails) + wStab*stabilityScore(onlineSec, disconnects) + wServ*serviceScore(tokens, onlineSec) + wBw*bandwidthScore(bw)` | 同上 |
| `clientHealthy(c, model)` 是**包级函数**（非方法），检查 status/ControlConn/latency | `server.go` 约 L195 |
| `IncrFailures()`/`ResetFailures()`/`GetFailures()` 是 `Client` 上的原子方法 | `internal/models/client.go` L111–121 |
| `IncrFailures` 调用点：**只有 2 个文件**（设计稿写的 embedding.go 是错的，那里没有）：`chat.go` 8 处（L241/254/264/274/290/309/336/345）、`format_adapter.go` 8 处（L203/216/226/236/250/268/292/301）；`ResetFailures`：chat.go L304、format_adapter.go L263 | grep 核实 |
| 4xx 判定 `isClientRequestError(content)` 分支**不调用** IncrFailures（chat.go L316 附近） | `chat.go` |
| `NewServer()` 已有一个 1h ticker goroutine（CleanupExpiredTokens + checkMembershipExpiry），新 ticker 与其并列 | `server.go` L141–150 |
| 在线 client 快照：`s.clients.Load().(map[string]map[string]*Client)`（model → clientID → *Client） | `server.go` |
| config 模式：`Configuration` 结构体字段 + `loadConfig()` 里 `getEnv`+`strconv`，包级 `var Config` | `config/config.go` |
| 测试基建：`internal/models/server_smart_test.go` 直接测包级评分函数；DB 测试用 `github.com/glebarez/sqlite` + `gorm.Open(sqlite.Open(":memory:"))` | 已有 |

## 1. 改动一：配置（config/config.go）

`Configuration` 增加字段：

```go
// P0 打分离线化 + 熔断冷却
LBScoreOffline       bool // LB_SCORE_OFFLINE，默认 false
ScoreRefreshInterval int  // SCORE_REFRESH_INTERVAL 秒，默认 30
LBCooldownEnabled    bool // LB_COOLDOWN_ENABLED，默认 false
LBCooldownBaseMs     int  // LB_COOLDOWN_BASE_MS，默认 5000
LBCooldownMaxMs      int  // LB_COOLDOWN_MAX_MS，默认 300000
LBNeutralScore       float64 // LB_NEUTRAL_SCORE，默认 0.5（新 client stab/serv 中性值）
```

`loadConfig()` 中按现有模式解析（bool 用 `strconv.ParseBool`，非法值回退默认）；负数/0 回退默认。

## 2. 改动二：Client 新字段与方法（internal/models/client.go）

在 `Client` struct「not in db」区追加（全部 `gorm:"-"`）：

```go
// P0: perfScore 慢变维度离线缓存（math.Float64bits 存储，原子读写；ScoreUpdatedAt==0 表示未初始化）
CachedStabilityScore uint64 `json:"-" gorm:"-"`
CachedServiceScore   uint64 `json:"-" gorm:"-"`
ScoreUpdatedAt       int64  `json:"-" gorm:"-"` // unix 秒

// P0: 熔断冷却截止时间（unix 纳秒，0=无冷却）
CooldownUntil int64 `json:"-" gorm:"-"`
```

新方法（放在 `GetFailures` 之后）：

```go
// SetCachedScores 由 ScoreRefresher 写入慢变维度分（stab/serv ∈ [0,1]）。
func (c *Client) SetCachedScores(stab, serv float64) {
    atomic.StoreUint64(&c.CachedStabilityScore, math.Float64bits(stab))
    atomic.StoreUint64(&c.CachedServiceScore, math.Float64bits(serv))
    atomic.StoreInt64(&c.ScoreUpdatedAt, time.Now().Unix())
}

// CachedScores 返回缓存分；ok=false 表示从未刷新过（调用方应使用中性值）。
func (c *Client) CachedScores() (stab, serv float64, ok bool) {
    if atomic.LoadInt64(&c.ScoreUpdatedAt) == 0 {
        return 0, 0, false
    }
    return math.Float64frombits(atomic.LoadUint64(&c.CachedStabilityScore)),
        math.Float64frombits(atomic.LoadUint64(&c.CachedServiceScore)), true
}

// TripCooldown 按 RecentFailures 指数退避设置冷却：base × 2^(failures-1)，上限 max。
func (c *Client) TripCooldown() {
    base := time.Duration(configs.Config.LBCooldownBaseMs) * time.Millisecond
    max := time.Duration(configs.Config.LBCooldownMaxMs) * time.Millisecond
    if base <= 0 { base = 5 * time.Second }
    if max <= 0 { max = 5 * time.Minute }
    n := c.GetFailures()
    if n < 1 { n = 1 }
    d := base << uint(n-1) // 溢出防护：n-1 > 30 时直接取 max
    if n > 30 || d > max || d <= 0 { d = max }
    atomic.StoreInt64(&c.CooldownUntil, time.Now().Add(d).UnixNano())
}

// InCooldown 是否处于冷却期。
func (c *Client) InCooldown() bool {
    u := atomic.LoadInt64(&c.CooldownUntil)
    return u != 0 && time.Now().UnixNano() < u
}

// ClearCooldown 清除冷却（成功恢复）。
func (c *Client) ClearCooldown() {
    atomic.StoreInt64(&c.CooldownUntil, 0)
}
```

**冷却触发点集成方案（比路线图更简单，选定此方案）**：不改 16 个调用点，改 2 个方法本体：

```go
func (c *Client) IncrFailures() {
    atomic.AddInt32(&c.RecentFailures, 1)
    if configs.Config.LBCooldownEnabled {
        c.TripCooldown()
    }
}

func (c *Client) ResetFailures() {
    atomic.StoreInt32(&c.RecentFailures, 0)
    c.ClearCooldown()
}
```

语义核对：4xx（`isClientRequestError`）路径本来就不调 `IncrFailures` ⇒ 自动满足"4xx 不触发冷却"；成功路径调 `ResetFailures` ⇒ 自动满足"成功即恢复"。`chat.go`/`format_adapter.go` **零改动**。

## 3. 改动三：perfScore 读缓存 + clientHealthy 冷却过滤（internal/models/server.go）

### 3.1 perfScore

```go
func (s *Server) perfScore(c *Client) float64 {
    wCap, wLat, wFail, wStab, wServ, wBw := s.smartWeights()
    maxConn := s.effectiveMaxConnections(c)

    var stabS, servS float64
    if configs.Config.LBScoreOffline {
        // 离线模式：读缓存分；未初始化用中性值
        var ok bool
        stabS, servS, ok = c.CachedScores()
        if !ok {
            neutral := configs.Config.LBNeutralScore
            if neutral <= 0 || neutral > 1 { neutral = 0.5 }
            stabS, servS = neutral, neutral
        }
    } else {
        // 现行为：热路径 DB 查询（保留原代码块原样搬入此分支）
        var totalOnlineSec, disconnectCount, totalTokens int64
        ...（现有 GetStats + GetIncomeTokenUsage 逻辑，不改）...
        stabS = stabilityScore(totalOnlineSec, disconnectCount)
        servS = serviceScore(totalTokens, totalOnlineSec)
    }

    bw := c.BandwidthMbps
    if bw <= 0 { bw = configs.Config.ClientBandwidthMbps }

    return wCap*capacityScore(maxConn, c.GetActiveConnections()) +
        wLat*latencyScore(c.GetLatency()) +
        wFail*failureScore(c.GetFailures()) +
        wStab*stabS + wServ*servS +
        wBw*bandwidthScore(bw)
}
```

注意：离线分支里 `stabilityScore`/`serviceScore` 的计算移到 ScoreRefresher（§4），缓存里存的是**已算好的 [0,1] 分**，不是原始输入。

### 3.2 clientHealthy

在函数开头（读 maxLatencyMS 之前）加：

```go
if configs.Config.LBCooldownEnabled && c.InCooldown() {
    return false
}
```

**副作用核对**：`LoadBalanceExcluding` 里 `!clientHealthy(...)` 的 client 会进 `dead` 列表被 `RemoveClient` 删除并 `notifyLatencyExceeded`——冷却中的 client **不能**走这条路（冷却是暂时隔离，不是摘除）。因此 `LoadBalanceExcluding` 的循环体需改为：

```go
if configs.Config.LBCooldownEnabled && c.InCooldown() {
    continue // 冷却中：跳过但不删除、不通知
}
if !clientHealthy(c, model) {
    ...现有 dead 处理...
}
```

即：冷却检查放在 `LoadBalanceExcluding`（和 `LoadBalanceEmbedding` 同样位置），`clientHealthy` 本身**不改**（避免其他调用方误删）。这是对路线图 P0.2C 的修正。

## 4. 改动四：批量查询 + ScoreRefresher

### 4.1 internal/models/client_stats.go 新增

```go
// GetStatsBatch 批量拉取统计（不存在的 ID 不在结果里，调用方按缺失处理）。
func (csdb *ClientStatsDB) GetStatsBatch(clientIDs []string) (map[string]*ClientStats, error) {
    var rows []*ClientStats
    if err := csdb.db.Where("client_id IN ?", clientIDs).Find(&rows).Error; err != nil {
        return nil, err
    }
    out := make(map[string]*ClientStats, len(rows))
    for _, r := range rows { out[r.ClientID] = r }
    return out, nil
}
```

### 4.2 internal/models/token_usage.go 新增

```go
// GetTotalTokensByClientIDs 按 client 聚合 total_tokens（since 为零值 = 全部历史，对齐现 perfScore 语义）。
func (tdb *TokenUsageDB) GetTotalTokensByClientIDs(clientIDs []string, since time.Time) (map[string]int64, error) {
    type row struct { ClientID string; Total int64 }
    var rows []row
    q := tdb.db.Model(&TokenUsage{}).
        Select("client_id, SUM(total_tokens) AS total").
        Where("client_id IN ?", clientIDs)
    if !since.IsZero() { q = q.Where("timestamp >= ?", since) }
    if err := q.Group("client_id").Scan(&rows).Error; err != nil { return nil, err }
    out := make(map[string]int64, len(rows))
    for _, r := range rows { out[r.ClientID] = r.Total }
    return out, nil
}
```

命中索引 `idx_token_usages_client_timestamp`（已存在）。

### 4.3 internal/models/server.go：ScoreRefresher

`NewServer()` 里现有 1h ticker goroutine 之后追加：

```go
if configs.Config.LBScoreOffline {
    go server.runScoreRefresher()
}
```

```go
// runScoreRefresher 周期性批量刷新在线 client 的慢变维度分（stab/serv）。
func (s *Server) runScoreRefresher() {
    interval := time.Duration(configs.Config.ScoreRefreshInterval) * time.Second
    if interval <= 0 { interval = 30 * time.Second }
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    s.refreshScoresOnce() // 启动先刷一轮，缩短冷启动中性值窗口
    for range ticker.C {
        s.refreshScoresOnce()
    }
}

func (s *Server) refreshScoresOnce() {
    all := s.clients.Load().(map[string]map[string]*Client)
    // 去重收集在线 client（同一 client 可能注册多个模型）
    uniq := map[string]*Client{}
    for _, inner := range all {
        for id, c := range inner { uniq[id] = c }
    }
    if len(uniq) == 0 { return }
    ids := make([]string, 0, len(uniq))
    for id := range uniq { ids = append(ids, id) }

    statsMap, err := s.ClientStatsDB.GetStatsBatch(ids)
    if err != nil { log.Printf("score refresher: stats batch failed: %v", err); return }
    tokensMap, err := s.TokenUsageDB.GetTotalTokensByClientIDs(ids, time.Time{})
    if err != nil { log.Printf("score refresher: tokens batch failed: %v", err); return }

    for id, c := range uniq {
        var onlineSec, disconnects int64
        if st, ok := statsMap[id]; ok {
            onlineSec, disconnects = st.TotalOnlineSeconds, st.DisconnectCount
        }
        c.SetCachedScores(
            stabilityScore(onlineSec, disconnects),
            serviceScore(tokensMap[id], onlineSec),
        )
    }
}
```

单轮失败只打日志（旧缓存分继续生效）。ID 超过 ~1000 时可分批 IN（首期不做，SQLite/MySQL 均可承受千级 IN）。

## 5. 编码顺序（可独立提交的 4 个 commit）

1. config 字段 + Client 新字段/方法（含 IncrFailures/ResetFailures 集成）+ 单测。
2. `GetStatsBatch` + `GetTotalTokensByClientIDs` + 单测。
3. `perfScore` 双分支 + ScoreRefresher + 单测。
4. `LoadBalanceExcluding`/`LoadBalanceEmbedding` 冷却过滤 + 单测 + 仿真更新。

## 6. 单元测试清单

新增/修改于 `internal/models/`（沿用现有测试风格：直接改 `configs.Config` 字段，`defer` 恢复）：

| 测试 | 文件 | 断言 |
|---|---|---|
| `TestCachedScoresRoundTrip` | client_test 或 server_smart_test.go | `SetCachedScores(0.7,0.3)` 后 `CachedScores()` 返回 (0.7,0.3,true)；新 client 返回 ok=false |
| `TestPerfScoreUsesCachedSlowDims` | server_smart_test.go | `LBScoreOffline=true`、Server 无 DB（`&Server{}`）：未刷新 client 的 stab/serv 贡献 = 0.5×(wStab+wServ)；`SetCachedScores(1,0)` 后精确等于 wStab×1 |
| `TestPerfScoreOfflineNoDBAccess` | server_smart_test.go | `LBScoreOffline=true` 时 `ClientStatsDB=nil`/`TokenUsageDB=nil` 不 panic（离线分支不碰 DB） |
| `TestCooldownTripAndExpire` | server_smart_test.go | `LBCooldownEnabled=true, BaseMs=10`：`IncrFailures` 后 `InCooldown()==true`；sleep 15ms 后 false |
| `TestCooldownExponential` | 同上 | failures=1→base，=2→2×base，=3→4×base（读 `CooldownUntil` 差值断言，容差 ±5ms）；≥上限截断为 MaxMs |
| `TestResetClearsCooldown` | 同上 | `IncrFailures`→`ResetFailures`→`InCooldown()==false` 且 `GetFailures()==0` |
| `TestCooldownFilteredInLoadBalance` | 同上 | 构造 2 个 client（用 `RegisterModel` 注册），一个 `TripCooldown`：`LoadBalanceExcluding` 只返回另一个；且冷却 client **未被 RemoveClient**（再次 Load 快照仍存在）；冷却到期后可再次被选 |
| `TestGetStatsBatch` | client_stats_test.go（新） | :memory: sqlite，3 条记录 + 1 个不存在 ID：返回 map 3 条 |
| `TestGetTotalTokensByClientIDs` | token_usage_test.go（新） | 2 个 client 各 2 条 usage：SUM 正确；since 过滤生效；空结果不报错 |
| `TestScoreRefresherRefreshOnce` | server_smart_test.go | :memory: sqlite 构造 Server（ClientStatsDB+TokenUsageDB）+ RegisterModel 2 个 client + 预置 stats/usage 行 → `refreshScoresOnce()` → 两个 client `CachedScores()` ok=true 且值等于 `stabilityScore`/`serviceScore` 手算值 |

全部跑：`go test -race ./internal/models/`。

## 7. 集成测试

1. **开关关闭回归**（默认 env）：`go test ./internal/... ./api/... && go build .`，行为与现网完全一致（离线分支不进）。
2. **打开开关手工验证**：`$env:LB_SCORE_OFFLINE='true'; $env:LB_COOLDOWN_ENABLED='true'; go run .` + 一个真实 client 连接：
   - 日志确认 refresher 启动、30s 一轮无报错；
   - 临时在 `GetStats`/`GetIncomeTokenUsage` 加计数日志，发起 20 个 chat 请求，确认路由期间两者调用次数为 0（仅 refresher 每轮 2 次批量）；
   - 杀掉 client 后端模型（制造 MODEL_ERROR）：观察该 client 进入冷却、下一请求路由到其他 client 或 503；冷却到期自动恢复。
3. **仿真对比**：`docs/lb_saturation_test.py` 加 cooldown 模型（失败节点隔离 base×2^n），对比失败节点重复选中次数（预期显著下降），结果记录到本文档附录。

## 8. 验收标准

- [ ] 默认 env 下全量测试通过、行为无变化（`go test -race ./internal/... `，重点回归 `server_smart_test.go`/`server_price_test.go`）。
- [ ] `LB_SCORE_OFFLINE=true` 时路由热路径零 DB 查询（计数日志验证）。
- [ ] 冷却生效/到期恢复/成功清零 三条链路手工验证通过。
- [ ] `go vet ./internal/models/` 干净；`go build .` exit 0。
