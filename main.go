// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server东侧接受client的注册，西侧接受用户端的大模型请求，通过分配算法将这些请求转发到client端。
// client注册和问答都是通过客户端websocket的方式，不需要西侧client用户提供任何互联网入口。
package main

import (
	"log"
	"star-fire/config"
	"star-fire/internal/models"
	"star-fire/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	server := models.NewServer()
	r := gin.Default()
	routes.SetupRoutes(r, server)

	// starting the server
	log.Println("Starting server on", configs.Config.ServerPort)
	err := r.Run(configs.Config.ServerPort)
	if err != nil {
		log.Fatal("Error while starting server:", err)
	}
}
