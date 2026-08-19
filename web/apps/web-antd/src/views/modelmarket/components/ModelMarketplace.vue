<template>
  <div class="w-full">
    
    <!-- 模型状态统计卡片 -->
    <div class="mb-6 grid grid-cols-2 gap-4 md:grid-cols-4">
      <div class="group rounded-xl bg-gradient-to-br from-green-500/10 to-green-600/5 p-6 text-center border border-green-500/20 hover:border-green-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-green-500/10">
        <div class="text-3xl font-bold text-green-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.serving }}
        </div>
        <div class="text-sm text-green-600 dark:text-green-400 font-medium">{{ $t('business.marketplace.online') }}</div>
        <div class="mt-2 w-full bg-green-500/20 rounded-full h-1">
          <div class="bg-green-500 h-1 rounded-full transition-all duration-500" :style="{ width: `${(modelStats.serving / modelStats.total) * 100}%` }"></div>
        </div>
      </div>
      <div class="group rounded-xl bg-gradient-to-br from-yellow-500/10 to-yellow-600/5 p-6 text-center border border-yellow-500/20 hover:border-yellow-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-yellow-500/10">
        <div class="text-3xl font-bold text-yellow-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.restricted }}
        </div>
        <div class="text-sm text-yellow-600 dark:text-yellow-400 font-medium">{{ $t('business.marketplace.restricted') }}</div>
        <div class="mt-2 w-full bg-yellow-500/20 rounded-full h-1">
          <div class="bg-yellow-500 h-1 rounded-full transition-all duration-500" :style="{ width: `${(modelStats.restricted / modelStats.total) * 100}%` }"></div>
        </div>
      </div>
      <div class="group rounded-xl bg-gradient-to-br from-blue-500/10 to-blue-600/5 p-6 text-center border border-blue-500/20 hover:border-blue-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-blue-500/10">
        <div class="text-3xl font-bold text-blue-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.maintenance }}
        </div>
        <div class="text-sm text-blue-600 dark:text-blue-400 font-medium">{{ $t('business.marketplace.maintenance') }}</div>
        <div class="mt-2 w-full bg-blue-500/20 rounded-full h-1">
          <div class="bg-blue-500 h-1 rounded-full transition-all duration-500" :style="{ width: `${(modelStats.maintenance / modelStats.total) * 100}%` }"></div>
        </div>
      </div>
      <div class="group rounded-xl bg-gradient-to-br from-gray-500/10 to-gray-600/5 p-6 text-center border border-gray-500/20 hover:border-gray-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-gray-500/10">
        <div class="text-3xl font-bold text-gray-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.total }}
        </div>
        <div class="text-sm text-gray-600 dark:text-gray-400 font-medium">{{ $t('business.marketplace.total') }}</div>
        <div class="mt-2 w-full bg-gray-500/20 rounded-full h-1">
          <div class="bg-gray-500 h-1 rounded-full transition-all duration-500" style="width: 100%"></div>
        </div>
      </div>
    </div>
    
    <!-- 模型列表标题 -->
    <div class="mb-6 flex items-center justify-between">
      <div>
        <div class="flex items-center space-x-4">
          <div>
            <h3 class="text-2xl font-bold text-[var(--text-primary)]">
              {{ $t('business.marketplace.modelList') }}
            </h3>
            <p class="mt-1 text-[var(--text-secondary)]">
              {{ allModels.length > 0 ? `${allModels.length} ${$t('business.marketplace.modelList')}` : $t('business.marketplace.noModels') }}
            </p>
          </div>
        </div>
      </div>
      
      <!-- 视图切换按钮 -->
      <div class="flex items-center space-x-2">
        <!-- 网格视图按钮 -->
        <button 
          :class="[
            'p-2 rounded-lg transition-colors',
            viewMode === 'grid' 
              ? 'text-blue-500 bg-blue-500/10' 
              : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--hover-bg)]'
          ]"
          @click="viewMode = 'grid'"
          :title="$t('business.marketplace.gridView')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
          </svg>
        </button>
        <!-- 列表视图按钮 -->
        <button 
          :class="[
            'p-2 rounded-lg transition-colors',
            viewMode === 'list' 
              ? 'text-blue-500 bg-blue-500/10' 
              : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--hover-bg)]'
          ]"
          @click="viewMode = 'list'"
          :title="$t('business.marketplace.listView')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"/>
          </svg>
        </button>
      </div>
    </div>
    
    <!-- 筛选栏：分类按钮 + 搜索框（同一行） -->
    <div class="mb-6 flex flex-wrap items-center gap-3">
      <!-- 分类筛选按钮 -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-for="cat in categoryOptions"
          :key="cat.value"
          @click="toggleCategoryFilter(cat.value)"
          class="inline-flex items-center rounded-full px-3.5 py-1.5 text-xs font-medium transition-all duration-200"
          :class="selectedCategories.includes(cat.value)
            ? 'bg-blue-500 text-white shadow-sm'
            : 'border border-[var(--border-color)] text-[var(--text-secondary)] hover:border-blue-500/40 hover:text-blue-500'">
          <span v-html="cat.icon" class="mr-1"></span>
          {{ cat.label }}
        </button>
      </div>

      <!-- 搜索框（右侧） -->
      <div class="relative ml-auto">
        <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
        </div>
        <input
          :value="localSearchKeyword"
          @input="handleSearchInput"
          type="text"
          :placeholder="$t('business.marketplace.searchPlaceholder')"
          class="w-56 pl-10 pr-4 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-[var(--content-bg)] text-[var(--text-primary)] placeholder-[var(--text-tertiary)] focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-colors"
        >
        <div v-if="localSearchKeyword" class="absolute inset-y-0 right-0 pr-3 flex items-center">
          <button
            @click="clearSearch"
            class="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="flex items-center space-x-3 text-[var(--text-secondary)]">
        <div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
        <span class="font-medium">{{ $t('business.marketplace.loading') }}</span>
      </div>
    </div>
    
    <!-- 模型列表 -->
    <div v-else>
      <!-- 网格视图 -->
      <div v-if="viewMode === 'grid'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div
          v-for="model in displayedModels"
          :key="model.id"
          class="group relative flex min-h-[360px] cursor-pointer flex-col overflow-hidden rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-5 transition-all duration-300 hover:border-blue-500/50 hover:shadow-lg"
          @click="handleModelClick(model)"
        >
          <!-- 悬浮效果背景 -->
          <div class="absolute inset-0 bg-gradient-to-r from-blue-500/5 to-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          
          <div class="relative z-10 flex h-full flex-1 flex-col">
            <!-- 模型图标和状态 -->
            <div class="mb-4 flex items-start justify-between gap-3">
              <div class="flex min-w-0 items-center gap-3">
                <div
                  :style="{ background: `linear-gradient(135deg, ${model.color}, ${model.color}dd)` }"
                  class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-white shadow-sm transition-transform duration-300 group-hover:scale-105"
                >
                <svg class="h-6 w-6" fill="currentColor" viewBox="0 0 24 24">
                  <path v-if="model.icon === 'lucide:brain-circuit'" d="M12 2c5.523 0 10 4.477 10 10s-4.477 10-10 10S2 17.523 2 12 6.477 2 12 2zm0 2a8 8 0 100 16 8 8 0 000-16zm0 3a5 5 0 110 10 5 5 0 010-10zm0 2a3 3 0 100 6 3 3 0 000-6z"/>
                  <path v-else-if="model.icon === 'lucide:cpu'" d="M4 6h16v12H4V6zm2 2v8h12V8H6zm2 2h8v4H8v-4z"/>
                  <path v-else-if="model.icon === 'lucide:message-circle'" d="M12 2C6.477 2 2 6.477 2 12c0 1.89.525 3.66 1.438 5.168L2.546 20.2a1 1 0 001.254 1.254l3.032-.892A9.958 9.958 0 0012 22c5.523 0 10-4.477 10-10S17.523 2 12 2z"/>
                  <path v-else-if="model.icon === 'lucide:bot'" d="M12 2a2 2 0 012 2v1h3a1 1 0 011 1v14a1 1 0 01-1 1H7a1 1 0 01-1-1V6a1 1 0 011-1h3V4a2 2 0 012-2zm-2 5H8v12h8V7h-2v1a1 1 0 01-2 0V7z"/>
                  <path v-else-if="model.icon === 'lucide:code'" d="M8.293 6.293a1 1 0 011.414 0L12 8.586l2.293-2.293a1 1 0 111.414 1.414L13.414 10l2.293 2.293a1 1 0 01-1.414 1.414L12 11.414l-2.293 2.293a1 1 0 01-1.414-1.414L10.586 10 8.293 7.707a1 1 0 010-1.414z"/>
                  <path v-else d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                </svg>
                </div>
                <div class="min-w-0">
                  <h4 class="truncate text-sm font-semibold text-[var(--text-primary)]" :title="model.name">{{ model.name }}</h4>
                  <p class="mt-0.5 text-xs text-[var(--text-secondary)]">{{ model.category || 'LLM' }}</p>
                </div>
              </div>
              <span
                :class="getStatusClass(model.status)"
                class="inline-flex shrink-0 items-center rounded-full border px-2 py-1 text-xs font-medium"
              >
                {{ getStatusText(model.status) }}
              </span>
            </div>
            
            <!-- 模型ID - 紧凑显示并支持复制 -->
            <div class="mb-3 flex h-9 items-center rounded-lg border border-[var(--border-color)] bg-[var(--hover-bg)] px-2.5">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 min-w-0 flex-1">
                  <span class="shrink-0 text-xs font-medium text-[var(--text-secondary)]">ID</span>
                  <code class="truncate font-mono text-xs font-medium text-[var(--text-primary)] transition-colors hover:text-blue-600" 
                        @click.stop="copyToClipboard(model.id)"
                        :title="$t('business.marketplace.copyModelIdWithValue', { id: model.id })">
                    {{ model.id }}
                  </code>
                </div>
                <button 
                  @click.stop="copyToClipboard(model.id)"
                  class="p-1 rounded hover:bg-blue-200 dark:hover:bg-blue-800 transition-colors group/copy flex-shrink-0"
                  :title="$t('business.marketplace.copyModelId')"
                >
                  <svg class="w-3 h-3 text-blue-500 group-hover/copy:scale-110 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                  </svg>
                </button>
              </div>
            </div>
            
            <!-- 模型标签：仅参数量 -->
            <div v-if="model.parameterSize && model.parameterSize !== 'Unknown'" class="mb-3 flex h-7 flex-nowrap items-center gap-1.5 overflow-hidden">
              <span class="inline-flex shrink-0 items-center rounded-md border border-blue-500/20 bg-blue-500/10 px-2 py-1 text-xs font-medium text-blue-500">
                {{ model.parameterSize }}
              </span>
            </div>

            <!-- 价格区间：输入 / 输出 / 缓存输入（¥/M tokens） -->
            <div v-if="model.priceRange" class="mb-3 rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-2.5">
              <div class="mb-1.5 flex items-center text-xs font-medium text-emerald-600">
                <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                {{ $t('business.marketplace.pricing') }}
              </div>
              <div class="grid grid-cols-3 gap-1.5 text-center">
                <div class="rounded bg-emerald-500/10 px-1 py-1">
                  <div class="text-[10px] text-[var(--text-secondary)]">{{ $t('business.marketplace.input') }}</div>
                  <div class="truncate text-xs font-semibold text-emerald-600">{{ formatPricePerMillion(model.priceRange.input) }}</div>
                </div>
                <div class="rounded bg-blue-500/10 px-1 py-1">
                  <div class="text-[10px] text-[var(--text-secondary)]">{{ $t('business.marketplace.output') }}</div>
                  <div class="truncate text-xs font-semibold text-blue-500">{{ formatPricePerMillion(model.priceRange.output) }}</div>
                </div>
                <div class="rounded bg-amber-500/10 px-1 py-1">
                  <div class="text-[10px] text-[var(--text-secondary)]">{{ $t('business.marketplace.cached') }}</div>
                  <div class="truncate text-xs font-semibold text-amber-500">{{ formatPricePerMillion(model.priceRange.cached) }}</div>
                </div>
              </div>
            </div>

            <!-- 模型运行统计：调用情况 + 贡献者情况 -->
            <div class="mb-4 grid grid-cols-2 gap-2">
              <div class="min-w-0 rounded-lg border border-[var(--border-color)] bg-[var(--hover-bg)] p-2.5">
                <div class="flex items-center text-[var(--text-secondary)] text-xs mb-1">
                  <svg class="w-3.5 h-3.5 mr-1 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                  </svg>
                  {{ $t('business.marketplace.calls') }}
                </div>
                <div class="truncate text-base font-semibold text-[var(--text-primary)]">{{ formatNumber(model.totalCalls) }}</div>
                <div class="truncate text-[11px] text-[var(--text-tertiary)]">{{ formatTokens(model.totalTokens) }} tokens · {{ model.avgLatency || 0 }}ms</div>
              </div>
              <div class="min-w-0 rounded-lg border border-[var(--border-color)] bg-[var(--hover-bg)] p-2.5">
                <div class="flex items-center text-[var(--text-secondary)] text-xs mb-1">
                  <svg class="w-3.5 h-3.5 mr-1 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                  </svg>
                  {{ $t('business.marketplace.contributors') }}
                </div>
                <div class="truncate text-base font-semibold text-[var(--text-primary)]">{{ model.contributorCount || 0 }}</div>
                <div class="truncate text-[11px] text-[var(--text-tertiary)]">{{ formatDuration(model.onlineTime) }}</div>
              </div>
            </div>
            
            <!-- 快速操作 -->
            <div class="mt-auto flex items-center justify-end gap-2 border-t border-[var(--border-color)] pt-4">
              <div class="flex items-center gap-2">
                <button
                  class="p-2 rounded-lg bg-blue-500/10 text-blue-500 hover:bg-blue-500/20 transition-colors duration-200"
                  :title="$t('business.marketplace.favorite')"
                  @click.stop="toggleFavorite(model)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"/>
                  </svg>
                </button>
              </div>
              
              <div class="flex items-center space-x-2">
                <button
                  :disabled="isEmbeddingModel(model)"
                  :title="isEmbeddingModel(model) ? $t('business.marketplace.embeddingChatUnavailable') : $t('business.marketplace.startChat')"
                  class="inline-flex h-8 items-center rounded-lg border px-3 text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                  :class="isEmbeddingModel(model) 
                    ? 'bg-gray-500/20 text-gray-500 border-gray-500/30 hover:from-gray-500/20 hover:to-gray-500/20' 
                    : 'bg-gradient-to-r from-green-500 to-emerald-600 hover:from-green-600 hover:to-emerald-700 text-white border-green-400/20'"
                  @click.stop="openChatDialog(model)"
                >
                  <!-- 动画光效 -->
                  <div v-if="!isEmbeddingModel(model)" class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-500"></div>
                  <svg class="mr-1.5 h-3 w-3 relative z-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path v-if="isEmbeddingModel(model)" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636m12.728 12.728L18 21l-1.636-.636m1.636-1.636a9 9 0 01-12.728 0"/>
                    <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                  </svg>
                  <span class="relative z-10">{{ isEmbeddingModel(model) ? $t('business.marketplace.unsupportedChat') : $t('business.marketplace.chat') }}</span>
                </button>
                <button
                  class="inline-flex h-8 items-center rounded-lg bg-blue-500 px-3 text-xs font-medium text-white transition-colors hover:bg-blue-600"
                  @click.stop="handleViewDetails(model)"
                >
                  {{ $t('business.marketplace.details') }}
                  <svg class="ml-1 h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 列表视图 -->
      <div v-else class="space-y-4">
        <div
          v-for="model in displayedModels"
          :key="model.id"
          class="group relative cursor-pointer overflow-hidden rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-5 transition-all duration-300 hover:border-blue-500/50 hover:shadow-lg"
          @click="handleModelClick(model)"
        >
          <!-- 悬浮效果背景 -->
          <div class="absolute inset-0 bg-gradient-to-r from-blue-500/5 to-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          
          <div class="relative z-10 grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-4">
            <!-- 模型图标 -->
            <div class="flex-shrink-0">
              <div
                :style="{ background: `linear-gradient(135deg, ${model.color}, ${model.color}dd)` }"
                class="flex h-12 w-12 items-center justify-center rounded-lg text-white shadow-sm transition-transform duration-300 group-hover:scale-105"
              >
                <svg class="h-6 w-6" fill="currentColor" viewBox="0 0 24 24">
                  <path v-if="model.icon === 'lucide:brain-circuit'" d="M12 2c5.523 0 10 4.477 10 10s-4.477 10-10 10S2 17.523 2 12 6.477 2 12 2zm0 2a8 8 0 100 16 8 8 0 000-16zm0 3a5 5 0 110 10 5 5 0 010-10zm0 2a3 3 0 100 6 3 3 0 000-6z"/>
                  <path v-else-if="model.icon === 'lucide:cpu'" d="M4 6h16v12H4V6zm2 2v8h12V8H6zm2 2h8v4H8v-4z"/>
                  <path v-else-if="model.icon === 'lucide:message-circle'" d="M12 2C6.477 2 2 6.477 2 12c0 1.89.525 3.66 1.438 5.168L2.546 20.2a1 1 0 001.254 1.254l3.032-.892A9.958 9.958 0 0012 22c5.523 0 10-4.477 10-10S17.523 2 12 2z"/>
                  <path v-else-if="model.icon === 'lucide:bot'" d="M12 2a2 2 0 012 2v1h3a1 1 0 011 1v14a1 1 0 01-1 1H7a1 1 0 01-1-1V6a1 1 0 011-1h3V4a2 2 0 012-2zm-2 5H8v12h8V7h-2v1a1 1 0 01-2 0V7z"/>
                  <path v-else-if="model.icon === 'lucide:code'" d="M8.293 6.293a1 1 0 011.414 0L12 8.586l2.293-2.293a1 1 0 111.414 1.414L13.414 10l2.293 2.293a1 1 0 01-1.414 1.414L12 11.414l-2.293 2.293a1 1 0 01-1.414-1.414L10.586 10 8.293 7.707a1 1 0 010-1.414z"/>
                  <path v-else d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                </svg>
              </div>
            </div>
            
            <!-- 模型信息 -->
            <div class="flex-1 min-w-0">
              <div>
                <div>
                  <!-- 模型ID显示区域 -->
                  <div class="mb-3 flex items-center gap-3">
                    <div class="flex h-9 min-w-0 max-w-full items-center gap-2 rounded-lg border border-[var(--border-color)] bg-[var(--hover-bg)] px-2.5">
                      <div class="flex items-center gap-2">
                        <span class="text-xs font-medium text-blue-600 dark:text-blue-400 uppercase tracking-wide">ID:</span>
                        <code class="truncate font-mono text-sm font-medium text-[var(--text-primary)]" :title="model.id">
                          {{ model.id }}
                        </code>
                        <button 
                          @click.stop="copyToClipboard(model.id)"
                          class="p-1 rounded hover:bg-blue-200 dark:hover:bg-blue-800 transition-colors group/copy flex-shrink-0"
                          title="复制模型ID"
                        >
                          <svg class="w-3 h-3 text-blue-500 group-hover/copy:scale-110 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </button>
                      </div>
                    </div>
                  </div>
                  
                  <div class="flex flex-wrap items-center gap-1.5 text-sm">
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-gray-100 dark:bg-gray-800 text-[var(--text-primary)] border border-gray-200 dark:border-gray-700">
                      <span class="text-xs text-gray-500 mr-1">{{ $t('business.marketplace.modelName') }}:</span>{{ model.name }}
                    </span>
                    <!-- 模型类别 -->
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-purple-500/10 text-purple-500 border border-purple-500/20">
                      {{ model.category || 'LLM' }}
                    </span>
                    <!-- 参数量（如果有） -->
                    <span v-if="model.parameterSize && model.parameterSize !== 'Unknown'" class="inline-flex items-center px-3 py-1 rounded-full bg-blue-500/10 text-blue-500 border border-blue-500/20">
                      {{ model.parameterSize }}
                    </span>
                    <!-- 调用量标签 -->
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-orange-500/10 text-orange-500 border border-orange-500/20">
                      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                      </svg>
                      <span class="text-xs">{{ $t('business.marketplace.callsAndTokens', { calls: formatNumber(model.totalCalls), tokens: formatTokens(model.totalTokens) }) }}</span>
                    </span>
                    <!-- 稳定在线时长标签 -->
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-green-500/10 text-green-500 border border-green-500/20">
                      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                      </svg>
                      <span class="text-xs">{{ model.contributorCount || 0 }} {{ $t('business.marketplace.contributors') }} · {{ formatDuration(model.onlineTime) }}</span>
                    </span>
                    <!-- 价格区间：输入 / 输出 / 缓存输入（¥/M tokens） -->
                    <span v-if="model.priceRange" class="inline-flex items-center px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">
                      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                      </svg>
                      <span class="text-xs">{{ $t('business.marketplace.input') }} {{ formatPricePerMillion(model.priceRange.input) }} · {{ $t('business.marketplace.output') }} {{ formatPricePerMillion(model.priceRange.output) }} · {{ $t('business.marketplace.cached') }} {{ formatPricePerMillion(model.priceRange.cached) }}</span>
                    </span>
                </div>
              </div>
              
              <!-- TODO: 模型详细信息 - 暂时注释，后续需要时再启用 -->

              <!-- 快速操作按钮 -->
              <div class="mt-4 flex items-center gap-2 border-t border-[var(--border-color)] pt-4">
                <button
                  :disabled="isEmbeddingModel(model)"
                  :title="isEmbeddingModel(model) ? $t('business.marketplace.embeddingChatUnavailable') : $t('business.marketplace.startChat')"
                  class="inline-flex h-9 items-center rounded-lg border px-4 text-sm font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                  :class="isEmbeddingModel(model) 
                    ? 'bg-gray-500/20 text-gray-500 border-gray-500/30' 
                    : 'bg-gradient-to-r from-green-500 to-emerald-600 hover:from-green-600 hover:to-emerald-700 text-white border-green-400/20'"
                  @click.stop="openChatDialog(model)"
                >
                  <!-- 动画光效 -->
                  <div v-if="!isEmbeddingModel(model)" class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700"></div>
                  <svg class="mr-2 h-4 w-4 relative z-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path v-if="isEmbeddingModel(model)" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636m12.728 12.728L18 21l-1.636-.636m1.636-1.636a9 9 0 01-12.728 0"/>
                    <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                  </svg>
                  <span class="relative z-10">{{ isEmbeddingModel(model) ? $t('business.marketplace.unsupportedChat') : $t('business.marketplace.chat') }}</span>
                </button>
                <button
                  class="inline-flex h-9 items-center rounded-lg bg-blue-500 px-4 text-sm font-medium text-white transition-colors hover:bg-blue-600"
                  @click.stop="handleViewDetails(model)"
                >
                  {{ $t('business.marketplace.details') }}
                  <svg class="ml-1 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                </button>
              </div>
            </div>
            <span
              :class="getStatusClass(model.status)"
              class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium"
            >
              {{ getStatusText(model.status) }}
            </span>
          </div>
        </div>
      </div>
      
      <!-- 模型动态懒加载更多按钮 -->
      <div class="py-8">
        <!-- 加载更多按钮 -->
        <div v-if="hasMore && !loading" class="text-center mb-6">
          <button
            @click="loadMoreModels"
            :disabled="loading"
            class="inline-flex items-center px-8 py-4 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 disabled:from-gray-400 disabled:to-gray-500 text-white text-base font-medium rounded-xl transition-all duration-200 shadow-lg hover:shadow-xl transform hover:-translate-y-1 disabled:transform-none disabled:cursor-not-allowed"
          >
            <svg v-if="!loading" class="mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
            </svg>
            <svg v-else class="mr-2 h-5 w-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            <span v-if="!loading">{{ $t('business.marketplace.loadMore') }}</span>
            <span v-else>{{ $t('business.marketplace.loading') }}</span>
          </button>
          <div class="mt-3 text-sm text-[var(--text-secondary)]">
            {{ $t('business.marketplace.displayedRecords', { displayed: displayedModels.length, total: totalModels }) }}
          </div>
        </div>
        
        <!-- 没有更多数据提示 -->
        <div v-if="!hasMore && displayedModels.length > 0" class="text-center">
          <div class="inline-flex items-center px-6 py-3 bg-[var(--content-bg)] border border-[var(--border-color)] rounded-xl text-[var(--text-secondary)]">
            <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
            {{ $t('business.marketplace.allModelsLoaded') }}
          </div>
        </div>
        
        <!-- 无数据提示 -->
        <div v-if="displayedModels.length === 0 && !loading" class="text-center py-16">
          <div class="w-20 h-20 bg-gradient-to-br from-gray-500/20 to-gray-600/20 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg class="w-10 h-10 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2 2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
            </svg>
          </div>
          <h3 class="text-lg font-medium text-[var(--text-primary)] mb-2">
            {{ $t('business.marketplace.noModelData') }}
          </h3>
          <p class="text-[var(--text-secondary)]">
            {{ localSearchKeyword ? $t('business.marketplace.noMatchingModels') : $t('business.marketplace.noAvailableModels') }}
          </p>
        </div>
        
        <!-- 无搜索结果 -->
        <div v-if="filteredModels.length === 0 && localSearchKeyword" class="text-center py-16">
          <div class="w-20 h-20 bg-[var(--hover-bg)] rounded-full flex items-center justify-center mx-auto mb-4">
            <svg class="w-10 h-10 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6-4h6m2 5.291A7.962 7.962 0 0118 12a8 8 0 10-2.343 5.657l2.343 2.343"/>
            </svg>
          </div>
          <h3 class="text-lg font-medium text-[var(--text-primary)] mb-2">
            {{ $t('business.marketplace.noRelatedModels') }}
          </h3>
          <p class="text-[var(--text-secondary)]">
            {{ $t('business.marketplace.noMatchingModelsWithKeyword', { keyword: localSearchKeyword }) }}
          </p>
        </div>
      </div>
    </div>
  </div>
  </div>
