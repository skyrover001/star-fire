<template>
  <div class="space-y-6">
    <!-- 模型列表 -->
    <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] overflow-hidden">
      <!-- 表头 -->
      <div class="px-6 py-4 bg-[var(--bg-color-secondary)] border-b border-[var(--border-color)]">
        <div class="grid grid-cols-12 gap-4 text-sm font-medium text-[var(--text-secondary)]">
          <div class="col-span-3">{{ $t('business.myModels.modelName') }}</div>
          <div class="col-span-2">{{ $t('business.myModels.provider') }}</div>
          <div class="col-span-2">{{ $t('business.myModels.addedAt') }}</div>
          <div class="col-span-2">{{ $t('business.myModels.usageCount') }}</div>
          <div class="col-span-2">{{ $t('business.myModels.status') }}</div>
          <div class="col-span-1">{{ $t('business.myModels.actions') }}</div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="px-6 py-12 text-center">
        <div class="inline-flex items-center">
          <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-blue-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ $t('business.myModels.loading') }}
        </div>
      </div>

      <!-- 模型列表项 -->
      <div v-else-if="usedModels.length > 0" class="divide-y divide-[var(--border-color)]">
        <div
          v-for="model in usedModels"
          :key="model.id"
          class="px-6 py-4 hover:bg-[var(--bg-color-secondary)] transition-colors"
        >
          <div class="grid grid-cols-12 gap-4 items-center">
            <!-- 模型名称 -->
            <div class="col-span-3">
              <div class="flex items-center">
                <div class="flex-shrink-0 h-10 w-10">
                  <img 
                    class="h-10 w-10 rounded-lg object-cover" 
                    :src="model.avatar || '/default-model-avatar.png'" 
                    :alt="model.name"
                    @error="handleImageError"
                  >
                </div>
                <div class="ml-3">
                  <p class="text-sm font-medium text-[var(--text-primary)]">{{ model.name }}</p>
                  <p class="text-xs text-[var(--text-secondary)]">{{ model.description }}</p>
                </div>
              </div>
            </div>

            <!-- 提供者 -->
            <div class="col-span-2">
              <p class="text-sm text-[var(--text-secondary)]">{{ model.provider }}</p>
            </div>

            <!-- 添加时间 -->
            <div class="col-span-2">
              <p class="text-sm text-[var(--text-secondary)]">{{ formatDate(model.addedTime) }}</p>
            </div>

            <!-- 使用次数 -->
            <div class="col-span-2">
              <p class="text-sm text-[var(--text-primary)]">{{ model.usageCount.toLocaleString() }}</p>
            </div>

            <!-- 状态 -->
            <div class="col-span-2">
              <span 
                class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                :class="getStatusClass(model.status)"
              >
                {{ getStatusText(model.status) }}
              </span>
            </div>

            <!-- 操作 -->
            <div class="col-span-1">
              <div class="flex items-center space-x-2">
                <button
                  class="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 text-sm"
                  @click="viewModelDetails(model)"
                  :title="$t('business.myModels.viewDetails')"
                >
                  {{ $t('business.myModels.details') }}
                </button>
                <button
                  class="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 text-sm"
                  @click="removeModel(model)"
                  :title="$t('business.myModels.removeTitle')"
                >
                  {{ $t('business.myModels.remove') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="px-6 py-12 text-center">
        <svg class="mx-auto h-12 w-12 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
        </svg>
        <h3 class="mt-2 text-sm font-medium text-[var(--text-primary)]">{{ $t('business.myModels.noUsedModels') }}</h3>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">{{ $t('business.marketplace.noModels') }}</p>
        <div class="mt-6">
          <button
            class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            @click="goToMarketplace"
          >
            <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
            {{ $t('business.myModels.exploreModels') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { message } from 'ant-design-vue';
import { requestClient } from '#/api/request';
import { $t } from '#/locales';

// 接口类型定义
interface UsedModel {
  id: string;
  name: string;
  description: string;
  provider: string;
  avatar: string;
  addedTime: string;
  usageCount: number;
  status: 'active' | 'inactive' | 'error';
}

const router = useRouter();

// 响应式数据
const loading = ref(false);
const usedModels = ref<UsedModel[]>([]);

// 加载使用的模型
const loadUsedModels = async () => {
  loading.value = true;
  try {
    const response = await requestClient.get('/user/used-models');
    usedModels.value = response.data || [];
  } catch (error) {
    console.error('加载使用的模型失败:', error);
    message.error($t('business.myModels.loadModelsFailed'));
  } finally {
    loading.value = false;
  }
};

// 处理图片加载错误
const handleImageError = (event: Event) => {
  const img = event.target as HTMLImageElement;
  img.src = '/default-model-avatar.png';
};

// 获取状态样式
const getStatusClass = (status: string) => {
  switch (status) {
    case 'active':
      return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400';
    case 'inactive':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400';
    case 'error':
      return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400';
  }
};

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'active':
      return $t('business.myModels.active');
    case 'inactive':
      return $t('business.myModels.inactive');
    case 'error':
      return $t('business.myModels.error');
    default:
      return $t('business.myModels.unknown');
  }
};

// 查看模型详情
const viewModelDetails = (model: UsedModel) => {
  message.info($t('business.myModels.viewModelDetails', { model: model.name }));
};

// 移除模型
const removeModel = async (model: UsedModel) => {
  if (!confirm($t('business.myModels.removeConfirm', { model: model.name }))) {
    return;
  }

  try {
    await requestClient.delete(`/user/used-models/${model.id}`);
    message.success($t('business.myModels.modelRemoved'));
    await loadUsedModels();
  } catch (error) {
    console.error('移除模型失败:', error);
    message.error($t('business.myModels.removeFailed'));
  }
};

// 前往模型广场
const goToMarketplace = () => {
  router.push('/model-marketplace');
};

// 格式化日期
const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleDateString('zh-CN');
};

// 页面挂载时加载数据
onMounted(() => {
  loadUsedModels();
});

// 模拟数据 (开发期间使用)
if (import.meta.env.DEV) {
  usedModels.value = [
    {
      id: '1',
      name: 'GPT-4o',
      description: '最新的多模态大语言模型',
      provider: 'OpenAI',
      avatar: 'https://avatar.vercel.sh/gpt4o',
      addedTime: '2025-05-20T10:30:00Z',
      usageCount: 1250,
      status: 'active',
    },
    {
      id: '2',
      name: 'Claude-3.5-Sonnet',
      description: '高质量的对话和推理模型',
      provider: 'Anthropic',
      avatar: 'https://avatar.vercel.sh/claude',
      addedTime: '2025-05-18T14:20:00Z',
      usageCount: 856,
      status: 'active',
    },
    {
      id: '3',
      name: 'Gemini-1.5-Pro',
      description: '谷歌最新的多模态模型',
      provider: 'Google',
      avatar: 'https://avatar.vercel.sh/gemini',
      addedTime: '2025-05-15T09:45:00Z',
      usageCount: 432,
      status: 'inactive',
    },
  ];
}

// 暴露方法供父组件调用
defineExpose({
  refresh: loadUsedModels,
});
</script>
