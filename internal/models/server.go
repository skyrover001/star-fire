package models

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	configs "star-fire/config"
	"star-fire/pkg/public"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

type MailService struct {
	SMTPServer   string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromAddress  string
}

type Server struct {
	clientsMu sync.Mutex
	clients   atomic.Value // stores map[string]map[string]*Client

	clientRBMu            sync.RWMutex
	clientRoundRobinIndex map[string]int // for round-robin load balancing

	respClientsMu sync.RWMutex
	RespClients   map[string]*websocket.Conn

	respClientReadyChans   map[string]chan struct{}
	respClientReadyChansMu sync.Mutex

	Port               string
	RegisterTokenStore *RegisterTokenStore

	DB                  *gorm.DB
	APIKeyDB            *APIKeyDB
	UserDB              *UserDB
	TokenUsageDB        *TokenUsageDB
	ClientDB            *ClientDB
	ClientFingerprintDB *ClientFingerprintDB
	ClientStatsDB       *ClientStatsDB
	TrendDB             *TrendDB
	RechargeDB          *RechargeDB
	UserPriceCapDB      *UserPriceCapDB
	SystemConfigDB      *SystemConfigDB
	NotificationDB      *NotificationDB

	// 消费者端限流器（RPM/TPM）
	RateLimiter *RateLimiter

	// reasoning 状态存储：思考模型工具循环续接需要回传 reasoning_content，而 Codex
	// 只回传加密内容，故网关在此保存上一轮模型输出的 reasoning_content。
	// key: 会话标识（Responses 的 prompt_cache_key）→ tool_call_id → reasoning_text。
	reasoningMu sync.RWMutex
	reasoning   map[string]map[string]string

	LoadBalanceAlgorithm string // Load balancing algorithm, e.g., "round-robin", "random", etc.

	MailService *MailService // optional, for sending emails

	Conf *configs.Configuration
}

func NewServer() *Server {
	gormDB, err := OpenDatabase()
	if err != nil {
		log.Fatalf("init database failed: %v", err)
	}
	log.Printf("database: %s (max_open=%d, max_idle=%d)",
		map[bool]string{true: "mysql", false: "sqlite"}[IsMySQL()],
		configs.Config.DBMaxOpenConns, configs.Config.DBMaxIdleConns)

	apiKeyDB := NewAPIKeyDB(gormDB)
	tokenUsageDB := NewTokenUsageDB(gormDB)
	userDB := NewUserDB(gormDB)
	clientDB := NewClientDB(gormDB)
	clientFingerprintDB := NewClientFingerprintDB(gormDB)
	clientStatsDB := NewClientStatsDB(gormDB)
	trendDB := NewTrendDB(gormDB)
	userPriceCapDB := NewUserPriceCapDB(gormDB)
	rechargeDB := NewRechargeDB(gormDB)
	systemConfigDB := NewSystemConfigDB(gormDB)
	notificationDB := NewNotificationDB(gormDB)

	// 初始化默认用户
	err = userDB.InitDefaultUsers()
	if err != nil {
		log.Printf("init default user failed: %v", err)
	}

	clientsVal := new(atomic.Value)
	clientsVal.Store(make(map[string]map[string]*Client))

	server := &Server{
		clients:               *clientsVal,
		Port:                  configs.Config.ServerPort,
		RespClients:           make(map[string]*websocket.Conn),
		clientRoundRobinIndex: make(map[string]int),
		respClientReadyChans:  make(map[string]chan struct{}),
		reasoning:             make(map[string]map[string]string),

		DB:                   gormDB,
		APIKeyDB:             apiKeyDB,
		UserDB:               userDB,
		TokenUsageDB:         tokenUsageDB,
		RegisterTokenStore:   NewRegisterTokenStore(),
		ClientDB:             clientDB,
		ClientFingerprintDB:  clientFingerprintDB,
		ClientStatsDB:        clientStatsDB,
		TrendDB:              trendDB,
		UserPriceCapDB:       userPriceCapDB,
		RechargeDB:           rechargeDB,
		SystemConfigDB:       systemConfigDB,
		NotificationDB:       notificationDB,
		RateLimiter:          NewRateLimiter(),
		LoadBalanceAlgorithm: configs.Config.LBA, // default load balancing algorithm
		MailService: &MailService{
			SMTPServer:   configs.Config.EmailHost,
			SMTPPort:     configs.Config.EmailPort,
			SMTPUsername: configs.Config.EmailUser,
			SMTPPassword: configs.Config.EmailPassword,
			FromAddress:  configs.Config.EmailFrom,
		},
		Conf: &configs.Config,
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			server.RegisterTokenStore.CleanupExpiredTokens()
			server.checkMembershipExpiry()
		}
	}()
	return server
}