</template>

<script lang="ts" setup>
import type { WorkbenchProjectItem } from '@vben/common-ui';

import { computed, ref, watch, onMounted, onActivated } from 'vue';
import { useRouter } from 'vue-router';
// 导入请求工具
import { requestClient } from '#/api/request';
import { $t } from '#/locales';

// 定义API返回的原始模型数据类型
interface ClientModel {
  name: string;
  type: string;
  size: string;
  arch?: string; // 量化方式
  ippm?: number; // 输入定价 (每百万Token)
  oppm?: number; // 输出定价 (每百万Token)
  cippm?: number; // 缓存命中输入定价 (每百万Token)
  openai_model: {
    created: number;
    id: string;
    object: string;
    owned_by: string;
    permission: null;
    root: string;
    parent: string;
  };
}

interface User {
  id: string;
  username: string;
  email: string;
  role: string;
  created_at: string;
  updated_at: string;
}

interface Client {
  id: string;
  ip: string;
  token: string;
  status: string;
  register_time: string;
  latency: number;
  user_id: string;
  created_at: string;
  updated_at: string;
  models: ClientModel[];
  embedding_models?: ClientModel[];
  user: User;
  inference_engine?: {
    name: string;
    max_tokens: number;
    num_parallel: number;
  };
}

interface ClientModelPair {
  client: Client;
  model: ClientModel;
}

