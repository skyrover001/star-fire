package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	configs "star-fire/config"
	"star-fire/internal/models"
	"star-fire/internal/service/format"
	"star-fire/pkg/public"
)

// HandleMultiFormatChatRequest 是多格式 API 的统一入口。
// userFormat 是调用方使用的格式（openai | anthropic | responses）。
//
// 流程（对应 docs/multi-format-api-design.md 第四章）：
//  1. 按 userFormat 解析请求体 → CanonicalRequest
//  2. 复用现有鉴权/余额/限流（基于 canonical.Model）
//  3. 负载均衡选 client，读取 client 上游格式 UpstreamFormat
//  4. Canonical → 上游格式请求，发给 client（携带 Format）
//  5. 收到上游响应 → Canonical → 用户格式响应，返回给调用方
func HandleMultiFormatChatRequest(c *gin.Context, server *models.Server, userFormat string) {
	// 1. 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	// 2. 按用户格式解析 → Canonical
	userConv, err := format.GetConverter(userFormat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported format: %s", userFormat)})
		return
	}
	canonical, err := userConv.ParseRequest(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid %s request: %v", userFormat, err)})
		return
	}

	// ===== 链路日志：server 收到的原始请求体（用户格式）=====
	// log.Printf("[TRACE] server RAW %s request body=%s", userFormat, string(body))
	// ===== 链路日志：server 解析后的 Canonical 请求（含 tool_calls 的 id/name）=====
	// if canonicalBody, e := json.Marshal(canonical); e == nil {
	// 	log.Printf("[TRACE] server canonical request model=%s messages=%d body=%s", canonical.Model, len(canonical.Messages), string(canonicalBody))
	// }

	// 2.5 思考模型工具循环续接：注入上一轮保存的 reasoning_content，并记录会话键，
	// 以便本轮流式结束时保存新产生的 reasoning_content。
	convKey := extractConversationKey(canonical)
	if convKey != "" {
		c.Set("reasoning_conv_key", convKey)
		if reasoning := server.GetReasoning(convKey); len(reasoning) > 0 {
			format.InjectReasoningContent(canonical, reasoning)
		}
	}

	// 3. 鉴权/余额/限流（复用现有逻辑，基于 canonical.Model）
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	balance, _, _ := server.UserDB.GetBalance(userIDStr)
	if balance <= 0 {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": gin.H{
				"message": "You exceeded your current quota, please check your plan and billing details.",
				"type":    "insufficient_quota",
				"param":   nil,
				"code":    "insufficient_quota",
			},
		})
		return
	}

	// 限流
	if server.RateLimiter != nil && configs.Config.RateLimitEnabled {
		membership := server.UserDB.GetEffectiveMembership(userIDStr)
		cfg := rateLimitConfigFor(membership)
		estimatedTokens := estimateCanonicalTokens(canonical)
		limitKey := userIDStr
		if apiKeyID, ok := c.Get("api_key_id"); ok && apiKeyID.(string) != "" {
			limitKey = "key:" + apiKeyID.(string)
		} else {
			limitKey = "user:" + userIDStr
		}
		allowed, limitType := server.RateLimiter.Allow(limitKey, cfg, estimatedTokens)
		if !allowed {
			limitName := "requests"
			if limitType == "tpm" {
				limitName = "tokens"
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "You have exceeded your rate limit of " + limitName + " per minute.",
					"type":    "rate_limit_exceeded",
					"param":   nil,
					"code":    "rate_limit_exceeded",
				},
			})
			return
		}
	}

	// 4. 负载均衡选 client + 转发（含重试）
	handleMultiFormatWithRetry(c, server, canonical, userConv, userIDStr)
}

