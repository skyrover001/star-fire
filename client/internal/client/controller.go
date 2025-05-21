package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

func RegisterClient(c *Client, host, token string) error {
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
		conn.Close()
		return fmt.Errorf("send register info error: %w", err)
	}

	log.Println("client register success...", registerInfo.Content)
	return nil
}

func HandleMessages(c *Client) {
	defer c.controlConn.Close()
	for {
		var message public.WSMessage
		err := c.controlConn.ReadJSON(&message)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		switch message.Type {
		case public.KEEPALIVE:
			handleKeepAlive(c, message)
		case public.MESSAGE:
			handleChatMessage(c, message)
		case public.CLOSE:
			log.Println("server close message:", message.Content)
			return
		default:
			log.Printf("not defined message type: %v", message.Type)
		}
	}
}

func handleKeepAlive(c *Client, message public.WSMessage) {
	log.Printf("recieve pong")
	_ = c.refreshModels()
	pong := public.PPMessage{
		Type:            public.PONG,
		Timestamp:       message.Content.(map[string]interface{})["timestamp"].(string),
		AvaliableModels: c.Models,
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

func openResponseConn(host, fingerprint string) (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("/response/%s", fingerprint)}
	log.Printf("open response connection: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("websocket connect error: %w", err)
	}

	return conn, nil
}
