# 本地网关 Responses 流式 SSE 规范事件序列修复清单

> 目标：让 `http://localhost:8080` 网关的 `POST /v1/responses`（`stream: true`）输出符合
> OpenAI **Responses API** SSE 规范的事件序列，从而能被 **Codex CLI**（`~/.codex`，`wire_api = "responses"`）正常消费。
>
> 测试工具：Codex CLI `0.149.0-alpha.4.3`
> 网关现状：`vLLM` 类实现（系统指纹 `vllm-0.26.0-*`）

---

## 一、问题现象（实测）

Codex CLI 通过网关 `wire_api = "responses"` 发起流式请求后失败：

```
model: DeepSeek-V4
provider: local
ERROR codex_core::util: OutputTextDelta without active item
ERROR: Reconnecting... 1/5 ... 5/5
ERROR: stream disconnected before completion: stream closed before response.completed
```

直接抓取网关 `POST /v1/responses {stream:true}` 的响应体，**当前完整输出只有**：

```http
data: {"delta":"<文本>","type":"response.output_text.delta"}

data: [DONE]
```

（且不稳定：多模型/多请求时部分请求连 `delta` 都没有，直接 `data: [DONE]`；`deepseek-v4-pro` 还偶发 `503`。）

**结论**：网关只实现了 Responses **非流式**的响应结构 + 流式的**部分文本增量事件**，缺失了 OpenAI 规范要求的一整套**结构性事件**（`response.created`、`output_item.added`、`content_part.added/done`、`output_item.done`、`response.completed`）。

---

## 二、OpenAI Responses stream 规范事件序列

`stream: true` 时，服务端应发送以下事件（按序）。每个事件一行，格式 `data: <JSON>`，事件间以空行分隔，最终以 `data: [DONE]` 结束。

### 最小必需序列（单条纯文本回复）

```
(data: 事件1)  response.created                            ← 会话开始，含完整 response 骨架
(data: 事件2)  response.output_item.added   {type: message}
(data: 事件3)  response.content_part.added  {type: output_text}
(data: ...)    response.output_text.delta   {delta: "..."}   ← 0..N 次，文本增量
(data: ...)    response.output_text.done              ← 文本部分结束
(data: ...)    response.content_part.done
(data: ...)    response.output_item.done      {type: message}
(data: ...)    response.completed             ← 会话完成，客户端等它（关键）
data: [DONE]
```

### 各事件负载字段

以下 JSON 结构与官方 Responses 一致（非流式响应里的对象就是这些事件的累积结果）。

**1) `response.created`**
```json
{
  "type": "response.created",
  "response": {
    "id": "resp_xxx",
    "object": "response",
    "created_at": 1787627578,
    "status": "in_progress",
    "model": "DeepSeek-V4",
    "output": [],
    "usage": null,
    "instructions": null,
    "max_output_tokens": null,
    "temperature": null,
    "tool_choice": null,
    "tools": [],
    "parallel_tool_calls": true,
    "top_p": null,
    "metadata": {},
    "reasoning": null
  }
}
```

**2) `response.output_item.added`**
```json
{
  "type": "response.output_item.added",
  "output_index": 0,
  "item": {
    "id": "msg_xxx",
    "type": "message",
    "status": "in_progress",
    "role": "assistant",
    "content": []
  }
}
```

**3) `response.content_part.added`**
```json
{
  "type": "response.content_part.added",
  "item_id": "msg_xxx",
  "output_index": 0,
  "content_index": 0,
  "part": {
    "type": "output_text",
    "text": "",
    "annotations": []
  }
}
```

**4) `response.output_text.delta`** （0..N 次）
```json
{
  "type": "response.output_text.delta",
  "item_id": "msg_xxx",
  "output_index": 0,
  "content_index": 0,
  "delta": "Hello "
}
```

**5) `response.output_text.done`**
```json
{
  "type": "response.output_text.done",
  "item_id": "msg_xxx",
  "output_index": 0,
  "content_index": 0,
  "text": "Hello world"
}
```

**6) `response.content_part.done`**
```json
{
  "type": "response.content_part.done",
  "item_id": "msg_xxx",
  "output_index": 0,
  "content_index": 0,
  "part": {
    "type": "output_text",
    "text": "Hello world",
    "annotations": []
  }
}
```

**7) `response.output_item.done`**
```json
{
  "type": "response.output_item.done",
  "output_index": 0,
  "item": {
    "id": "msg_xxx",
    "type": "message",
    "status": "completed",
    "role": "assistant",
    "content": [
      { "type": "output_text", "text": "Hello world", "annotations": [] }
    ]
  }
}
```

