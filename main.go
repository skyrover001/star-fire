// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server东侧接受client的注册，西侧接受用户端的大模型请求，通过分配算法将这些请求转发到client端。
// client注册和问答都是通过客户端websocket的方式，不需要西侧client用户提供任何互联网入口。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	configs "star-fire/config"
	"star-fire/internal/models"
	"star-fire/routes"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 命令行模式：配置注册赠送余额（不启动 HTTP 服务）
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "set-bonus":
			// 用法: starfire set-bonus <金额>
			if len(os.Args) < 3 {
				log.Fatal("用法: starfire set-bonus <金额>，例如 starfire set-bonus 10")
			}
			amount, err := strconv.ParseFloat(os.Args[2], 64)
			if err != nil || amount < 0 {
				log.Fatal("金额必须是大于等于 0 的数字")
			}
			server := models.NewServer()
			if err := server.SystemConfigDB.Set(models.ConfigKeyRegisterBonus,
				strconv.FormatFloat(amount, 'f', -1, 64)); err != nil {
				log.Fatalf("设置失败: %v", err)
			}
			log.Printf("✓ 已设置注册赠送余额为 %.2f 元", amount)
			return
		case "get-bonus":
			server := models.NewServer()
			bonus := server.SystemConfigDB.GetFloat(models.ConfigKeyRegisterBonus, 0)
			log.Printf("当前注册赠送余额: %.2f 元", bonus)
			return
		}
	}

	server := models.NewServer()
	r := gin.Default()
	routes.SetupRoutes(r, server)

	srv := &http.Server{
		Addr:    configs.Config.ServerPort,
		Handler: r,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Println("Starting server on", configs.Config.ServerPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Error while starting server:", err)
	}
}
