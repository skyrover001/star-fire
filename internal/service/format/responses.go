package format

import (
	"encoding/json"
	"errors"
	"strings"

	"star-fire/pkg/public"
)

// ResponsesConverter 实现 OpenAI Responses API ↔ Canonical 的双向转换。
//
// 详见 docs/multi-format-api-design.md 3.3 节。
type ResponsesConverter struct{}

func init() {
	Register(public.FormatResponses, &ResponsesConverter{})
}

// ---- Responses 请求/响应局部结构 ----

type responsesRequest struct {
	Model           string          `json:"model"`
	Stream          bool            `json:"stream,omitempty"`
	Input           json.RawMessage `json:"input,omitempty"` // string 或 item 数组
	Instructions    json.RawMessage `json:"instructions,omitempty"`
	Tools           []responsesTool `json:"tools,omitempty"`
	MaxOutputTokens *int            `json:"max_output_tokens,omitempty"`
	Temperature     *float64        `json:"temperature,omitempty"`
	TopP            *float64        `json:"top_p,omitempty"`
	Reasoning       json.RawMessage `json:"reasoning,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Truncation      json.RawMessage `json:"truncation,omitempty"`
	// prompt_cache_key 与 client_metadata 是 Codex CLI 每次请求都携带的会话字段，
	// 用于跨请求回传 reasoning_content（思考模型工具循环续接）。
	PromptCacheKey string          `json:"prompt_cache_key,omitempty"`
	ClientMetadata json.RawMessage `json:"client_metadata,omitempty"`
}

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	// Raw 保留原始工具 JSON（用于非 function 工具透传）。
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON 保留原始工具 JSON，供非 function 工具（web_search/tool_search）透传。
func (t *responsesTool) UnmarshalJSON(data []byte) error {
	type alias responsesTool
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = responsesTool(a)
	t.Raw = append(json.RawMessage(nil), data...)
	return nil
}

// responsesItem 表示 input/output 数组中的一个 item。
type responsesItem struct {
	Type string `json:"type"`
	Role string `json:"role,omitempty"`
	// item id（web_search_call / custom_tool_call 等用 "id"；function_call 用 "call_id"）
	ID string `json:"id,omitempty"`
	// message content（string 或 content part 数组）
	Content json.RawMessage `json:"content,omitempty"`
	// function_call
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	// custom_tool_call（Codex 要求 custom_tool_call 用 input 字段，不是 arguments）
	Input string `json:"input,omitempty"`
	// function_call_output
	Output json.RawMessage `json:"output,omitempty"`
	// reasoning
	Summary json.RawMessage `json:"summary,omitempty"`
	// web_search_call / custom_tool_call 的 action（如 {"type":"search","query":"..."}）
	Action json.RawMessage `json:"action,omitempty"`
	// Chat 风格工具消息（兼容直接在 Responses input 内嵌 Chat 的 tool_calls / tool_call_id）
	ToolCalls  []responsesToolCall `json:"tool_calls,omitempty"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
}

// responsesToolCall 表示 Chat 风格 message item 里内嵌的工具调用。
type responsesToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// responsesContentPart 表示 message item 的 content part。
type responsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// input_image / output_image
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
	// input_video / output_video
	VideoURL string `json:"video_url,omitempty"`
	// reasoning 的 summary（summary_text 数组）
	Summary json.RawMessage `json:"summary,omitempty"`
	// web_search_result 的附加字段
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

