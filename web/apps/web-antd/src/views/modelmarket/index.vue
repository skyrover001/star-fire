<script lang="ts" setup>
import type {
  WorkbenchProjectItem,
  WorkbenchQuickNavItem,
} from '@vben/common-ui';

import { computed, ref, watch, onMounted, onUnmounted, onActivated } from 'vue';
import { useRouter, useRoute } from 'vue-router';

import { openWindow } from '@vben/utils';
import { useAppConfig } from '@vben/hooks';
// 导入请求工具
import { requestClient } from '#/api/request';
import { $t as t } from '#/locales';

import { message } from 'ant-design-vue';

import ModelMarketplace from './components/ModelMarketplace.vue';
import ModelTrends from './components/ModelTrends.vue';

const router = useRouter();
const route = useRoute();

// 获取应用配置
const { serverHost } = useAppConfig(import.meta.env, import.meta.env.PROD);

// 搜索相关状态
const searchKeyword = ref('');

// 模型广场组件的引用
const modelMarketplaceRef = ref(null);
const modelTrendsRef = ref(null);

// 对话框状态
const showUsageModal = ref(false);
const showDownloadModal = ref(false);
const showTokenModal = ref(false);

// Token相关状态
const isGeneratingToken = ref(false);
const currentToken = ref('');

// 定时刷新相关
const autoRefreshInterval = ref(10 * 1000); // 10秒，单位毫秒
let refreshTimer: NodeJS.Timeout | null = null;

// 监听路由变化和用户交互 - 简化版本
watch(() => route.path, (newPath) => {
  if (newPath === '/model-marketplace') {
    setTimeout(() => {
      refreshData();
    }, 100);
  }
}, { immediate: true });

// 页面挂载时刷新数据
onMounted(() => {
  refreshData();
  startAutoRefresh();

  // 监听页面可见性变化
  document.addEventListener('visibilitychange', handleVisibilityChange);
  window.addEventListener('focus', handleWindowFocus);
  window.addEventListener('blur', handleWindowBlur);
});

// 组件激活时也刷新数据
onActivated(() => {
  refreshData();
});

// 页面卸载时清理定时器
onUnmounted(() => {
  stopAutoRefresh();

  // 清理事件监听器
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  window.removeEventListener('focus', handleWindowFocus);
  window.removeEventListener('blur', handleWindowBlur);
});

// 处理页面可见性变化
const handleVisibilityChange = () => {
  if (document.hidden) {
    // 页面不可见时暂停自动刷新
    stopAutoRefresh();
  } else {
    // 页面可见时恢复自动刷新
    startAutoRefresh();
    // 如果当前在模型广场页面，重新加载数据
    if (route.name === 'ModelMarketplace') {
      setTimeout(() => {
        refreshData();
      }, 100);
    }
  }
};

// 处理窗口获得焦点
const handleWindowFocus = () => {
  if (!document.hidden) {
    startAutoRefresh();
  }
  // 如果当前在模型广场页面，重新加载数据
  if (route.name === 'ModelMarketplace') {
    setTimeout(() => {
      refreshData();
    }, 100);
  }
};

// 处理窗口失去焦点
const handleWindowBlur = () => {
  // 窗口失去焦点时可以选择是否暂停刷新
  // 这里不暂停，只有在页面不可见时才暂停
};

// 启动自动刷新
const startAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
  refreshTimer = setInterval(() => {
    refreshData(true); // 自动刷新使用静默模式，避免页面闪烁
  }, autoRefreshInterval.value);
};

// 停止自动刷新
const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
};

// 刷新数据方法
const refreshData = (silent = false) => {
  if (!silent) {
    searchKeyword.value = '';
  }

  // 直接调用子组件的数据刷新方法
  setTimeout(() => {
    if (modelMarketplaceRef.value) {
      const comp = modelMarketplaceRef.value as any;
      // 静默刷新：不显示全页 loading，只更新变化的数据，避免页面闪烁
      if (silent && comp.silentRefresh) {
        comp.silentRefresh();
      } else if (comp.refreshData) {
        comp.refreshData();
      }
    }
    if (!silent && modelTrendsRef.value && (modelTrendsRef.value as any).refreshData) {
      (modelTrendsRef.value as any).refreshData();
    }
  }, 100);
};