// handleMultiFormatWithRetry 多格式请求的重试转发。
// 与 handleChatWithRetry 结构一致，但基于 CanonicalRequest，且响应按用户格式回传。
func handleMultiFormatWithRetry(c *gin.Context, server *models.Server, canonical *public.CanonicalRequest, userConv format.Converter, userIDStr string) {
	failedClients := map[string]bool{}
	start := time.Now()

	// P1-M2: 多格式路径无 body 路由字段，从 header 解析路由偏好 + 容忍度。
	routing := resolveRouting(c, "")
	maxLatencyMs, minStability := resolveTolerance(c)

	// P2: 会话亲和 L0 短路（多格式）。优先用显式 prompt_cache_key/thread_id，否则回退内容指纹。
	var affinityKey string
	var affinityClient *models.Client
	var affinityHit bool
	if server.Affinity != nil {
		affinityKey = extractConversationKey(canonical)
		if affinityKey == "" {
			affinityKey = affinityKeyFromCanonical(canonical)
		}
		if affinityKey == "" {
			affinityKey = userIDStr + ":" + canonical.Model
		}
		if entry, ok := server.Affinity.Get(affinityKey); ok {
			if ac := server.ResolveAffinity(entry, canonical.Model, userIDStr); ac != nil {
				affinityClient = ac
			} else {
				server.Affinity.MarkLease(affinityKey)
			}
		}
	}

	for attempt := 0; attempt < public.MAX_CHAT_RETRY; attempt++ {
		if time.Since(start) > public.CHAT_RETRY_TOTAL_TIMEOUT*time.Second {
			break
		}

		// 0. Direct 主力池优先（M1 stability 路由，仅支持 openai 后端）：
		//    命中则把 canonical 转成 openai 上游请求体，经 RawBody 直连固定后端。
		//    cost 路由：PickCheapest 已并入 Direct，此处不再单独优先。
		if configs.Config.DirectBackendsEnabled && routing != RoutingCost {
			if b := server.PickDirect(canonical.Model, userIDStr, failedClients); b != nil {
				failedClients["direct:"+b.ID] = true
				// M1 只支持 openai 后端（非 openai 在 LoadDirectBackends 时已跳过）
				openaiConv, err := format.GetConverter(public.FormatOpenAI)
				if err == nil {
					if upstreamBody, berr := openaiConv.BuildUpstreamRequest(canonical); berr == nil {
						directReq := public.ExtendedChatRequest{}
						directReq.Model = canonical.Model
						directReq.Stream = canonical.Stream
						directReq.RawBody = json.RawMessage(upstreamBody)
						if handleDirectChat(c, server, b, directReq, userIDStr, userConv) {
							return
						}
						time.Sleep(backoff(attempt))
						continue
					}
				}
				// 转换失败：记失败并回退 crowdsource
				b.IncrFailures()
				time.Sleep(backoff(attempt))
				continue
			}
		}

		// 1. 选 client（排除已失败的）。attempt 0 且亲和命中 → 直接用粘住 client。
		var client *models.Client
		if attempt == 0 && affinityClient != nil {
			client = affinityClient
			affinityHit = true
		} else {
			switch routing {
			case RoutingCost:
				cc, cb := server.PickCheapest(canonical.Model, userIDStr, failedClients)
				if cb != nil {
					failedClients["direct:"+cb.ID] = true
					openaiConv, err := format.GetConverter(public.FormatOpenAI)
					if err == nil {
						if upstreamBody, berr := openaiConv.BuildUpstreamRequest(canonical); berr == nil {
							directReq := public.ExtendedChatRequest{}
							directReq.Model = canonical.Model
							directReq.Stream = canonical.Stream
							directReq.RawBody = json.RawMessage(upstreamBody)
							if handleDirectChat(c, server, cb, directReq, userIDStr, userConv) {
								return
							}
							time.Sleep(backoff(attempt))
							continue
						}
					}
					cb.IncrFailures()
					time.Sleep(backoff(attempt))
					continue
				}
				client = cc
			case RoutingBalanced:
				client = server.LoadBalanceBalanced(canonical.Model, userIDStr, failedClients, maxLatencyMs, minStability)
				if client == nil && configs.Config.DirectBackendsEnabled {
					if b := server.PickDirect(canonical.Model, userIDStr, failedClients); b != nil {
						failedClients["direct:"+b.ID] = true
						openaiConv, err := format.GetConverter(public.FormatOpenAI)
						if err == nil {
							if upstreamBody, berr := openaiConv.BuildUpstreamRequest(canonical); berr == nil {
								directReq := public.ExtendedChatRequest{}
								directReq.Model = canonical.Model
								directReq.Stream = canonical.Stream
								directReq.RawBody = json.RawMessage(upstreamBody)
								if handleDirectChat(c, server, b, directReq, userIDStr, userConv) {
									return
								}
								time.Sleep(backoff(attempt))
								continue
							}
						}
						b.IncrFailures()
						time.Sleep(backoff(attempt))
						continue
					}
				}
			default: // stability
				client = server.LoadBalanceWithTolerance(canonical.Model, userIDStr, failedClients, maxLatencyMs, minStability)
			}
		}
		if client == nil {
			break
		}
		failedClients[client.ID] = true

		// 2. 读取 client 上游格式
		upstreamFormat := clientUpstreamFormat(client, canonical.Model)
		if upstreamFormat == "" {
			upstreamFormat = public.FormatOpenAI
		}
		upstreamConv, err := format.GetConverter(upstreamFormat)
		if err != nil {
			log.Printf("unsupported upstream format %q: %v", upstreamFormat, err)
			continue
		}

		// 3. 从该 client 提取价格
		ippm, oppm, cippm := 9.0, 9.0, 0.0
		for _, m := range client.Models {
			if m.Name == canonical.Model {
				ippm, oppm, cippm = m.IPPM, m.OPPM, m.CIPPM
				break
			}
		}

		// 4. Canonical → 上游格式请求
		upstreamBody, err := upstreamConv.BuildUpstreamRequest(canonical)
		if err != nil {
			log.Printf("convert to upstream format %q failed: %v", upstreamFormat, err)
			continue
		}

		// ===== 链路日志：server 转成上游格式后发给 client 的请求体（含 tool_calls 的 id/name）=====
		// log.Printf("[TRACE] server send to client %s format=%s body=%s", client.ID, upstreamFormat, string(upstreamBody))

		// 5. 生成 fingerprint
		fingerPrint := uuid.NewString()
		if err := server.ClientFingerprintDB.SaveFingerprint(fingerPrint, client.ID, "preparing"); err != nil {
			log.Printf("save fingerprint failed: %v", err)
		}

		// 6. 发送请求到 client（携带上游格式）
		content := json.RawMessage(upstreamBody)
		// 视频透传：上游格式为 openai 时，client 端会把 Content 反序列化为
		// go-openai 的 ExtendedChatRequest，而 ChatMessagePart 不支持 video part，
		// video_url 对象会在反序列化时被丢弃（只剩 {"type":"video_url"}），
		// 导致上游 400 "video_url Field required"。与 chat completions 路径一致，
		// 这里把上游请求体放入 RawBody，client 的 openai 引擎据此走原始 JSON
		// 直连上游，保证视频 part 完整送达。model/stream 提到顶层，供 client
		// 端引擎选择与流式判断使用。
		if upstreamFormat == public.FormatOpenAI && containsVideoInput(upstreamBody) {
			var probe struct {
				Model  string `json:"model"`
				Stream bool   `json:"stream"`
			}
			_ = json.Unmarshal(upstreamBody, &probe)
			if wrapped, werr := json.Marshal(struct {
				Model   string          `json:"model"`
				Stream  bool            `json:"stream"`
				RawBody json.RawMessage `json:"raw_body"`
			}{Model: probe.Model, Stream: probe.Stream, RawBody: json.RawMessage(upstreamBody)}); werr == nil {
				content = wrapped
			}
		}
		if err := client.ControlConn.WriteJSON(public.WSMessage{
			Type:        public.MESSAGE,
			Content:     content,
			FingerPrint: fingerPrint,
			Format:      upstreamFormat,
		}); err != nil {
			log.Printf("attempt %d: send to client %s failed: %v", attempt, client.ID, err)
			client.IncrFailures()
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 7. 等待响应连接就绪
		readyCh := server.AddRespClientChan(fingerPrint)
		select {
		case <-readyCh:
		case <-time.After(public.CHAT_MAX_TIME * time.Second):
			server.RemoveRespClientChan(fingerPrint)
			log.Printf("attempt %d: response conn timeout for client %s", attempt, client.ID)
			client.IncrFailures()
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 8. 获取响应连接
		respConn, ok := server.GetRespClient(fingerPrint)
		if !ok {
			client.IncrFailures()
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 9. 更新 fingerprint 为 transmitting
		if err := server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, client.ID, "transmitting"); err != nil {
			log.Printf("update fingerprint failed: %v", err)
			client.IncrFailures()
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}
		client.IncrActiveConnections()

		// 10. 读取第一条消息
		var response public.WSMessage
		if err := respConn.ReadJSON(&response); err != nil {
			log.Printf("attempt %d: read first msg failed: %v", attempt, err)
			client.IncrFailures()
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			client.DecrActiveConnections()
			time.Sleep(backoff(attempt))
			continue
		}

		// 11. 判断第一条消息类型
		switch response.Type {
		case public.MESSAGE, public.MESSAGE_STREAM:
			client.ResetFailures()
			// P2: 成功写回亲和（同 chat.go 语义）。
			if server.Affinity != nil {
				if affinityHit || server.Affinity.LeaseExpired(affinityKey) {
					server.Affinity.Touch(affinityKey, client.ID, false)
				}
				c.Set("affinity_key", affinityKey)
				if affinityHit {
					c.Set("affinity_hit", true)
				}
			}
			handleMultiFormatResponse(c, server, fingerPrint, client.ID, ippm, oppm, cippm, canonical.Model, response, respConn, userConv, upstreamConv, upstreamFormat)
			return
		case public.CLOSE:
			log.Printf("attempt %d: client %s closed before first token", attempt, client.ID)
			client.IncrFailures()
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			client.DecrActiveConnections()
			time.Sleep(backoff(attempt))
			continue
		case public.MODEL_ERROR:
			log.Printf("attempt %d: model error from client %s: %v", attempt, client.ID, response.Content)
			if isClientRequestError(response.Content) {
				respConn.Close()
				server.RemoveRespClient(fingerPrint)
				server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
				client.DecrActiveConnections()
				c.JSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("%v", response.Content),
						"type":    "invalid_request_error",
						"param":   nil,
						"code":    "invalid_request_error",
					},
				})
				return
			}
			client.IncrFailures()
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			client.DecrActiveConnections()
			time.Sleep(backoff(attempt))
			continue
		default:
			log.Printf("attempt %d: unexpected first msg type %s", attempt, response.Type)
			client.IncrFailures()
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			client.DecrActiveConnections()
			time.Sleep(backoff(attempt))
			continue
		}
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "All clients failed, please retry"})
}

