PROJECT_NAME = star-fire
SERVER_BINARY = starfire
CLIENT_BINARY = starfire
BUILD_DIR = build

# 源码路径 - 根据实际项目结构修改这些路径
SERVER_SRC = ./
CLIENT_SRC = ./client/cmd

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
LINUX_AMD64 = GOOS=linux GOARCH=amd64

# DMG 相关配置
APP_NAME = StarFire
DMG_NAME = starfire
TEMP_DIR = $(BUILD_DIR)/temp

.PHONY: all server client install clean package-dmg universal-mac package-dmg-advanced help

# 默认目标：构建所有
all: server client

# 显示帮助信息
help:
	@echo "StarFire Build System"
	@echo "===================="
	@echo ""
	@echo "Available targets:"
	@echo "  all                    - Build server and all clients"
	@echo "  server                 - Build server only"
	@echo "  client                 - Build all client variants"
	@echo "  install                - Build server and clients (alias of all)"
	@echo "  client-windows-amd64   - Build Windows AMD64 client"
	@echo "  client-linux-amd64     - Build Linux AMD64 client"
	@echo "  client-darwin-amd64    - Build macOS Intel client"
	@echo "  client-darwin-arm64    - Build macOS Apple Silicon client"
	@echo "  universal-mac          - Build Universal Binary for macOS"
	@echo "  package-dmg            - Create installer DMG package for macOS"
	@echo "  package-dmg-advanced   - Create advanced DMG package (requires create-dmg)"
	@echo "  clean                  - Clean build artifacts"
	@echo "  help                   - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make all               - Build server and all clients"
	@echo "  make client            - Build all client variants"
	@echo "  make package-dmg       - Build and package macOS client as installer DMG"
	@echo "  make universal-mac     - Create Universal Binary for both Intel and M1/M2"
	@echo ""
	@echo "After DMG installation:"
	@echo "  sudo ./StarFire/install.sh              - Install starfire command"
	@echo "  starfire -host HOST -token TOKEN        - Connect to server"
	@echo "  starfire -daemon -host HOST -token TOKEN - Run in background"

# 创建构建目录
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# 构建服务端
server: $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/server/$(SERVER_BINARY) $(SERVER_SRC)

# 构建所有客户端
client: client-windows-amd64 client-linux-amd64 client-darwin-amd64 client-darwin-arm64