// 这是一个示例方法，实际项目中需要根据实际情况进行调整
// This is a sample method, adjust according to the actual project requirements
function navTo(nav: WorkbenchProjectItem | WorkbenchQuickNavItem) {
  if (nav.url?.startsWith('http')) {
    openWindow(nav.url);
    return;
  }
  if (nav.url?.startsWith('/')) {
    router.push(nav.url).catch((error) => {
      console.error('Navigation failed:', error);
    });
  } else {
    console.warn(`Unknown URL for navigation item: ${nav.title} -> ${nav.url}`);
  }
}

// 处理搜索
const handleSearch = () => {
  // 搜索逻辑在 ModelMarketplace 组件中处理
};

// 处理来自子组件的搜索事件
const handleSearchFromChild = (keyword: string) => {
  searchKeyword.value = keyword;
};

// 显示注册Token对话框
const showRegisterToken = async () => {
  showTokenModal.value = true;
  await generateToken();
};

// 关闭Token对话框
const closeTokenModal = () => {
  showTokenModal.value = false;
  currentToken.value = '';
};

// 生成Token
const generateToken = async () => {
  if (isGeneratingToken.value) return;

  isGeneratingToken.value = true;
  try {
    const response = await requestClient.post('/user/register-token');
    if (response.token) {
      currentToken.value = response.token;
      message.success(t('business.marketplace.tokenGenerated'));
    } else {
      message.error(t('business.marketplace.tokenGenerationFailed'));
    }
  } catch (error) {
    console.error('生成Token失败:', error);
    message.error(t('business.marketplace.tokenGenerationRetry'));
  } finally {
    isGeneratingToken.value = false;
  }
};

// 复制Token到剪贴板
const copyToken = async () => {
  if (!currentToken.value) return;

  try {
    // 优先使用现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(currentToken.value);
      message.success(t('business.marketplace.tokenCopied'));
      return;
    }

    // Fallback: 使用传统方法
    const textArea = document.createElement('textarea');
    textArea.value = currentToken.value;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    const successful = document.execCommand('copy');
    document.body.removeChild(textArea);

    if (successful) {
      message.success(t('business.marketplace.tokenCopied'));
    } else {
      throw new Error('Copy failed');
    }
  } catch (error) {
    console.error('复制失败:', error);
    message.error(t('business.marketplace.copyFailed'));
  }
};

// 复制使用命令到剪贴板
const copyCommand = async () => {
  if (!currentToken.value) return;

  const command = `starfire.exe -host ${serverHost} -token ${currentToken.value} -ippm 3.8 -oppm 8.3 -cippm 1.0`;

  try {
    // 优先使用现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(command);
      message.success(t('business.marketplace.commandCopied'));
      return;
    }

    // Fallback: 使用传统方法
    const textArea = document.createElement('textarea');
    textArea.value = command;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    const successful = document.execCommand('copy');
    document.body.removeChild(textArea);

    if (successful) {
      message.success(t('business.marketplace.commandCopied'));
    } else {
      throw new Error('Copy failed');
    }
  } catch (error) {
    console.error('复制失败:', error);
    message.error(t('business.marketplace.copyFailed'));
  }
};

// 重新生成Token
const regenerateToken = () => {
  currentToken.value = '';
  generateToken();
};