// handleMultiFormatResponse 处理多格式响应（非流式 + 流式），按用户格式回传。
func handleMultiFormatResponse(c *gin.Context, server *models.Server, fingerPrint string, clientID string, ippm, oppm, cippm float64, reqModel string, response public.WSMessage, respConn *websocket.Conn, userConv, upstreamConv format.Converter, upstreamFormat string) {
	switch response.Type {
	case public.MESSAGE:
		// 非流式：上游响应 → Canonical → 用户格式
		contentBytes, err := json.Marshal(response.Content)
		if err != nil {
			log.Println("marshal upstream response error:", err)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		}
		// ===== 链路日志：server 收到的上游非流式响应（含 tool_calls 的 id/name）=====
		// log.Printf("[TRACE] server upstream response format=%s body=%s", upstreamFormat, string(contentBytes))
		canonicalResp, err := upstreamConv.ParseUpstreamResponse(contentBytes)
		if err != nil {
			log.Println("parse upstream response error:", err)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		}
		// 思考模型：保存本轮 reasoning_content，供后续工具循环续接回传。
		saveResponseReasoning(c, server, canonicalResp)
		userBody, err := userConv.BuildResponse(canonicalResp)
		if err != nil {
			log.Println("build user response error:", err)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		}
		// ===== 链路日志：server 回传给用户的响应（含 function_call 的 call_id/name）=====
		// log.Printf("[TRACE] server send to user body=%s", string(userBody))
		c.Data(http.StatusOK, "application/json", userBody)

		// 记录 token 用量
		recordCanonicalUsage(c, server, fingerPrint, reqModel, canonicalResp.Usage, clientID, ippm, oppm, cippm)
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return

	case public.MESSAGE_STREAM:
		// 流式：设置 SSE 头
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		// 创建用户流式 writer（Responses 需要状态化输出结构性事件）
		userWriter := newUserStreamWriter(userConv, reqModel)

		// 处理第一条流式事件
		finished := handleMultiFormatStreamEvent(c, server, fingerPrint, clientID, ippm, oppm, cippm, reqModel, response, respConn, userConv, upstreamConv, upstreamFormat, userWriter)
		if finished {
			return
		}
		// 继续读取流
		readMultiFormatStreamLoop(c, server, fingerPrint, respConn, time.Now(), clientID, ippm, oppm, cippm, reqModel, userConv, upstreamConv, upstreamFormat, userWriter)
		return

	case public.CLOSE:
		log.Println("Client closed connection")
		if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
			_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
		}
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return

	case public.MODEL_ERROR:
		log.Println("Model error:", response.Content)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Model error: " + fmt.Sprintf("%v", response.Content)})
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return

	default:
		log.Println("Unknown message type:", response.Type)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown message type: " + response.Type})
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return
	}
}

