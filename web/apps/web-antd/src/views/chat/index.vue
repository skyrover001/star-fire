<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 顶部导航 -->
    <div class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-10">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
          <!-- 左侧：返回按钮和模型信息 -->
          <div class="flex items-center space-x-4">
            <button
              @click="goBack"
              class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            >
              <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
              </svg>
            </button>
            
            <div class="flex items-center space-x-3">
              <div 
                class="w-10 h-10 rounded-xl flex items-center justify-center text-white"
                :style="{ backgroundColor: modelColor }"
              >
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
                </svg>
              </div>
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ modelName || modelId }}</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">AI 对话助手</p>
              </div>
            </div>
          </div>
          
          <!-- 右侧：操作按钮 -->
          <div class="flex items-center space-x-2">
            <button
              @click="clearChat"
              class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              title="清空对话"
            >
              <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1-1H8a1 1 0 00-1 1v3M4 7h16"/>
              </svg>
            </button>
            <button
              @click="showSettings = !showSettings"
              class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              :class="{ 'bg-gray-100 dark:bg-gray-700': showSettings }"
              title="设置"
            >
              <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- API Key 配置区域 -->
    <div v-if="!hasApiKey" class="max-w-4xl mx-auto p-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-8">
        <div class="text-center mb-6">
          <svg class="w-16 h-16 mx-auto mb-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
          </svg>
          <h3 class="text-2xl font-bold text-gray-900 dark:text-white mb-3">配置 API Key</h3>
          <p class="text-gray-600 dark:text-gray-400 max-w-md mx-auto">
            为了使用 AI 对话功能，请先配置您的 API Key。系统会自动获取可用的 API Key，或您可以手动输入。
          </p>
        </div>
        
        <div class="space-y-4">
          <!-- 自动获取按钮 -->
          <button
            @click="fetchApiKey"
            :disabled="fetchingApiKey"
            class="w-full px-6 py-3 bg-blue-500 hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-lg transition-colors font-medium flex items-center justify-center space-x-2"
          >
            <svg v-if="fetchingApiKey" class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            <span>{{ fetchingApiKey ? '获取中...' : '自动获取 API Key' }}</span>
          </button>
          
          <!-- 分割线 -->
          <div class="relative">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-gray-300 dark:border-gray-600"></div>
            </div>
            <div class="relative flex justify-center text-sm">
              <span class="px-2 bg-white dark:bg-gray-800 text-gray-500">或</span>
            </div>
          </div>
          
          <!-- 手动输入 -->
          <div class="space-y-3">
            <input
              v-model="apiKeyInput"
              type="password"
              placeholder="手动输入 API Key..."
              class="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              @keypress.enter="saveApiKey"
            />
            <button
              @click="saveApiKey"
              :disabled="!apiKeyInput.trim()"
              class="w-full px-6 py-3 bg-green-500 hover:bg-green-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-lg transition-colors font-medium"
            >
              保存 API Key
            </button>
          </div>
        </div>
        
        <div class="mt-6 text-center">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            API Key 将安全保存在本地，不会上传到服务器
          </p>
        </div>
      </div>
    </div>

    <!-- 对话区域 -->
    <div v-if="hasApiKey" class="flex h-screen">
      <!-- 预留顶部导航栏高度 -->
      <div class="w-full flex flex-col" style="height: calc(100vh - 4rem);">
        <!-- 主对话区域 -->
        <div class="flex flex-1 min-h-0">
          <!-- 对话内容区域 -->
          <div class="flex-1 flex flex-col min-h-0">
            <!-- 消息列表 -->
            <div ref="chatContainer" class="flex-1 overflow-y-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6 scroll-smooth">
              <!-- 欢迎消息 -->
              <div v-if="chatMessages.length === 0" class="text-center py-16">
                <div class="w-20 h-20 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center mx-auto mb-6">
                  <svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                  </svg>
                </div>
                <h3 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">开始与 AI 对话</h3>
                <p class="text-gray-500 dark:text-gray-400 max-w-md mx-auto">
                  您正在与 <span class="font-medium text-blue-600">{{ modelName || modelId }}</span> 对话。请在下方输入您的问题。
                </p>
              </div>
            
              <!-- 消息列表 -->
              <div v-for="(message, index) in chatMessages" :key="index" class="flex" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
                <div class="max-w-4xl w-full">
                  <!-- 思考过程显示 -->
                  <div v-if="message.thinking && message.role === 'assistant'" class="mb-4 relative">
                    <div class="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border border-blue-200 dark:border-blue-700 rounded-xl p-4">
                      <div class="flex items-center space-x-3 mb-3">
                        <div class="relative">
                          <div class="w-5 h-5 rounded-full border-2 border-blue-400 border-t-transparent animate-spin"></div>
                        </div>
                        <span class="text-sm font-semibold text-blue-700 dark:text-blue-300 uppercase tracking-wide">思考过程</span>
                      </div>
                      <div class="text-sm text-blue-800 dark:text-blue-200 leading-relaxed whitespace-pre-wrap font-mono bg-blue-100/50 dark:bg-blue-800/30 rounded-lg p-3 border border-blue-200/50 dark:border-blue-600/30">
                        {{ message.thinking }}
                      </div>
                    </div>
                  </div>
                  
                  <!-- 消息内容 -->
                  <div class="relative group">
                    <div class="px-5 py-4 rounded-2xl shadow-sm relative" :class="message.role === 'user' 
                      ? 'bg-gradient-to-r from-blue-500 to-blue-600 text-white ml-auto max-w-3xl' 
                      : 'bg-white dark:bg-gray-800 text-gray-900 dark:text-white border border-gray-200 dark:border-gray-700'">
                      
                      <!-- 流式输出指示器 -->
                      <div v-if="message.streaming" class="absolute -bottom-1 -right-1 flex items-center space-x-1">
                        <div class="w-2 h-2 bg-blue-400 rounded-full animate-pulse"></div>
                        <div class="w-2 h-2 bg-blue-400 rounded-full animate-pulse" style="animation-delay: 0.2s"></div>
                        <div class="w-2 h-2 bg-blue-400 rounded-full animate-pulse" style="animation-delay: 0.4s"></div>
                      </div>
                      
                      <!-- 消息内容 -->
                      <div 
                        class="text-sm leading-relaxed"
                        :class="message.role === 'user' ? 'whitespace-pre-wrap' : 'markdown-content'"
                      >
                        <div 
                          v-if="message.role === 'assistant'"
                          v-html="renderMarkdown(message.content)"
                        ></div>
                        <div 
                          v-else
                          class="whitespace-pre-wrap"
                        >{{ message.content }}</div>
                      </div>
                      
                      <!-- 时间戳 -->
                      <div class="flex items-center justify-between mt-3">
                        <div class="text-xs opacity-70" :class="message.role === 'user' ? 'text-blue-100' : 'text-gray-500 dark:text-gray-400'">
                          {{ formatTime(message.timestamp) }}
                        </div>
                        
                        <!-- 复制按钮 -->
                        <button
                          v-if="message.content && !message.streaming"
                          @click="copyMessage(message.content)"
                          class="opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                          :class="message.role === 'user' ? 'text-blue-100 hover:bg-blue-600' : 'text-gray-400'"
                          title="复制消息"
                        >
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              
              <!-- 加载指示器 -->
              <div v-if="chatLoading" class="flex justify-start">
                <div class="bg-white dark:bg-gray-800 px-5 py-4 rounded-2xl border border-gray-200 dark:border-gray-700 shadow-sm max-w-xs">
                  <div class="flex items-center space-x-3">
                    <div class="flex space-x-1">
                      <div class="w-2.5 h-2.5 bg-gray-400 rounded-full animate-bounce"></div>
                      <div class="w-2.5 h-2.5 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0.15s"></div>
                      <div class="w-2.5 h-2.5 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0.3s"></div>
                    </div>
                    <span class="text-sm text-gray-500 dark:text-gray-400">AI 正在思考...</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 输入区域 -->
            <div class="flex-shrink-0 border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 sm:px-6 lg:px-8 py-4">
              <div class="max-w-6xl mx-auto">
                <div class="flex space-x-4">
                  <div class="flex-1 relative">
                    <input
                      v-model="chatInput"
                      type="text"
                      placeholder="输入您的消息..."
                      class="w-full px-4 py-3 pr-12 border border-gray-300 dark:border-gray-600 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all"
                      @keypress.enter="sendMessage"
                      :disabled="chatLoading"
                    />
                    <!-- 字符计数 -->
                    <div class="absolute right-3 top-1/2 transform -translate-y-1/2 text-xs text-gray-400">
                      {{ chatInput.length }}
                    </div>
                  </div>
                  <button
                    @click="sendMessage"
                    :disabled="!chatInput.trim() || chatLoading"
                    class="px-6 py-3 bg-blue-500 hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-xl transition-all font-medium flex items-center space-x-2 shadow-sm hover:shadow-md"
                  >
                    <svg v-if="chatLoading" class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                    </svg>
                    <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
                    </svg>
                    <span>{{ chatLoading ? '发送中' : '发送' }}</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 设置侧边栏 -->
          <div v-if="showSettings" class="w-80 border-l border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 flex-shrink-0 overflow-y-auto">
            <div class="p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-6">对话设置</h3>
              
              <div class="space-y-6">
                <!-- 模型信息 -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    当前模型
                  </label>
                  <div class="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg border border-gray-200 dark:border-gray-600">
                    <div class="text-sm font-medium text-gray-900 dark:text-white">{{ modelName || modelId }}</div>
                    <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ modelId }}</div>
                  </div>
                </div>
                
                <!-- Temperature 设置 -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                    Temperature: {{ temperature }}
                  </label>
                  <input
                    v-model.number="temperature"
                    type="range"
                    min="0"
                    max="2"
                    step="0.1"
                    class="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-600"
                  />
                  <div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1">
                    <span>更精确</span>
                    <span>更创意</span>
                  </div>
                </div>
                
                <!-- Max Tokens 设置 -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    最大 Token 数
                  </label>
                  <input
                    v-model.number="maxTokens"
                    type="number"
                    min="1"
                    max="8000"
                    step="100"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    控制回复的最大长度
                  </div>
                </div>
                
                <!-- 统计信息 -->
                <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
                  <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">对话统计</h4>
                  <div class="space-y-2">
                    <div class="flex justify-between text-sm">
                      <span class="text-gray-500 dark:text-gray-400">消息数量:</span>
                      <span class="text-gray-900 dark:text-white font-medium">{{ chatMessages.length }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                      <span class="text-gray-500 dark:text-gray-400">估算 Token:</span>
                      <span class="text-gray-900 dark:text-white font-medium">{{ estimatedTokens }}</span>
                    </div>
                  </div>
                </div>
                
                <!-- 操作按钮 -->
                <div class="pt-4 border-t border-gray-200 dark:border-gray-700 space-y-3">
                  <button
                    @click="exportChat"
                    class="w-full px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors text-sm font-medium"
                  >
                    导出对话
                  </button>
                  <button
                    @click="clearChat"
                    class="w-full px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors text-sm font-medium"
                  >
                    清空对话
                  </button>
                  <button
                    @click="resetApiKey"
                    class="w-full px-4 py-2 bg-gray-500 hover:bg-gray-600 text-white rounded-lg transition-colors text-sm font-medium"
                  >
                    重置 API Key
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, nextTick } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { requestClient } from '#/api/request';
import { marked } from 'marked';
import hljs from 'highlight.js/lib/core';
// 导入常用语言支持
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import python from 'highlight.js/lib/languages/python';
import java from 'highlight.js/lib/languages/java';
import cpp from 'highlight.js/lib/languages/cpp';
import css from 'highlight.js/lib/languages/css';
import html from 'highlight.js/lib/languages/xml';
import json from 'highlight.js/lib/languages/json';
import bash from 'highlight.js/lib/languages/bash';
import sql from 'highlight.js/lib/languages/sql';
import go from 'highlight.js/lib/languages/go';
import rust from 'highlight.js/lib/languages/rust';
import php from 'highlight.js/lib/languages/php';