// checkMembershipExpiry 检查会员到期，提前 7 天提醒用户。
// 分别检查消费者会员与贡献者会员。
func (s *Server) checkMembershipExpiry() {
	if s.NotificationDB == nil || s.UserDB == nil {
		return
	}
	users, _, err := s.UserDB.ListUsers(1, 1000)
	if err != nil {
		return
	}
	now := time.Now()
	for _, u := range users {
		// 消费者会员
		if u.Membership != "" && u.Membership != MembershipNormal && u.MembershipExpireAt != nil {
			daysLeft := int(u.MembershipExpireAt.Sub(now).Hours() / 24)
			if daysLeft >= 0 && daysLeft <= 7 {
				_ = s.NotificationDB.Create(
					u.ID,
					"membership_expire",
					"会员即将到期",
					fmt.Sprintf("您的消费者 %s 会员将于 %s 到期，剩余 %d 天，请及时续费。",
						u.Membership, u.MembershipExpireAt.Format("2006-01-02"), daysLeft),
				)
			}
		}
		// 贡献者会员
		if u.ContributorMembership != "" && u.ContributorMembership != MembershipNormal && u.ContributorMembershipExpireAt != nil {
			daysLeft := int(u.ContributorMembershipExpireAt.Sub(now).Hours() / 24)
			if daysLeft >= 0 && daysLeft <= 7 {
				_ = s.NotificationDB.Create(
					u.ID,
					"membership_expire",
					"会员即将到期",
					fmt.Sprintf("您的贡献者 %s 会员将于 %s 到期，剩余 %d 天，请及时续费。",
						u.ContributorMembership, u.ContributorMembershipExpireAt.Format("2006-01-02"), daysLeft),
				)
			}
		}
	}
}

// Predicate is a filter function used in the routing Predicate phase.
// It returns true if the client should be included in the eligible set for the given model.
type Predicate func(c *Client, model string) bool

// clientHealthy returns true when the client is online, connected, and has acceptable latency.
// Used both as a Predicate and to identify dead clients for cleanup.
// 延迟阈值由环境变量 MAX_LATENCY（秒）控制，默认 30s；超过该阈值的 client 会被剔除，不参与算力调度。
func clientHealthy(c *Client, model string) bool {
	maxLatencyMS := configs.Config.MaxLatency * 1000
	if maxLatencyMS <= 0 {
		maxLatencyMS = public.MAXLATENCE
	}
	for _, m := range c.Models {
		if m.Name == model {
			return c.Status == "online" && c.ControlConn != nil && c.GetLatency() < maxLatencyMS
		}
	}
	return false
}

// notifyLatencyExceeded 通知 client app：由于网络延迟过高，暂不采纳该用户的模型算力。
// 通过控制连接发送 LATENCY_EXCEEDED 消息，client 端（Go 客户端 → Python 桌面应用）据此提示用户。
func (s *Server) notifyLatencyExceeded(c *Client, model string) {
	c.ControlConnMutex.Lock()
	defer c.ControlConnMutex.Unlock()
	if c.ControlConn == nil {
		return
	}
	message := public.WSMessage{
		Type: public.LATENCY_EXCEEDED,
		Content: map[string]interface{}{
			"model":   model,
			"latency": c.GetLatency(),
			"limit":   configs.Config.MaxLatency * 1000,
		},
	}
	if err := c.ControlConn.WriteJSON(message); err != nil {
		log.Printf("notify latency exceeded to client %s failed: %v", c.ID, err)
	}
}

// priceEligible returns a Predicate that passes only clients whose model price is within
// the user-configured caps. math.MaxFloat64 caps mean "no restriction".
func priceEligible(maxIPPM, maxOPPM float64) Predicate {
	return func(c *Client, model string) bool {
		for _, m := range c.Models {
			if m.Name == model {
				return m.IPPM <= maxIPPM && m.OPPM <= maxOPPM
			}
		}
		return false
	}
}

// connectionLimitEligible 返回一个 Predicate，过滤掉已达到贡献者会员连接数上限的 client。
// 贡献者会员等级决定上限：普通=3，VIP=10，SVIP=无限（-1）。
// 使用 Client 内存原子计数（ActiveConnections），无 DB 开销。
// 贡献者会员等级实时从数据库查询（GetEffectiveContributorMembership），确保 set-membership 后立即生效。
// 过滤规则：可用连接数必须 > 0（activeConnections < maxConnections），否则直接过滤掉。
// SVIP 现在也是有限连接数（默认 20），达到上限同样被过滤。
func (s *Server) connectionLimitEligible(c *Client, model string) bool {
	if c == nil {
		return false
	}
	limit := s.effectiveMaxConnections(c)
	if limit <= 0 {
		return false // 无可用连接数，直接过滤
	}
	return c.GetActiveConnections() < int32(limit)
}

// effectiveMaxConnections 返回 client 的有效连接数上限。
// 优先使用 client 上报的自定义上限（MaxConnectionsOverride，Python 滑块配置，0~会员上限）；
// 未配置（0）时使用会员等级默认上限。
func (s *Server) effectiveMaxConnections(c *Client) int {
	if c == nil {
		return 0
	}
	// 实时查询该 client 所属用户的贡献者会员等级（过期自动降为普通）
	membership := MembershipNormal
	if c.UserID != "" && s.UserDB != nil {
		membership = s.UserDB.GetEffectiveContributorMembership(c.UserID)
	}
	base := GetMaxConnections(membership)
	// 自定义上限：0 < override <= base 时生效；否则用会员默认
	if c.MaxConnectionsOverride > 0 && c.MaxConnectionsOverride <= base {
		return c.MaxConnectionsOverride
	}
	return base
}

// LoadBalance selects a client for model+user using a Predicate → (Score) → Pick pipeline.
// userID is used to look up per-user price caps; pass an empty string to skip price filtering.
func (s *Server) LoadBalance(model, userID string) *Client {
	return s.LoadBalanceExcluding(model, userID, nil)
}