interface ApiModelItem {
  name: string;
  type: string;
  size: string;
  arch?: string; // 量化方式
  client_models: ClientModelPair[];
}

// 定义显示用的模型接口类型
interface ModelItem {
  id: string;
  name: string;
  parameterSize: string;
  modelType: string;
  status: 'serving' | 'restricted' | 'offline' | 'maintenance';
  description: string;
  icon: string;
  color: string;
  createDate: string;
  size: string;
  type: string;
  clientCount?: number; // 可用客户端数量
  onlineClients?: number; // 在线客户端数量
  offlineClients?: number; // 离线客户端数量
  contributorCount?: number; // 贡献者（贡献用户）数量
  totalTokens?: number; // 累计调用 tokens
  totalCalls?: number; // 累计调用次数
  avgLatency?: number; // 平均延迟 (ms)
  onlineTime?: number; // 稳定在线时长（秒）
  priceRange?: {
    input: [number, number];
    output: [number, number];
    cached: [number, number];
  } | null;
  category?: string; // 模型类别：LLM / VLM / Embedding 等
  modalities?: string[];
}

// 定义Props
interface Props {
  searchKeyword?: string;
}

const props = withDefaults(defineProps<Props>(), {
  searchKeyword: '',
});

// 定义事件
const emit = defineEmits<{
  navTo: [item: WorkbenchProjectItem];
}>();