// handleMultiFormatStreamEvent 处理单个流式事件，返回是否结束。
func handleMultiFormatStreamEvent(c *gin.Context, server *models.Server, fingerPrint string, clientID string, ippm, oppm, cippm float64, reqModel string, response public.WSMessage, respConn *websocket.Conn, userConv, upstreamConv format.Converter, upstreamFormat string, userWriter format.UserStreamWriter) bool {
	contentBytes, err := json.Marshal(response.Content)
	if err != nil {
		log.Println("marshal stream event error:", err)
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return true
	}

	// ===== 链路日志：server 收到的上游流式 chunk（含 tool_calls 的 id/name）=====
	// log.Printf("[TRACE] server upstream stream chunk format=%s body=%s", upstreamFormat, string(contentBytes))

	// 使用累积器处理（Anthropic/Responses 需要跨事件状态）
	acc, ok := upstreamConv.(format.AccumulatingConverter)
	if !ok {
		// OpenAI Chat：逐帧处理
		ev, err := upstreamConv.ParseUpstreamStreamEvent(contentBytes)
		if err != nil {
			log.Println("parse upstream stream event error:", err)
			return false
		}
		// ===== 链路日志：server 解析后的 Canonical 流式事件（含 tool_call 的 id/name）=====
		// if ev != nil && (ev.Type == public.StreamEventToolCallDelta || ev.Type == public.StreamEventDone) {
		// 	if evBody, e := json.Marshal(ev); e == nil {
		// 		log.Printf("[TRACE] server canonical stream event type=%s body=%s", ev.Type, string(evBody))
		// 	}
		// }
		return writeUserStreamEvent(c, server, fingerPrint, clientID, ippm, oppm, cippm, reqModel, ev, userConv, respConn, userWriter)
	}

	// 累积器：Feed 当前帧
	events, err := acc.NewStreamAccumulator().Feed(contentBytes)
	if err != nil {
		log.Println("feed stream accumulator error:", err)
		return false
	}
	finished := false
	for _, ev := range events {
		// ===== 链路日志：累积器产出的 Canonical 流式事件（含 tool_call 的 id/name）=====
		// if ev != nil && (ev.Type == public.StreamEventToolCallDelta || ev.Type == public.StreamEventDone) {
		// 	if evBody, e := json.Marshal(ev); e == nil {
		// 		log.Printf("[TRACE] server canonical stream event (acc) type=%s body=%s", ev.Type, string(evBody))
		// 	}
		// }
		if ev.Type == public.StreamEventDone {
			finished = true
		}
		if writeUserStreamEvent(c, server, fingerPrint, clientID, ippm, oppm, cippm, reqModel, ev, userConv, respConn, userWriter) {
			finished = true
		}
	}
	return finished
}