// LoadBalanceExcluding 与 LoadBalance 相同，但会排除 excludeIDs 中已失败的 client，
// 避免重试时反复 pick 到同一个失效 client。
func (s *Server) LoadBalanceExcluding(model, userID string, excludeIDs map[string]bool) *Client {
	// Resolve price cap (math.MaxFloat64 = no cap configured, i.e. unlimited).
	maxIPPM, maxOPPM := math.MaxFloat64, math.MaxFloat64
	if s.UserPriceCapDB != nil && userID != "" {
		maxIPPM, maxOPPM, _ = s.UserPriceCapDB.GetPriceCap(userID, model)
	}

	// Load the immutable snapshot (lock-free).
	allClients := s.clients.Load().(map[string]map[string]*Client)
	snapshot := allClients[model]

	// Predicate phase.
	// Health is checked first and also identifies dead clients for background cleanup.
	// Additional predicates (price, capacity, geo …) are applied to the survivors.
	extraPredicates := []Predicate{priceEligible(maxIPPM, maxOPPM), s.connectionLimitEligible}

	var eligible []*Client
	var dead []string
	for id, c := range snapshot {
		if excludeIDs != nil && excludeIDs[id] {
			continue
		}
		if !clientHealthy(c, model) {
			// 区分：在线但延迟过高（网络差）→ 剔除并通知 client app；否则视为失效连接待清理。
			if c.Status == "online" && c.ControlConn != nil {
				s.notifyLatencyExceeded(c, model)
			}
			dead = append(dead, id)
			continue
		}
		pass := true
		for _, pred := range extraPredicates {
			if !pred(c, model) {
				pass = false
				break
			}
		}
		if pass {
			eligible = append(eligible, c)
		}
	}

	for _, id := range dead {
		s.RemoveClient(model, id)
	}

	if len(eligible) == 0 {
		log.Println("no eligible client for model:", model)
		return nil
	}

	// Score phase: currently implicit in the pick algorithm (future: weighted scoring).
	// Pick phase.
	return s.pick(model, eligible)
}

// pick selects one client from eligible using the configured load-balance algorithm.
func (s *Server) pick(model string, eligible []*Client) *Client {
	switch s.LoadBalanceAlgorithm {
	case "round-robin":
		// Sort by ID for a stable, deterministic order across goroutines.
		sort.Slice(eligible, func(i, j int) bool { return eligible[i].ID < eligible[j].ID })
		s.clientRBMu.Lock()
		defer s.clientRBMu.Unlock()
		if _, exists := s.clientRoundRobinIndex[model]; !exists {
			s.clientRoundRobinIndex[model] = 0
		}
		index := s.clientRoundRobinIndex[model] % len(eligible)
		s.clientRoundRobinIndex[model] = index + 1
		return eligible[index]

	case "random":
		return eligible[rand.Intn(len(eligible))]

	case "min-conn":
		clientIDs := make([]string, 0, len(eligible))
		eligibleMap := make(map[string]*Client, len(eligible))
		for _, c := range eligible {
			clientIDs = append(clientIDs, c.ID)
			eligibleMap[c.ID] = c
		}
		chatConnections, err := s.ClientFingerprintDB.GetClientChatConnections(clientIDs)
		if err != nil {
			log.Println("get client chat connections error:", err)
			return nil
		}
		selectedID := ""
		minIdleCount := 65535
		for _, result := range chatConnections {
			c, ok := eligibleMap[result.ClientID]
			if !ok {
				continue
			}
			var idle int
			if c.InferenceEngine.Name == "ollama" && c.InferenceEngine.NumParallel > 0 {
				idle = c.InferenceEngine.NumParallel - result.Count
			} else {
				idle = 1 - result.Count
			}
			if idle < minIdleCount {
				minIdleCount = idle
				selectedID = result.ClientID
			}
		}
		if selectedID != "" {
			log.Println("found client:", selectedID, "for model:", model)
			return eligibleMap[selectedID]
		}
		return nil

	case "smart":
		return s.pickSmart(model, eligible)
	}
	log.Println("unknown load balance algorithm:", s.LoadBalanceAlgorithm)
	return nil
}

// ============ smart 两阶段负载均衡算法 ============

// smart 算法性能维度权重（阶段1，从 configs.Config 读取，可通过环境变量配置）。
// 注意：会员等级不再参与性能评分，而是作为阶段2的独立权重。
func (s *Server) smartWeights() (capacity, latency, failure, stability, service, bandwidth float64) {
	cfg := configs.Config
	capacity = cfg.LBWeightCapacity
	latency = cfg.LBWeightLatency
	failure = cfg.LBWeightFailure
	stability = cfg.LBWeightStability
	service = cfg.LBWeightService
	bandwidth = cfg.LBWeightBandwidth
	// 兜底：若权重未配置或和为 0，使用默认值
	if capacity+latency+failure+stability+service+bandwidth <= 0 {
		capacity, latency, failure, stability, service, bandwidth = 0.20, 0.20, 0.15, 0.10, 0.10, 0.25
	}
	return
}

