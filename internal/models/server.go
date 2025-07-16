package models

import (
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"math/rand"
	"os"
	configs "star-fire/config"
	"star-fire/pkg/public"
	"sync"
	"time"
)

type MailService struct {
	SMTPServer   string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromAddress  string
}

type Server struct {
	clientsMu sync.RWMutex
	Clients   map[string]map[string]*Client

	clientRBMu            sync.RWMutex
	clientRoundRobinIndex map[string]int // for round-robin load balancing

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
	TrendDB             *TrendDB

	LoadBalanceAlgorithm string // Load balancing algorithm, e.g., "round-robin", "random", etc.

	MailService *MailService // optional, for sending emails
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
	trendDB := NewTrendDB(gormDB)

	// 初始化默认用户
	err = userDB.InitDefaultUsers()
	if err != nil {
		log.Printf("init default user failed: %v", err)
	}

	server := &Server{
		Clients:               make(map[string]map[string]*Client),
		Port:                  configs.Config.ServerPort,
		RespClients:           make(map[string]*websocket.Conn),
		clientRoundRobinIndex: make(map[string]int),

		DB:                   gormDB,
		APIKeyDB:             apiKeyDB,
		UserDB:               userDB,
		TokenUsageDB:         tokenUsageDB,
		RegisterTokenStore:   NewRegisterTokenStore(),
		ClientDB:             clientDB,
		ClientFingerprintDB:  clientFingerprintDB,
		TrendDB:              trendDB,
		LoadBalanceAlgorithm: configs.Config.LBA, // default load balancing algorithm
		MailService: &MailService{
			SMTPServer:   configs.Config.EmailHost,
			SMTPPort:     configs.Config.EmailPort,
			SMTPUsername: configs.Config.EmailUser,
			SMTPPassword: configs.Config.EmailPassword,
			FromAddress:  configs.Config.EmailFrom,
		},
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
	clientsCopy := make(map[string]map[string]*Client)
	for modelName, clients := range s.Clients {
		clientsCopy[modelName] = make(map[string]*Client)
		for clientID, client := range clients {
			clientsCopy[modelName][clientID] = client
		}
	}
	s.clientsMu.RUnlock()
	var toRemove []struct{ model, client string }
	for modelName, clients := range clientsCopy {
		if modelName == model {
			log.Println("found model:", modelName, "clients:", len(clients))
			for clientID, client := range clients {
				if client.Models == nil {
					log.Println("client models is nil, skip client:", client.ID)
					continue
				}
				existModel := false
				for _, m := range client.Models {
					if m.Name == modelName && client.Status == "online" && client.ControlConn != nil && client.Latency < public.MAXLATENCE {
						log.Println("found online client for model:", modelName, "client:", client.ID)
						existModel = true
					}
				}
				if !existModel {
					toRemove = append(toRemove, struct{ model, client string }{modelName, clientID})
				}
			}
		}
	}

	for _, item := range toRemove {
		s.RemoveClient(item.model, item.client)
	}

	switch s.LoadBalanceAlgorithm {
	case "round-robin":
		// round-robin load balancing
		s.clientRBMu.Lock()
		defer s.clientRBMu.Unlock()
		if len(s.Clients[model]) == 0 {
			return nil
		}
		if _, exists := s.clientRoundRobinIndex[model]; !exists {
			s.clientRoundRobinIndex[model] = 0
		}
		index := s.clientRoundRobinIndex[model]
		clients := make([]*Client, 0, len(s.Clients[model]))
		for _, client := range s.Clients[model] {
			clients = append(clients, client)
		}
		if index >= len(clients) {
			index = 0
		}
		s.clientRoundRobinIndex[model] = index + 1
		return clients[index]
	case "random":
		// randomly select a client for the model
		if len(s.Clients[model]) == 0 {
			return nil
		}
		keys := make([]string, 0, len(s.Clients[model]))
		for k := range s.Clients[model] {
			keys = append(keys, k)
		}
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(keys))
		randomKey := keys[randomIndex]
		client := s.Clients[model][randomKey]
		return client
	case "min-conn":
		// the client which have max idle connections
		// step 1: get all client ids for the model
		clientIDs := make([]string, 0, len(s.Clients[model]))
		for k := range s.Clients[model] {
			clientIDs = append(clientIDs, k)
		}
		// step 2: get all client chat connections count
		chatConnections, err := s.ClientFingerprintDB.GetClientChatConnections(clientIDs)
		if err != nil {
			log.Println("get client chat connections error:", err)
			return nil
		}
		maxIdleConnectionsCounts := make(map[string]int)
		clientID := ""
		MaxIdleConnectionsCount := 65535 // default max idle connections count
		for _, result := range chatConnections {
			if s.Clients[model][result.ClientID].Status == "online" && s.Clients[model][result.ClientID].ControlConn != nil {
				if s.Clients[model][result.ClientID].InferenceEngine.Name == "ollama" && s.Clients[model][result.ClientID].InferenceEngine.NumParallel > 0 {
					maxIdleConnectionsCounts[result.ClientID] = s.Clients[model][result.ClientID].InferenceEngine.NumParallel - result.Count
				} else {
					maxIdleConnectionsCounts[result.ClientID] = 1 - result.Count
				}
			} else {
				maxIdleConnectionsCounts[result.ClientID] = 0
			}
			if maxIdleConnectionsCounts[result.ClientID] < MaxIdleConnectionsCount {
				MaxIdleConnectionsCount = maxIdleConnectionsCounts[result.ClientID]
				clientID = result.ClientID
			}
		}

		if clientID != "" {
			if c, exists := s.Clients[model][clientID]; exists {
				log.Println("found client:", c.ID, "for model:", model)
				return c
			} else {
				log.Println("client:", clientID, "not found for model:", model)
				return nil
			}
		}
		return nil
	}
	log.Println("unknown load balance algorithm:", s.LoadBalanceAlgorithm)
	return nil
}

