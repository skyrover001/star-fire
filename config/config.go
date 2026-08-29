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

	// 限流配置（RPM/TPM，0 = 不限制）
	RateLimitEnabled bool
	RateLimitNormal  string // "rpm:tpm" 普通会员
	RateLimitVIP     string // "rpm:tpm" VIP
	RateLimitSVIP    string // "rpm:tpm" SVIP

	// 智能负载均衡权重配置（smart 算法）
	// 性能维度权重（阶段1：性能综合评分）
	LBWeightCapacity  float64 // 可用连接数权重
	LBWeightLatency   float64 // 延迟权重
	LBWeightFailure   float64 // 失败率权重
	LBWeightStability float64 // 在线稳定性权重
	LBWeightService   float64 // 服务等级权重
	LBWeightBandwidth float64 // 上行带宽权重
	LBJitter          float64 // 随机扰动幅度（±比例）
	LBEMAAplha        float64 // 延迟 EMA 平滑系数

	// 各等级连接池上限（可通过环境变量配置）
	NormalMaxConnections int // normal 默认 1
	VIPMaxConnections    int // vip 默认 5
	SVIPMaxConnections   int // svip 默认 50，最高 100

	// 客户端默认上行带宽（Mbps），用于带宽评分（client 未上报时使用）
	ClientBandwidthMbps float64
	// 带宽评分目标值（Mbps），达到该值带宽分=1.0
	BandwidthTargetMbps float64

	// 性能评分后保留的候选 client 数（默认 = 重试次数 MAX_CHAT_RETRY）
	LBCandidateCount int

	// 会员等级权重（阶段2：在候选 client 中按等级加权选择，可配置）
	// 体现充值价值：性能相近时 svip > vip > normal
	LBWeightNormal float64 // normal 权重
	LBWeightVIP    float64 // vip 权重
	LBWeightSVIP   float64 // svip 权重

	// 数据库配置（支持 sqlite / mysql 一键切换）
	DBDriver             string // sqlite | mysql
	DBDSN                string // 完整 DSN（可选，优先级最高，设置后忽略 DB_HOST 等单项）
	DBHost               string
	DBPort               int
	DBUser               string
	DBPassword           string
	DBName               string
	DBPath               string // sqlite 文件路径
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetimeMin int
	DBLogLevel           string // gorm 日志级别: silent | error | warn | info
}

var Config = loadConfig()