// membershipWeights 返回阶段2的会员等级权重（normal/vip/svip，可配置）。
// 权重越高，该等级在候选 client 中被选中的概率越大。
// 默认 normal=1.0, vip=3.0, svip=8.0，体现充值价值：性能相近时 svip > vip > normal。
func (s *Server) membershipWeights() (normal, vip, svip float64) {
	cfg := configs.Config
	normal = cfg.LBWeightNormal
	vip = cfg.LBWeightVIP
	svip = cfg.LBWeightSVIP
	// 兜底：若未配置或全为 0，使用默认值
	if normal <= 0 && vip <= 0 && svip <= 0 {
		normal, vip, svip = 1.0, 3.0, 8.0
	}
	return
}

// membershipWeight 返回指定会员等级在阶段2的权重。
func (s *Server) membershipWeight(membership string) float64 {
	n, v, sv := s.membershipWeights()
	switch membership {
	case MembershipSVIP:
		return sv
	case MembershipVIP:
		return v
	default:
		return n
	}
}

// capacityScore 可用连接数分：可用连接数 / 上限，范围 [0,1]。
// 各等级连接池有限（normal=1, vip=5, svip=50），饱和时降权。
func capacityScore(maxConn int, activeConn int32) float64 {
	if maxConn <= 0 {
		return 0.0
	}
	avail := int32(maxConn) - activeConn
	if avail < 0 {
		avail = 0
	}
	return float64(avail) / float64(maxConn)
}

// latencyScore 延迟分：基于 MAXLATENCE 的绝对映射，延迟越低分越高
func latencyScore(latency int) float64 {
	if latency <= 0 {
		return 1.0
	}
	score := 1 - float64(latency)/float64(public.MAXLATENCE)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// failureScore 失败率分：1/(1+failures)，失败越多越低。
// 只要失败一次就会显著影响性能评分（0 失败=1.0，1 次失败=0.5）。
func failureScore(failures int32) float64 {
	return 1.0 / (1.0 + float64(failures))
}

// stabilityScore 在线稳定性分：平均在线时长 / 目标时长（1小时）
func stabilityScore(totalOnlineSec, disconnectCount int64) float64 {
	avgOnline := float64(totalOnlineSec) / float64(disconnectCount+1)
	score := avgOnline / float64(public.LB_TARGET_ONLINE_SEC)
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}
	return score
}

// serviceScore 服务等级分：贡献token/小时 / 目标产能
func serviceScore(totalTokens, totalOnlineSec int64) float64 {
	if totalOnlineSec <= 0 {
		return 0
	}
	tokensPerHour := float64(totalTokens) / (float64(totalOnlineSec) / 3600.0)
	score := tokensPerHour / float64(public.LB_TARGET_TOKENS_PER_HOUR)
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}
	return score
}

// bandwidthScore 上行带宽分：带宽 / 目标带宽，范围 [0,1]。
// 考虑 client 到 server 的上行带宽：带宽越高，能承载的并发输出越大，分越高。
func bandwidthScore(bandwidthMbps float64) float64 {
	if bandwidthMbps <= 0 {
		return 0
	}
	target := configs.Config.BandwidthTargetMbps
	if target <= 0 {
		target = 50
	}
	score := bandwidthMbps / target
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}
	return score
}

// perfScore 计算单个 client 的性能综合评分（阶段1，不含会员等级）。
func (s *Server) perfScore(c *Client) float64 {
	wCap, wLat, wFail, wStab, wServ, wBw := s.smartWeights()

	// 会员等级（仅用于获取连接数上限，不参与性能评分）
	maxConn := s.effectiveMaxConnections(c)

	// 在线稳定性 / 服务等级（从 ClientStatsDB 读取）
	var totalOnlineSec, disconnectCount int64
	if s.ClientStatsDB != nil {
		if stats, err := s.ClientStatsDB.GetStats(c.ID); err == nil && stats != nil {
			totalOnlineSec = stats.TotalOnlineSeconds
			disconnectCount = stats.DisconnectCount
		}
	}

	// 服务等级：历史贡献 token（从 TokenUsageDB 聚合）
	var totalTokens int64
	if s.TokenUsageDB != nil {
		usages, err := s.TokenUsageDB.GetIncomeTokenUsage([]string{c.ID}, time.Time{}, time.Now())
		if err == nil {
			for _, u := range usages {
				totalTokens += int64(u.TotalTokens)
			}
		}
	}

	// 上行带宽：client 上报值，未上报则用默认值
	bw := c.BandwidthMbps
	if bw <= 0 {
		bw = configs.Config.ClientBandwidthMbps
	}

	return wCap*capacityScore(maxConn, c.GetActiveConnections()) +
		wLat*latencyScore(c.GetLatency()) +
		wFail*failureScore(c.GetFailures()) +
		wStab*stabilityScore(totalOnlineSec, disconnectCount) +
		wServ*serviceScore(totalTokens, totalOnlineSec) +
		wBw*bandwidthScore(bw)
}