type responsesResponse struct {
	ID                string          `json:"id"`
	Status            string          `json:"status"`
	Output            []responsesItem `json:"output"`
	Usage             responsesUsage  `json:"usage"`
	IncompleteDetails struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

type responsesUsage struct {
	InputTokens        int `json:"input_tokens"`
	OutputTokens       int `json:"output_tokens"`
	TotalTokens        int `json:"total_tokens"`
	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
}

const defaultResponsesMaxOutputTokens = 4096

// ---- 请求解析 ----

func (c *ResponsesConverter) ParseRequest(body []byte) (*public.CanonicalRequest, error) {
	var req responsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	if req.Model == "" {
		return nil, errors.New("responses: missing required field 'model'")
	}

	cr := &public.CanonicalRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	if req.MaxOutputTokens != nil {
		v := *req.MaxOutputTokens
		cr.MaxTokens = &v
	}
	if len(req.Instructions) > 0 {
		cr.System = parseResponsesTextContent(req.Instructions)
	}
	if len(req.Reasoning) > 0 {
		cr.Thinking = req.Reasoning
	}
	if len(req.Truncation) > 0 || len(req.Metadata) > 0 || len(req.PromptCacheKey) > 0 || len(req.ClientMetadata) > 0 {
		cr.Extra = map[string]json.RawMessage{}
		if len(req.Metadata) > 0 {
			cr.Extra["metadata"] = req.Metadata
		}
		if len(req.Truncation) > 0 {
			cr.Extra["truncation"] = req.Truncation
		}
		if len(req.PromptCacheKey) > 0 {
			cr.Extra["prompt_cache_key"] = marshalString(req.PromptCacheKey)
		}
		if len(req.ClientMetadata) > 0 {
			cr.Extra["client_metadata"] = req.ClientMetadata
		}
	}

	for _, t := range req.Tools {
		ct := public.CanonicalTool{Type: t.Type, Name: t.Name, Description: t.Description}
		if ct.Type == "" {
			ct.Type = "function"
		}
		if len(t.Parameters) > 0 {
			ct.Parameters = t.Parameters
		}
		// 非 function 工具（web_search/tool_search 等）保留原始 JSON，供下游透传。
		if ct.Type != "function" && len(t.Raw) > 0 {
			ct.Raw = t.Raw
		}
		cr.Tools = append(cr.Tools, ct)
	}

	// pendingThinking 暂存上一条 reasoning item 的思考内容（reasoning 模型要求
	// reasoning_content 随工具调用回传）。在遇到 function_call / assistant 消息时，
	// 把它作为 thinking 块合并进同一条 assistant 消息。
	var pendingThinking string

	for _, item := range parseResponsesInput(req.Input) {
		switch item.Type {
		case "function_call":
			// 若上一条是 assistant 消息（可能含 thinking/text），把工具调用合并进去，
			// 避免拆成两条 assistant 消息（reasoning 模型要求 reasoning_content 与
			// tool_calls 在同一条 assistant 消息里回传）。
			// 优先用 call_id，其次 item.id；都为空则合成一个稳定 ID，避免下游 Chat
			// 请求因 go-openai 的 omitempty 标签丢失 tool_call.id 导致后端 400。
			callID := firstNonEmpty(item.CallID, item.ID)
			if callID == "" {
				callID = "call_" + randSuffix()
			}
			// name 必须从真实值取（item.Name 或工具定义），不能盲目默认，
			// 否则模型无法识别要调用的工具，影响推理质量。
			toolName := item.Name
			if toolName == "" {
				toolName = lookupToolName(cr.Tools, callID, item.Type)
			}
			tc := public.CanonicalToolCall{
				ID:        callID,
				Name:      toolName,
				Arguments: marshalString(item.Arguments),
			}
			if n := len(cr.Messages); n > 0 && cr.Messages[n-1].Role == "assistant" {
				last := &cr.Messages[n-1]
				if pendingThinking != "" && !hasThinkingBlock(last.Content) {
					last.Content = append([]public.CanonicalContent{{Type: "thinking", Text: pendingThinking}}, last.Content...)
				}
				last.ToolCalls = append(last.ToolCalls, tc)
				last.Content = append(last.Content, public.CanonicalContent{
					Type:  "tool_use",
					ID:    callID,
					Name:  toolName,
					Input: parseArgsToInput(item.Arguments),
				})
			} else {
				content := make([]public.CanonicalContent, 0, 2)
				if pendingThinking != "" {
					content = append(content, public.CanonicalContent{Type: "thinking", Text: pendingThinking})
				}
				content = append(content, public.CanonicalContent{
					Type:  "tool_use",
					ID:    callID,
					Name:  toolName,
					Input: parseArgsToInput(item.Arguments),
				})
				cr.Messages = append(cr.Messages, public.CanonicalMessage{
					Role:      "assistant",
					Content:   content,
					ToolCalls: []public.CanonicalToolCall{tc},
				})
			}
			pendingThinking = ""
		case "reasoning":
			// 思考模型的 reasoning 输出作为独立 item 回传：提取 summary 文本，
			// 暂存为 thinking，随后续 function_call / assistant 消息合并。
			if text := extractResponsesReasoningText(item.Summary); text != "" {
				pendingThinking = text
			}
		case "function_call_output", "custom_tool_call_output":
			// function_call_output / custom_tool_call_output 结构相同（call_id + output），
			// 统一转成 tool 结果消息。custom_tool_call_output 若落入 default 分支会因
			// 无 role/content 被误转成 {"role":"user","content":null}，导致后端 400。
			cr.Messages = append(cr.Messages, public.CanonicalMessage{
				Role:       "tool",
				ToolCallID: item.CallID,
				Content: []public.CanonicalContent{{
					Type:    "tool_result",
					ID:      item.CallID,
					Content: parseResponsesTextContent(item.Output),
				}},
			})
		case "web_search_call", "custom_tool_call":
			// 搜索结果 / 自定义工具执行结果回传。Responses 里 web_search_call /
			// custom_tool_call 一个 item 同时承载「模型发起调用」与「调用结果」，下游
			// Chat 格式必须拆成两条消息：assistant(tool_calls) + tool(结果)，否则后端会
			// 报 "Messages with role 'tool' must be a response to a preceding message
			// with 'tool_calls'"。
			callID := firstNonEmpty(item.CallID, item.ID)
			toolName := item.Name
			if toolName == "" {
				if item.Type == "web_search_call" {
					toolName = "web_search"
				} else {
					toolName = "custom"
				}
			}
			args := extractToolCallArgs(item)
			tc := public.CanonicalToolCall{
				ID:        callID,
				Name:      toolName,
				Arguments: marshalString(args),
			}
			toolUse := public.CanonicalContent{
				Type:  "tool_use",
				ID:    callID,
				Name:  toolName,
				Input: parseArgsToInput(args),
			}

			// 若上一条 assistant 消息已含同 ID 的工具调用（如前面已出现 function_call），
			// 则复用；否则新建 assistant 消息（同时注入 pendingThinking）。
			n := len(cr.Messages)
			if n > 0 && cr.Messages[n-1].Role == "assistant" && hasToolCallID(cr.Messages[n-1].ToolCalls, callID) {
				// 已在前面 function_call 中建立，无需重复。
			} else {
				content := make([]public.CanonicalContent, 0, 2)
				if pendingThinking != "" {
					content = append(content, public.CanonicalContent{Type: "thinking", Text: pendingThinking})
					pendingThinking = ""
				}
				content = append(content, toolUse)
				cr.Messages = append(cr.Messages, public.CanonicalMessage{
					Role:      "assistant",
					Content:   content,
					ToolCalls: []public.CanonicalToolCall{tc},
				})
			}
			// 工具结果消息：仅当 custom_tool_call/web_search_call 自带 output 时才附加
			// tool result。Codex 协议中 custom_tool_call 的结果通常由独立的
			// custom_tool_call_output item 承载，此时 custom_tool_call 本身无 output，
			// 不应生成空 tool result（否则下游会收到两条同 ID 的 tool 消息，第一条为空）。
			if len(item.Output) > 0 && string(item.Output) != "null" {
				cr.Messages = append(cr.Messages, public.CanonicalMessage{
					Role:       "tool",
					ToolCallID: callID,
					Content: []public.CanonicalContent{{
						Type:    "tool_result",
						ID:      callID,
						Content: parseResponsesTextContent(item.Output),
					}},
				})
			}
		default:
			// developer / system 角色：合并进顶层 System（归一为单条 system 消息，
			// 避免多条 system 落在 messages 中间被 Qwen 等后端拒绝）。
			if item.Role == "developer" || item.Role == "system" {
				for _, p := range parseResponsesContentParts(item.Content) {
					switch p.Type {
					case "input_text", "output_text", "text", "":
						if p.Text != "" {
							cr.System = append(cr.System, public.CanonicalContent{Type: "text", Text: p.Text})
						}
					}
				}
				continue
			}
			// type 缺省（如 {"role":"user","content":[...]}）视作 message。
			msg := responsesMessageToCanonical(item)
			if pendingThinking != "" && msg.Role == "assistant" && !hasThinkingBlock(msg.Content) {
				msg.Content = append([]public.CanonicalContent{{Type: "thinking", Text: pendingThinking}}, msg.Content...)
				pendingThinking = ""
			}
			// 防护：跳过既无 content 又无 tool_calls 的消息（避免下游收到
			// {"role":"user","content":null} 之类的畸形消息触发 400）。
			if len(msg.Content) == 0 && len(msg.ToolCalls) == 0 {
				continue
			}
			cr.Messages = append(cr.Messages, msg)
		}
	}

	return cr, nil
}

// BuildUpstreamRequest 将 Canonical 请求 → Responses API 请求。
func (c *ResponsesConverter) BuildUpstreamRequest(cr *public.CanonicalRequest) ([]byte, error) {
	req := responsesRequest{
		Model:       cr.Model,
		Stream:      cr.Stream,
		Temperature: cr.Temperature,
		TopP:        cr.TopP,
	}
	if cr.MaxTokens != nil {
		v := *cr.MaxTokens
		req.MaxOutputTokens = &v
	} else {
		v := defaultResponsesMaxOutputTokens
		req.MaxOutputTokens = &v
	}
	if len(cr.System) > 0 {
		req.Instructions = marshalString(blocksToText(cr.System))
	}
	if len(cr.Thinking) > 0 {
		req.Reasoning = cr.Thinking
	}
	if cr.Extra != nil {
		if m, ok := cr.Extra["metadata"]; ok {
			req.Metadata = m
		}
		if t, ok := cr.Extra["truncation"]; ok {
			req.Truncation = t
		}
	}

	for _, t := range cr.Tools {
		req.Tools = append(req.Tools, responsesTool{
			Type:        "function",
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}

	items := make([]responsesItem, 0, len(cr.Messages))
	for _, cm := range cr.Messages {
		items = append(items, canonicalMessageToResponsesItem(cm)...)
	}
	req.Input = marshalResponsesItems(items)

	return json.Marshal(req)
}

// ParseUpstreamResponse 解析 Responses 响应 → Canonical。
func (c *ResponsesConverter) ParseUpstreamResponse(data []byte) (*public.CanonicalResponse, error) {
	var resp responsesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	cr := &public.CanonicalResponse{
		ID:           resp.ID,
		FinishReason: responsesFinishReason(resp),
	}
	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range parseResponsesContentParts(item.Content) {
				switch part.Type {
				case "output_text":
					cr.Content = append(cr.Content, public.CanonicalContent{Type: "text", Text: part.Text})
				case "output_image":
					cr.Content = append(cr.Content, public.CanonicalContent{Type: "image", ImageURL: part.ImageURL})
				case "output_video":
					cr.Content = append(cr.Content, public.CanonicalContent{Type: "video", VideoURL: part.VideoURL})
				}
			}
		case "function_call":
			cr.Content = append(cr.Content, public.CanonicalContent{
				Type:     "tool_use",
				ID:       item.CallID,
				Name:     item.Name,
				Input:    parseArgsToInput(item.Arguments),
				ToolType: "function",
			})
		case "web_search_call":
			cr.Content = append(cr.Content, public.CanonicalContent{
				Type:     "tool_use",
				ID:       item.CallID,
				Name:     "web_search",
				Input:    parseArgsToInput(item.Arguments),
				ToolType: "web_search",
			})
		case "custom_tool_call":
			// custom_tool_call 用 input 字段（Codex 规范），但兼容旧版 arguments。
			args := item.Input
			if args == "" {
				args = item.Arguments
			}
			cr.Content = append(cr.Content, public.CanonicalContent{
				Type:     "tool_use",
				ID:       item.CallID,
				Name:     item.Name,
				Input:    parseArgsToInput(args),
				ToolType: "custom",
			})
		case "reasoning":
			cc := public.CanonicalContent{Type: "thinking"}
			if item.Summary != nil {
				if text := extractResponsesReasoningText(item.Summary); text != "" {
					cc.Text = text
				}
			}
			cr.Content = append(cr.Content, cc)
		}
	}
	cr.Usage = responsesUsageToCanonical(resp.Usage)
	return cr, nil
}

// BuildResponse 将 Canonical 响应 → Responses 响应。
func (c *ResponsesConverter) BuildResponse(cr *public.CanonicalResponse) ([]byte, error) {
	resp := responsesResponse{
		ID:     cr.ID,
		Status: canonicalFinishToResponsesStatus(cr.FinishReason),
	}
	for _, b := range cr.Content {
		switch b.Type {
		case "text":
			resp.Output = append(resp.Output, responsesItem{
				Type:    "message",
				Role:    "assistant",
				Content: marshalResponsesParts([]responsesContentPart{{Type: "output_text", Text: b.Text}}),
			})
		case "image":
			resp.Output = append(resp.Output, responsesItem{
				Type:    "message",
				Role:    "assistant",
				Content: marshalResponsesParts([]responsesContentPart{{Type: "output_image", ImageURL: b.ImageURL}}),
			})
		case "video":
			resp.Output = append(resp.Output, responsesItem{
				Type:    "message",
				Role:    "assistant",
				Content: marshalResponsesParts([]responsesContentPart{{Type: "output_video", VideoURL: b.VideoURL}}),
			})
		case "tool_use":
			// 按工具类型映射到标准 Responses item 类型：
			//   function → function_call
			//   web_search → web_search_call
			//   custom → custom_tool_call
			//   tool_search / namespace → function_call（Codex 专有，保留 name）
			itemType := "function_call"
			switch b.ToolType {
			case "web_search":
				itemType = "web_search_call"
			case "custom":
				itemType = "custom_tool_call"
			}
			argsStr := rawToArgsString(b.Input)
			item := responsesItem{
				Type:   itemType,
				CallID: b.ID,
				Name:   b.Name,
			}
			// custom_tool_call 用 input 字段（Codex 要求），且对 FREEFORM 工具
			// 提取原始内容（{"input":"<patch>"} → <patch>）；
			// web_search_call 用 action 字段（标准 Responses）；
			// 其他类型（function_call）用 arguments。
			switch itemType {
			case "custom_tool_call":
				if isFreeformTool(b.Name, nil) {
					item.Input = extractFreeformInput(argsStr)
				} else {
					item.Input = argsStr
				}
			case "web_search_call":
				item.Action, _ = json.Marshal(webSearchActionMap(argsStr))
			default:
				item.Arguments = argsStr
			}
			resp.Output = append(resp.Output, item)
		case "thinking":
			resp.Output = append(resp.Output, responsesItem{
				Type:    "reasoning",
				Summary: marshalResponsesParts([]responsesContentPart{{Type: "summary_text", Text: b.Text}}),
			})
		}
	}
	resp.Usage = canonicalToResponsesUsage(cr.Usage)
	return json.Marshal(resp)
}

// ---- 流式 ----

func (c *ResponsesConverter) NewStreamAccumulator() StreamAccumulator {
	return &responsesAccumulator{}
}

func (c *ResponsesConverter) ParseUpstreamStreamEvent(line []byte) (*public.CanonicalStreamEvent, error) {
	acc := c.NewStreamAccumulator()
	evs, err := acc.Feed(line)
	if err != nil {
		return nil, err
	}
	if len(evs) > 0 {
		return evs[0], nil
	}
	return acc.Flush()
}

// BuildUserStreamEvent 将 Canonical 流式事件 → Responses 流式事件 JSON。
func (c *ResponsesConverter) BuildUserStreamEvent(ev *public.CanonicalStreamEvent) ([]byte, error) {
	switch ev.Type {
	case public.StreamEventTextDelta:
		return json.Marshal(map[string]any{
			"type":  "response.output_text.delta",
			"delta": ev.Text,
		})
	case public.StreamEventToolCallDelta:
		return json.Marshal(map[string]any{
			"type":  "response.function_call_arguments.delta",
			"delta": rawToString(ev.ToolCall.Arguments),
		})
	case public.StreamEventDone:
		return json.Marshal(map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"status": canonicalFinishToResponsesStatus(ev.FinishReason),
			},
		})
	default:
		return json.Marshal(map[string]any{"type": "response.failed"})
	}
}

