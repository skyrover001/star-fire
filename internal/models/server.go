package models

import (
	"github.com/gorilla/websocket"
	"log"
	"star-fire/pkg/public"
	"sync"
	"time"
)

type Server struct {
	clientsMu sync.RWMutex // 保护Clients映射
	Clients   map[string]*Client

	respClientsMu sync.RWMutex // 保护RespClients映射
	RespClients   map[string]*websocket.Conn

	Port string

	UserStore   *UserStore
	APIKeyStore *APIKeyStore
	TokenStore  *TokenStore
}

func NewServer() *Server {
	server := &Server{
		Clients:     make(map[string]*Client),
		Port:        ":8080",
		RespClients: make(map[string]*websocket.Conn),
		UserStore:   NewUserStore(),
		APIKeyStore: NewAPIKeyStore(),
		TokenStore:  NewTokenStore(),
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			server.TokenStore.CleanupExpiredTokens()
		}
	}()
	return server
}

func (s *Server) LoadBalance(model string) *Client {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	log.Println("查找模型:", model)
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
