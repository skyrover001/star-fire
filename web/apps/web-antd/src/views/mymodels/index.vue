<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import { requestClient } from '#/api/request';
import MyUsedModels from './components/MyUsedModels.vue';
import MyProvidedModels from './components/MyProvidedModels.vue';

// 当前标签页
const activeTab = ref('used');

// 统计数据
const stats = reactive({
  usedModels: 0,
  providedModels: 0,
  totalApiCalls: 0,
  activeApiKeys: 0,
});

// 加载统计数据
const loadStats = async () => {
  try {
    const response = await requestClient.get('/user/model-stats');
    Object.assign(stats, response);
  } catch (error) {
    console.error('加载统计数据失败:', error);
  }
};

// 页面挂载时加载数据
onMounted(() => {
  loadStats();
});

// 刷新数据
const refreshData = () => {
  loadStats();
  // 触发子组件刷新
  if (activeTab.value === 'used') {
    // 通过 ref 或事件通知子组件刷新
  } else {
    // 通知提供的模型组件刷新
  }
  message.success('数据已刷新');
};

</script>

<template>
  <div>
    <!-- 全屏布局，与侧边栏一致的背景 -->
    <div class="min-h-screen bg-[var(--bg-color)]">
      <!-- 顶部标题栏 -->
      <div class="px-6 py-6">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-3xl font-bold text-[var(--text-primary)]">我的模型</h1>
            <p class="mt-2 text-[var(--text-secondary)]">管理您使用的模型和提供的模型服务</p>
          </div>
          <div class="flex items-center space-x-4">
            <!-- 刷新按钮 -->
            <button
              class="inline-flex items-center rounded-lg bg-[var(--color-neutral-700)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--color-neutral-600)] focus:outline-none focus:ring-2 focus:ring-gray-500"
              @click="refreshData"
            >
              <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              刷新数据
            </button>
          </div>
        </div>
      </div>

      <!-- 统计卡片 -->
      <div class="px-6 pb-6">
        <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <!-- 使用的模型数量 -->
          <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
            <div class="flex items-center">
              <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/20">
                <svg class="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                </svg>
              </div>
              <div class="ml-4">
                <p class="text-2xl font-semibold text-[var(--text-primary)]">{{ stats.usedModels }}</p>
                <p class="text-sm text-[var(--text-secondary)]">使用的模型</p>
              </div>
            </div>
          </div>

          <!-- 提供的模型数量 -->
          <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
            <div class="flex items-center">
              <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/20">
                <svg class="h-6 w-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/>
                </svg>
              </div>
              <div class="ml-4">
                <p class="text-2xl font-semibold text-[var(--text-primary)]">{{ stats.providedModels }}</p>
                <p class="text-sm text-[var(--text-secondary)]">提供的模型</p>
              </div>
            </div>
          </div>

          <!-- API调用总数 -->
          <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
            <div class="flex items-center">
              <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-purple-100 dark:bg-purple-900/20">
                <svg class="h-6 w-6 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                </svg>
              </div>
              <div class="ml-4">
                <p class="text-2xl font-semibold text-[var(--text-primary)]">{{ stats.totalApiCalls.toLocaleString() }}</p>
                <p class="text-sm text-[var(--text-secondary)]">API调用总数</p>
              </div>
            </div>
          </div>

          <!-- 活跃API密钥 -->
          <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
            <div class="flex items-center">
              <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-orange-100 dark:bg-orange-900/20">
                <svg class="h-6 w-6 text-orange-600 dark:text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                </svg>
              </div>
              <div class="ml-4">
                <p class="text-2xl font-semibold text-[var(--text-primary)]">{{ stats.activeApiKeys }}</p>
                <p class="text-sm text-[var(--text-secondary)]">活跃API密钥</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 标签页导航 -->
      <div class="px-6">
        <div class="border-b border-[var(--border-color)]">
          <nav class="-mb-px flex space-x-8">
            <button
              class="py-4 px-1 border-b-2 font-medium text-sm transition-colors"
              :class="activeTab === 'used' 
                ? 'border-blue-500 text-blue-600 dark:text-blue-400' 
                : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-gray-300'"
              @click="activeTab = 'used'"
            >
              <div class="flex items-center">
                <svg class="mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                </svg>
                我使用的模型
              </div>
            </button>
            <button
              class="py-4 px-1 border-b-2 font-medium text-sm transition-colors"
              :class="activeTab === 'provided' 
                ? 'border-blue-500 text-blue-600 dark:text-blue-400' 
                : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-gray-300'"
              @click="activeTab = 'provided'"
            >
              <div class="flex items-center">
                <svg class="mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/>
                </svg>
                我提供的模型
              </div>
            </button>
          </nav>
        </div>
      </div>

      <!-- 标签页内容 -->
      <div class="px-6 py-6">
        <!-- 我使用的模型 -->
        <div v-if="activeTab === 'used'">
          <!-- <MyUsedModels /> -->
          <div class="text-center py-12">
            <p class="text-[var(--text-secondary)]">我使用的模型组件（开发中）</p>
          </div>
        </div>

        <!-- 我提供的模型 -->
        <div v-if="activeTab === 'provided'">
          <!-- <MyProvidedModels /> -->
          <div class="text-center py-12">
            <p class="text-[var(--text-secondary)]">我提供的模型组件（开发中）</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