// NewUserStreamWriter 返回一个状态化的用户流式 writer。
// OpenAI Responses 流式要求输出一整套结构性事件（response.created、
// output_item.added、content_part.added、output_text.delta×N、output_text.done、
// content_part.done、output_item.done、response.completed），Codex CLI 依赖这些
// 事件建立上下文。本 writer 在首个 delta 前合成结构性事件，在 done 时合成收尾事件。
func (c *ResponsesConverter) NewUserStreamWriter() UserStreamWriter {
	return &responsesUserWriter{}
}

// SetModel 设置 writer 的模型名（用于 response.created / completed 的 model 字段）。
func (w *responsesUserWriter) SetModel(model string) {
	w.model = model
}

// responsesUserWriter 累积 Responses 用户流式输出的状态。
// 一个响应可包含多个 output item（例如“文本 + 工具调用”，或一次多个工具调用），
// 每个 item 单独累积、按 output_index 递增，最终在 response.completed 中按序输出。
type responsesUserWriter struct {
	started       bool
	createdSent   bool
	usage         public.CanonicalUsage
	finishReason  string
	model         string
	responseID    string
	createdAt     int64
	reasoning     string                 // 累积的 reasoning_content（思考模型工具循环回传）
	functionID    string                 // 最近一次 tool_call_id（reasoning 回传用）
	functionIDs   []string               // 本流出现过的全部 tool_call_id（按每个调用回传同一段 reasoning）
	items         []*responsesStreamItem // 已完成的 output item（按序）
	current       *responsesStreamItem   // 当前进行中的 item（可能为 nil）
	completedSent bool
}