// 注册语言
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('python', python);
hljs.registerLanguage('java', java);
hljs.registerLanguage('cpp', cpp);
hljs.registerLanguage('css', css);
hljs.registerLanguage('html', html);
hljs.registerLanguage('xml', html);
hljs.registerLanguage('json', json);
hljs.registerLanguage('bash', bash);
hljs.registerLanguage('shell', bash);
hljs.registerLanguage('sql', sql);
hljs.registerLanguage('go', go);
hljs.registerLanguage('rust', rust);
hljs.registerLanguage('php', php);

// 路由相关
const router = useRouter();
const route = useRoute();

// 配置 marked
marked.use({
  renderer: {
    code(token: any) {
      const code = token.text;
      const language = token.lang;
      
      if (language && hljs.getLanguage(language)) {
        try {
          const highlighted = hljs.highlight(code, { language }).value;
          return `<pre class="hljs language-${language}"><code>${highlighted}</code></pre>`;
        } catch (err) {
          console.warn('代码高亮失败:', err);
        }
      }
      const autoHighlight = hljs.highlightAuto(code);
      return `<pre class="hljs"><code>${autoHighlight.value}</code></pre>`;
    },
    codespan(token: any) {
      return `<code class="inline-code">${token.text}</code>`;
    }
  },
  breaks: true,
  gfm: true,
});