// 路由实例
const router = useRouter();

// 响应式状态
const loading = ref(false);
const currentPage = ref(1);
const pageSize = 12; // 每页显示数量
const viewMode = ref<'grid' | 'list'>('grid'); // 默认网格视图
const sortBy = ref('name');
const sortOrder = ref<'asc' | 'desc'>('asc');
const localSearchKeyword = ref('');

// 处理搜索输入
const handleSearchInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  localSearchKeyword.value = target.value;
};

// 清除搜索
const clearSearch = () => {
  localSearchKeyword.value = '';
};

// 分类筛选选项
const categoryOptions = [
  { value: 'LLM', label: $t('business.marketplace.categoryLlm'), icon: '💬' },
  { value: 'VLM', label: $t('business.marketplace.categoryVlm'), icon: '🖼️' },
  { value: 'Video', label: $t('business.marketplace.categoryVideo'), icon: '🎬' },
  { value: 'Embedding', label: $t('business.marketplace.categoryEmbedding'), icon: '🔤' },
  { value: 'Reranker', label: $t('business.marketplace.categoryReranker'), icon: '⚡' },
  { value: 'ASR', label: $t('business.marketplace.categoryAsr'), icon: '🎤' },
  { value: 'TTS', label: $t('business.marketplace.categoryTts'), icon: '🔊' },
];
const selectedCategories = ref<string[]>([]);

