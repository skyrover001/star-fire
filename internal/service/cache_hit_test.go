package service

import (
	"math"
	"testing"

	"star-fire/internal/models"
)

func TestUpdateCacheHitEMA(t *testing.T) {
	c := &models.Client{}
	// 首次直接赋值
	c.UpdateCacheHit(1.0)
	if got := c.GetCacheHitEMA(); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("first update should assign directly, got %v", got)
	}
	// 第二次 0 → 0.2*0 + 0.8*1 = 0.8
	c.UpdateCacheHit(0.0)
	if got := c.GetCacheHitEMA(); math.Abs(got-0.8) > 1e-9 {
		t.Fatalf("expected 0.8, got %v", got)
	}
}

func TestUpdateCacheHitClamps(t *testing.T) {
	c := &models.Client{}
	c.UpdateCacheHit(5.0) // clamp to 1
	if got := c.GetCacheHitEMA(); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("expected clamp to 1, got %v", got)
	}
	c.UpdateCacheHit(-3.0) // clamp to 0
	if got := c.GetCacheHitEMA(); math.Abs(got-0.8) > 1e-9 {
		t.Fatalf("expected 0.8 after clamp-to-0 update, got %v", got)
	}
}
