package configs

import (
	"os"
	"strconv"
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

	return Configuration{
		ServerPort:        port,
		KeepAliveTime:     keepAliveTime,
		MaxLatency:        maxLatency,
		ChatMaxTime:       chatMaxTime,
		WebsocketBuffer:   wsBuffer,
		JWTSecret:         jwtSecret,
		JWTExpiry:         jwtExpiry,
		MaxAPIKeysPerUser: maxAPIKeysPerUser,
		DefaultKeyExpiry:  defaultKeyExpiry,
		LBA:               lba,
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
