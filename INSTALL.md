# StarFire 客户端安装指南

## 快速安装

### 步骤 1: 下载和挂载
1. 下载 `starfire.dmg` 文件
2. 双击 DMG 文件进行挂载

### 步骤 2: 安装客户端
1. 在打开的 DMG 窗口中，进入 `StarFire` 文件夹
2. 打开终端，导航到 StarFire 文件夹
3. 运行安装脚本:
   ```bash
   sudo ./install.sh
   ```
4. 输入管理员密码确认安装

### 步骤 3: 使用客户端
安装完成后，你可以在任何地方使用 `starfire` 命令：

```bash
# 查看帮助
starfire --help

# 连接服务器
starfire --host your-server.com --token your-registration-token

# 后台运行（推荐）
starfire --daemon --host your-server.com --token your-registration-token

# 指定推理引擎
starfire --engine ollama --host your-server.com --token your-token
```

## 使用说明

### 基本命令格式

```bash
starfire [选项]
```

### 主要选项

- `--host` : 服务器地址
- `--token` : 注册令牌  
- `--daemon` : 后台运行模式
- `--engine` : 推理引擎类型 (ollama, openai, all)
- `--help` : 显示帮助信息

### 使用示例

```bash
# 基本连接
starfire --host starfire.example.com --token abc123

# 后台运行
starfire --daemon --host starfire.example.com --token abc123

# 指定引擎
starfire --engine ollama --host starfire.example.com --token abc123

# 使用配置文件
starfire --config ~/.starfire/config.yaml --daemon
```

## 配置

### 创建配置文件
你可以创建配置文件 `~/.starfire/config.yaml` 来简化使用：

```yaml
host: starfire.example.com
token: your-registration-token
engine: ollama
```

然后直接运行：
```bash
starfire --daemon
```

### 环境变量
也可以使用环境变量：

```bash
export STARFIRE_HOST=starfire.example.com
export STARFIRE_TOKEN=your-registration-token
starfire --daemon
```

## 进程管理

### 查看运行状态
```bash
# 查找 StarFire 进程
ps aux | grep starfire

# 检查进程是否运行
pgrep starfire
```

### 停止后台进程
```bash
# 优雅停止
pkill starfire

# 强制停止
pkill -9 starfire

# 或者使用 PID
kill <进程ID>
```

### 查看日志
```bash
# 实时查看日志
tail -f ~/.starfire/logs/starfire-client.log

# 查看最近日志
tail -n 100 ~/.starfire/logs/starfire-client.log
```

## 卸载

如需卸载 StarFire 客户端：

1. **卸载程序**:
   ```bash
   sudo /usr/local/bin/starfire/uninstall.sh
   ```
   或在原 DMG 文件夹中：
   ```bash
   sudo ./uninstall.sh
   ```

2. **清理配置和日志** (可选):
   ```bash
   rm -rf ~/.starfire
   ```

## 故障排除

### 安装问题

**权限错误**:
- 确保使用 `sudo ./install.sh` 安装
- 检查是否有管理员权限

**找不到文件**:
- 确保在正确的目录下运行安装脚本
- 检查 DMG 是否正确挂载

### 运行问题

**命令找不到**:
```bash
# 检查 PATH 环境变量
echo $PATH

# 如果 /usr/local/bin 不在 PATH 中，添加它
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**连接失败**:
- 检查网络连接
- 验证服务器地址是否正确
- 确认注册令牌有效
- 检查防火墙设置

**推理引擎问题**:
- 确保选择的推理引擎已安装并运行
- 对于 Ollama: `ollama serve`
- 对于 OpenAI: 检查 API 密钥和端点

### 日志调试

**启用详细日志**:
```bash
# 不使用 --daemon 查看实时输出
starfire --host your-server.com --token your-token

# 或查看日志文件
tail -f ~/.starfire/logs/starfire-client.log
```

**常见错误**:
- `connection refused`: 服务器不可达或端口被阻止
- `authentication failed`: 令牌无效或过期
- `engine not found`: 指定的推理引擎未安装

## 高级用法

### 多实例运行
```bash
# 使用不同配置运行多个实例
starfire --config ~/.starfire/config1.yaml --daemon &
starfire --config ~/.starfire/config2.yaml --daemon &
```

### 系统服务
可以将 StarFire 设置为系统服务自动启动（需要创建 launchd plist 文件）。

### 性能监控
```bash
# 监控资源使用
top -p $(pgrep starfire)

# 网络连接状态  
netstat -an | grep starfire
```

## 技术支持

- 查看详细文档: `BUILD.md`
- 命令行帮助: `starfire --help`
- 项目主页: https://github.com/your-repo/star-fire

---

**注意**: StarFire 是一个基于算力共享的客户端应用，运行时会将您的计算资源贡献给网络中的其他用户。请确保您了解并同意这种使用方式。
