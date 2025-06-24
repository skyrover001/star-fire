<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { requestClient } from '#/api/request';

// 我提供的模型列表
const providedModels = ref([]);
const modelsLoading = ref(false);

// 显示模型详情对话框
const showModelDetailModal = ref(false);
const selectedModel = ref(null);

// 显示添加模型对话框
const showAddModelModal = ref(false);
const isAddingModel = ref(false);

// 新模型表单
const newModelForm = reactive({
  name: '',
  description: '',
  type: 'text-generation',
  version: '1.0.0',
  endpoint: '',
  isPublic: true,
  pricing: {
    type: 'free',
    pricePerCall: 0,
  },
});

// 模型类型选项
const modelTypes = [
  { label: '文本生成', value: 'text-generation' },
  { label: '图像生成', value: 'image-generation' },
  { label: '语音合成', value: 'text-to-speech' },
  { label: '语音识别', value: 'speech-to-text' },
  { label: '图像识别', value: 'image-recognition' },
  { label: '翻译', value: 'translation' },
  { label: '其他', value: 'other' },
];

// 加载我提供的模型
const loadProvidedModels = async () => {
  modelsLoading.value = true;
  try {
    const response = await requestClient.get('/user/provided-models');
    providedModels.value = response.models || [];
  } catch (error) {
    console.error('加载提供的模型失败:', error);
    message.error('加载模型列表失败');
  } finally {
    modelsLoading.value = false;
  }
};

// 添加新模型
const addModel = async () => {
  if (!newModelForm.name.trim()) {
    message.error('请输入模型名称');
    return;
  }
  if (!newModelForm.endpoint.trim()) {
    message.error('请输入模型接口地址');
    return;
  }

  isAddingModel.value = true;
  try {
    const response = await requestClient.post('/user/provided-models', newModelForm);
    if (response.model) {
      message.success('模型添加成功');
      loadProvidedModels();
      closeAddModelModal();
    }
  } catch (error) {
    console.error('添加模型失败:', error);
    message.error('添加模型失败');
  } finally {
    isAddingModel.value = false;
  }
};

// 删除模型
const deleteModel = async (modelId: string) => {
  Modal.confirm({
    title: '确认删除',
    content: '删除后模型将无法被其他用户使用，确定要删除吗？',
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        await requestClient.delete(`/user/provided-models/${modelId}`);
        message.success('模型已删除');
        loadProvidedModels();
      } catch (error) {
        console.error('删除模型失败:', error);
        message.error('删除模型失败');
      }
    },
  });
};

// 切换模型状态
const toggleModelStatus = async (modelId: string, isActive: boolean) => {
  try {
    await requestClient.patch(`/user/provided-models/${modelId}`, {
      isActive: !isActive,
    });
    message.success(isActive ? '模型已下线' : '模型已上线');
    loadProvidedModels();
  } catch (error) {
    console.error('切换模型状态失败:', error);
    message.error('操作失败');
  }
};

// 显示模型详情
const showModelDetail = (model: any) => {
  selectedModel.value = model;
  showModelDetailModal.value = true;
};

// 关闭模型详情对话框
const closeModelDetailModal = () => {
  showModelDetailModal.value = false;
  selectedModel.value = null;
};

// 显示添加模型对话框
const showAddModel = () => {
  showAddModelModal.value = true;
  // 重置表单
  Object.assign(newModelForm, {
    name: '',
    description: '',
    type: 'text-generation',
    version: '1.0.0',
    endpoint: '',
    isPublic: true,
    pricing: {
      type: 'free',
      pricePerCall: 0,
    },
  });
};

// 关闭添加模型对话框
const closeAddModelModal = () => {
  showAddModelModal.value = false;
};

// 格式化日期
const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString('zh-CN');
};

// 获取模型状态样式
const getStatusClass = (isActive: boolean) => {
  return isActive
    ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
    : 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400';
};

// 页面挂载时加载数据
onMounted(() => {
  loadProvidedModels();
});

