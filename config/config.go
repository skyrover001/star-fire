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
	EmailHost         string
	EmailPort         int
	EmailUser         string
	EmailPassword     string
	EmailFrom         string
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
		EmailHost:         emailHost,
		EmailPort:         emailPort,
		EmailUser:         emailUser,
		EmailPassword:     emailPassword,
		EmailFrom:         emailFrom,
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
