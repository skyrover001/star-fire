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
        <h3 class="text-2xl font-bold text-[var(--text-primary)]">
          模型列表
        </h3>
        <p class="mt-1 text-[var(--text-secondary)]">
          {{ filteredModels.length > 0 ? `共找到 ${filteredModels.length} 个模型` : '暂无模型' }}
        </p>
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
            
            <!-- 模型名称和类型 -->
            <h4 class="text-lg font-bold text-[var(--text-primary)] group-hover:text-blue-500 transition-colors duration-200 mb-2">
              {{ model.name }}
            </h4>
            
            <!-- 模型标签 -->
            <div class="flex flex-wrap gap-2 mb-3">
              <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
                {{ model.parameterSize }}
              </span>
              <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-purple-500/10 text-purple-500 border border-purple-500/20">
                {{ model.modelType }}
              </span>
            </div>
            
            <!-- 模型规格信息 -->
            <div class="space-y-1 mb-4 text-xs text-[var(--text-secondary)]">
              <div class="flex justify-between">
                <span>参数大小:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.parameterSize }}</span>
              </div>
              <div class="flex justify-between">
                <span>模型量化:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.quantization }}</span>
              </div>
              <div class="flex justify-between">
                <span>推理引擎:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.modelType }}</span>
              </div>
              <div class="flex justify-between">
                <span>模型贡献人数:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.clientCount }}人</span>
              </div>
              <div class="flex justify-between">
                <span>可用客户端:</span>
                <span class="font-medium text-[var(--text-primary)]">{{ model.clientCount }}个</span>
              </div>
            </div>
            
            <!-- 描述 -->
            <p class="text-sm text-[var(--text-secondary)] leading-relaxed line-clamp-2 mb-4">
              {{ model.description }}
            </p>
            
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
                <button
                  class="p-2 rounded-lg bg-green-500/10 text-green-500 hover:bg-green-500/20 transition-colors duration-200"
                  title="下载"
                  @click.stop="downloadModel(model)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                  </svg>
                </button>
              </div>
              
              <!-- 底部信息 -->
              <div class="flex items-center justify-between pt-4 border-t border-[var(--border-color)]">
                <div class="flex items-center space-x-3 text-sm">
                  <div v-if="model.clientCount" class="flex items-center text-blue-500">
                    <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857M13 7a4 4 0 11-8 0 4 4 0 018 0z"/>
                    </svg>
                    {{ model.clientCount }}人贡献
                  </div>
                </div>
                <button
                  class="opacity-0 group-hover:opacity-100 inline-flex items-center px-3 py-1 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white text-xs font-medium rounded-lg transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5"
                  @click.stop="handleViewDetails(model)"
                >
                  查看
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
                  <h4 class="text-xl font-bold text-[var(--text-primary)] group-hover:text-blue-500 transition-colors duration-200">
                    {{ model.name }}
                  </h4>
                  <div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-[var(--hover-bg)] text-[var(--text-primary)] border border-[var(--border-color)]">
                      {{ model.modelType }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-blue-500/10 text-blue-500 border border-blue-500/20">
                      {{ model.parameterSize }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-purple-500/10 text-purple-500 border border-purple-500/20">
                      {{ model.size }}
                    </span>
                    <span class="inline-flex items-center px-3 py-1 rounded-full bg-green-500/10 text-green-500 border border-green-500/20">
                      {{ model.quantization }}
                    </span>
                    <span class="text-[var(--text-secondary)]">{{ model.creator || model.type }}</span>
                  </div>
                </div>
                
                <!-- 状态和评分 -->
                <div class="flex flex-col items-end space-y-3 ml-4">
                  <span
                    :class="getStatusClass(model.status)"
                    class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border"
                  >
                    {{ getStatusText(model.status) }}
                  </span>
                  <div v-if="model.clientCount" class="flex items-center text-blue-500">
                    <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857M13 7a4 4 0 11-8 0 4 4 0 018 0z"/>
                    </svg>
                    <span class="ml-1 text-sm font-medium">{{ model.clientCount }}人贡献</span>
                  </div>
                </div>
              </div>
              
              <p class="mt-4 text-[var(--text-secondary)] leading-relaxed">
                {{ model.description }}
              </p>
              
              <!-- 性能和操作区域 -->
              <div class="mt-5 flex items-center justify-between">
                <div class="flex items-center space-x-8">
                  <!-- 左侧信息 -->
                  <div class="flex items-center space-x-6 text-sm text-[var(--text-secondary)]">
                    <span class="flex items-center">
                      <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                      </svg>
                      {{ model.createDate }}
                    </span>
                  </div>
                </div>
                
                <div class="flex items-center space-x-3">
                  <!-- 快速操作按钮 -->
                  <button
                    class="p-2 rounded-lg bg-blue-500/10 text-blue-500 hover:bg-blue-500/20 transition-colors duration-200"
                    title="收藏"
                    @click.stop="toggleFavorite(model)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"/>
                    </svg>
                  </button>
                  <button
                    class="p-2 rounded-lg bg-green-500/10 text-green-500 hover:bg-green-500/20 transition-colors duration-200"
                    title="下载"
                    @click.stop="downloadModel(model)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                    </svg>
                  </button>
                  <button
                    class="inline-flex items-center px-4 py-2 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white text-sm font-medium rounded-xl transition-all duration-200 shadow-lg hover:shadow-xl transform hover:-translate-y-0.5 opacity-0 group-hover:opacity-100"
                    @click.stop="handleViewDetails(model)"
                  >
                    查看详情
                    <svg class="ml-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 懒加载触发器 -->
      <div ref="loadTrigger" class="py-8">
        <div v-if="hasMore && !loading" class="text-center">
          <div class="inline-flex items-center text-gray-400">
            <svg class="mr-2 h-4 w-4 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3"/>
            </svg>
            <span class="text-sm">向下滚动加载更多</span>
          </div>
        </div>
        
        <!-- 没有更多数据提示 -->
        <div v-if="!hasMore && displayedModels.length > 0" class="text-center">
          <div class="inline-flex items-center px-4 py-2 bg-[var(--content-bg)] border border-[var(--border-color)] rounded-xl text-[var(--text-secondary)]">
            <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
            已加载全部模型
          </div>
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

import { computed, ref, watch, onMounted, onUnmounted, onActivated } from 'vue';
import { useRouter } from 'vue-router';
// 导入请求工具
import { requestClient } from '#/api/request';

// 定义API返回的原始模型数据类型
interface ClientModel {
  name: string;
  type: string;
  size: string;
  quantization?: string;
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
  user: User;
}

interface ClientModelPair {
  client: Client;
  model: ClientModel;
}

interface ApiModelItem {
  name: string;
  type: string;
  size: string;
  quantization: string;
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
  quantization: string; // 替代 arch
  type: string;
  clientCount?: number; // 新增可用客户端数量
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
const pageSize = 12; // 网格布局适合的数量
const viewMode = ref<'grid' | 'list'>('grid'); // 默认网格视图
const statusFilter = ref('');
const typeFilter = ref('');
const parameterSizeFilter = ref(''); // 新增参数大小筛选
const sortBy = ref('name');
const sortOrder = ref<'asc' | 'desc'>('asc');

// DOM引用
const loadTrigger = ref<HTMLElement>();

// 所有模型数据
const allModels = ref<ModelItem[]>([]);
const totalModels = ref(0);

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
    const firstClientModel = apiModel.client_models?.[0];
    const modelData = firstClientModel?.model;
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
      description: `${apiModel.type || 'unknown'} 模型，量化：${apiModel.quantization || 'N/A'}，大小：${formatSize(apiModel.size || '0')}，可用客户端：${apiModel.client_models?.length || 0}个`,
      icon,
      color,
      createDate: modelData?.openai_model?.created ? new Date(modelData.openai_model.created * 1000).toLocaleDateString() : new Date().toLocaleDateString(),
      size: formatSize(apiModel.size || '0'),
      quantization: apiModel.quantization || 'N/A', // 使用量化方式
      type: apiModel.type || 'unknown',
      clientCount: apiModel.client_models?.length || 0
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
    clientCount: 0
  };
};

// API获取模型数据
const fetchModels = async (page: number = 1, limit: number = pageSize) => {
  try {
    loading.value = true;
    const response = await requestClient.get('/market/models');
    
    console.log('Models API 响应:', response);
    
    // 首先检查响应是否存在
    if (!response) {
      console.warn('API 返回空响应');
      return {
        models: [],
        total: 0,
        hasMore: false
      };
    }
    
    // 检查响应是否是数组格式
    if (Array.isArray(response)) {
      // 直接处理数组响应
      const apiModels: ApiModelItem[] = response;
      
      // 过滤搜索关键词
      let filteredModels = apiModels;
      if (props.searchKeyword.trim()) {
        const keyword = props.searchKeyword.toLowerCase();
        filteredModels = apiModels.filter(model => 
          model?.name?.toLowerCase().includes(keyword) ||
          model?.type?.toLowerCase().includes(keyword) ||
          model?.quantization?.toLowerCase().includes(keyword)
        );
      }
      
      // 分页处理
      const startIndex = (page - 1) * limit;
      const endIndex = startIndex + limit;
      const paginatedModels = filteredModels.slice(startIndex, endIndex);
      
      // 转换数据格式
      const transformedModels = paginatedModels.map(transformApiModel);
      console.log('转换后的模型数据:', transformedModels);
      
      return {
        models: transformedModels,
        total: filteredModels.length,
        hasMore: endIndex < filteredModels.length
      };
    } else if (response && response.success && response.data) {
      // 处理包装的响应格式
      const apiModels: ApiModelItem[] = response.data.models || response.data || [];
      const transformedModels = apiModels.map(transformApiModel);
      
      return {
        models: transformedModels,
        total: response.data.total || apiModels.length,
        hasMore: response.data.hasMore || false
      };
    } else {
      // 处理其他响应格式或错误情况
      const errorMessage = response?.message || response?.error || '未知错误';
      console.error('获取模型数据失败:', errorMessage, response);
      
      // 如果有其他可能的数据格式，可以在这里尝试处理
      if (response && response.data && Array.isArray(response.data)) {
        console.log('尝试处理备用数据格式...');
        const apiModels: ApiModelItem[] = response.data;
        const transformedModels = apiModels.map(transformApiModel);
        
        return {
          models: transformedModels,
          total: apiModels.length,
          hasMore: false
        };
      }
      
      return {
        models: [],
        total: 0,
        hasMore: false
      };
    }
  } catch (error) {
    console.error('获取模型数据失败:', error);
    
    // 检查是否是网络错误
    if (error instanceof TypeError && error.message.includes('fetch')) {
      console.error('网络连接错误，可能是API服务未启动');
    }
    
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
  const result = await fetchModels(1);
  allModels.value = result.models;
  totalModels.value = result.total;
  console.log('模型数据加载完成:', result.models.length, '个模型');
};

// 根据搜索关键词过滤模型
const filteredModels = computed(() => {
  let result = allModels.value;
  
  // 搜索关键词过滤
  if (props.searchKeyword.trim()) {
    const keyword = props.searchKeyword.toLowerCase();
    result = result.filter(model => 
      model.name.toLowerCase().includes(keyword) ||
      model.creator.toLowerCase().includes(keyword) ||
      model.modelType.toLowerCase().includes(keyword) ||
      model.quantization.toLowerCase().includes(keyword) ||
      model.description.toLowerCase().includes(keyword)
    );
  }
  
  // 状态过滤
  if (statusFilter.value) {
    result = result.filter(model => model.status === statusFilter.value);
  }
  
  // 类型过滤
  if (typeFilter.value) {
    result = result.filter(model => model.modelType === typeFilter.value);
  }
  
  // 参数大小过滤
  if (parameterSizeFilter.value) {
    result = result.filter(model => {
      const category = getParameterSizeCategory(model.parameterSize);
      return category === parameterSizeFilter.value;
    });
  }
  
  // 排序
  const sortOrderMultiplier = sortOrder.value === 'asc' ? 1 : -1;
  result.sort((a, b) => {
    switch (sortBy.value) {
      case 'createDate':
        return (new Date(a.createDate).getTime() - new Date(b.createDate).getTime()) * sortOrderMultiplier;
      case 'parameterSize':
        // 按参数数值大小排序
        const aNum = parseFloat(a.parameterSize.match(/(\d+(\.\d+)?)/)?.[1] || '0');
        const bNum = parseFloat(b.parameterSize.match(/(\d+(\.\d+)?)/)?.[1] || '0');
        return (aNum - bNum) * sortOrderMultiplier;
      case 'clientCount':
        return ((a.clientCount || 0) - (b.clientCount || 0)) * sortOrderMultiplier;
      default: // name
        return a.name.localeCompare(b.name) * sortOrderMultiplier;
    }
  });
  
  return result;
});

// 当前显示的模型（分页后的）
const displayedModels = computed(() => {
  return filteredModels.value.slice(0, currentPage.value * pageSize);
});

// 是否还有更多数据
const hasMore = computed(() => {
  if (props.searchKeyword.trim()) {
    // 搜索模式下，显示所有匹配结果
    return false;
  }
  // 正常模式下，基于总数判断
  return allModels.value.length < totalModels.value;
});

// 计算模型状态统计
const modelStats = computed(() => {
  const stats = {
    serving: 0,
    restricted: 0,
    offline: 0,
    maintenance: 0,
    total: filteredModels.value.length,
  };
  
  filteredModels.value.forEach(model => {
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

// 检查是否有活动的筛选器
const hasActiveFilters = computed(() => {
  return !!(statusFilter.value || typeFilter.value || parameterSizeFilter.value || props.searchKeyword);
});

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

// 切换收藏状态
const toggleFavorite = (model: ModelItem) => {
  console.log('切换收藏状态:', model.name);
  // TODO: 实现收藏功能
};

// 下载模型
const downloadModel = (model: ModelItem) => {
  console.log('下载模型:', model.name);
  // TODO: 实现下载功能
};

// 加载更多
const loadMore = async () => {
  if (loading.value || !hasMore.value) return;
  
  currentPage.value++;
  const result = await fetchModels(currentPage.value);
  allModels.value.push(...result.models);
  totalModels.value = result.total;
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

// 监听搜索关键词变化，重置分页
watch(() => props.searchKeyword, () => {
  currentPage.value = 1;
});

// 懒加载逻辑
let observer: IntersectionObserver;

// 组件挂载时初始化数据
onMounted(() => {
  console.log('ModelMarketplace 组件挂载');
  // 初始化数据
  initializeModels();
  
  // 设置懒加载
  if (loadTrigger.value) {
    observer = new IntersectionObserver(
      (entries) => {
        // 修复 entries 可能为空的问题
        if (entries && entries.length > 0) {
          const [entry] = entries;
          if (entry && entry.isIntersecting && hasMore.value && !loading.value) {
            loadMore();
          }
        }
      },
      { threshold: 0.1 }
    );
    
    observer.observe(loadTrigger.value);
  }
});

// 监听搜索关键词变化
watch(() => props.searchKeyword, () => {
  currentPage.value = 1;
  initializeModels();
});

// 暴露刷新方法给父组件
const refreshData = () => {
  console.log('ModelMarketplace 收到刷新指令');
  currentPage.value = 1;
  allModels.value = [];
  totalModels.value = 0;
  loading.value = false;
  
  // 强制重新初始化数据
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

onUnmounted(() => {
  if (observer) {
    observer.disconnect();
  }
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