// 搜索关键词监听到分类变化时，重新初始化
watch([() => localSearchKeyword.value, () => selectedCategories.value], () => {
  initializeModels();
});

// 切换分类筛选
const toggleCategoryFilter = (category: string) => {
  const idx = selectedCategories.value.indexOf(category);
  if (idx >= 0) {
    selectedCategories.value.splice(idx, 1);
  } else {
    selectedCategories.value.push(category);
  }
  initializeModels();
};

// 模型数据
const allModels = ref<ModelItem[]>([]); // 已加载的所有模型数据
const totalModels = ref(0); // 服务器端总数量

// 模型广场全局使用统计（调用量/tokens），用于合并展示
const marketStats = ref<Record<string, any>>({});

// 获取模型广场全局使用统计（跨所有用户）
const fetchMarketStats = async () => {
  try {
    const response = await requestClient.get('/market/models/stats');
    if (!response) return;
    const statsArr = Array.isArray(response) ? response : (response.data && Array.isArray(response.data) ? response.data : []);
    const map: Record<string, any> = {};
    statsArr.forEach((s: any) => {
      if (s?.model) map[s.model] = s;
    });
    marketStats.value = map;
  } catch (error) {
    console.warn('获取模型使用统计失败:', error);
  }
};

// 将全局使用统计合并到模型数据中
const mergeMarketStats = (model: ModelItem): ModelItem => {
  const stat = marketStats.value[model.id];
  return {
    ...model,
    totalTokens: stat?.total_tokens ?? model.totalTokens ?? 0,
    totalCalls: stat?.calls ?? model.totalCalls ?? 0,
  };
};