// Markdown 渲染函数
const renderMarkdown = (content: string): string => {
  if (!content) return '';
  
  try {
    const rendered = marked.parse(content) as string;
    // 在下次 tick 时添加复制按钮
    addCopyButtonsToCodeBlocks();
    return rendered;
  } catch (error) {
    console.error('Markdown 渲染失败:', error);
    return content.replace(/\n/g, '<br>'); // 如果渲染失败，至少保持换行
  }
};

// 响应式状态
const modelId = ref<string>('');
const modelName = ref<string>('');
const modelColor = ref<string>('#3b82f6');
const apiKey = ref<string>('');
const apiKeyInput = ref<string>('');
const showSettings = ref(false);
const fetchingApiKey = ref(false);

// 聊天相关状态
interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  thinking?: string;
  streaming?: boolean;
}

const chatMessages = ref<ChatMessage[]>([]);
const chatInput = ref('');
const chatLoading = ref(false);
const chatContainer = ref<HTMLElement>();

// 模型设置
const temperature = ref(0.7);
const maxTokens = ref(2000);

// 计算属性
const hasApiKey = computed(() => !!apiKey.value);

const estimatedTokens = computed(() => {
  // 简单的 Token 估算：大约 4 个字符 = 1 Token
  return Math.ceil(chatMessages.value.reduce((total, msg) => total + msg.content.length, 0) / 4);
});