// writeUserStreamEvent 将 Canonical 流式事件转成用户格式并写出。
func writeUserStreamEvent(c *gin.Context, server *models.Server, fingerPrint string, clientID string, ippm, oppm, cippm float64, reqModel string, ev *public.CanonicalStreamEvent, userConv format.Converter, respConn *websocket.Conn, userWriter format.UserStreamWriter) bool {
	if ev == nil {
		return false
	}
	// 记录 usage
	if ev.Usage != nil && ev.Usage.TotalTokens > 0 {
		recordCanonicalUsage(c, server, fingerPrint, reqModel, *ev.Usage, clientID, ippm, oppm, cippm)
	}

	// 若用户格式需要状态化 writer（Responses），用它生成事件序列
	if userWriter != nil {
		events, err := userWriter.Write(ev)
		if err != nil {
			log.Println("user stream writer error:", err)
			return false
		}
		// ===== 链路日志：server 回传给用户的流式事件（含 function_call 的 call_id/name）=====
		for _, e := range events {
			// log.Printf("[TRACE] server send to user stream event=%s", string(e))
			_, _ = c.Writer.Write([]byte("data: " + string(e) + "\n\n"))
		}
		c.Writer.Flush()
		// done 事件：writer 内部已生成 response.completed，这里补 [DONE] 并收尾
		if ev.Type == public.StreamEventDone {
			saveStreamReasoning(c, server, userWriter)
			_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return true
		}
		if ev.Type == public.StreamEventError {
			_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return true
		}
		return false
	}

	// 结束事件
	if ev.Type == public.StreamEventDone {
		_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
		c.Writer.Flush()
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return true
	}
	// 错误事件
	if ev.Type == public.StreamEventError {
		_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
		c.Writer.Flush()
		cleanupChatRequest(server, fingerPrint, clientID, respConn)
		return true
	}
	// 普通事件 → 用户格式 SSE
	userEvent, err := userConv.BuildUserStreamEvent(ev)
	if err != nil {
		log.Println("build user stream event error:", err)
		return false
	}
	_, _ = c.Writer.Write([]byte("data: " + string(userEvent) + "\n\n"))
	c.Writer.Flush()
	return false
}

