package models

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

// 限流策略参考 OpenAI / Anthropic / DeepSeek 等官方实现：
//   - RPM (Requests Per Minute)：每分钟允许的请求数
//   - TPM (Tokens Per Minute)：每分钟允许消耗的 token 数
// 采用令牌桶（token bucket）算法：每个用户/API Key 一个桶，按固定速率补充令牌，
// 桶有容量上限（突发量）。请求到达时消耗 1 个请求令牌 + 预估 token 令牌，
// 任一不足则拒绝（返回 429）。

// RateLimitConfig 单个用户的限流配置。
type RateLimitConfig struct {
	RPM int // 每分钟请求数上限（0 = 不限制）
	TPM int // 每分钟 token 数上限（0 = 不限制）
}

// 各会员等级默认限流配置（参考主流平台量级，可按需调整）。
func DefaultRateLimit(membership string) RateLimitConfig {
	switch membership {
	case MembershipSVIP:
		return RateLimitConfig{RPM: 600, TPM: 2000000}
	case MembershipVIP:
		return RateLimitConfig{RPM: 300, TPM: 1000000}
	default:
		return RateLimitConfig{RPM: 120, TPM: 500000}
	}
}

// ParseRateLimit 解析 "rpm:tpm" 格式的限流配置字符串。
// 解析失败或格式错误时返回零值（不限制）。
func ParseRateLimit(s string) RateLimitConfig {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return RateLimitConfig{}
	}
	rpm, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	tpm, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return RateLimitConfig{}
	}
	if rpm < 0 {
		rpm = 0
	}
	if tpm < 0 {
		tpm = 0
	}
	return RateLimitConfig{RPM: rpm, TPM: tpm}
}

// tokenBucket 单个令牌桶。
type tokenBucket struct {
	mu        sync.Mutex
	capacity  float64 // 桶容量（突发上限）
	tokens    float64 // 当前令牌数
	refillPer float64 // 每秒补充速率
	last      time.Time
}

func newTokenBucket(capacity float64, refillPerSec float64) *tokenBucket {
	return &tokenBucket{
		capacity:  capacity,
		tokens:    capacity,
		refillPer: refillPerSec,
		last:      time.Now(),
	}
}

// take 尝试消耗 n 个令牌，成功返回 true，不足返回 false。
func (b *tokenBucket) take(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	// 补充令牌（按经过时间）
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.refillPer
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.last = now
	}

	if b.tokens < n {
		return false
	}
	b.tokens -= n
	return true
}

// RateLimiter 管理所有用户/API Key 的限流桶。
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*userRateBuckets
}

// userRateBuckets 一个用户的两个桶（RPM + TPM）。
// 记录创建时的配置，用于检测会员等级变化后重建桶。
type userRateBuckets struct {
	rpm    *tokenBucket
	tpm    *tokenBucket
	cfgRPM int
	cfgTPM int
}

// NewRateLimiter 创建限流器。
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*userRateBuckets),
	}
}

// getOrCreate 获取或创建指定 key 的令牌桶。
// 若传入的配置与桶内记录的配置不一致（例如会员等级变化），则重建桶，
// 使新的限流额度立即生效，避免旧的低限额一直卡住用户。
func (rl *RateLimiter) getOrCreate(key string, cfg RateLimitConfig) *userRateBuckets {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if b, ok := rl.buckets[key]; ok {
		// 配置未变化则复用现有桶
		if b.cfgRPM == cfg.RPM && b.cfgTPM == cfg.TPM {
			return b
		}
		// 配置变化：重建桶（重置为新额度）
		b.rpm = nil
		b.tpm = nil
		b.cfgRPM = cfg.RPM
		b.cfgTPM = cfg.TPM
		if cfg.RPM > 0 {
			b.rpm = newTokenBucket(float64(cfg.RPM), float64(cfg.RPM)/60.0)
		}
		if cfg.TPM > 0 {
			b.tpm = newTokenBucket(float64(cfg.TPM), float64(cfg.TPM)/60.0)
		}
		return b
	}

	b := &userRateBuckets{cfgRPM: cfg.RPM, cfgTPM: cfg.TPM}
	if cfg.RPM > 0 {
		// 容量 = RPM（允许整分钟突发），补充速率 = RPM/60 每秒
		b.rpm = newTokenBucket(float64(cfg.RPM), float64(cfg.RPM)/60.0)
	}
	if cfg.TPM > 0 {
		b.tpm = newTokenBucket(float64(cfg.TPM), float64(cfg.TPM)/60.0)
	}
	rl.buckets[key] = b
	return b
}

// Allow 检查并尝试消耗令牌。
// 返回 (是否允许, 被哪个维度限制)。allowed=false 时 limitType 为 "rpm" 或 "tpm"。
func (rl *RateLimiter) Allow(key string, cfg RateLimitConfig, estimatedTokens int) (bool, string) {
	b := rl.getOrCreate(key, cfg)

	// 先消耗请求令牌
	if b.rpm != nil && !b.rpm.take(1) {
		return false, "rpm"
	}
	// 再消耗 token 令牌（预估）
	if b.tpm != nil && !b.tpm.take(float64(estimatedTokens)) {
		return false, "tpm"
	}
	return true, ""
}

// Reset 删除指定 key 的令牌桶，使其在下一次请求时按最新配置重建。
// 用于会员升级/降级后立即生效新的限流额度。
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, key)
}

// EstimateTokens 粗略估算请求消耗的 token 数。
// 参考 OpenAI 的近似规则：约 4 字符 ≈ 1 token，中文约 1.5 字符 ≈ 1 token。
// 这里做保守估算：按字符数 / 3 估算，避免低估导致超限。
func EstimateTokens(messages []openai.ChatCompletionMessage) int {
	total := 0
	for _, m := range messages {
		total += len([]rune(m.Content)) / 3
		// 多模态内容
		for _, part := range m.MultiContent {
			if part.Text != "" {
				total += len([]rune(part.Text)) / 3
			}
		}
	}
	// 加上系统提示等固定开销
	total += 50
	return total
}
