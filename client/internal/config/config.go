package config

import (
	"flag"
	"os"
)

type Config struct {
	StarFireHost       string
	JoinToken          string
	LocalInferenceType string
	OllamaHost         string
	OpenAIKey          string
	OpenAIBaseURL      string
}

func LoadConfig() *Config {
	cfg := &Config{
		OllamaHost: "http://localhost:11434",
	}

	flag.StringVar(&cfg.StarFireHost, "host", "", "StarFire host")
	flag.StringVar(&cfg.JoinToken, "token", "", "StarFire join token")
	flag.StringVar(&cfg.LocalInferenceType, "engine", "ollama", "local inference(ollama, openai, all)")
	flag.StringVar(&cfg.OllamaHost, "ollama-host", cfg.OllamaHost, "Ollama api host")
	flag.StringVar(&cfg.OpenAIKey, "openai-key", "", "OpenAI API key")
	flag.StringVar(&cfg.OpenAIBaseURL, "openai-url", cfg.OpenAIBaseURL, "OpenAI base URL")
	flag.Parse()

	// 环境变量覆盖
	if host := os.Getenv("STARFIRE_HOST"); host != "" {
		cfg.StarFireHost = host
	}
	if token := os.Getenv("STARFIRE_TOKEN"); token != "" {
		cfg.JoinToken = token
	}
	if engine := os.Getenv("STARFIRE_ENGINE"); engine != "" {
		cfg.LocalInferenceType = engine
	}
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		cfg.OllamaHost = ollamaHost
	}
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		cfg.OpenAIKey = openaiKey
	}
	if openaiURL := os.Getenv("OPENAI_API_BASE"); openaiURL != "" {
		cfg.OpenAIBaseURL = openaiURL
	}

	// for openai api test
	cfg.OpenAIKey = "sk-Iv8nxLn6yE2r9iL7OvMXYh6IkQty29hCZyoWExXtrJEgWRkD"
	cfg.OpenAIBaseURL = "https://api.moonshot.cn/v1"
	return cfg
}
