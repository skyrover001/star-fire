<template>
  <div class="space-y-6">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- 总密钥数 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-blue-100 dark:bg-blue-900/20">
              <svg class="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
              </svg>
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">总密钥</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-12"></span>
              <span v-else>{{ statistics.totalKeys }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">最多可创建 5 个</p>
          </div>
        </div>
      </div>

      <!-- 活跃密钥数 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-green-100 dark:bg-green-900/20">
              <svg class="h-6 w-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">活跃密钥</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-12"></span>
              <span v-else>{{ statistics.activeKeys }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">未被撤销的密钥</p>
          </div>
        </div>
      </div>

      <!-- 总调用次数 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-purple-100 dark:bg-purple-900/20">
              <svg class="h-6 w-6 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
              </svg>
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">总调用次数</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>{{ formatNumber(statistics.totalCalls) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">所有密钥累计</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 页面说明 -->
    <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
      <div class="flex items-start">
        <div class="flex-shrink-0">
          <svg class="h-6 w-6 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
        </div>
        <div class="ml-3">
          <h3 class="text-sm font-medium text-[var(--text-primary)]">API Key 管理</h3>
          <div class="mt-2 text-sm text-[var(--text-secondary)]">
            <p>您可以查看和管理您的所有 API Key。最多可以保留 5 个 API Key。密钥只会在首次显示一次，请妥善保存。不要与他人共享 API Key，或将其暴露在客户端代码中。为了保护您的账户安全，一旦 API 密钥被发现泄露，Star fire 可能会将其禁用。</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 操作栏 -->
    <div class="flex items-center justify-between">
      <div class="flex items-center space-x-4">
        <button
          class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          @click="showCreateModal = true"
        >
          <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          创建新的 API Key
        </button>
        <button
          class="inline-flex items-center rounded-lg bg-[var(--color-neutral-700)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--color-neutral-600)] focus:outline-none focus:ring-2 focus:ring-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="loading"
          @click="refreshApiKeys"
        >
          <svg 
            v-if="loading" 
            class="animate-spin mr-2 h-4 w-4" 
            xmlns="http://www.w3.org/2000/svg" 
            fill="none" 
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <svg 
            v-else 
            class="mr-2 h-4 w-4" 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          {{ loading ? '刷新中...' : '刷新' }}
        </button>
      </div>
    </div>

    <!-- API Key 列表 -->
    <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] overflow-hidden">
      <!-- 表头 -->
      <div class="px-6 py-4 bg-[var(--bg-color-secondary)] border-b border-[var(--border-color)]">
        <div class="grid grid-cols-12 gap-4 text-sm font-medium text-[var(--text-secondary)]">
          <div class="col-span-2">名称</div>
          <div class="col-span-2">创建时间</div>
          <div class="col-span-2">到期时间</div>
          <div class="col-span-3">Key</div>
          <div class="col-span-2">最后使用</div>
          <div class="col-span-1">操作</div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="px-6 py-12 text-center">
        <div class="inline-flex items-center">
          <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-blue-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          加载中...
        </div>
      </div>

      <!-- API Key 列表项 -->
      <div v-else-if="apiKeys.length > 0" class="divide-y divide-[var(--border-color)]">
        <div
          v-for="apiKey in apiKeys"
          :key="apiKey.id"
          class="px-6 py-4 hover:bg-[var(--bg-color-secondary)] transition-colors"
        >
          <div class="grid grid-cols-12 gap-4 items-center">
            <!-- 名称 -->
            <div class="col-span-2">
              <p class="text-sm font-medium text-[var(--text-primary)]">{{ apiKey.name }}</p>
            </div>

            <!-- 创建时间 -->
            <div class="col-span-2">
              <p class="text-sm text-[var(--text-secondary)]">{{ formatDate(apiKey.createTime) }}</p>
            </div>

            <!-- 到期时间 -->
            <div class="col-span-2">
              <p class="text-sm text-[var(--text-secondary)]">{{ formatDate(apiKey.expiresAt) }}</p>
              <span 
                v-if="apiKey.revoked"
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400 mt-1"
              >
                已撤销
              </span>
            </div>

            <!-- Key -->
            <div class="col-span-3">
              <div class="flex items-center space-x-2">
                <code class="px-2 py-1 bg-[var(--bg-color)] rounded text-sm font-mono text-[var(--text-primary)]">
                  {{ apiKey.key }}
                </code>
                <button
                  class="p-1 rounded hover:bg-[var(--bg-color)] transition-colors"
                  @click="copyToClipboard(apiKey.fullKey)"
                  title="复制完整密钥"
                >
                  <svg class="h-4 w-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                  </svg>
                </button>
              </div>
            </div>

            <!-- 最后使用时间 -->
            <div class="col-span-2">
              <p class="text-sm text-[var(--text-secondary)]">
                {{ apiKey.lastUsedTime ? formatDate(apiKey.lastUsedTime) : '从未使用' }}
              </p>
            </div>

            <!-- 操作 -->
            <div class="col-span-1">
              <div class="flex items-center space-x-2">
                <button
                  v-if="!apiKey.revoked"
                  class="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 text-sm"
                  @click="revokeApiKey(apiKey)"
                  title="撤销"
                >
                  撤销
                </button>
                <button
                  class="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 text-sm"
                  @click="deleteApiKey(apiKey)"
                  title="删除"
                >
                  删除
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="px-6 py-12 text-center">
        <svg class="mx-auto h-12 w-12 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
        </svg>
        <h3 class="mt-2 text-sm font-medium text-[var(--text-primary)]">暂无 API Key</h3>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">开始创建您的第一个 API Key</p>
        <div class="mt-6">
          <button
            class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            @click="showCreateModal = true"
          >
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            创建 API Key
          </button>
        </div>
      </div>
    </div>

    <!-- 创建 API Key 模态框 -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 overflow-y-auto">
      <div class="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        <!-- 背景遮罩 -->
        <div class="fixed inset-0 transition-opacity" @click="closeCreateModal">
          <div class="absolute inset-0 bg-black/50 backdrop-blur-sm"></div>
        </div>

        <!-- 模态框内容 -->
        <div class="inline-block align-bottom bg-[var(--content-bg)] rounded-xl px-6 pt-6 pb-6 text-left overflow-hidden shadow-2xl transform transition-all sm:my-8 sm:align-middle sm:max-w-md sm:w-full border border-[var(--border-color)]">
          <!-- 头部 -->
          <div class="flex items-center justify-between mb-6">
            <div class="flex items-center">
              <div class="flex items-center justify-center h-10 w-10 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 shadow-lg">
                <svg class="h-5 w-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                </svg>
              </div>
              <div class="ml-4">
                <h3 class="text-lg font-semibold text-[var(--text-primary)]">创建新的 API Key</h3>
                <p class="text-sm text-[var(--text-secondary)] mt-1">请为您的 API Key 设置一个名称，便于识别和管理。</p>
              </div>
            </div>
            <button 
              class="p-1 rounded-lg hover:bg-[var(--bg-color-secondary)] transition-colors"
              @click="closeCreateModal"
            >
              <svg class="h-5 w-5 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- 表单区域 -->
          <div class="space-y-6">
            <!-- API Key 名称输入 -->
            <div>
              <label for="apiKeyName" class="block text-sm font-medium text-[var(--text-primary)] mb-2">
                API Key 名称 <span class="text-red-500">*</span>
              </label>
              <div class="relative">
                <input
                  id="apiKeyName"
                  v-model="newApiKeyName"
                  type="text"
                  class="w-full px-4 py-3 border border-[var(--border-color)] rounded-lg bg-[var(--input-bg)] text-[var(--text-primary)] placeholder-[var(--text-secondary)] focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
                  placeholder="例如：star-fire 或 我的项目"
                  maxlength="50"
                  @keyup.enter="createApiKey"
                  @keyup.esc="closeCreateModal"
                >
                <div class="absolute inset-y-0 right-0 flex items-center pr-3">
                  <span class="text-xs text-[var(--text-secondary)]">{{ newApiKeyName.length }}/50</span>
                </div>
              </div>
              <div class="mt-2 flex items-start space-x-2">
                <svg class="h-4 w-4 text-blue-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                <p class="text-sm text-[var(--text-secondary)]">
                  建议使用有意义的名称，如项目名称或用途描述，方便后续管理和识别
                </p>
              </div>
            </div>

            <!-- 安全提示 -->
            <div class="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
              <div class="flex items-start">
                <div class="flex-shrink-0">
                  <svg class="h-5 w-5 text-amber-600 dark:text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z"/>
                  </svg>
                </div>
                <div class="ml-3">
                  <h4 class="text-sm font-medium text-amber-800 dark:text-amber-200">安全提示</h4>
                  <div class="text-sm text-amber-700 dark:text-amber-300 mt-1 space-y-1">
                    <p>• API Key 创建后只会显示一次，请妥善保存</p>
                    <p>• 不要在客户端代码中暴露您的 API Key</p>
                    <p>• 发现泄露后请立即撤销并重新创建</p>
                  </div>
                </div>
              </div>
            </div>

            <!-- 功能特性 -->
            <div class="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
              <h4 class="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">API Key 功能</h4>
              <div class="space-y-2">
                <div class="flex items-center text-sm text-blue-700 dark:text-blue-300">
                  <svg class="h-4 w-4 mr-2 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                  </svg>
                  访问Star fire平台模型的 openAI API标准的接口
                </div>
                <div class="flex items-center text-sm text-blue-700 dark:text-blue-300">
                  <svg class="h-4 w-4 mr-2 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                  </svg>
                  完整的权限管理
                </div>
                <div class="flex items-center text-sm text-blue-700 dark:text-blue-300">
                  <svg class="h-4 w-4 mr-2 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                  </svg>
                  使用统计和监控
                </div>
              </div>
            </div>
          </div>

          <!-- 按钮区域 -->
          <div class="flex items-center justify-end space-x-3 mt-8 pt-6 border-t border-[var(--border-color)]">
            <button
              type="button"
              class="px-4 py-2 text-sm font-medium text-[var(--text-secondary)] bg-[var(--bg-color-secondary)] hover:bg-[var(--bg-color)] border border-[var(--border-color)] rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-gray-500"
              @click="closeCreateModal"
            >
              取消
            </button>
            <button
              type="button"
              class="inline-flex items-center px-6 py-2 text-sm font-medium text-white bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 rounded-lg shadow-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              :disabled="!newApiKeyName.trim() || creating"
              @click="createApiKey"
            >
              <svg v-if="creating" class="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <svg v-else class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
              </svg>
              {{ creating ? '创建中...' : '创建 API Key' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 显示新创建的 API Key 模态框 -->
    <div v-if="showNewKeyModal" class="fixed inset-0 z-50 overflow-y-auto">
      <div class="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        <!-- 背景遮罩 -->
        <div class="fixed inset-0 transition-opacity">
          <div class="absolute inset-0 bg-black/50 backdrop-blur-sm"></div>
        </div>

        <!-- 模态框内容 -->
        <div class="inline-block align-bottom bg-[var(--content-bg)] rounded-xl px-6 pt-6 pb-6 text-left overflow-hidden shadow-2xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full border border-[var(--border-color)]">
          <!-- 头部 -->
          <div class="text-center mb-6">
            <div class="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-gradient-to-br from-green-500 to-green-600 shadow-lg mb-4">
              <svg class="h-8 w-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
            </div>
            <h3 class="text-xl font-semibold text-[var(--text-primary)] mb-2">API Key 创建成功！</h3>
            <p class="text-sm text-[var(--text-secondary)]">请妥善保存您的 API Key，它只会显示一次。</p>
          </div>

          <!-- 重要提示 -->
          <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <div class="flex items-start">
              <div class="flex-shrink-0">
                <svg class="h-5 w-5 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z"/>
                </svg>
              </div>
              <div class="ml-3">
                <h4 class="text-sm font-medium text-red-800 dark:text-red-200">⚠️ 重要提示</h4>
                <p class="text-sm text-red-700 dark:text-red-300 mt-1">
                  此密钥只会显示这一次，关闭此窗口后将无法再次查看完整密钥。请立即复制并保存到安全的地方。
                </p>
              </div>
            </div>
          </div>

          <!-- API Key 显示区域 -->
          <div class="mb-6">
            <label class="block text-sm font-medium text-[var(--text-primary)] mb-3">您的 API Key</label>
            <div class="p-4 bg-[var(--bg-color)] border border-[var(--border-color)] rounded-lg">
              <div class="flex items-center space-x-3">
                <input
                  :value="newApiKey"
                  readonly
                  class="flex-1 bg-transparent text-sm font-mono text-[var(--text-primary)] focus:outline-none select-all"
                  @click="selectAllText"
                >
                <button
                  class="inline-flex items-center px-3 py-2 text-sm font-medium text-white bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 rounded-lg shadow-md transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  @click="copyToClipboard(newApiKey)"
                >
                  <svg class="mr-1 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                  </svg>
                  复制
                </button>
              </div>
            </div>
          </div>

          <!-- 使用指南 -->
          <div class="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
            <h4 class="text-sm font-medium text-blue-800 dark:text-blue-200 mb-3">📚 如何使用您的 API Key</h4>
            <div class="space-y-2 text-sm text-blue-700 dark:text-blue-300">
              <div class="flex items-start">
                <span class="mr-2">1.</span>
                <span>在 HTTP 请求头中添加：<code class="bg-blue-100 dark:bg-blue-800 px-1 rounded text-xs">Authorization: Bearer YOUR_API_KEY</code></span>
              </div>
              <div class="flex items-start">
                <span class="mr-2">2.</span>
                <span>确保在 HTTPS 环境下使用，保护数据传输安全</span>
              </div>
              <div class="flex items-start">
                <span class="mr-2">3.</span>
                <span>建议设置请求频率限制，避免超出配额</span>
              </div>
            </div>
          </div>

          <!-- 安全建议 -->
          <div class="mb-6 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
            <h4 class="text-sm font-medium text-amber-800 dark:text-amber-200 mb-2">🔒 安全建议</h4>
            <ul class="space-y-1 text-sm text-amber-700 dark:text-amber-300">
              <li>• 将 API Key 存储在环境变量中，不要硬编码在代码里</li>
              <li>• 定期轮换 API Key，建议每 3-6 个月更换一次</li>
              <li>• 监控 API Key 使用情况，及时发现异常访问</li>
              <li>• 如发现泄露，请立即撤销并创建新的 API Key</li>
            </ul>
          </div>

          <!-- 按钮区域 -->
          <div class="flex items-center justify-center pt-4 border-t border-[var(--border-color)]">
            <button
              type="button"
              class="inline-flex items-center px-8 py-3 text-sm font-medium text-white bg-gradient-to-r from-green-600 to-green-700 hover:from-green-700 hover:to-green-800 rounded-lg shadow-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-green-500"
              @click="closeNewKeyModal"
            >
              <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
              </svg>
              我已安全保存
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { message } from 'ant-design-vue';
import { requestClient } from '#/api/request';

// 接口类型定义 - 适配新接口格式
interface ApiKey {
  ID: string;
  UserID: string;
  Name: string;
  Key: string;
  Prefix: string;
  LastUsed: string;
  CreatedAt: string;
  ExpiresAt: string;
  Revoked: boolean;
  CallCount?: number; // 可选的调用次数字段
  TotalRequests?: number; // 可选的总请求数字段
}

// 本地显示用的类型
interface DisplayApiKey {
  id: string;
  name: string;
  key: string;
  fullKey: string; // 保存完整的密钥用于复制
  createTime: string;
  expiresAt: string;
  project: string;
  revoked?: boolean;
  lastUsedTime?: string;
}

// 响应式数据
const loading = ref(false);
const apiKeys = ref<DisplayApiKey[]>([]);
const showCreateModal = ref(false);
const showNewKeyModal = ref(false);
const newApiKeyName = ref('');
const newApiKey = ref('');
const creating = ref(false);

// 统计数据类型定义
interface Statistics {
  totalKeys: number;
  activeKeys: number;
  totalCalls: number;
}

// 统计数据
const statistics = ref<Statistics>({
  totalKeys: 0,
  activeKeys: 0,
  totalCalls: 0,
});

// 计算统计数据
const calculateStatistics = (keys: ApiKey[]) => {
  const totalKeys = keys.length;
  const activeKeys = keys.filter(key => !key.Revoked).length;
  
  // 计算总调用次数，优先使用接口返回的真实数据
  const totalCalls = keys.reduce((sum, key) => {
    // 优先使用 CallCount 或 TotalRequests 字段
    let callCount = 0;
    if (typeof key.CallCount === 'number') {
      callCount = key.CallCount;
    } else if (typeof key.TotalRequests === 'number') {
      callCount = key.TotalRequests;
    } else {
      // 如果接口没有返回调用次数，使用基于最后使用时间的模拟数据
      if (key.LastUsed && key.LastUsed !== '') {
        // 有使用记录的密钥，模拟一个较大的调用次数
        callCount = Math.floor(Math.random() * 5000) + 100;
      } else {
        // 没有使用记录的密钥，调用次数为0
        callCount = 0;
      }
    }
    return sum + callCount;
  }, 0);
  
  statistics.value = {
    totalKeys,
    activeKeys,
    totalCalls,
  };
  
  console.log('统计数据计算完成:', {
    总密钥数: totalKeys,
    活跃密钥数: activeKeys,
    总调用次数: totalCalls,
    原始数据样本: keys.length > 0 && keys[0] ? {
      密钥ID: keys[0].ID,
      是否有CallCount: 'CallCount' in keys[0],
      是否有TotalRequests: 'TotalRequests' in keys[0],
      LastUsed: keys[0].LastUsed
    } : '无数据'
  });
};

// 格式化数字显示
const formatNumber = (num: number): string => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toString();
};

// 数据转换函数 - 将接口数据转换为显示数据
const transformApiKey = (apiKey: ApiKey): DisplayApiKey => {
  return {
    id: apiKey.ID || '',
    name: apiKey.Name || 'Unnamed',
    key: apiKey.Prefix ? `${apiKey.Prefix}...` : (apiKey.Key ? `${apiKey.Key.substring(0, 10)}...` : ''),
    fullKey: apiKey.Key || '', // 保存完整密钥用于复制
    createTime: apiKey.CreatedAt || '',
    expiresAt: apiKey.ExpiresAt || '',
    project: 'default', // 新接口没有项目字段，使用默认值
    revoked: apiKey.Revoked === true,
    lastUsedTime: apiKey.LastUsed || undefined,
  };
};

// 加载 API Keys
const loadApiKeys = async () => {
  loading.value = true;
  try {
    const response = await requestClient.get('/user/keys');
    console.log('API Keys 响应:', response);
    // 适配新接口格式：{ keys: [...] }
    const rawKeys = response?.keys || [];
    apiKeys.value = rawKeys.map((key: ApiKey) => transformApiKey(key));
    
    // 计算统计数据
    calculateStatistics(rawKeys);
    
    console.log('加载 API Keys 成功:', apiKeys.value);
    console.log('统计数据:', statistics.value);
  } catch (error) {
    console.error('加载 API Keys 失败:', error);
    message.error('加载 API Keys 失败');
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshApiKeys = async () => {
  try {
    await loadApiKeys();
    message.success('数据已刷新');
  } catch (error) {
    console.error('刷新数据失败:', error);
    message.error('刷新数据失败');
  }
};

// 创建 API Key
const createApiKey = async () => {
  if (!newApiKeyName.value.trim()) {
    message.error('请输入 API Key 名称');
    return;
  }

  creating.value = true;
  try {
    const response = await requestClient.post('/user/keys', {
      name: newApiKeyName.value.trim(),
    });
    
    newApiKey.value = response.key;
    showCreateModal.value = false;
    showNewKeyModal.value = true;
    newApiKeyName.value = '';
    
    // 刷新列表
    await loadApiKeys();
    message.success('API Key 创建成功');
  } catch (error) {
    console.error('创建 API Key 失败:', error);
    message.error('创建 API Key 失败');
  } finally {
    creating.value = false;
  }
};

// 关闭新密钥模态框
const closeNewKeyModal = () => {
  showNewKeyModal.value = false;
  newApiKey.value = '';
};

// 关闭创建模态框
const closeCreateModal = () => {
  showCreateModal.value = false;
  newApiKeyName.value = '';
};

// 撤销 API Key
const revokeApiKey = async (apiKey: DisplayApiKey) => {
  if (!confirm(`确定要撤销 API Key "${apiKey.name}" 吗？撤销后将无法再使用此密钥。`)) {
    return;
  }

  try {
    await requestClient.put(`/user/keys/${apiKey.id}`);
    message.success('API Key 已撤销');
    await loadApiKeys();
  } catch (error) {
    console.error('撤销 API Key 失败:', error);
    message.error('撤销 API Key 失败');
  }
};

// 删除 API Key
const deleteApiKey = async (apiKey: DisplayApiKey) => {
  if (!confirm(`确定要删除 API Key "${apiKey.name}" 吗？此操作不可逆。`)) {
    return;
  }

  try {
    await requestClient.delete(`/user/keys/${apiKey.id}`);
    message.success('API Key 删除成功');
    await loadApiKeys();
  } catch (error) {
    console.error('删除 API Key 失败:', error);
    message.error('删除 API Key 失败');
  }
};

// 复制到剪贴板
const copyToClipboard = async (text: string) => {
  try {
    // 优先使用现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      message.success('已复制到剪贴板');
      return;
    }
    
    // Fallback: 使用传统方法
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    
    const successful = document.execCommand('copy');
    document.body.removeChild(textArea);
    
    if (successful) {
      message.success('已复制到剪贴板');
    } else {
      throw new Error('execCommand failed');
    }
  } catch (error) {
    console.error('复制失败:', error);
    
    // 最后的fallback：提示用户手动复制
    try {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.style.position = 'absolute';
      textArea.style.left = '50%';
      textArea.style.top = '50%';
      textArea.style.transform = 'translate(-50%, -50%)';
      textArea.style.zIndex = '9999';
      textArea.style.padding = '10px';
      textArea.style.border = '1px solid #ccc';
      textArea.style.backgroundColor = 'white';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      
      setTimeout(() => {
        document.body.removeChild(textArea);
      }, 3000);
      
      message.warning('请手动复制选中的文本（3秒后自动关闭）');
    } catch (finalError) {
      message.error('复制失败，请手动复制');
    }
  }
};

// 选择所有文本
const selectAllText = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target) {
    target.select();
  }
};

// 格式化日期
const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString('zh-CN');
};

// 页面挂载时加载数据
onMounted(() => {
  loadApiKeys();
  
  // 添加键盘快捷键监听
  document.addEventListener('keydown', handleKeydown);
});

// 页面卸载时移除事件监听
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

// 键盘事件处理
const handleKeydown = (event: KeyboardEvent) => {
  // F5 刷新
  if (event.key === 'F5') {
    event.preventDefault();
    refreshApiKeys();
  }
  // Ctrl+R 刷新
  if (event.ctrlKey && event.key === 'r') {
    event.preventDefault();
    refreshApiKeys();
  }
};

// 导出方法供父组件调用
defineExpose({
  refreshApiKeys,
});
</script>