// pickSmart 两阶段选择：
//
//	阶段1：对所有合格 client 计算性能综合评分（容量/延迟/失败率/稳定性/产能），
//	       选出性能评分最高的前 N 个作为候选（N = LBCandidateCount，默认=重试次数）。
//	阶段2：在候选 client 中，按会员等级权重（normal/vip/svip，可配置）加权随机选择。
//
// 这样保证：性能差的 client 不会进入候选；同一性能水平下，高等级会员获得更多流量，
// 但不会因为"无限连接"而垄断（SVIP 现在也是有限连接，饱和时性能评分会下降）。
func (s *Server) pickSmart(model string, eligible []*Client) *Client {
	if len(eligible) == 0 {
		return nil
	}
	jitter := configs.Config.LBJitter
	candidateCount := configs.Config.LBCandidateCount
	if candidateCount <= 0 {
		candidateCount = public.MAX_CHAT_RETRY
	}

	// 阶段1：性能评分 + 随机扰动，选出前 N 名候选
	type scored struct {
		c     *Client
		score float64
	}
	scoredClients := make([]scored, 0, len(eligible))
	for _, c := range eligible {
		score := s.perfScore(c)
		score *= 1 + (rand.Float64()*2-1)*jitter
		scoredClients = append(scoredClients, scored{c: c, score: score})
	}

	// 按性能评分降序排序
	sort.Slice(scoredClients, func(i, j int) bool {
		return scoredClients[i].score > scoredClients[j].score
	})

	// 取前 N 名候选
	n := candidateCount
	if n > len(scoredClients) {
		n = len(scoredClients)
	}
	candidates := scoredClients[:n]

	// 阶段2：在候选中按会员等级权重加权随机选择
	totalWeight := 0.0
	for _, sc := range candidates {
		membership := MembershipNormal
		if sc.c.UserID != "" && s.UserDB != nil {
			membership = s.UserDB.GetEffectiveContributorMembership(sc.c.UserID)
		}
		totalWeight += s.membershipWeight(membership)
	}
	if totalWeight <= 0 {
		return candidates[0].c
	}
	r := rand.Float64() * totalWeight
	for _, sc := range candidates {
		membership := MembershipNormal
		if sc.c.UserID != "" && s.UserDB != nil {
			membership = s.UserDB.GetEffectiveContributorMembership(sc.c.UserID)
		}
		r -= s.membershipWeight(membership)
		if r <= 0 {
			log.Printf("smart LB: model=%s selected=%s perf=%.4f (candidates=%d)", model, sc.c.ID, sc.score, len(candidates))
			return sc.c
		}
	}
	return candidates[len(candidates)-1].c
}

func copyClientsMap(src map[string]map[string]*Client) map[string]map[string]*Client {
	dst := make(map[string]map[string]*Client, len(src))
	for modelName, inner := range src {
		newInner := make(map[string]*Client, len(inner))
		for id, c := range inner {
			newInner[id] = c
		}
		dst[modelName] = newInner
	}
	return dst
}

// RegisterModel 注册模型到指定 client。
// 返回 true 表示本次是新增注册（模型首次注册或 client 实例变化），
// 返回 false 表示该 client 已注册过该模型（心跳场景下的重复注册，无需重复处理）。
func (s *Server) RegisterModel(model *public.Model, client *Client) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	oldMap := s.clients.Load().(map[string]map[string]*Client)
	if inner, exists := oldMap[model.Name]; exists {
		if current, exists := inner[client.ID]; exists && current == client {
			// 心跳场景下重复注册：静默返回，避免刷屏日志
			return false
		}
	}

	newMap := copyClientsMap(oldMap)
	if _, exists := newMap[model.Name]; !exists {
		newMap[model.Name] = make(map[string]*Client)
	}
	newMap[model.Name][client.ID] = client
	s.clients.Store(newMap)
	log.Println("register model:", model.Name, "for client:", client.ID)
	return true
}

// for model marketplace
func (s *Server) GetAllModels() []*MarketplaceModel {
	allClients := s.clients.Load().(map[string]map[string]*Client)

	var marketplaceModels []*MarketplaceModel
	var toRemove []struct{ model, client string }
	for modelName, clientMaps := range allClients {
		if len(clientMaps) == 0 {
			continue
		}

		model := &MarketplaceModel{
			Name:         modelName,
			Type:         "model",
			Size:         "unknown",
			ClientModels: make([]*ClientModel, 0, len(clientMaps)),
		}

		for clientID, client := range clientMaps {
			existModel := false
			for _, m := range client.Models {
				if m.Name == modelName && client.Status == "online" && client.ControlConn != nil && client.GetLatency() < public.MAXLATENCE {
					existModel = true
					model.Size = m.Size
					model.Type = m.Type
					model.Quantization = m.Arch
					model.ClientModels = append(model.ClientModels, &ClientModel{
						Client: client,
						Model:  m,
					})
				}
			}
			if !existModel {
				toRemove = append(toRemove, struct{ model, client string }{modelName, clientID})
			}
		}
		marketplaceModels = append(marketplaceModels, model)
	}
	for _, item := range toRemove {
		s.RemoveClient(item.model, item.client)
	}
	return marketplaceModels
}

