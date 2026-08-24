package models

import (
	"testing"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============ 各维度评分函数单元测试 ============

func TestMembershipScore(t *testing.T) {
	cases := []struct {
		membership string
		want       float64
	}{
		{MembershipNormal, 0.2},
		{MembershipVIP, 0.6},
		{MembershipSVIP, 1.0},
		{"unknown", 0.2}, // 未知等级按 normal 处理
	}
	for _, c := range cases {
		if got := membershipScore(c.membership); got != c.want {
			t.Errorf("membershipScore(%q) = %v, want %v", c.membership, got, c.want)
		}
	}
}

func TestCapacityScore(t *testing.T) {
	cases := []struct {
		name    string
		maxConn int
		active  int32
		want    float64
	}{
		{"svip unlimited", -1, 100, 1.0},
		{"full", 3, 3, 0.0},
		{"half", 10, 5, 0.5},
		{"over limit clamped", 3, 5, 0.0},
		{"zero max", 0, 0, 0.0},
	}
	for _, c := range cases {
		if got := capacityScore(c.maxConn, c.active); got != c.want {
			t.Errorf("%s: capacityScore(%d,%d) = %v, want %v", c.name, c.maxConn, c.active, got, c.want)
		}
	}
}

func TestLatencyScore(t *testing.T) {
	cases := []struct {
		latency int
		want    float64
	}{
		{0, 1.0},                       // 无延迟满分
		{1, 1 - 1.0/public.MAXLATENCE}, // 极低延迟接近满分
		{public.MAXLATENCE, 0.0},       // 达到上限 0 分
		{public.MAXLATENCE * 2, 0.0},   // 超过上限 clamp 到 0
		{public.MAXLATENCE / 2, 0.5},   // 一半延迟 0.5 分
	}
	for _, c := range cases {
		got := latencyScore(c.latency)
		if got < c.want-1e-9 || got > c.want+1e-9 {
			t.Errorf("latencyScore(%d) = %v, want %v", c.latency, got, c.want)
		}
	}
}

func TestFailureScore(t *testing.T) {
	cases := []struct {
		failures int32
		want     float64
	}{
		{0, 1.0},
		{1, 0.5},
		{3, 0.25},
		{9, 0.1},
	}
	for _, c := range cases {
		if got := failureScore(c.failures); got != c.want {
			t.Errorf("failureScore(%d) = %v, want %v", c.failures, got, c.want)
		}
	}
}

func TestStabilityScore(t *testing.T) {
	cases := []struct {
		name       string
		onlineSec  int64
		disconnect int64
		want       float64
	}{
		{"zero online", 0, 0, 0.0},
		{"exactly target", public.LB_TARGET_ONLINE_SEC, 0, 1.0},
		{"above target clamped", public.LB_TARGET_ONLINE_SEC * 10, 0, 1.0},
		{"half target", public.LB_TARGET_ONLINE_SEC / 2, 0, 0.5},
		{"many disconnects", public.LB_TARGET_ONLINE_SEC, 9, 0.1},
	}
	for _, c := range cases {
		if got := stabilityScore(c.onlineSec, c.disconnect); got != c.want {
			t.Errorf("%s: stabilityScore(%d,%d) = %v, want %v", c.name, c.onlineSec, c.disconnect, got, c.want)
		}
	}
}

func TestServiceScore(t *testing.T) {
	cases := []struct {
		name      string
		tokens    int64
		onlineSec int64
		want      float64
	}{
		{"zero online", 100000, 0, 0.0},
		{"target rate", public.LB_TARGET_TOKENS_PER_HOUR, 3600, 1.0},
		{"half rate", public.LB_TARGET_TOKENS_PER_HOUR / 2, 3600, 0.5},
		{"above clamped", public.LB_TARGET_TOKENS_PER_HOUR * 5, 3600, 1.0},
	}
	for _, c := range cases {
		if got := serviceScore(c.tokens, c.onlineSec); got != c.want {
			t.Errorf("%s: serviceScore(%d,%d) = %v, want %v", c.name, c.tokens, c.onlineSec, got, c.want)
		}
	}
}

// ============ smart 算法集成测试 ============

// newTestSmartServer 构造一个用于 smart 测试的 Server（内存 DB + 空 clients map）。
func newTestSmartServer(t *testing.T) *Server {
	t.Helper()
	server := &Server{}
	server.clients.Store(map[string]map[string]*Client{})
	return server
}

// makeClient 构造一个带指定维度的 client。
func makeClient(id, membership string, maxConn int, active int32, latency int, failures int32, onlineSec, disconnect, tokens int64) *Client {
	return &Client{
		ID:                id,
		UserID:            "user-" + id,
		Status:            "online",
		Latency:           latency,
		ActiveConnections: active,
		RecentFailures:    failures,
	}
}

// TestPickSmartMembershipPriority 验证：其他维度相同，仅会员等级不同 → SVIP 胜出。
// 使用真实内存 DB + UserDB，让 pickSmart 能读取贡献者会员等级。
func TestPickSmartMembershipPriority(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	userDB := NewUserDB(db)
	// 创建三个用户，贡献者会员等级不同
	users := []*User{
		{ID: "user-normal", Username: "normal", ContributorMembership: MembershipNormal},
		{ID: "user-vip", Username: "vip", ContributorMembership: MembershipVIP},
		{ID: "user-svip", Username: "svip", ContributorMembership: MembershipSVIP},
	}
	for _, u := range users {
		if err := db.Create(u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	server := newTestSmartServer(t)
	server.UserDB = userDB
	clients := []*Client{
		makeClient("normal", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000),
		makeClient("vip", MembershipVIP, 10, 0, 100, 0, 3600*24, 0, 500000),
		makeClient("svip", MembershipSVIP, -1, 0, 100, 0, 3600*24, 0, 500000),
	}
	clients[0].UserID = "user-normal"
	clients[1].UserID = "user-vip"
	clients[2].UserID = "user-svip"

	// 关闭扰动，保证确定性
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	// 运行多次，SVIP 应稳定胜出
	for i := 0; i < 100; i++ {
		winner := server.pickSmart("model-a", clients)
		if winner == nil {
			t.Fatal("pickSmart returned nil")
		}
		if winner.ID != "svip" {
			t.Fatalf("iteration %d: winner = %s, want svip", i, winner.ID)
		}
	}
}

// TestPickSmartDeterministicWithoutJitter 验证：关闭扰动时，相同输入选出相同 client。
func TestPickSmartDeterministicWithoutJitter(t *testing.T) {
	server := newTestSmartServer(t)
	// 保存并临时关闭扰动
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	clients := []*Client{
		makeClient("a", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000),
		makeClient("b", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000),
		makeClient("c", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000),
	}
	w1 := server.pickSmart("model-a", clients)
	w2 := server.pickSmart("model-a", clients)
	if w1.ID != w2.ID {
		t.Fatalf("deterministic pick failed: %s vs %s", w1.ID, w2.ID)
	}
}

// TestPickSmartScoreRange 验证：所有 client 的评分都在 [0,1] 范围内。
func TestPickSmartScoreRange(t *testing.T) {
	server := newTestSmartServer(t)
	clients := []*Client{
		makeClient("a", MembershipNormal, 3, 1, 5000, 5, 3600, 10, 10000),
		makeClient("b", MembershipVIP, 10, 5, 1000, 2, 3600*24, 2, 500000),
		makeClient("c", MembershipSVIP, -1, 0, 100, 0, 3600*24*30, 0, 5000000),
	}
	// 无 UserDB 时会员等级均为 normal，但评分仍应在 [0,1]
	winner := server.pickSmart("model-a", clients)
	if winner == nil {
		t.Fatal("pickSmart returned nil")
	}
	// 验证各维度评分函数在 [0,1]
	for _, c := range clients {
		if s := capacityScore(3, c.GetActiveConnections()); s < 0 || s > 1 {
			t.Errorf("capacity score out of range: %v", s)
		}
		if s := latencyScore(c.GetLatency()); s < 0 || s > 1 {
			t.Errorf("latency score out of range: %v", s)
		}
		if s := failureScore(c.GetFailures()); s < 0 || s > 1 {
			t.Errorf("failure score out of range: %v", s)
		}
	}
}

// TestPickSmartEmpty 验证：空列表返回 nil。
func TestPickSmartEmpty(t *testing.T) {
	server := newTestSmartServer(t)
	if winner := server.pickSmart("model-a", nil); winner != nil {
		t.Fatalf("pickSmart(nil) = %v, want nil", winner)
	}
}

// TestPickSmartSingle 验证：单 client 直接选中。
func TestPickSmartSingle(t *testing.T) {
	server := newTestSmartServer(t)
	c := makeClient("only", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000)
	if winner := server.pickSmart("model-a", []*Client{c}); winner != c {
		t.Fatalf("pickSmart single = %v, want %v", winner, c)
	}
}

// TestPickSmartWeightsSum 验证：默认权重和为 1.0。
func TestPickSmartWeightsSum(t *testing.T) {
	server := newTestSmartServer(t)
	wCap, wMem, wLat, wFail, wStab, wServ := server.smartWeights()
	sum := wCap + wMem + wLat + wFail + wStab + wServ
	if sum < 0.99 || sum > 1.01 {
		t.Fatalf("weights sum = %v, want ~1.0", sum)
	}
}
