package models

import (
	"testing"
)

// TestDefaultRateLimit 验证各会员等级默认限流配置。
func TestDefaultRateLimit(t *testing.T) {
	cases := []struct {
		membership string
		rpm        int
		tpm        int
	}{
		{MembershipSVIP, 600, 2000000},
		{MembershipVIP, 300, 1000000},
		{MembershipNormal, 120, 500000},
		{"unknown", 120, 500000},
	}
	for _, c := range cases {
		cfg := DefaultRateLimit(c.membership)
		if cfg.RPM != c.rpm || cfg.TPM != c.tpm {
			t.Errorf("DefaultRateLimit(%q) = %+v, want rpm=%d tpm=%d", c.membership, cfg, c.rpm, c.tpm)
		}
	}
}

// TestRateLimiterConfigChangeRebuildsBucket 验证会员等级变化后桶会重建，
// 使新的限流额度立即生效（核心 bug 修复）。
func TestRateLimiterConfigChangeRebuildsBucket(t *testing.T) {
	rl := NewRateLimiter()
	key := "user:test"

	// 先用普通会员配置，耗尽 RPM 桶
	normal := DefaultRateLimit(MembershipNormal) // RPM 120
	for i := 0; i < normal.RPM; i++ {
		allowed, _ := rl.Allow(key, normal, 10)
		if !allowed {
			t.Fatalf("request %d should be allowed under normal config", i+1)
		}
	}
	// 第 121 次应被 RPM 限制
	if allowed, lt := rl.Allow(key, normal, 10); allowed || lt != "rpm" {
		t.Fatalf("expected rpm limit after exhausting normal bucket, got allowed=%v type=%q", allowed, lt)
	}

	// 升级为 SVIP：配置变化，桶应重建，立即恢复满额度
	svip := DefaultRateLimit(MembershipSVIP) // RPM 600
	allowed, lt := rl.Allow(key, svip, 10)
	if !allowed {
		t.Fatalf("expected allowed after upgrading to SVIP (bucket rebuilt), got type=%q", lt)
	}
}

// TestRateLimiterSameConfigReusesBucket 验证配置不变时复用同一桶（不重置额度）。
func TestRateLimiterSameConfigReusesBucket(t *testing.T) {
	rl := NewRateLimiter()
	key := "user:test"
	cfg := DefaultRateLimit(MembershipNormal)

	// 消耗到只剩 1 个请求令牌
	for i := 0; i < cfg.RPM-1; i++ {
		if allowed, _ := rl.Allow(key, cfg, 10); !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	// 再次用相同配置：应复用桶，第 RPM 次仍允许，第 RPM+1 次被限
	if allowed, _ := rl.Allow(key, cfg, 10); !allowed {
		t.Fatal("expected allowed (same config reuses bucket)")
	}
	if allowed, _ := rl.Allow(key, cfg, 10); allowed {
		t.Fatal("expected rpm limit after exhausting bucket with same config")
	}
}

// TestRateLimiterReset 验证 Reset 后桶被删除，下次请求按最新配置重建。
func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter()
	key := "user:test"
	cfg := DefaultRateLimit(MembershipNormal)

	// 耗尽桶
	for i := 0; i < cfg.RPM; i++ {
		rl.Allow(key, cfg, 10)
	}
	if allowed, _ := rl.Allow(key, cfg, 10); allowed {
		t.Fatal("expected rpm limit after exhausting bucket")
	}

	// Reset 后应恢复满额度
	rl.Reset(key)
	if allowed, _ := rl.Allow(key, cfg, 10); !allowed {
		t.Fatal("expected allowed after Reset")
	}
}