// responsesStreamItem 表示 Responses 流式输出中的一个 output item。
type responsesStreamItem struct {
	itemType     string // message | function_call | web_search_call | custom_tool_call
	id           string
	contentIndex int // message 的 content part 序号（目前恒为 0）
	text         string
	functionName string
	functionArgs string
	functionID   string
	toolType     string // function | web_search | custom | tool_search | namespace | ...
	// freeformBuffering 为 true 时，工具调用参数不逐帧下发，而是累积后在
	// item 关闭时提取 input 字段并作为单帧发送。用于 apply_patch 等 FREEFORM
	// 工具：模型按注入的 schema 生成 {"input":"<content>"}，网关需还原为
	// 原始自由格式文本，让 Codex 收到的是 patch 内容而非 JSON 包裹。
	freeformBuffering bool
}

func (w *responsesUserWriter) Write(ev *public.CanonicalStreamEvent) ([][]byte, error) {
	// 记录 usage / finish_reason
	if ev.Usage != nil {
		w.usage = *ev.Usage
	}
	if ev.FinishReason != "" {
		w.finishReason = ev.FinishReason
	}

	switch ev.Type {
	case public.StreamEventThinkingDelta:
		// 思考内容增量：透传累积 reasoning_content，供工具循环续接时回传。
		w.reasoning += ev.Text
		return nil, nil

	case public.StreamEventTextDelta:
		if !w.started {
			w.startResponse()
		}
		var out [][]byte
		// 当前不是 message item → 先收尾，再开新的 message item。
		if w.current == nil || w.current.itemType != "message" {
			out = append(out, w.closeCurrentItem()...)
			out = append(out, w.openMessageItem()...)
		}
		w.current.text += ev.Text
		out = append(out, w.buildTextDeltaEvent(ev.Text))
		return out, nil

	case public.StreamEventToolCallDelta:
		if !w.started {
			w.startResponse()
		}
		tc := ev.ToolCall
		args := ""
		if tc != nil {
			args = rawToString(tc.Arguments)
		}
		var out [][]byte
		// 新工具调用（当前非 function item，或 call_id 变化）→ 先收尾再开新 function item。
		if w.current == nil || w.current.itemType == "message" ||
			(tc != nil && tc.ID != "" && w.current.functionID != tc.ID) {
			out = append(out, w.closeCurrentItem()...)
			out = append(out, w.openFunctionItem(tc)...)
		}
		if tc != nil {
			if tc.Name != "" {
				w.current.functionName = tc.Name
				w.current.toolType = inferToolType(tc.Name)
				// 检测 FREEFORM 工具：apply_patch 等工具的参数需缓冲后提取 input 字段。
				if isFreeformTool(tc.Name, nil) {
					w.current.freeformBuffering = true
				}
			}
			if tc.ID != "" {
				w.current.functionID = tc.ID
				w.functionID = tc.ID
				w.recordFunctionID(tc.ID)
			}
			w.current.functionArgs += args
		}
		// FREEFORM 工具：不逐帧下发参数，累积后在 closeCurrentItem 提取 input。
		if !w.current.freeformBuffering {
			out = append(out, w.buildFunctionDeltaEvent(args))
		}
		return out, nil

	case public.StreamEventDone:
		var out [][]byte
		if !w.started {
			w.startResponse()
			// 空回复时也必须先发 response.created，再发 response.completed。
			out = append(out, w.ensureResponseCreated()...)
		}
		out = append(out, w.closeCurrentItem()...)
		out = append(out, w.buildCompletedEvent()...)
		w.completedSent = true
		return out, nil

	case public.StreamEventError:
		return [][]byte{
			[]byte(`{"type":"response.failed","response":{"status":"failed"}}`),
		}, nil

	default:
		return nil, nil
	}
}

// ReasoningMap 返回本流累积到的 reasoning_content，keyed by 最近一次 tool_call_id。
// 若没有工具调用，则 key 为 ""。供适配层保存到会话状态，以便后续续接请求回传 reasoning。
func (w *responsesUserWriter) ReasoningMap() map[string]string {
	out := map[string]string{}
	if w.reasoning == "" {
		return out
	}
	// 为每个出现过的 tool_call_id 都关联同一段 reasoning_content：
	// 同一 assistant 回合的并行工具调用共享一段 reasoning，续接请求可能只回传
	// 其中任意一个（或第一个）call_id，必须全部命中才能正确回传。
	for _, id := range w.functionIDs {
		if id != "" {
			out[id] = w.reasoning
		}
	}
	return out
}

// recordFunctionID 记录本流出现过的 tool_call_id（去重）。
func (w *responsesUserWriter) recordFunctionID(id string) {
	if id == "" {
		return
	}
	for _, existing := range w.functionIDs {
		if existing == id {
			return
		}
	}
	w.functionIDs = append(w.functionIDs, id)
}

func (w *responsesUserWriter) Flush() ([][]byte, error) {
	if w.completedSent {
		return nil, nil
	}
	if !w.started {
		w.startResponse()
	}
	out := w.ensureResponseCreated()
	out = append(out, w.closeCurrentItem()...)
	out = append(out, w.buildCompletedEvent()...)
	w.completedSent = true
	return out, nil
}

// startResponse 初始化响应级状态（responseID / createdAt）。
func (w *responsesUserWriter) startResponse() {
	w.started = true
	w.responseID = "resp_" + randSuffix()
	w.createdAt = timeNowUnix()
}