func (s *Server) RegisterModel(model *public.Model, client *Client) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if _, exists := s.Clients[model.Name]; !exists {
		s.Clients[model.Name] = make(map[string]*Client)
	}
	if _, exists := s.Clients[model.Name][client.ID]; !exists {
		s.Clients[model.Name][client.ID] = client
		log.Println("register model:", model.Name, "for client:", client.ID)
	} else {
		log.Println("model:", model.Name, "already registered for client:", client.ID)
	}
}

// for model marketplace
func (s *Server) GetAllModels() []*MarketplaceModel {
	s.clientsMu.RLock()
	clientsCopy := make(map[string]map[string]*Client)
	for modelName, clients := range s.Clients {
		clientsCopy[modelName] = make(map[string]*Client)
		for clientID, client := range clients {
			clientsCopy[modelName][clientID] = client
		}
	}
	s.clientsMu.RUnlock()

	var marketplaceModels []*MarketplaceModel
	var toRemove []struct{ model, client string }
	for modelName, clientMaps := range s.Clients {
		if len(clientMaps) == 0 {
			continue
		}

		model := &MarketplaceModel{
			Name:         modelName,
			Type:         "model",
			Size:         "unknown",
			ClientModels: make([]*ClientModel, 0, len(clientMaps)),
		}

		for clientID, client := range clientMaps {
			existModel := false
			for _, m := range client.Models {
				if m.Name == modelName && client.Status == "online" && client.ControlConn != nil && client.Latency < public.MAXLATENCE {
					existModel = true
					model.Size = m.Size
					model.Type = m.Type
					model.Quantization = m.Arch
					model.ClientModels = append(model.ClientModels, &ClientModel{
						Client: client,
						Model:  m,
					})
				}
			}
			if !existModel {
				toRemove = append(toRemove, struct{ model, client string }{modelName, clientID})
			}
		}
		marketplaceModels = append(marketplaceModels, model)
	}
	for _, item := range toRemove {
		s.RemoveClient(item.model, item.client)
	}
	return marketplaceModels
}

// for openAI api compatibility
func (s *Server) GetModels() map[string]interface{} {
	s.clientsMu.RLock()
	clientsCopy := make(map[string]map[string]*Client)
	for modelName, clients := range s.Clients {
		clientsCopy[modelName] = make(map[string]*Client)
		for clientID, client := range clients {
			clientsCopy[modelName][clientID] = client
		}
	}
	s.clientsMu.RUnlock()

	var models []*public.Model
	var toRemove []struct{ model, client string }
	for modelName, clientMaps := range clientsCopy {
		for clientID, client := range clientMaps {
			existModel := false
			for _, m := range client.Models {
				if m.Name == modelName && client.Status == "online" && client.ControlConn != nil && client.Latency < public.MAXLATENCE {
					existModel = true
					models = append(models, &public.Model{
						Name: m.Name,
						Type: m.Type,
						Size: m.Size,
						Arch: m.Arch,
					})
					break
				}
			}
			if !existModel {
				toRemove = append(toRemove, struct{ model, client string }{modelName, clientID})
			}
		}
	}

	for _, item := range toRemove {
		s.RemoveClient(item.model, item.client)
	}

	modelMap := make(map[string]*public.Model)
	for _, model := range models {
		modelMap[model.Name] = model
	}
	resultModels := make([]*openai.Model, 0, len(modelMap))
	for _, model := range modelMap {
		resultModels = append(resultModels, &openai.Model{
			ID:        model.Name,
			Object:    "model",
			OwnedBy:   "star-fire",
			CreatedAt: time.Now().Unix(),
			Root:      "",
			Permission: []openai.Permission{
				{
					ID:                 model.Name + "-permission",
					Object:             "permission",
					AllowCreateEngine:  true,
					AllowSampling:      true,
					AllowLogprobs:      true,
					AllowSearchIndices: false,
					AllowView:          true,
				},
			},
		})
	}
	result := make(map[string]interface{})
	result["data"] = resultModels
	result["object"] = "list"
	return result
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

func (s *Server) RemoveClient(modelName string, clientID string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if clients, exists := s.Clients[modelName]; exists {
		delete(clients, clientID)
	}
}

func (s *Server) GetTrends(startDate, endDate string) []*Trend {
	if startDate == "" || endDate == "" {
		// use today 00:00:00 as start date today 23:59:59 as end date
		startDate = time.Now().Format("2006-01-02")
		endDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}
	trends, err := s.TrendDB.GetTrendsByTimeRange(startDate, endDate)
	if err != nil {
		log.Printf("get trends failed: %v", err)
		return nil
	}
	return trends
}
