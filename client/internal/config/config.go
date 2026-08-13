package config

import (
	"encoding/json"
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

// ModelPrice 单个模型的价格配置
// 价格字段在 JSON 中可能是字符串（如 "2.99"），因此使用自定义反序列化
// 支持字符串或数字两种格式
type ModelPrice struct {
	Engine           string  `json:"engine"`
	InputPrice       float64 `json:"-"`
	OutputPrice      float64 `json:"-"`
	CachedInputPrice float64 `json:"-"`
}

type ProxyBackend struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Enabled bool   `json:"enabled"`
}

func (backend *ProxyBackend) UnmarshalJSON(data []byte) error {
	type proxyBackendJSON struct {
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled *bool  `json:"enabled"`
	}
	var decoded proxyBackendJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	backend.Name = decoded.Name
	backend.BaseURL = decoded.BaseURL
	backend.APIKey = decoded.APIKey
	backend.Enabled = decoded.Enabled == nil || *decoded.Enabled
	return nil
}

// UnmarshalJSON 支持字符串或数字格式的价格字段
func (m *ModelPrice) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Engine           string      `json:"engine"`
		InputPrice       interface{} `json:"ippm"`
		OutputPrice      interface{} `json:"oppm"`
		CachedInputPrice interface{} `json:"cippm"`
	}
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	m.Engine = alias.Engine
	m.InputPrice = toFloat(alias.InputPrice)
	m.OutputPrice = toFloat(alias.OutputPrice)
	m.CachedInputPrice = toFloat(alias.CachedInputPrice)
	return nil
}

