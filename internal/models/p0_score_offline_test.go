package models

import (
	"testing"
	"time"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// ============ P0: 打分离线化 + 熔断冷却 单元测试 ============

// TestCachedScoresRoundTrip 验证 SetCachedScores/CachedScores 往返一致，新 client 返回 ok=false。
func TestCachedScoresRoundTrip(t *testing.T) {
	c := &Client{}
	if _, _, ok := c.CachedScores(); ok {
		t.Fatal("new client CachedScores should return ok=false")
	}
	c.SetCachedScores(0.7, 0.3)
	stab, serv, ok := c.CachedScores()
	if !ok {
		t.Fatal("after SetCachedScores, CachedScores should return ok=true")
	}
	if stab != 0.7 || serv != 0.3 {
		t.Fatalf("CachedScores = (%v,%v), want (0.7,0.3)", stab, serv)
	}
}

// TestCooldownTripAndExpire 验证：IncrFailures 触发冷却，到期后自动恢复。
func TestCooldownTripAndExpire(t *testing.T) {
	origEnabled := configs.Config.LBCooldownEnabled
	origBase := configs.Config.LBCooldownBaseMs
	configs.Config.LBCooldownEnabled = true
	configs.Config.LBCooldownBaseMs = 10
	defer func() {
		configs.Config.LBCooldownEnabled = origEnabled
		configs.Config.LBCooldownBaseMs = origBase
	}()

	c := &Client{}
	if c.InCooldown() {
		t.Fatal("new client should not be in cooldown")
	}
	c.IncrFailures()
	if !c.InCooldown() {
		t.Fatal("after IncrFailures, client should be in cooldown")
	}
	time.Sleep(15 * time.Millisecond)
	if c.InCooldown() {
		t.Fatal("cooldown should have expired after sleep")
	}
}

// TestCooldownExponential 验证指数退避：failures=1→base，=2→2×base，=3→4×base；超上限截断。
func TestCooldownExponential(t *testing.T) {
	origEnabled := configs.Config.LBCooldownEnabled
	origBase := configs.Config.LBCooldownBaseMs
	origMax := configs.Config.LBCooldownMaxMs
	configs.Config.LBCooldownEnabled = true
	configs.Config.LBCooldownBaseMs = 100
	configs.Config.LBCooldownMaxMs = 1000
	defer func() {
		configs.Config.LBCooldownEnabled = origEnabled
		configs.Config.LBCooldownBaseMs = origBase
		configs.Config.LBCooldownMaxMs = origMax
	}()

	c := &Client{}
	// failures=1 → base=100ms
	c.IncrFailures()
	d1 := time.Duration(c.CooldownUntil - time.Now().UnixNano())
	if d1 < 90*time.Millisecond || d1 > 110*time.Millisecond {
		t.Fatalf("failures=1 cooldown = %v, want ~100ms", d1)
	}
	// failures=2 → 2×base=200ms
	c.IncrFailures()
	d2 := time.Duration(c.CooldownUntil - time.Now().UnixNano())
	if d2 < 190*time.Millisecond || d2 > 210*time.Millisecond {
		t.Fatalf("failures=2 cooldown = %v, want ~200ms", d2)
	}
	// failures=3 → 4×base=400ms
	c.IncrFailures()
	d3 := time.Duration(c.CooldownUntil - time.Now().UnixNano())
	if d3 < 390*time.Millisecond || d3 > 410*time.Millisecond {
		t.Fatalf("failures=3 cooldown = %v, want ~400ms", d3)
	}
	// 大量失败 → 截断到 max=1000ms
	for i := 0; i < 20; i++ {
		c.IncrFailures()
	}
	dMax := time.Duration(c.CooldownUntil - time.Now().UnixNano())
	if dMax > 1010*time.Millisecond {
		t.Fatalf("cooldown should be capped at max, got %v", dMax)
	}
}

// TestResetClearsCooldown 验证：ResetFailures 清零失败数并清除冷却。
func TestResetClearsCooldown(t *testing.T) {
	origEnabled := configs.Config.LBCooldownEnabled
	origBase := configs.Config.LBCooldownBaseMs
	configs.Config.LBCooldownEnabled = true
	configs.Config.LBCooldownBaseMs = 10000
	defer func() {
		configs.Config.LBCooldownEnabled = origEnabled
		configs.Config.LBCooldownBaseMs = origBase
	}()

	c := &Client{}
	c.IncrFailures()
	if !c.InCooldown() {
		t.Fatal("should be in cooldown after IncrFailures")
	}
	c.ResetFailures()
	if c.InCooldown() {
		t.Fatal("ResetFailures should clear cooldown")
	}
	if c.GetFailures() != 0 {
		t.Fatalf("GetFailures = %d, want 0", c.GetFailures())
	}
}

// TestCooldownDisabledNoTrip 验证：开关关闭时 IncrFailures 不触发冷却（默认行为不变）。
func TestCooldownDisabledNoTrip(t *testing.T) {
	origEnabled := configs.Config.LBCooldownEnabled
	configs.Config.LBCooldownEnabled = false
	defer func() { configs.Config.LBCooldownEnabled = origEnabled }()

	c := &Client{}
	c.IncrFailures()
	if c.InCooldown() {
		t.Fatal("cooldown should not trip when disabled")
	}
	if c.GetFailures() != 1 {
		t.Fatalf("GetFailures = %d, want 1", c.GetFailures())
	}
}

// ============ P0: 批量查询 单元测试 ============

// TestGetStatsBatch 验证批量拉取统计：返回存在的记录，缺失 ID 不在结果里。
func TestGetStatsBatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	csdb := NewClientStatsDB(db)
	for _, id := range []string{"c1", "c2", "c3"} {
		if _, err := csdb.GetOrCreate(id); err != nil {
			t.Fatalf("create stats for %s: %v", id, err)
		}
	}
	got, err := csdb.GetStatsBatch([]string{"c1", "c2", "c4"})
	if err != nil {
		t.Fatalf("GetStatsBatch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetStatsBatch len = %d, want 2", len(got))
	}
	if _, ok := got["c1"]; !ok {
		t.Fatal("c1 missing from batch result")
	}
	if _, ok := got["c2"]; !ok {
		t.Fatal("c2 missing from batch result")
	}
	if _, ok := got["c4"]; ok {
		t.Fatal("c4 (nonexistent) should not be in batch result")
	}
}

