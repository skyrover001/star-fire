package models

import (
	"encoding/json"
	"log"
	"star-fire/pkg/public"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"gorm.io/gorm"
)

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
	Models      []*public.Model        `json:"models" gorm:"-"`
	ControlConn *websocket.Conn        `json:"-" gorm:"-"`
	MessageChan chan *api.ChatResponse `json:"-" gorm:"-"`
	PongChan    chan *public.PPMessage `json:"-" gorm:"-"`
	ErrChan     chan error             `json:"-" gorm:"-"`
	User        *User                  `json:"user" gorm:"-"`
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
}

type ClientFingerprint struct {
	Fingerprint string `json:"fingerprint" gorm:"primaryKey"`
	ClientID    string `json:"client_id" gorm:"index"`
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

func (cfdb *ClientFingerprintDB) SaveFingerprint(fingerprint, clientID string) error {
	cf := &ClientFingerprint{
		Fingerprint: fingerprint,
		ClientID:    clientID,
	}
	return cfdb.db.Save(cf).Error
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
