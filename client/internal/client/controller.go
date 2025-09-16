package client

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"net/url"
	configs "star-fire/client/internal/config"
	"star-fire/pkg/public"
)

func RegisterClient(conf *configs.Config, c *Client, host, token string) error {
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("/register/%s", c.ID)}
	log.Printf("link %s", u.String())

	requestHeader := http.Header{}
	requestHeader.Set("X-Registration-Token", token)

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), requestHeader)
	if err != nil {
		return fmt.Errorf("WebSocket connet error: %w", err)
	}

	c.controlConn = conn
	_ = c.refreshModels()

	testInfo, _ := json.Marshal(c)
	log.Println("register info:", string(testInfo), "c.models:", c.Models)
	registerInfo := public.WSMessage{
		Type:    public.REGISTER,
		Content: c,
	}
	if err := conn.WriteJSON(registerInfo); err != nil {
		_ = conn.Close()
		return fmt.Errorf("send register info error: %w", err)
	}

	log.Println("client register success...", registerInfo.Content)
	return nil
}

func HandleMessages(c *Client) {
	defer c.controlConn.Close()

	// 创建消息通道
	messageCh := make(chan public.WSMessage, 1)
	errorCh := make(chan error, 1)

	// 启动消息读取 goroutine
	go func() {
		for {
			var message public.WSMessage
			err := c.controlConn.ReadJSON(&message)
			if err != nil {
				errorCh <- err
				return
			}
			messageCh <- message
		}
	}()

	// 主循环
	for {
		select {
		case <-c.ctx.Done():
			log.Println("收到取消信号，停止消息处理")
			return
		case err := <-errorCh:
			log.Printf("WebSocket read error: %v", err)
			return
		case message := <-messageCh:
			switch message.Type {
			case public.KEEPALIVE:
				handleKeepAlive(c, message)
			case public.MESSAGE:
				handleChatMessage(c, message)
			case public.EMBEDDING_REQUEST:
				handleEmbeddingMessage(c, message)
			case public.RECONNECT:
				handleReconnect(c, message)
			case public.CLOSE:
				log.Println("server close message:", message.Content)
				return
			default:
				log.Printf("not defined message type: %v", message.Type)
			}
		}
	}
}

func handleKeepAlive(c *Client, message public.WSMessage) {
	log.Printf("recieve pong")
	_ = c.refreshModels()
	pong := public.PPMessage{
		Type:            public.PONG,
		Timestamp:       message.Content.(map[string]interface{})["timestamp"].(string),
		AvailableModels: c.Models,
	}
	response := public.WSMessage{
		Type:    public.KEEPALIVE,
		Content: pong,
	}

	if err := c.controlConn.WriteJSON(response); err != nil {
		log.Printf("send pong error: %v", err)
	}
	fmt.Println("pong message is:", response)
}

func handleReconnect(c *Client, message public.WSMessage) {
	// update fingerprint
	log.Printf("recieve reconnect message: %v", message.FingerPrint)
	c.cfg.JoinToken = message.FingerPrint
}

func handleChatMessage(c *Client, message public.WSMessage) {
	log.Printf("recieve chat message request: %v", message.FingerPrint)

	tmp, _ := json.Marshal(message.Content)
	var openaiReq openai.ChatCompletionRequest
	if err := json.Unmarshal(tmp, &openaiReq); err != nil {
		log.Printf("parse message error: %v", err)
		return
	}

	go func() {
		engine, err := c.findEngineForModel(openaiReq.Model)
		if err != nil {
			log.Printf("not found support model %s engine: %v", openaiReq.Model, err)
			return
		}

		responseConn, err := openResponseConn(c.starFireHost, message.FingerPrint)
		if err != nil {
			log.Printf("open response connection error: %v", err)
			return
		}
		defer responseConn.Close()
		if err := engine.HandleChat(c.ctx, message.FingerPrint, &openaiReq, responseConn); err != nil {
			log.Printf("handle chat message error: %v", err)
		}
	}()
}

func handleEmbeddingMessage(c *Client, message public.WSMessage) {
	log.Printf("recieve embedding request: %v", message.FingerPrint)

	tmp, _ := json.Marshal(message.Content)
	var openaiReq openai.EmbeddingRequest
	if err := json.Unmarshal(tmp, &openaiReq); err != nil {
		log.Printf("parse embedding request error: %v", err)
		return
	}

	go func() {
		engine, err := c.findEngineForModel(string(openaiReq.Model))
		if err != nil {
			log.Printf("not found support model %s engine: %v", openaiReq.Model, err)
			return
		}

		responseConn, err := openResponseConn(c.starFireHost, message.FingerPrint)
		if err != nil {
			log.Printf("open response connection error: %v", err)
			return
		}
		defer responseConn.Close()
		if err := engine.HandleEmbedding(c.ctx, message.FingerPrint, &openaiReq, responseConn); err != nil {
			log.Printf("handle embedding request error: %v", err)
		}
	}()
}

func openResponseConn(host, fingerprint string) (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("/response/%s", fingerprint)}
	log.Printf("open response connection: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("websocket connect error: %w", err)
	}

	return conn, nil
}
