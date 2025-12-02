package client

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"log"
	"net"
	"net/http"
	"net/url"
	configs "star-fire/client/internal/config"
	"star-fire/pkg/public"
	"time"
)

func RegisterClient(conf *configs.Config, c *Client, host, token string) error {
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("/register/%s", c.ID)}
	log.Printf("link %s", u.String())

	requestHeader := http.Header{}
	requestHeader.Set("X-Registration-Token", token)

	conn, res, err := websocket.DefaultDialer.Dial(u.String(), requestHeader)
	if res != nil && res.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("bad status: %v", res.Status)
	}
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
			case public.INCOME:
				handleIncome(message)
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
		if err = engine.HandleChat(c.ctx, message.FingerPrint, &openaiReq, responseConn); err != nil {
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

func handleIncome(message public.WSMessage) {
	content, ok := message.Content.(map[string]interface{})
	if !ok {
		log.Printf("invalid income message content format")
		return
	}

	// 提取收益信息
	log.Printf("收到收益消息: %+v", content)

	// 提取各个字段
	usage, hasUsage := content["usage"].(map[string]interface{})
	income, hasIncome := content["income"]

	if !hasUsage || !hasIncome {
		log.Printf("income message missing required fields (usage: %v, income: %v)", hasUsage, hasIncome)
		return
	}

	// 将 income 转换为 float64
	var incomeValue float64
	switch v := income.(type) {
	case float64:
		incomeValue = v
	case float32:
		incomeValue = float64(v)
	case int:
		incomeValue = float64(v)
	case int64:
		incomeValue = float64(v)
	default:
		log.Printf("income value has unexpected type: %T", income)
		return
	}

	// 提取可选字段
	model := content["model"]
	totalIncome := content["total_income"]
	timestamp := content["timestamp"]

	// 构造 JSON 消息（发送给 Python 桌面应用）
	incomeData := map[string]interface{}{
		"type":     "income",
		"amount":   fmt.Sprintf("%.8f", incomeValue),
		"currency": "¥",
		"usage":    usage,
		"model":    model,
	}

	// 添加可选字段
	if totalIncome != nil {
		incomeData["total_income"] = totalIncome
	}
	if timestamp != nil {
		incomeData["timestamp"] = timestamp
	}

	jsonBytes, err := json.Marshal(incomeData)
	if err != nil {
		log.Printf("marshal income message error: %v", err)
		return
	}

	// 发送到 TCP 服务器
	if err := sendToTCPServer("127.0.0.1:19527", string(jsonBytes)); err != nil {
		log.Printf("发送收益到 TCP 服务器失败: %v", err)
	} else {
		log.Printf("✓ 收益已发送: %.8f ¥ (模型: %v, 总收益: %v)", incomeValue, model, totalIncome)
	}
}

// sendToTCPServer 发送消息到 TCP 服务器（协议: 4字节长度头 + UTF-8内容）
func sendToTCPServer(address string, message string) error {
	// 连接到 TCP 服务器
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect to TCP server error: %w", err)
	}
	defer conn.Close()

	// 编码消息为 UTF-8
	messageBytes := []byte(message)
	length := uint32(len(messageBytes))

	// 发送长度头（4字节，网络字节序）
	lengthBytes := make([]byte, 4)
	lengthBytes[0] = byte(length >> 24)
	lengthBytes[1] = byte(length >> 16)
	lengthBytes[2] = byte(length >> 8)
	lengthBytes[3] = byte(length)

	if _, err := conn.Write(lengthBytes); err != nil {
		return fmt.Errorf("write length header error: %w", err)
	}

	// 发送消息内容
	if _, err := conn.Write(messageBytes); err != nil {
		return fmt.Errorf("write message content error: %w", err)
	}

	return nil
}
