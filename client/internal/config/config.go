package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	StarFireHost               string
	JoinToken                  string
	LocalInferenceType         string
	OllamaHost                 string
	OpenAIKey                  string
	OpenAIBaseURL              string
	InputTokenPricePerMillion  float64 // 每输入百万tokens定价
	OutputTokenPricePerMillion float64
	Deamon                     bool // 是否以守护进程方式运行
}

func LoadConfig() *Config {
	cfg := &Config{
		OllamaHost:                 "http://localhost:11434",
		InputTokenPricePerMillion:  4.0, // 默认值为4
		OutputTokenPricePerMillion: 8.0, // 默认值为8
	}

	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "显示帮助信息")
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")
	flag.StringVar(&cfg.StarFireHost, "host", "", "StarFire 服务器地址 (必填)")
	flag.StringVar(&cfg.JoinToken, "token", "", "StarFire 连接令牌 (必填)")
	flag.StringVar(&cfg.LocalInferenceType, "engine", "ollama", "本地推理引擎类型 (ollama, openai, all)")
	flag.StringVar(&cfg.OllamaHost, "ollama-host", cfg.OllamaHost, "Ollama API 服务器地址")
	flag.StringVar(&cfg.OpenAIKey, "openai-key", "", "OpenAI API 密钥")
	flag.StringVar(&cfg.OpenAIBaseURL, "openai-url", cfg.OpenAIBaseURL, "OpenAI API 基础URL")
	flag.Float64Var(&cfg.InputTokenPricePerMillion, "ippm", cfg.InputTokenPricePerMillion, "每输入百万tokens定价 (默认: 4.0)")
	flag.Float64Var(&cfg.OutputTokenPricePerMillion, "oppm", cfg.OutputTokenPricePerMillion, "每输出百万tokens定价 (默认: 8.0)")
	flag.BoolVar(&cfg.Deamon, "daemon", false, "以守护进程方式运行")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "StarFire 客户端\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "使用方法:\n")
		_, _ = fmt.Fprintf(os.Stderr, "  %s [选项]\n\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(os.Stderr, "\n环境变量:\n")
		_, _ = fmt.Fprintf(os.Stderr, "  STARFIRE_HOST         StarFire 服务器地址\n")
		_, _ = fmt.Fprintf(os.Stderr, "  STARFIRE_TOKEN        StarFire 连接令牌\n")
		_, _ = fmt.Fprintf(os.Stderr, "  STARFIRE_ENGINE       本地推理引擎类型\n")
		_, _ = fmt.Fprintf(os.Stderr, "  OLLAMA_HOST           Ollama API 服务器地址\n")
		_, _ = fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY        OpenAI API 密钥\n")
		_, _ = fmt.Fprintf(os.Stderr, "  OPENAI_API_BASE       OpenAI API 基础URL\n")
		_, _ = fmt.Fprintf(os.Stderr, "  STARFIRE_PRICE_PER_M  每百万tokens定价\n")
		_, _ = fmt.Fprintf(os.Stderr, "\n示例:\n")
		_, _ = fmt.Fprintf(os.Stderr, "  %s -host=http://localhost:8080 -token=your-token\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "  %s -host=http://localhost:8080 -token=your-token -price-per-million=10.0\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "  %s -host=http://localhost:8080 -token=your-token -engine=openai -openai-key=your-key\n", os.Args[0])
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

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
	if priceStr := os.Getenv("STAR_FIRE_INPUT_TOKEN_PRICE_PER_M"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cfg.InputTokenPricePerMillion = price
		}
	}
	if priceStr := os.Getenv("STAR_FIRE_OUTPUT_TOKEN_PRICE_PER_M"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cfg.OutputTokenPricePerMillion = price
		}
	}

	// for openai api test
	cfg.OpenAIKey = "sk-Iv8nxLn6yE2r9iL7OvMXYh6IkQty29hCZyoWExXtrJEgWRkD"

	return cfg
}

// ValidateConfig 验证配置参数
func ValidateConfig(cfg *Config) error {
	if cfg.StarFireHost == "" {
		return fmt.Errorf("StarFire 服务器地址不能为空，请使用 -host 参数或设置 STARFIRE_HOST 环境变量")
	}
	if cfg.JoinToken == "" {
		return fmt.Errorf("StarFire 连接令牌不能为空，请使用 -token 参数或设置 STARFIRE_TOKEN 环境变量")
	}

	// 验证引擎类型
	validEngines := map[string]bool{
		"ollama": true,
		"openai": true,
		"all":    true,
	}
	if !validEngines[cfg.LocalInferenceType] {
		return fmt.Errorf("无效的引擎类型: %s，支持的类型: ollama, openai, all", cfg.LocalInferenceType)
	}

	// 如果使用 OpenAI 引擎，检查是否提供了 API 密钥
	if (cfg.LocalInferenceType == "openai" || cfg.LocalInferenceType == "all") && cfg.OpenAIKey == "" {
		return fmt.Errorf("使用 OpenAI 引擎时必须提供 API 密钥，请使用 -openai-key 参数或设置 OPENAI_API_KEY 环境变量")
	}

	// 验证价格参数
	if cfg.InputTokenPricePerMillion < 0 {
		return fmt.Errorf("每百万输入tokens定价不能为负数: %f", cfg.InputTokenPricePerMillion)
	}
	if cfg.OutputTokenPricePerMillion < 0 {
		return fmt.Errorf("每百万输出tokens定价不能为负数: %f", cfg.OutputTokenPricePerMillion)
	}

	return nil
}