// TestGetTotalTokensByClientIDs 验证按 client 聚合 total_tokens，since 过滤生效，空结果不报错。
func TestGetTotalTokensByClientIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	tdb := NewTokenUsageDB(db)
	now := time.Now()
	usages := []*TokenUsage{
		{RequestID: "r1", UserID: "u1", ClientID: "c1", Model: "m", TotalTokens: 100, Timestamp: now.Add(-2 * time.Hour)},
		{RequestID: "r2", UserID: "u1", ClientID: "c1", Model: "m", TotalTokens: 50, Timestamp: now.Add(-1 * time.Hour)},
		{RequestID: "r3", UserID: "u2", ClientID: "c2", Model: "m", TotalTokens: 200, Timestamp: now.Add(-30 * time.Minute)},
		{RequestID: "r4", UserID: "u2", ClientID: "c2", Model: "m", TotalTokens: 300, Timestamp: now.Add(-10 * time.Minute)},
	}
	for _, u := range usages {
		if err := tdb.SaveTokenUsage(u); err != nil {
			t.Fatalf("save usage: %v", err)
		}
	}

	// 全部历史
	got, err := tdb.GetTotalTokensByClientIDs([]string{"c1", "c2"}, time.Time{})
	if err != nil {
		t.Fatalf("GetTotalTokensByClientIDs: %v", err)
	}
	if got["c1"] != 150 {
		t.Fatalf("c1 total = %d, want 150", got["c1"])
	}
	if got["c2"] != 500 {
		t.Fatalf("c2 total = %d, want 500", got["c2"])
	}

	// since 过滤：只统计最近 45 分钟
	got2, err := tdb.GetTotalTokensByClientIDs([]string{"c1", "c2"}, now.Add(-45*time.Minute))
	if err != nil {
		t.Fatalf("GetTotalTokensByClientIDs since: %v", err)
	}
	if got2["c1"] != 0 {
		t.Fatalf("c1 since-filtered total = %d, want 0", got2["c1"])
	}
	if got2["c2"] != 500 {
		t.Fatalf("c2 since-filtered total = %d, want 500", got2["c2"])
	}

	// 空结果不报错
	got3, err := tdb.GetTotalTokensByClientIDs([]string{"nonexistent"}, time.Time{})
	if err != nil {
		t.Fatalf("GetTotalTokensByClientIDs empty: %v", err)
	}
	if len(got3) != 0 {
		t.Fatalf("empty result len = %d, want 0", len(got3))
	}
}

