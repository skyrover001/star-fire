# StarFire 构建指南

## 概述

StarFire 项目包含服务端和客户端两个部分，支持多平台构建和打包。

## 构建命令

### 基本构建

```bash
# 构建所有组件
make all

# 仅构建服务端
make server

# 构建所有客户端变体
make client

# 构建特定平台的客户端
make client-windows-amd64   # Windows AMD64
make client-darwin-amd64    # macOS Intel
make client-darwin-arm64    # macOS Apple Silicon
```

### macOS 特殊构建

```bash
# 创建 Universal Binary（同时支持 Intel 和 Apple Silicon）
make universal-mac

# 打包为 DMG 文件
make package-dmg

# 创建高级 DMG（需要安装 create-dmg）
make package-dmg-advanced
```

## macOS DMG 打包

### 创建安装器 DMG

使用 `make package-dmg` 创建命令行安装器：

- 自动创建 Universal Binary（支持 Intel 和 Apple Silicon）
- 生成安装和卸载脚本
- 创建用户友好的安装器 DMG

### DMG 内容

打包后的 DMG 包含：
```
StarFire.dmg
├── README.txt              # 快速安装说明
└── StarFire/
    ├── starfire            # Universal Binary 可执行文件
    ├── install.sh          # 系统安装脚本
    ├── uninstall.sh        # 卸载脚本
    └── USAGE.md           # 详细使用说明
```

### 用户安装流程

1. **下载并挂载** DMG 文件
2. **进入 StarFire 文件夹**
3. **运行安装命令**:
   ```bash
   sudo ./install.sh
   ```
4. **全局使用**:
   ```bash
   starfire --host your-server.com --token your-token
   starfire --daemon --host your-server.com --token your-token
   ```

### 高级 DMG 打包

使用 `make package-dmg-advanced` 创建更专业的 DMG：

1. 安装 create-dmg 工具：
   ```bash
   brew install create-dmg
   ```

2. 运行高级打包：
   ```bash
   make package-dmg-advanced
   ```

高级 DMG 功能：
- 自定义窗口大小和位置
- 拖拽安装到 Applications 文件夹
- 更好的视觉体验

## 文件结构

构建后的文件结构：

```
build/
├── client/
│   ├── starfire_darwin_amd64        # macOS Intel 客户端
│   ├── starfire_darwin_arm64        # macOS Apple Silicon 客户端
│   ├── starfire_darwin_universal    # Universal Binary
│   └── starfire_windows_amd64.exe   # Windows 客户端
├── server/
│   └── starfire                     # 服务端
└── starfire.dmg                     # macOS 安装器 DMG
```

## 自定义配置

### 修改应用信息

在 Makefile 中可以修改以下变量：

```makefile
APP_NAME = StarFire          # 应用名称
DMG_NAME = starfire         # DMG 文件名
VERSION = $(shell git describe --tags --always 2>/dev/null || echo "unknown")
```

### 添加应用图标

1. 将图标文件 `icon.icns` 放入 `assets/` 目录
2. 重新运行 `make package-dmg`

图标要求：
- 格式：`.icns`
- 推荐尺寸：1024x1024
- 可使用 `iconutil` 工具从 PNG 转换

### 创建图标示例

```bash
# 创建 iconset 目录
mkdir MyIcon.iconset

# 添加不同尺寸的 PNG 文件
# icon_16x16.png, icon_32x32.png, 等等...

# 转换为 icns
iconutil -c icns MyIcon.iconset -o assets/icon.icns
```

## 清理构建

```bash
# 清理所有构建文件
make clean
```

## 帮助信息

```bash
# 显示所有可用命令
make help
```

## 注意事项

1. **Universal Binary**: 创建的 Universal Binary 同时支持 Intel 和 Apple Silicon Mac
2. **代码签名**: 生产环境建议添加 Apple Developer 代码签名
3. **公证**: App Store 分发需要通过 Apple 公证流程
4. **系统要求**: 最低支持 macOS 10.14+

## 安装和使用

### 从 DMG 安装（推荐）

1. 下载 `starfire.dmg` 文件
2. 双击挂载 DMG 文件
3. 将 `StarFire.app` 拖拽到 `Applications` 文件夹
4. 从 Applications 启动 StarFire，或从终端运行：
   ```bash
   /Applications/StarFire.app/Contents/MacOS/StarFire --daemon
   ```

### 直接使用二进制文件

如果你不想安装应用包，可以直接使用 Universal Binary：

```bash
# 运行客户端
./build/client/starfire_darwin_universal --daemon

# 查看帮助
./build/client/starfire_darwin_universal --help
```

### 安装说明

StarFire 是一个命令行应用，当从 Finder 启动时：
- 会自动打开终端并以守护进程模式运行
- 显示启动状态信息
- 在后台持续运行

从终端启动时：
- 直接运行命令行版本
- 支持所有命令行参数
- 可以看到详细的输出信息

## 故障排除

### 常见问题

1. **lipo 命令失败**: 确保已安装 Xcode Command Line Tools
2. **hdiutil 权限问题**: 检查目标目录的写入权限
3. **create-dmg 未找到**: 使用 `brew install create-dmg` 安装
4. **应用无法启动**: 检查 macOS 安全设置，可能需要在"系统偏好设置 > 安全性与隐私"中允许运行

### 验证构建

```bash
# 检查 Universal Binary 架构
file build/client/starfire_darwin_universal

# 验证 DMG 文件
hdiutil verify build/starfire.dmg

# 测试应用启动
build/client/starfire_darwin_universal --help
```