// curOutputIndex 返回当前 item 的 output_index（= 已完成的 item 数量）。
func (w *responsesUserWriter) curOutputIndex() int {
	return len(w.items)
}

// ensureResponseCreated 确保 response.created 仅发出一次。
func (w *responsesUserWriter) ensureResponseCreated() [][]byte {
	if w.createdSent {
		return nil
	}
	w.createdSent = true
	created := map[string]any{
		"type": "response.created",
		"response": map[string]any{
			"id":                  w.responseID,
			"object":              "response",
			"created_at":          w.createdAt,
			"status":              "in_progress",
			"model":               w.model,
			"output":              []any{},
			"usage":               nil,
			"instructions":        nil,
			"max_output_tokens":   nil,
			"temperature":         nil,
			"tool_choice":         nil,
			"tools":               []any{},
			"parallel_tool_calls": true,
			"top_p":               nil,
			"metadata":            map[string]any{},
			"reasoning":           nil,
		},
	}
	b, _ := json.Marshal(created)
	return [][]byte{b}
}

// openMessageItem 开启一个新的 message item（含 content_part）。
func (w *responsesUserWriter) openMessageItem() [][]byte {
	out := w.ensureResponseCreated()
	w.current = &responsesStreamItem{
		itemType: "message",
		id:       "msg_" + randSuffix(),
	}
	out = append(out, w.buildOutputItemAdded(), w.buildContentPartAdded())
	return out
}

// openFunctionItem 开启一个新的工具调用 item。
func (w *responsesUserWriter) openFunctionItem(tc *public.CanonicalToolCall) [][]byte {
	out := w.ensureResponseCreated()
	name, id, toolType := "", "", "function"
	if tc != nil {
		name, id = tc.Name, tc.ID
		toolType = inferToolType(name)
	}
	w.current = &responsesStreamItem{
		itemType:     mappedToolItemType(toolType),
		id:           "fc_" + randSuffix(),
		functionName: name,
		functionID:   id,
		toolType:     toolType,
	}
	if id != "" {
		w.functionID = id
		w.recordFunctionID(id)
	}
	out = append(out, w.buildOutputItemAdded())
	return out
}

// closeCurrentItem 收尾当前 item（message 额外发 output_text.done / content_part.done），
// 归入 items 并返回事件。
func (w *responsesUserWriter) closeCurrentItem() [][]byte {
	if w.current == nil {
		return nil
	}
	cur := w.current
	w.current = nil
	idx := w.curOutputIndex()
	var out [][]byte
	if cur.itemType == "message" {
		out = append(out, w.buildTextDone(cur, idx), w.buildContentPartDone(cur, idx))
	}
	// FREEFORM 工具：把缓冲的 {"input":"<content>"} 提取为原始内容，
	// 作为单帧 delta 下发，并替换 functionArgs 为提取后的原始内容。
	if cur.freeformBuffering {
		if input := extractFreeformInput(cur.functionArgs); input != "" {
			out = append(out, w.buildFunctionDeltaEventFor(cur, idx, input))
			cur.functionArgs = input
		}
	}
	out = append(out, w.buildOutputItemDone(cur, idx))
	w.items = append(w.items, cur)
	return out
}

// mappedToolItemType 把工具类型映射为标准 Responses item 类型。
func mappedToolItemType(toolType string) string {
	switch toolType {
	case "web_search":
		return "web_search_call"
	case "custom":
		return "custom_tool_call"
	default:
		return "function_call"
	}
}

// streamItemToMap 把 stream item 序列化为 output item（用于 output_item.done / response.completed）。
func streamItemToMap(it *responsesStreamItem) map[string]any {
	item := map[string]any{
		"id":     it.id,
		"type":   it.itemType,
		"status": "completed",
	}
	if it.itemType == "message" {
		item["role"] = "assistant"
		item["content"] = []any{map[string]any{
			"type":        "output_text",
			"text":        it.text,
			"annotations": []any{},
		}}
	} else {
		item["name"] = it.functionName
		item["call_id"] = it.functionID
		// 不同工具类型使用各自的字段约定：
		//   custom_tool_call → input（Codex 要求）
		//   web_search_call  → action（标准 Responses：{"type":"search","query":"..."}）
		//   其他（function_call）→ arguments
		switch it.itemType {
		case "custom_tool_call":
			item["input"] = it.functionArgs
		case "web_search_call":
			item["action"] = webSearchActionMap(it.functionArgs)
		default:
			item["arguments"] = it.functionArgs
		}
	}
	return item
}

func (w *responsesUserWriter) buildOutputItemAdded() []byte {
	item := map[string]any{
		"id":     w.current.id,
		"type":   w.current.itemType,
		"status": "in_progress",
	}
	if w.current.itemType == "message" {
		item["role"] = "assistant"
		item["content"] = []any{}
	} else {
		item["name"] = w.current.functionName
		item["call_id"] = w.current.functionID
		// 不同工具类型使用各自的字段约定（in_progress 阶段先给空值占位）。
		switch w.current.itemType {
		case "custom_tool_call":
			item["input"] = ""
		case "web_search_call":
			item["action"] = webSearchActionMap("")
		default:
			item["arguments"] = ""
		}
	}
	ev := map[string]any{
		"type":         "response.output_item.added",
		"output_index": w.curOutputIndex(),
		"item":         item,
	}
	b, _ := json.Marshal(ev)
	return b
}

func (w *responsesUserWriter) buildContentPartAdded() []byte {
	ev := map[string]any{
		"type":          "response.content_part.added",
		"item_id":       w.current.id,
		"output_index":  w.curOutputIndex(),
		"content_index": w.current.contentIndex,
		"part": map[string]any{
			"type":        "output_text",
			"text":        "",
			"annotations": []any{},
		},
	}
	b, _ := json.Marshal(ev)
	return b
}

func (w *responsesUserWriter) buildTextDeltaEvent(delta string) []byte {
	ev := map[string]any{
		"type":          "response.output_text.delta",
		"item_id":       w.current.id,
		"output_index":  w.curOutputIndex(),
		"content_index": w.current.contentIndex,
		"delta":         delta,
	}
	b, _ := json.Marshal(ev)
	return b
}

// buildFunctionDeltaEvent 构建工具调用参数增量事件。
// custom_tool_call 用 response.custom_tool_call_input.delta（Codex 要求），
// 其他用 response.function_call_arguments.delta。
func (w *responsesUserWriter) buildFunctionDeltaEvent(delta string) []byte {
	evType := "response.function_call_arguments.delta"
	if w.current != nil && w.current.itemType == "custom_tool_call" {
		evType = "response.custom_tool_call_input.delta"
	}
	ev := map[string]any{
		"type":         evType,
		"item_id":      w.current.id,
		"output_index": w.curOutputIndex(),
		"delta":        delta,
	}
	b, _ := json.Marshal(ev)
	return b
}

