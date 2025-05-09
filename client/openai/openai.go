package openai

import (
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	FingerPrint string
	Request     *openai.ChatCompletionRequest
}

type Openai struct {
	Clients []*Client
}

type ChatClient struct {
	ChatReq      *ChatRequest
	ReqClient    *openai.Client
	ResponseConn *websocket.Conn
}

type Client struct {
	Models     []openai.Model `json:"models"`
	BaseURL    string         `json:"base_url"`
	ApiKey     string         `json:"apiKey"`
	ChatClient *ChatClient
}

func (oa *Openai) Init() {
	oa.Clients = make([]*Client, 0)
	m := openai.Model{ID: "moonshot-v1-8k"}
	c := &Client{
		Models:  []openai.Model{m},
		BaseURL: "https://api.moonshot.cn/v1",
		ApiKey:  "sk-mOpHCEZ6vFp462K9txFwQgBylTfQcDsKAM5EV574iqJnuWLS",
	}
	oa.Clients = append(oa.Clients, c)
	m = openai.Model{ID: "deepseek-r1-1.5B"}
	c = &Client{
		Models:  []openai.Model{m},
		BaseURL: "https://hcc-subcenter2.tianhe-tech.com/maas/service/c96a3a6b8cb7/v1/completions",
		ApiKey:  "d73f81ca4b9ce65411eb39e84772ff3e",
	}
	oa.Clients = append(oa.Clients, c)
}
