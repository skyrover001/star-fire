package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"star-fire/client/internal/client"
	configs "star-fire/client/internal/config"
	"syscall"
)

func main() {
	cfg := configs.LoadConfig()

	// 验证配置参数
	if err := configs.ValidateConfig(cfg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "配置错误: %v\n\n", err)
		_, _ = fmt.Fprintf(os.Stderr, "使用 -h 或 --help 查看帮助信息\n")
		os.Exit(1)
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer c.Close()

	if err := client.RegisterClient(cfg, c, cfg.StarFireHost, cfg.JoinToken); err != nil {
		log.Fatalf("注册客户端失败: %v", err)
	}

	log.Printf("客户端已启动，连接到 %s", cfg.StarFireHost)
	go client.HandleMessages(c)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("正在关闭服务...")
}
