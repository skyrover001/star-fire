<template>
  <div class="w-full">
    
    <!-- 模型状态统计卡片 -->
    <div class="mb-6 grid grid-cols-2 gap-4 md:grid-cols-4">
      <div class="group rounded-xl bg-gradient-to-br from-green-500/10 to-green-600/5 p-6 text-center border border-green-500/20 hover:border-green-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-green-500/10">
        <div class="text-3xl font-bold text-green-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.serving }}
        </div>
        <div class="text-sm text-green-600 dark:text-green-400 font-medium">服务中</div>
        <div class="mt-2 w-full bg-green-500/20 rounded-full h-1">
          <div class="bg-green-500 h-1 rounded-full transition-all duration-500" :style="{ width: `${(modelStats.serving / modelStats.total) * 100}%` }"></div>
        </div>
      </div>
      <div class="group rounded-xl bg-gradient-to-br from-yellow-500/10 to-yellow-600/5 p-6 text-center border border-yellow-500/20 hover:border-yellow-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-yellow-500/10">
        <div class="text-3xl font-bold text-yellow-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.restricted }}
        </div>
        <div class="text-sm text-yellow-600 dark:text-yellow-400 font-medium">限制访问</div>
        <div class="mt-2 w-full bg-yellow-500/20 rounded-full h-1">
          <div class="bg-yellow-500 h-1 rounded-full transition-all duration-500" :style="{ width: `${(modelStats.restricted / modelStats.total) * 100}%` }"></div>
        </div>
      </div>
      <div class="group rounded-xl bg-gradient-to-br from-blue-500/10 to-blue-600/5 p-6 text-center border border-blue-500/20 hover:border-blue-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-blue-500/10">
        <div class="text-3xl font-bold text-blue-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.maintenance }}
        </div>
        <div class="text-sm text-blue-600 dark:text-blue-400 font-medium">维护中</div>
        <div class="mt-2 w-full bg-blue-500/20 rounded-full h-1">
          <div class="bg-blue-500 h-1 rounded-full transition-all duration-500" :style="{ width: `${(modelStats.maintenance / modelStats.total) * 100}%` }"></div>
        </div>
      </div>
      <div class="group rounded-xl bg-gradient-to-br from-gray-500/10 to-gray-600/5 p-6 text-center border border-gray-500/20 hover:border-gray-500/40 transition-all duration-300 hover:shadow-lg hover:shadow-gray-500/10">
        <div class="text-3xl font-bold text-gray-500 group-hover:scale-110 transition-transform duration-300">
          {{ modelStats.total }}
        </div>
        <div class="text-sm text-gray-600 dark:text-gray-400 font-medium">总数</div>
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
              模型列表
            </h3>
            <p class="mt-1 text-[var(--text-secondary)]">
              {{ allModels.length > 0 ? `共找到 ${allModels.length} 个模型` : '暂无模型' }}
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
          title="网格视图"
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
          title="列表视图"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"/>
          </svg>
        </button>
      </div>
    </div>
    
    <!-- 高级筛选和搜索控制面板 -->
    <div class="mb-6 rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
      <!-- 搜索栏 -->
      <div class="mb-4">
        <label class="block text-sm font-medium text-[var(--text-primary)] mb-2">
          模型搜索
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <svg class="w-5 h-5 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
          </div>
          <input
            :value="props.searchKeyword"
            @input="handleSearchInput"
            type="text"
            placeholder="搜索模型名称、创建者、类型、量化方式..."
            class="w-full pl-10 pr-4 py-3 text-sm rounded-lg border border-[var(--border-color)] bg-[var(--content-bg)] text-[var(--text-primary)] placeholder-[var(--text-tertiary)] focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-colors"
          >
          <div v-if="props.searchKeyword" class="absolute inset-y-0 right-0 pr-3 flex items-center">
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
      
      <!-- 筛选器网格 -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 xl:grid-cols-5 gap-4 mb-4">
        <!-- 状态筛选 -->
        <div class="space-y-2">
          <label class="block text-sm font-medium text-[var(--text-primary)]">状态</label>
          <div class="relative">
            <select 
              v-model="statusFilter" 
              class="w-full px-3 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-[var(--content-bg)] text-[var(--text-primary)] focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-colors appearance-none cursor-pointer"
            >
              <option value="">全部状态</option>
              <option value="serving">🟢 服务中</option>
              <option value="restricted">🟡 限制访问</option>
              <option value="maintenance">🔵 维护中</option>
              <option value="offline">⚫ 离线</option>
            </select>
            <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
              </svg>
            </div>
          </div>
        </div>
        
        <!-- 类型筛选 -->
        <div class="space-y-2">
          <label class="block text-sm font-medium text-[var(--text-primary)]">模型类型</label>
          <div class="relative">
            <select 
              v-model="typeFilter" 
              class="w-full px-3 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-[var(--content-bg)] text-[var(--text-primary)] focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-colors appearance-none cursor-pointer"
            >
              <option value="">全部类型</option>
              <option value="OLLAMA">🦙 Ollama</option>
              <option value="HUGGINGFACE">🤗 HuggingFace</option>
              <option value="OPENAI">🤖 OpenAI</option>
              <option value="ANTHROPIC">🧠 Anthropic</option>
              <option value="GOOGLE">🌐 Google</option>
            </select>
            <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
              </svg>
            </div>
          </div>
        </div>
        
        <!-- 参数大小筛选 -->
        <div class="space-y-2">
          <label class="block text-sm font-medium text-[var(--text-primary)]">参数规模</label>
          <div class="relative">
            <select 
              v-model="parameterSizeFilter" 
              class="w-full px-3 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-[var(--content-bg)] text-[var(--text-primary)] focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-colors appearance-none cursor-pointer"
            >
              <option value="">全部规模</option>
              <option value="small">📱 小型 (< 7B)</option>
              <option value="medium">💻 中型 (7B - 20B)</option>
              <option value="large">🖥️ 大型 (20B - 70B)</option>
              <option value="xlarge">🏢 超大型 (> 70B)</option>
            </select>
            <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
              </svg>
            </div>
          </div>
        </div>
        
        <!-- 排序字段 -->
        <div class="space-y-2">
          <label class="block text-sm font-medium text-[var(--text-primary)]">排序依据</label>
          <div class="relative">
            <select 
              v-model="sortBy" 
              class="w-full px-3 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-[var(--content-bg)] text-[var(--text-primary)] focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-colors appearance-none cursor-pointer"
            >
              <option value="name">📝 名称</option>

              <option value="createDate">� 创建时间</option>
              <option value="parameterSize">� 参数大小</option>
              <option value="clientCount">� 贡献人数</option>
            </select>
            <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <svg class="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
              </svg>
            </div>
          </div>
        </div>
        
        <!-- 排序方向和操作 -->
        <div class="space-y-2">
          <label class="block text-sm font-medium text-[var(--text-primary)]">操作</label>
          <div class="flex items-center space-x-2">
            <button
              class="flex-1 px-3 py-2 rounded-lg border border-[var(--border-color)] hover:bg-[var(--hover-bg)] transition-colors text-sm font-medium text-[var(--text-primary)]"
              @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'"
              :title="sortOrder === 'asc' ? '点击切换为降序' : '点击切换为升序'"
            >
              <div class="flex items-center justify-center space-x-1">
                <svg class="w-4 h-4 transition-transform" :class="{ 'rotate-180': sortOrder === 'desc' }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12"/>
                </svg>
                <span>{{ sortOrder === 'asc' ? '升序' : '降序' }}</span>
              </div>
            </button>
            <button
              class="px-3 py-2 rounded-lg bg-gray-500/10 hover:bg-gray-500/20 transition-colors text-sm font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
              @click="resetFilters"
              title="重置所有筛选条件"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
      
      <!-- 活动筛选器显示 -->
      <div v-if="hasActiveFilters" class="flex flex-wrap items-center gap-2">
        <span class="text-sm text-[var(--text-secondary)]">活动筛选器:</span>
        <span v-if="statusFilter" class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-green-500/10 text-green-500 border border-green-500/20">
          状态: {{ getStatusText(statusFilter as any) }}
          <button @click="statusFilter = ''" class="ml-1 hover:text-green-700">
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </span>
        <span v-if="typeFilter" class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
          类型: {{ typeFilter }}
          <button @click="typeFilter = ''" class="ml-1 hover:text-blue-700">
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </span>
        <span v-if="parameterSizeFilter" class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-purple-500/10 text-purple-500 border border-purple-500/20">
          规模: {{ getParameterSizeText(parameterSizeFilter) }}
          <button @click="parameterSizeFilter = ''" class="ml-1 hover:text-purple-700">
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </span>
        <button @click="resetFilters" class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-500/10 text-gray-500 hover:bg-gray-500/20 transition-colors">
          清除全部
          <svg class="w-3 h-3 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
          </svg>
        </button>
      </div>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="flex items-center space-x-3 text-[var(--text-secondary)]">
        <div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
        <span class="font-medium">加载中...</span>
      </div>
    </div>
    
    <!-- 模型列表 -->
    <div v-else>
      <!-- 网格视图 -->
      <div v-if="viewMode === 'grid'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div
          v-for="model in displayedModels"
          :key="model.id"
          class="group relative overflow-hidden cursor-pointer rounded-2xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6 transition-all duration-300 hover:shadow-xl hover:scale-[1.02] hover:border-blue-500/50"
          @click="handleModelClick(model)"
        >
          <!-- 悬浮效果背景 -->
          <div class="absolute inset-0 bg-gradient-to-r from-blue-500/5 to-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          
          <div class="relative z-10">
            <!-- 模型图标和状态 -->
            <div class="flex items-start justify-between mb-4">
              <div
                :style="{ background: `linear-gradient(135deg, ${model.color}, ${model.color}dd)` }"
                class="flex h-12 w-12 items-center justify-center rounded-xl text-white shadow-lg group-hover:scale-110 transition-transform duration-300"
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
              <span
                :class="getStatusClass(model.status)"
                class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium border"
              >
                {{ getStatusText(model.status) }}
              </span>
            </div>
            
            <!-- 模型ID - 紧凑显示并支持复制 -->
            <div class="mb-3 p-2 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border border-blue-200 dark:border-blue-700 rounded-lg">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 min-w-0 flex-1">
                  <span class="text-xs font-medium text-blue-600 dark:text-blue-400 uppercase tracking-wide flex-shrink-0">ID:</span>
                  <code class="text-xs font-mono font-bold text-gray-800 dark:text-gray-200 truncate cursor-pointer hover:text-blue-600 transition-colors" 
                        @click.stop="copyToClipboard(model.id)"
                        :title="'点击复制: ' + model.id">
                    {{ model.id }}
                  </code>
                </div>
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
            
            <!-- 模型标签 -->
            <div class="flex flex-wrap gap-2 mb-3">
              <!-- Embedding模型特殊标识 -->
              <span v-if="isEmbeddingModel(model)" class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-amber-500/10 text-amber-600 border border-amber-500/20">
                <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v6a2 2 0 002 2h6a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4 0V3a2 2 0 012-2h2a2 2 0 012 2v2m-4 0v2m0 0v4"/>
                </svg>
                Embedding模型
              </span>
              <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
                {{ model.parameterSize }}
              </span>
              <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-purple-500/10 text-purple-500 border border-purple-500/20">
                {{ model.modelType }}
              </span>
              <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-green-500/10 text-green-500 border border-green-500/20">
                {{ model.quantization }}
              </span>
              <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-orange-500/10 text-orange-500 border border-orange-500/20">
                {{ getModelSeries(model.id) }}
              </span>
              <!-- 定价标签 - 显示最低价格和提供商 -->
              <div class="space-y-2">
                <!-- 输入Token最低价格 -->
                <div class="inline-flex items-center px-3 py-1.5 rounded-md text-xs font-medium bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">
                  <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M13 10l3 4M9 12l4-4"/>
                  </svg>
                  <span>输入￥{{ model.inputPPM?.toFixed(1) || '10.0' }}/百万</span>
                  <span v-if="model.inputClientInfo" class="ml-2 text-xs opacity-75">
                    by {{ model.inputClientInfo.username }}
                  </span>
                </div>
                
                <!-- 输出Token最低价格 -->
                <div class="inline-flex items-center px-3 py-1.5 rounded-md text-xs font-medium bg-blue-500/10 text-blue-600 border border-blue-500/20">
                  <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8a3 3 0 016 0m-6 0v10c0 3.314 2.686 6 6 6s6-2.686 6-6V8m-6 0V5a2 2 0 012-2h2a2 2 0 012 2v3"/>
                  </svg>
                  <span>输出￥{{ model.outputPPM?.toFixed(1) || '20.0' }}/百万</span>
                  <span v-if="model.outputClientInfo" class="ml-2 text-xs opacity-75">
                    by {{ model.outputClientInfo.username }}
                  </span>
                </div>
              </div>
            </div>
            
            <!-- 模型规格信息 -->
            <div class="space-y-1 mb-4 text-xs text-[var(--text-secondary)]">
              <div class="flex justify-between">
                <span>模型名称:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.name }}</span>
              </div>
              <div class="flex justify-between">
                <span>创建者:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.creator || '未知' }}</span>
              </div>
              <div class="flex justify-between items-center">
                <span>可用客户端:</span>
                <span class="font-bold text-blue-600 bg-blue-50 dark:bg-blue-900/20 px-2 py-1 rounded">{{ model.clientCount || 0 }}个</span>
              </div>
            </div>

            <!-- 快速操作 -->
            <div class="mt-4 flex items-center justify-between">
              <div class="flex items-center space-x-2">
                <button
                  class="p-2 rounded-lg bg-blue-500/10 text-blue-500 hover:bg-blue-500/20 transition-colors duration-200"
                  title="收藏"
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
                  :title="isEmbeddingModel(model) ? 'Embedding模型不支持对话功能' : '开始对话'"
                  class="inline-flex items-center px-3 py-1.5 text-xs font-semibold rounded-lg transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5 hover:scale-105 border relative overflow-hidden group disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
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
                  <span class="relative z-10">{{ isEmbeddingModel(model) ? '不支持对话' : '对话' }}</span>
                </button>
                <button
                  class="opacity-0 group-hover:opacity-100 inline-flex items-center px-3 py-1 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white text-xs font-medium rounded-lg transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5"
                  @click.stop="handleViewDetails(model)"
                >
                  查看详情
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
          class="group relative overflow-hidden cursor-pointer rounded-2xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6 transition-all duration-300 hover:shadow-xl hover:scale-[1.01] hover:border-blue-500/50"
          @click="handleModelClick(model)"
        >
          <!-- 悬浮效果背景 -->
          <div class="absolute inset-0 bg-gradient-to-r from-blue-500/5 to-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          
          <div class="relative z-10 flex items-start space-x-6">
            <!-- 模型图标 -->
            <div class="flex-shrink-0">
              <div
                :style="{ background: `linear-gradient(135deg, ${model.color}, ${model.color}dd)` }"
                class="flex h-16 w-16 items-center justify-center rounded-2xl text-white shadow-lg group-hover:scale-110 transition-transform duration-300"
              >
                <svg class="h-8 w-8" fill="currentColor" viewBox="0 0 24 24">
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
              <div class="flex items-start justify-between">
                <div class="flex-1">
                  <!-- 模型ID显示区域 -->
                  <div class="flex items-center gap-3 mb-3">
                    <div class="inline-flex items-center gap-2 p-2 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border border-blue-200 dark:border-blue-700 rounded-lg max-w-md">
                      <div class="flex items-center gap-2">
                        <span class="text-xs font-medium text-blue-600 dark:text-blue-400 uppercase tracking-wide">ID:</span>
                        <code class="text-sm font-mono font-bold text-gray-800 dark:text-gray-200" :title="model.id">
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
                  
                  <div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-gray-100 dark:bg-gray-800 text-[var(--text-primary)] border border-gray-200 dark:border-gray-700">
                      <span class="text-xs text-gray-500 mr-1">名称:</span>{{ model.name }}
                    </span>
                    <!-- Embedding模型特殊标识 -->
                    <span v-if="isEmbeddingModel(model)" class="inline-flex items-center px-3 py-1 rounded-full bg-amber-500/10 text-amber-600 border border-amber-500/20">
                      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v6a2 2 0 002 2h6a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4 0V3a2 2 0 012-2h2a2 2 0 012 2v2m-4 0v2m0 0v4"/>
                      </svg>
                      Embedding模型
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-blue-500/10 text-blue-500 border border-blue-500/20">
                      {{ model.parameterSize }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-purple-500/10 text-purple-500 border border-purple-500/20">
                      {{ model.modelType }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-green-500/10 text-green-500 border border-green-500/20">
                      {{ model.quantization }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-orange-500/10 text-orange-500 border border-orange-500/20">
                      {{ getModelSeries(model.id) }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-700">
                      <span class="text-xs mr-1">创建者:</span>{{ model.creator || '未知' }}
                    </span>
                    <!-- 客户端数量标签 -->
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-blue-500/10 text-blue-500 border border-blue-500/20">
                      <span class="text-xs mr-1">客户端:</span>{{ model.clientCount || 0 }}个
                    </span>
                    <!-- 定价标签 - 列表视图显示最低价格 -->
                    <div class="inline-flex items-center space-x-2">
                      <!-- 输入Token最低价格 -->
                      <span class="inline-flex items-center px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">
                        <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M13 10l3 4M9 12l4-4"/>
                        </svg>
                        <span class="text-xs">输入￥{{ model.inputPPM?.toFixed(1) || '10.0' }}/百万</span>
                        <span v-if="model.inputClientInfo" class="ml-1 text-xs opacity-75">
                          ({{ model.inputClientInfo.username }})
                        </span>
                      </span>
                      
                      <!-- 输出Token最低价格 -->
                      <span class="inline-flex items-center px-3 py-1 rounded-full bg-blue-500/10 text-blue-600 border border-blue-500/20">
                        <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8a3 3 0 016 0m-6 0v10c0 3.314 2.686 6 6 6s6-2.686 6-6V8m-6 0V5a2 2 0 012-2h2a2 2 0 012 2v3"/>
                        </svg>
                        <span class="text-xs">输出￥{{ model.outputPPM?.toFixed(1) || '20.0' }}/百万</span>
                        <span v-if="model.outputClientInfo" class="ml-1 text-xs opacity-75">
                          ({{ model.outputClientInfo.username }})
                        </span>
                      </span>
                    </div>
                  </div>
                </div>
                
                <!-- 状态 -->
                <div class="flex flex-col items-end space-y-2 ml-4">
                  <span
                    :class="getStatusClass(model.status)"
                    class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border"
                  >
                    {{ getStatusText(model.status) }}
                  </span>
                </div>
              </div>
              
              <!-- TODO: 模型详细信息 - 暂时注释，后续需要时再启用 -->

              <!-- 快速操作按钮 -->
              <div class="mt-4 flex items-center justify-end space-x-3">
                <button
                  :disabled="isEmbeddingModel(model)"
                  :title="isEmbeddingModel(model) ? 'Embedding模型不支持对话功能' : '开始对话'"
                  class="inline-flex items-center px-5 py-2.5 text-sm font-semibold rounded-xl transition-all duration-200 shadow-lg hover:shadow-xl transform hover:-translate-y-1 hover:scale-105 border relative overflow-hidden group disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
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
                  <span class="relative z-10">{{ isEmbeddingModel(model) ? '不支持对话' : '立即对话' }}</span>
                </button>
                <button
                  class="inline-flex items-center px-4 py-2 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white text-sm font-medium rounded-lg transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5"
                  @click.stop="handleViewDetails(model)"
                >
                  查看详情
                  <svg class="ml-1 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                </button>
              </div>
            </div>
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
            <span v-if="!loading">加载更多</span>
            <span v-else>加载中...</span>
          </button>
          <div class="mt-3 text-sm text-[var(--text-secondary)]">
            已显示 {{ displayedModels.length }} 条，共 {{ totalModels }} 条记录
          </div>
        </div>
        
        <!-- 没有更多数据提示 -->
        <div v-if="!hasMore && displayedModels.length > 0" class="text-center">
          <div class="inline-flex items-center px-6 py-3 bg-[var(--content-bg)] border border-[var(--border-color)] rounded-xl text-[var(--text-secondary)]">
            <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
            已加载全部模型
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
            暂无模型数据
          </h3>
          <p class="text-[var(--text-secondary)]">
            {{ props.searchKeyword ? '没有找到匹配的模型' : '暂时没有可用的模型' }}
          </p>
        </div>
        
        <!-- 无搜索结果 -->
        <div v-if="filteredModels.length === 0 && searchKeyword" class="text-center py-16">
          <div class="w-20 h-20 bg-[var(--hover-bg)] rounded-full flex items-center justify-center mx-auto mb-4">
            <svg class="w-10 h-10 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6-4h6m2 5.291A7.962 7.962 0 0118 12a8 8 0 10-2.343 5.657l2.343 2.343"/>
            </svg>
          </div>
          <h3 class="text-lg font-medium text-[var(--text-primary)] mb-2">
            没有找到相关模型
          </h3>
          <p class="text-[var(--text-secondary)]">
            没有找到匹配"{{ searchKeyword }}"的模型，请尝试其他关键词
          </p>
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
  creator: string;
  status: 'serving' | 'restricted' | 'offline' | 'maintenance';
  description: string;
  icon: string;
  color: string;
  createDate: string;
  size: string;
  quantization: string; // 量化方式
  type: string;
  clientCount?: number; // 新增可用客户端数量
  inputPPM?: number; // 输入定价 (每百万Token)
  outputPPM?: number; // 输出定价 (每百万Token)
  inputClientInfo?: { // 最低输入价格的客户端信息
    clientId: string;
    username: string;
    price: number;
  } | null;
  outputClientInfo?: { // 最低输出价格的客户端信息
    clientId: string;
    username: string;
    price: number;
  } | null;
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
  search: [keyword: string];
}>();

// 路由实例
const router = useRouter();

// 响应式状态
const loading = ref(false);
const currentPage = ref(1);
const pageSize = 12; // 每页显示数量
const viewMode = ref<'grid' | 'list'>('grid'); // 默认网格视图
const statusFilter = ref('');
const typeFilter = ref('');
const parameterSizeFilter = ref(''); // 新增参数大小筛选
const sortBy = ref('name');
const sortOrder = ref<'asc' | 'desc'>('asc');

// 模型数据
const allModels = ref<ModelItem[]>([]); // 已加载的所有模型数据
const totalModels = ref(0); // 服务器端总数量

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
    
    // 计算所有客户端中的最低价格
    const getLowestPricing = () => {
      if (!apiModel.client_models || apiModel.client_models.length === 0) {
        return {
          minInputPPM: 10,
          minOutputPPM: 20,
          inputClientInfo: null,
          outputClientInfo: null
        };
      }

      let minInputPPM = Number.MAX_VALUE;
      let minOutputPPM = Number.MAX_VALUE;
      let inputClientInfo = null;
      let outputClientInfo = null;

      apiModel.client_models.forEach(clientModel => {
        const model = clientModel.model;
        const client = clientModel.client;
        
        if (model?.ippm !== undefined && model.ippm < minInputPPM) {
          minInputPPM = model.ippm;
          inputClientInfo = {
            clientId: client?.id?.substring(0, 8) + '...',
            username: client?.user?.username || '匿名用户',
            price: model.ippm
          };
        }
        
        if (model?.oppm !== undefined && model.oppm < minOutputPPM) {
          minOutputPPM = model.oppm;
          outputClientInfo = {
            clientId: client?.id?.substring(0, 8) + '...',
            username: client?.user?.username || '匿名用户',
            price: model.oppm
          };
        }
      });

      return {
        minInputPPM: minInputPPM === Number.MAX_VALUE ? 10 : minInputPPM,
        minOutputPPM: minOutputPPM === Number.MAX_VALUE ? 20 : minOutputPPM,
        inputClientInfo,
        outputClientInfo
      };
    };

    const pricingInfo = getLowestPricing();
    
    // 获取第一个客户端模型的其他信息作为默认值
    const firstClientModel = apiModel.client_models?.[0];
    const modelData = firstClientModel?.model;
    const [name, version] = modelName.split(':');
    
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

    // 获取第一个客户端模型的信息作为默认值
    const clientData = firstClientModel?.client;
    
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
    
    return {
      id: modelName,
      name: name || modelName,
      parameterSize: version || 'Unknown',
      modelType: (apiModel.type || 'unknown').toUpperCase(),
      creator: clientData?.user?.username || modelData?.openai_model?.owned_by || 'Unknown',
      status: getModelStatus(),
      description: `${apiModel.type || 'unknown'} 模型，量化：${modelData?.arch || 'N/A'}，大小：${formatSize(apiModel.size || '0')}，可用客户端：${apiModel.client_models?.length || 0}个`,
      icon,
      color,
      createDate: modelData?.openai_model?.created ? new Date(modelData.openai_model.created * 1000).toLocaleDateString() : new Date().toLocaleDateString(),
      size: formatSize(apiModel.size || '0'),
      quantization: modelData?.arch || 'N/A', // 使用量化方式
      type: apiModel.type || 'unknown',
      clientCount: apiModel.client_models?.length || 0,
      inputPPM: pricingInfo.minInputPPM,
      outputPPM: pricingInfo.minOutputPPM,
      inputClientInfo: pricingInfo.inputClientInfo,
      outputClientInfo: pricingInfo.outputClientInfo
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
    creator: 'Unknown',
    status: 'offline',
    description: '数据异常的模型',
    icon: 'lucide:alert-triangle',
    color: '#ff4d4f',
    createDate: new Date().toLocaleDateString(),
    size: '0B',
    quantization: 'N/A',
    type: 'unknown',
    clientCount: 0,
    inputPPM: 10,
    outputPPM: 20,
    inputClientInfo: null,
    outputClientInfo: null
  };
};



// API获取模型数据 - 真正的分页版本
const fetchModels = async (page: number = 1, limit: number = pageSize) => {
  try {
    loading.value = true;
    
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
    const transformedModels = apiModels.map(transformApiModel);
    
    // 应用搜索和筛选
    let filteredModels = transformedModels;
    if (props.searchKeyword.trim()) {
      const keyword = props.searchKeyword.toLowerCase();
      filteredModels = transformedModels.filter(model => 
        model.name.toLowerCase().includes(keyword) ||
        model.creator.toLowerCase().includes(keyword) ||
        model.modelType.toLowerCase().includes(keyword) ||
        model.quantization.toLowerCase().includes(keyword) ||
        model.description.toLowerCase().includes(keyword)
      );
    }
    
    // 状态筛选
    if (statusFilter.value) {
      filteredModels = filteredModels.filter(model => model.status === statusFilter.value);
    }
    
    // 类型筛选
    if (typeFilter.value) {
      filteredModels = filteredModels.filter(model => model.modelType === typeFilter.value);
    }
    
    // 参数大小筛选
    if (parameterSizeFilter.value) {
      filteredModels = filteredModels.filter(model => {
        const category = getParameterSizeCategory(model.parameterSize);
        return category === parameterSizeFilter.value;
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
    loading.value = false;
  }
};

// 初始化加载模型数据
const initializeModels = async () => {
  console.log('初始化模型数据');
  currentPage.value = 1;
  allModels.value = [];
  
  const result = await fetchModels(1);
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
  if (props.searchKeyword.trim()) {
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
    serving: '服务中',
    restricted: '限制访问',
    maintenance: '维护中',
    offline: '离线',
  };
  return texts[status];
};

// 获取参数大小文本
const getParameterSizeText = (size: string): string => {
  const sizeMap: { [key: string]: string } = {
    small: '小型',
    medium: '中型', 
    large: '大型',
    xlarge: '超大型'
  };
  return sizeMap[size] || size;
};

// 检查是否有活动的筛选器
const hasActiveFilters = computed(() => {
  return !!(statusFilter.value || typeFilter.value || parameterSizeFilter.value || props.searchKeyword);
});

// 根据参数大小分类
const getParameterSizeCategory = (parameterSize: string): string => {
  const size = parameterSize.toLowerCase();
  const numMatch = size.match(/(\d+(\.\d+)?)/);
  if (!numMatch || !numMatch[1]) return 'small';
  
  const num = parseFloat(numMatch[1]);
  if (size.includes('b')) {
    if (num < 7) return 'small';
    if (num <= 20) return 'medium';
    if (num <= 70) return 'large';
    return 'xlarge';
  }
  return 'small';
};

// 处理搜索输入
const handleSearchInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit('search', target.value);
};

// 清除搜索
const clearSearch = () => {
  emit('search', '');
};

// 重置所有筛选器
const resetFilters = () => {
  statusFilter.value = '';
  typeFilter.value = '';
  parameterSizeFilter.value = '';
  sortBy.value = 'name';
  sortOrder.value = 'asc';
  emit('search', '');
};

// 生成模型系列名称
const getModelSeries = (modelName: string): string => {
  const name = modelName.toLowerCase();
  if (name.includes('qwen3')) {
    return 'Qwen3系列';
  } else if (name.includes('qwen2.5')) {
    return 'Qwen2.5系列';
  } else if (name.includes('qwen2')) {
    return 'Qwen2系列';
  } else if (name.includes('qwen')) {
    return 'Qwen系列';
  } else if (name.includes('deepseek-r1')) {
    return 'DeepSeek-R1系列';
  } else if (name.includes('deepseek-v3')) {
    return 'DeepSeek-V3系列';
  } else if (name.includes('deepseek-v2')) {
    return 'DeepSeek-V2系列';
  } else if (name.includes('deepseek')) {
    return 'DeepSeek系列';
  } else if (name.includes('llama3.3')) {
    return 'Llama 3.3系列';
  } else if (name.includes('llama3.2')) {
    return 'Llama 3.2系列';
  } else if (name.includes('llama3.1')) {
    return 'Llama 3.1系列';
  } else if (name.includes('llama3')) {
    return 'Llama 3系列';
  } else if (name.includes('llama2')) {
    return 'Llama 2系列';
  } else if (name.includes('llama')) {
    return 'Llama系列';
  } else if (name.includes('chatglm4')) {
    return 'ChatGLM4系列';
  } else if (name.includes('chatglm3')) {
    return 'ChatGLM3系列';
  } else if (name.includes('chatglm')) {
    return 'ChatGLM系列';
  } else if (name.includes('gemma2')) {
    return 'Gemma 2系列';
  } else if (name.includes('gemma')) {
    return 'Gemma系列';
  } else if (name.includes('mistral')) {
    return 'Mistral系列';
  } else if (name.includes('phi')) {
    return 'Phi系列';
  } else {
    // 提取模型基础名称作为系列
    const baseName = modelName.split(':')[0];
    return `${baseName}系列`;
  }
};

// 检查是否为embedding模型
const isEmbeddingModel = (model: ModelItem): boolean => {
  if (!model) return false;
  
  // 检查模型类型
  const type = model.type?.toLowerCase();
  if (type === 'embedding' || type === 'embeddings') {
    return true;
  }
  
  // 检查模型名称中的关键词
  const name = model.name?.toLowerCase() || '';
  const id = model.id?.toLowerCase() || '';
  
  // 常见的embedding模型名称关键词
  const embeddingKeywords = [
    'embedding',
    'embed',
    'bge-',
    'text-embedding',
    'sentence-transformer',
    'all-minilm',
    'e5-',
    'gte-',
    'multilingual-e5',
    'text2vec'
  ];
  
  return embeddingKeywords.some(keyword => 
    name.includes(keyword) || id.includes(keyword)
  );
};

// 复制到剪贴板
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    console.log('已复制模型ID:', text);
    // 这里可以添加一个成功提示
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
    group: model.creator,
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

// 监听搜索关键词变化，重置分页
watch(() => props.searchKeyword, () => {
  initializeModels();
});



// 组件挂载时初始化数据
onMounted(() => {
  console.log('ModelMarketplace 组件挂载');
  initializeModels();
});

// 监听搜索关键词变化
watch(() => props.searchKeyword, () => {
  initializeModels();
});

// 暴露刷新方法给父组件
const refreshData = () => {
  console.log('ModelMarketplace 收到刷新指令');
  initializeModels();
};

// 使用 defineExpose 暴露方法
defineExpose({
  refreshData,
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
    space-x: 0;
  }
  
  .flex.items-center.space-x-8 > * + * {
    margin-left: 0;
    margin-top: 0.5rem;
  }
}
</style>
