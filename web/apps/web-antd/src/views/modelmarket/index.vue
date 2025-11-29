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
    refreshData();
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
const refreshData = () => {
  searchKeyword.value = '';

  // 直接调用子组件的数据刷新方法
  setTimeout(() => {
    if (modelMarketplaceRef.value && (modelMarketplaceRef.value as any).refreshData) {
      (modelMarketplaceRef.value as any).refreshData();
    }
    if (modelTrendsRef.value && (modelTrendsRef.value as any).refreshData) {
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
      message.success('Token生成成功');
    } else {
      message.error('Token生成失败');
    }
  } catch (error) {
    console.error('生成Token失败:', error);
    message.error('Token生成失败，请稍后重试');
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
      message.success('Token已复制到剪贴板');
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
      message.success('Token已复制到剪贴板');
    } else {
      throw new Error('Copy failed');
    }
  } catch (error) {
    console.error('复制失败:', error);
    message.error('复制失败，请手动复制');
  }
};

// 复制使用命令到剪贴板
const copyCommand = async () => {
  if (!currentToken.value) return;

  const command = `starfire.exe -host ${serverHost} -token ${currentToken.value} -ippm 3.8 -oppm 8.3`;

  try {
    // 优先使用现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(command);
      message.success('使用命令已复制到剪贴板');
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
      message.success('使用命令已复制到剪贴板');
    } else {
      throw new Error('Copy failed');
    }
  } catch (error) {
    console.error('复制失败:', error);
    message.error('复制失败，请手动复制');
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
    message.error('请先生成Token');
    return;
  }

  try {
    message.loading('正在注册到Star Fire平台...', 0);
    const response = await requestClient.post('/api/starfire/register', {
      token: currentToken.value
    });

    message.destroy();
    if (response.success) {
      message.success('已成功注册到Star Fire平台！您的模型将自动加入算力网络');
      closeTokenModal();
      refreshData(); // 刷新页面数据
    } else {
      message.error(response.message || '注册失败，请稍后重试');
    }
  } catch (error) {
    message.destroy();
    console.error('注册到Star Fire失败:', error);
    message.error('注册失败，请检查网络连接后重试');
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
      size: '~45MB'
    },
    macos: {
      url: '/download/macos/starfire',
      filename: 'starfire.zip',
      size: '~52MB'
    }
  };

  const clientInfo = downloadUrls[platform];
  const platformName = platform === 'windows' ? 'Windows' : 'macOS';
  
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
    
    message.success(`正在下载 ${platformName} 客户端 (${clientInfo.size})`);
    
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
    message.error(`下载 ${platformName} 客户端失败，请稍后重试`);
  }
};

// 使用说明的Markdown内容
const usageGuideMarkdown = `
# 客户端注册使用说明

## 1. 系统要求

### Windows 系统
- Windows 10 或更高版本
- 至少 4GB 内存
- 500MB 可用磁盘空间

### macOS 系统
- macOS 10.15 (Catalina) 或更高版本
- 至少 4GB 内存
- 500MB 可用磁盘空间

## 2. 安装步骤

### Windows 安装
1. 下载 \`starfire.rar\` 压缩包
2. 解压缩到目标目录
3. 运行解压后的starfire.exe程序
4. 启动应用程序

### macOS 安装
1. 下载 \`starfire\` 应用程序
2. 将下载的文件移动到 Applications 文件夹
3. 在终端中运行 chmod +x starfire 赋予执行权限
4. 在应用程序文件夹中启动客户端

## 3. 客户端配置

### 首次配置
1. 启动客户端后，进入设置页面
2. 配置服务器地址：\`https://your-server.com/api\`
3. 输入您的用户名和密码
4. 点击"测试连接"验证配置

### 注册Token配置
1. 在模型广场页面生成注册Token
2. 复制Token到客户端的"Token"输入框
3. 点击"验证Token"确认有效性

## 4. 模型注册流程

### 准备工作
- 确保模型文件完整且格式正确
- 准备模型描述和相关文档
- 确认模型兼容性要求

### 注册步骤
1. 在客户端中选择"注册新模型"
2. 填写模型基本信息：
   - 模型名称
   - 版本号
   - 描述信息
   - 参数规模
   - 模型类型
3. 上传模型文件（支持多种格式）
4. 配置模型运行参数
5. 提交注册申请

### 状态跟踪
- **等待审核**：模型已提交，等待管理员审核
- **审核通过**：模型已通过审核，正在部署
- **部署完成**：模型已成功部署，可以使用
- **审核拒绝**：模型未通过审核，需要修改后重新提交

## 5. 常见问题

### Q: 客户端连接不上服务器？
A: 请检查：
- 网络连接是否正常
- 服务器地址是否正确
- 防火墙设置是否阻止了连接

### Q: Token验证失败？
A: 请确认：
- Token是否过期
- Token是否复制完整
- 用户账户是否有注册权限

### Q: 模型上传失败？
A: 可能的原因：
- 文件格式不支持
- 文件大小超过限制
- 网络连接中断

### Q: 如何更新客户端？
A: 客户端会自动检查更新，也可以：
- 在菜单中选择"检查更新"
- 重新下载最新版本安装包

## 6. 技术支持

如果您在使用过程中遇到问题，可以通过以下方式获取帮助：

- 📧 邮箱：support@example.com
- 💬 在线客服：工作日 9:00-18:00
- 📖 文档中心：https://docs.example.com
- 🐛 问题反馈：https://github.com/example/issues

---

*最后更新时间：2024年12月*
`;

