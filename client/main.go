// client端 使用ollama命令行运行一个指定的模型，然后启动一个websocket客户端，注册到服务端
// 注册成功后一直保持连接，等待服务端的消息，并将服务端的对话内容通过ollama的http服务发送给模型，将模型的返回结果通过websocket发送给服务端
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"star-fire/client/ollama"
	sfopenai "star-fire/client/openai"
	"star-fire/public"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
)

type Client struct {
	ID           string `json:"id"`
	Ollama       *ollama.Ollama
	Openai       *sfopenai.Openai
	ControlConn  *websocket.Conn
	StarFireHost string
	Models       []*public.Model `json:"models"`
	Ctx          context.Context
}

func (c *Client) GenerateClientID() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
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

func (c *Client) ScanModels() {
	if c.Models == nil {
		c.Models = make([]*public.Model, 0)
	}
	log.Println("scan models:....", c.Openai)
	if c.Ollama != nil {
		for _, cl := range c.Ollama.Clients {
			if cl.Type == ollama.ENV_CLIENT {
				tmpClient, err := api.ClientFromEnvironment()
				if err != nil {
					log.Println("ollama client from environment error:", err)
					continue
				}
				resp, err := tmpClient.ListRunning(c.Ctx)
				if err == nil {
					log.Println("ollama list models:", err)
					for _, model := range resp.Models {
						c.Models = append(c.Models, &public.Model{
							Name: model.Name,
							Type: model.Digest,
							Size: strings.Split(model.Name, ":")[1],
							Arch: model.Details.QuantizationLevel,
						})
					}
				}
			}
		}
		if c.Openai != nil {
			log.Println("openai models:", c.Openai.Clients)
			for _, cl := range c.Openai.Clients {
				for _, model := range cl.Models {
					c.Models = append(c.Models, &public.Model{
						Name: model.ID,
						Type: model.Object,
						Size: model.Root,
						Arch: model.Parent,
					})
				}
			}
		}
	}
}

func (c *Client) init() {
	log.Println("init client")
	ctx := context.Background()
	c.Ctx = ctx
	c.Ollama = &ollama.Ollama{}
	c.Ollama.Init()
	c.Openai = &sfopenai.Openai{}
	c.Openai.Init()
	c.ScanModels()
}

