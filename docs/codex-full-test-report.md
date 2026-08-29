# Codex 接入本地网关：全面测试报告（第二轮）

- **日期**: 2026-08-25
- **Codex CLI**: `0.149.0-alpha.4.3`
- **本地网关**: `http://localhost:8080/v1`（OpenAI 兼容，底层 vLLM 0.26.0）
- **接入方式**: `~/.codex/config.toml` `wire_api = "responses"`
- **测试范围**: 全部 5 个模型的 基础对话 / web_search / function 工具（两段式）

---

## 一、测试方法

1. **基础对话**：Codex CLI `exec`（`Reply with exactly: PONG`），判定 stdout 是否出现 PONG 且无网关错误。
2. **web_search 第一段**：HTTP 直连网关，请求带 `web_search` 工具，观察模型是否决定调用、返回什么类型。
3. **功能工具两段式**：直接 POST 会话，构造 `function_call` + `function_call_output`（含工具结果回传），观察模型能否继续作答。
4. **web_search 结果回传（第二段）**：构造 `web_search_call` item（含搜索结果），观察网关能否转下游并让模型作答。

---

## 二、总结果矩阵

### 2.1 基础对话
| 模型 | 基础对话 |
|---|---|
| DeepSeek-V4 | ✅ |
| GLM-5.2 | ✅ |
| deepseek-v4-pro | ✅ |
| deepseek-v4-flash | ✅ |
| Qwen3.5-397B-A17B | ❌ `System message must be at the beginning` |

### 2.2 web_search 工具
| 环节 | 所有 5 模型 |
|---|---|
| 工具定义透传（模型识别 web_search） | ✅ |
| 模型决定调用 → 返回标准 `web_search_call` | ✅（已修复，原为 function_call） |
| **搜索结果回传（第二段）→ 模型作答** | ❌ 全部 400 |

### 2.3 function 工具（两段式：function_call + 结果回传）
| 模型 | 结果 |
|---|---|
| DeepSeek-V4 | ✅ `The result of 2+2 is 4` |
| GLM-5.2 | ✅ `The result of 2+2 is 4` |
| Qwen3.5-397B-A17B | ✅ `The result of 2+2 is 4` |
| deepseek-v4-pro | ❌ `reasoning_content ... must be passed back` |
| deepseek-v4-flash | ❌ `reasoning_content ... must be passed back` |

---

## 三、已确认修复/正常的能力

1. **web_search 工具返回类型已修复**：从错误的 `function_call name=web_search` 修正为标准 `web_search_call`。Codex 不再报 `unsupported call: web_search`。
2. **工具定义透传正常**：Codex 发的 `web_search` 等工具能被模型识别。
3. **4 个模型基础对话正常**。

---

## 四、三类现存问题（根因分析）

### 问题 1：web_search 结果回传 — 全部模型失败

**表现**：模型决定搜索 → 返回 `web_search_call` → Codex 执行搜索 → 把结果回传给网关 → **网关转下游 Chat 消息失败** → 模型拿不到搜索结果 → 无答案（Codex 显示 `web search:` 但 `tokens used 0`）。

**报错**（4 模型空 400；pro/flash 具体）：
```
Failed to deserialize the JSON body ... messages[1]: missing field `content`
```

**根因**：网关不支持把 Responses 的 `web_search_call` item（含搜索结果 output）转成下游 Chat 消息，生成了缺 `content` 的畸形消息。

### 问题 2：pro / flash 的 function 工具回传 — thinking 模式特有

**表现**：这两个**思考型**模型，纯对话正常（`reasoning=minimal` 回答 OK），但**工具结果回传 + thinking 模式**时报错。

**报错**：
```
The `reasoning_content` in the thinking mode must be passed back to the model when continuing generation
```

**根因**：DeepSeek 思考模型的机制要求——**对话继续生成（如工具结果回传后）必须把上一轮的 `reasoning_content`（思维链）带回来**。网关在把 Codex 的工具循环（function_call + 结果回传 → 继续生成）转成下游 Chat 时，**没有回传 reasoning_content**，导致 pro/flash 报错。而非思考模型（DeepSeek-V4/GLM/Qwen）无此要求，所以正常。

### 问题 3：Qwen 基础对话 — system 消息位置

**表现**：`System message must be at the beginning`。

**根因**：网关把 `developer`/`system` 消息放在了非首位，或 Qwen 后端对 system 消息在 `messages` 数组中的位置有严格要求（必须是第一条）。Codex 发多段 developer（系统/技能/权限）消息，网关转换后某条 system 落到了中间位置。

---

## 五、结论

| # | 能力 | 状态 | 根因 |
|---|---|---|---|
| 1 | 基础对话（4/5 模型） | ✅ | — |
| 2 | web_search 工具定义 + 返回调用决策 | ✅ | 已修复 |
| 3 | web_search 结果回传 | ❌ 全模型 | 网关不支持 `web_search_call` 转下游消息 |
| 4 | function 工具回传 | ✅ 非思考型 / ❌ pro+flash | thinking 模式未回传 `reasoning_content` |
| 5 | Qwen 基础对话 | ❌ | system 消息位置错误 |

---

## 六、修复建议

**问题 1（web_search 结果回传）**：网关需支持把 Responses 的 `web_search_call`（含 output/搜索结果的 item）正确转成下游 Chat 工具消息（带 content），而非生成缺 content 的畸形消息。参考 OpenAI：搜索结果作为 `web_search_call` → 下游应为带内容的 tool/消息。

**问题 2（pro/flash thinking 工具回传）**：网关对思考型模型做**多轮工具循环**时，须把上一轮模型的 `reasoning_content`（思维链）随对话历史回传；或在该模型的工具回传场景下调低/关闭 thinking。参照 DeepSeek 官方思考模式文档（推理内容需随上下文继续回传）。

**问题 3（Qwen system 位置）**：网关归一化 role 时，确保 system/developer 转换后的 system 消息**排在 messages 数组首位**（或与该模型后端要求一致），不能出现 system 在中间。

---

## 附：相关文件
- `E:\Study\test\codex-tool-transparency.md` — 工具透传问题报告
- `E:\Study\test\codex-developer-role-issue.md` — developer role 早期报告（已过时）
- `C:\Users\skyro\AppData\Local\Temp\opencode\full_test.ps1` — 综合测试脚本
- `C:\Users\skyro\AppData\Local\Temp\opencode\tool_deep.ps1` — 工具深度测试脚本
- `C:\Users\skyro\AppData\Local\Temp\opencode\thinking_test.ps1` — thinking 模式测试脚本
