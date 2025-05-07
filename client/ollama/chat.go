// 使用 Ollama 的 Chat 接口
package ollama

import (
	"bytes"
	"encoding/json"
	"star-fire/star-fire-main/public"
	"time"

	//"context"
	"fmt"
	"github.com/ollama/ollama/api"
	"io/ioutil"
	"net/http"
)

func (o *Ollama) Chat(messages []api.Message, model string) (*api.ChatResponse, error) {
	//var response api.ChatResponse
	//// 构建聊天请求
	//chatReq := &api.ChatRequest{
	//	Model:    model, // 替换为你要使用的模型名称
	//	Messages: messages,
	//	Stream:   nil, // 可以设置为true启用流式响应
	//	Options:  make(map[string]any),
	//}
	//
	//// 创建一个通道来接收响应
	//responseChan := make(chan api.ChatResponse)
	//var finalErr error
	//
	//// 发送聊天请求并处理响应
	//err := o.Clients[0].Chat(context.Background(), chatReq, func(resp api.ChatResponse) error {
	//	if resp.Done {
	//		response = resp
	//		close(responseChan)
	//	} else {
	//		responseChan <- resp
	//	}
	//	return nil
	//})
	//if err != nil {
	//	finalErr = err
	//	close(responseChan)
	//}
	//
	//// 从通道中获取最终的响应（如果有）
	//if finalErr == nil {
	//	for r := range responseChan {
	//		response = r
	//	}
	//}
	//
	//return &response, finalErr

	// 使用kimi的Chat接口进行模拟
	// 将 api.Message 转换为 Message
	messes := make([]public.OpenAIMessage, len(messages))
	for i, msg := range messages {
		messes[i] = public.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	result, resMessages, err := kimi_chat(messes, "sk-Ka8PYTujkEimddSx3MqhY2XNAmAUh6ACItXbd5s24Sz3bh4O", "https://api.moonshot.cn/v1")
	if err != nil {
		return nil, err
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
	return &resp, nil
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

func kimi_chat(messages []public.OpenAIMessage, apiKey, baseURL string) (string, []public.OpenAIMessage, error) {
	// Create chat request payload
	chatReq := ChatRequest{
		Model:       "moonshot-v1-8k",
		Messages:    messages,
		Temperature: 0.3,
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return "", messages, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send POST request to Moonshot API
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/chat/completions", baseURL), bytes.NewBuffer(reqBody))
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