// 初始化
onMounted(async () => {
  // 从路由参数获取模型信息
  modelId.value = route.query.modelId as string || '';
  modelName.value = route.query.modelName as string || '';
  modelColor.value = route.query.modelColor as string || '#3b82f6';
  
  // 从localStorage获取API Key
  const savedApiKey = localStorage.getItem('openai_api_key');
  if (savedApiKey) {
    apiKey.value = savedApiKey;
  } else {
    // 自动尝试获取API Key
    await fetchApiKey();
  }
});

// 获取API Key
const fetchApiKey = async () => {
  try {
    fetchingApiKey.value = true;
    const response = await requestClient.get('/user/keys');
    
    if (response && response.keys && Array.isArray(response.keys) && response.keys.length > 0) {
      const validApiKey = response.keys.find((key: any) => {
        return key && key.key && key.status === 'active';
      });
      
      if (validApiKey) {
        apiKey.value = validApiKey.key;
        localStorage.setItem('openai_api_key', apiKey.value);
        console.log('自动获取API Key成功');
      } else {
        console.warn('未找到有效的API Key');
      }
    } else {
      console.warn('API响应中没有找到keys数组');
    }
  } catch (error) {
    console.error('获取API Key失败:', error);
  } finally {
    fetchingApiKey.value = false;
  }
};

// 保存API Key
const saveApiKey = () => {
  if (apiKeyInput.value.trim()) {
    apiKey.value = apiKeyInput.value.trim();
    localStorage.setItem('openai_api_key', apiKey.value);
    apiKeyInput.value = '';
  }
};

// 重置API Key
const resetApiKey = () => {
  apiKey.value = '';
  localStorage.removeItem('openai_api_key');
  apiKeyInput.value = '';
};

// 返回上一页
const goBack = () => {
  router.back();
};

