// internal/client/client.go
package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"star-fire/client/internal/config"
	"star-fire/client/internal/inference"
	"star-fire/client/internal/inference/ollama"
	"star-fire/client/internal/inference/openai"
	"star-fire/pkg/public"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID           string `json:"id"`
	engines      []inference.Engine
	controlConn  *websocket.Conn
	starFireHost string
	joinToken    string
	Models       []*public.Model `json:"models"`
	ctx          context.Context
	cancel       context.CancelFunc
	cfg          *config.Config
}

func NewClient(cfg *config.Config) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		starFireHost: cfg.StarFireHost,
		joinToken:    cfg.JoinToken,
		ctx:          ctx,
		cancel:       cancel,
		engines:      []inference.Engine{},
		Models:       []*public.Model{},
		cfg:          cfg,
	}
	if err := client.generateID(); err != nil {
		return nil, fmt.Errorf("generate id error: %w", err)
	}
	if err := client.initializeEngines(cfg); err != nil {
		return nil, fmt.Errorf("init engine error: %w", err)
	}
	if err := client.refreshModels(); err != nil {
		log.Printf("alert: refresh models error: %v", err)
	}
	return client, nil
}

func (c *Client) generateID() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		c.ID = uuid.NewString()
		return nil
	}
	for _, iface := range interfaces {
		mac := iface.HardwareAddr.String()
		if mac != "" {
			hash := sha256.Sum256([]byte(mac))
			c.ID = fmt.Sprintf("%x", hash)
			return nil
		}
	}
	c.ID = uuid.NewString()
	return nil
}

func (c *Client) initializeEngines(cfg *config.Config) error {
	switch cfg.LocalInferenceType {
	case "ollama":
		ollamaEngine, err := ollama.NewEngine(c.ctx, cfg.OllamaHost, cfg)
		if err != nil {
			return fmt.Errorf("init Ollama engine error: %w", err)
		}
		c.engines = append(c.engines, ollamaEngine)

	case "openai":
		if cfg.OpenAIKey == "" {
			return fmt.Errorf("not set OpenAIKey")
		}
		openaiEngine, err := openai.NewEngine(c.ctx, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg)
		if err != nil {
			return fmt.Errorf("init openai engine error: %w", err)
		}
		c.engines = append(c.engines, openaiEngine)

	case "all":
		ollamaEngine, err := ollama.NewEngine(c.ctx, cfg.OllamaHost, cfg)
		if err != nil {
			log.Printf("init ollama engine error: %v", err)
		} else {
			c.engines = append(c.engines, ollamaEngine)
		}

		if cfg.OpenAIKey != "" {
			openaiEngine, err := openai.NewEngine(c.ctx, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg)
			if err != nil {
				log.Printf("init openai engine error: %v", err)
			} else {
				c.engines = append(c.engines, openaiEngine)
			}
		}

	default:
		return fmt.Errorf("no avaliable engine : %s", cfg.LocalInferenceType)
	}

	if len(c.engines) == 0 {
		return fmt.Errorf("no engine found")
	}

	return nil
}

func (c *Client) refreshModels() error {
	c.Models = make([]*public.Model, 0)

	log.Println("c.engines:", c.engines, len(c.engines))
	for _, engine := range c.engines {
		models, err := engine.ListModels(c.ctx, c.cfg)
		if err != nil {
			log.Printf("get models from %s error: %v", engine.Name(), err)
			continue
		}

		c.Models = append(c.Models, models...)
	}

	log.Printf("discovery %d models", len(c.Models), c.Models)
	return nil
}

func (c *Client) findEngineForModel(modelName string) (inference.Engine, error) {
	for _, engine := range c.engines {
		if engine.SupportsModel(modelName, c.cfg) {
			return engine, nil
		}
	}

	return nil, fmt.Errorf("no enging support model: %s", modelName)
}

func (c *Client) Close() {
	c.cancel()
	if c.controlConn != nil {
		_ = c.controlConn.Close()
	}
}
