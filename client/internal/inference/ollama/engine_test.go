package ollama

import "testing"

func TestShouldRegisterOllamaModel(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		running   bool
		want      bool
	}{
		{name: "running chat", modelName: "qwen3:8b", running: true, want: true},
		{name: "stopped chat", modelName: "qwen3:8b", running: false, want: false},
		{name: "stopped embedding", modelName: "nomic-embed-text", running: false, want: true},
		{name: "stopped reranker", modelName: "bge-reranker-v2-m3", running: false, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRegisterOllamaModel(test.modelName, test.running); got != test.want {
				t.Fatalf("shouldRegisterOllamaModel(%q, %t) = %t, want %t", test.modelName, test.running, got, test.want)
			}
		})
	}
}
