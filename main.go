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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
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
