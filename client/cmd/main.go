package main

import (
	"log"
	"os"
	"os/signal"
	"star-fire/client/internal/client"
	configs "star-fire/client/internal/config"
	"syscall"
)

func main() {
	cfg := configs.LoadConfig()
	if cfg.StarFireHost == "" || cfg.JoinToken == "" {
		log.Println("error: not set StarFireHost or JoinToken")
		return
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("new client error: %v", err)
	}
	defer c.Close()

	if err := client.RegisterClient(c, cfg.StarFireHost, cfg.JoinToken); err != nil {
		log.Fatalf("registe client error: %v", err)
	}
	go client.HandleMessages(c)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("shut down service...")
}
