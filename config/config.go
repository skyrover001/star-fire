package configs

import (
	"os"
	"strconv"
	"strings"
)

type Configuration struct {
	ServerPort        string
	KeepAliveTime     int
	MaxLatency        int
	ChatMaxTime       int
	WebsocketBuffer   int
	JWTSecret         string
	JWTExpiry         int
	MaxAPIKeysPerUser int
	DefaultKeyExpiry  int
	LBA               string
	EmailHost         string
	EmailPort         int
	EmailUser         string
	EmailPassword     string
	EmailFrom         string
	// 新增embedding相关配置
	EnableEmbeddingModels        bool
	EmbeddingInputTokenPricePerM float64
	SupportedEmbeddingModels     []string

	// 设置所有模型价格上限，提示用户设置超过这个值，将会被重置为这个值
	AllModelOutPutMaxPrice float64
	AllModelInputMaxPrice  float64
}

var Config = loadConfig()

func loadConfig() Configuration {
	port := getEnv("SERVER_PORT", ":8080")
	keepAliveTime, _ := strconv.Atoi(getEnv("KEEPALIVE_TIME", "30"))
	maxLatency, _ := strconv.Atoi(getEnv("MAX_LATENCY", "5"))
	chatMaxTime, _ := strconv.Atoi(getEnv("CHAT_MAX_TIME", "300"))
	wsBuffer, _ := strconv.Atoi(getEnv("WS_BUFFER", "1048576")) // 1MB
	jwtSecret := getEnv("JWT_SECRET", "123456789qwertyuiasdfghjkzxcvbnm")
	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY", "24"))
	maxAPIKeysPerUser, _ := strconv.Atoi(getEnv("MAX_API_KEYS_PER_USER", "3"))
	defaultKeyExpiry, _ := strconv.Atoi(getEnv("DEFAULT_KEY_EXPIRY", "30"))
	lba := getEnv("LBA", "round-robin")
	emailHost := getEnv("EMAIL_HOST", "")
	emailPort, _ := strconv.Atoi(getEnv("EMAIL_PORT", "587"))
	emailUser := getEnv("EMAIL_USER", "")
	emailPassword := getEnv("EMAIL_PASSWORD", "")
	emailFrom := getEnv("EMAIL_FROM", "")
	if emailHost != "" && (emailPort == 0 || emailUser == "" || emailPassword == "" || emailFrom == "") {
		panic("Email configuration is incomplete. Please set EMAIL_HOST, EMAIL_PORT, EMAIL_USER, EMAIL_PASSWORD, and EMAIL_FROM.")
	}

	// 新增embedding配置加载
	enableEmbedding, _ := strconv.ParseBool(getEnv("ENABLE_EMBEDDING_MODELS", "true"))
	embeddingInputPrice, _ := strconv.ParseFloat(getEnv("STAR_FIRE_EMBEDDING_INPUT_TOKEN_PRICE_PER_M", "0.1"), 64)

	// 设置共享到平台的所有模型的输入输出token价格的上限
	allModelInputMaxPrice, _ := strconv.ParseFloat(getEnv("INPUT_TOKEN_PRICE_PER_MAX", "10.0"), 64)
	allModelOutputMaxPrice, _ := strconv.ParseFloat(getEnv("OUTPUT_TOKEN_PRICE_PER_MAX", "20.0"), 64)

	// 解析支持的embedding模型列表
	embeddingModelsStr := getEnv("SUPPORTED_EMBEDDING_MODELS", "text-embedding-ada-002,text-embedding-3-small,text-embedding-3-large")
	var supportedEmbeddingModels []string
	if embeddingModelsStr != "" {
		// 简单的逗号分割
		models := strings.Split(embeddingModelsStr, ",")
		for _, model := range models {
			if trimmed := strings.TrimSpace(model); trimmed != "" {
				supportedEmbeddingModels = append(supportedEmbeddingModels, trimmed)
			}
		}
	}

	return Configuration{
		ServerPort:                   port,
		KeepAliveTime:                keepAliveTime,
		MaxLatency:                   maxLatency,
		ChatMaxTime:                  chatMaxTime,
		WebsocketBuffer:              wsBuffer,
		JWTSecret:                    jwtSecret,
		JWTExpiry:                    jwtExpiry,
		MaxAPIKeysPerUser:            maxAPIKeysPerUser,
		DefaultKeyExpiry:             defaultKeyExpiry,
		LBA:                          lba,
		EmailHost:                    emailHost,
		EmailPort:                    emailPort,
		EmailUser:                    emailUser,
		EmailPassword:                emailPassword,
		EmailFrom:                    emailFrom,
		EnableEmbeddingModels:        enableEmbedding,
		EmbeddingInputTokenPricePerM: embeddingInputPrice,
		SupportedEmbeddingModels:     supportedEmbeddingModels,

		AllModelInputMaxPrice:  allModelInputMaxPrice,
		AllModelOutPutMaxPrice: allModelOutputMaxPrice,
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
