<script lang="ts" setup>
import { useRouter } from 'vue-router';
import { preferences } from '@vben/preferences';
import { $t } from '#/locales';
import { marked } from 'marked';

const router = useRouter();

const goBack = () => {
  router.push('/model-marketplace');
};

// 判断当前语言
const isEn = preferences.app.locale === 'en-US';

// 贡献者使用说明 (Markdown)
const contributorGuideZh = `
# 贡献者使用说明

## 1. 注册与登录

使用邮箱在 Star Fire 平台注册并登录。

## 2. 分享模型

### 方式一：PC 客户端（推荐）

目前支持 macOS 和 Windows 图形客户端，详见 [Release](https://github.com/star-fire/releases) 页面。

- 支持使用用户名密码登录，自动注册和发现模型
- 支持自定义模型价格，实时调整
- 支持实时查看收益
- 支持混合推理引擎运行

### 方式二：命令行模式

**步骤 1：下载客户端**

从 [Release](https://github.com/star-fire/releases) 下载对应平台的客户端，或本地编译：

\`\`\`bash
make client
\`\`\`

编译产物在 \`build/client\` 目录下。

**步骤 2：获取注册 Token**

在模型广场页面点击 **"注册到 Star Fire"**，获取注册 Token。

**步骤 3：注册客户端**

Windows：

\`\`\`bash
starfire.exe -host {host} -token {register token} -ippm {input price} -oppm {output price}
\`\`\`

macOS / Linux：

\`\`\`bash
starfire -host {host} -token {register token} -ippm {input price} -oppm {output price}
\`\`\`

参数说明：
- \`host\`：服务器地址
- \`token\`：注册 Token（单次有效，请勿泄露）
- \`ippm\`：输入价格（每百万 tokens，默认 4.0）
- \`oppm\`：输出价格（每百万 tokens，默认 8.0）
- \`cippm\`：缓存命中输入价格（每百万 tokens，可选）

**步骤 4：启动模型**

本地使用 Ollama 运行模型后，客户端会自动将模型信息推送到服务器，开始提供服务。

**步骤 5：查看收益**

可以在"模型收益"页面查看所有提供模型的收益情况。

## 3. 通过配置文件设置每个模型的价格

客户端默认读取当前目录下的 \`starfire_config.json\`，也可用 \`-config\` 指定路径。配置文件支持为每个模型单独设置价格：

\`\`\`json
{
  "host": "http://your-server.com",
  "token": "your-registration-token",
  "ippm": "3.8",
  "oppm": "8.3",
  "model_prices": {
    "DeepSeek-V4": { "engine": "openai", "ippm": "2.99", "oppm": "14.99", "cippm": "0.3" },
    "GLM-5.2": { "engine": "openai", "ippm": "3.99", "oppm": "19.99", "cippm": "0.4" }
  }
}
\`\`\`

**价格优先级（从高到低）：** 命令行参数 > 环境变量 > 配置文件。未在 \`model_prices\` 中配置的模型使用顶层 \`ippm/oppm\` 或默认价格。

## 4. 支持的推理引擎

- **Ollama** — 完整支持
- **Proxy（代理模式）** — 转发模型服务
- **llama.cpp** — 开发中
- **vllm** — 开发中
`;

const apiGuideZh = `
# 模型 API 调用说明

## 1. 前置条件

- 已注册 Star Fire 平台账号
- 已创建 API Key（在"API Key 管理"页面创建）

## 2. API 端点

所有 API 遵循 OpenAI API 标准格式，基础地址为：

\`\`\`
{server_host}/v1
\`\`\`

## 3. 获取模型列表

\`\`\`bash
curl {server_host}/v1/models \\
  -H "Authorization: Bearer {your_api_key}"
\`\`\`

## 4. 对话补全 (Chat Completions)

\`\`\`bash
curl {server_host}/v1/chat/completions \\
  -H "Authorization: Bearer {your_api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "模型ID",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'
\`\`\`

### 参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| \`model\` | string | 模型 ID（在模型广场复制） |
| \`messages\` | array | 对话消息列表 |
| \`stream\` | boolean | 是否使用流式输出（推荐 true） |
| \`max_tokens\` | integer | 最大生成 Token 数 |
| \`temperature\` | number | 生成温度（0-2，默认 1） |

## 5. Embedding 向量嵌入

\`\`\`bash
curl {server_host}/v1/embeddings \\
  -H "Authorization: Bearer {your_api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "模型ID",
    "input": "需要向量化的文本"
  }'
\`\`\`

## 6. 价格说明

模型调用费用按 Token 用量计算，单位：**人民币 / 百万 Tokens（¥/M）**。

- **IPPM**：输入 Token 价格（未命中缓存部分）
- **OPPM**：输出 Token 价格
- **CIPPM**：缓存命中输入 Token 价格

收益计算公式：
\`\`\`
收益 = ((输入tokens - 缓存命中) × IPPM + 缓存命中 × CIPPM + 输出tokens × OPPM) / 1,000,000
\`\`\`

## 7. Python SDK 示例

\`\`\`python
from openai import OpenAI

client = OpenAI(
    base_url="{server_host}/v1",
    api_key="your-api-key"
)

# 对话
response = client.chat.completions.create(
    model="模型ID",
    messages=[{"role": "user", "content": "你好"}],
    stream=True
)

for chunk in response:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
\`\`\`

## 8. 注意事项

1. API Key 请妥善保管，不要泄露给他人
2. 建议启用流式传输（\`stream: true\`）以获得更好的响应体验
3. 可以在"我的使用"页面查看用量详情
4. 如果遇到问题，请检查 API Key 是否有效、余额是否充足
`;