// 发送消息
const sendMessage = async () => {
  if (!chatInput.value.trim() || chatLoading.value || !apiKey.value) return;

  const userMessage = chatInput.value.trim();
  chatInput.value = '';
  
  // 添加用户消息
  chatMessages.value.push({
    role: 'user',
    content: userMessage,
    timestamp: new Date()
  });
  
  // 立即滚动到底部
  await scrollToBottom();
  
  chatLoading.value = true;

  try {
    // 添加AI消息占位符
    const aiMessageIndex = chatMessages.value.length;
    chatMessages.value.push({
      role: 'assistant',
      content: '',
      timestamp: new Date(),
      thinking: '',
      streaming: true
    });

    // 流式请求处理
    const response = await fetch('/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey.value}`
      },
      body: JSON.stringify({
        model: modelId.value,
        messages: chatMessages.value
          .filter(msg => msg.role && msg.content && !msg.streaming)
          .map(msg => ({
            role: msg.role,
            content: msg.content
          })),
        temperature: temperature.value,
        max_tokens: maxTokens.value,
        stream: true
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error('无法获取响应流');
    }

    const decoder = new TextDecoder();
    let buffer = '';
    let assistantContent = '';
    let thinkingContent = '';
    let isInThinking = false;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.trim() === '') continue;
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data === '[DONE]') {
            const aiMessage = chatMessages.value[aiMessageIndex];
            if (aiMessage) {
              aiMessage.streaming = false;
            }
            await scrollToBottom();
            continue;
          }

          try {
            const parsed = JSON.parse(data);
            const delta = parsed.choices?.[0]?.delta;
            
            if (delta?.content) {
              const content = delta.content;
              const aiMessage = chatMessages.value[aiMessageIndex];
              if (!aiMessage) continue;
              
              // 处理思考过程
              if (content.includes('<thinking>')) {
                isInThinking = true;
                const parts = content.split('<thinking>');
                if (parts[0]) {
                  assistantContent += parts[0];
                  aiMessage.content = assistantContent;
                }
                if (parts[1]) {
                  thinkingContent += parts[1];
                }
              } else if (content.includes('</thinking>')) {
                isInThinking = false;
                const parts = content.split('</thinking>');
                if (parts[0]) {
                  thinkingContent += parts[0];
                  aiMessage.thinking = thinkingContent;
                }
                if (parts[1]) {
                  assistantContent += parts[1];
                  aiMessage.content = assistantContent;
                }
              } else if (isInThinking) {
                thinkingContent += content;
                aiMessage.thinking = thinkingContent;
              } else {
                assistantContent += content;
                aiMessage.content = assistantContent;
              }
              
              // 为流式更新添加延迟渲染，避免频繁重新渲染
              setTimeout(() => {
                addCopyButtonsToCodeBlocks();
              }, 100);
              
              await scrollToBottom();
            }
          } catch (e) {
            console.warn('解析SSE数据失败:', e, data);
          }
        }
      }
    }

    const finalMessage = chatMessages.value[aiMessageIndex];
    if (finalMessage) {
      finalMessage.streaming = false;
    }
    
  } catch (error) {
    console.error('Chat error:', error);
    // 移除失败的消息
    if (chatMessages.value[chatMessages.value.length - 1]?.streaming) {
      chatMessages.value.pop();
    }
    // 添加错误消息
    chatMessages.value.push({
      role: 'assistant',
      content: `抱歉，发生了错误：${error instanceof Error ? error.message : '未知错误'}`,
      timestamp: new Date()
    });
  } finally {
    chatLoading.value = false;
    await scrollToBottom();
  }
};

// 清空对话
const clearChat = () => {
  chatMessages.value = [];
};