// 注册到Star Fire平台
const registerToStarFire = async () => {
  if (!currentToken.value) {
    message.error(t('business.marketplace.generateTokenFirst'));
    return;
  }

  try {
    message.loading(t('business.marketplace.registering'), 0);
    const response = await requestClient.post('/api/starfire/register', {
      token: currentToken.value
    });

    message.destroy();
    if (response.success) {
      message.success(t('business.marketplace.registrationSucceeded'));
      closeTokenModal();
      refreshData(); // 刷新页面数据
    } else {
      message.error(response.message || t('business.marketplace.registrationRetry'));
    }
  } catch (error) {
    message.destroy();
    console.error('注册到Star Fire失败:', error);
    message.error(t('business.marketplace.registrationNetworkRetry'));
  }
};

const showUsageGuide = () => {
  showUsageModal.value = true;
};

// 关闭使用说明对话框
const closeUsageModal = () => {
  showUsageModal.value = false;
};

// 关闭客户端下载对话框
const closeDownloadModal = () => {
  showDownloadModal.value = false;
};

// 下载客户端
const downloadClient = (platform: 'windows' | 'macos') => {
  // 客户端下载链接配置
  const downloadUrls = {
    windows: {
      url: '/download/windows/starfire.rar',
      filename: 'starfire.rar',
      size: '~20MB'
    },
    macos: {
      url: '/download/macos/starfire.zip',
      filename: 'starfire',
      size: '~20MB'
    }
  };

  const clientInfo = downloadUrls[platform];
  const platformName = t(`business.marketplace.${platform}Platform`);

  try {
    // 创建下载链接
    const link = document.createElement('a');
    link.href = clientInfo.url;
    link.download = clientInfo.filename;
    link.style.display = 'none';
    document.body.appendChild(link);

    // 触发下载
    link.click();

    // 清理
    document.body.removeChild(link);

    message.success(t('business.marketplace.downloadingClient', { platform: platformName, size: clientInfo.size }));

    // 可选：记录下载统计
    requestClient.post('/api/stats/client-download', {
      platform: platform,
      timestamp: Date.now(),
      userAgent: navigator.userAgent
    }).catch(() => {
      // 忽略统计错误，不影响下载
    });

  } catch (error) {
    console.error('下载客户端失败:', error);
    message.error(t('business.marketplace.downloadFailed', { platform: platformName }));
  }
};

// 使用说明的Markdown内容
const usageGuideMarkdown = computed(() => t('business.marketplace.usageGuideMarkdown'));

