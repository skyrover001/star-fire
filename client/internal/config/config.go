package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const IPPM_MAX = 3.99
const OPPM_MAX = 7.99
const CIPPM_MAX = 1.99

type Config struct {
	StarFireHost                    string
	JoinToken                       string
	LocalInferenceType              string
	OllamaHost                      string
	OpenAIKey                       string
	OpenAIBaseURL                   string
	InputTokenPricePerMillion       float64 // 每输入百万tokens定价
	OutputTokenPricePerMillion      float64
	CachedInputTokenPricePerMillion float64 // 缓存命中输入tokens每百万定价
	Deamon                          bool    // 是否以守护进程方式运行
	APPPort                         int
	IPPMMax                         float64
	OPPMMax                         float64
	CIPPMMax                        float64
	OpenAIOnly                      bool // 仅使用 OpenAI 引擎，不包含本地引擎模型
}

func LoadConfig() *Config {
	cfg := &Config{
		OllamaHost:                      "http://localhost:11434",
		InputTokenPricePerMillion:       4.0,  // 默认初始价格
		OutputTokenPricePerMillion:      8.0,  // 默认初始价格
		CachedInputTokenPricePerMillion: 1.0,  // 默认初始价格
		IPPMMax:                         10.0, // 平台输入价格上限
		OPPMMax:                         20.0, // 平台输出价格上限
		CIPPMMax:                        2.0,  // 平台缓存输入价格上限
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
	flag.Float64Var(&cfg.InputTokenPricePerMillion, "ippm", IPPM_MAX, "每输入百万tokens初始定价 (默认: 3.99)")
	flag.Float64Var(&cfg.OutputTokenPricePerMillion, "oppm", OPPM_MAX, "每输出百万tokens初始定价 (默认: 7.99)")
	flag.Float64Var(&cfg.CachedInputTokenPricePerMillion, "cippm", CIPPM_MAX, "缓存命中输入百万tokens初始定价 (默认: 1.99)")
	flag.BoolVar(&cfg.Deamon, "daemon", false, "以守护进程方式运行")
	flag.IntVar(&cfg.APPPort, "port", 19527, "服务端口 (默认:19527)")
	flag.BoolVar(&cfg.OpenAIOnly, "openai-only", false, "仅使用 OpenAI 引擎，不注册本地引擎模型到服务器")

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
	if priceStr := os.Getenv("STAR_FIRE_CACHED_INPUT_TOKEN_PRICE_PER_M"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cfg.CachedInputTokenPricePerMillion = price
		}
	}

	if cfg.OpenAIBaseURL != "" {
		cfg.OpenAIBaseURL = normalizeOpenAIURL(cfg.OpenAIBaseURL)
	}

	// for openai api test
	// cfg.OpenAIKey = "sk-USmmhjs0kiEh9IeXMOSW566ksu64srnqghDDx2YMGdiymArt"
	// cfg.OpenAIKey = "sk-7970b09e7b1b4448843a874faedee1e5"
	// cfg.openAIkey = "NIcuEe8vW7g7bcDa80Db30E4F1684d6aAb7dF015C0D5E2E3"

	return cfg
}

// ValidateConfig 验证配置参数
func ValidateConfig(cfg *Config) error {
	if cfg.StarFireHost == "" {
		return fmt.Errorf("StarFire 服务器地址不能为空，请使用 -host 参数或设置 STARFIRE_HOST 环境变量")
	}

	if err := validateHost(cfg.StarFireHost); err != nil {
		return err
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

func normalizeOpenAIURL(rawURL string) string {
	rawURL = strings.TrimRight(rawURL, "/")
	if !strings.HasSuffix(rawURL, "/v1") {
		rawURL += "/v1"
	}
	return rawURL
}

func validateHost(host string) error {
	parsed, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("无效的服务器地址格式: %s", host)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("服务器地址必须以 http:// 或 https:// 开头: %s", host)
	}

	if parsed.Host == "" {
		return fmt.Errorf("服务器地址缺少主机名或端口: %s", host)
	}

	hostWithoutPort := parsed.Hostname()
	if hostWithoutPort == "" {
		return fmt.Errorf("服务器地址格式无效: %s", host)
	}

	return nil
}
