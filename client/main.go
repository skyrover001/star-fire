// client端 使用ollama命令行运行一个指定的模型，然后启动一个websocket客户端，注册到服务端
// 注册成功后一直保持连接，等待服务端的消息，并将服务端的对话内容通过ollama的http服务发送给模型，将模型的返回结果通过websocket发送给服务端
package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"star-fire/star-fire-main/client/ollama"
	"star-fire/star-fire-main/public"
)

// 定义一个client结构体，包含ollama信息和websocket连接
type Client struct {
	ID     string `json:"id"` // client的id
	Model  string `json:"model"`
	Ollama *ollama.Ollama
	Conn   *websocket.Conn
}

func (c *Client) RegisterClient() {
	c.ID = "client1" // 这里可以使用uuid生成一个唯一的id
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:8080", Path: fmt.Sprintf("/register/%s", c.ID)}
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	c.Conn = conn
	c.Ollama = &ollama.Ollama{}
	c.Model = "qwen3:1.5b" // 这里可以使用命令行参数传入模型名称

	// 发送注册信息
	RegisterInfo := public.WSMessage{
		Type:    public.REGISTER,
		Content: c,
	}
	err = conn.WriteJSON(RegisterInfo)
	if err != nil {
		log.Println("write:", err)
		return
	}
	c.Conn = conn
}

func (c *Client) Serving() {
	done := make(chan struct{})
	defer close(done)
	for {
		var message public.WSMessage
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			log.Println("read:", err)
			return
		}
		if message.Type == "keepalive" {
			log.Printf("keepalive: %s", message.Content)
			pong := public.PPMessage{
				Type:      public.PONG,
				Timestamp: message.Content.(map[string]interface{})["timestamp"].(string),
			}
			err = c.Conn.WriteJSON(public.WSMessage{
				Type:    public.KEEPALIVE,
				Content: pong,
			})
			if err != nil {
				log.Println("write:", err)
				return
			}
		} else if message.Type == "message" {
			log.Printf("chat message: %s", message.Content)
			messages, err := ollama.ConvertMessages(message.Content.(map[string]interface{}))
			resp, err := c.Ollama.Chat(messages, c.Model)
			if err != nil {
				log.Println("error:", err)
				return
			}
			err = c.Conn.WriteJSON(public.WSMessage{
				Type:    public.MESSAGE,
				Content: resp,
			})
			if err != nil {
				log.Println("write:", err)
				return
			}
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
	client := &Client{}
	client.RegisterClient()
	client.Serving()
}