func loadConfig() Configuration {
	port := getEnv("SERVER_PORT", ":8080")
	keepAliveTime, _ := strconv.Atoi(getEnv("KEEPALIVE_TIME", "30"))
	maxLatency, _ := strconv.Atoi(getEnv("MAX_LATENCY", "30"))
	chatMaxTime, _ := strconv.Atoi(getEnv("CHAT_MAX_TIME", "300"))
	wsBuffer, _ := strconv.Atoi(getEnv("WS_BUFFER", "1048576")) // 1MB
	jwtSecret := getEnv("JWT_SECRET", "123456789qwertyuiasdfghjkzxcvbnm")
	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY", "24"))
	maxAPIKeysPerUser, _ := strconv.Atoi(getEnv("MAX_API_KEYS_PER_USER", "3"))
	defaultKeyExpiry, _ := strconv.Atoi(getEnv("DEFAULT_KEY_EXPIRY", "30"))
	lba := getEnv("LBA", "smart")
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

	// 限流配置
	rateLimitEnabled, _ := strconv.ParseBool(getEnv("RATE_LIMIT_ENABLED", "true"))
	rateLimitNormal := getEnv("RATE_LIMIT_NORMAL", "120:500000")
	rateLimitVIP := getEnv("RATE_LIMIT_VIP", "300:1000000")
	rateLimitSVIP := getEnv("RATE_LIMIT_SVIP", "600:2000000")

	// 智能负载均衡权重配置
	lbWeightCapacity, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_CAPACITY", "0.20"), 64)
	lbWeightLatency, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_LATENCY", "0.20"), 64)
	lbWeightFailure, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_FAILURE", "0.15"), 64)
	lbWeightStability, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_STABILITY", "0.10"), 64)
	lbWeightService, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_SERVICE", "0.10"), 64)
	lbWeightBandwidth, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_BANDWIDTH", "0.25"), 64)
	lbJitter, _ := strconv.ParseFloat(getEnv("LB_JITTER", "0.05"), 64)
	lbEMAAplha, _ := strconv.ParseFloat(getEnv("LB_EMA_ALPHA", "0.3"), 64)

	// 各等级连接池上限
	normalMaxConn, _ := strconv.Atoi(getEnv("NORMAL_MAX_CONNECTIONS", "1"))
	if normalMaxConn <= 0 {
		normalMaxConn = 1
	}
	vipMaxConn, _ := strconv.Atoi(getEnv("VIP_MAX_CONNECTIONS", "5"))
	if vipMaxConn <= 0 {
		vipMaxConn = 5
	}
	svipMaxConn, _ := strconv.Atoi(getEnv("SVIP_MAX_CONNECTIONS", "50"))
	if svipMaxConn <= 0 {
		svipMaxConn = 50
	}
	if svipMaxConn > 100 {
		svipMaxConn = 100
	}

	// 客户端上行带宽（Mbps）
	clientBandwidthMbps, _ := strconv.ParseFloat(getEnv("CLIENT_BANDWIDTH_MBPS", "10"), 64)
	if clientBandwidthMbps <= 0 {
		clientBandwidthMbps = 10
	}
	bandwidthTargetMbps, _ := strconv.ParseFloat(getEnv("BANDWIDTH_TARGET_MBPS", "50"), 64)
	if bandwidthTargetMbps <= 0 {
		bandwidthTargetMbps = 50
	}

	// 性能评分后保留的候选 client 数（默认 = 重试次数）
	lbCandidateCount, _ := strconv.Atoi(getEnv("LB_CANDIDATE_COUNT", "3"))
	if lbCandidateCount <= 0 {
		lbCandidateCount = 3
	}

	// 会员等级权重（阶段2），体现充值价值：svip > vip > normal
	lbWeightNormal, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_NORMAL", "1.0"), 64)
	lbWeightVIP, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_VIP", "3.0"), 64)
	lbWeightSVIP, _ := strconv.ParseFloat(getEnv("LB_WEIGHT_SVIP", "8.0"), 64)

	// 数据库配置（sqlite / mysql 一键切换）
	dbDriver := strings.ToLower(strings.TrimSpace(getEnv("DB_DRIVER", "sqlite")))
	if dbDriver != "mysql" {
		dbDriver = "sqlite" // 未知值回退 sqlite，保证默认行为不变
	}
	dbDSN := getEnv("DB_DSN", "")
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "3306"))
	if dbPort <= 0 {
		dbPort = 3306
	}
	dbUser := getEnv("DB_USER", "starfire")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "starfire")
	dbPath := getEnv("DATABASE_PATH", "./data/star_fire.db")

	// 连接池参数：未显式配置时按驱动取默认值
	// （sqlite 单写者，4 连接足够；mysql 可支撑高并发，默认 100）
	dbMaxOpenDefault := "4"
	dbMaxIdleDefault := "10"
	if dbDriver == "mysql" {
		dbMaxOpenDefault = "100"
		dbMaxIdleDefault = "20"
	}
	dbMaxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", dbMaxOpenDefault))
	if dbMaxOpenConns <= 0 {
		dbMaxOpenConns = 4
	}
	dbMaxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", dbMaxIdleDefault))
	if dbMaxIdleConns <= 0 {
		dbMaxIdleConns = 10
	}
	dbConnMaxLifetimeMin, _ := strconv.Atoi(getEnv("DB_CONN_MAX_LIFETIME_MIN", "60"))
	if dbConnMaxLifetimeMin <= 0 {
		dbConnMaxLifetimeMin = 60
	}
	dbLogLevel := strings.ToLower(strings.TrimSpace(getEnv("DB_LOG_LEVEL", "silent")))

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

		RateLimitEnabled: rateLimitEnabled,
		RateLimitNormal:  rateLimitNormal,
		RateLimitVIP:     rateLimitVIP,
		RateLimitSVIP:    rateLimitSVIP,

		LBWeightCapacity:  lbWeightCapacity,
		LBWeightLatency:   lbWeightLatency,
		LBWeightFailure:   lbWeightFailure,
		LBWeightStability: lbWeightStability,
		LBWeightService:   lbWeightService,
		LBWeightBandwidth: lbWeightBandwidth,
		LBJitter:          lbJitter,
		LBEMAAplha:        lbEMAAplha,

		NormalMaxConnections: normalMaxConn,
		VIPMaxConnections:    vipMaxConn,
		SVIPMaxConnections:   svipMaxConn,

		ClientBandwidthMbps: clientBandwidthMbps,
		BandwidthTargetMbps: bandwidthTargetMbps,

		LBCandidateCount: lbCandidateCount,
		LBWeightNormal:   lbWeightNormal,
		LBWeightVIP:      lbWeightVIP,
		LBWeightSVIP:     lbWeightSVIP,

		DBDriver:             dbDriver,
		DBDSN:                dbDSN,
		DBHost:               dbHost,
		DBPort:               dbPort,
		DBUser:               dbUser,
		DBPassword:           dbPassword,
		DBName:               dbName,
		DBPath:               dbPath,
		DBMaxOpenConns:       dbMaxOpenConns,
		DBMaxIdleConns:       dbMaxIdleConns,
		DBConnMaxLifetimeMin: dbConnMaxLifetimeMin,
		DBLogLevel:           dbLogLevel,
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