// buildFunctionDeltaEventFor 与 buildFunctionDeltaEvent 类似，但使用指定的 item 和 index。
// 用于 closeCurrentItem 中 w.current 已被置 nil 的场景（如 FREEFORM 工具提取后下发）。
func (w *responsesUserWriter) buildFunctionDeltaEventFor(cur *responsesStreamItem, idx int, delta string) []byte {
	evType := "response.function_call_arguments.delta"
	if cur.itemType == "custom_tool_call" {
		evType = "response.custom_tool_call_input.delta"
	}
	ev := map[string]any{
		"type":         evType,
		"item_id":      cur.id,
		"output_index": idx,
		"delta":        delta,
	}
	b, _ := json.Marshal(ev)
	return b
}

func (w *responsesUserWriter) buildTextDone(cur *responsesStreamItem, idx int) []byte {
	ev := map[string]any{
		"type":          "response.output_text.done",
		"item_id":       cur.id,
		"output_index":  idx,
		"content_index": cur.contentIndex,
		"text":          cur.text,
	}
	b, _ := json.Marshal(ev)
	return b
}

func (w *responsesUserWriter) buildContentPartDone(cur *responsesStreamItem, idx int) []byte {
	ev := map[string]any{
		"type":          "response.content_part.done",
		"item_id":       cur.id,
		"output_index":  idx,
		"content_index": cur.contentIndex,
		"part": map[string]any{
			"type":        "output_text",
			"text":        cur.text,
			"annotations": []any{},
		},
	}
	b, _ := json.Marshal(ev)
	return b
}

func (w *responsesUserWriter) buildOutputItemDone(cur *responsesStreamItem, idx int) []byte {
	ev := map[string]any{
		"type":         "response.output_item.done",
		"output_index": idx,
		"item":         streamItemToMap(cur),
	}
	b, _ := json.Marshal(ev)
	return b
}

// buildCompletedEvent 合成 response.completed，output 按序包含所有 item。
func (w *responsesUserWriter) buildCompletedEvent() [][]byte {
	output := make([]any, 0, len(w.items))
	for _, it := range w.items {
		output = append(output, streamItemToMap(it))
	}
	completed := map[string]any{
		"type": "response.completed",
		"response": map[string]any{
			"id":         w.responseID,
			"object":     "response",
			"created_at": w.createdAt,
			"status":     "completed",
			"model":      w.model,
			"output":     output,
			"usage": map[string]any{
				"input_tokens":         w.usage.InputTokens,
				"output_tokens":        w.usage.OutputTokens,
				"total_tokens":         w.usage.TotalTokens,
				"input_tokens_details": map[string]any{"cached_tokens": w.usage.CachedTokens},
			},
		},
	}
	b, _ := json.Marshal(completed)
	return [][]byte{b}
}

type responsesAccumulator struct {
	inputTokens     int
	outputTokens    int
	cachedTokens    int
	totalTokens     int
	status          string
	sawFunctionCall bool
	done            bool
	toolsByOutput   map[int]*responseStreamTool
	toolsByItemID   map[string]*responseStreamTool
	pendingArgs     map[int]string
}

type responseStreamTool struct {
	callID string
	name   string
	args   string
}

func (a *responsesAccumulator) Feed(data []byte) ([]*public.CanonicalStreamEvent, error) {
	var ev struct {
		Type        string          `json:"type"`
		Delta       string          `json:"delta"`
		Arguments   string          `json:"arguments"`
		ItemID      string          `json:"item_id"`
		OutputIndex *int            `json:"output_index"`
		Response    json.RawMessage `json:"response"`
		Item        json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, err
	}

	switch ev.Type {
	case "response.output_text.delta":
		return []*public.CanonicalStreamEvent{{Type: public.StreamEventTextDelta, Text: ev.Delta}}, nil

	case "response.output_item.added", "response.output_item.done":
		var item responsesItem
		if err := json.Unmarshal(ev.Item, &item); err != nil {
			return nil, err
		}
		if item.Type == "function_call" || item.Type == "custom_tool_call" || item.Type == "web_search_call" {
			return a.addTool(item, ev.OutputIndex)
		}
		return nil, nil

	case "response.function_call_arguments.delta", "response.custom_tool_call_input.delta":
		return a.addToolArgs(ev.ItemID, ev.OutputIndex, ev.Delta)

	case "response.function_call_arguments.done", "response.custom_tool_call_input.done":
		return a.completeToolArgs(ev.ItemID, ev.OutputIndex, ev.Arguments)

	case "response.completed", "response.incomplete", "response.done":
		a.done = true
		var resp responsesResponse
		_ = json.Unmarshal(ev.Response, &resp)
		if resp.Status == "" {
			if ev.Type == "response.incomplete" {
				resp.Status = "incomplete"
			} else {
				resp.Status = "completed"
			}
		}
		a.status = resp.Status
		a.inputTokens = resp.Usage.InputTokens
		a.outputTokens = resp.Usage.OutputTokens
		a.cachedTokens = resp.Usage.InputTokensDetails.CachedTokens
		a.totalTokens = resp.Usage.TotalTokens
		finishReason := a.finishReason()
		if resp.IncompleteDetails.Reason == "content_filter" {
			finishReason = "content_filter"
		}
		u := a.buildUsage()
		return []*public.CanonicalStreamEvent{{
			Type:         public.StreamEventDone,
			FinishReason: finishReason,
			Usage:        &u,
		}}, nil

	case "response.failed", "response.error":
		a.done = true
		return []*public.CanonicalStreamEvent{{Type: public.StreamEventError, Text: ev.Delta}}, nil

	default:
		return nil, nil
	}
}

func (a *responsesAccumulator) addTool(item responsesItem, outputIndex *int) ([]*public.CanonicalStreamEvent, error) {
	if a.toolsByOutput == nil {
		a.toolsByOutput = make(map[int]*responseStreamTool)
		a.toolsByItemID = make(map[string]*responseStreamTool)
	}
	if a.pendingArgs == nil {
		a.pendingArgs = make(map[int]string)
	}
	itemID := item.ID
	tool := a.toolsByItemID[itemID]
	if tool == nil && outputIndex != nil {
		tool = a.toolsByOutput[*outputIndex]
	}
	if tool == nil {
		tool = &responseStreamTool{}
	}
	tool.callID = firstNonEmpty(item.CallID, item.ID)
	tool.name = item.Name
	args := streamToolArgs(item)
	if args != "" {
		tool.args = args
	}
	if outputIndex != nil {
		a.toolsByOutput[*outputIndex] = tool
		if pending := a.pendingArgs[*outputIndex]; pending != "" {
			tool.args += pending
			delete(a.pendingArgs, *outputIndex)
		}
	}
	if itemID != "" {
		a.toolsByItemID[itemID] = tool
	}
	a.sawFunctionCall = true
	return []*public.CanonicalStreamEvent{{
		Type:     public.StreamEventToolCallDelta,
		ToolCall: &public.CanonicalToolCall{ID: tool.callID, Name: tool.name, Arguments: marshalString(tool.args)},
	}}, nil
}

