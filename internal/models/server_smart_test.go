package models

import (
	"testing"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============ 各维度评分函数单元测试 ============

func TestCapacityScore(t *testing.T) {
	cases := []struct {
		name    string
		maxConn int
		active  int32
		want    float64
	}{
		{"svip finite full", 50, 50, 0.0}, // SVIP 有限，饱和=0
		{"svip half", 50, 25, 0.5},
		{"full", 1, 1, 0.0},
		{"half", 5, 2, 0.6},
		{"over limit clamped", 3, 5, 0.0},
		{"zero max", 0, 0, 0.0},
		{"negative max", -1, 0, 0.0}, // 不再有无限
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

// TestEffectiveMaxConnections 验证：client 自定义连接数上限（Python 滑块）覆盖会员默认上限。
func TestEffectiveMaxConnections(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	userDB := NewUserDB(db)
	users := []*User{
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

	// VIP 默认上限 5
	vip := &Client{ID: "vip", UserID: "user-vip"}
	if got := server.effectiveMaxConnections(vip); got != 5 {
		t.Fatalf("vip default max = %d, want 5", got)
	}
	// VIP 自定义上限 3（<= 5 生效）
	vip.MaxConnectionsOverride = 3
	if got := server.effectiveMaxConnections(vip); got != 3 {
		t.Fatalf("vip override max = %d, want 3", got)
	}
	// VIP 自定义上限超过会员上限（6 > 5）→ 回退到会员默认
	vip.MaxConnectionsOverride = 6
	if got := server.effectiveMaxConnections(vip); got != 5 {
		t.Fatalf("vip over-limit override = %d, want 5", got)
	}
	// SVIP 默认上限 50
	svip := &Client{ID: "svip", UserID: "user-svip"}
	if got := server.effectiveMaxConnections(svip); got != 50 {
		t.Fatalf("svip default max = %d, want 50", got)
	}
	// SVIP 自定义上限 20（<= 50 生效）
	svip.MaxConnectionsOverride = 20
	if got := server.effectiveMaxConnections(svip); got != 20 {
		t.Fatalf("svip override max = %d, want 20", got)
	}
	// 无 UserDB 时按 normal（默认 1）
	server.UserDB = nil
	anon := &Client{ID: "anon"}
	if got := server.effectiveMaxConnections(anon); got != 1 {
		t.Fatalf("anon default max = %d, want 1", got)
	}
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

// TestPickSmartMembershipPriority 验证：性能相同（都进入候选）时，高等级会员权重更高，
// 因此 SVIP 被选中的概率更高（但不是 100% 垄断）。
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
		makeClient("normal", MembershipNormal, 1, 0, 100, 0, 3600*24, 0, 500000),
		makeClient("vip", MembershipVIP, 5, 0, 100, 0, 3600*24, 0, 500000),
		makeClient("svip", MembershipSVIP, 50, 0, 100, 0, 3600*24, 0, 500000),
	}
	clients[0].UserID = "user-normal"
	clients[1].UserID = "user-vip"
	clients[2].UserID = "user-svip"

	// 关闭扰动，保证性能评分确定（三者性能相同，都进入候选）
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	// 运行多次，统计各等级被选中次数。三者性能相同，阶段2按会员权重选择。
	// 默认权重 normal=1.0, vip=3.0, svip=8.0，因此 svip 概率最高，但非 100%。
	counts := map[string]int{}
	for i := 0; i < 12000; i++ {
		winner := server.pickSmart("model-a", clients)
		if winner == nil {
			t.Fatal("pickSmart returned nil")
		}
		counts[winner.ID]++
	}
	// svip 应占比最高（约 8.0/(1.0+3.0+8.0)=66.7%），且三者都应有被选中
	if counts["svip"] <= counts["normal"] || counts["svip"] <= counts["vip"] {
		t.Fatalf("svip should have highest pick count, got %v", counts)
	}
	if counts["vip"] <= counts["normal"] {
		t.Fatalf("vip should be picked more than normal (paid value), got %v", counts)
	}
	if counts["normal"] == 0 || counts["vip"] == 0 {
		t.Fatalf("normal/vip should be picked sometimes (not starved), got %v", counts)
	}
	t.Logf("pick distribution: %v", counts)
}

// TestPickSmartDeterministicWithoutJitter 验证：只有一个候选时，无论扰动与否都返回该 client。
// 注意：新算法阶段2是"按会员权重加权随机选择"，因此多候选时天然非确定性（这是设计意图，
// 用于在性能相近的候选中分散流量）。确定性只体现在单候选场景。
func TestPickSmartDeterministicWithoutJitter(t *testing.T) {
	server := newTestSmartServer(t)
	// 保存并临时关闭扰动
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	// 单候选：应始终返回该 client
	clients := []*Client{
		makeClient("a", MembershipNormal, 3, 0, 100, 0, 3600*24, 0, 500000),
	}
	for i := 0; i < 50; i++ {
		if w := server.pickSmart("model-a", clients); w.ID != "a" {
			t.Fatalf("single candidate should always be picked, got %s", w.ID)
		}
	}
}

// TestPickSmartScoreRange 验证：所有 client 的评分都在 [0,1] 范围内。
func TestPickSmartScoreRange(t *testing.T) {
	server := newTestSmartServer(t)
	clients := []*Client{
		makeClient("a", MembershipNormal, 1, 1, 5000, 5, 3600, 10, 10000),
		makeClient("b", MembershipVIP, 5, 5, 1000, 2, 3600*24, 2, 500000),
		makeClient("c", MembershipSVIP, 50, 0, 100, 0, 3600*24*30, 0, 5000000),
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

// TestPickSmartWeightsSum 验证：默认性能权重和为 1.0。
func TestPickSmartWeightsSum(t *testing.T) {
	server := newTestSmartServer(t)
	wCap, wLat, wFail, wStab, wServ, wBw := server.smartWeights()
	sum := wCap + wLat + wFail + wStab + wServ + wBw
	if sum < 0.99 || sum > 1.01 {
		t.Fatalf("weights sum = %v, want ~1.0", sum)
	}
}

// TestBandwidthScore 验证带宽评分。
func TestBandwidthScore(t *testing.T) {
	origTarget := configs.Config.BandwidthTargetMbps
	configs.Config.BandwidthTargetMbps = 50
	defer func() { configs.Config.BandwidthTargetMbps = origTarget }()

	cases := []struct {
		bw   float64
		want float64
	}{
		{0, 0.0},
		{25, 0.5},
		{50, 1.0},
		{100, 1.0}, // 超过目标 clamp 到 1
	}
	for _, c := range cases {
		if got := bandwidthScore(c.bw); got != c.want {
			t.Errorf("bandwidthScore(%v) = %v, want %v", c.bw, got, c.want)
		}
	}
}
