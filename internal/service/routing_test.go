package service

import (
	"net/http/httptest"
	"testing"

	configs "star-fire/config"

	"github.com/gin-gonic/gin"
)

func newTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestResolveRoutingPriority(t *testing.T) {
	orig := configs.Config.RoutingDefault
	defer func() { configs.Config.RoutingDefault = orig }()
	configs.Config.RoutingDefault = "stability"

	c, _ := newTestGinContext()

	// body 优先
	if got := resolveRouting(c, "cost"); got != RoutingCost {
		t.Fatalf("body routing = %q, want cost", got)
	}
	// body 为空 → header
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("X-SF-Routing", "balanced")
	if got := resolveRouting(c, ""); got != RoutingBalanced {
		t.Fatalf("header routing = %q, want balanced", got)
	}
	// body 非法 → header
	if got := resolveRouting(c, "bogus"); got != RoutingBalanced {
		t.Fatalf("invalid body should fall back to header, got %q", got)
	}
	// 都空 → config 默认
	c.Request.Header.Del("X-SF-Routing")
	if got := resolveRouting(c, ""); got != RoutingStability {
		t.Fatalf("default routing = %q, want stability", got)
	}
}

func TestResolveRoutingInvalidFallsBackToDefault(t *testing.T) {
	orig := configs.Config.RoutingDefault
	defer func() { configs.Config.RoutingDefault = orig }()
	configs.Config.RoutingDefault = "cost"

	c, _ := newTestGinContext()
	c.Request = httptest.NewRequest("POST", "/", nil)
	// body 非法 + header 非法 → 默认
	c.Request.Header.Set("X-SF-Routing", "weird")
	if got := resolveRouting(c, "nope"); got != RoutingCost {
		t.Fatalf("invalid routing = %q, want default cost", got)
	}
}

func TestResolveTolerance(t *testing.T) {
	c, _ := newTestGinContext()
	c.Request = httptest.NewRequest("POST", "/", nil)

	// 无 header → 0,0
	ml, ms := resolveTolerance(c)
	if ml != 0 || ms != 0 {
		t.Fatalf("no header = (%d,%v), want (0,0)", ml, ms)
	}

	// 合法 header
	c.Request.Header.Set("X-SF-Max-Latency-Ms", "500")
	c.Request.Header.Set("X-SF-Min-Stability", "0.7")
	ml, ms = resolveTolerance(c)
	if ml != 500 || ms != 0.7 {
		t.Fatalf("valid header = (%d,%v), want (500,0.7)", ml, ms)
	}

	// 非法值 → 0
	c.Request.Header.Set("X-SF-Max-Latency-Ms", "abc")
	c.Request.Header.Set("X-SF-Min-Stability", "2.0")
	ml, ms = resolveTolerance(c)
	if ml != 0 || ms != 0 {
		t.Fatalf("invalid header = (%d,%v), want (0,0)", ml, ms)
	}
}
