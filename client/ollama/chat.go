// 使用 Ollama 的 Chat 接口
package ollama

import (
	"context"
	"fmt"
	"log"
	"star-fire/public"

	"github.com/ollama/ollama/api"
)

func (oc *ChatClient) Chat() (err error) {
	// Ensure ConvertOllamaResponseToOpenAIResponse accepts *api.ChatResponse and returns the correct type
	respFunc := func(resp api.ChatResponse) error {
		fmt.Println("resp:...........", resp.Message.Content, "oc=", oc)
		if resp.Done {
			openAIResp, err := ConvertOllamaResponseToOpenAIResponse(&resp)
			if err != nil {
				return err
			}
			log.Println("ollama chat response:", resp)
			err = oc.ResponseConn.WriteJSON(public.WSMessage{
				Type:    public.MESSAGE,
				Content: openAIResp,
			})
			if err != nil {
				log.Fatalf("通过 WebSocket 发送消息失败: %v", err)
				fmt.Println("发送关闭链接消息")
				err = oc.ResponseConn.WriteJSON(public.WSMessage{
					Type:    public.CLOSE,
					Content: nil,
				})
				fmt.Println("发送关闭链接消息完成")
				return err
			}
			fmt.Println("发送关闭链接消息")
			return oc.ResponseConn.WriteJSON(public.WSMessage{
				Type:    public.CLOSE,
				Content: nil,
			})
		}
		return nil
	}
	ctx := context.Background()
	err = oc.ReqClient.Chat(ctx, oc.ChatReq.Request, respFunc)
	if err != nil {
		log.Println("ollama chat error:", err)
		return err
	}
	return nil
}

/*//打开新的连接发送数据，不阻塞接收消息的通道
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:8080", Path: fmt.Sprintf("/response/%s", fingerPrint)}
	log.Printf("connecting to %s", u.String())
	respConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer respConn.Close()
	messes := make([]public.OpenAIMessage, len(messages))
	for i, msg := range messages {
		messes[i] = public.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	//result, resMessages, err := kimi_chat(messes, "sk-Ka8PYTujkEimddSx3MqhY2XNAmAUh6ACItXbd5s24Sz3bh4O", "https://api.moonshot.cn/v1")
	result, resMessages, err := SimulateOllamaChat(messes, "d73f81ca4b9ce65411eb39e84772ff3e", "https://api.moonshot.cn/v1", stream, conn)
	if err != nil {
		fmt.Println("kimi_chat error:", err)
		// 处理错误
		err = respConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     err.Error(),
			FingerPrint: fingerPrint,
		})
		return
	}
	fmt.Println("result:", result, "messages:", resMessages)
	var resp = api.ChatResponse{
		Model:     model,
		CreatedAt: time.Time{},
		Message: api.Message{
			Role:      "assistant",
			Content:   result,
			Images:    nil,
			ToolCalls: nil,
		},
		DoneReason: "",
		Done:       false,
		Metrics:    api.Metrics{},
	}
	err = respConn.WriteJSON(public.WSMessage{
		Type:        public.MESSAGE,
		Content:     resp,
		FingerPrint: fingerPrint,
	})
	fmt.Println("write json:", err)
	return
}

type ChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []public.OpenAIMessage `json:"messages"`
	Temperature float64                `json:"temperature"`
}

type ChatResponse struct {
	Choices []struct {
		Message public.OpenAIMessage `json:"message"`
	} `json:"choices"`
}

func SimulateOllamaChat(messages []public.OpenAIMessage, apiKey, baseURL string, stream bool, conn *websocket.Conn) (string, []public.OpenAIMessage, error) {
	// Create chat request payload
	chatReq := ChatRequest{
		Model:       "deepseek-r1-1.5B",
		Messages:    messages,
		Temperature: 0.3,
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return "", messages, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send POST request to Moonshot API
	//https://hcc-subcenter2.tianhe-tech.com/maas/service/c96a3a6b8cb7/v1/completions
	if stream {
		reqBody = append(reqBody, []byte(`,"stream":true}`)...)
	} else {
		reqBody = append(reqBody, []byte(`}`)...)
	}
	fmt.Println("request:", string(reqBody))

	if stream {
		// 流式获取聊天响应
		openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
		for i, msg := range messages {
			openaiMessages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
		config := openai.DefaultConfig(apiKey)
		config.BaseURL = baseURL
		client := openai.NewClientWithConfig(config)
		streamResp, err := client.CreateChatCompletionStream(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    openai.GPT3Dot5Turbo,
				Messages: openaiMessages,
				Stream:   true,
			},
		)
		if err != nil {
			log.Fatalf("创建流式聊天失败: %v", err)
		}
		defer streamResp.Close()

		// 处理流式响应并通过 WebSocket 发送
		for {
			response, err := streamResp.Recv()
			if err != nil {
				log.Fatalf("接收流式响应失败: %v", err)
			}
			for _, choice := range response.Choices {
				message := choice.Delta.Content
				if message != "" {
					err = conn.WriteJSON(public.WSMessage{
						Type:    public.MESSAGE_STREAM,
						Content: message,
					})
					if err != nil {
						log.Fatalf("通过 WebSocket 发送消息失败: %v", err)
					}
				}
			}
			break
		}
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://hcc-subcenter2.tianhe-tech.com/maas/service/c96a3a6b8cb7/v1/completions"), bytes.NewBuffer(reqBody))
	if err != nil {
		return "", messages, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", messages, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	// Read and parse response
	respBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println("response:", string(respBody))
	if err != nil {
		return "", messages, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", messages, fmt.Errorf("API error: %s", string(respBody))
	}

	var chatResp ChatResponse
	err = json.Unmarshal(respBody, &chatResp)
	if err != nil {
		return "", messages, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Extract assistant's response
	result := chatResp.Choices[0].Message.Content

	// Append assistant's response to history
	messages = append(messages, public.OpenAIMessage{
		Role:    "assistant",
		Content: result,
	})

	return result, messages, nil
}
*/