func (c *Client) RegisterClient() {
	var err error
	err = c.GenerateClientID()
	c.init()
	if err != nil {
		log.Fatal("generate client id:", err)
	}
	u := url.URL{Scheme: "ws", Host: c.StarFireHost, Path: fmt.Sprintf("/register/%s", c.ID)}
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	c.ControlConn = conn

	RegisterInfo := public.WSMessage{
		Type:    public.REGISTER,
		Content: c,
	}
	err = conn.WriteJSON(RegisterInfo)
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func Chat(c Client, Fingerprint string, request *openai.ChatCompletionRequest) error {
	if c.Ollama == nil && c.Openai == nil {
		log.Println("no local model found")
		return fmt.Errorf("no local model found")
	}
	// 首选ollama
	if c.Ollama != nil {
		picked := false
		for _, cl := range c.Ollama.Clients {
			if cl.Type != ollama.ENV_CLIENT {
				continue
			}
			tmpClient, err := api.ClientFromEnvironment()
			if err != nil {
				log.Println("ollama client from environment error:", err)
				continue
			}
			cl.ChatClient = &ollama.ChatClient{
				ReqClient:    tmpClient,
				ResponseConn: nil,
			}
			resp, err := cl.ChatClient.ReqClient.ListRunning(c.Ctx)
			if err != nil {
				log.Println("list models:", err)
			}
			if resp != nil {
				for _, model := range resp.Models {
					if model.Name == request.Model {
						picked = true
						break
					}
				}
			}
		}
		if picked {
			log.Println("ollama chat.................")
			ollamaReq, err := ollama.ConvertOpenAIToOllamaRequest(request)
			if err != nil {
				log.Println("convert request error:", err)
				return err
			}
			ollamaClient := &ollama.ChatClient{
				ChatReq: &ollama.ChatRequest{
					FingerPrint: Fingerprint,
					Request:     ollamaReq,
				},
				ReqClient:    c.Ollama.Clients[0].ChatClient.ReqClient,
				ResponseConn: nil,
			}
			ollamaClient.ResponseConn, err = c.OpenResponseConn(Fingerprint)
			if err != nil {
				log.Println("open response conn:", err)
				return err
			}
			err = ollamaClient.Chat()
			if err != nil {
				log.Println("ollama chat error:", err)
				return err
			}
			log.Println("ollama chat success")
			return nil
		} else {
			log.Println("no ollama model found")
		}
	}
	if c.Openai != nil {
		log.Println("openai chat.................")
		var err error
		var openaiClient *sfopenai.Client
		for _, cl := range c.Openai.Clients {
			picked := false
			for _, model := range cl.Models {
				if model.ID == request.Model {
					picked = true
					openaiClient = cl
					break
				}
			}
			if picked {
				break
			}
		}
		if openaiClient == nil {
			log.Println("no openai client found")
			return fmt.Errorf("no openai client found")
		}
		config := openai.DefaultConfig(openaiClient.ApiKey)
		config.BaseURL = openaiClient.BaseURL
		client := openai.NewClientWithConfig(config)
		chatClient := &sfopenai.ChatClient{
			ChatReq: &sfopenai.ChatRequest{
				FingerPrint: Fingerprint,
				Request:     request,
			},
			ReqClient:    client,
			ResponseConn: nil,
		}
		chatClient.ResponseConn, err = c.OpenResponseConn(Fingerprint)
		if err != nil {
			log.Println("open response conn:", err)
			return err
		}
		err = chatClient.Chat()
		if err != nil {
			log.Println("openai chat error:", err)
			return err
		}
	}
	return nil
}

func (c *Client) OpenResponseConn(FingerPrint string) (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: c.StarFireHost, Path: fmt.Sprintf("/response/%s", FingerPrint)}
	log.Printf("connecting to %s", u.String())
	respConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("dial:", err)
		return nil, err
	}
	return respConn, nil
}

func (c *Client) Serving() {
	done := make(chan struct{})
	defer close(done)
	for {
		var message public.WSMessage
		err := c.ControlConn.ReadJSON(&message)
		if err != nil {
			log.Println("read:", err)
			return
		}
		if message.Type == public.KEEPALIVE {
			log.Printf("keepalive: %s", message.Content)
			pong := public.PPMessage{
				Type:      public.PONG,
				Timestamp: message.Content.(map[string]interface{})["timestamp"].(string),
			}
			err = c.ControlConn.WriteJSON(public.WSMessage{
				Type:    public.KEEPALIVE,
				Content: pong,
			})
			if err != nil {
				log.Println("write:", err)
				return
			}
		} else if message.Type == public.MESSAGE {
			// 用户端只接收openai 格式的chat request
			log.Printf("chat message: %s", message.Content)
			tmp, _ := json.Marshal(message.Content)
			var openaiReq openai.ChatCompletionRequest
			err = json.Unmarshal(tmp, &openaiReq)
			if err != nil {
				log.Println("Error converting messages:", err)
				return
			}
			log.Println("openaiReq:", openaiReq)
			go func(fingerPrint string, openaiReq *openai.ChatCompletionRequest) {
				err := Chat(*c, fingerPrint, openaiReq)
				if err != nil {
					log.Println("chat error:", err)
				}
			}(message.FingerPrint, &openaiReq)
		} else if message.Type == public.CLOSE {
			log.Println("server closed the connection")
			return
		}
	}
}

func main() {
	model := ollama.Model{Name: "qwen3:0.6b"}
	err := model.Run()
	if err != nil {
		panic(err)
	}
	client := &Client{
		StarFireHost: "localhost:8080",
	}
	client.RegisterClient()
	client.Serving()
}
