package models

import (
	"encoding/json"
	"log"
	"math"
	"sync/atomic"
	"time"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"gorm.io/gorm"
)

// DirectBackend 固定后端（平台直连的模型供给：自建 vLLM 集群 / 合作方 OpenAI 兼容 API）。
// 与 Client（众包）不同，Direct 走 HTTP 直连，无 WS 隧道，可水平扩展作为主力供给。
type DirectBackend struct {
	ID         string    `json:"id" gorm:"primaryKey;size:64"`
	Name       string    `json:"name" gorm:"size:128"`
	BaseURL    string    `json:"base_url" gorm:"size:256"` // e.g. http://vllm-1:8000/v1（不含尾斜杠）
	APIKey     string    `json:"-" gorm:"size:256"`        // 出站鉴权，日志必须脱敏
	Format     string    `json:"format" gorm:"size:32"`    // 首期仅 "openai"
	Enabled    bool      `json:"enabled"`
	MaxConns   int       `json:"max_conns"`                        // 硬并发上限，<=0 视为 1
	Priority   int       `json:"priority"`                         // 越小越优先
	ModelsJSON string    `json:"-" gorm:"column:models;type:text"` // []public.Model
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 运行态（内存，非 DB）
	Models         []*public.Model `json:"models" gorm:"-"`
	ActiveConns    int32           `json:"-" gorm:"-"` // atomic
	RecentFailures int32           `json:"-" gorm:"-"` // atomic
	CooldownUntil  int64           `json:"-" gorm:"-"` // atomic, unix ns
	LatencyEMA     uint64          `json:"-" gorm:"-"` // atomic float64bits（健康检查 RTT ms）
	Healthy        int32           `json:"-" gorm:"-"` // atomic bool（健康检查结果）
	CacheHitEMA    uint64          `json:"-" gorm:"-"` // atomic float64bits（实测 cache 命中率 EMA，仅 tiebreak）
}

// UpdateCacheHit 以 EMA(alpha=0.2) 更新实测命中率，h ∈ [0,1]。
func (b *DirectBackend) UpdateCacheHit(h float64) {
	if h < 0 {
		h = 0
	}
	if h > 1 {
		h = 1
	}
	const alpha = 0.2
	for {
		old := atomic.LoadUint64(&b.CacheHitEMA)
		if old == 0 {
			if atomic.CompareAndSwapUint64(&b.CacheHitEMA, 0, math.Float64bits(h)) {
				return
			}
			continue
		}
		prev := math.Float64frombits(old)
		next := alpha*h + (1-alpha)*prev
		if atomic.CompareAndSwapUint64(&b.CacheHitEMA, old, math.Float64bits(next)) {
			return
		}
	}
}

// GetCacheHitEMA 返回当前 EMA（未初始化返回 0）。
func (b *DirectBackend) GetCacheHitEMA() float64 {
	return math.Float64frombits(atomic.LoadUint64(&b.CacheHitEMA))
}

// BeforeSave 序列化 Models → ModelsJSON（照抄 Client 实现）。
func (b *DirectBackend) BeforeSave(tx *gorm.DB) error {
	if b.Models != nil {
		data, err := json.Marshal(b.Models)
		if err != nil {
			return err
		}
		b.ModelsJSON = string(data)
	}
	return nil
}

// AfterFind 反序列化 ModelsJSON → Models（照抄 Client 实现）。
func (b *DirectBackend) AfterFind(tx *gorm.DB) error {
	if b.ModelsJSON != "" {
		var models []*public.Model
		if err := json.Unmarshal([]byte(b.ModelsJSON), &models); err != nil {
			return err
		}
		b.Models = models
	}
	return nil
}

// ---- 原子方法（与 Client 同名同语义，独立实现） ----

func (b *DirectBackend) IncrActive() {
	atomic.AddInt32(&b.ActiveConns, 1)
}

func (b *DirectBackend) DecrActive() {
	atomic.AddInt32(&b.ActiveConns, -1)
}

func (b *DirectBackend) GetActive() int32 {
	return atomic.LoadInt32(&b.ActiveConns)
}

