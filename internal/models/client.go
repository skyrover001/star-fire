package models

import (
	"star-fire/pkg/public"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
)

// Client
type Client struct {
	ID     string          `json:"id"`
	IP     string          `json:"ip"`
	Token  string          `json:"token"`
	Models []*public.Model `json:"models"`

	Status       string `json:"status"`
	RegisterTime string `json:"register_time"`
	Latency      int    `json:"latency"`

	ControlConn *websocket.Conn
	MessageChan chan *api.ChatResponse
	PongChan    chan *public.PPMessage
	ErrChan     chan error
	User        *User `json:"user"`
}

func NewClient(id, ip string, conn *websocket.Conn) *Client {
	return &Client{
		ID:           id,
		IP:           ip,
		ControlConn:  conn,
		Status:       "connecting",
		RegisterTime: time.Now().Format("2006-01-02 15:04:05"),
		Latency:      public.MAXLATENCE,
		PongChan:     make(chan *public.PPMessage),
		MessageChan:  make(chan *api.ChatResponse),
		ErrChan:      make(chan error),
	}
}

func (c *Client) SetUser(user *User) {
	c.User = user
}
