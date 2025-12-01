package openai

import (
	"github.com/sashabaranov/go-openai"
)

// KimiStreamChoice 兼容 Kimi 把 usage 放在 choice 里的格式
type KimiStreamChoice struct {
	Index        int                                    `json:"index"`
	Delta        openai.ChatCompletionStreamChoiceDelta `json:"delta"`
	FinishReason openai.FinishReason                    `json:"finish_reason,omitempty"`
	Usage        *openai.Usage                          `json:"usage,omitempty"` // Kimi 特有
}

// KimiStreamResponse 兼容响应结构
type KimiStreamResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []KimiStreamChoice `json:"choices"`
	Usage   *openai.Usage      `json:"usage,omitempty"` // 标准位置
}

// ToStandardResponse 转换为标准 OpenAI 响应格式
func (k *KimiStreamResponse) ToStandardResponse(usage *openai.Usage) openai.ChatCompletionStreamResponse {
	choices := make([]openai.ChatCompletionStreamChoice, len(k.Choices))
	for i, c := range k.Choices {
		choices[i] = openai.ChatCompletionStreamChoice{
			Index:        c.Index,
			Delta:        c.Delta,
			FinishReason: c.FinishReason,
		}
	}

	return openai.ChatCompletionStreamResponse{
		ID:      k.ID,
		Object:  k.Object,
		Created: k.Created,
		Model:   k.Model,
		Choices: choices,
		Usage:   usage,
	}
}

// ExtractUsage 提取 usage（优先从 choices，其次从顶层）
func (k *KimiStreamResponse) ExtractUsage() *openai.Usage {
	// 优先从 choices 里提取（Kimi 格式）
	for _, choice := range k.Choices {
		if choice.Usage != nil {
			return choice.Usage
		}
	}

	// 其次从顶层提取（标准格式）
	if k.Usage != nil {
		return k.Usage
	}

	return nil
}