# 构建Windows客户端
client-windows-amd64: $(BUILD_DIR)
	$(WINDOWS_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/client/$(CLIENT_BINARY)_windows_amd64.exe $(CLIENT_SRC)

# 构建Linux客户端
client-linux-amd64: $(BUILD_DIR)
	$(LINUX_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/client/$(CLIENT_BINARY)_linux_amd64 $(CLIENT_SRC)

# 构建Mac Intel客户端
client-darwin-amd64: $(BUILD_DIR)
	$(DARWIN_AMD64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_amd64 $(CLIENT_SRC)

# 构建Mac M系列芯片客户端
client-darwin-arm64: $(BUILD_DIR)
	$(DARWIN_ARM64) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_arm64 $(CLIENT_SRC)

# 编译并安装（构建 server 和所有客户端到 build 目录）
install: all
	@echo "✅ Build complete. Binaries are in $(BUILD_DIR)/"
	@echo "   Server: $(BUILD_DIR)/server/$(SERVER_BINARY)"
	@echo "   Client: $(BUILD_DIR)/client/"

# 创建Universal Binary (Intel + Apple Silicon)
universal-mac: client-darwin-amd64 client-darwin-arm64
	@echo "Creating Universal Binary..."
	@mkdir -p $(BUILD_DIR)/client
	lipo -create -output $(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_universal \
		$(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_amd64 \
		$(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_arm64

# 打包Mac版本为DMG
package-dmg: universal-mac
	@echo "Creating macOS installer package..."
	@rm -rf $(TEMP_DIR)
	@mkdir -p $(TEMP_DIR)
	
	# 复制二进制文件到安装目录
	@mkdir -p $(TEMP_DIR)/StarFire
	@cp $(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_universal $(TEMP_DIR)/StarFire/starfire
	@chmod +x $(TEMP_DIR)/StarFire/starfire
	
	# 创建安装脚本
	@echo 'Creating install script...'
	@echo '#!/bin/bash' > $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'echo "StarFire Client Installer"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'echo "========================="' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'echo ""' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '# 检查是否有管理员权限' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'if [ "$$EUID" -ne 0 ]; then' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "需要管理员权限来安装到系统目录。"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "请使用: sudo ./install.sh"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    exit 1' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'fi' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '# 安装目录' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'INSTALL_DIR="/usr/local/bin"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'BINARY_NAME="starfire"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'SOURCE_BINARY="./starfire"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '# 检查源文件是否存在' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'if [ ! -f "$$SOURCE_BINARY" ]; then' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "错误: 找不到 starfire 二进制文件"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    exit 1' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'fi' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '# 创建安装目录（如果不存在）' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'mkdir -p "$$INSTALL_DIR"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '# 复制二进制文件' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'echo "正在安装 $$BINARY_NAME 到 $$INSTALL_DIR..."' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'cp "$$SOURCE_BINARY" "$$INSTALL_DIR/$$BINARY_NAME"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'chmod +x "$$INSTALL_DIR/$$BINARY_NAME"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '# 验证安装' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'if [ -f "$$INSTALL_DIR/$$BINARY_NAME" ]; then' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "✅ StarFire 客户端安装成功!"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo ""' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "使用方法:"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "  starfire -help                    # 查看帮助"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "  starfire -host HOST -token TOKEN # 连接服务器"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "  starfire -daemon -host HOST -token TOKEN # 后台运行"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo ""' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "注意: /usr/local/bin 已经在大多数系统的 PATH 中"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "如果命令不可用，请将以下行添加到 ~/.zshrc 或 ~/.bash_profile:"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "  export PATH=\"/usr/local/bin:\$$PATH\""' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'else' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    echo "❌ 安装失败"' >> $(TEMP_DIR)/StarFire/install.sh
	@echo '    exit 1' >> $(TEMP_DIR)/StarFire/install.sh
	@echo 'fi' >> $(TEMP_DIR)/StarFire/install.sh
	@chmod +x $(TEMP_DIR)/StarFire/install.sh
	
	# 创建卸载脚本
	@echo '#!/bin/bash' > $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'echo "StarFire Client Uninstaller"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'echo "==========================="' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'echo ""' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '# 检查是否有管理员权限' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'if [ "$$EUID" -ne 0 ]; then' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    echo "需要管理员权限来从系统目录卸载。"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    echo "请使用: sudo ./uninstall.sh"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    exit 1' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'fi' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'INSTALL_DIR="/usr/local/bin"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'BINARY_NAME="starfire"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'if [ -f "$$INSTALL_DIR/$$BINARY_NAME" ]; then' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    echo "正在卸载 StarFire 客户端..."' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    rm "$$INSTALL_DIR/$$BINARY_NAME"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    echo "✅ StarFire 客户端已卸载"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'else' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo '    echo "StarFire 客户端未安装或已被移除"' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@echo 'fi' >> $(TEMP_DIR)/StarFire/uninstall.sh
	@chmod +x $(TEMP_DIR)/StarFire/uninstall.sh
	
	# 创建用户友好的安装说明
	@echo 'StarFire Client v$(VERSION)' > $(TEMP_DIR)/README.txt
	@echo '===========================' >> $(TEMP_DIR)/README.txt
	@echo '' >> $(TEMP_DIR)/README.txt
	@echo '安装方法:' >> $(TEMP_DIR)/README.txt
	@echo '1. 打开 StarFire 文件夹' >> $(TEMP_DIR)/README.txt
	@echo '2. 在终端中运行: sudo ./install.sh' >> $(TEMP_DIR)/README.txt
	@echo '3. 安装完成后可以在任何地方使用 starfire 命令' >> $(TEMP_DIR)/README.txt
	@echo '' >> $(TEMP_DIR)/README.txt
	@echo '使用示例:' >> $(TEMP_DIR)/README.txt
	@echo '  starfire -help' >> $(TEMP_DIR)/README.txt
	@echo '  starfire -host your-server.com -token your-token' >> $(TEMP_DIR)/README.txt
	@echo '  starfire -daemon -host your-server.com -token your-token' >> $(TEMP_DIR)/README.txt
	@echo '' >> $(TEMP_DIR)/README.txt
	@echo '卸载方法:' >> $(TEMP_DIR)/README.txt
	@echo '  sudo ./uninstall.sh' >> $(TEMP_DIR)/README.txt
	
	# 创建详细的使用说明
	@echo '# StarFire 客户端使用指南' > $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '## 安装' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '1. 打开终端，进入 StarFire 目录' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '2. 运行安装脚本:' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '   ```bash' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '   sudo ./install.sh' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '   ```' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '## 使用方法' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '### 基本命令' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '```bash' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '# 查看帮助信息' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo 'starfire -help' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '# 连接到服务器' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo 'starfire -host your-server.com -token your-registration-token' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '# 后台运行（推荐）' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo 'starfire -daemon -host your-server.com -token your-registration-token' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '# 指定推理引擎' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo 'starfire -engine ollama -host your-server.com -token your-token' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '```' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '### 通过配置文件设置价格' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '客户端默认读取当前目录下的 starfire_config.json，也可用 -config 指定路径。' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '配置文件支持为每个模型单独设置价格：' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '```json' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '{' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '  "host": "http://your-server.com",' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '  "token": "your-registration-token",' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '  "ippm": "3.8",' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '  "oppm": "8.3",' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '  "model_prices": {' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '    "DeepSeek-V4": { "engine": "openai", "ippm": "2.99", "oppm": "14.99", "cippm": "0.3" },' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '    "GLM-5.2": { "engine": "openai", "ippm": "3.99", "oppm": "19.99", "cippm": "0.4" }' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '  }' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '}' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '```' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '价格优先级（从高到低）：命令行参数 > 环境变量 > 配置文件。' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '未在 model_prices 中配置的模型使用顶层 ippm/oppm 或默认价格。' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '## 卸载' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '```bash' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo 'sudo ./uninstall.sh' >> $(TEMP_DIR)/StarFire/USAGE.md
	@echo '```' >> $(TEMP_DIR)/StarFire/USAGE.md
	
	# 创建DMG
	@echo "Creating DMG file..."
	@rm -f $(BUILD_DIR)/$(DMG_NAME).dmg
	@hdiutil create -volname "$(APP_NAME) $(VERSION)" \
		-srcfolder $(TEMP_DIR) \
		-ov -format UDZO \
		$(BUILD_DIR)/$(DMG_NAME).dmg
	
	@echo "✅ DMG created successfully: $(BUILD_DIR)/$(DMG_NAME).dmg"
	@echo "� Installer package created in StarFire folder"
	@echo "� Users can install by running: sudo ./StarFire/install.sh"
	@echo "🚀 After installation, use: starfire --host HOST --token TOKEN"
	
	# 清理临时文件
	@rm -rf $(TEMP_DIR)

# 创建高级DMG（使用create-dmg工具，需要brew install create-dmg）
package-dmg-advanced: universal-mac
	@echo "Creating advanced DMG with create-dmg..."
	@rm -rf $(TEMP_DIR)
	@mkdir -p $(TEMP_DIR)
	
	# 创建应用包
	@mkdir -p $(TEMP_DIR)/$(APP_NAME).app/Contents/MacOS
	@mkdir -p $(TEMP_DIR)/$(APP_NAME).app/Contents/Resources
	@cp $(BUILD_DIR)/client/$(CLIENT_BINARY)_darwin_universal $(TEMP_DIR)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)
	@chmod +x $(TEMP_DIR)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)
	
	# 创建Info.plist (简化版)
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '<plist version="1.0">' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '<dict>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundleExecutable</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>$(APP_NAME)</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundleIdentifier</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>com.starfire.client</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundleName</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>$(APP_NAME)</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundleDisplayName</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>StarFire Client</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundleVersion</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>$(VERSION)</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundleShortVersionString</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>$(VERSION)</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>CFBundlePackageType</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>APPL</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>LSMinimumSystemVersion</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<string>10.14</string>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<key>LSUIElement</key>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '	<true/>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '</dict>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	@echo '</plist>' >> $(TEMP_DIR)/$(APP_NAME).app/Contents/Info.plist
	
	# 使用create-dmg创建DMG（如果安装了的话）
	@if command -v create-dmg >/dev/null 2>&1; then \
		echo "Using create-dmg for advanced DMG creation..."; \
		create-dmg \
			--volname "$(APP_NAME) $(VERSION)" \
			--window-pos 200 120 \
			--window-size 600 400 \
			--icon-size 100 \
			--icon "$(APP_NAME).app" 150 200 \
			--hide-extension "$(APP_NAME).app" \
			--app-drop-link 450 200 \
			--no-internet-enable \
			$(BUILD_DIR)/$(DMG_NAME)-advanced.dmg \
			$(TEMP_DIR); \
	else \
		echo "create-dmg not found, using hdiutil instead..."; \
		hdiutil create -volname "$(APP_NAME) $(VERSION)" \
			-srcfolder $(TEMP_DIR) \
			-ov -format UDZO \
			$(BUILD_DIR)/$(DMG_NAME)-advanced.dmg; \
	fi
	
	@echo "✅ Advanced DMG created: $(BUILD_DIR)/$(DMG_NAME)-advanced.dmg"
	@rm -rf $(TEMP_DIR)

# 清理构建文件
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