// 数据转换函数：将API数据转换为显示用的模型数据
const transformApiModel = (apiModel: ApiModelItem): ModelItem => {
  // 验证必要字段
  if (!apiModel || typeof apiModel !== 'object') {
    console.warn('Invalid model data:', apiModel);
    return createDefaultModel();
  }

  try {
    // 从模型名称解析信息
    const modelName = apiModel.name || 'Unknown Model';
    
    // 获取第一个客户端模型的其他信息作为默认值
    const firstClientModel = apiModel.client_models?.[0];
    const modelData = firstClientModel?.model;
    const [name, version] = (modelName || '').split(':');

    // 从模型名/标签中解析参数量（如 7b/8b/70b/0.5b），
    // 忽略 latest、v1、fp16 等非参数量标签
    const extractParameterSize = (raw: string): string => {
      const tag = (raw || '').trim().toLowerCase();
      // 匹配形如 7b / 8b / 70b / 0.5b / 1.5b 的参数量
      const sizeMatch = tag.match(/^(\d+(?:\.\d+)?)b$/);
      if (sizeMatch) return `${sizeMatch[1]}B`;
      // 也支持形如 7b-instruct / 8b-q4 等以参数量开头的标签
      const prefixMatch = tag.match(/^(\d+(?:\.\d+)?)b[-_]/);
      if (prefixMatch) return `${prefixMatch[1]}B`;
      return '';
    };

    const parameterSize =
      extractParameterSize(version || '') ||
      extractParameterSize(name || '') ||
      'Unknown';
    
    // 计算文件大小（从字节转换为可读格式）
    const formatSize = (bytes: string | number): string => {
      try {
        const size = typeof bytes === 'string' ? parseInt(bytes) : bytes;
        if (isNaN(size) || size < 0) return '0B';
        
        if (size >= 1024 ** 3) {
          return `${(size / (1024 ** 3)).toFixed(1)}GB`;
        } else if (size >= 1024 ** 2) {
          return `${(size / (1024 ** 2)).toFixed(1)}MB`;
        } else if (size >= 1024) {
          return `${(size / 1024).toFixed(1)}KB`;
        }
        return `${size}B`;
      } catch (error) {
        console.warn('Size formatting error:', bytes, error);
        return '0B';
      }
    };

    // 根据模型类型确定图标和颜色
    const getModelIcon = (type: string, name: string): { icon: string; color: string } => {
      if (type === 'ollama') {
        if (name.includes('qwen') || name.includes('deepseek')) {
          return { icon: 'lucide:brain-circuit', color: '#1890ff' };
        } else if (name.includes('llama')) {
          return { icon: 'lucide:cpu', color: '#52c41a' };
        } else if (name.includes('code')) {
          return { icon: 'lucide:code', color: '#722ed1' };
        } else if (name.includes('chat')) {
          return { icon: 'lucide:message-circle', color: '#faad14' };
        }
      }
      return { icon: 'lucide:bot', color: '#13c2c2' };
    };

    // 确定模型状态：根据客户端状态来判断
    const getModelStatus = (): 'serving' | 'restricted' | 'offline' | 'maintenance' => {
      if (!apiModel.client_models || apiModel.client_models.length === 0) {
        return 'offline';
      }
      
      const onlineClients = apiModel.client_models.filter(cm => cm.client?.status === 'online');
      if (onlineClients.length > 0) {
        return 'serving';
      } else {
        return 'offline';
      }
    };

    const { icon, color } = getModelIcon(apiModel.type || 'unknown', modelName);

    // 计算客户端统计：在线/离线数量、贡献者数量、平均延迟、稳定在线时长
    const clientModels = apiModel.client_models || [];
    const onlineClients = clientModels.filter(cm => cm.client?.status === 'online');
    const offlineClients = clientModels.filter(cm => cm.client?.status !== 'online');
    const contributorSet = new Set<string>();
    clientModels.forEach(cm => {
      if (cm.client?.user?.id) contributorSet.add(cm.client.user.id);
    });
    const avgLatency = clientModels.length > 0
      ? Math.round(clientModels.reduce((sum, cm) => sum + (cm.client?.latency || 0), 0) / clientModels.length)
      : 0;
    const inputPrices = clientModels
      .map(cm => cm.model?.ippm)
      .filter((price): price is number => typeof price === 'number');
    const outputPrices = clientModels
      .map(cm => cm.model?.oppm)
      .filter((price): price is number => typeof price === 'number');
    const cachedPrices = clientModels
      .map(cm => cm.model?.cippm)
      .filter((price): price is number => typeof price === 'number');
    const priceRange = inputPrices.length > 0 && outputPrices.length > 0
      ? {
          input: [Math.min(...inputPrices), Math.max(...inputPrices)] as [number, number],
          output: [Math.min(...outputPrices), Math.max(...outputPrices)] as [number, number],
          cached: cachedPrices.length > 0
            ? [Math.min(...cachedPrices), Math.max(...cachedPrices)] as [number, number]
            : [0, 0] as [number, number],
        }
      : null;
    const category = getModelCategory(modelName, apiModel.type);
    const modalities = isEmbeddingModelName(modelName, apiModel.type)
      ? ['向量嵌入']
      : ['文本输入', '文本输出'];
    // 稳定在线时长：取在线客户端中最早注册时间到现在的时间差（秒）
    let onlineTime = 0;
    if (onlineClients.length > 0) {
      const earliest = onlineClients.reduce((min, cm) => {
        const t = cm.client?.register_time ? new Date(cm.client.register_time).getTime() : 0;
        return min === 0 || (t > 0 && t < min) ? t : min;
      }, 0);
      if (earliest > 0) {
        onlineTime = Math.max(0, Math.floor((Date.now() - earliest) / 1000));
      }
    }

    return {
      id: modelName,
      name: name || modelName,
      parameterSize,
      modelType: (apiModel.type || 'unknown').toUpperCase(),
      status: getModelStatus(),
      description: `${apiModel.type || 'unknown'} 模型，大小：${formatSize(apiModel.size || '0')}，可用客户端：${clientModels.length}个`,
      icon,
      color,
      createDate: modelData?.openai_model?.created ? new Date(modelData.openai_model.created * 1000).toLocaleDateString() : new Date().toLocaleDateString(),
      size: formatSize(apiModel.size || '0'),
      type: apiModel.type || 'unknown',
      clientCount: clientModels.length,
      onlineClients: onlineClients.length,
      offlineClients: offlineClients.length,
      contributorCount: contributorSet.size,
      avgLatency,
      onlineTime,
      priceRange,
      category,
      modalities,
    };
  } catch (error) {
    console.error('转换模型数据时出错:', error, apiModel);
    return createDefaultModel();
  }
};

// 创建默认模型数据
const createDefaultModel = (): ModelItem => {
  return {
    id: 'unknown',
    name: 'Unknown Model',
    parameterSize: 'Unknown',
    modelType: 'UNKNOWN',
    status: 'offline',
    description: '数据异常的模型',
    icon: 'lucide:alert-triangle',
    color: '#ff4d4f',
    createDate: new Date().toLocaleDateString(),
    size: '0B',
    type: 'unknown',
    clientCount: 0,
    onlineClients: 0,
    offlineClients: 0,
    contributorCount: 0,
    avgLatency: 0,
    onlineTime: 0,
    priceRange: null,
    category: 'LLM',
    modalities: [],
  };
};



// API获取模型数据 - 真正的分页版本
const fetchModels = async (
  page: number = 1,
  limit: number = pageSize,
  showLoading = true,
) => {
  try {
    if (showLoading) loading.value = true;
    
    console.log(`获取模型数据：第 ${page} 页，每页 ${limit} 条`);
    
    const response = await requestClient.get('/market/models');
    
    console.log('Models API 响应:', response);
    
    if (!response) {
      console.warn('API 返回空响应');
      return {
        models: [],
        total: 0,
        hasMore: false
      };
    }
    
    // 检查响应是否是数组格式
    let apiModels: ApiModelItem[] = [];
    if (Array.isArray(response)) {
      apiModels = response;
    } else if (response && response.success && response.data) {
      apiModels = response.data.models || response.data || [];
    } else if (response && response.data && Array.isArray(response.data)) {
      apiModels = response.data;
    } else {
      console.error('获取模型数据失败:', response?.message || response?.error || '未知错误');
      return {
        models: [],
        total: 0,
        hasMore: false
      };
    }
    
    // 转换数据格式
    const transformedModels = apiModels.map(transformApiModel).map(mergeMarketStats);
    
    // 应用搜索和筛选
    let filteredModels = transformedModels;
    if (localSearchKeyword.value.trim()) {
      const keyword = localSearchKeyword.value.toLowerCase();
      filteredModels = transformedModels.filter(model => 
        model.name.toLowerCase().includes(keyword) ||
        model.id.toLowerCase().includes(keyword) ||
        model.modelType.toLowerCase().includes(keyword) ||
        model.description.toLowerCase().includes(keyword)
      );
    }
    
    // 分类筛选
    if (selectedCategories.value.length > 0) {
      filteredModels = filteredModels.filter(model => {
        return selectedCategories.value.includes(model.category || 'LLM');
      });
    }
    
    // 排序
    const sortOrderMultiplier = sortOrder.value === 'asc' ? 1 : -1;
    filteredModels.sort((a, b) => {
      switch (sortBy.value) {
        case 'createDate':
          return (new Date(a.createDate).getTime() - new Date(b.createDate).getTime()) * sortOrderMultiplier;
        case 'parameterSize':
          const aNum = parseFloat(a.parameterSize.match(/(\d+(\.\d+)?)/)?.[1] || '0');
          const bNum = parseFloat(b.parameterSize.match(/(\d+(\.\d+)?)/)?.[1] || '0');
          return (aNum - bNum) * sortOrderMultiplier;
        case 'clientCount':
          return ((a.clientCount || 0) - (b.clientCount || 0)) * sortOrderMultiplier;
        default: // name
          return a.name.localeCompare(b.name) * sortOrderMultiplier;
      }
    });
    
    // 保存完整的筛选后数据（用于判断总数）
    const totalFiltered = filteredModels.length;
    
    // 分页处理 - 只返回当前页的数据
    const startIndex = (page - 1) * limit;
    const endIndex = startIndex + limit;
    const paginatedModels = filteredModels.slice(startIndex, endIndex);
    
    console.log(`分页后的模型数据: ${paginatedModels.length} 条，总共 ${totalFiltered} 条，当前页: ${page}`);
    
    return {
      models: paginatedModels,
      total: totalFiltered,
      hasMore: endIndex < totalFiltered
    };
  } catch (error) {
    console.error('获取模型数据失败:', error);
    return {
      models: [],
      total: 0,
      hasMore: false
    };
  } finally {
    if (showLoading) loading.value = false;
  }
};