// 将Markdown转换为HTML（简单实现）
const usageGuideHtml = computed(() => {
  return usageGuideMarkdown.value
    .replace(/^# (.+)$/gm, '<h1 class="text-2xl font-bold mb-4 text-gray-900 dark:text-white">$1</h1>')
    .replace(/^## (.+)$/gm, '<h2 class="text-xl font-semibold mb-3 mt-6 text-gray-800 dark:text-gray-100">$1</h2>')
    .replace(/^### (.+)$/gm, '<h3 class="text-lg font-medium mb-2 mt-4 text-gray-700 dark:text-gray-200">$1</h3>')
    .replace(/^\*\*(.+)\*\*[:：](.+)$/gm, '<p class="mb-2"><strong class="text-gray-900 dark:text-white">$1</strong>：$2</p>')
    .replace(/^- (.+)$/gm, '<li class="mb-1 text-gray-600 dark:text-gray-300">$1</li>')
    .replace(/(\d+)\. (.+)$/gm, '<div class="mb-2"><span class="font-medium text-blue-600 dark:text-blue-400">$1.</span> $2</div>')
    .replace(/`([^`]+)`/g, '<code class="px-2 py-1 bg-gray-100 rounded text-sm font-mono dark:bg-gray-700">$1</code>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
    .replace(/\n\n/g, '</p><p class="mb-3 text-gray-600 dark:text-gray-300">')
    .replace(/^(.+)$/gm, '<p class="mb-3 text-gray-600 dark:text-gray-300">$1</p>')
    .replace(/---/g, '<hr class="my-6 border-gray-200 dark:border-gray-700">')
    .replace(/📧|💬|📖|🐛/g, '<span class="mr-1">$&</span>');
});
</script>

<template>
  <div>
  <!-- 全屏布局，与侧边栏一致的背景 -->
  <div class="min-h-screen bg-[var(--bg-color)]">
    <!-- 顶部标题栏 -->
    <div class="px-6 py-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-bold text-[var(--text-primary)]">{{ $t('business.marketplace.title') }}</h1>
          <p class="mt-2 text-[var(--text-secondary)]">{{ $t('business.marketplace.subtitle') }}</p>
        </div>
        <div class="flex items-center space-x-4">
          <!-- 手动刷新按钮 -->
          <button
            class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
            @click="refreshData()">
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ $t('business.marketplace.refresh') }}
          </button>
          <!-- 下载客户端按钮 -->
          <button
            class="inline-flex items-center rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 transition-colors"
            @click="showDownloadModal = true">
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
            </svg>
            {{ $t('business.marketplace.downloadClient') }}
          </button>
          <!-- 使用说明 -->
          <button
            class="inline-flex items-center rounded-lg bg-[var(--color-neutral-700)] px-4 py-2 text-sm font-medium text-white backdrop-blur-sm hover:bg-[var(--color-neutral-600)] focus:outline-none focus:ring-2 focus:ring-blue-500"
            @click="showUsageGuide">
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {{ $t('business.marketplace.usageGuide') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="px-6 pb-6">
      <!-- 搜索区域 -->
      <div class="mb-6">
        <div class="relative">
          <div class="absolute inset-y-0 left-0 flex items-center pl-4">
            <svg class="h-5 w-5 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <input v-model="searchKeyword" type="text" :placeholder="$t('business.marketplace.marketSearchPlaceholder')"
            class="w-full rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] py-4 pl-12 pr-4 text-[var(--text-primary)] placeholder-[var(--text-secondary)] focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
            @input="handleSearch" />
        </div>
      </div>

      <!-- 左右分栏布局 -->
      <div class="grid grid-cols-1 xl:grid-cols-4 gap-6">
        <!-- 左侧：模型广场 - 占3/4宽度 -->
        <div class="xl:col-span-3 order-2 xl:order-1">
          <ModelMarketplace ref="modelMarketplaceRef" :search-keyword="searchKeyword"
            @nav-to="navTo" @search="handleSearchFromChild" />
        </div>

        <!-- 右侧：模型动态 - 占1/4宽度 -->
        <div class="xl:col-span-1 order-1 xl:order-2">
          <!-- 注册到Star Fire按钮 -->
          <div class="mb-6">
            <button
              class="w-full inline-flex items-center justify-center rounded-xl bg-gradient-to-r from-purple-600 to-pink-600 px-4 py-3 text-sm font-medium text-white hover:from-purple-700 hover:to-pink-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 transition-all duration-200 shadow-lg"
              @click="showRegisterToken">
              <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
              {{ $t('business.marketplace.registerToStarFire') }}
            </button>
          </div>

          <!-- 模型动态面板 -->
          <div class="rounded-xl bg-[var(--content-bg)] p-6">
            <h3 class="mb-4 text-lg font-semibold text-[var(--text-primary)] flex items-center">
              <svg class="mr-2 h-5 w-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              {{ $t('business.marketplace.modelTrends') }}
            </h3>
            <ModelTrends ref="modelTrendsRef" />
          </div>
        </div>
      </div>
    </div>

    <!-- Token注册对话框 -->
    <div v-if="showTokenModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      @click="closeTokenModal">
      <div class="relative mx-4 w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-gray-800"
        @click.stop>
        <!-- 对话框头部 -->
        <div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <div class="flex items-center">
            <div class="mr-3 rounded-lg bg-gradient-to-r from-purple-500 to-pink-500 p-2">
              <svg class="h-6 w-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
            </div>
            <div>
              <h3 class="text-xl font-semibold text-gray-900 dark:text-white">{{ $t('business.marketplace.registerModelTitle') }}</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('business.marketplace.registerModelSubtitle') }}</p>
            </div>
          </div>
          <button
            class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
            @click="closeTokenModal">
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- 对话框内容 -->
        <div class="p-6">
          <div
            class="mb-6 rounded-lg bg-gradient-to-r from-purple-50 to-pink-50 p-4 dark:from-purple-900/20 dark:to-pink-900/20">
            <div class="flex items-start">
              <div class="mr-3 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 p-1">
                <svg class="h-5 w-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div>
                <h4 class="text-sm font-medium text-purple-800 dark:text-purple-200 mb-1">{{ $t('business.marketplace.aboutPlatform') }}</h4>
                <p class="text-sm text-purple-700 dark:text-purple-300">
                  {{ $t('business.marketplace.platformDescription') }}
                </p>
              </div>
            </div>
          </div>

          <!-- Token显示区域 -->
          <div class="mb-6">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              <span class="flex items-center">
                <svg class="mr-2 h-4 w-4 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-3.586l4.293-4.293A6 6 0 0118 9z" />
                </svg>
                {{ $t('business.marketplace.registrationToken') }}
              </span>
            </label>
            <div class="relative">
              <input :value="currentToken" type="text" readonly :placeholder="$t('business.marketplace.generatingToken')"
                class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 pr-20 text-sm font-mono text-gray-900 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400" />
              <div class="absolute inset-y-0 right-0 flex items-center space-x-1 pr-2">
                <button v-if="currentToken" @click="copyToken"
                  class="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-600 dark:hover:text-gray-300"
                  :title="$t('business.marketplace.copyToken')">
                  <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
                <button @click="regenerateToken" :disabled="isGeneratingToken"
                  class="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 disabled:opacity-50 dark:hover:bg-gray-600 dark:hover:text-gray-300"
                  :title="$t('business.marketplace.regenerate')">
                  <svg class="h-4 w-4" :class="{ 'animate-spin': isGeneratingToken }" fill="none"
                    stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- 使用命令介绍 -->
          <div v-if="currentToken" class="mb-6 rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
            <div class="flex">
              <svg class="h-5 w-5 text-blue-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v14a2 2 0 002 2z" />
              </svg>
              <div class="ml-3 flex-1">
                <h4 class="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">{{ $t('business.marketplace.usageCommand') }}</h4>
                <p class="text-sm text-blue-700 dark:text-blue-300 mb-3">
                  {{ $t('business.marketplace.usageCommandDescription') }}
                </p>
                <div class="relative">
                  <code
                    class="block w-full rounded-lg bg-gray-100 dark:bg-gray-800 p-3 text-sm font-mono text-gray-900 dark:text-gray-100 pr-12 break-all">
                  starfire.exe -host {{ serverHost }} -token {{ currentToken }} -ippm 3.8 -oppm 8.3 -cippm 1.0
                </code>
                  <button @click="copyCommand"
                    class="absolute top-2 right-2 rounded p-1.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                    :title="$t('business.marketplace.copyCommand')">
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </div>
                <div class="mt-2 text-xs text-blue-600 dark:text-blue-400">
                  <p>{{ $t('business.marketplace.pathHint') }}</p>
                  <p>{{ $t('business.marketplace.hostHint') }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- 使用说明 -->
          <div class="mb-6 rounded-lg bg-yellow-50 p-4 dark:bg-yellow-900/20">
            <div class="flex">
              <svg class="h-5 w-5 text-yellow-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clip-rule="evenodd" />
              </svg>
              <div class="ml-3">
                <h4 class="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-1">{{ $t('business.marketplace.importantNotice') }}</h4>
                <ul class="text-sm text-yellow-700 dark:text-yellow-300 space-y-1">
                  <li>{{ $t('business.marketplace.tokenNotice') }}</li>
                  <li>{{ $t('business.marketplace.parallelismNotice') }}</li>
                  <li>{{ $t('business.marketplace.networkNotice') }}</li>
                  <li>{{ $t('business.marketplace.earningsNotice') }}</li>
                </ul>
              </div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-between items-center">
            <div class="flex items-center space-x-2 text-sm text-gray-500 dark:text-gray-400">
              <svg class="h-4 w-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{{ $t('business.marketplace.computeNetwork') }}</span>
            </div>
            <div class="flex space-x-3">
              <button @click="closeTokenModal"
                class="inline-flex items-center rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600">
                {{ $t('business.marketplace.cancel') }}
              </button>
              <button v-if="currentToken" @click="copyToken"
                class="inline-flex items-center rounded-lg bg-gradient-to-r from-purple-600 to-pink-600 px-4 py-2 text-sm font-medium text-white hover:from-purple-700 hover:to-pink-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                {{ $t('business.marketplace.copyToken') }}
              </button>
              <button v-if="currentToken" @click="registerToStarFire"
                class="inline-flex items-center rounded-lg bg-gradient-to-r from-green-600 to-emerald-600 px-4 py-2 text-sm font-medium text-white hover:from-green-700 hover:to-emerald-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
                {{ $t('business.marketplace.registerToStarFire') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 使用说明对话框 -->
    <div v-if="showUsageModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      @click="closeUsageModal">
      <div
        class="relative mx-4 max-h-[80vh] w-full max-w-4xl overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-gray-800"
        @click.stop>
        <!-- 对话框头部 -->
        <div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <h3 class="text-xl font-semibold text-gray-900 dark:text-white">{{ $t('business.marketplace.usageGuide') }}</h3>
          <button
            class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
            @click="closeUsageModal">
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <!-- 对话框内容 -->
        <div class="max-h-[60vh] overflow-y-auto p-6">
          <div class="prose prose-gray max-w-none dark:prose-invert" v-html="usageGuideHtml"></div>
        </div>
      </div>
    </div>

    <!-- 客户端下载对话框 -->
    <div v-if="showDownloadModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      @click="closeDownloadModal">
      <div class="relative mx-4 w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-gray-800"
        @click.stop>
        <!-- 对话框头部 -->
        <div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <h3 class="text-xl font-semibold text-gray-900 dark:text-white">{{ $t('business.marketplace.downloadClient') }}</h3>
          <button
            class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
            @click="closeDownloadModal">
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <!-- 对话框内容 -->
        <div class="p-6">
          <p class="mb-6 text-gray-600 dark:text-gray-300">{{ $t('business.marketplace.downloadDescription') }}</p>

          <div class="space-y-4">
            <!-- Windows版本 -->
            <div class="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              <div class="flex items-center space-x-3">
                <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/20">
                  <svg class="h-6 w-6 text-blue-600 dark:text-blue-400" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-13.051-1.351" />
                  </svg>
                </div>
                <div>
                  <h4 class="font-medium text-gray-900 dark:text-white">{{ $t('business.marketplace.windowsApp') }}</h4>
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('business.marketplace.windowsDescription') }}</p>
                </div>
              </div>
              <button
                class="inline-flex items-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
                @click="downloadClient('windows')">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
                </svg>
                {{ $t('business.marketplace.download') }}
              </button>
            </div>

            <!-- macOS版本 -->
            <div class="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              <div class="flex items-center space-x-3">
                <div class="rounded-lg bg-gray-100 p-2 dark:bg-gray-700">
                  <svg class="h-6 w-6 text-gray-600 dark:text-gray-400" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
                  </svg>
                </div>
                <div>
                  <h4 class="font-medium text-gray-900 dark:text-white">macOS</h4>
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('business.marketplace.macosDescription') }}</p>
                </div>
              </div>
              <button
                class="inline-flex items-center rounded-md bg-gray-600 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500"
                @click="downloadClient('macos')">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
                </svg>
                {{ $t('business.marketplace.download') }}
              </button>
            </div>
          </div>

          <div class="mt-6 rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
            <div class="flex">
              <svg class="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                  clip-rule="evenodd" />
              </svg>
              <div class="ml-3">
                <p class="text-sm text-blue-800 dark:text-blue-200">
                  {{ $t('business.marketplace.downloadHint') }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  </div>
</template>