func (a *responsesAccumulator) addToolArgs(itemID string, outputIndex *int, args string) ([]*public.CanonicalStreamEvent, error) {
	if args == "" {
		return nil, nil
	}
	var tool *responseStreamTool
	if itemID != "" && a.toolsByItemID != nil {
		tool = a.toolsByItemID[itemID]
	}
	if tool == nil && outputIndex != nil && a.toolsByOutput != nil {
		tool = a.toolsByOutput[*outputIndex]
	}
	if tool == nil {
		if outputIndex != nil {
			if a.pendingArgs == nil {
				a.pendingArgs = make(map[int]string)
			}
			a.pendingArgs[*outputIndex] += args
		}
		return nil, nil
	}
	tool.args += args
	return []*public.CanonicalStreamEvent{{
		Type:     public.StreamEventToolCallDelta,
		ToolCall: &public.CanonicalToolCall{Arguments: marshalString(args)},
	}}, nil
}

func (a *responsesAccumulator) completeToolArgs(itemID string, outputIndex *int, args string) ([]*public.CanonicalStreamEvent, error) {
	if args == "" {
		return nil, nil
	}
	var tool *responseStreamTool
	if itemID != "" && a.toolsByItemID != nil {
		tool = a.toolsByItemID[itemID]
	}
	if tool == nil && outputIndex != nil && a.toolsByOutput != nil {
		tool = a.toolsByOutput[*outputIndex]
	}
	if tool == nil {
		return a.addToolArgs(itemID, outputIndex, args)
	}
	if tool.args == args || strings.HasSuffix(tool.args, args) {
		return nil, nil
	}
	if strings.HasPrefix(args, tool.args) {
		return a.addToolArgs(itemID, outputIndex, args[len(tool.args):])
	}
	return a.addToolArgs(itemID, outputIndex, args)
}

func streamToolArgs(item responsesItem) string {
	if item.Type == "custom_tool_call" && item.Input != "" {
		return item.Input
	}
	if item.Arguments != "" {
		return item.Arguments
	}
	if len(item.Action) > 0 {
		return string(compactJSON(item.Action))
	}
	return ""
}

func (a *responsesAccumulator) Flush() (*public.CanonicalStreamEvent, error) {
	if a.done {
		return nil, nil
	}
	u := a.buildUsage()
	return &public.CanonicalStreamEvent{
		Type:         public.StreamEventDone,
		FinishReason: a.finishReason(),
		Usage:        &u,
	}, nil
}

// finishReason 计算流式完成事件的原因：若流中出现过 function_call，
// 则归一为 tool_calls（对应设计文档 (4) ④ 的断言）。
func (a *responsesAccumulator) finishReason() string {
	if a.sawFunctionCall && a.status == "completed" {
		return "tool_calls"
	}
	return responsesStatusToCanonical(a.status)
}

func (a *responsesAccumulator) buildUsage() public.CanonicalUsage {
	if a.totalTokens == 0 {
		a.totalTokens = a.inputTokens + a.outputTokens
	}
	return public.CanonicalUsage{
		InputTokens:  a.inputTokens,
		OutputTokens: a.outputTokens,
		TotalTokens:  a.totalTokens,
		CachedTokens: a.cachedTokens,
	}
}

// ---- 辅助 ----

func parseResponsesInput(raw json.RawMessage) []responsesItem {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []responsesItem{{Type: "message", Role: "user", Content: marshalResponsesParts([]responsesContentPart{{Type: "input_text", Text: s}})}}
	}
	var items []responsesItem
	if json.Unmarshal(raw, &items) == nil {
		return items
	}
	return nil
}

func parseResponsesContentParts(raw json.RawMessage) []responsesContentPart {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []responsesContentPart{{Type: "output_text", Text: s}}
	}
	var parts []responsesContentPart
	if json.Unmarshal(raw, &parts) == nil {
		return parts
	}
	// 单个 content part 对象（如 {"type":"output_text","text":"..."}）。
	var single responsesContentPart
	if json.Unmarshal(raw, &single) == nil {
		return []responsesContentPart{single}
	}
	return nil
}

// parseResponsesTextContent 把 string 或 content part 数组统一成 Canonical 文本块。
// 额外处理 web_search_call 的 output（web_search_result 数组），把标题/链接/正文透传给
// 下游模型，避免搜索结果内容被吞。
func parseResponsesTextContent(raw json.RawMessage) []public.CanonicalContent {
	parts := parseResponsesContentParts(raw)
	out := make([]public.CanonicalContent, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "input_text", "output_text", "text", "":
			if p.Text != "" {
				out = append(out, public.CanonicalContent{Type: "text", Text: p.Text})
			}
		case "input_image", "output_image", "image":
			out = append(out, public.CanonicalContent{Type: "image", ImageURL: p.ImageURL})
		case "input_video", "output_video", "video":
			out = append(out, public.CanonicalContent{Type: "video", VideoURL: p.VideoURL})
		case "web_search_result":
			// 搜索结果：合并 title / url / text，确保下游模型能看到搜索内容。
			var sb strings.Builder
			if p.Title != "" {
				sb.WriteString(p.Title)
				sb.WriteString("\n")
			}
			if p.URL != "" {
				sb.WriteString(p.URL)
				sb.WriteString("\n")
			}
			sb.WriteString(p.Text)
			if sb.Len() > 0 {
				out = append(out, public.CanonicalContent{Type: "text", Text: sb.String()})
			}
		}
	}
	return out
}

func responsesMessageToCanonical(item responsesItem) public.CanonicalMessage {
	cm := public.CanonicalMessage{Role: item.Role}
	if cm.Role == "" {
		cm.Role = "user"
	}

	// Chat 风格 tool 消息：{"role":"tool","tool_call_id":"...","content":"..."}。
	// 即使 tool_call_id 为空（如 Codex CLI 对不支持的调用生成的 "unsupported call"
	// 结果），也保留为 tool 消息，由下游 BuildUpstreamRequest 兜底关联 ID。
	if item.Role == "tool" {
		cm.ToolCallID = item.ToolCallID
		cm.Content = []public.CanonicalContent{{
			Type:    "tool_result",
			ID:      item.ToolCallID,
			Content: parseResponsesTextContent(item.Content),
		}}
		return cm
	}

	for _, p := range parseResponsesContentParts(item.Content) {
		switch p.Type {
		case "input_text", "output_text", "text", "":
			if p.Text != "" {
				cm.Content = append(cm.Content, public.CanonicalContent{Type: "text", Text: p.Text})
			}
		case "input_image", "output_image", "image":
			cm.Content = append(cm.Content, public.CanonicalContent{Type: "image", ImageURL: p.ImageURL})
		case "input_video", "output_video", "video":
			cm.Content = append(cm.Content, public.CanonicalContent{Type: "video", VideoURL: p.VideoURL})
		case "reasoning", "summary_text":
			// 思考内容 → thinking 块（reasoning 模型要求回传 reasoning_content）。
			text := p.Text
			if text == "" && len(p.Summary) > 0 {
				text = extractResponsesReasoningText(p.Summary)
			}
			if text != "" {
				cm.Content = append(cm.Content, public.CanonicalContent{Type: "thinking", Text: text})
			}
		}
	}

	// Chat 风格 assistant 消息内嵌 tool_calls：{"role":"assistant","tool_calls":[...]}。
	if item.Role == "assistant" {
		for _, tc := range item.ToolCalls {
			cm.ToolCalls = append(cm.ToolCalls, public.CanonicalToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: marshalString(tc.Function.Arguments),
			})
			cm.Content = append(cm.Content, public.CanonicalContent{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: parseArgsToInput(tc.Function.Arguments),
			})
		}
	}

	return cm
}

