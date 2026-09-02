package models

import (
	"testing"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDirectBackendDB(t *testing.T) *DirectBackendDB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return NewDirectBackendDB(db)
}

func testBackend(id string, prio int, maxConns int, models ...string) *DirectBackend {
	ms := make([]*public.Model, 0, len(models))
	for _, name := range models {
		ms = append(ms, &public.Model{Name: name, IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5})
	}
	return &DirectBackend{
		ID:       id,
		Name:     "backend-" + id,
		BaseURL:  "http://" + id + ":8000/v1",
		Format:   "openai",
		Enabled:  true,
		MaxConns: maxConns,
		Priority: prio,
		Models:   ms,
	}
}

// registryBackend 从 server 注册表中取回指定 id 的后端（LoadDirectBackends 会从 DB 重建指针，
// 测试对本地对象做的修改不会反映到注册表，必须取注册表里的真实对象）。
func registryBackend(t *testing.T, server *Server, id string) *DirectBackend {
	t.Helper()
	server.directBackendsMu.RLock()
	defer server.directBackendsMu.RUnlock()
	for _, list := range server.directBackends {
		for _, b := range list {
			if b.ID == id {
				return b
			}
		}
	}
	t.Fatalf("backend %s not in registry", id)
	return nil
}

func TestDirectBackendModelsRoundTrip(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	b := testBackend("b1", 10, 4, "qwen3-32b", "qwen3-8b")
	if err := ddb.Save(b); err != nil {
		t.Fatalf("save: %v", err)
	}
	list, err := ddb.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	got := list[0]
	if len(got.Models) != 2 {
		t.Fatalf("models len = %d, want 2 (AfterFind restore failed)", len(got.Models))
	}
	if got.Models[0].Name != "qwen3-32b" || got.Models[0].IPPM != 2.0 {
		t.Fatalf("model[0] = %+v, want qwen3-32b/2.0", got.Models[0])
	}
}

func TestDirectBackendPriceFor(t *testing.T) {
	b := testBackend("b1", 10, 4, "qwen3-32b")
	ippm, oppm, cippm, ok := b.PriceFor("qwen3-32b")
	if !ok || ippm != 2.0 || oppm != 6.0 || cippm != 0.5 {
		t.Fatalf("PriceFor known = %v/%v/%v/%v, want 2/6/0.5/true", ippm, oppm, cippm, ok)
	}
	if _, _, _, ok := b.PriceFor("nope"); ok {
		t.Fatalf("PriceFor unknown should be !ok")
	}
}

func TestDirectCooldownAlwaysOn(t *testing.T) {
	// 即使 LB_COOLDOWN_ENABLED=false，Direct 冷却仍生效
	old := configs.Config.LBCooldownEnabled
	configs.Config.LBCooldownEnabled = false
	defer func() { configs.Config.LBCooldownEnabled = old }()

	b := testBackend("b1", 10, 4, "qwen3-32b")
	if b.InCooldown() {
		t.Fatalf("fresh backend should not be in cooldown")
	}
	b.IncrFailures()
	if !b.InCooldown() {
		t.Fatalf("IncrFailures should trip cooldown even when LB_COOLDOWN_ENABLED=false")
	}
	b.ResetFailures()
	if b.InCooldown() {
		t.Fatalf("ResetFailures should clear cooldown")
	}
}

func TestDirectBackendLatencyEMA(t *testing.T) {
	b := testBackend("b1", 10, 4, "qwen3-32b")
	b.SetLatencyEMA(100)
	if got := b.GetLatencyEMA(); got != 100 {
		t.Fatalf("first EMA = %v, want 100", got)
	}
	b.SetLatencyEMA(200)
	// alpha 默认 0.3：0.3*200 + 0.7*100 = 130
	if got := b.GetLatencyEMA(); got < 129 || got > 131 {
		t.Fatalf("second EMA = %v, want ~130", got)
	}
}