// 初始化加载模型数据
const initializeModels = async () => {
  console.log('初始化模型数据');
  currentPage.value = 1;
  allModels.value = [];
  
  // 并行获取模型列表和全局使用统计
  const [result] = await Promise.all([fetchModels(1), fetchMarketStats()]);
  allModels.value = result.models;
  totalModels.value = result.total;
  console.log('模型数据加载完成:', result.models.length, '个模型，总计:', result.total);
};

// 加载更多模型数据（点击按钮）
const loadMoreModels = async () => {
  if (loading.value) return;
  
  console.log('加载更多模型数据');
  currentPage.value++;
  
  const result = await fetchModels(currentPage.value);
  // 将新数据追加到现有数据中
  allModels.value.push(...result.models);
  console.log(`加载第 ${currentPage.value} 页，新增 ${result.models.length} 个模型，总计已加载 ${allModels.value.length} 个`);
};

// 根据搜索关键词过滤模型
const filteredModels = computed(() => {
  return allModels.value;
});

// 当前显示的模型
const displayedModels = computed(() => {
  return allModels.value;
});

// 是否还有更多数据
const hasMore = computed(() => {
  if (localSearchKeyword.value.trim()) {
    // 搜索模式下，显示所有匹配结果
    return false;
  }
  
  // 基于已加载数量和服务端总数量判断
  const loadedCount = allModels.value.length;
  const total = totalModels.value;
  
  console.log(`hasMore 计算: 已加载 ${loadedCount}, 总计 ${total}, 是否有更多: ${loadedCount < total}`);
  
  return loadedCount < total && loadedCount > 0;
});

// 计算模型状态统计
const modelStats = computed(() => {
  const stats = {
    serving: 0,
    restricted: 0,
    offline: 0,
    maintenance: 0,
    total: allModels.value.length,
  };
  
  allModels.value.forEach(model => {
    stats[model.status]++;
  });
  
  return stats;
});

// 获取状态样式类
const getStatusClass = (status: ModelItem['status']) => {
  const classes = {
    serving: 'bg-green-500/20 text-green-300 border-green-500/30',
    restricted: 'bg-yellow-500/20 text-yellow-300 border-yellow-500/30',
    maintenance: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
    offline: 'bg-gray-500/20 text-gray-300 border-gray-500/30',
  };
  return classes[status];
};

// 获取状态文本
const getStatusText = (status: ModelItem['status']) => {
  const texts = {
    serving: $t('business.marketplace.serving'),
    restricted: $t('business.marketplace.restricted'),
    maintenance: $t('business.marketplace.maintenance'),
    offline: $t('business.marketplace.offline'),
  };
  return texts[status];
};

// 格式化每百万 token 价格（¥/M tokens）
const formatPricePerMillion = (range: [number, number] | undefined): string => {
  if (!range) return 'N/A';
  const [minimum, maximum] = range;
  if (!Number.isFinite(minimum) || !Number.isFinite(maximum)) return 'N/A';
  if (minimum === 0 && maximum === 0) return 'N/A';
  const formatted = minimum === maximum
    ? minimum.toFixed(2)
    : `${minimum.toFixed(2)}-${maximum.toFixed(2)}`;
  return `¥${formatted}/M`;
};

const isEmbeddingModelName = (name: string, type?: string): boolean => {
  const normalizedName = name.toLowerCase();
  const normalizedType = type?.toLowerCase();
  return normalizedType === 'embedding' || normalizedType === 'embeddings' || [
    'embedding', 'embed', 'bge-', 'text-embedding', 'sentence-transformer',
    'all-minilm', 'e5-', 'gte-', 'multilingual-e5', 'text2vec',
  ].some((keyword) => normalizedName.includes(keyword));
};

// 根据模型名称和类型推断模型类别
const getModelCategory = (name: string, type?: string): string => {
  const normalizedName = name.toLowerCase();
  const normalizedType = type?.toLowerCase();

  // Embedding 模型
  if (normalizedType === 'embedding' || normalizedType === 'embeddings' ||
    ['embedding', 'embed', 'bge-', 'text-embedding', 'sentence-transformer',
      'all-minilm', 'e5-', 'gte-', 'multilingual-e5', 'text2vec', 'reranker',
    ].some((keyword) => normalizedName.includes(keyword))) {
    return 'Embedding';
  }

  // Reranker 模型
  if (['reranker', 'rerank', 're-ranker'].some((keyword) => normalizedName.includes(keyword))) {
    return 'Reranker';
  }

  // 语音识别（ASR）
  if (['whisper', 'asr', 'speech-recognition', 'voice-recognition', 'wav2vec',
    'hubert', 'conformer', 'citrinet', 'quartznet',
  ].some((keyword) => normalizedName.includes(keyword))) {
    return 'ASR';
  }

  // 语音合成（TTS）
  if (['tts', 'speech-synthesis', 'text-to-speech', 'vits', 'bark',
    'cosyvoice', 'chattts', 'gpt-sovits', 'vall-e',
  ].some((keyword) => normalizedName.includes(keyword))) {
    return 'TTS';
  }

  // 视觉语言模型（VLM）：名称包含视觉/图像相关关键词
  if (['vl', 'vision', 'visual', 'image', 'llava', 'qwen2-vl', 'qwen-vl', 'gpt-4o', 'gpt-4v',
    'gemini', 'internvl', 'minicpm-v', 'cogvlm', 'glm-4v', 'llama3.2-vision', 'moondream',
  ].some((keyword) => normalizedName.includes(keyword))) {
    return 'VLM';
  }

  // 视频理解（Video）
  if (['video', 'video-llava', 'video-chat', 'videomae', 'timesformer', 'vivit',
    'internvideo', 'video-blip',
  ].some((keyword) => normalizedName.includes(keyword))) {
    return 'Video';
  }

  // 默认视为 LLM
  return 'LLM';
};

