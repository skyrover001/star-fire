# 项目信息
PROJECT_NAME = star-fire
SERVER_BINARY = server
CLIENT_BINARY = client
BUILD_DIR = build

# Go编译环境
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean

# 版本信息
VERSION = $(shell git describe --tags --always 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date +%FT%T%z)
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 目标平台
WINDOWS_AMD64 = GOOS=windows GOARCH=amd64
DARWIN_AMD64 = GOOS=darwin GOARCH=amd64
DARWIN_ARM64 = GOOS=darwin GOARCH=arm64

.PHONY: all server client clean

# 默认目标：构建所有
all: server client

# 创建构建目录
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# 构建服务端
server: $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) ./cmd/server

# 构建所有客户端
client: client-windows-amd64 client-darwin-amd64 client-darwin-arm64

# 构建Windows客户端
client-windows-amd64: $(BUILD_DIR)
	$(WINDOWS_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_BINARY)_windows_amd64.exe ./cmd/client

# 构建Mac Intel客户端
client-darwin-amd64: $(BUILD_DIR)
	$(DARWIN_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_BINARY)_darwin_amd64 ./cmd/client

# 构建Mac M系列芯片客户端
client-darwin-arm64: $(BUILD_DIR)
	$(DARWIN_ARM64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_BINARY)_darwin_arm64 ./cmd/client

# 清理构建文件
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)