// 贡献者使用说明 (English)
const contributorGuideEn = `
# Contributor Guide

## 1. Registration & Login

Register and log in to the Star Fire platform with your email address.

## 2. Sharing Models

### Method 1: PC Client (Recommended)

Supports macOS and Windows GUI clients. See [Release](https://github.com/star-fire/releases) page.

- Log in with username/password; models are automatically registered and discovered
- Customize model pricing in real time
- View real-time revenue
- Mixed inference engine support

### Method 2: CLI Mode

**Step 1: Download the Client**

Download the client for your platform from [Release](https://github.com/star-fire/releases), or compile locally:

\`\`\`bash
make client
\`\`\`

The compiled binary is in the \`build/client\` directory.

**Step 2: Obtain a Registration Token**

Click **"Register to Star Fire"** on the Model Marketplace page to get a registration token.

**Step 3: Register the Client**

Windows:

\`\`\`bash
starfire.exe -host {host} -token {register token} -ippm {input price} -oppm {output price}
\`\`\`

macOS / Linux:

\`\`\`bash
starfire -host {host} -token {register token} -ippm {input price} -oppm {output price}
\`\`\`

Parameters:
- \`host\`: Server address
- \`token\`: Registration token (one-time use, do not share)
- \`ippm\`: Input price per million tokens (default: 4.0)
- \`oppm\`: Output price per million tokens (default: 8.0)
- \`cippm\`: Cached input price per million tokens (optional)

**Step 4: Start the Model**

Run a model locally with Ollama; the client will automatically push the model info to the server and begin serving.

**Step 5: View Revenue**

Go to the "Model Revenue" page to view earnings from all your shared models.

## 3. Per-Model Pricing via Config File

The client reads \`starfire_config.json\` from the current directory by default, or use \`-config\` to specify a path. The config file supports setting a price for each model individually:

\`\`\`json
{
  "host": "http://your-server.com",
  "token": "your-registration-token",
  "ippm": "3.8",
  "oppm": "8.3",
  "model_prices": {
    "DeepSeek-V4": { "engine": "openai", "ippm": "2.99", "oppm": "14.99", "cippm": "0.3" },
    "GLM-5.2": { "engine": "openai", "ippm": "3.99", "oppm": "19.99", "cippm": "0.4" }
  }
}
\`\`\`

**Price priority (highest to lowest):** command-line flags > environment variables > config file. Models not listed in \`model_prices\` use the top-level \`ippm/oppm\` or the default price.

## 4. Supported Inference Engines

- **Ollama** — Full support
- **Proxy (proxy mode)** — Forward model services
- **llama.cpp** — In development
- **vllm** — In development
`;

const apiGuideEn = `
# Model API Reference

## 1. Prerequisites

- Registered Star Fire account
- Created API Key (create one on the "API Key Management" page)

## 2. API Endpoint

All APIs follow the OpenAI API standard format. The base URL is:

\`\`\`
{server_host}/v1
\`\`\`

## 3. List Models

\`\`\`bash
curl {server_host}/v1/models \\
  -H "Authorization: Bearer {your_api_key}"
\`\`\`

## 4. Chat Completions

\`\`\`bash
curl {server_host}/v1/chat/completions \\
  -H "Authorization: Bearer {your_api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "模型ID",
    "messages": [
      {"role": "user", "content": "Hello"}
    ],
    "stream": true
  }'
\`\`\`

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| \`model\` | string | Model ID (copy from the Model Marketplace) |
| \`messages\` | array | List of chat messages |
| \`stream\` | boolean | Use streaming output (recommended: true) |
| \`max_tokens\` | integer | Maximum generated tokens |
| \`temperature\` | number | Sampling temperature (0-2, default: 1) |

## 5. Embeddings

\`\`\`bash
curl {server_host}/v1/embeddings \\
  -H "Authorization: Bearer {your_api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "模型ID",
    "input": "Text to embed"
  }'
\`\`\`

## 6. Pricing

Model usage is billed per token, in **CNY per million tokens (¥/M)**.

- **IPPM**: Input token price (uncached portion)
- **OPPM**: Output token price
- **CIPPM**: Cached input token price

Revenue formula:
\`\`\`
Revenue = ((input_tokens - cached_hits) × IPPM + cached_hits × CIPPM + output_tokens × OPPM) / 1,000,000
\`\`\`

## 7. Python SDK Example

\`\`\`python
from openai import OpenAI

client = OpenAI(
    base_url="{server_host}/v1",
    api_key="your-api-key"
)

# Chat
response = client.chat.completions.create(
    model="模型ID",
    messages=[{"role": "user", "content": "Hello"}],
    stream=True
)

for chunk in response:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
\`\`\`

## 8. Notes

1. Keep your API Key secure and do not share it with others
2. Enable streaming (\`stream: true\`) for a better response experience
3. Check the "My Usage" page to view usage details
4. If you encounter issues, verify that your API Key is valid and your balance is sufficient
`;