// for openAI api compatibility
func (s *Server) GetModels() map[string]interface{} {
	allClients := s.clients.Load().(map[string]map[string]*Client)

	var models []*public.Model
	var toRemove []struct{ model, client string }
	for modelName, clientMaps := range allClients {
		for clientID, client := range clientMaps {
			existModel := false
			for _, m := range client.Models {
				if m.Name == modelName && client.Status == "online" && client.ControlConn != nil && client.GetLatency() < public.MAXLATENCE {
					existModel = true
					models = append(models, &public.Model{
						Name: m.Name,
						Type: m.Type,
						Size: m.Size,
						Arch: m.Arch,
					})
					break
				}
			}
			if !existModel {
				toRemove = append(toRemove, struct{ model, client string }{modelName, clientID})
			}
		}
	}

	for _, item := range toRemove {
		s.RemoveClient(item.model, item.client)
	}

	modelMap := make(map[string]*public.Model)
	for _, model := range models {
		modelMap[model.Name] = model
	}
	resultModels := make([]*openai.Model, 0, len(modelMap))
	for _, model := range modelMap {
		resultModels = append(resultModels, &openai.Model{
			ID:        model.Name,
			Object:    "model",
			OwnedBy:   "star-fire",
			CreatedAt: time.Now().Unix(),
			Root:      "",
			Permission: []openai.Permission{
				{
					ID:                 model.Name + "-permission",
					Object:             "permission",
					AllowCreateEngine:  true,
					AllowSampling:      true,
					AllowLogprobs:      true,
					AllowSearchIndices: false,
					AllowView:          true,
				},
			},
		})
	}
	result := make(map[string]interface{})
	result["data"] = resultModels
	result["object"] = "list"
	return result
}

func (s *Server) AddRespClient(id string, conn *websocket.Conn) {
	s.respClientsMu.Lock()
	defer s.respClientsMu.Unlock()

	s.RespClients[id] = conn
}

func (s *Server) GetRespClient(id string) (*websocket.Conn, bool) {
	s.respClientsMu.RLock()
	defer s.respClientsMu.RUnlock()

	conn, ok := s.RespClients[id]
	return conn, ok
}

func (s *Server) RemoveRespClient(id string) {
	s.respClientsMu.Lock()
	defer s.respClientsMu.Unlock()

	delete(s.RespClients, id)
}

// AddRespClientChan 注册一个 channel 通知 handleChatResponse conn 已就绪
func (s *Server) AddRespClientChan(fingerPrint string) chan struct{} {
	ch := make(chan struct{})
	s.respClientReadyChansMu.Lock()
	s.respClientReadyChans[fingerPrint] = ch
	s.respClientReadyChansMu.Unlock()
	return ch
}

// RemoveRespClientChan 移除等待 channel
func (s *Server) RemoveRespClientChan(fingerPrint string) {
	s.respClientReadyChansMu.Lock()
	delete(s.respClientReadyChans, fingerPrint)
	s.respClientReadyChansMu.Unlock()
}

// NotifyRespClientReady 通知 handleChatResponse 响应连接已就绪
func (s *Server) NotifyRespClientReady(fingerPrint string) {
	s.respClientReadyChansMu.Lock()
	ch, ok := s.respClientReadyChans[fingerPrint]
	if ok {
		close(ch)
		delete(s.respClientReadyChans, fingerPrint)
	}
	s.respClientReadyChansMu.Unlock()
}

// SaveReasoning 合并保存某会话的 reasoning_content（tool_call_id → text）。
func (s *Server) SaveReasoning(convKey string, m map[string]string) {
	if convKey == "" || len(m) == 0 {
		return
	}
	s.reasoningMu.Lock()
	defer s.reasoningMu.Unlock()
	if s.reasoning == nil {
		s.reasoning = make(map[string]map[string]string)
	}
	slot, ok := s.reasoning[convKey]
	if !ok {
		slot = make(map[string]string)
		s.reasoning[convKey] = slot
	}
	for k, v := range m {
		if v != "" {
			slot[k] = v
		}
	}
}

// GetReasoning 返回某会话已保存的 reasoning_content（副本），未命中返回 nil。
func (s *Server) GetReasoning(convKey string) map[string]string {
	if convKey == "" {
		return nil
	}
	s.reasoningMu.RLock()
	defer s.reasoningMu.RUnlock()
	slot, ok := s.reasoning[convKey]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(slot))
	for k, v := range slot {
		out[k] = v
	}
	return out
}

func (s *Server) RemoveClient(modelName string, clientID string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	oldMap := s.clients.Load().(map[string]map[string]*Client)
	inner, exists := oldMap[modelName]
	if !exists || inner[clientID] == nil {
		return
	}

	newMap := copyClientsMap(oldMap)
	delete(newMap[modelName], clientID)
	if len(newMap[modelName]) == 0 {
		delete(newMap, modelName)
	}
	s.clients.Store(newMap)
}

func (s *Server) RemoveClientInstance(modelName string, client *Client) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	oldMap := s.clients.Load().(map[string]map[string]*Client)
	inner, exists := oldMap[modelName]
	if !exists || inner[client.ID] != client {
		return
	}

	newMap := copyClientsMap(oldMap)
	delete(newMap[modelName], client.ID)
	if len(newMap[modelName]) == 0 {
		delete(newMap, modelName)
	}
	s.clients.Store(newMap)
}

func (s *Server) GetClientByModel(model, clientID string) *Client {
	allClients := s.clients.Load().(map[string]map[string]*Client)
	modelClients := allClients[model]
	if modelClients == nil {
		return nil
	}
	return modelClients[clientID]
}