// toFloat 将 interface{} 转换为 float64，支持字符串和数字
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		_, _ = fmt.Sscanf(val, "%f", &f)
		return f
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

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
	ModelPrices                     map[string]ModelPrice
	RegisteredModels                []string
	ConfigFile                      string
	ProxyBackends                   []ProxyBackend
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
		ModelPrices:                     map[string]ModelPrice{},
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
	flag.StringVar(&cfg.ConfigFile, "config", "starfire_config.json", "配置文件路径 (默认: starfire_config.json)")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "StarFire 客户端\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "使用方法:\n")
		_, _ = fmt.Fprintf(os.Stderr, "  %s [选项]\n\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(os.Stderr, "\n提示: host/token/engine/api密钥等参数也可写入配置文件（默认 starfire_config.json，用 -config 指定路径），从而减少命令行必填参数。\n")
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
		_, _ = fmt.Fprintf(os.Stderr, "  %s -config=./starfire_config.json\n", os.Args[0])
	}

	flag.Parse()
	explicitFlags := make(map[string]bool)
	flag.Visit(func(current *flag.Flag) {
		explicitFlags[current.Name] = true
	})

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// 环境变量覆盖
	if host := os.Getenv("STARFIRE_HOST"); host != "" && !explicitFlags["host"] {
		cfg.StarFireHost = host
	}
	if token := os.Getenv("STARFIRE_TOKEN"); token != "" && !explicitFlags["token"] {
		cfg.JoinToken = token
	}
	if engine := os.Getenv("STARFIRE_ENGINE"); engine != "" && !explicitFlags["engine"] {
		cfg.LocalInferenceType = engine
	}
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" && !explicitFlags["ollama-host"] {
		cfg.OllamaHost = ollamaHost
	}
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" && !explicitFlags["openai-key"] {
		cfg.OpenAIKey = openaiKey
	}
	if openaiURL := os.Getenv("OPENAI_API_BASE"); openaiURL != "" && !explicitFlags["openai-url"] {
		cfg.OpenAIBaseURL = openaiURL
	}
	if priceStr := os.Getenv("STAR_FIRE_INPUT_TOKEN_PRICE_PER_M"); priceStr != "" && !explicitFlags["ippm"] {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cfg.InputTokenPricePerMillion = price
		}
	}
	if priceStr := os.Getenv("STAR_FIRE_OUTPUT_TOKEN_PRICE_PER_M"); priceStr != "" && !explicitFlags["oppm"] {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cfg.OutputTokenPricePerMillion = price
		}
	}
	if priceStr := os.Getenv("STAR_FIRE_CACHED_INPUT_TOKEN_PRICE_PER_M"); priceStr != "" && !explicitFlags["cippm"] {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cfg.CachedInputTokenPricePerMillion = price
		}
	}

	// 读取配置文件（命令行参数优先级最高，配置文件作为兜底）
	loadConfigFile(cfg, explicitFlags)

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

	// 如果使用 OpenAI 引擎，检查是否提供了 API 密钥。
	// 密钥可来自：-openai-key 参数、OPENAI_API_KEY 环境变量、配置文件中的 proxy_api_key 或 proxy_backends。
	if (cfg.LocalInferenceType == "openai" || cfg.LocalInferenceType == "all") && !hasProxyCredentials(cfg) {
		return fmt.Errorf("使用 OpenAI 引擎时必须提供 API 密钥：请使用 -openai-key 参数、设置 OPENAI_API_KEY 环境变量，或在配置文件（-config）中填写 proxy_api_key / proxy_backends")
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

// hasProxyCredentials 判断是否已配置 OpenAI 兼容引擎所需的 API 密钥。
// 密钥可来自顶层 OpenAIKey，或任一启用且带密钥的代理后端。
func hasProxyCredentials(cfg *Config) bool {
	if cfg.OpenAIKey != "" {
		return true
	}
	for _, backend := range cfg.ProxyBackends {
		if backend.Enabled && backend.APIKey != "" {
			return true
		}
	}
	return false
}

// loadConfigFile 从配置文件读取设置（静默忽略缺失/格式错误）。
// 显式命令行参数和环境变量始终优先于文件。
func loadConfigFile(cfg *Config, explicitFlags map[string]bool) {
	data, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		return // 文件不存在，静默忽略
	}

	var fileCfg struct {
		Host             string                `json:"host"`
		Token            string                `json:"token"`
		Engine           string                `json:"engine"`
		ModelMode        string                `json:"model_mode"`
		OllamaHost       string                `json:"ollama_host"`
		ProxyBaseURL     string                `json:"proxy_base_url"`
		ProxyAPIKey      string                `json:"proxy_api_key"`
		ProxyBackends    []ProxyBackend        `json:"proxy_backends"`
		OpenAIOnly       bool                  `json:"openai_only"`
		APPPort          int                   `json:"app_port"`
		IPPM             interface{}           `json:"ippm"`
		OPPM             interface{}           `json:"oppm"`
		CIPPM            interface{}           `json:"cippm"`
		ModelPrices      map[string]ModelPrice `json:"model_prices"`
		RegisteredModels []string              `json:"registered_models"`
	}
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return // 格式错误，静默忽略
	}

	canUseFile := func(flagName, envName string) bool {
		return !explicitFlags[flagName] && (envName == "" || os.Getenv(envName) == "")
	}
	if fileCfg.Host != "" && canUseFile("host", "STARFIRE_HOST") {
		cfg.StarFireHost = fileCfg.Host
	}
	if fileCfg.Token != "" && canUseFile("token", "STARFIRE_TOKEN") {
		cfg.JoinToken = fileCfg.Token
	}
	if canUseFile("engine", "STARFIRE_ENGINE") {
		if fileCfg.Engine != "" {
			cfg.LocalInferenceType = fileCfg.Engine
		} else if fileCfg.ModelMode == "proxy" {
			cfg.LocalInferenceType = "all"
		} else if fileCfg.ModelMode == "ollama" {
			cfg.LocalInferenceType = "ollama"
		}
	}
	if fileCfg.OllamaHost != "" && canUseFile("ollama-host", "OLLAMA_HOST") {
		cfg.OllamaHost = fileCfg.OllamaHost
	}
	if fileCfg.ProxyBaseURL != "" && canUseFile("openai-url", "OPENAI_API_BASE") {
		cfg.OpenAIBaseURL = fileCfg.ProxyBaseURL
	}
	if fileCfg.ProxyAPIKey != "" && canUseFile("openai-key", "OPENAI_API_KEY") {
		cfg.OpenAIKey = fileCfg.ProxyAPIKey
	}
	if len(fileCfg.ProxyBackends) > 0 && !explicitFlags["openai-url"] && !explicitFlags["openai-key"] && os.Getenv("OPENAI_API_BASE") == "" && os.Getenv("OPENAI_API_KEY") == "" {
		cfg.ProxyBackends = fileCfg.ProxyBackends
		for index := range cfg.ProxyBackends {
			if cfg.ProxyBackends[index].BaseURL != "" {
				cfg.ProxyBackends[index].BaseURL = normalizeOpenAIURL(cfg.ProxyBackends[index].BaseURL)
			}
		}
	}
	if fileCfg.OpenAIOnly && !explicitFlags["openai-only"] {
		cfg.OpenAIOnly = true
	}
	if fileCfg.APPPort > 0 && !explicitFlags["port"] {
		cfg.APPPort = fileCfg.APPPort
	}
	cfg.RegisteredModels = append([]string(nil), fileCfg.RegisteredModels...)

	// 顶层默认价格（仅当命令行和环境变量未显式指定时使用）
	if fileCfg.IPPM != nil && canUseFile("ippm", "STAR_FIRE_INPUT_TOKEN_PRICE_PER_M") {
		cfg.InputTokenPricePerMillion = toFloat(fileCfg.IPPM)
	}
	if fileCfg.OPPM != nil && canUseFile("oppm", "STAR_FIRE_OUTPUT_TOKEN_PRICE_PER_M") {
		cfg.OutputTokenPricePerMillion = toFloat(fileCfg.OPPM)
	}
	if fileCfg.CIPPM != nil && canUseFile("cippm", "STAR_FIRE_CACHED_INPUT_TOKEN_PRICE_PER_M") {
		cfg.CachedInputTokenPricePerMillion = toFloat(fileCfg.CIPPM)
	}

	// 每个模型的价格配置
	if fileCfg.ModelPrices != nil {
		cfg.ModelPrices = fileCfg.ModelPrices
	}
}

// SaveJoinToken persists a replacement registration credential without
// disturbing settings owned by the Python application.
func SaveJoinToken(configFile, token string) error {
	if configFile == "" || token == "" {
		return nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	var fileCfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode registration token: %w", err)
	}
	fileCfg["token"] = tokenJSON

	updated, err := json.MarshalIndent(fileCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}
	updated = append(updated, '\n')

	tempFile := configFile + ".tmp"
	if err := os.WriteFile(tempFile, updated, 0600); err != nil {
		return fmt.Errorf("write temporary config file: %w", err)
	}
	if err := os.Rename(tempFile, configFile); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("replace config file: %w", err)
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