// saveStreamReasoning 在流结束时把用户 writer 累积到的 reasoning_content 保存到会话状态。
func saveStreamReasoning(c *gin.Context, server *models.Server, userWriter format.UserStreamWriter) {
	if userWriter == nil {
		return
	}
	convKey, ok := c.Get("reasoning_conv_key")
	if !ok {
		return
	}
	key, _ := convKey.(string)
	if key == "" {
		return
	}
	rp, ok := userWriter.(format.ReasoningProvider)
	if !ok {
		return
	}
	server.SaveReasoning(key, rp.ReasoningMap())
}

// saveResponseReasoning 在非流式响应结束时把 Canonical 响应中的 reasoning_content 保存到会话状态。
func saveResponseReasoning(c *gin.Context, server *models.Server, cr *public.CanonicalResponse) {
	convKey, ok := c.Get("reasoning_conv_key")
	if !ok {
		return
	}
	key, _ := convKey.(string)
	if key == "" {
		return
	}
	server.SaveReasoning(key, format.ExtractToolCallReasoning(cr))
}

// newUserStreamWriter 为需要状态化输出的用户格式（Responses）创建 writer。
func newUserStreamWriter(userConv format.Converter, model string) format.UserStreamWriter {
	if wc, ok := userConv.(format.UserStreamWriterConverter); ok {
		w := wc.NewUserStreamWriter()
		if setter, ok := w.(interface{ SetModel(string) }); ok {
			setter.SetModel(model)
		}
		return w
	}
	return nil
}

// finishUserStream 在流异常退出时补发收尾事件，保证客户端（Codex 等）始终收到
// 流终止标志，避免 "stream closed before response.completed"。
// failed=true 时以 response.failed 收尾（上游报错），否则调用 userWriter.Flush()
// 合成 response.completed（异常断开/超时/未知消息）。
func finishUserStream(c *gin.Context, userWriter format.UserStreamWriter, failed bool) {
	if userWriter != nil {
		if failed {
			_, _ = c.Writer.Write([]byte(`data: {"type":"response.failed","response":{"status":"failed"}}` + "\n\n"))
		} else {
			events, err := userWriter.Flush()
			if err != nil {
				log.Println("user stream writer flush error:", err)
			}
			for _, e := range events {
				_, _ = c.Writer.Write([]byte("data: " + string(e) + "\n\n"))
			}
		}
	}
	if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
		_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	}
	c.Writer.Flush()
}

// readMultiFormatStreamLoop 持续读取多格式流。
func readMultiFormatStreamLoop(c *gin.Context, server *models.Server, fingerPrint string, respConn *websocket.Conn, waitStart time.Time, clientID string, ippm, oppm, cippm float64, reqModel string, userConv, upstreamConv format.Converter, upstreamFormat string, userWriter format.UserStreamWriter) {
	for {
		var response public.WSMessage
		err := respConn.ReadJSON(&response)
		if err != nil {
			log.Println("Error while reading json from client:", err)
			finishUserStream(c, userWriter, false)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		}
		switch response.Type {
		case public.MESSAGE_STREAM:
			finished := handleMultiFormatStreamEvent(c, server, fingerPrint, clientID, ippm, oppm, cippm, reqModel, response, respConn, userConv, upstreamConv, upstreamFormat, userWriter)
			if finished {
				return
			}
		case public.CLOSE:
			log.Println("Client closed connection")
			finishUserStream(c, userWriter, false)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		case public.MODEL_ERROR:
			log.Println("Model error:", response.Content)
			finishUserStream(c, userWriter, true)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		default:
			log.Println("Unknown message type:", response.Type)
			finishUserStream(c, userWriter, false)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		}
		if time.Since(waitStart) > public.CHAT_MAX_TIME*time.Second {
			log.Println("Chat timeout")
			finishUserStream(c, userWriter, false)
			cleanupChatRequest(server, fingerPrint, clientID, respConn)
			return
		}
	}
}

