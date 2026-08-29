# deepseek-v4-flash / deepseek-v4-pro 接入 Codex 失败问题分析报告

- **日期**: 2026-08-25
- **Codex CLI**: `0.149.0-alpha.4.3`
- **本地网关**: `http://localhost:8080/v1`（OpenAI 兼容，底层 vLLM 0.26.0）
- **接入方式**: `~/.codex/config.toml` `wire_api = "responses"`

---

## 〇、进展更新（developer 已修复，真凶是 tools 映射）

> 本报告最初定位的问题是 `role: developer` 不被后端接受。经网关修复后，`developer` 报错已消失，但重新测试暴露了**真正的根因：网关对这两个模型的下游 `tools` 转换 bug**。本节是最新结论，以下各部分为初始的 developer 分析（已过时部分仅作历史参考）。

### 最新根因（决定性）

网关在把 Responses 请求转成这两个模型的**下游 Chat 调用**时，把**没有 `name` 的非 function 工具**（尤其是 Codex 专有的 `tool_search`，以及 `web_search`）当作普通 `function` 透传，导致下游 `tools[n].function.name` 为空字符串，被后端校验拒绝：

```
Invalid 'tools[13].function.name': empty string. Expected a string with minimum length 1, but got an empty string instead.
```

### 决定性对比实验（完全相同请求，仅 model 不同）

| 测试 | 结果 |
|---|---|
| `DeepSeek-V4` + 完整 15 工具 | ✅ HTTP 200 |
| `deepseek-v4-pro` + 完整 15 工具 | ❌ HTTP 400 `tools[13].function.name empty` |
| `deepseek-v4-pro` + **空 tools** | ✅ HTTP 200 |
| `deepseek-v4-pro` + 标准 `web_search`（无 function 包裹） | ❌ HTTP 400 `tools[0].function.name empty` |
| `DeepSeek-V4` + 标准 `web_search` | ✅ HTTP 200 |

### 结论

- **不是** `developer`（已修复）
- **不是** 配额、模型本身、Codex 配置、流式/非流式（pro + 空 tools 即成功）
- **是** 网关对这两个模型**下游 tools 的映射/归一无做**：把 `tool_search`、`web_search` 等非 function 类型错误地套成 `function` 并要求 `name`
- `DeepSeek-V4`/`GLM-5.2`/`Qwen` 之所以成功：它们的后端正确处理了这些工具类型

### 修复要求：**必须映射，不能删除**

这些工具是 Codex 的功能来源，**不能删**：
- `web_search`：Codex 的联网搜索能力 → 应映射为下游 Chat 的 `web_search` 工具（`{"type":"web_search"}`），或按后端能力透传/等价映射
- `tool_search`：Codex 的 deferred tool discovery 机制 → 应**识别并处理为 Codex 专有类型**（不传给只认 function 的旧后端，或按网关对 Codex 的协议做等价映射）
- `custom apply_patch`：Codex 的代码编辑工具 → 需保留
- `namespace collaboration`：Codex 多代理协作 → 需保留

正确做法是**按工具类型逐一映射/归一**到下游 Chat 支持的形式（参照 DeepSeek-V4 等模型的后端对这几类工具的处理），而不是简单丢弃。否则 Codex 会丢失搜索、工具发现、编辑、多代理等能力。

**修复目标**：对 `deepseek-v4-flash`/`deepseek-v4-pro`，让其下游 tools 转换逻辑与 `DeepSeek-V4`/`GLM-5.2`/`Qwen` 保持一致——把非 function 工具正确映射（web_search → 搜索工具、tool_search → Codex 专有处理、custom/namespace → 保留等价形式），并保证所有 function 工具都带合法 `name`。

---

## 一、结论摘要

- 5 个本地模型中，`DeepSeek-V4` / `GLM-5.2` / `Qwen3.5-397B-A17B` 已能通过 Codex CLI 正常对话。
- **`deepseek-v4-flash` / `deepseek-v4-pro` 无法使用**，根因是**网关**在把 Responses 请求转成内部 Chat 消息时未处理 `role: developer`，而这两个模型映射到的后端只认 `system/user/assistant/tool` 四种角色。
- 该问题**不是配额、不是流式/非流式、不是 Codex 配置问题**，而是网关的 `developer` role 映射缺失。

---

## 二、测试方法（如何抓到的证据）

由于 Codex 的错误只给了摘要，为拿到完整请求/响应，我在网关前部署了一个**本地捕获代理**（`127.0.0.1:8081` → 转发到 `localhost:8080`），并通过 `codex -c 'model_providers.local.base_url="http://127.0.0.1:8081/v1"'` 让 Codex 走代理，从而记录 Codex 真实发出的完整 HTTP 请求与网关返回的 400 响应。

触发命令（与直接测试一致）：
```powershell
"Reply with exactly: OK" | & codex exec -m deepseek-v4-pro --skip-git-repo-check
"Reply with exactly: OK" | & codex exec -m deepseek-v4-flash --skip-git-repo-check
```

---

## 三、实战输入（Codex 发给网关的完整请求，约 32KB）

`POST /v1/responses`，`Authorization: Bearer sk-...`。关键负载如下（两模型结构一致，仅 `model` 与消息 id 不同）：

