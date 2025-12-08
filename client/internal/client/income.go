package client

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// handleIncomeSimple 处理收益消息的简化版本
func (c *Client) handleIncomeSimple(income float64, usage map[string]interface{}) error {
	// 构建收益消息
	message := map[string]interface{}{
		"type":      "income",
		"amount":    fmt.Sprintf("%.8f", income),
		"currency":  "¥",
		"timestamp": time.Now().Unix(),
		"usage":     usage,
	}

	// 转换为 JSON
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal income message error: %w", err)
	}

	// 发送到 TCP 服务器
	if err := c.sendToTCPServer(string(jsonBytes)); err != nil {
		return fmt.Errorf("send income to TCP server failed: %w", err)
	}

	log.Printf("✓ Income sent: %.8f ¥", income)
	return nil
}

// SendIncome 公开的发送收益消息方法
func (c *Client) SendIncome(income float64, inputTokens, outputTokens int) error {
	usage := map[string]interface{}{
		"prompt_tokens":     inputTokens,
		"completion_tokens": outputTokens,
		"total_tokens":      inputTokens + outputTokens,
	}

	return c.handleIncomeSimple(income, usage)
}