// ============ P0: perfScore 离线分支 + ScoreRefresher 单元测试 ============

// TestPerfScoreUsesCachedSlowDims 验证：离线模式下 perfScore 读缓存分，未刷新用中性值。
func TestPerfScoreUsesCachedSlowDims(t *testing.T) {
	origOffline := configs.Config.LBScoreOffline
	origNeutral := configs.Config.LBNeutralScore
	configs.Config.LBScoreOffline = true
	configs.Config.LBNeutralScore = 0.5
	defer func() {
		configs.Config.LBScoreOffline = origOffline
		configs.Config.LBNeutralScore = origNeutral
	}()

	server := newTestSmartServer(t) // 无 DB（ClientStatsDB/TokenUsageDB 均为 nil）
	_, _, _, wStab, wServ, _ := server.smartWeights()

	// 未刷新 client：stab/serv 贡献 = 0.5×(wStab+wServ)
	c := makeClient("a", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000)
	// 让快变维度贡献为 0，便于精确断言 stab/serv 贡献
	c.ActiveConnections = 3       // capacityScore=0
	c.Latency = public.MAXLATENCE // latencyScore=0
	c.BandwidthMbps = 0           // bandwidthScore=0（默认 10/50=0.2，非 0，需注意）
	// 为精确断言，直接比较"刷新前后差值"而非绝对值
	before := server.perfScore(c)

	c.SetCachedScores(1, 0)
	after := server.perfScore(c)

	// 差值应等于 (1-0.5)*wStab + (0-0.5)*wServ
	want := 0.5*wStab - 0.5*wServ
	got := after - before
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("perfScore delta = %v, want %v (wStab=%v wServ=%v)", got, want, wStab, wServ)
	}
}

// TestPerfScoreOfflineNoDBAccess 验证：离线模式下 ClientStatsDB/TokenUsageDB 为 nil 不 panic。
func TestPerfScoreOfflineNoDBAccess(t *testing.T) {
	origOffline := configs.Config.LBScoreOffline
	configs.Config.LBScoreOffline = true
	defer func() { configs.Config.LBScoreOffline = origOffline }()

	server := newTestSmartServer(t) // 无 DB
	c := makeClient("a", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000)
	if s := server.perfScore(c); s < 0 || s > 1 {
		t.Fatalf("perfScore offline = %v, want in [0,1]", s)
	}
}

