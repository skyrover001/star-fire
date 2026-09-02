package models

import (
	"testing"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
)

// newM2Server 构造一个带 clients map 与 directBackends 的测试 Server。
func newM2Server(t *testing.T) *Server {
	t.Helper()
	s := &Server{}
	s.clients.Store(map[string]map[string]*Client{})
	s.directBackends = map[string][]*DirectBackend{}
	s.LoadBalanceAlgorithm = "smart"
	return s
}

func m2Client(id string, ippm, oppm float64) *Client {
	return &Client{
		ID:     id,
		Status: "online",
		Models: []*public.Model{{Name: "m1", IPPM: ippm, OPPM: oppm}},
	}
}

func TestCostPrice(t *testing.T) {
	if got := costPrice(10, 20); got != 10*0.7+20*0.3 {
		t.Fatalf("costPrice(10,20) = %v", got)
	}
}

func TestPickCheapestPrefersLowestPriceLayer(t *testing.T) {
	s := newM2Server(t)
	// 两个众包 client：c1 便宜，c2 贵
	c1 := m2Client("c1", 1, 1)
	c2 := m2Client("c2", 10, 10)
	c1.ControlConn = &websocket.Conn{}
	c2.ControlConn = &websocket.Conn{}
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"c1": c1, "c2": c2},
	})

	// 关闭扰动，保证 pickSmart 确定
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	cc, cb := s.PickCheapest("m1", "", nil)
	if cb != nil {
		t.Fatalf("expected no direct backend, got %v", cb.ID)
	}
	if cc == nil || cc.ID != "c1" {
		t.Fatalf("expected cheapest client c1, got %v", cc)
	}
}

func TestPickCheapestDirectInLowestLayer(t *testing.T) {
	s := newM2Server(t)
	// 众包贵，Direct 便宜 → 选 Direct
	c1 := m2Client("c1", 10, 10)
	c1.ControlConn = &websocket.Conn{}
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"c1": c1},
	})
	b := &DirectBackend{
		ID:       "d1",
		MaxConns: 4,
		Models:   []*public.Model{{Name: "m1", IPPM: 1, OPPM: 1}},
	}
	b.SetHealthy(true)
	s.directBackends = map[string][]*DirectBackend{"m1": {b}}

	cc, cb := s.PickCheapest("m1", "", nil)
	if cc != nil {
		t.Fatalf("expected no client, got %v", cc.ID)
	}
	if cb == nil || cb.ID != "d1" {
		t.Fatalf("expected direct d1, got %v", cb)
	}
}

func TestPickCheapestExclude(t *testing.T) {
	s := newM2Server(t)
	c1 := m2Client("c1", 1, 1)
	c2 := m2Client("c2", 2, 2)
	c1.ControlConn = &websocket.Conn{}
	c2.ControlConn = &websocket.Conn{}
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"c1": c1, "c2": c2},
	})
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	// 排除 c1 → 选 c2
	cc, _ := s.PickCheapest("m1", "", map[string]bool{"c1": true})
	if cc == nil || cc.ID != "c2" {
		t.Fatalf("expected c2 after excluding c1, got %v", cc)
	}
	// 全排除 → nil,nil
	cc, cb := s.PickCheapest("m1", "", map[string]bool{"c1": true, "c2": true})
	if cc != nil || cb != nil {
		t.Fatalf("expected nil,nil when all excluded, got %v,%v", cc, cb)
	}
}

func TestPickCheapestNoSupply(t *testing.T) {
	s := newM2Server(t)
	cc, cb := s.PickCheapest("nope", "", nil)
	if cc != nil || cb != nil {
		t.Fatalf("expected nil,nil for no supply, got %v,%v", cc, cb)
	}
}

func TestLoadBalanceWithToleranceLatency(t *testing.T) {
	s := newM2Server(t)
	fast := m2Client("fast", 1, 1)
	slow := m2Client("slow", 1, 1)
	fast.ControlConn = &websocket.Conn{}
	slow.ControlConn = &websocket.Conn{}
	fast.Latency = 50
	slow.Latency = 5000
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"fast": fast, "slow": slow},
	})
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	// maxLatencyMs=100 → 只留 fast
	got := s.LoadBalanceWithTolerance("m1", "", nil, 100, 0)
	if got == nil || got.ID != "fast" {
		t.Fatalf("expected fast within latency tolerance, got %v", got)
	}
	// maxLatencyMs=0 → 不限制，两者都合格（pickSmart 选一个）
	if got := s.LoadBalanceWithTolerance("m1", "", nil, 0, 0); got == nil {
		t.Fatalf("expected a client when no latency limit")
	}
}

