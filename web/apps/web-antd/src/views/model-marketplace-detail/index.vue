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
          <span>{{ $t('business.marketplace.back') }}</span>
        </button>
        <div>
          <h1 class="text-3xl font-bold text-[var(--text-primary)]">
            {{ modelName }}
          </h1>
          <p class="mt-1 text-[var(--text-secondary)]">
            {{ $t('business.marketplace.clientList') }} ({{ clientModels.length }})
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
        <span>{{ loading ? $t('business.marketplace.refreshing') : $t('business.marketplace.refresh') }}</span>
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
          <div class="text-sm text-[var(--text-secondary)]">{{ $t('business.marketplace.modelType') }}</div>
        </div>
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-green-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-green-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"/>
            </svg>
          </div>
          <div class="text-2xl font-bold text-green-500">{{ parameterSize }}</div>
          <div class="text-sm text-[var(--text-secondary)]">{{ $t('business.marketplace.parameters') }}</div>
        </div>
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-purple-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-purple-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
            </svg>
          </div>
          <div class="text-2xl font-bold text-purple-500">{{ maxContext }}</div>
          <div class="text-sm text-[var(--text-secondary)]">{{ $t('business.marketplace.maxContext') }}</div>
        </div>
        <div class="text-center p-4 rounded-lg bg-[var(--hover-bg)] border border-orange-500/20">
          <div class="w-8 h-8 mx-auto mb-2 rounded-lg bg-orange-500/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
            </svg>
          </div>
          <div class="text-sm font-bold text-orange-500">{{ supportedModalities.join(' / ') }}</div>
          <div class="text-sm text-[var(--text-secondary)]">{{ $t('business.marketplace.modalities') }}</div>
        </div>
      </div>
    </div>

    <!-- 贡献者报价与调用分析 -->
    <div class="mb-6 grid grid-cols-1 gap-4 xl:grid-cols-2">
      <section class="rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-5">
        <div class="mb-4 flex items-center gap-2">
          <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10">
            <svg class="h-4 w-4 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8V7m0 9v1m9-5a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
          </div>
          <div><h3 class="text-base font-semibold text-[var(--text-primary)]">{{ $t('business.marketplace.contributorPrices') }}</h3><p class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.priceRangeDescription') }}</p></div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="rounded-lg bg-[var(--hover-bg)] p-3"><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.input') }}</div><div class="mt-1 font-semibold text-emerald-500">{{ formatPricePerMillion(contributorPriceRange.input) }}</div></div>
          <div class="rounded-lg bg-[var(--hover-bg)] p-3"><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.output') }}</div><div class="mt-1 font-semibold text-blue-500">{{ formatPricePerMillion(contributorPriceRange.output) }}</div></div>
          <div class="rounded-lg bg-[var(--hover-bg)] p-3"><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.cached') }}</div><div class="mt-1 font-semibold text-amber-500">{{ formatPricePerMillion(contributorPriceRange.cached) }}</div></div>
        </div>
      </section>
      <section class="rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-5">
        <div class="mb-4"><h3 class="text-base font-semibold text-[var(--text-primary)]">{{ $t('business.marketplace.usageAnalysis') }}</h3><p class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.usageDescription') }}</p></div>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.calls') }}</div><div class="mt-1 text-lg font-semibold text-[var(--text-primary)]">{{ formatNumber(modelUsage.calls) }}</div></div>
          <div><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.totalTokens') }}</div><div class="mt-1 text-lg font-semibold text-[var(--text-primary)]">{{ formatTokens(modelUsage.total_tokens) }}</div></div>
          <div><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.callers') }}</div><div class="mt-1 text-lg font-semibold text-[var(--text-primary)]">{{ formatNumber(modelUsage.user_count) }}</div></div>
          <div><div class="text-xs text-[var(--text-secondary)]">{{ $t('business.marketplace.callingClients') }}</div><div class="mt-1 text-lg font-semibold text-[var(--text-primary)]">{{ formatNumber(modelUsage.client_count) }}</div></div>
        </div>
      </section>
    </div>

    <!-- 客户端状态统计 -->
    <div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="rounded-xl bg-gradient-to-br from-green-500/10 to-green-600/5 p-4 text-center border border-green-500/20">
        <div class="text-2xl font-bold text-green-500">{{ clientStats.online }}</div>
        <div class="text-sm text-green-600 dark:text-green-400">{{ $t('business.marketplace.onlineClients') }}</div>
      </div>
      <div class="rounded-xl bg-gradient-to-br from-red-500/10 to-red-600/5 p-4 text-center border border-red-500/20">
        <div class="text-2xl font-bold text-red-500">{{ clientStats.offline }}</div>
        <div class="text-sm text-red-600 dark:text-red-400">{{ $t('business.marketplace.offlineClients') }}</div>
      </div>
      <div class="rounded-xl bg-gradient-to-br from-blue-500/10 to-blue-600/5 p-4 text-center border border-blue-500/20">
        <div class="text-2xl font-bold text-blue-500">{{ clientStats.uniqueUsers }}</div>
        <div class="text-sm text-blue-600 dark:text-blue-400">{{ $t('business.marketplace.contributorUsers') }}</div>
      </div>
      <div class="rounded-xl bg-gradient-to-br from-purple-500/10 to-purple-600/5 p-4 text-center border border-purple-500/20">
        <div class="text-2xl font-bold text-purple-500">{{ calculateAverageLatency() }}ms</div>
        <div class="text-sm text-purple-600 dark:text-purple-400">{{ $t('business.marketplace.averageLatency') }}</div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="flex items-center space-x-3 text-[var(--text-secondary)]">
        <div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
        <span class="font-medium">{{ $t('business.marketplace.loading') }}</span>
      </div>
    </div>

    <!-- 客户端列表 -->
    <div v-else class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)]">
      <div class="p-6 border-b border-[var(--border-color)]">
        <h3 class="text-lg font-semibold text-[var(--text-primary)]">{{ $t('business.marketplace.clientList') }}</h3>
        <p class="mt-1 text-[var(--text-secondary)]">{{ $t('business.marketplace.clientListDescription') }}</p>
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
                    {{ $t('business.marketplace.client') }} #{{ index + 1 }}
                  </h4>
                  <div class="flex items-center space-x-2">
                    <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                      {{ $t('business.marketplace.inputPerMillionShort', { price: clientModel.model.ippm || 0 }) }}
                    </span>
                    <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
                      {{ $t('business.marketplace.outputPerMillionShort', { price: clientModel.model.oppm || 0 }) }}
                    </span>
                    <span v-if="(clientModel.model.cippm || 0) > 0" class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-500/10 text-orange-500 border border-orange-500/20">
                      {{ $t('business.marketplace.cachedPerMillionShort', { price: clientModel.model.cippm }) }}
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
                    {{ clientModel.client.status === 'online' ? $t('business.marketplace.online') : $t('business.marketplace.offline') }}
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
                      <p class="text-sm text-[var(--text-secondary)]">{{ $t('business.marketplace.contributor') }}</p>
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
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.email') }}:</span>
                        <span class="font-medium text-[var(--text-primary)] truncate">{{ clientModel.client.user.email }}</span>
                      </div>
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.registeredAt') }}:</span>
                        <span class="font-medium text-[var(--text-primary)]">{{ formatDate(clientModel.client.user.created_at) }}</span>
                      </div>
                    </div>
                    <div class="space-y-2">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.clientId') }}:</span>
                        <span class="font-mono text-xs text-[var(--text-primary)]">{{ clientModel.client.user.id }}</span>
                      </div>
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.updatedAt') }}:</span>
                        <span class="font-medium text-[var(--text-primary)]">{{ formatDate(clientModel.client.user.updated_at) }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 客户端详细信息 -->
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.clientId') }}:</span>
                    <div class="mt-1 font-mono text-xs text-[var(--text-primary)] break-all">
                      {{ clientModel.client.id.substring(0, 16) }}...
                    </div>
                  </div>
                  <div>
                    <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.registeredAt') }}:</span>
                    <div class="mt-1 font-medium text-[var(--text-primary)]">
                      {{ formatDate(clientModel.client.register_time) }}
                    </div>
                  </div>
                  <div>
                    <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.latency') }}:</span>
                    <div class="mt-1 font-medium text-[var(--text-primary)]">
                      {{ clientModel.client.latency }}ms
                    </div>
                  </div>
                  <div>
                    <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.updatedAt') }}:</span>
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
                    <span class="font-semibold text-[var(--text-primary)]">{{ $t('business.marketplace.modelDetails') }}</span>
                  </div>
                  <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.modelName') }}</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)] break-all">{{ clientModel.model.name }}</div>
                    </div>
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.modelType') }}</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)]">{{ clientModel.model.type.toUpperCase() }}</div>
                    </div>
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.modelSize') }}</span>
                      </div>
                      <div class="font-medium text-[var(--text-primary)]">{{ formatSize(clientModel.model.size) }}</div>
                    </div>
                    <div class="space-y-1">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                        </svg>
                        <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.quantizationArchitecture') }}</span>
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
                      <span class="font-semibold text-[var(--text-primary)]">{{ $t('business.marketplace.pricing') }}</span>
                    </div>
                    <div class="grid grid-cols-3 gap-4">
                      <div class="text-center p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                        <div class="text-xl font-bold text-emerald-500">¥{{ clientModel.model.ippm || 0 }}/M</div>
                        <div class="text-xs text-emerald-600 dark:text-emerald-400">{{ $t('business.marketplace.inputPerMillion') }}</div>
                      </div>
                      <div class="text-center p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                        <div class="text-xl font-bold text-blue-500">¥{{ clientModel.model.oppm || 0 }}/M</div>
                        <div class="text-xs text-blue-600 dark:text-blue-400">{{ $t('business.marketplace.outputPerMillion') }}</div>
                      </div>
                      <div class="text-center p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                        <div class="text-xl font-bold text-amber-500">¥{{ clientModel.model.cippm || 0 }}/M</div>
                        <div class="text-xs text-amber-600 dark:text-amber-400">{{ $t('business.marketplace.inputPerMillion') }} ({{ $t('business.marketplace.cached') }})</div>
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
                        <span class="font-medium text-[var(--text-primary)]">{{ $t('business.marketplace.openaiModelInfo') }}</span>
                      </div>
                      <div class="grid grid-cols-2 gap-4 text-xs">
                        <div v-if="clientModel.model.openai_model.id">
                          <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.modelId') }}:</span>
                          <span class="ml-2 font-mono text-[var(--text-primary)]">{{ clientModel.model.openai_model.id }}</span>
                        </div>
                        <div>
                          <span class="text-[var(--text-secondary)]">{{ $t('business.marketplace.apiSupportType') }}:</span>
                          <span class="ml-2 font-medium text-[var(--text-primary)]">{{ $t('business.marketplace.openaiCompletionApi') }}</span>
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
                {{ clientModel.client.status === 'online' ? $t('business.marketplace.available') : $t('business.marketplace.offline') }}
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
          {{ modelName ? $t('business.marketplace.noClientData') : $t('business.marketplace.modelNotFound') }}
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
            {{ $t('business.marketplace.reload') }}
          </button>
          <button
            @click="goBack"
            class="inline-flex items-center px-4 py-2 bg-[var(--content-bg)] text-[var(--text-primary)] border border-[var(--border-color)] rounded-lg hover:bg-[var(--hover-bg)] transition-colors"
          >
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
            </svg>
            {{ $t('business.marketplace.back') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { requestClient } from '#/api/request';
import { $t } from '#/locales';

const router = useRouter();
const route = useRoute();

// 响应式状态
const loading = ref(false);
const clientModels = ref<any[]>([]);
const modelInfo = ref<any>({});
const modelUsage = ref({ calls: 0, total_tokens: 0, user_count: 0, client_count: 0 });

// 从路由参数获取模型名称
const modelName = computed(() => route.query.name as string || '');

const parameterSize = computed(() => modelName.value.split(':')[1] || $t('business.marketplace.parameters'));
// 支持的最大上下文：默认输入 128K，输出 32K
const maxContext = computed(() => {
  const input = 128 * 1024;
  const output = 32 * 1024;
  return `${formatTokens(input)} / ${formatTokens(output)}`;
});
const supportedModalities = computed(() => isEmbeddingModel()
  ? [$t('business.marketplace.embedding')]
  : [$t('business.marketplace.textInput'), $t('business.marketplace.textOutput')]);
const contributorPriceRange = computed(() => ({
  input: getPriceRange('ippm'),
  output: getPriceRange('oppm'),
  cached: getPriceRange('cippm'),
}));

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
    return $t('business.marketplace.loading');
  }
  
  const { type } = modelInfo.value;
  const typeText = type ? type.toUpperCase() : 'Unknown';
  return `${typeText} • ${$t('business.marketplace.parameters')}: ${parameterSize.value} • ${$t('business.marketplace.maxContext')}: ${maxContext.value}`;
};

const isEmbeddingModel = (): boolean => {
  const name = modelName.value.toLowerCase();
  const type = modelInfo.value?.type?.toLowerCase();
  return type === 'embedding' || type === 'embeddings' || [
    'embedding', 'embed', 'bge-', 'text-embedding', 'sentence-transformer',
    'all-minilm', 'e5-', 'gte-', 'multilingual-e5', 'text2vec',
  ].some((keyword) => name.includes(keyword));
};

const getPriceRange = (field: 'ippm' | 'oppm' | 'cippm'): [number, number] | null => {
  const prices = clientModels.value
    .map((item) => item.model?.[field])
    .filter((price): price is number => typeof price === 'number');
  return prices.length > 0 ? [Math.min(...prices), Math.max(...prices)] : null;
};

// 格式化每百万 token 价格（¥/M tokens）
const formatPricePerMillion = (range: [number, number] | null): string => {
  if (!range) return $t('business.marketplace.noPrice');
  const [minimum, maximum] = range;
  if (minimum === 0 && maximum === 0) return $t('business.marketplace.noPrice');
  const formatted = minimum === maximum
    ? minimum.toFixed(2)
    : `${minimum.toFixed(2)} - ${maximum.toFixed(2)}`;
  return `¥${formatted}/M`;
};

const formatNumber = (value?: number): string => (value || 0).toLocaleString('zh-CN');
const formatTokens = (value?: number): string => {
  const amount = value || 0;
  if (amount >= 1_000_000_000) return `${(amount / 1_000_000_000).toFixed(1)}B`;
  if (amount >= 1_000_000) return `${(amount / 1_000_000).toFixed(1)}M`;
  if (amount >= 1_000) return `${(amount / 1_000).toFixed(1)}K`;
  return `${amount}`;
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

// 获取模型客户端详情（silent 为 true 时不显示全页 loading，避免闪烁）
const fetchModelDetails = async (silent = false) => {
  try {
    if (!silent) loading.value = true;
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
    if (!silent) loading.value = false;
  }
};

const fetchModelUsage = async () => {
  if (!modelName.value) return;
  try {
    const response = await requestClient.get('/market/models/stats');
    const stats = Array.isArray(response) ? response : response?.data;
    const found = Array.isArray(stats) ? stats.find((item) => item.model === modelName.value) : null;
    modelUsage.value = found ?? { calls: 0, total_tokens: 0, user_count: 0, client_count: 0 };
  } catch (error) {
    console.warn('获取模型调用统计失败:', error);
  }
};

// 刷新数据
const refreshData = () => {
  fetchModelDetails();
  fetchModelUsage();
};

// 静默刷新：不显示全页 loading，只更新变化的数据，避免页面闪烁
const silentRefresh = () => {
  fetchModelDetails(true);
  fetchModelUsage();
};

// 返回上一页
const goBack = () => {
  router.push('/model-marketplace');
};

// 自动静默刷新：定时更新客户端状态等动态数据，不显示全页 loading
let autoRefreshTimer: ReturnType<typeof setInterval> | null = null;
const startAutoRefresh = () => {
  stopAutoRefresh();
  autoRefreshTimer = setInterval(() => {
    silentRefresh();
  }, 15 * 1000);
};
const stopAutoRefresh = () => {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer);
    autoRefreshTimer = null;
  }
};

// 监听模型名称变化
watch(() => modelName.value, () => {
  if (modelName.value) {
    fetchModelDetails();
    fetchModelUsage();
    startAutoRefresh();
  } else {
    stopAutoRefresh();
  }
}, { immediate: true });

// 组件挂载时获取数据
onMounted(() => {
  if (modelName.value) {
    fetchModelDetails();
    fetchModelUsage();
    startAutoRefresh();
  }
});

// 组件卸载时清理定时器
onUnmounted(() => {
  stopAutoRefresh();
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