// TestScoreRefresherRefreshOnce 验证：refreshScoresOnce 批量刷新后 client 缓存分等于手算值。
func TestScoreRefresherRefreshOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	csdb := NewClientStatsDB(db)
	tdb := NewTokenUsageDB(db)

	// 预置 stats：c1 在线 3600s、掉线 0；c2 在线 7200s、掉线 2
	csdb.GetOrCreate("c1")
	csdb.GetOrCreate("c2")
	db.Model(&ClientStats{}).Where("client_id = ?", "c1").Updates(map[string]interface{}{
		"total_online_seconds": 3600, "disconnect_count": 0,
	})
	db.Model(&ClientStats{}).Where("client_id = ?", "c2").Updates(map[string]interface{}{
		"total_online_seconds": 7200, "disconnect_count": 2,
	})

	// 预置 token usage：c1 贡献 1000 tokens，c2 贡献 2000 tokens
	now := time.Now()
	for _, u := range []*TokenUsage{
		{RequestID: "r1", UserID: "u1", ClientID: "c1", Model: "m", TotalTokens: 1000, Timestamp: now},
		{RequestID: "r2", UserID: "u2", ClientID: "c2", Model: "m", TotalTokens: 2000, Timestamp: now},
	} {
		if err := tdb.SaveTokenUsage(u); err != nil {
			t.Fatalf("save usage: %v", err)
		}
	}

	server := newTestSmartServer(t)
	server.ClientStatsDB = csdb
	server.TokenUsageDB = tdb

	c1 := &Client{ID: "c1", Models: []*public.Model{{Name: "model-a"}}}
	c2 := &Client{ID: "c2", Models: []*public.Model{{Name: "model-a"}}}
	server.RegisterModel(&public.Model{Name: "model-a"}, c1)
	server.RegisterModel(&public.Model{Name: "model-a"}, c2)

	server.refreshScoresOnce()

	wantStab1 := stabilityScore(3600, 0)
	wantServ1 := serviceScore(1000, 3600)
	stab1, serv1, ok1 := c1.CachedScores()
	if !ok1 || stab1 != wantStab1 || serv1 != wantServ1 {
		t.Fatalf("c1 cached = (%v,%v,ok=%v), want (%v,%v,true)", stab1, serv1, ok1, wantStab1, wantServ1)
	}

	wantStab2 := stabilityScore(7200, 2)
	wantServ2 := serviceScore(2000, 7200)
	stab2, serv2, ok2 := c2.CachedScores()
	if !ok2 || stab2 != wantStab2 || serv2 != wantServ2 {
		t.Fatalf("c2 cached = (%v,%v,ok=%v), want (%v,%v,true)", stab2, serv2, ok2, wantStab2, wantServ2)
	}
}

// TestCooldownFilteredInLoadBalance 验证：冷却中的 client 在 LoadBalanceExcluding 中被过滤，
// 且不被 RemoveClient 删除；冷却到期后可再次被选。
func TestCooldownFilteredInLoadBalance(t *testing.T) {
	origEnabled := configs.Config.LBCooldownEnabled
	origBase := configs.Config.LBCooldownBaseMs
	configs.Config.LBCooldownEnabled = true
	configs.Config.LBCooldownBaseMs = 20
	defer func() {
		configs.Config.LBCooldownEnabled = origEnabled
		configs.Config.LBCooldownBaseMs = origBase
	}()

	server := newTestSmartServer(t)
	server.LoadBalanceAlgorithm = "smart"
	model := &public.Model{Name: "model-a"}
	dummyConn := &websocket.Conn{}
	healthy := &Client{ID: "healthy", Status: "online", ControlConn: dummyConn, Models: []*public.Model{{Name: "model-a", IPPM: 1, OPPM: 1}}}
	cooldown := &Client{ID: "cooldown", Status: "online", ControlConn: dummyConn, Models: []*public.Model{{Name: "model-a", IPPM: 1, OPPM: 1}}}
	server.RegisterModel(model, healthy)
	server.RegisterModel(model, cooldown)

	// 触发冷却
	cooldown.IncrFailures()
	if !cooldown.InCooldown() {
		t.Fatal("cooldown client should be in cooldown")
	}

	// 冷却中：只应返回 healthy
	winner := server.LoadBalanceExcluding("model-a", "", nil)
	if winner == nil || winner.ID != "healthy" {
		t.Fatalf("LoadBalanceExcluding during cooldown = %v, want healthy", winner)
	}
	// 冷却 client 未被删除
	if got := server.GetClientByModel("model-a", "cooldown"); got == nil {
		t.Fatal("cooldown client should NOT be removed")
	}

	// 冷却到期后：两个 client 都可被选
	time.Sleep(25 * time.Millisecond)
	if cooldown.InCooldown() {
		t.Fatal("cooldown should have expired")
	}
	winner2 := server.LoadBalanceExcluding("model-a", "", nil)
	if winner2 == nil {
		t.Fatal("LoadBalanceExcluding after cooldown should return a client")
	}
}