// clientUpstreamFormat 读取 client 上某个模型的上游格式。
func clientUpstreamFormat(client *models.Client, model string) string {
	if client == nil {
		return ""
	}
	for _, m := range client.Models {
		if m.Name == model {
			return m.UpstreamFormat
		}
	}
	return ""
}

// extractConversationKey 从 Canonical 请求中提取会话标识，用于跨请求保存/回传
// reasoning_content。Codex CLI 的 Responses 请求会携带：
//   - 顶层 prompt_cache_key（会话内稳定，但 fork 子代理时会变化）
//   - client_metadata.thread_id / session_id
//
// 优先级：prompt_cache_key → client_metadata.thread_id → client_metadata.session_id
// → metadata.prompt_cache_key（历史兼容）。未命中返回空串。
func extractConversationKey(cr *public.CanonicalRequest) string {
	if cr == nil || cr.Extra == nil {
		return ""
	}
	// 1. 顶层 prompt_cache_key（ParseRequest 已存入 cr.Extra，值为 JSON 字符串字面量）。
	if key := rawStringValue(cr.Extra["prompt_cache_key"]); key != "" {
		return key
	}
	// 2. client_metadata.thread_id / session_id。
	if cm := cr.Extra["client_metadata"]; len(cm) > 0 {
		var meta map[string]json.RawMessage
		if json.Unmarshal(cm, &meta) == nil {
			if key := rawStringValue(meta["thread_id"]); key != "" {
				return key
			}
			if key := rawStringValue(meta["session_id"]); key != "" {
				return key
			}
		}
	}
	// 3. 历史兼容：metadata.prompt_cache_key。
	if metaRaw := cr.Extra["metadata"]; len(metaRaw) > 0 {
		var meta map[string]json.RawMessage
		if json.Unmarshal(metaRaw, &meta) == nil {
			return rawStringValue(meta["prompt_cache_key"])
		}
	}
	return ""
}

// affinityKeyFromCanonical 会话指纹（多格式回退）：sha256(model + 前两条消息文本前 256B)[:16]。
// 与 chat.go 的 affinityKeyFromChat 语义一致，保证同会话同 key。
func affinityKeyFromCanonical(cr *public.CanonicalRequest) string {
	if cr == nil {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(cr.Model))
	written := 0
	for _, m := range cr.Messages {
		if written >= 2 {
			break
		}
		h.Write([]byte(m.Role))
		for _, b := range m.Content {
			text := b.Text
			if len(text) > 256 {
				text = text[:256]
			}
			h.Write([]byte(text))
		}
		written++
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}

// rawStringValue 把 json.RawMessage 解析为字符串（空/无效时返回空串）。
func rawStringValue(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(r, &s) == nil {
		return s
	}
	return ""
}

// estimateCanonicalTokens 估算 Canonical 请求的 token 数（用于限流）。
func estimateCanonicalTokens(cr *public.CanonicalRequest) int {
	total := 0
	for _, m := range cr.Messages {
		for _, b := range m.Content {
			total += len(b.Text) / 4
		}
	}
	for _, s := range cr.System {
		total += len(s.Text) / 4
	}
	return total
}

// recordCanonicalUsage 记录 Canonical 格式的 token 用量。
func recordCanonicalUsage(c *gin.Context, server *models.Server, requestID string, model string, usage public.CanonicalUsage, clientID string, ippm, oppm, cippm float64) {
	recordTokenUsage(c, server, requestID, model,
		usage.InputTokens, usage.OutputTokens, usage.TotalTokens, usage.CachedTokens,
		clientID, ippm, oppm, cippm)
}
