package openai

import (
	"context"
	"fmt"
	"io"
	"log"
	"star-fire/pkg/public"
)

func (oc *ChatClient) Chat() error {
	if oc.ChatReq.Request.Stream {
		streamResp, err := oc.ReqClient.CreateChatCompletionStream(
			context.Background(),
			*oc.ChatReq.Request,
		)
		if err != nil {
			log.Println("创建流式聊天失败: %v", err)
			return err
		}
		defer streamResp.Close()
		// 处理流式响应并通过 WebSocket 发送
		for {
			response, err := streamResp.Recv()
			if err != nil {
				if err != io.EOF {
					fmt.Println("流式响应非正常结束:", err)
					break
				}
			} else {
				fmt.Println("流式响应:", response)
				err = oc.ResponseConn.WriteJSON(public.WSMessage{
					Type:    public.MESSAGE_STREAM,
					Content: response,
				})
				if err != nil {
					log.Println("通过 WebSocket 发送消息失败: %v", err)
					return err
				}
				if response.Choices[0].FinishReason == "stop" {
					fmt.Println("流式响应结束")
					break
				}
			}
		}
		fmt.Println("发送关闭链接消息")
		err = oc.ResponseConn.WriteJSON(public.WSMessage{
			Type:    public.CLOSE,
			Content: nil,
		})
		fmt.Println("发送关闭链接消息完成")
	} else {
		resp, err := oc.ReqClient.CreateChatCompletion(
			context.Background(),
			*oc.ChatReq.Request,
		)
		if err != nil {
			log.Println("创建聊天失败: %v", err)
			return err
		}
		fmt.Println("发送消息:", resp)
		err = oc.ResponseConn.WriteJSON(public.WSMessage{
			Type:    public.MESSAGE,
			Content: resp,
		})
		fmt.Println("发送关闭链接消息")
		err = oc.ResponseConn.WriteJSON(public.WSMessage{
			Type:    public.CLOSE,
			Content: nil,
		})
		fmt.Println("发送关闭链接消息完成")
		if err != nil {
			log.Println("通过 WebSocket 发送消息失败: %v", err)
			return err
		}
	}
	return nil
}