// 暴露刷新方法给父组件
defineExpose({
  refreshData: loadProvidedModels,
});
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h3 class="text-lg font-semibold text-[var(--text-primary)]">我提供的模型</h3>
      <div class="flex items-center space-x-3">
        <button
          class="inline-flex items-center rounded-lg bg-[var(--color-neutral-700)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--color-neutral-600)] focus:outline-none focus:ring-2 focus:ring-gray-500"
          @click="loadProvidedModels"
        >
          <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          刷新
        </button>
        <button
          class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          @click="showAddModel"
        >
          <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          添加模型
        </button>
      </div>
    </div>

    <!-- 模型网格 -->
    <div v-if="!modelsLoading && providedModels.length > 0" class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="model in providedModels"
        :key="model.id"
        class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6 hover:border-blue-500 transition-colors"
      >
        <div class="flex items-start justify-between mb-4">
          <div class="flex-1">
            <div class="flex items-center mb-2">
              <h4 class="text-lg font-semibold text-[var(--text-primary)]">{{ model.name }}</h4>
              <span
                class="ml-2 inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
                :class="getStatusClass(model.isActive)"
              >
                {{ model.isActive ? '运行中' : '已停止' }}
              </span>
            </div>
            <p class="text-sm text-[var(--text-secondary)] mb-3">{{ model.description }}</p>
            
            <div class="space-y-2">
              <div class="flex items-center text-sm">
                <span class="text-[var(--text-secondary)] w-16">类型:</span>
                <span class="text-[var(--text-primary)]">{{ modelTypes.find(t => t.value === model.type)?.label || model.type }}</span>
              </div>
              <div class="flex items-center text-sm">
                <span class="text-[var(--text-secondary)] w-16">版本:</span>
                <span class="text-[var(--text-primary)]">{{ model.version }}</span>
              </div>
              <div class="flex items-center text-sm">
                <span class="text-[var(--text-secondary)] w-16">调用量:</span>
                <span class="text-[var(--text-primary)]">{{ model.totalCalls?.toLocaleString() || 0 }}</span>
              </div>
              <div class="flex items-center text-sm">
                <span class="text-[var(--text-secondary)] w-16">收入:</span>
                <span class="text-[var(--text-primary)]">¥{{ model.totalEarnings?.toFixed(2) || '0.00' }}</span>
              </div>
            </div>
          </div>
          
          <div class="ml-4">
            <div class="flex items-center justify-center w-12 h-12 rounded-lg bg-green-100 dark:bg-green-900/20">
              <svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/>
              </svg>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex items-center justify-between pt-4 border-t border-[var(--border-color)]">
          <button
            class="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
            @click="showModelDetail(model)"
          >
            查看详情
          </button>
          <div class="flex items-center space-x-2">
            <button
              class="text-sm hover:text-blue-800 dark:hover:text-blue-300"
              :class="model.isActive ? 'text-orange-600 dark:text-orange-400' : 'text-green-600 dark:text-green-400'"
              @click="toggleModelStatus(model.id, model.isActive)"
            >
              {{ model.isActive ? '下线' : '上线' }}
            </button>
            <button
              class="text-sm text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
              @click="deleteModel(model.id)"
            >
              删除
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!modelsLoading && providedModels.length === 0" class="text-center py-12">
      <svg class="mx-auto h-12 w-12 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/>
      </svg>
      <h3 class="mt-2 text-sm font-medium text-[var(--text-primary)]">暂无提供的模型</h3>
      <p class="mt-1 text-sm text-[var(--text-secondary)]">添加您的第一个模型开始赚钱</p>
      <button
        class="mt-4 inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        @click="showAddModel"
      >
        添加模型
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-else class="text-center py-12">
      <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <p class="mt-2 text-sm text-[var(--text-secondary)]">加载中...</p>
    </div>

    <!-- 模型详情对话框 -->
    <div
      v-if="showModelDetailModal && selectedModel"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      @click="closeModelDetailModal"
    >
      <div
        class="relative mx-4 w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-gray-800"
        @click.stop
      >
        <!-- 对话框头部 -->
        <div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <h3 class="text-xl font-semibold text-gray-900 dark:text-white">{{ selectedModel.name }}</h3>
          <button
            class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
            @click="closeModelDetailModal"
          >
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        
        <!-- 对话框内容 -->
        <div class="p-6">
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">描述</label>
              <p class="text-sm text-gray-600 dark:text-gray-400">{{ selectedModel.description || '无描述' }}</p>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">类型</label>
                <p class="text-sm text-gray-600 dark:text-gray-400">{{ modelTypes.find(t => t.value === selectedModel.type)?.label || selectedModel.type }}</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">版本</label>
                <p class="text-sm text-gray-600 dark:text-gray-400">{{ selectedModel.version }}</p>
              </div>
            </div>
            
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">接口地址</label>
              <code class="block bg-gray-100 dark:bg-gray-700 px-3 py-2 rounded text-sm font-mono">{{ selectedModel.endpoint }}</code>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">总调用量</label>
                <p class="text-sm text-gray-600 dark:text-gray-400">{{ selectedModel.totalCalls?.toLocaleString() || 0 }}</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">总收入</label>
                <p class="text-sm text-gray-600 dark:text-gray-400">¥{{ selectedModel.totalEarnings?.toFixed(2) || '0.00' }}</p>
              </div>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">创建时间</label>
                <p class="text-sm text-gray-600 dark:text-gray-400">{{ formatDate(selectedModel.createdAt) }}</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">最后调用</label>
                <p class="text-sm text-gray-600 dark:text-gray-400">{{ selectedModel.lastCalled ? formatDate(selectedModel.lastCalled) : '未调用' }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加模型对话框 -->
    <div
      v-if="showAddModelModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      @click="closeAddModelModal"
    >
      <div
        class="relative mx-4 w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-gray-800"
        @click.stop
      >
        <!-- 对话框头部 -->
        <div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <h3 class="text-xl font-semibold text-gray-900 dark:text-white">添加模型</h3>
          <button
            class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
            @click="closeAddModelModal"
          >
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        
        <!-- 对话框内容 -->
        <div class="p-6">
          <form @submit.prevent="addModel">
            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  模型名称 <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="newModelForm.name"
                  type="text"
                  placeholder="请输入模型名称"
                  class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400"
                  required
                />
              </div>
              
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  描述
                </label>
                <textarea
                  v-model="newModelForm.description"
                  placeholder="请输入模型描述"
                  rows="3"
                  class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400"
                ></textarea>
              </div>
              
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    模型类型 <span class="text-red-500">*</span>
                  </label>
                  <select
                    v-model="newModelForm.type"
                    class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                    required
                  >
                    <option v-for="type in modelTypes" :key="type.value" :value="type.value">
                      {{ type.label }}
                    </option>
                  </select>
                </div>
                
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    版本
                  </label>
                  <input
                    v-model="newModelForm.version"
                    type="text"
                    placeholder="1.0.0"
                    class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400"
                  />
                </div>
              </div>
              
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  接口地址 <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="newModelForm.endpoint"
                  type="url"
                  placeholder="https://api.example.com/v1/model"
                  class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400"
                  required
                />
              </div>
              
              <div class="flex items-center">
                <input
                  v-model="newModelForm.isPublic"
                  type="checkbox"
                  class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <label class="ml-2 text-sm text-gray-700 dark:text-gray-300">
                  公开模型（允许其他用户使用）
                </label>
              </div>
            </div>
            
            <div class="flex justify-end space-x-3 mt-6">
              <button
                type="button"
                class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-500 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
                @click="closeAddModelModal"
              >
                取消
              </button>
              <button
                type="submit"
                class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                :disabled="isAddingModel"
              >
                {{ isAddingModel ? '添加中...' : '添加' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>
