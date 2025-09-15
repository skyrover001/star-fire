package models

import (
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"star-fire/pkg/public"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"gorm.io/gorm"
)

// InferenceEngine represents the type of inference engine used by the client.
type InferenceEngine struct {
	Name        string `json:"name" gorm:"default:ollama"` // e.g. "ollama", "vllm", "openai"
	MaxTokens   int    `json:"max_tokens"`
	NumParallel int    `json:"num_parallel"`
}

type Client struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	IP           string    `json:"ip"`
	Token        string    `json:"token"`
	ModelsJSON   string    `json:"-" gorm:"column:models"`
	Status       string    `json:"status"`
	RegisterTime time.Time `json:"register_time"`
	Latency      int       `json:"latency"`
	UserID       string    `json:"user_id" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// not in db
	Models          []*public.Model          `json:"models" gorm:"-"`
	EmbeddingModels []*openai.EmbeddingModel `json:"embedding_models" gorm:"-"`
	ControlConn     *websocket.Conn          `json:"-" gorm:"-"`
	MessageChan     chan *api.ChatResponse   `json:"-" gorm:"-"`
	PongChan        chan *public.PPMessage   `json:"-" gorm:"-"`
	ErrChan         chan error               `json:"-" gorm:"-"`
	User            *User                    `json:"user" gorm:"-"`
	InferenceEngine InferenceEngine          `json:"inference_engine" gorm:"-"`
}

type ConnectionResult struct {
	ClientID string `json:"client_id"`
	Count    int    `json:"count"`
}

func (c *Client) BeforeSave(tx *gorm.DB) error {
	if c.Models != nil {
		data, err := json.Marshal(c.Models)
		if err != nil {
			return err
		}
		c.ModelsJSON = string(data)
	}
	return nil
}

func (c *Client) AfterFind(tx *gorm.DB) error {
	if c.ModelsJSON != "" {
		var models []*public.Model
		if err := json.Unmarshal([]byte(c.ModelsJSON), &models); err != nil {
			return err
		}
		c.Models = models
	}
	return nil
}

type ClientDB struct {
	db *gorm.DB
}

func NewClientDB(db *gorm.DB) *ClientDB {
	if err := db.AutoMigrate(&Client{}); err != nil {
		log.Fatalf("迁移Client表失败: %v", err)
	}
	return &ClientDB{db: db}
}

func (cdb *ClientDB) GetClient(id string) (*Client, error) {
	var client Client
	result := cdb.db.Where("id = ?", id).First(&client)
	return &client, result.Error
}

func (cdb *ClientDB) SaveClient(client *Client) error {
	return cdb.db.Save(client).Error
}

func (cdb *ClientDB) UpdateStatus(id, status string) error {
	return cdb.db.Model(&Client{}).Where("id = ?", id).Update("status", status).Error
}

func (cdb *ClientDB) GetClientsByUser(userID string) ([]*Client, error) {
	var clients []*Client
	result := cdb.db.Where("user_id = ?", userID).Find(&clients)
	return clients, result.Error
}

func (cdb *ClientDB) GetActiveClients() ([]*Client, error) {
	var clients []*Client
	result := cdb.db.Where("status = ?", "connected").Find(&clients)
	return clients, result.Error
}

func (cdb *ClientDB) GetClientsByUserID(userID string) ([]*Client, error) {
	var clients []*Client
	result := cdb.db.Where("user_id = ?", userID).Find(&clients)
	if result.Error != nil {
		return nil, result.Error
	}
	for _, client := range clients {
		if err := client.AfterFind(cdb.db); err != nil {
			log.Printf("after find client error: %v", err)
		}
	}
	return clients, nil
}

func NewClient(id, ip string, conn *websocket.Conn) *Client {
	return &Client{
		ID:           id,
		IP:           ip,
		ControlConn:  conn,
		Status:       "connecting",
		RegisterTime: time.Now(),
		Latency:      public.MAXLATENCE,
		PongChan:     make(chan *public.PPMessage),
		MessageChan:  make(chan *api.ChatResponse),
		ErrChan:      make(chan error),
	}
}

func (c *Client) SetUser(user *User) {
	c.User = user
	c.UserID = user.ID
}

type ClientFingerprint struct {
	Fingerprint string `json:"fingerprint" gorm:"primaryKey"`
	ClientID    string `json:"client_id" gorm:"index"`
	Status      string `json:"status"` // e.g. "preparing", "transmitting", "completed"
}

type ClientFingerprintDB struct {
	db *gorm.DB
}

func NewClientFingerprintDB(db *gorm.DB) *ClientFingerprintDB {
	if err := db.AutoMigrate(&ClientFingerprint{}); err != nil {
		log.Fatalf("migrate client fingerprint table: %v", err)
	}
	return &ClientFingerprintDB{db: db}
}

func (cfdb *ClientFingerprintDB) SaveFingerprint(fingerprint, clientID, status string) error {
	cf := &ClientFingerprint{
		Fingerprint: fingerprint,
		ClientID:    clientID,
		Status:      status,
	}
	return cfdb.db.Save(cf).Error
}

func (cfdb *ClientFingerprintDB) UpdateFingerprint(fingerprint, clientID, status string) error {
	cf := &ClientFingerprint{
		Fingerprint: fingerprint,
		ClientID:    clientID,
		Status:      status,
	}
	result := cfdb.db.Where("fingerprint = ?", fingerprint).FirstOrCreate(cf)
	if result.Error != nil {
		return result.Error
	}
	return cfdb.db.Model(cf).Update("status", status).Error
}
func (cfdb *ClientFingerprintDB) GetClientChatConnections(clientIDs []string) ([]*ConnectionResult, error) {
	// 如果没有可用的客户端，返回错误
	if len(clientIDs) == 0 {
		return nil, fmt.Errorf("没有可用的客户端")
	}

	var results []*ConnectionResult
	// 查询每个客户端状态为"transmitting"的连接数
	err := cfdb.db.Model(&ClientFingerprint{}).
		Select("client_id, count(*) as count").
		Where("client_id IN ? AND status = ?", clientIDs, "transmitting").
		Group("client_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}
func (cfdb *ClientFingerprintDB) GetClientID(fingerprint string) (string, error) {
	var cf ClientFingerprint
	result := cfdb.db.Where("fingerprint = ?", fingerprint).First(&cf)
	if result.Error != nil {
		return "", result.Error
	}
	return cf.ClientID, nil
}

func (cfdb *ClientFingerprintDB) DeleteFingerprint(fingerprint string) error {
	return cfdb.db.Where("fingerprint = ?", fingerprint).Delete(&ClientFingerprint{}).Error
}