// 将Markdown转换为HTML（简单实现）
const usageGuideHtml = computed(() => {
  return usageGuideMarkdown
    .replace(/^# (.+)$/gm, '<h1 class="text-2xl font-bold mb-4 text-gray-900 dark:text-white">$1</h1>')
    .replace(/^## (.+)$/gm, '<h2 class="text-xl font-semibold mb-3 mt-6 text-gray-800 dark:text-gray-100">$1</h2>')
    .replace(/^### (.+)$/gm, '<h3 class="text-lg font-medium mb-2 mt-4 text-gray-700 dark:text-gray-200">$1</h3>')
    .replace(/^\*\*(.+)\*\*：(.+)$/gm, '<p class="mb-2"><strong class="text-gray-900 dark:text-white">$1</strong>：$2</p>')
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
          <h1 class="text-3xl font-bold text-[var(--text-primary)]">模型广场</h1>
          <p class="mt-2 text-[var(--text-secondary)]">分享你的模型、使用他人模型，开启智能应用之旅</p>
        </div>
        <div class="flex items-center space-x-4">
          <!-- 手动刷新按钮 -->
          <button
            class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
            @click="refreshData">
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            刷新
          </button>
          <!-- 下载客户端按钮 -->
          <button
            class="inline-flex items-center rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 transition-colors"
            @click="showDownloadModal = true">
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
            </svg>
            下载客户端
          </button>
          <!-- 使用说明 -->
          <button
            class="inline-flex items-center rounded-lg bg-[var(--color-neutral-700)] px-4 py-2 text-sm font-medium text-white backdrop-blur-sm hover:bg-[var(--color-neutral-600)] focus:outline-none focus:ring-2 focus:ring-blue-500"
            @click="showUsageGuide">
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            使用说明
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
          <input v-model="searchKeyword" type="text" placeholder="搜索模型名称、类型或创建者..."
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
              注册到 Star Fire
            </button>
          </div>

          <!-- 模型动态面板 -->
          <div class="rounded-xl bg-[var(--content-bg)] p-6">
            <h3 class="mb-4 text-lg font-semibold text-[var(--text-primary)] flex items-center">
              <svg class="mr-2 h-5 w-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              模型动态
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
              <h3 class="text-xl font-semibold text-gray-900 dark:text-white">注册模型到 Star Fire</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">共享算力，让更多人使用您的模型</p>
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
                <h4 class="text-sm font-medium text-purple-800 dark:text-purple-200 mb-1">关于 Star Fire 平台</h4>
                <p class="text-sm text-purple-700 dark:text-purple-300">
                  Star Fire 是一个分布式算力共享平台，您可以将本地运行的模型注册到平台上，让其他用户通过API调用您的模型，同时获得算力收益。
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
                注册Token
              </span>
            </label>
            <div class="relative">
              <input :value="currentToken" type="text" readonly placeholder="正在生成Token..."
                class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 pr-20 text-sm font-mono text-gray-900 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400" />
              <div class="absolute inset-y-0 right-0 flex items-center space-x-1 pr-2">
                <button v-if="currentToken" @click="copyToken"
                  class="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-600 dark:hover:text-gray-300"
                  title="复制Token">
                  <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
                <button @click="regenerateToken" :disabled="isGeneratingToken"
                  class="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 disabled:opacity-50 dark:hover:bg-gray-600 dark:hover:text-gray-300"
                  title="重新生成">
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
                <h4 class="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">使用命令</h4>
                <p class="text-sm text-blue-700 dark:text-blue-300 mb-3">
                  将以下命令复制到终端中运行，启动 Star Fire 客户端：
                </p>
                <div class="relative">
                  <code
                    class="block w-full rounded-lg bg-gray-100 dark:bg-gray-800 p-3 text-sm font-mono text-gray-900 dark:text-gray-100 pr-12 break-all">
                  starfire.exe -host {{ serverHost }} -token {{ currentToken }} -ippm 3.8 -oppm 8.3
                </code>
                  <button @click="copyCommand"
                    class="absolute top-2 right-2 rounded p-1.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                    title="复制命令">
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </div>
                <div class="mt-2 text-xs text-blue-600 dark:text-blue-400">
                  <p>💡 确保 starfire.exe 在您的系统 PATH 中，或使用完整路径</p>
                  <p>🌐 主机地址已自动配置，如需修改请联系管理员</p>
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
                <h4 class="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-1">重要提示</h4>
                <ul class="text-sm text-yellow-700 dark:text-yellow-300 space-y-1">
                  <li>• 注册Token单次有效，请不要将自己Token泄露给其他人</li>
                  <li>• 根据您的本地配置，可适当设置并发数（ollam: OLLAMA_NUM_PARALLEL=4），提升模型被调用概率</li>
                  <li>• Token注册成功后，您的模型将自动加入Star Fire算力网络</li>
                  <li>• 系统会根据您贡献模型的被调用tokens数给予相应的算力收益</li>
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
              <span>Star Fire 算力网络</span>
            </div>
            <div class="flex space-x-3">
              <button @click="closeTokenModal"
                class="inline-flex items-center rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600">
                取消
              </button>
              <button v-if="currentToken" @click="copyToken"
                class="inline-flex items-center rounded-lg bg-gradient-to-r from-purple-600 to-pink-600 px-4 py-2 text-sm font-medium text-white hover:from-purple-700 hover:to-pink-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                复制Token
              </button>
              <button v-if="currentToken" @click="registerToStarFire"
                class="inline-flex items-center rounded-lg bg-gradient-to-r from-green-600 to-emerald-600 px-4 py-2 text-sm font-medium text-white hover:from-green-700 hover:to-emerald-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
                注册到 Star Fire
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
          <h3 class="text-xl font-semibold text-gray-900 dark:text-white">使用说明</h3>
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
          <h3 class="text-xl font-semibold text-gray-900 dark:text-white">下载客户端</h3>
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
          <p class="mb-6 text-gray-600 dark:text-gray-300">选择适合您操作系统的客户端版本，下载后即可注册和管理模型：</p>

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
                  <h4 class="font-medium text-gray-900 dark:text-white">Windows APP</h4>
                  <p class="text-sm text-gray-500 dark:text-gray-400">支持 Windows 10/11 压缩包 (~45MB)</p>
                </div>
              </div>
              <button
                class="inline-flex items-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
                @click="downloadClient('windows')">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
                </svg>
                下载
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
                  <p class="text-sm text-gray-500 dark:text-gray-400">支持 macOS 10.15+ 可执行文件 (~52MB)</p>
                </div>
              </div>
              <button
                class="inline-flex items-center rounded-md bg-gray-600 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500"
                @click="downloadClient('macos')">
                <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
                </svg>
                下载
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
                  Windows: 下载rar压缩包并解压后运行；macOS: 下载可执行文件并添加执行权限。使用方式：starfire -host {host_ip:{port}} -token YOUR_TOKEN
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