func TestLoadBalanceWithToleranceStability(t *testing.T) {
	s := newM2Server(t)
	good := m2Client("good", 1, 1)
	bad := m2Client("bad", 1, 1)
	good.ControlConn = &websocket.Conn{}
	bad.ControlConn = &websocket.Conn{}
	// good 缓存稳定性 0.9，bad 0.1
	good.SetCachedScores(0.9, 0.9)
	bad.SetCachedScores(0.1, 0.1)
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"good": good, "bad": bad},
	})
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()

	// minStability=0.5 → 只留 good
	got := s.LoadBalanceWithTolerance("m1", "", nil, 0, 0.5)
	if got == nil || got.ID != "good" {
		t.Fatalf("expected good within stability tolerance, got %v", got)
	}
	// minStability=0 → 不限制
	if got := s.LoadBalanceWithTolerance("m1", "", nil, 0, 0); got == nil {
		t.Fatalf("expected a client when no stability limit")
	}
}

func TestLoadBalanceBalancedMinScore(t *testing.T) {
	s := newM2Server(t)
	// 两个 client，性能差异明显：high 空闲+低延迟+高分，low 饱和+高延迟+低分
	high := m2Client("high", 1, 1)
	low := m2Client("low", 1, 1)
	high.ControlConn = &websocket.Conn{}
	low.ControlConn = &websocket.Conn{}
	high.Latency = 10
	low.Latency = 5000
	high.ActiveConnections = 0
	low.ActiveConnections = 1 // normal 默认 max=1 → 饱和
	high.BandwidthMbps = 100
	low.BandwidthMbps = 1
	// 离线评分模式：用缓存分，避免依赖 DB
	high.SetCachedScores(1.0, 1.0)
	low.SetCachedScores(0.0, 0.0)
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"high": high, "low": low},
	})
	origJitter := configs.Config.LBJitter
	configs.Config.LBJitter = 0
	defer func() { configs.Config.LBJitter = origJitter }()
	origOffline := configs.Config.LBScoreOffline
	configs.Config.LBScoreOffline = true
	defer func() { configs.Config.LBScoreOffline = origOffline }()

	// 高阈值 → 只留 high
	origMin := configs.Config.LBBalancedMinScore
	configs.Config.LBBalancedMinScore = 0.9
	defer func() { configs.Config.LBBalancedMinScore = origMin }()

	got := s.LoadBalanceBalanced("m1", "", nil, 0, 0)
	if got == nil || got.ID != "high" {
		t.Fatalf("expected high to pass balanced min score, got %v", got)
	}
}

func TestGetModelsTiers(t *testing.T) {
	s := newM2Server(t)
	c := m2Client("c1", 1, 1)
	c.ControlConn = &websocket.Conn{}
	s.clients.Store(map[string]map[string]*Client{
		"m1": {"c1": c},
	})
	// 无 direct → tiers 只有 community
	res := s.GetModels()
	data, ok := res["data"].([]map[string]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("expected 1 model, got %#v", res["data"])
	}
	tiers, ok := data[0]["tiers"].([]string)
	if !ok || len(tiers) != 1 || tiers[0] != "community" {
		t.Fatalf("expected [community] tiers, got %#v", data[0]["tiers"])
	}

	// 加 direct → tiers 含 direct
	b := &DirectBackend{ID: "d1", Models: []*public.Model{{Name: "m1"}}}
	s.directBackends = map[string][]*DirectBackend{"m1": {b}}
	res = s.GetModels()
	data = res["data"].([]map[string]interface{})
	tiers = data[0]["tiers"].([]string)
	if len(tiers) != 2 || tiers[0] != "direct" || tiers[1] != "community" {
		t.Fatalf("expected [direct community] tiers, got %#v", tiers)
	}

	// Direct-only model must be returned even without a connected community client.
	s.directBackends = map[string][]*DirectBackend{
		"direct-only": {{ID: "d2", Models: []*public.Model{{Name: "direct-only"}}}},
	}
	res = s.GetModels()
	data = res["data"].([]map[string]interface{})
	if len(data) != 2 {
		t.Fatalf("expected community and Direct-only models, got %#v", data)
	}
	foundDirectOnly := false
	for _, item := range data {
		if item["id"] == "direct-only" {
			foundDirectOnly = true
			tiers = item["tiers"].([]string)
			if len(tiers) != 1 || tiers[0] != "direct" {
				t.Fatalf("expected Direct-only tier, got %#v", tiers)
			}
		}
	}
	if !foundDirectOnly {
		t.Fatalf("Direct-only model missing from %#v", data)
	}
}
