<template>
  <div class="w-full">
    <!-- 页面头部 -->
    <div class="mb-6 flex items-center justify-between">
      <div class="flex items-center space-x-4">
        <button
          @click="goBack"
          class="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-[var(--text-primary)] bg-[var(--hover-bg)] rounded-lg hover:bg-[var(--border-color)] transition-colors"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
          </svg>
          <span>返回模型广场</span>
        </button>
        <div>
          <h1 class="text-3xl font-bold text-[var(--text-primary)]">
            {{ modelName }}
          </h1>
          <p class="mt-1 text-[var(--text-secondary)]">
            模型详情和客户端列表 ({{ clientModels.length }} 个客户端)
          </p>
        </div>
      </div>
      
      <!-- 刷新按钮 -->
      <button
        @click="refreshData"
        :disabled="loading"
        class="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        <svg class="w-4 h-4" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        <span>{{ loading ? '刷新中...' : '刷新' }}</span>
      </button>
    </div>

    <!-- 模型基本信息卡片 -->
    <div class="mb-6 rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
      <div class="flex items-center space-x-3 mb-6">
        <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
          <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/>
          </svg>
        </div>
        <div>
          <h3 class="text-xl font-bold text-[var(--text-primary)]">{{ modelInfo.name || modelName }}</h3>
          <p class="text-[var(--text-secondary)]">{{ getModelDescription() }}</p>
        </div>
      </div>
      
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-blue-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-blue-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
            </svg>
          </div>
          <div class="text-2xl font-bold text-blue-500">{{ (modelInfo.type || 'Unknown').toUpperCase() }}</div>
          <div class="text-sm text-[var(--text-secondary)]">模型类型</div>
        </div>
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-green-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-green-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"/>
            </svg>
          </div>
          <div class="text-2xl font-bold text-green-500">{{ formatSize(modelInfo.size || '0') }}</div>
          <div class="text-sm text-[var(--text-secondary)]">模型大小</div>
        </div>
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-purple-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-purple-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
            </svg>
          </div>
          <div class="text-2xl font-bold text-purple-500">{{ modelInfo.quantization || 'N/A' }}</div>
          <div class="text-sm text-[var(--text-secondary)]">量化方式</div>
        </div>
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-orange-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-orange-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
            </svg>
          </div>
          <div class="text-2xl font-bold text-orange-500">{{ clientModels.length }}</div>
          <div class="text-sm text-[var(--text-secondary)]">可用客户端</div>
        </div>
      </div>
    </div>

    <!-- 客户端状态统计 -->
    <div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="rounded-xl bg-gradient-to-br from-green-500/10 to-green-600/5 p-4 text-center border border-green-500/20">
        <div class="text-2xl font-bold text-green-500">{{ clientStats.online }}</div>
        <div class="text-sm text-green-600 dark:text-green-400">在线客户端</div>
      </div>
      <div class="rounded-xl bg-gradient-to-br from-red-500/10 to-red-600/5 p-4 text-center border border-red-500/20">
        <div class="text-2xl font-bold text-red-500">{{ clientStats.offline }}</div>
        <div class="text-sm text-red-600 dark:text-red-400">离线客户端</div>
      </div>
      <div class="rounded-xl bg-gradient-to-br from-blue-500/10 to-blue-600/5 p-4 text-center border border-blue-500/20">
        <div class="text-2xl font-bold text-blue-500">{{ clientStats.uniqueUsers }}</div>
        <div class="text-sm text-blue-600 dark:text-blue-400">贡献用户</div>
      </div>
      <div class="rounded-xl bg-gradient-to-br from-purple-500/10 to-purple-600/5 p-4 text-center border border-purple-500/20">
        <div class="text-2xl font-bold text-purple-500">{{ calculateAverageLatency() }}ms</div>
        <div class="text-sm text-purple-600 dark:text-purple-400">平均延迟</div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="flex items-center space-x-3 text-[var(--text-secondary)]">
        <div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
        <span class="font-medium">加载中...</span>
      </div>
    </div>

    <!-- 客户端列表 -->
    <div v-else class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)]">
      <div class="p-6 border-b border-[var(--border-color)]">
        <h3 class="text-lg font-semibold text-[var(--text-primary)]">客户端列表</h3>
        <p class="mt-1 text-[var(--text-secondary)]">所有提供此模型的客户端详情</p>
      </div>

      <!-- 客户端列表内容 -->
      <div class="divide-y divide-[var(--border-color)]">
        <div
          v-for="(clientModel, index) in clientModels"
          :key="clientModel.client.id"
          class="p-6 hover:bg-[var(--hover-bg)] transition-colors"
        >
          <div class="flex items-start justify-between">
            <!-- 左侧信息 -->
            <div class="flex items-start space-x-4 flex-1">
              <!-- 状态指示器 -->
              <div class="flex-shrink-0 mt-1">
                <div
                  :class="[
                    'w-3 h-3 rounded-full',
                    clientModel.client.status === 'online' 
                      ? 'bg-green-500 shadow-green-500/50 shadow-lg' 
                      : 'bg-gray-400'
                  ]"
                ></div>
              </div>

              <!-- 客户端信息 -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center space-x-3 mb-2">
                  <h4 class="text-lg font-medium text-[var(--text-primary)]">
                    客户端 #{{ index + 1 }}
                  </h4>
                  <div class="flex items-center space-x-2">
                    <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                      输入￥{{ clientModel.model.ippm || 10 }}/百万
                    </span>
                    <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
                      输出￥{{ clientModel.model.oppm || 20 }}/百万
                    </span>
                  </div>
                  <span
                    :class="[
                      'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium border',
                      clientModel.client.status === 'online'
                        ? 'bg-green-500/10 text-green-500 border-green-500/20'
                        : 'bg-gray-500/10 text-gray-500 border-gray-500/20'
                    ]"
                  >
                    {{ clientModel.client.status === 'online' ? '在线' : '离线' }}
                  </span>
                </div>

                <!-- 用户信息 -->
                <div class="mb-4 p-4 rounded-lg bg-gradient-to-r from-blue-500/5 to-purple-500/5 border border-blue-500/20">
                  <div class="flex items-center space-x-3 mb-3">
                    <div class="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                      <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                      </svg>
                    </div>
                    <div>
                      <h5 class="font-semibold text-[var(--text-primary)]">{{ clientModel.client.user.username }}</h5>
                      <p class="text-sm text-[var(--text-secondary)]">贡献用户</p>
                    </div>
                    <div class="ml-auto">
                      <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
                        {{ clientModel.client.user.role }}
                      </span>
                    </div>
                  </div>
                  <div class="grid grid-cols-2 gap-4 text-sm">
                    <div class="space-y-2">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">邮箱:</span>
                        <span class="font-medium text-[var(--text-primary)] truncate">{{ clientModel.client.user.email }}</span>
                      </div>
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">注册时间:</span>
                        <span class="font-medium text-[var(--text-primary)]">{{ formatDate(clientModel.client.user.created_at) }}</span>
                      </div>
                    </div>
                    <div class="space-y-2">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">用户ID:</span>
                        <span class="font-mono text-xs text-[var(--text-primary)]">{{ clientModel.client.user.id }}</span>
                      </div>
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">更新时间:</span>
                        <span class="font-medium text-[var(--text-primary)]">{{ formatDate(clientModel.client.user.updated_at) }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 客户端详细信息 -->
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <span class="text-[var(--text-secondary)]">客户端ID:</span>
                    <div class="mt-1 font-mono text-xs text-[var(--text-primary)] break-all">
                      {{ clientModel.client.id.substring(0, 16) }}...
                    </div>
                  </div>
                  <div>
                    <span class="text-[var(--text-secondary)]">注册时间:</span>
                    <div class="mt-1 font-medium text-[var(--text-primary)]">
                      {{ formatDate(clientModel.client.register_time) }}
                    </div>
                  </div>
                  <div>
                    <span class="text-[var(--text-secondary)]">延迟:</span>
                    <div class="mt-1 font-medium text-[var(--text-primary)]">
                      {{ clientModel.client.latency }}ms
                    </div>
                  </div>
                  <div>
                    <span class="text-[var(--text-secondary)]">更新时间:</span>
                    <div class="mt-1 font-medium text-[var(--text-primary)]">
                      {{ formatDate(clientModel.client.updated_at) }}
                    </div>
                  </div>
                </div>

                <!-- 模型详细信息 -->
                <div class="mt-4 p-4 rounded-lg bg-gradient-to-r from-green-500/5 to-blue-500/5 border border-green-500/20">
                  <div class="flex items-center space-x-2 mb-3">
                    <svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/>
                    </svg>
                    <span class="font-semibold text-[var(--text-primary)]">模型规格详情</span>
                  </div>
                  <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">模型名称</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)] break-all">{{ clientModel.model.name }}</div>
                    </div>
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">模型类型</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)]">{{ clientModel.model.type.toUpperCase() }}</div>
                    </div>
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">模型大小</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)]">{{ formatSize(clientModel.model.size) }}</div>
                    </div>
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">量化架构</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)]">
                        <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-500/10 text-orange-500 border border-orange-500/20">
                          {{ clientModel.model.quantization || clientModel.model.arch || 'N/A' }}
                        </span>
                      </div>
                    </div>
                  </div>
                  
                  <!-- 此客户端的定价信息 -->
                  <div class="mt-4 pt-4 border-t border-emerald-500/20">
                    <div class="flex items-center space-x-2 mb-3">
                      <svg class="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1"/>
                      </svg>
                      <span class="font-semibold text-[var(--text-primary)]">Token 定价</span>
                    </div>
                    <div class="grid grid-cols-2 gap-4">
                      <div class="text-center p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                        <div class="text-xl font-bold text-emerald-500">￥{{ clientModel.model.ippm || 10 }}</div>
                        <div class="text-xs text-emerald-600 dark:text-emerald-400">输入Token/百万</div>
                      </div>
                      <div class="text-center p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                        <div class="text-xl font-bold text-blue-500">￥{{ clientModel.model.oppm || 20 }}</div>
                        <div class="text-xs text-blue-600 dark:text-blue-400">输出Token/百万</div>
                      </div>
                    </div>
                  </div>
                  
                  <!-- OpenAI Model 信息 (如果存在) -->
                  <div v-if="clientModel.model.openai_model && (clientModel.model.openai_model.id || clientModel.model.openai_model.owned_by)" class="mt-4 pt-3 border-t border-[var(--border-color)]">
                    <div class="text-sm">
                      <div class="flex items-center space-x-2 mb-2">
                        <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="font-medium text-[var(--text-primary)]">OpenAI 模型信息</span>
                      </div>
                      <div class="grid grid-cols-2 gap-4 text-xs">
                        <div v-if="clientModel.model.openai_model.id">
                          <span class="text-[var(--text-secondary)]">模型ID:</span>
                          <span class="ml-2 font-mono text-[var(--text-primary)]">{{ clientModel.model.openai_model.id }}</span>
                        </div>
                        <div v-if="clientModel.model.openai_model.owned_by">
                          <span class="text-[var(--text-secondary)]">所有者:</span>
                          <span class="ml-2 font-medium text-[var(--text-primary)]">{{ clientModel.model.openai_model.owned_by }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 右侧操作按钮 -->
            <div class="flex flex-col space-y-2 ml-4">
              <button
                :disabled="clientModel.client.status !== 'online'"
                class="px-3 py-1 text-xs font-medium rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                :class="clientModel.client.status === 'online' 
                  ? 'bg-green-500/10 text-green-500 border border-green-500/20 hover:bg-green-500/20' 
                  : 'bg-gray-500/10 text-gray-500 border border-gray-500/20'"
              >
                {{ clientModel.client.status === 'online' ? '可用' : '离线' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="clientModels.length === 0 && !loading" class="p-12 text-center">
        <div class="w-20 h-20 bg-[var(--hover-bg)] rounded-full flex items-center justify-center mx-auto mb-4">
          <svg class="w-10 h-10 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
          </svg>
        </div>
        <h3 class="text-xl font-semibold text-[var(--text-primary)] mb-2">
          {{ modelName ? '暂无客户端数据' : '模型未找到' }}
        </h3>
        <p class="text-[var(--text-secondary)] mb-4">
          {{ modelName 
            ? `模型 "${modelName}" 当前没有可用的客户端，可能是网络问题或模型尚未部署。` 
            : '请检查 URL 中的模型名称参数是否正确。' 
          }}
        </p>
        <div class="flex flex-col sm:flex-row gap-3 justify-center">
          <button
            @click="refreshData"
            class="inline-flex items-center px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            重新加载
          </button>
          <button
            @click="goBack"
            class="inline-flex items-center px-4 py-2 bg-[var(--content-bg)] text-[var(--text-primary)] border border-[var(--border-color)] rounded-lg hover:bg-[var(--hover-bg)] transition-colors"
          >
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
            </svg>
            返回模型广场
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, onMounted, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { requestClient } from '#/api/request';

const router = useRouter();
const route = useRoute();

// 响应式状态
const loading = ref(false);
const clientModels = ref<any[]>([]);
const modelInfo = ref<any>({});

// 从路由参数获取模型名称
const modelName = computed(() => route.query.name as string || '');

// 计算客户端统计
const clientStats = computed(() => {
  const stats = {
    online: 0,
    offline: 0,
    uniqueUsers: new Set<string>(),
    total: clientModels.value.length
  };

  clientModels.value.forEach(cm => {
    if (cm.client.status === 'online') {
      stats.online++;
    } else {
      stats.offline++;
    }
    stats.uniqueUsers.add(cm.client.user.id);
  });

  return {
    ...stats,
    uniqueUsers: stats.uniqueUsers.size
  };
});

// 格式化文件大小
const formatSize = (bytes: string | number): string => {
  const size = typeof bytes === 'string' ? parseInt(bytes) : bytes;
  if (isNaN(size) || size <= 0) return 'N/A';
  
  if (size >= 1024 ** 3) {
    return `${(size / (1024 ** 3)).toFixed(1)}GB`;
  } else if (size >= 1024 ** 2) {
    return `${(size / (1024 ** 2)).toFixed(1)}MB`;
  } else if (size >= 1024) {
    return `${(size / 1024).toFixed(1)}KB`;
  }
  return `${size}B`;
};

// 获取模型描述
const getModelDescription = (): string => {
  if (!modelInfo.value || Object.keys(modelInfo.value).length === 0) {
    return '加载中...';
  }
  
  const { type, quantization, size } = modelInfo.value;
  const sizeFormatted = formatSize(size || '0');
  const typeText = type ? type.toUpperCase() : 'Unknown';
  const quantText = quantization || 'N/A';
  
  return `${typeText} 模型 • 量化: ${quantText} • 大小: ${sizeFormatted}`;
};

// 格式化日期
const formatDate = (dateString: string): string => {
  if (!dateString || dateString === '0001-01-01T00:00:00Z') {
    return 'N/A';
  }
  return new Date(dateString).toLocaleString('zh-CN');
};

// 计算平均延迟
const calculateAverageLatency = (): number => {
  if (clientModels.value.length === 0) return 0;
  const totalLatency = clientModels.value.reduce((sum, cm) => sum + (cm.client.latency || 0), 0);
  return Math.round(totalLatency / clientModels.value.length);
};

// 获取模型客户端详情
const fetchModelDetails = async () => {
  try {
    loading.value = true;
    console.log('正在获取模型详情，模型名称:', modelName.value);
    
    const response = await requestClient.get('/market/models');
    console.log('API 响应:', response);
    
    if (!response) {
      console.warn('API 返回空响应');
      clientModels.value = [];
      modelInfo.value = {};
      return;
    }
    
    // 处理不同的响应格式
    let modelsData: any[] = [];
    
    if (Array.isArray(response)) {
      modelsData = response;
    } else if (response.data && Array.isArray(response.data)) {
      modelsData = response.data;
    } else if (response.success && response.data && Array.isArray(response.data.models)) {
      modelsData = response.data.models;
    } else {
      console.error('无法解析响应数据格式:', response);
      clientModels.value = [];
      modelInfo.value = {};
      return;
    }
    
    // 查找匹配的模型
    const model = modelsData.find(m => m.name === modelName.value);
    
    if (model) {
      console.log('找到模型:', model);
      clientModels.value = model.client_models || [];
      modelInfo.value = {
        name: model.name,
        type: model.type,
        size: model.size,
        quantization: model.quantization,
        client_models: model.client_models
      };
      
      console.log('解析的模型信息:', modelInfo.value);
      console.log('客户端列表:', clientModels.value);
    } else {
      console.warn('未找到指定模型:', modelName.value);
      console.log('可用模型列表:', modelsData.map(m => m.name));
      clientModels.value = [];
      modelInfo.value = {};
    }
  } catch (error) {
    console.error('获取模型详情失败:', error);
    clientModels.value = [];
    modelInfo.value = {};
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshData = () => {
  fetchModelDetails();
};

// 返回上一页
const goBack = () => {
  router.push('/model-marketplace');
};

// 监听模型名称变化
watch(() => modelName.value, () => {
  if (modelName.value) {
    fetchModelDetails();
  }
}, { immediate: true });

// 组件挂载时获取数据
onMounted(() => {
  if (modelName.value) {
    fetchModelDetails();
  }
});
</script>

<style scoped>
/* 自定义滚动条样式 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: var(--border-color);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb {
  background: var(--text-secondary);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-primary);
}

/* 响应式优化 */
@media (max-width: 768px) {
  .grid-cols-2.md\\:grid-cols-4 {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }
}
</style>
