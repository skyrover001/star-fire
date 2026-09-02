package service

import (
	"strconv"
	"strings"

	configs "star-fire/config"

	"github.com/gin-gonic/gin"
)

// 合法路由偏好值。
const (
	RoutingStability = "stability"
	RoutingCost      = "cost"
	RoutingBalanced  = "balanced"
)

// resolveRouting 解析本次请求的路由偏好。
// 优先级：body `routing` 字段 > header `X-SF-Routing` > configs.Config.RoutingDefault。
// （API Key 级 Routing 列可再后置，M2 先不做。）
// 非法值回退默认。返回值为小写合法值之一。
func resolveRouting(c *gin.Context, bodyRouting string) string {
	if v := normalizeRouting(bodyRouting); v != "" {
		return v
	}
	if v := normalizeRouting(c.GetHeader("X-SF-Routing")); v != "" {
		return v
	}
	return normalizeRouting(configs.Config.RoutingDefault)
}

// normalizeRouting 校验并规范化路由偏好值；非法/空返回 ""。
func normalizeRouting(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case RoutingStability, RoutingCost, RoutingBalanced:
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}

// resolveTolerance 从 header 解析容忍度参数（多格式路径无 body 字段时使用）。
// header：X-SF-Max-Latency-Ms（int ms）、X-SF-Min-Stability（float [0,1]）。
// 解析失败或非法返回 0（不限制）。
func resolveTolerance(c *gin.Context) (maxLatencyMs int, minStability float64) {
	if v := c.GetHeader("X-SF-Max-Latency-Ms"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			maxLatencyMs = n
		}
	}
	if v := c.GetHeader("X-SF-Min-Stability"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 && f <= 1 {
			minStability = f
		}
	}
	return
}
