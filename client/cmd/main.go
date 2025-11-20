package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"star-fire/client/internal/client"
	configs "star-fire/client/internal/config"
	"syscall"
	"time"
)

func main() {
	// 首先创建一个临时日志文件用于调试
	if isChild() {
		debugLogFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(debugLogFile)
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.Println("=== 后台进程启动 ===")
		}
	}

	cfg := configs.LoadConfig()

	if isChild() {
		log.Println("配置加载完成")
	}

	// 验证配置参数
	if err := configs.ValidateConfig(cfg); err != nil {
		if isChild() {
			log.Printf("配置验证失败: %v", err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "配置错误: %v\n\n", err)
		_, _ = fmt.Fprintf(os.Stderr, "使用 -h 或 --help 查看帮助信息\n")
		os.Exit(1)
	}

	if isChild() {
		log.Println("配置验证通过")
	}

	// 检查是否有 -daemon 参数
	isDaemon := false
	isChildProcess := isChild()
	for _, arg := range os.Args[1:] {
		if arg == "-daemon" || arg == "--daemon" {
			isDaemon = true
		}
	}

	// 如果需要后台运行且不是子进程，则启动后台进程
	if isDaemon && !isChildProcess {
		startDaemon()
		return
	}

	// 设置日志
	if isDaemon || isChildProcess {
		if err := setupLogging(); err != nil {
			log.Fatalf("设置日志失败: %v", err)
		}
		log.Println("程序已启动为后台进程")
	}

	// 运行客户端主逻辑
	runClient(cfg)
}

// isChild 检查是否为子进程
func isChild() bool {
	for _, arg := range os.Args[1:] {
		if arg == "-child" {
			return true
		}
	}
	return false
}

func runClient(cfg *configs.Config) {
	// 创建退出信号通道
	quit := make(chan struct{})
	defer close(quit)

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("收到退出信号，正在关闭服务...")
		close(quit)
	}()

	for {
		select {
		case <-quit:
			log.Println("程序正常退出")
			return
		default:
			c, err := client.NewClient(cfg)
			if err != nil {
				log.Printf("创建客户端失败: %v，5秒后重试", err)
				select {
				case <-time.After(5 * time.Second):
				case <-quit:
					log.Println("程序退出")
					return
				}
				continue
			}

			if err := client.RegisterClient(cfg, c, cfg.StarFireHost, cfg.JoinToken); err != nil {
				log.Printf("注册客户端失败: %v，5秒后重试", err)
				c.Close()
				select {
				case <-time.After(5 * time.Second):
				case <-quit:
					log.Println("程序退出")
					return
				}
				continue
			}

			log.Printf("客户端已成功连接到 %s", cfg.StarFireHost)

			// 启动消息处理
			done := make(chan bool)
			go func() {
				client.HandleMessages(c)
				done <- true
			}()

			select {
			case <-quit:
				log.Println("正在关闭连接...")
				c.Close()
				return
			case <-done:
				log.Println("连接断开，正在重连...")
				c.Close()
				select {
				case <-time.After(3 * time.Second):
				case <-quit:
					log.Println("程序退出")
					return
				}
			}
		}
	}
}

// startDaemon 启动后台进程
func startDaemon() {
	// 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("获取可执行文件路径失败: %v", err)
	}

	// 构建新的命令行参数，添加 -child 标志
	args := []string{"-child"}
	for _, arg := range os.Args[1:] {
		if arg != "-daemon" && arg != "--daemon" {
			args = append(args, arg)
		}
	}

	// 根据操作系统选择不同的启动方式
	cmd := exec.Command(execPath, args...)

	if runtime.GOOS == "windows" {
		// Windows下重定向到nul
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		// Unix/Linux 下重定向标准输入输出到 /dev/null
		devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if err != nil {
			log.Fatalf("无法打开 /dev/null: %v", err)
		}
		defer func() {
			if closeErr := devNull.Close(); closeErr != nil {
				log.Printf("关闭 /dev/null 失败: %v", closeErr)
			}
		}()
		cmd.Stdin = devNull
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	// 启动后台进程
	if err := cmd.Start(); err != nil {
		log.Fatalf("启动后台进程失败: %v", err)
	}

	fmt.Printf("客户端已启动为后台进程，PID: %d\n", cmd.Process.Pid)
	fmt.Printf("日志文件位置: logs/starfire-client.log\n")

	// 分离进程
	if err := cmd.Process.Release(); err != nil {
		log.Printf("警告: 释放进程失败: %v", err)
	}
}

// setupLogging 设置日志输出到文件
func setupLogging() error {
	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// 在可执行文件同目录创建日志目录
	logDir := filepath.Join(execDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件
	logFile := filepath.Join(logDir, "starfire-client.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}

	// 设置日志输出到文件
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return nil
}