// 检查是否为embedding模型
const isEmbeddingModel = (model: ModelItem): boolean => {
  if (!model) return false;
  return isEmbeddingModelName(model.name || model.id, model.type);
};

// 复制到剪贴板
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    console.log('已复制模型ID:', text);
  } catch (err) {
    // 降级方案：使用传统的复制方法
    const textArea = document.createElement('textarea');
    textArea.value = text;
    document.body.appendChild(textArea);
    textArea.select();
    try {
      document.execCommand('copy');
      console.log('已复制模型ID (降级方案):', text);
    } catch (fallbackErr) {
      console.error('复制失败:', fallbackErr);
    }
    document.body.removeChild(textArea);
  }
};

// 格式化大数字（如 12345 -> 1.2万）
const formatNumber = (num?: number): string => {
  const n = num || 0;
  if (n >= 100000000) return `${(n / 100000000).toFixed(1)}亿`;
  if (n >= 10000) return `${(n / 10000).toFixed(1)}万`;
  return `${n}`;
};

// 格式化 tokens 数量
const formatTokens = (num?: number): string => {
  const n = num || 0;
  if (n >= 1000000000) return `${(n / 1000000000).toFixed(1)}B`;
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
  return `${n}`;
};

// 格式化时长（秒 -> 可读文本）
const formatDuration = (seconds?: number): string => {
  const s = seconds || 0;
  if (s <= 0) return $t('business.marketplace.noUptime');
  if (s < 60) return $t('business.marketplace.uptimeSeconds', { count: s });
  if (s < 3600) return $t('business.marketplace.uptimeMinutes', { count: Math.floor(s / 60) });
  if (s < 86400) return $t('business.marketplace.uptimeHours', { count: Math.floor(s / 3600) });
  return $t('business.marketplace.uptimeDays', { count: Math.floor(s / 86400) });
};

// 切换收藏状态
const toggleFavorite = (model: ModelItem) => {
  console.log('切换收藏状态:', model.name);
  // TODO: 实现收藏功能
};

// 处理模型点击
const handleModelClick = (model: ModelItem) => {
  const projectItem: WorkbenchProjectItem = {
    color: model.color,
    content: model.description,
    date: model.createDate,
    group: model.modelType,
    icon: model.icon,
    title: model.name,
    url: `/model-marketplace-detail?name=${model.id}`,
  };
  emit('navTo', projectItem);
};

// 查看详情
const handleViewDetails = (model: ModelItem) => {
  console.log('查看模型详情:', model);
  // 跳转到新的详情页面
  router.push({
    path: '/model-marketplace-detail',
    query: {
      name: model.id
    }
  });
};

// 对话相关方法
const openChatDialog = (model: ModelItem) => {
  // 检查是否为embedding模型，如果是则不允许对话
  if (isEmbeddingModel(model)) {
    console.warn('Embedding模型不支持对话功能:', model.name);
    return;
  }
  
  // 跳转到对话页面，传递模型信息
  router.push({
    path: '/chat',
    query: {
      modelId: model.id,
      modelName: model.name,
      modelColor: model.color
    }
  });
};

// 组件挂载时初始化数据
onMounted(() => {
  console.log('ModelMarketplace 组件挂载');
  initializeModels();
});

// 暴露刷新方法给父组件
const refreshData = () => {
  console.log('ModelMarketplace 收到刷新指令');
  initializeModels();
};

// 静默刷新：不显示全页 loading，不清空列表，只更新变化的数据，避免页面闪烁
const silentRefresh = async () => {
  try {
    // 并行获取最新模型列表和全局统计
    const [result] = await Promise.all([fetchModels(1, pageSize, false), fetchMarketStats()]);
    const freshModels = result.models;

    // 保留已有数组与卡片对象引用，仅更新发生变化的字段，避免整页重渲染。
    const existingMap = new Map(allModels.value.map(m => [m.id, m]));
    freshModels.forEach(fresh => {
      const existing = existingMap.get(fresh.id);
      if (!existing) {
        allModels.value.push(fresh);
        return;
      }
      // 仅更新可能变化的字段
      existing.status = fresh.status;
      existing.clientCount = fresh.clientCount;
      existing.onlineClients = fresh.onlineClients;
      existing.offlineClients = fresh.offlineClients;
      existing.contributorCount = fresh.contributorCount;
      existing.avgLatency = fresh.avgLatency;
      existing.onlineTime = fresh.onlineTime;
      existing.totalTokens = fresh.totalTokens;
      existing.totalCalls = fresh.totalCalls;
      existing.priceRange = fresh.priceRange;
      existing.category = fresh.category;
      existing.modalities = fresh.modalities;
    });

    totalModels.value = result.total;
  } catch (error) {
    console.warn('静默刷新失败:', error);
  }
};

// 使用 defineExpose 暴露方法
defineExpose({
  refreshData,
  silentRefresh,
});

// 当组件被激活时（例如路由切换后显示）重新加载数据
onActivated(() => {
  console.log('ModelMarketplace 组件被激活');
  refreshData();
});
</script>

<style scoped>
.line-clamp-2 {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
}

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

/* 模型卡片动画效果 */
@keyframes float {
  0%, 100% {
    transform: translateY(0px);
  }
  50% {
    transform: translateY(-2px);
  }
}

.model-card:hover {
  animation: float 2s ease-in-out infinite;
}

/* 骨架屏动画 */
@keyframes skeleton {
  0% {
    background-position: -200px 0;
  }
  100% {
    background-position: calc(200px + 100%) 0;
  }
}

.skeleton {
  background: linear-gradient(90deg, var(--bg-color-secondary) 25%, var(--hover-bg) 50%, var(--bg-color-secondary) 75%);
  background-size: 200px 100%;
  animation: skeleton 1.5s infinite;
}

/* 响应式优化 */
@media (max-width: 768px) {
  .grid-cols-1.md\\:grid-cols-2.lg\\:grid-cols-3.xl\\:grid-cols-4 {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }
  
  .space-x-6 > * + * {
    margin-left: 0.75rem;
  }
  
  .space-x-8 > * + * {
    margin-left: 1rem;
  }
}

@media (max-width: 640px) {
  .flex.items-center.space-x-8 {
    flex-direction: column;
    align-items: flex-start;
    gap: 0;
  }
  
  .flex.items-center.space-x-8 > * + * {
    margin-left: 0;
    margin-top: 0.5rem;
  }
}
</style>