**8) `response.completed`**（客户端同步点，**必发**）
```json
{
  "type": "response.completed",
  "response": {
    "id": "resp_xxx",
    "object": "response",
    "created_at": 1787627578,
    "status": "completed",
    "model": "DeepSeek-V4",
    "output": [
      {
        "id": "msg_xxx",
        "type": "message",
        "status": "completed",
        "role": "assistant",
        "content": [
          { "type": "output_text", "text": "Hello world", "annotations": [] }
        ]
      }
    ],
    "usage": {
      "input_tokens": 7,
      "output_tokens": 5,
      "total_tokens": 12,
      "input_tokens_details": { "cached_tokens": 0 }
    }
  }
}
```

**9) 结束**
```
data: [DONE]
```

---

## 三、Codex CLI 消费的关键点（务必满足）

1. **必须有 `response.completed`**。Codex 报错 `stream closed before response.completed`，说明它以收到 `response.completed` 作为成功完成的标志。缺失 → 直接失败重连。

2. **`response.output_text.delta` 之前必须有 active item**。Codex 报 `OutputTextDelta without active item`，说明它依赖 `response.created` + `response.output_item.added` + `response.content_part.added` 先建立上下文，再接收 delta。你网关现在直接发 delta（没有前面的 created/item/part 事件），所以 Codex 报这个错。

3. **事件顺序**：created → item.added → part.added → (delta ×N) → part.done → item.done → completed，内部字段（`item_id`、`output_index`、`content_index`）前后必须一致，Codex 用它们关联增量与 item。

4. **`[DONE]` 必须放在最后**，仅在 `response.completed` 之后。

---

## 四、与 OpenAI Chat Completions 的对接建议（若想最省事）

你的网关已经有一套**成熟的 Chat Completions 流式**（`/v1/chat/completions` `stream:true` 完整返回 `choices[].delta` + `finish_reason`）。若不想大改内核，可在网关的 Responses 流式**适配层**里做事件翻译：

```
Chat Completions 流式事件                     →  Responses 流式事件
-------------------------------------------    -------------------------------
(请求开始，已知 model/request)                  response.created（合成 skeleton）
data: {choices:[{delta:{role:"assistant"}}]}   response.output_item.added (message)
data: {choices:[{delta:{content:"Hello "}}]}   response.content_part.added (output_text, 空)
                                               response.output_text.delta {delta:"Hello "}
data: {choices:[{delta:{content:"world"}}]}    response.output_text.delta {delta:"world"}
(finish_reason:"stop" 到来)                    response.output_text.done {text:"Hello world"}
                                               response.content_part.done
                                               response.output_item.done
                                               response.completed（含完整 output + usage）
data: [DONE]                                   data: [DONE]
```

即：**在两种协议之间加一层状态机翻译**，把 Chat 流式事件累积成 Responses 事件序列。这是对现有 vLLM 网关改动最小的方案。

---

## 五、稳定性要求（实测暴露）

1. **不要出现"只发 `[DONE]` 不发任何事件"**。必须保证至少 `response.created` → `response.completed` 总是出现；文本为空也照发这两个。
2. **`deepseek-v4-pro` 偶发 503**：需检查该通道的客户端/负载，避免流式中断。
3. **`usage` 在 `response.completed` 里给全**（`input_tokens`/`output_tokens`/`total_tokens`），Codex 会读。

---

## 六、验证方式（改完后）

改完网关后，用任何 HTTP 客户端验证事件序列：

```
POST http://localhost:8080/v1/responses
Authorization: Bearer <key>
Content-Type: application/json
Accept: text/event-stream

{ "model": "DeepSeek-V4", "input": "Say hi", "stream": true }
```

期望输出应包含（顺序）：
```
data: {"type":"response.created","response":{"status":"in_progress",...}}

data: {"type":"response.output_item.added","item":{"type":"message",...}}

data: {"type":"response.content_part.added","part":{"type":"output_text",...}}

data: {"type":"response.output_text.delta","delta":"..."}

data: {"type":"response.output_text.done","text":"..."}

data: {"type":"response.content_part.done","part":{"type":"output_text","text":"..."}}

data: {"type":"response.output_item.done","item":{"type":"message","status":"completed",...}}

data: {"type":"response.completed","response":{"status":"completed","output":[...],"usage":{...}}}

data: [DONE]
```

然后跑 Codex CLI 验证：
```powershell
$env:CODEX_HOME = "C:\Users\skyro\.codex"
$env:STARFIRE_API_KEY = "sk-xxx"   # 网关 key（当前在 config.toml 的 experimental_bearer_token）
"Reply with exactly: PONG" | & "C:\Users\skyro\AppData\Local\OpenAI\Codex\bin\8fffe69425752027\codex.exe" exec -m DeepSeek-V4 --skip-git-repo-check
```

不再出现 `Response stream closed before response.completed` 即成功。
