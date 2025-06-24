<template>
  <div>
    <!-- 全屏布局，与侧边栏一致的背景 -->
    <div class="min-h-screen bg-[var(--bg-color)]">
      <!-- 顶部标题栏 -->
      <div class="px-6 py-6">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-3xl font-bold text-[var(--text-primary)]">API Key 管理</h1>
            <p class="mt-2 text-[var(--text-secondary)]">管理您的 API 密钥，安全访问模型服务</p>
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

      <!-- API Key 管理组件（包含统计卡片） -->
      <div class="px-6 pb-6">
        <ApiKeyManagement ref="apiKeyManagementRef" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { message } from 'ant-design-vue';
import ApiKeyManagement from './components/ApiKeyManagement.vue';

// API Key管理组件引用
const apiKeyManagementRef = ref();

// 刷新数据
const refreshData = () => {
  // 刷新API Key管理组件数据
  if (apiKeyManagementRef.value && typeof apiKeyManagementRef.value.refreshApiKeys === 'function') {
    apiKeyManagementRef.value.refreshApiKeys();
  }
  message.success('数据已刷新');
};
</script>