// 根据语言选择内容
const contributorGuideMarkdown = isEn ? contributorGuideEn : contributorGuideZh;
const apiGuideMarkdown = isEn ? apiGuideEn : apiGuideZh;

// 渲染 Markdown
const contributorGuideHtml = marked.parse(contributorGuideMarkdown) as string;
const apiGuideHtml = marked.parse(apiGuideMarkdown) as string;
</script>

<template>
  <div class="min-h-screen bg-[var(--bg-color)]">
    <!-- 顶部导航 -->
    <div class="sticky top-0 z-10 border-b border-[var(--border-color)] bg-[var(--bg-color)]/80 backdrop-blur-md">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <button
          @click="goBack"
          class="inline-flex items-center rounded-lg px-3 py-2 text-sm font-medium text-[var(--text-primary)] hover:bg-[var(--hover-bg)] transition-colors"
        >
          <svg class="mr-1.5 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
          </svg>
          {{ $t('business.marketplace.back') }}
        </button>
        <h1 class="text-lg font-bold text-[var(--text-primary)]">{{ $t('business.marketplace.usageGuide') }}</h1>
        <div class="w-20"></div>
      </div>
    </div>

    <div class="mx-auto max-w-6xl px-6 py-8">
      <!-- 贡献者使用说明 -->
      <section class="mb-12 rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-8">
        <div class="mb-6 flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-green-500 to-emerald-600">
            <svg class="h-5 w-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"/>
            </svg>
          </div>
          <h2 class="text-2xl font-bold text-[var(--text-primary)]">{{ $t('business.marketplace.contributorGuideTitle') }}</h2>
        </div>
        <div class="prose prose-gray max-w-none dark:prose-invert" v-html="contributorGuideHtml"></div>
      </section>

      <!-- 模型API调用说明 -->
      <section class="rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-8">
        <div class="mb-6 flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600">
            <svg class="h-5 w-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v14a2 2 0 002 2z"/>
            </svg>
          </div>
          <h2 class="text-2xl font-bold text-[var(--text-primary)]">{{ $t('business.marketplace.apiGuideTitle') }}</h2>
        </div>
        <div class="prose prose-gray max-w-none dark:prose-invert" v-html="apiGuideHtml"></div>
      </section>
    </div>
  </div>
</template>

<style scoped>
/* Markdown 内容样式 */
.prose h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-bottom: 1rem;
  color: var(--text-primary);
}

.prose h2 {
  font-size: 1.375rem;
  font-weight: 600;
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  color: var(--text-primary);
  padding-bottom: 0.375rem;
  border-bottom: 1px solid var(--border-color);
}

.prose h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.prose p {
  margin-bottom: 0.75rem;
  line-height: 1.75;
  color: var(--text-secondary);
}

.prose ul,
.prose ol {
  margin-bottom: 0.75rem;
  padding-left: 1.5rem;
}

.prose li {
  margin-bottom: 0.25rem;
  line-height: 1.75;
  color: var(--text-secondary);
}

.prose code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.875em;
  padding: 0.2em 0.4em;
  border-radius: 0.25rem;
  background-color: var(--hover-bg);
  color: var(--text-primary);
}

.prose pre {
  margin-bottom: 1rem;
  padding: 1rem;
  border-radius: 0.5rem;
  overflow-x: auto;
  background-color: var(--hover-bg);
  border: 1px solid var(--border-color);
}

.prose pre code {
  padding: 0;
  background: none;
  font-size: 0.8125rem;
  line-height: 1.6;
}

.prose table {
  width: 100%;
  margin-bottom: 1rem;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.prose th,
.prose td {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border-color);
  text-align: left;
}

.prose th {
  font-weight: 600;
  background-color: var(--hover-bg);
  color: var(--text-primary);
}

.prose td {
  color: var(--text-secondary);
}

.prose strong {
  color: var(--text-primary);
  font-weight: 600;
}

.prose hr {
  margin: 1.5rem 0;
  border-color: var(--border-color);
}

.prose blockquote {
  margin-bottom: 0.75rem;
  padding: 0.5rem 1rem;
  border-left: 3px solid var(--color-primary);
  background-color: var(--hover-bg);
  border-radius: 0 0.375rem 0.375rem 0;
  color: var(--text-secondary);
}

.prose a {
  color: var(--color-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.prose a:hover {
  opacity: 0.8;
}
</style>