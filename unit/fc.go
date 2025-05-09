package main

import (
	"context"
	"encoding/json"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
)

// 定义天气查询函数的参数结构
type GetWeatherParams struct {
	Location string `json:"location"`
	Unit     string `json:"unit,omitempty"`
}

// 模拟天气查询函数
func getWeather(params GetWeatherParams) string {
	// 在真实应用中，这里会调用天气API
	unit := params.Unit
	if unit == "" {
		unit = "celsius"
	}
	return fmt.Sprintf("天气在 %s 现在是晴朗，温度是 22 %s", params.Location, unit)
}

func main() {
	// 从环境变量获取API密钥
	apiKey := "d73f81ca4b9ce65411eb39e84772ff3e"
	baseUrl := "https://hcc-subcenter2.tianhe-tech.com/maas/service/c96a3a6b8cb7/v1/completions"

	// 创建OpenAI客户端
	client := openai.NewClient(apiKey)
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseUrl
	client = openai.NewClientWithConfig(config)

	ctx := context.Background()

	// 定义可用的函数
	functions := []openai.FunctionDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定位置的当前天气",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "城市名称，如'北京'、'上海'等",
					},
					"unit": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"celsius", "fahrenheit"},
						"description": "温度单位",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	tools := []openai.Tool{}
	for _, f := range functions {
		tools = append(tools, openai.Tool{Type: openai.ToolTypeFunction, Function: &f})
	}

	// 创建聊天请求
	req := openai.ChatCompletionRequest{
		Model: "deepseek-r1-1.5B",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "北京今天的天气怎么样？",
			},
		},
		Tools: tools,
	}

	// 发送请求给OpenAI API
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletion错误: %v\n", err)
		return
	}

	// 处理响应
	message := resp.Choices[0].Message

	// 检查是否有函数调用
	if message.FunctionCall != nil {
		funcName := message.FunctionCall.Name
		fmt.Printf("调用函数: %s\n", funcName)

		// 解析函数参数
		if funcName == "get_weather" {
			var params GetWeatherParams
			err := json.Unmarshal([]byte(message.FunctionCall.Arguments), &params)
			if err != nil {
				fmt.Printf("参数解析错误: %v\n", err)
				return
			}

			// 执行函数并获取结果
			result := getWeather(params)
			fmt.Printf("函数结果: %s\n", result)

			// 将函数结果发送回模型，继续对话
			req.Messages = append(req.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: "",
				FunctionCall: &openai.FunctionCall{
					Name:      funcName,
					Arguments: message.FunctionCall.Arguments,
				},
			})
			fmt.Println("函数调用参数:", message.FunctionCall.Arguments)

			req.Messages = append(req.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleFunction,
				Name:    funcName,
				Content: result,
			})

			// 再次请求GPT获取最终回复
			followUpResp, err := client.CreateChatCompletion(ctx, req)
			if err != nil {
				fmt.Printf("后续ChatCompletion错误: %v\n", err)
				return
			}

			finalResponse := followUpResp.Choices[0].Message.Content
			fmt.Printf("AI最终回复: %s\n", finalResponse)
		} else {
			fmt.Printf("未知函数: %s\n", funcName)
		}
	} else {
		// 模型直接回复，没有调用函数
		fmt.Printf("AI回复: %s\n", message.Content)
	}
}