func (b *DirectBackend) IncrFailures() {
	atomic.AddInt32(&b.RecentFailures, 1)
	// Direct 冷却始终开启（不受 LB_COOLDOWN_ENABLED 开关限制）：
	// 无 P0 前置时也需要故障隔离。
	b.TripCooldown()
}

func (b *DirectBackend) ResetFailures() {
	atomic.StoreInt32(&b.RecentFailures, 0)
	b.ClearCooldown()
}

func (b *DirectBackend) GetFailures() int32 {
	return atomic.LoadInt32(&b.RecentFailures)
}

// TripCooldown 按 RecentFailures 指数退避设置冷却：base × 2^(failures-1)，上限 max。
// 冷却参数复用 P0 的 LBCooldownBaseMs/MaxMs。
func (b *DirectBackend) TripCooldown() {
	base := time.Duration(configs.Config.LBCooldownBaseMs) * time.Millisecond
	max := time.Duration(configs.Config.LBCooldownMaxMs) * time.Millisecond
	if base <= 0 {
		base = 5 * time.Second
	}
	if max <= 0 {
		max = 5 * time.Minute
	}
	n := b.GetFailures()
	if n < 1 {
		n = 1
	}
	d := base << uint(n-1)
	if n > 30 || d > max || d <= 0 {
		d = max
	}
	atomic.StoreInt64(&b.CooldownUntil, time.Now().Add(d).UnixNano())
}

func (b *DirectBackend) InCooldown() bool {
	u := atomic.LoadInt64(&b.CooldownUntil)
	return u != 0 && time.Now().UnixNano() < u
}

func (b *DirectBackend) ClearCooldown() {
	atomic.StoreInt64(&b.CooldownUntil, 0)
}

// SetLatencyEMA 更新健康检查 RTT 的 EMA 平滑值（alpha 复用 LBEMAAplha）。
func (b *DirectBackend) SetLatencyEMA(ms float64) {
	alpha := configs.Config.LBEMAAplha
	if alpha <= 0 || alpha > 1 {
		alpha = 0.3
	}
	prev := math.Float64frombits(atomic.LoadUint64(&b.LatencyEMA))
	var next float64
	if prev == 0 {
		next = ms
	} else {
		next = alpha*ms + (1-alpha)*prev
	}
	atomic.StoreUint64(&b.LatencyEMA, math.Float64bits(next))
}

func (b *DirectBackend) GetLatencyEMA() float64 {
	return math.Float64frombits(atomic.LoadUint64(&b.LatencyEMA))
}

func (b *DirectBackend) SetHealthy(ok bool) {
	v := int32(0)
	if ok {
		v = 1
	}
	atomic.StoreInt32(&b.Healthy, v)
}

func (b *DirectBackend) IsHealthy() bool {
	return atomic.LoadInt32(&b.Healthy) == 1
}

// PriceFor 遍历 Models 查找指定模型的价格。
func (b *DirectBackend) PriceFor(model string) (ippm, oppm, cippm float64, ok bool) {
	for _, m := range b.Models {
		if m.Name == model {
			return m.IPPM, m.OPPM, m.CIPPM, true
		}
	}
	return 0, 0, 0, false
}

// DirectBackendDB 持久化访问层。
type DirectBackendDB struct {
	db *gorm.DB
}

func NewDirectBackendDB(db *gorm.DB) *DirectBackendDB {
	if err := db.AutoMigrate(&DirectBackend{}); err != nil {
		log.Fatalf("迁移DirectBackend表失败: %v", err)
	}
	return &DirectBackendDB{db: db}
}

// List 返回全部后端（含 disabled，CLI 用）。
func (d *DirectBackendDB) List() ([]*DirectBackend, error) {
	var backends []*DirectBackend
	result := d.db.Order("priority ASC, created_at ASC").Find(&backends)
	return backends, result.Error
}

func (d *DirectBackendDB) Save(b *DirectBackend) error {
	return d.db.Save(b).Error
}

func (d *DirectBackendDB) SetEnabled(id string, enabled bool) error {
	return d.db.Model(&DirectBackend{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (d *DirectBackendDB) Delete(id string) error {
	return d.db.Delete(&DirectBackend{}, "id = ?", id).Error
}