func TestPickDirectFilters(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	server := &Server{DirectBackendDB: ddb}

	// disabled 后端载入时被排除
	disabled := testBackend("disabled", 1, 4, "qwen3-32b")
	disabled.Enabled = false
	if err := ddb.Save(disabled); err != nil {
		t.Fatalf("save disabled: %v", err)
	}
	// 正常后端
	healthy := testBackend("healthy", 10, 4, "qwen3-32b")
	if err := ddb.Save(healthy); err != nil {
		t.Fatalf("save healthy: %v", err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load: %v", err)
	}
	// 只有 healthy 被载入
	if got := server.PickDirect("qwen3-32b", "", nil); got == nil || got.ID != "healthy" {
		t.Fatalf("PickDirect = %+v, want healthy", got)
	}

	// unhealthy 被过滤
	regHealthy := registryBackend(t, server, "healthy")
	regHealthy.SetHealthy(false)
	if got := server.PickDirect("qwen3-32b", "", nil); got != nil {
		t.Fatalf("PickDirect should filter unhealthy, got %+v", got)
	}
	regHealthy.SetHealthy(true)

	// 冷却被过滤
	regHealthy.IncrFailures()
	if got := server.PickDirect("qwen3-32b", "", nil); got != nil {
		t.Fatalf("PickDirect should filter cooldown, got %+v", got)
	}
	regHealthy.ResetFailures()

	// 饱和（Active==MaxConns）被过滤
	regHealthy.MaxConns = 2
	regHealthy.IncrActive()
	regHealthy.IncrActive()
	if got := server.PickDirect("qwen3-32b", "", nil); got != nil {
		t.Fatalf("PickDirect should filter saturated, got %+v", got)
	}
	regHealthy.DecrActive()
	regHealthy.DecrActive()

	// 无该模型价格被过滤
	if got := server.PickDirect("no-such-model", "", nil); got != nil {
		t.Fatalf("PickDirect should filter no-price model, got %+v", got)
	}
}

func TestPickDirectPriceCap(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	server := &Server{DirectBackendDB: ddb}
	b := testBackend("b1", 10, 4, "qwen3-32b") // IPPM=2, OPPM=6
	if err := ddb.Save(b); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// 无价格帽 → 命中
	if got := server.PickDirect("qwen3-32b", "", nil); got == nil {
		t.Fatalf("PickDirect without cap should hit")
	}

	// 价格帽低于后端价格 → 过滤
	server.UserPriceCapDB = NewUserPriceCapDB(ddb.db)
	server.UserPriceCapDB.setCache("u1:qwen3-32b", 1.0, 100.0, 100.0)
	if got := server.PickDirect("qwen3-32b", "u1", nil); got != nil {
		t.Fatalf("PickDirect should filter over-price-cap, got %+v", got)
	}
	// 价格帽足够 → 命中
	server.UserPriceCapDB.setCache("u1:qwen3-32b", 100.0, 100.0, 100.0)
	if got := server.PickDirect("qwen3-32b", "u1", nil); got == nil {
		t.Fatalf("PickDirect with sufficient cap should hit")
	}
}

func TestPickDirectPriorityThenLoad(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	server := &Server{DirectBackendDB: ddb}

	// 两层 Priority：低值层优先
	lowPrio := testBackend("low", 1, 4, "qwen3-32b")
	highPrio := testBackend("high", 10, 4, "qwen3-32b")
	if err := ddb.Save(lowPrio); err != nil {
		t.Fatalf("save low: %v", err)
	}
	if err := ddb.Save(highPrio); err != nil {
		t.Fatalf("save high: %v", err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := server.PickDirect("qwen3-32b", "", nil); got == nil || got.ID != "low" {
		t.Fatalf("PickDirect = %+v, want low (lower priority)", got)
	}

	// 同层选 Active/MaxConns 低者
	lowPrio2 := testBackend("low2", 1, 4, "qwen3-32b")
	if err := ddb.Save(lowPrio2); err != nil {
		t.Fatalf("save low2: %v", err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load: %v", err)
	}
	// 让 low 更忙（Active=3/4=0.75），low2 空闲（0/4=0）
	regLow := registryBackend(t, server, "low")
	regLow.IncrActive()
	regLow.IncrActive()
	regLow.IncrActive()
	if got := server.PickDirect("qwen3-32b", "", nil); got == nil || got.ID != "low2" {
		t.Fatalf("PickDirect = %+v, want low2 (lower load in same priority)", got)
	}
}

func TestPickDirectExclude(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	server := &Server{DirectBackendDB: ddb}
	b := testBackend("b1", 10, 4, "qwen3-32b")
	if err := ddb.Save(b); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load: %v", err)
	}
	exclude := map[string]bool{"direct:b1": true}
	if got := server.PickDirect("qwen3-32b", "", exclude); got != nil {
		t.Fatalf("PickDirect should respect exclude, got %+v", got)
	}
}

func TestPickDirectNoSupply(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	server := &Server{DirectBackendDB: ddb}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := server.PickDirect("qwen3-32b", "", nil); got != nil {
		t.Fatalf("PickDirect with no supply should return nil, got %+v", got)
	}
}

func TestDirectBackendDBSetEnabledDelete(t *testing.T) {
	ddb := newTestDirectBackendDB(t)
	b := testBackend("b1", 10, 4, "qwen3-32b")
	if err := ddb.Save(b); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := ddb.SetEnabled("b1", false); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	list, _ := ddb.List()
	if len(list) != 1 || list[0].Enabled {
		t.Fatalf("after disable: %+v", list)
	}
	if err := ddb.Delete("b1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = ddb.List()
	if len(list) != 0 {
		t.Fatalf("after delete len = %d, want 0", len(list))
	}
}
