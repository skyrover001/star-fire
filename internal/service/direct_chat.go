package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	configs "star-fire/config"
	"star-fire/internal/models"
	"star-fire/internal/service/format"
	"star-fire/pkg/public"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// handleDirectChat 通过 HTTP 直连固定后端处理 chat completions。
// 返回值语义：
//
//	done=true  已向调用方写响应（成功或 4xx 直接回传），调用方 return；
//	done=false 后端失败（5xx/超时/断流且未写出任何字节），调用方记失败后继续重试循环。
func handleDirectChat(c *gin.Context, server *models.Server, b *models.DirectBackend,
	extendedRequest public.ExtendedChatRequest, userIDStr string, userConverters ...format.Converter) (done bool) {

	var userConv format.Converter
	if len(userConverters) > 0 {
		userConv = userConverters[0]
	}

	// 进入即计数，defer 递减
	b.IncrActive()
	defer b.DecrActive()

	// 请求体：优先用 RawBody（视频等多模态透传），否则序列化扩展请求
	var reqBody []byte
	if len(extendedRequest.RawBody) > 0 {
		reqBody = extendedRequest.RawBody
	} else {
		var err error
		reqBody, err = extendedRequest.BuildRequestBody()
		if err != nil {
			log.Printf("direct %s: marshal request error: %v", b.ID, err)
			b.IncrFailures()
			// 本地序列化错误，非后端过错，不采样可靠性
			return false
		}
	}

	// 流式不能设 Client.Timeout（会截断 SSE），用 context 超时控制整体时长
	timeout := time.Duration(configs.Config.ChatMaxTime) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	url := strings.TrimRight(b.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("direct %s: create request error: %v", b.ID, err)
		b.IncrFailures()
		// 本地构造请求错误，非后端过错，不采样可靠性
		return false
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if b.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.APIKey)
	}
	if extendedRequest.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	client := &http.Client{} // Timeout=0，靠 context 控制
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("direct %s: request error: %v", b.ID, err)
		b.IncrFailures()
		b.UpdateReliability(0)
		return false
	}
	defer resp.Body.Close()

	// 4xx：请求本身错误，原样回传，不重试、不记失败
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		log.Printf("direct %s: 4xx status=%d body=%s", b.ID, resp.StatusCode, string(body))
		return true
	}

	// 5xx：后端失败，记失败 + 冷却，返回 done=false 让调用方重试
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("direct %s: 5xx status=%d body=%s", b.ID, resp.StatusCode, string(body))
		b.IncrFailures()
		b.UpdateReliability(0)
		return false
	}

	// 非 2xx 兜底（如 3xx 重定向未跟随）视为失败
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("direct %s: unexpected status=%d", b.ID, resp.StatusCode)
		b.IncrFailures()
		b.UpdateReliability(0)
		return false
	}

	// 成功路径
	if extendedRequest.Stream {
		return handleDirectStream(c, server, b, resp, extendedRequest.Model, userIDStr, userConv)
	}
	return handleDirectNonStream(c, server, b, resp, extendedRequest.Model, userIDStr, userConv)
}

// handleDirectNonStream 非流式：读全量 body → 解析 usage → 原样写给用户 → 计费。
func handleDirectNonStream(c *gin.Context, server *models.Server, b *models.DirectBackend,
	resp *http.Response, model, userIDStr string, userConv format.Converter) bool {

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("direct %s: read body error: %v", b.ID, err)
		b.IncrFailures()
		b.UpdateReliability(0)
		return false
	}

	// 解析 usage 用于计费
	var chatResp openai.ChatCompletionResponse
	_ = json.Unmarshal(body, &chatResp)

	if userConv == nil {
		c.Data(http.StatusOK, "application/json", body)
	} else {
		upstreamConv, convErr := format.GetConverter(public.FormatOpenAI)
		if convErr != nil {
			log.Printf("direct %s: get OpenAI converter error: %v", b.ID, convErr)
			b.IncrFailures()
			// 我方适配器错误，非后端过错，不采样可靠性
			return false
		}
		canonicalResp, convErr := upstreamConv.ParseUpstreamResponse(body)
		if convErr != nil {
			log.Printf("direct %s: parse OpenAI response error: %v", b.ID, convErr)
			b.IncrFailures()
			// 我方适配器错误，非后端过错，不采样可靠性
			return false
		}
		userBody, convErr := userConv.BuildResponse(canonicalResp)
		if convErr != nil {
			log.Printf("direct %s: build user response error: %v", b.ID, convErr)
			b.IncrFailures()
			// 我方适配器错误，非后端过错，不采样可靠性
			return false
		}
		c.Data(http.StatusOK, "application/json", userBody)
	}

	// 计费
	recordDirectUsage(c, server, b, model, userIDStr, &chatResp.Usage)

	b.ResetFailures()
	b.UpdateReliability(1)
	return true
}

