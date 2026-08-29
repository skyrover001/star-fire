# Codex 接入本地网关：三类工具/消息问题深度定位

- **日期**: 2026-08-25
- **Codex CLI**: `0.149.0-alpha.4.3`
- **本地网关**: `http://localhost:8080/v1`（OpenAI 兼容，底层 vLLM 0.26.0）
- **接入方式**: `~/.codex/config.toml` `wire_api = "responses"`

---

## 背景

前一轮全面测试发现三类问题，本报告通过受控 HTTP 实验逐一**精确定位触发条件与根因**。

---

## 问题 1：web_search 结果回传 — 全部模型失败

### 触发场景
Codex 让模型搜索 → 网关返回 `web_search_call` → Codex 执行搜索 → 把结果作为 `web_search_call` item（含 `output`）回传网关 → 网关转下游失败。

### 定位实验
| 实验 | 结果 |
|---|---|
| `web_search_call` + output（搜索结果）回传 | ❌ HTTP 400 |
| `web_search_call` 空 output 回传 | ❌ HTTP 400（同样报错） |

### 具体报错
```
19 validation errors:
  {'loc': ('body', 'messages', 1, 'ChatCompletionUserMessageParam', 'content'),
   'msg': 'Input should be a valid string/iterable', 'input': None}
  ... 其它 role 同类型错误 ...
```

### 根因
网关不支持把 Responses 的 **`web_search_call`** 输入 item 转成下游 Chat 消息。转换后生成的 `messages[1]` 是 `content: null` 的畸形消息，被下游 Chat 校验拒绝（体现为 19 项角色/内容校验全过不了）。

**影响**：所有 5 个模型都无法完成"搜索 → 拿结果 → 作答"的完整流程。Codex 显示 `web search:` 调用，但 `tokens used 0`（结果回传失败，拿不到搜索内容）。

---

## 问题 2：deepseek-v4-pro / deepseek-v4-flash 工具回传 — thinking 模式特有

### 触发场景
这两个**思考型**模型，在**工具结果回传**后继续生成时报错。

### 定位实验
| 实验 | 结果 |
|---|---|
| 纯单轮对话（reasoning effort=high） | ✅ 200 |
| 纯多轮对话（无工具，user/assistant/user） | ✅ 200 |
| **工具结果回传（function_call + function_call_output）** | ❌ 400 |
| 工具回传 + `reasoning={effort:"minimal"}` | ❌ 400 |

### 具体报错
```
The `reasoning_content` in the thinking mode must be passed back to the API.
```

### 根因
DeepSeek 思考模型的机制：模型在 thinking 模式下产生 `reasoning_content`（思维链）。当**后续继续生成**（如收到工具结果后继续作答）时，**上一轮的 `reasoning_content` 必须随请求回传给 API**，否则报错。

网关在把 Codex 的工具循环（`function_call` → 客户端执行 → `function_call_output` 回传 → 模型继续）转成下游 Chat 时：
- 把工具调用/回传消息转发给了下游思考模型
- **但没有附带之前生成的 `reasoning_content`**

于是下游思考模型要求回传 reasoning_content 时缺失 → 400。

**为何非思考模型正常**：DeepSeek-V4 / GLM-5.2 / Qwen 不产生/不要求 reasoning_content 回传，所以 function 工具回传正常（`2+2=4`）。

**为何纯多轮正常**：纯对话续写由网关内部状态处理，未触发"必须回传 reasoning_content"的校验路径；而工具回传路径没做这个处理。

---

## 问题 3：Qwen3.5-397B-A17B system 消息位置

### 触发场景
Codex 发**多条 `developer`（系统/技能/权限/主代理指令）**消息给 Qwen 时报错；单条 developer 正常。

### 定位实验
| 实验 | 结果 |
|---|---|
| 1 条 `instructions` + user | ✅ 200 |
| 1 条 `developer` + user | ✅ 200 |
| **2 条 `developer` + user** | ❌ 400 |

### 具体报错
```
System message must be at the beginning.
```

### 根因
Qwen 后端要求 **system 消息必须是 messages 数组的第一条**。网关把 Codex 的多条 `developer` 消息各自转成了多条 `system` 消息：
- 单条 developer → 一条 system 在首位 → 通过
- 两条 developer → 第二条 system 落在数组非首位 → Qwen 报"system 必须在开头"

**触发条件**：`developer`（或 system/instructions 组合）超过一条时触发。

---

## 汇总对照表

| # | 问题 | 影响模型 | 触发条件 | 报错 | 根因 |
|---|---|---|---|---|---|
| 1 | web_search 结果回传失败 | 全部 5 模型 | `web_search_call` input item | `content: None` (19 校验错) | 网关不支持 `web_search_call` 转下游消息 |
| 2 | 工具回传+thinking 报错 | pro / flash | 工具结果回传（思考模型） | `reasoning_content must be passed back` | 思考模型工具续写未回传 reasoning_content |
| 3 | system 消息位置 | Qwen | 2+ 条 developer/system | `System message must be at the beginning` | 多条 system 未合并/未放首位 |

---

## 修复建议

### 问题 1：web_search 结果回传
网关需支持把 Responses 的 **`web_search_call` input item**（含搜索结果的 `output`）转成下游 Chat 工具消息（带正确 `content`），而不是生成 `content:null`。参考 OpenAI 约定：搜索调用结果作为带内容的工具结果回传。

### 问题 2：pro/flash 思考模型工具回传
网关在思考模型的**工具循环**中，须把上一轮模型的 `reasoning_content`（思维链）随后续请求回传给下游 API；或对思考模型在工具回传场景做专门处理（DeepSeek 官方要求推理内容须随上下文继续回传）。

### 问题 3：Qwen system 位置
网关归一化角色时，把多条 `developer`/system 消息**合并为一条 system**（置于 messages 首位），避免出现非首位的 system。可对要求严格的后端统一应用。

---

## 附
- `E:\Study\test\codex-full-test-report.md` — 全面测试矩阵
- 相关：`codex-tool-transparency.md`、`codex-developer-role-issue.md`