// GetClientByID 在所有模型中查找指定 clientID 的 client。
// 用于请求结束时递减内存连接计数（会员连接数限制用）。
func (s *Server) GetClientByID(clientID string) *Client {
	if clientID == "" {
		return nil
	}
	allClients := s.clients.Load().(map[string]map[string]*Client)
	for _, modelClients := range allClients {
		if c, ok := modelClients[clientID]; ok {
			return c
		}
	}
	return nil
}

// GetUserClientConnections 汇总某用户所有在线 client 的当前连接数。
// 返回 (总连接数, 在线client数)。
func (s *Server) GetUserClientConnections(userID string) (int32, int) {
	if userID == "" {
		return 0, 0
	}
	allClients := s.clients.Load().(map[string]map[string]*Client)
	seen := make(map[string]bool)
	var total int32
	var count int
	for _, modelClients := range allClients {
		for _, c := range modelClients {
			if c.UserID == userID && !seen[c.ID] {
				seen[c.ID] = true
				total += c.GetActiveConnections()
				count++
			}
		}
	}
	return total, count
}

// UserModelInfo represents a model provided by the current user with its price info.
type UserModelInfo struct {
	ModelName string  `json:"model_name"`
	Engine    string  `json:"engine"`
	IPPM      float64 `json:"ippm"`
	OPPM      float64 `json:"oppm"`
	CIPPM     float64 `json:"cippm"`
	ClientID  string  `json:"client_id"`
	ClientIP  string  `json:"client_ip"`
	Online    bool    `json:"online"`
}

// GetUserModels returns all models provided by a specific user's connected clients.
func (s *Server) GetUserModels(userID string) []*UserModelInfo {
	allClients := s.clients.Load().(map[string]map[string]*Client)

	seen := make(map[string]bool) // clientID+modelName dedup
	var result []*UserModelInfo
	for modelName, clients := range allClients {
		for _, client := range clients {
			if client.UserID != userID {
				continue
			}
			for _, m := range client.Models {
				if m.Name != modelName {
					continue
				}
				key := client.ID + "|" + modelName
				if seen[key] {
					continue
				}
				seen[key] = true
				result = append(result, &UserModelInfo{
					ModelName: modelName,
					Engine:    m.Engine,
					IPPM:      m.IPPM,
					OPPM:      m.OPPM,
					CIPPM:     m.CIPPM,
					ClientID:  client.ID,
					ClientIP:  client.IP,
					Online:    client.Status == "online" && client.ControlConn != nil && client.GetLatency() < public.MAXLATENCE,
				})
			}
		}
	}
	return result
}

// UpdateModelPrice updates IPPM/OPPM/CIPPM for a model across all of a user's clients.
func (s *Server) UpdateModelPrice(userID, modelName string, ippm, oppm, cippm float64) (int, error) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	oldMap := s.clients.Load().(map[string]map[string]*Client)
	clients, exists := oldMap[modelName]
	if !exists {
		return 0, fmt.Errorf("model %s not found", modelName)
	}

	var updated []*Client
	for _, client := range clients {
		if client.UserID != userID {
			continue
		}
		for _, m := range client.Models {
			if m.Name == modelName {
				m.IPPM = ippm
				m.OPPM = oppm
				m.CIPPM = cippm
			}
		}
		updated = append(updated, client)
	}

	if len(updated) == 0 {
		return 0, fmt.Errorf("no client found for model %s", modelName)
	}

	// Persist to DB
	for _, client := range updated {
		if err := s.ClientDB.SaveClient(client); err != nil {
			log.Printf("save client %s price to db failed: %v", client.ID, err)
		}

		client.ControlConnMutex.Lock()
		if client.ControlConn != nil {
			message := public.WSMessage{
				Type: public.MODEL_PRICE_UPDATE,
				Content: public.ModelPriceUpdate{
					Model: modelName,
					IPPM:  ippm,
					OPPM:  oppm,
					CIPPM: cippm,
				},
			}
			if err := client.ControlConn.WriteJSON(message); err != nil {
				log.Printf("push model price update to client %s failed: %v", client.ID, err)
			}
		}
		client.ControlConnMutex.Unlock()
	}

	return len(updated), nil
}

func (s *Server) GetTrends(startDate, endDate string) []*Trend {
	if startDate == "" || endDate == "" {
		// use today 00:00:00 as start date today 23:59:59 as end date
		startDate = time.Now().Format("2006-01-02")
		endDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}
	trends, err := s.TrendDB.GetTrendsByTimeRange(startDate, endDate)
	if err != nil {
		log.Printf("get trends failed: %v", err)
		return nil
	}
	return trends
}

func (s *Server) GetTrendsWithPagination(startDate, endDate string, page, size int) *TrendsResponse {
	if startDate == "" || endDate == "" {
		// use today 00:00:00 as start date today 23:59:59 as end date
		startDate = time.Now().Format("2006-01-02")
		endDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}

	// 设置默认分页参数
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	// add timeout context to prevent long-running queries from locking the db
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	trends, total, err := s.TrendDB.GetTrendsByTimeRangeWithPaginationCtx(ctx, startDate, endDate, page, size)
	if err != nil {
		log.Printf("get trends with pagination failed: %v", err)
		return &TrendsResponse{
			Data:       []*Trend{},
			Total:      0,
			Page:       page,
			Size:       size,
			TotalPages: 0,
		}
	}

	// 计算总页数
	totalPages := int((total + int64(size) - 1) / int64(size))

	return &TrendsResponse{
		Data:       trends,
		Total:      total,
		Page:       page,
		Size:       size,
		TotalPages: totalPages,
	}
}

