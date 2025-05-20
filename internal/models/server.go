package models

import (
	"github.com/gorilla/websocket"
	"star-fire/pkg/public"
	"time"
)

type Server struct {
	Clients     map[*public.Model]*Client
	Port        string
	RespClients map[string]*websocket.Conn
	UserStore   *UserStore
	APIKeyStore *APIKeyStore
	TokenStore  *TokenStore
}

func NewServer() *Server {
	server := &Server{
		Clients:     make(map[*public.Model]*Client),
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

// model load balance
func (s *Server) LoadBalance(model string) *Client {
	// simple load balance
	for k, v := range s.Clients {
		// TODO: check model
		if k.Name == model && v.Status == "online" && v.Latency < public.MAXLATENCE && v.ControlConn != nil {
			return v
		}
	}
	return nil
}

// register model on reflection
func (s *Server) RegisterModel(model *public.Model, client *Client) {
	s.Clients[model] = client
}

// get all available models
func (s *Server) GetAllModels() []*public.Model {
	var models []*public.Model
	for m, client := range s.Clients {
		if client.Status == "online" && client.ControlConn != nil {
			models = append(models, m)
		}
	}
	return models
}