func canonicalMessageToResponsesItem(cm public.CanonicalMessage) []responsesItem {
	if cm.Role == "tool" {
		items := make([]responsesItem, 0, len(cm.Content))
		for _, b := range cm.Content {
			if b.Type != "tool_result" {
				continue
			}
			items = append(items, responsesItem{
				Type:   "function_call_output",
				CallID: firstNonEmpty(b.ID, cm.ToolCallID),
				Output: marshalString(blocksToText(b.Content)),
			})
		}
		if len(items) == 0 {
			items = append(items, responsesItem{
				Type:   "function_call_output",
				CallID: cm.ToolCallID,
				Output: marshalString(blocksToText(cm.Content)),
			})
		}
		return items
	}

	// assistant 带 tool_use → function_call item；否则 message item。
	parts := make([]responsesContentPart, 0, len(cm.Content))
	for _, b := range cm.Content {
		switch b.Type {
		case "text":
			parts = append(parts, responsesContentPart{Type: "input_text", Text: b.Text})
		case "image":
			parts = append(parts, responsesContentPart{Type: "input_image", ImageURL: b.ImageURL})
		case "video":
			parts = append(parts, responsesContentPart{Type: "input_video", VideoURL: b.VideoURL})
		case "thinking":
			// 保留思考内容（reasoning_content），reasoning 模型要求回传。
			parts = append(parts, responsesContentPart{Type: "summary_text", Text: b.Text})
		}
	}

	var items []responsesItem
	if len(parts) > 0 || len(cm.ToolCalls) == 0 {
		items = append(items, responsesItem{
			Type:    "message",
			Role:    cm.Role,
			Content: marshalResponsesParts(parts),
		})
	}
	for _, tc := range cm.ToolCalls {
		items = append(items, responsesItem{
			Type:      "function_call",
			CallID:    tc.ID,
			Name:      tc.Name,
			Arguments: rawToString(tc.Arguments),
		})
	}
	return items
}

func marshalResponsesItems(items []responsesItem) json.RawMessage {
	b, _ := json.Marshal(items)
	return b
}

func marshalResponsesParts(parts []responsesContentPart) json.RawMessage {
	b, _ := json.Marshal(parts)
	return b
}

func responsesStatusToCanonical(status string) string {
	switch status {
	case "completed", "":
		return "stop"
	case "incomplete":
		return "length"
	default:
		return status
	}
}

// responsesFinishReason 计算非流式响应的完成原因：completed 且 output 含
// function_call 时归一为 tool_calls。
func responsesFinishReason(resp responsesResponse) string {
	if resp.Status == "completed" {
		for _, item := range resp.Output {
			if item.Type == "function_call" {
				return "tool_calls"
			}
		}
	}
	return responsesStatusToCanonical(resp.Status)
}

func canonicalFinishToResponsesStatus(fr string) string {
	switch fr {
	case "stop", "":
		return "completed"
	case "length":
		return "incomplete"
	default:
		return fr
	}
}

func responsesUsageToCanonical(u responsesUsage) public.CanonicalUsage {
	return public.CanonicalUsage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalTokens:  u.TotalTokens,
		CachedTokens: u.InputTokensDetails.CachedTokens,
	}
}

func canonicalToResponsesUsage(cu public.CanonicalUsage) responsesUsage {
	u := responsesUsage{
		InputTokens:  cu.InputTokens,
		OutputTokens: cu.OutputTokens,
		TotalTokens:  cu.TotalTokens,
	}
	u.InputTokensDetails.CachedTokens = cu.CachedTokens
	return u
}

func extractResponsesReasoningText(summary json.RawMessage) string {
	parts := parseResponsesContentParts(summary)
	var text string
	for _, p := range parts {
		if p.Text != "" {
			text += p.Text
		}
	}
	return text
}

// hasThinkingBlock 判断 content 中是否已含 thinking 块。
func hasThinkingBlock(content []public.CanonicalContent) bool {
	for _, b := range content {
		if b.Type == "thinking" {
			return true
		}
	}
	return false
}

// hasToolCallID 判断工具调用列表里是否已存在指定 ID 的调用。
func hasToolCallID(tcs []public.CanonicalToolCall, id string) bool {
	for _, tc := range tcs {
		if tc.ID == id {
			return true
		}
	}
	return false
}

// lookupToolName 在工具定义里按名称查找工具名（用于 function_call 缺 name 时的兜底）。
// Responses 的 function_call item 理论上应自带 name，但某些客户端可能漏传；
// 此时从请求携带的 tools 定义里找第一个 function 类型工具作为兜底，避免下游
// Chat 请求因 function.name 为空被 go-openai omitempty 掉导致后端 400。
// 注意：这是兜底逻辑，正常路径应优先用 item.Name。
func lookupToolName(tools []public.CanonicalTool, callID, itemType string) string {
	// 优先按工具类型推断默认名（与 web_search_call / custom_tool_call 分支保持一致）。
	switch itemType {
	case "web_search_call":
		return "web_search"
	case "custom_tool_call":
		return "custom"
	}
	// 从工具定义里取第一个 function 工具名作为兜底。
	for _, t := range tools {
		if t.Type == "function" || t.Type == "" {
			if t.Name != "" {
				return t.Name
			}
		}
	}
	return ""
}

// extractToolCallArgs 从 web_search_call / custom_tool_call 还原工具调用参数字符串。
// 优先用 item.Input（custom_tool_call 的 Codex 规范字段）或 item.Arguments；
// 否则用 item.Action 的紧凑 JSON（如 {"type":"search","query":"..."}）；
// 都没有则回退为空对象 {}。
func extractToolCallArgs(item responsesItem) string {
	// custom_tool_call 用 input 字段（Codex 规范）。
	if item.Type == "custom_tool_call" && item.Input != "" {
		return item.Input
	}
	if item.Arguments != "" {
		return item.Arguments
	}
	if len(item.Action) > 0 {
		if b := compactJSON(item.Action); len(b) > 0 {
			return string(b)
		}
	}
	return "{}"
}
