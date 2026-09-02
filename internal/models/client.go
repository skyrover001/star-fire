package models

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	configs "star-fire/config"
	"star-fire/pkg/public"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"gorm.io/gorm"
)

// InferenceEngine represents the type of inference engine used by the client.
type InferenceEngine struct {
	Name        string `json:"name" gorm:"default:ollama"` // e.g. "ollama", "vllm", "openai"
	MaxTokens   int    `json:"max_tokens"`
	NumParallel int    `json:"num_parallel"`
}

type Client struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	IP           string    `json:"ip"`
	Token        string    `json:"token"`
	ModelsJSON   string    `json:"-" gorm:"column:models;type:text"` // 模型列表 JSON，可能超长，MySQL 下必须用 text（默认 varchar(191) 会截断报错）
	Status       string    `json:"status"`
	RegisterTime time.Time `json:"register_time"`
	Latency      int       `json:"latency"`
	UserID       string    `json:"user_id" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// not in db
	Models           []*public.Model          `json:"models" gorm:"-"`
	EmbeddingModels  []*openai.EmbeddingModel `json:"embedding_models" gorm:"-"`
	ControlConn      *websocket.Conn          `json:"-" gorm:"-"`
	ControlConnMutex sync.Mutex               `json:"-" gorm:"-"`
	LatencyMutex     sync.RWMutex             `json:"-" gorm:"-"`
	LastPingTime     int64                    `json:"-" gorm:"-"`
	MessageChan      chan *api.ChatResponse   `json:"-" gorm:"-"`
	PongChan         chan *public.PPMessage   `json:"-" gorm:"-"`
	ErrChan          chan error               `json:"-" gorm:"-"`
	User             *User                    `json:"user" gorm:"-"`
	InferenceEngine  InferenceEngine          `json:"inference_engine" gorm:"-"`

	// 内存原子计数：当前同时处理的请求数（用于会员连接数限制）
	ActiveConnections int32 `json:"-" gorm:"-"`

	// 内存原子计数：最近失败次数（用于 smart 负载均衡失败率维度）
	RecentFailures int32 `json:"-" gorm:"-"`

	// 客户端上行带宽（Mbps），由 client 上报或使用默认值。
	// 用于 smart 负载均衡带宽维度（考虑 client 到 server 的上行带宽）。
	BandwidthMbps float64 `json:"bandwidth_mbps" gorm:"-"`

	// 客户端自定义连接数上限（0 = 使用会员等级默认上限）。
	// 由 Python 客户端 app 通过滑块配置（0 ~ 会员上限），经 Go 客户端上报到 server。
	// 用于覆盖 connectionLimitEligible 和 perfScore 中的连接池大小。
	MaxConnectionsOverride int `json:"max_connections" gorm:"-"`

	// 延迟 EMA 平滑值（非 DB 字段，仅内存）
	LatencyEMA float64 `json:"-" gorm:"-"`

	// P0: perfScore 慢变维度离线缓存（math.Float64bits 存储，原子读写；ScoreUpdatedAt==0 表示未初始化）
	CachedStabilityScore uint64 `json:"-" gorm:"-"`
	CachedServiceScore   uint64 `json:"-" gorm:"-"`
	ScoreUpdatedAt       int64  `json:"-" gorm:"-"` // unix 秒

	// P0: 熔断冷却截止时间（unix 纳秒，0=无冷却）
	CooldownUntil int64 `json:"-" gorm:"-"`

	// P2: 实测 cache 命中率 EMA（float64bits 原子；仅用于 tiebreak 与 cost 有效单价，不进 perfScore）
	CacheHitEMA uint64 `json:"-" gorm:"-"`
}

// UpdateCacheHit 以 EMA(alpha=0.2) 更新实测命中率，h ∈ [0,1]。
func (c *Client) UpdateCacheHit(h float64) {
	if h < 0 {
		h = 0
	}
	if h > 1 {
		h = 1
	}
	const alpha = 0.2
	for {
		old := atomic.LoadUint64(&c.CacheHitEMA)
		if old == 0 {
			if atomic.CompareAndSwapUint64(&c.CacheHitEMA, 0, math.Float64bits(h)) {
				return
			}
			continue
		}
		prev := math.Float64frombits(old)
		next := alpha*h + (1-alpha)*prev
		if atomic.CompareAndSwapUint64(&c.CacheHitEMA, old, math.Float64bits(next)) {
			return
		}
	}
}

// GetCacheHitEMA 返回当前 EMA（未初始化返回 0）。
func (c *Client) GetCacheHitEMA() float64 {
	return math.Float64frombits(atomic.LoadUint64(&c.CacheHitEMA))
}

// IncrActiveConnections 原子增加当前处理请求数
func (c *Client) IncrActiveConnections() {
	atomic.AddInt32(&c.ActiveConnections, 1)
}

// DecrActiveConnections 原子减少当前处理请求数
func (c *Client) DecrActiveConnections() {
	atomic.AddInt32(&c.ActiveConnections, -1)
}

// GetActiveConnections 原子读取当前处理请求数
func (c *Client) GetActiveConnections() int32 {
	return atomic.LoadInt32(&c.ActiveConnections)
}

// SetLatency 线程安全地更新客户端延迟（毫秒），并维护 EMA 平滑值。
// 使用指数移动平均（EMA）抑制瞬时抖动：ema = α*new + (1-α)*prev。
// α 由 configs.Config.LBEMAAplha 控制，默认 0.3。
func (c *Client) SetLatency(latency int) {
	c.LatencyMutex.Lock()
	defer c.LatencyMutex.Unlock()
	alpha := configs.Config.LBEMAAplha
	if alpha <= 0 || alpha > 1 {
		alpha = 0.3
	}
	if c.LatencyEMA == 0 {
		c.LatencyEMA = float64(latency)
	} else {
		c.LatencyEMA = alpha*float64(latency) + (1-alpha)*c.LatencyEMA
	}
	c.Latency = int(c.LatencyEMA)
}

// GetLatency 线程安全地读取客户端延迟（毫秒，EMA 平滑后）。
func (c *Client) GetLatency() int {
	c.LatencyMutex.RLock()
	defer c.LatencyMutex.RUnlock()
	return c.Latency
}

// IncrFailures 原子增加最近失败次数（smart 负载均衡失败率维度）。
// 若熔断冷却开启，同时按指数退避触发冷却（4xx 路径不调用本方法，天然不触发冷却）。
func (c *Client) IncrFailures() {
	atomic.AddInt32(&c.RecentFailures, 1)
	if configs.Config.LBCooldownEnabled {
		c.TripCooldown()
	}
}

// ResetFailures 原子清零最近失败次数（请求成功时调用），并清除冷却（成功即完全恢复）。
func (c *Client) ResetFailures() {
	atomic.StoreInt32(&c.RecentFailures, 0)
	c.ClearCooldown()
}

// GetFailures 原子读取最近失败次数。
func (c *Client) GetFailures() int32 {
	return atomic.LoadInt32(&c.RecentFailures)
}

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
	if base <= 0 {
		base = 5 * time.Second
	}
	if max <= 0 {
		max = 5 * time.Minute
	}
	n := c.GetFailures()
	if n < 1 {
		n = 1
	}
	d := base << uint(n-1) // 溢出防护：n-1 > 30 时直接取 max
	if n > 30 || d > max || d <= 0 {
		d = max
	}
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

type ConnectionResult struct {
	ClientID string `json:"client_id"`
	Count    int    `json:"count"`
}

func (c *Client) BeforeSave(tx *gorm.DB) error {
	if c.Models != nil {
		data, err := json.Marshal(c.Models)
		if err != nil {
			return err
		}
		c.ModelsJSON = string(data)
	}
	return nil
}

func (c *Client) AfterFind(tx *gorm.DB) error {
	if c.ModelsJSON != "" {
		var models []*public.Model
		if err := json.Unmarshal([]byte(c.ModelsJSON), &models); err != nil {
			return err
		}
		c.Models = models
	}
	return nil
}

type ClientDB struct {
	db *gorm.DB
}

func NewClientDB(db *gorm.DB) *ClientDB {
	if err := db.AutoMigrate(&Client{}); err != nil {
		log.Fatalf("迁移Client表失败: %v", err)
	}
	return &ClientDB{db: db}
}

func (cdb *ClientDB) GetClient(id string) (*Client, error) {
	var client Client
	result := cdb.db.Where("id = ?", id).First(&client)
	return &client, result.Error
}

func (cdb *ClientDB) SaveClient(client *Client) error {
	return cdb.db.Save(client).Error
}

func (cdb *ClientDB) UpdateStatus(id, status string) error {
	return cdb.db.Model(&Client{}).Where("id = ?", id).Update("status", status).Error
}

func (cdb *ClientDB) GetClientsByUser(userID string) ([]*Client, error) {
	var clients []*Client
	result := cdb.db.Where("user_id = ?", userID).Find(&clients)
	return clients, result.Error
}

func (cdb *ClientDB) GetActiveClients() ([]*Client, error) {
	var clients []*Client
	result := cdb.db.Where("status = ?", "connected").Find(&clients)
	return clients, result.Error
}

func (cdb *ClientDB) GetClientsByUserID(userID string) ([]*Client, error) {
	var clients []*Client
	result := cdb.db.Where("user_id = ?", userID).Find(&clients)
	if result.Error != nil {
		return nil, result.Error
	}
	for _, client := range clients {
		if err := client.AfterFind(cdb.db); err != nil {
			log.Printf("after find client error: %v", err)
		}
	}
	return clients, nil
}

func NewClient(id, ip string, conn *websocket.Conn) *Client {
	return &Client{
		ID:           id,
		IP:           ip,
		ControlConn:  conn,
		Status:       "connecting",
		RegisterTime: time.Now(),
		Latency:      public.MAXLATENCE,
		PongChan:     make(chan *public.PPMessage),
		MessageChan:  make(chan *api.ChatResponse),
		ErrChan:      make(chan error),
	}
}

func (c *Client) SetUser(user *User) {
	c.User = user
	c.UserID = user.ID
}

type ClientFingerprint struct {
	Fingerprint string `json:"fingerprint" gorm:"primaryKey"`
	ClientID    string `json:"client_id" gorm:"index"`
	Status      string `json:"status"` // e.g. "preparing", "transmitting", "completed"
}

type ClientFingerprintDB struct {
	db *gorm.DB
}

func NewClientFingerprintDB(db *gorm.DB) *ClientFingerprintDB {
	if err := db.AutoMigrate(&ClientFingerprint{}); err != nil {
		log.Fatalf("migrate client fingerprint table: %v", err)
	}
	return &ClientFingerprintDB{db: db}
}

func (cfdb *ClientFingerprintDB) SaveFingerprint(fingerprint, clientID, status string) error {
	cf := &ClientFingerprint{
		Fingerprint: fingerprint,
		ClientID:    clientID,
		Status:      status,
	}
	return cfdb.db.Save(cf).Error
}

func (cfdb *ClientFingerprintDB) UpdateFingerprint(fingerprint, clientID, status string) error {
	cf := &ClientFingerprint{
		Fingerprint: fingerprint,
		ClientID:    clientID,
		Status:      status,
	}
	result := cfdb.db.Where("fingerprint = ?", fingerprint).FirstOrCreate(cf)
	if result.Error != nil {
		return result.Error
	}
	return cfdb.db.Model(cf).Update("status", status).Error
}
func (cfdb *ClientFingerprintDB) GetClientChatConnections(clientIDs []string) ([]*ConnectionResult, error) {
	// 如果没有可用的客户端，返回错误
	if len(clientIDs) == 0 {
		return nil, fmt.Errorf("没有可用的客户端")
	}

	var results []*ConnectionResult
	// 查询每个客户端状态为"transmitting"的连接数
	err := cfdb.db.Model(&ClientFingerprint{}).
		Select("client_id, count(*) as count").
		Where("client_id IN ? AND status = ?", clientIDs, "transmitting").
		Group("client_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}
func (cfdb *ClientFingerprintDB) GetClientID(fingerprint string) (string, error) {
	var cf ClientFingerprint
	result := cfdb.db.Where("fingerprint = ?", fingerprint).First(&cf)
	if result.Error != nil {
		return "", result.Error
	}
	return cf.ClientID, nil
}

func (cfdb *ClientFingerprintDB) DeleteFingerprint(fingerprint string) error {
	return cfdb.db.Where("fingerprint = ?", fingerprint).Delete(&ClientFingerprint{}).Error
}