```json
{
  "model": "deepseek-v4-pro",                  // flash 时 = "deepseek-v4-flash"
  "instructions": "You are Codex, an agent based on GPT-5...",
  "input": [
    { "type": "message", "role": "developer",
      "content": [ {"type": "input_text", "text": "<skills_instructions>..."} ] },
    { "type": "message", "role": "developer",
      "content": [ {"type": "input_text", "text": "<permissions instructions>..."} ] },
    { "type": "message", "role": "developer",
      "content": [ {"type": "input_text", "text": "You are `/root`, the primary agent..."} ] },
    { "type": "message", "role": "user",
      "content": [ {"type": "input_text", "text": "<environment_context>..."} ] },
    { "type": "message", "role": "user",
      "content": [ {"type": "input_text", "text": "Reply with exactly: OK"} ] }
  ],
  "tools": [ exec_command, apply_patch, web_search, ... ],
  "tool_choice": "auto",
  "reasoning": { "effort": "max" },
  "stream": true,
  "include": ["reasoning.encrypted_content"]
}
```

要点：
- `input` 数组含 **3 条 `role: "developer"`** 消息（Codex 用 developer 角色承载系统/技能/权限指令）。
- 发往网关的是**标准的 OpenAI Responses 格式**（合法），网关应能解析。
- 错误发生在网关**内部**把 Responses `input` 转成下游 Chat `messages` 之后。

---

## 四、网关返回的错误（实际输出）

两模型的 400 响应结构相同，仅错误字段的 `column` 号不同（pro=2761，flash=2763）。

```json
{
  "error": {
    "code": "invalid_request_error",
    "message": "create chat complation error: error, status code: 400, status: 400 Bad Request,
                 message: Failed to deserialize the JSON body into the target type:
                 messages[1].role: unknown variant `developer`,
                 expected one of `system`, `user`, `assistant`, `tool`, `latest_reminder`
                 at line 1 column 2761",
    "param": null,
    "type": "invalid_request_error"
  }
}
```

关键片段：`messages[1].role: unknown variant 'developer'`——指下游 Chat 消息数组的第 2 条（索引 1）role 为 `developer`，不被后端接受。

---

## 五、根因分析

1. **Codex 规范行为**：Codex 按 OpenAI 规范，用 `role: "developer"` 发送系统/技能/权限等指令，这是**正确且被 OpenAI Responses API 支持**的（DeepSeek 官方文档亦明确支持 developer 角色，见下）。

2. **网关转换缺失**：网关收到 Responses 请求后，需要把 `input` 里的 message 转成下游 Chat API 的 `messages`，每个 item 的 `role` 原样保留。转换后第 2 条消息的 role 仍是 `developer`。

3. **后端角色枚举限制**：`deepseek-v4-flash` / `deepseek-v4-pro` 映射到的上游后端，其 Chat 消息 `role` 只接受 `system | user | assistant | tool | latest_reminder`，不接受 `developer`，反序列化即报错。

4. **为何另外 3 个模型成功**：`DeepSeek-V4` / `GLM-5.2` / `Qwen3.5-397B-A17B` 映射到的后端要么接受 `developer` 角色，要么网关对它们做了 `developer → system` 之类的转换。说明网关的 role 映射是按模型/后端分别配置的，唯独这两个模型漏配。

5. **排除的因素**：
   - ❌ 不是配额——直接 HTTP 调用（非 Codex）时这两个模型能正常返回结果。
   - ❌ 不是流式/非流式——非流式直接调用正常，报错发生在请求解析阶段（Chat 反序列化），与响应传输方式无关。
   - ❌ 不是 Codex 配置/脚本——配置经 `codex doctor` 验证加载正常，且另外 3 个模型走同一套配置成功。

---

## 六、对照 DeepSeek 官方 Responses API 做法

依据 DeepSeek 官方文档《使用 Responses API》：

> 输入 `message` item：角色支持 `user` / `assistant` / `system` / **`developer`（`developer` 视同 `user`）**。

即官方**明确接受并处理 `developer` 角色**（语义上视同 user，实质上是开发指令）。这让 Codex 无需任何改动即可接入。

**对本地网关的建议**：在 Responses→Chat 转换层对角色做归一化，两选一（推荐前一种，语义更贴近）：
- `developer → system`（把开发者指令当作系统指令），或
- `developer → user`（DeepSeek 官方做法）

统一应用到 `deepseek-v4-flash` / `deepseek-v4-pro` 两个模型即可。

---

## 七、结论与交付物

### 根因一句话
> 网关在 Responses→内部 Chat 转换时未把 `role: developer` 映射为后端可接受的角色，导致 `deepseek-v4-flash`/`deepseek-v4-pro` 的下游后端返回 400 `unknown variant 'developer'`。

### 待网关处理
1. 对这两个模型补 `developer` 角色映射（→ system 或 → user）。
2. 建议对所有模型统一该转换，避免后续新增模型再踩同一坑。

### 修复后期望
```
model: deepseek-v4-pro   → 可用
model: deepseek-v4-flash → 可用
```

### 相关文件
- `E:\Study\test\codex-local-gateway-report.md` — 总报告
- `E:\Study\test\responses-stream-spec-fix.md` — 流式事件序列修复清单
- 本报告侧重点：`developer` role 映射缺失