// 导出对话
const exportChat = () => {
  const chatData = {
    model: modelId.value,
    modelName: modelName.value,
    timestamp: new Date().toISOString(),
    messages: chatMessages.value.map(msg => ({
      role: msg.role,
      content: msg.content,
      thinking: msg.thinking,
      timestamp: msg.timestamp.toISOString()
    }))
  };
  
  const blob = new Blob([JSON.stringify(chatData, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `chat-${modelName.value || modelId.value}-${new Date().toISOString().split('T')[0]}.json`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
};

// 复制消息
const copyMessage = async (content: string) => {
  try {
    await navigator.clipboard.writeText(content);
    // 可以添加一个简单的提示
    console.log('消息已复制到剪贴板');
  } catch (err) {
    console.error('复制失败:', err);
  }
};

// 复制代码块
const copyCodeBlock = async (code: string) => {
  try {
    await navigator.clipboard.writeText(code);
    console.log('代码已复制到剪贴板');
  } catch (err) {
    console.error('复制代码失败:', err);
  }
};

// 添加代码复制按钮的功能
const addCopyButtonsToCodeBlocks = () => {
  nextTick(() => {
    const codeBlocks = document.querySelectorAll('.markdown-content pre code');
    codeBlocks.forEach((codeBlock) => {
      const pre = codeBlock.parentElement;
      if (pre && !pre.querySelector('.copy-button')) {
        const copyButton = document.createElement('button');
        copyButton.className = 'copy-button absolute top-2 right-2 bg-gray-700 hover:bg-gray-600 text-white text-xs px-2 py-1 rounded opacity-0 group-hover:opacity-100 transition-opacity';
        copyButton.textContent = '复制';
        copyButton.onclick = () => copyCodeBlock(codeBlock.textContent || '');
        
        pre.style.position = 'relative';
        pre.className += ' group';
        pre.appendChild(copyButton);
      }
    });
  });
};

// 自动滚动到底部
const scrollToBottom = async () => {
  await nextTick();
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight;
  }
};

// 格式化时间
const formatTime = (date: Date) => {
  return date.toLocaleTimeString('zh-CN', { 
    hour: '2-digit', 
    minute: '2-digit',
    hour12: false 
  });
};
</script>

<style scoped>
/* 自定义滚动条 */
::-webkit-scrollbar {
  width: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: rgba(156, 163, 175, 0.5);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(156, 163, 175, 0.7);
}

/* 暗色模式下的滚动条 */
.dark ::-webkit-scrollbar-thumb {
  background: rgba(75, 85, 99, 0.5);
}

.dark ::-webkit-scrollbar-thumb:hover {
  background: rgba(75, 85, 99, 0.7);
}

/* 自定义 range input 样式 */
input[type="range"] {
  -webkit-appearance: none;
  appearance: none;
}

input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  height: 20px;
  width: 20px;
  border-radius: 50%;
  background: #3b82f6;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

input[type="range"]::-moz-range-thumb {
  height: 20px;
  width: 20px;
  border-radius: 50%;
  background: #3b82f6;
  cursor: pointer;
  border: none;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

/* Markdown 内容样式 */
.markdown-content {
  line-height: 1.6;
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4),
.markdown-content :deep(h5),
.markdown-content :deep(h6) {
  font-weight: 600;
  margin: 1rem 0 0.5rem 0;
  color: inherit;
}

.markdown-content :deep(h1) { font-size: 1.5rem; }
.markdown-content :deep(h2) { font-size: 1.375rem; }
.markdown-content :deep(h3) { font-size: 1.25rem; }
.markdown-content :deep(h4) { font-size: 1.125rem; }
.markdown-content :deep(h5) { font-size: 1rem; }
.markdown-content :deep(h6) { font-size: 0.875rem; }

.markdown-content :deep(p) {
  margin: 0.75rem 0;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  margin: 0.75rem 0;
  padding-left: 1.5rem;
}

.markdown-content :deep(li) {
  margin: 0.25rem 0;
}

.markdown-content :deep(blockquote) {
  border-left: 4px solid #e5e7eb;
  padding-left: 1rem;
  margin: 1rem 0;
  font-style: italic;
  color: #6b7280;
}

.dark .markdown-content :deep(blockquote) {
  border-left-color: #4b5563;
  color: #9ca3af;
}

.markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 1rem 0;
}

.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid #e5e7eb;
  padding: 0.5rem;
  text-align: left;
}

.dark .markdown-content :deep(th),
.dark .markdown-content :deep(td) {
  border-color: #4b5563;
}

.markdown-content :deep(th) {
  background-color: #f9fafb;
  font-weight: 600;
}

.dark .markdown-content :deep(th) {
  background-color: #374151;
}

.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 1.5rem 0;
}

.dark .markdown-content :deep(hr) {
  border-top-color: #4b5563;
}

