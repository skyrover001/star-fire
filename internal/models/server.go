package models

import (
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	configs "star-fire/config"
	"star-fire/pkg/public"
	"sync"
	"time"
)

type Server struct {
	clientsMu sync.RWMutex
	Clients   map[string]*Client

	respClientsMu sync.RWMutex
	RespClients   map[string]*websocket.Conn

	Port               string
	RegisterTokenStore *RegisterTokenStore

	DB                  *gorm.DB
	APIKeyDB            *APIKeyDB
	UserDB              *UserDB
	TokenUsageDB        *TokenUsageDB
	ClientDB            *ClientDB
	ClientFingerprintDB *ClientFingerprintDB
}

func NewServer() *Server {
	err := os.MkdirAll("./data", 0755)
	if err != nil {
		log.Fatalf("create data directory failed: %v", err)
	}

	gormDB, err := gorm.Open(sqlite.Open("./data/star_fire.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("init database failed: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	apiKeyDB := NewAPIKeyDB(gormDB)
	tokenUsageDB := NewTokenUsageDB(gormDB)
	userDB := NewUserDB(gormDB)
	clientDB := NewClientDB(gormDB)
	clientFingerprintDB := NewClientFingerprintDB(gormDB)

	// 初始化默认用户
	err = userDB.InitDefaultUsers()
	if err != nil {
		log.Printf("init default user failed: %v", err)
	}

	server := &Server{
		Clients:     make(map[string]*Client),
		Port:        configs.Config.ServerPort,
		RespClients: make(map[string]*websocket.Conn),

		DB:                  gormDB,
		APIKeyDB:            apiKeyDB,
		UserDB:              userDB,
		TokenUsageDB:        tokenUsageDB,
		RegisterTokenStore:  NewRegisterTokenStore(),
		ClientDB:            clientDB,
		ClientFingerprintDB: clientFingerprintDB,
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			server.RegisterTokenStore.CleanupExpiredTokens()
		}
	}()
	return server
}

func (s *Server) LoadBalance(model string) *Client {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	log.Println("search models:", model)
	for k := range s.Clients {
		if k == model {
			for _, c := range s.Clients {
				for _, m := range c.Models {
					if m.Name == model && c.Status == "online" && c.ControlConn != nil && c.Latency < public.MAXLATENCE {
						return c
					}
				}
			}
		}
	}
	return nil
}

func (s *Server) RegisterModel(model *public.Model, client *Client) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	s.Clients[model.Name] = client
	log.Printf("register models %s to client: %s", model.Name, client.ID)
}

func (s *Server) GetAllModels() []*public.Model {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	var models []*public.Model
	for _, client := range s.Clients {
		if client.Status == "online" && client.ControlConn != nil {
			models = append(models, client.Models...)
		}
	}

	modelMap := make(map[string]*public.Model)
	for _, model := range models {
		modelMap[model.Name] = model
	}
	models = make([]*public.Model, 0, len(modelMap))
	for _, model := range modelMap {
		models = append(models, model)
	}
	return models
}

func (s *Server) AddRespClient(id string, conn *websocket.Conn) {
	s.respClientsMu.Lock()
	defer s.respClientsMu.Unlock()

	s.RespClients[id] = conn
}

func (s *Server) GetRespClient(id string) (*websocket.Conn, bool) {
	s.respClientsMu.RLock()
	defer s.respClientsMu.RUnlock()

	conn, ok := s.RespClients[id]
	return conn, ok
}

func (s *Server) RemoveRespClient(id string) {
	s.respClientsMu.Lock()
	defer s.respClientsMu.Unlock()

	delete(s.RespClients, id)
}

func (s *Server) RemoveClient(modelName string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	delete(s.Clients, modelName)
}