// LoadBalanceEmbedding 专门为embedding模型进行负载均衡
func (s *Server) LoadBalanceEmbedding(model, userID string) *Client {
	// Resolve price cap for this user+model.
	maxIPPM, maxOPPM := math.MaxFloat64, math.MaxFloat64
	if s.UserPriceCapDB != nil && userID != "" {
		maxIPPM, maxOPPM, _ = s.UserPriceCapDB.GetPriceCap(userID, model)
	}

	allClients := s.clients.Load().(map[string]map[string]*Client)

	// 收集所有支持指定embedding模型的在线客户端
	var availableClients []*Client

	for modelName, clients := range allClients {
		if modelName == model {
			log.Printf("Found embedding model: %s, checking clients: %d", modelName, len(clients))

			for _, client := range clients {
				if !clientHealthy(client, model) {
					continue
				}
				// 会员连接数限制：达到上限则过滤
				if !s.connectionLimitEligible(client, model) {
					continue
				}
				for _, m := range client.Models {
					if m.Name == modelName && isEmbeddingModelName(modelName) &&
						m.IPPM <= maxIPPM && m.OPPM <= maxOPPM {
						log.Printf("Found online client for embedding model: %s, client: %s", modelName, client.ID)
						availableClients = append(availableClients, client)
						break
					}
				}
			}
		}
	}

	// 如果没有找到支持该模型的客户端，返回nil
	if len(availableClients) == 0 {
		log.Printf("No available clients found for embedding model: %s", model)
		return nil
	}

	// 从可用的客户端中随机选择一个
	randomIndex := rand.Intn(len(availableClients))
	selectedClient := availableClients[randomIndex]

	log.Printf("Selected client %s for embedding model %s (from %d available clients)",
		selectedClient.ID, model, len(availableClients))

	return selectedClient
}

// IsEmbeddingModel 检查模型名称是否为embedding模型（Server方法）
func (s *Server) IsEmbeddingModel(modelName string) bool {
	return isEmbeddingModelName(modelName)
}

// isEmbeddingModelName 检查模型名称是否为embedding模型
func isEmbeddingModelName(modelName string) bool {
	embeddingModels := []string{
		// OpenAI embedding models
		"text-embedding-ada-002",
		"text-embedding-3-small",
		"text-embedding-3-large",
		"text-similarity-davinci-001",
		"text-similarity-curie-001",
		"text-similarity-babbage-001",
		"text-similarity-ada-001",
		"text-search-ada-doc-001",
		"text-search-ada-query-001",
		"text-search-babbage-doc-001",
		"text-search-babbage-query-001",
		"text-search-curie-doc-001",
		"text-search-curie-query-001",
		"text-search-davinci-doc-001",
		"text-search-davinci-query-001",
		"code-search-ada-code-001",
		"code-search-ada-text-001",
		"code-search-babbage-code-001",
		"code-search-babbage-text-001",

		// BGE (BAAI General Embedding) models
		"bge-large-en",
		"bge-base-en",
		"bge-small-en",
		"bge-large-zh",
		"bge-base-zh",
		"bge-small-zh",
		"bge-large-en-v1.5",
		"bge-base-en-v1.5",
		"bge-small-en-v1.5",
		"bge-large-zh-v1.5",
		"bge-base-zh-v1.5",
		"bge-small-zh-v1.5",
		"bge-m3",
		"bge-multilingual-gemma2",
		"bge-reranker-large",
		"bge-reranker-base",
		"bge-reranker-v2-m3",
		"bge-reranker-v2-gemma",

		// BGE model variations with different naming patterns
		"BAAI/bge-large-en",
		"BAAI/bge-base-en",
		"BAAI/bge-small-en",
		"BAAI/bge-large-zh",
		"BAAI/bge-base-zh",
		"BAAI/bge-small-zh",
		"BAAI/bge-large-en-v1.5",
		"BAAI/bge-base-en-v1.5",
		"BAAI/bge-small-en-v1.5",
		"BAAI/bge-large-zh-v1.5",
		"BAAI/bge-base-zh-v1.5",
		"BAAI/bge-small-zh-v1.5",
		"BAAI/bge-m3",
		"BAAI/bge-multilingual-gemma2",
		"BAAI/bge-reranker-large",
		"BAAI/bge-reranker-base",
		"BAAI/bge-reranker-v2-m3",
		"BAAI/bge-reranker-v2-gemma",
	}

	// 精确匹配
	for _, embeddingModel := range embeddingModels {
		if modelName == embeddingModel {
			return true
		}
	}

	// 关键词匹配（增加BGE相关关键词）
	embeddingKeywords := []string{"embed", "embedding", "similarity", "search", "bge", "reranker"}
	modelLower := strings.ToLower(modelName)
	for _, keyword := range embeddingKeywords {
		if strings.Contains(modelLower, keyword) {
			return true
		}
	}

	// BGE模型的特殊模式匹配
	if strings.Contains(modelLower, "bge-") ||
		strings.Contains(modelLower, "baai/bge") ||
		strings.Contains(modelLower, "bge_") {
		return true
	}

	return false
}
