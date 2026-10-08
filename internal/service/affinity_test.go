package service

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestAffinityKeyFromChatStableAcrossTurns(t *testing.T) {
	base := openai.ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Explain quantum computing in simple terms."},
		},
	}
	k1 := affinityKeyFromChat(base)

	// 多轮对话：追加消息，前缀不变 ⇒ key 不变
	base.Messages = append(base.Messages,
		openai.ChatCompletionMessage{Role: "assistant", Content: "Quantum computing uses qubits..."},
		openai.ChatCompletionMessage{Role: "user", Content: "Give me an example."},
	)
	k2 := affinityKeyFromChat(base)

	if k1 == "" {
		t.Fatal("affinity key should not be empty")
	}
	if k1 != k2 {
		t.Fatalf("same conversation should map to same key, got %q vs %q", k1, k2)
	}
}

func TestAffinityKeyFromChatDiffersByPrefix(t *testing.T) {
	a := affinityKeyFromChat(openai.ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
	})
	b := affinityKeyFromChat(openai.ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a pirate."},
			{Role: "user", Content: "Hello"},
		},
	})
	if a == b {
		t.Fatalf("different system prefix should map to different keys, both %q", a)
	}
}

func TestAffinityKeyFromChatContentPrefixOnly(t *testing.T) {
	long := make([]byte, 512)
	for i := range long {
		long[i] = 'x'
	}
	req := openai.ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: string(long)},
		},
	}
	k1 := affinityKeyFromChat(req)

	// 只改 256B 之后的内容 ⇒ key 不变（只取前缀）
	req.Messages[0].Content = string(long) + "tail-differs"
	k2 := affinityKeyFromChat(req)
	if k1 != k2 {
		t.Fatalf("content beyond 256B prefix should not affect key, got %q vs %q", k1, k2)
	}
}