/* 内联代码样式 */
.markdown-content :deep(.inline-code),
.inline-code {
  background-color: #f1f5f9;
  color: #e11d48;
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875em;
  border: 1px solid #e2e8f0;
}

.dark .markdown-content :deep(.inline-code),
.dark .inline-code {
  background-color: #334155;
  color: #fbbf24;
  border-color: #475569;
}

/* 代码块样式 */
.markdown-content :deep(.hljs) {
  background-color: #f8fafc !important;
  border: 1px solid #e2e8f0;
  border-radius: 0.5rem;
  padding: 1rem;
  margin: 1rem 0;
  overflow-x: auto;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  line-height: 1.5;
}

.dark .markdown-content :deep(.hljs) {
  background-color: #1e293b !important;
  border-color: #475569;
}

/* 代码高亮主题 - 亮色模式 */
.markdown-content :deep(.hljs-comment),
.markdown-content :deep(.hljs-quote) {
  color: #64748b;
  font-style: italic;
}

.markdown-content :deep(.hljs-keyword),
.markdown-content :deep(.hljs-selector-tag),
.markdown-content :deep(.hljs-type) {
  color: #7c3aed;
  font-weight: 600;
}

.markdown-content :deep(.hljs-string),
.markdown-content :deep(.hljs-regexp) {
  color: #059669;
}

.markdown-content :deep(.hljs-number),
.markdown-content :deep(.hljs-literal) {
  color: #dc2626;
}

.markdown-content :deep(.hljs-variable),
.markdown-content :deep(.hljs-function) {
  color: #2563eb;
}

.markdown-content :deep(.hljs-title),
.markdown-content :deep(.hljs-class),
.markdown-content :deep(.hljs-section) {
  color: #ea580c;
  font-weight: 600;
}

.markdown-content :deep(.hljs-attr),
.markdown-content :deep(.hljs-attribute) {
  color: #7c2d12;
}

/* 暗色模式代码高亮 */
.dark .markdown-content :deep(.hljs-comment),
.dark .markdown-content :deep(.hljs-quote) {
  color: #94a3b8;
}

.dark .markdown-content :deep(.hljs-keyword),
.dark .markdown-content :deep(.hljs-selector-tag),
.dark .markdown-content :deep(.hljs-type) {
  color: #a78bfa;
}

.dark .markdown-content :deep(.hljs-string),
.dark .markdown-content :deep(.hljs-regexp) {
  color: #34d399;
}

.dark .markdown-content :deep(.hljs-number),
.dark .markdown-content :deep(.hljs-literal) {
  color: #f87171;
}

.dark .markdown-content :deep(.hljs-variable),
.dark .markdown-content :deep(.hljs-function) {
  color: #60a5fa;
}

.dark .markdown-content :deep(.hljs-title),
.dark .markdown-content :deep(.hljs-class),
.dark .markdown-content :deep(.hljs-section) {
  color: #fb923c;
}

.dark .markdown-content :deep(.hljs-attr),
.dark .markdown-content :deep(.hljs-attribute) {
  color: #fbbf24;
}

/* 复制按钮样式 */
.markdown-content :deep(pre) {
  position: relative;
}

.markdown-content :deep(.copy-button) {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background-color: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  cursor: pointer;
  border: none;
  opacity: 0;
  transition: opacity 0.2s;
  z-index: 10;
}

.markdown-content :deep(pre.group:hover .copy-button) {
  opacity: 1;
}

.dark .markdown-content :deep(.copy-button) {
  background-color: rgba(255, 255, 255, 0.9);
  color: black;
}

/* 链接样式 */
.markdown-content :deep(a) {
  color: #2563eb;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.markdown-content :deep(a):hover {
  color: #1d4ed8;
}

.dark .markdown-content :deep(a) {
  color: #60a5fa;
}

.dark .markdown-content :deep(a):hover {
  color: #93c5fd;
}

/* 强调文本 */
.markdown-content :deep(strong) {
  font-weight: 600;
  color: inherit;
}

.markdown-content :deep(em) {
  font-style: italic;
}

/* 删除线 */
.markdown-content :deep(del) {
  text-decoration: line-through;
  opacity: 0.7;
}
</style>