// handleDirectStream 流式：逐行读 SSE → 原样写 + Flush；解析 usage 尾块计费。
// 若已写出至少一个 chunk 后断流：终止流（写 [DONE]），done=true（不可重试）。
func handleDirectStream(c *gin.Context, server *models.Server, b *models.DirectBackend,
	resp *http.Response, model, userIDStr string, userConv format.Converter) bool {

	reader := bufio.NewReader(resp.Body)
	var usage *openai.Usage
	wroteAny := false
	finishReason := ""
	var upstreamConv format.Converter
	var userWriter format.UserStreamWriter
	if userConv != nil {
		var err error
		upstreamConv, err = format.GetConverter(public.FormatOpenAI)
		if err != nil {
			log.Printf("direct %s: get OpenAI converter error: %v", b.ID, err)
			b.IncrFailures()
			// 我方适配器错误，非后端过错，不采样可靠性
			return false
		}
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		userWriter = newUserStreamWriter(userConv, model)
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			// 断流
			log.Printf("direct %s: stream read error: %v", b.ID, err)
			if wroteAny {
				// 已写出内容，不可重试：终止流
				_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
				c.Writer.Flush()
				b.IncrFailures()
				b.UpdateReliability(0)
				return true
			}
			b.IncrFailures()
			b.UpdateReliability(0)
			return false
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))

		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}
		wroteAny = true
		if userConv == nil {
			if _, werr := c.Writer.Write(append(append([]byte("data: "), data...), '\n', '\n')); werr != nil {
				log.Printf("direct %s: write to user error: %v", b.ID, werr)
				b.IncrFailures()
				// 用户侧断连，非后端过错，不采样可靠性
				return true
			}
			c.Writer.Flush()
		} else {
			ev, convErr := upstreamConv.ParseUpstreamStreamEvent(data)
			if convErr != nil {
				log.Printf("direct %s: parse OpenAI stream event error: %v", b.ID, convErr)
				b.IncrFailures()
				// 我方适配器错误，非后端过错，不采样可靠性
				return true
			}
			if ev != nil && ev.Type == public.StreamEventDone {
				finishReason = ev.FinishReason
				if ev.Usage != nil {
					usage = &openai.Usage{
						PromptTokens:     ev.Usage.InputTokens,
						CompletionTokens: ev.Usage.OutputTokens,
						TotalTokens:      ev.Usage.TotalTokens,
					}
				}
				continue
			}
			// writeUserStreamEvent 返回 true 表示适配层/用户侧错误（非后端过错），
			// 有意不采样可靠性、不 ResetFailures；后端过错由上游读取错误路径处理。
			if writeUserStreamEvent(c, server, "", "", 0, 0, 0, model, ev, userConv, nil, userWriter) {
				return true
			}
		}

		// 解析 usage（尾块）
		var chunk openai.ChatCompletionStreamResponse
		if err := json.Unmarshal(data, &chunk); err == nil && chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	if userConv != nil {
		done := &public.CanonicalStreamEvent{Type: public.StreamEventDone, FinishReason: finishReason}
		if usage != nil {
			done.Usage = &public.CanonicalUsage{InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens, TotalTokens: usage.TotalTokens}
		}
		writeUserStreamEvent(c, server, "", "", 0, 0, 0, model, done, userConv, nil, userWriter)
	} else {
		_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
		c.Writer.Flush()
	}

	// 计费
	recordDirectUsage(c, server, b, model, userIDStr, usage)
	b.ResetFailures()
	b.UpdateReliability(1)
	return true
}

// recordDirectUsage 计费：与 WS 路径 recordTokenUsage 一致，clientID 用 "direct:"+ID。
func recordDirectUsage(c *gin.Context, server *models.Server, b *models.DirectBackend,
	model, userIDStr string, usage *openai.Usage) {

	if usage == nil {
		log.Printf("direct %s: no usage in response, skip billing", b.ID)
		return
	}
	ippm, oppm, cippm, ok := b.PriceFor(model)
	if !ok {
		log.Printf("direct %s: no price for model %s, skip billing", b.ID, model)
		return
	}
	cached := 0
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens > 0 {
		cached = usage.PromptTokensDetails.CachedTokens
	}
	requestID := uuid.NewString()
	recordTokenUsage(c, server, requestID, model,
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, cached,
		"direct:"+b.ID, ippm, oppm, cippm)
}
