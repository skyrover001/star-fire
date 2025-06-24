<script lang="ts" setup>
import { ref, onMounted } from 'vue';

interface ContributionModel {
  id: string;
  name: string;
  type: string;
  uploadDate: string;
  totalDownloads: number;
  totalCalls: number;
  activeUsers: number;
  rating: number;
  status: 'online' | 'offline' | 'maintenance';
}

const contributionModels = ref<ContributionModel[]>([]);
const loading = ref(false);

// 模拟数据
const mockData: ContributionModel[] = [
  {
    id: '1',
    name: 'my-custom-llama:7b',
    type: 'ollama',
    uploadDate: '2025-06-15',
    totalDownloads: 456,
    totalCalls: 2340,
    activeUsers: 89,
    rating: 4.8,
    status: 'online',
  },
  {
    id: '2',
    name: 'fine-tuned-qwen:1.8b',
    type: 'ollama',
    uploadDate: '2025-06-10',
    totalDownloads: 234,
    totalCalls: 1560,
    activeUsers: 45,
    rating: 4.6,
    status: 'online',
  },
  {
    id: '3',
    name: 'specialized-coder:3b',
    type: 'ollama',
    uploadDate: '2025-06-05',
    totalDownloads: 123,
    totalCalls: 890,
    activeUsers: 23,
    rating: 4.4,
    status: 'maintenance',
  },
];

const fetchContributionModels = async () => {
  loading.value = true;
  try {
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 500));
    contributionModels.value = mockData;
  } catch (error) {
    console.error('获取贡献模型数据失败:', error);
  } finally {
    loading.value = false;
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case 'online':
      return 'bg-green-100 text-green-800';
    case 'offline':
      return 'bg-red-100 text-red-800';
    case 'maintenance':
      return 'bg-yellow-100 text-yellow-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
};

const getStatusText = (status: string) => {
  switch (status) {
    case 'online':
      return '在线';
    case 'offline':
      return '离线';
    case 'maintenance':
      return '维护中';
    default:
      return '未知';
  }
};

const formatNumber = (num: number) => {
  if (num >= 1000) {
    return `${(num / 1000).toFixed(1)}K`;
  }
  return num.toString();
};

const renderStars = (rating: number) => {
  const fullStars = Math.floor(rating);
  const hasHalfStar = rating % 1 !== 0;
  const emptyStars = 5 - fullStars - (hasHalfStar ? 1 : 0);
  
  return {
    full: fullStars,
    half: hasHalfStar,
    empty: emptyStars
  };
};

onMounted(() => {
  fetchContributionModels();
});
</script>

<template>
  <div class="h-80 overflow-y-auto">
    <div v-if="loading" class="flex items-center justify-center h-full">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>
    
    <div v-else class="space-y-3">
      <div
        v-for="model in contributionModels"
        :key="model.id"
        class="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-md transition-shadow"
      >
        <div class="flex items-start justify-between mb-2">
          <div>
            <h4 class="font-semibold text-gray-900">{{ model.name }}</h4>
            <div class="flex items-center space-x-2 mt-1">
              <span class="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">{{ model.type.toUpperCase() }}</span>
              <span :class="getStatusColor(model.status)" class="text-xs px-2 py-1 rounded">
                {{ getStatusText(model.status) }}
              </span>
            </div>
          </div>
          <div class="text-right">
            <div class="text-sm font-medium text-gray-900">{{ formatNumber(model.totalDownloads) }} 下载</div>
            <div class="text-xs text-gray-500">{{ formatNumber(model.totalCalls) }} 调用</div>
          </div>
        </div>
        
        <div class="grid grid-cols-2 gap-4 mt-3 text-sm">
          <div>
            <span class="text-gray-500">上传时间:</span>
            <div class="font-medium">{{ model.uploadDate }}</div>
          </div>
          <div>
            <span class="text-gray-500">活跃用户:</span>
            <div class="font-medium">{{ model.activeUsers }} 人</div>
          </div>
        </div>
        
        <div class="mt-3 flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <span class="text-xs text-gray-500">评分:</span>
            <div class="flex items-center space-x-1">
              <template v-for="i in renderStars(model.rating).full" :key="`full-${i}`">
                <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                </svg>
              </template>
              <template v-if="renderStars(model.rating).half">
                <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" clip-path="polygon(0 0, 50% 0, 50% 100%, 0 100%)"/>
                </svg>
              </template>
              <template v-for="i in renderStars(model.rating).empty" :key="`empty-${i}`">
                <svg class="w-3 h-3 text-gray-300" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                </svg>
              </template>
              <span class="text-xs text-gray-600 ml-1">{{ model.rating }}</span>
            </div>
          </div>
          
          <div class="flex space-x-2">
            <button class="text-xs text-blue-600 hover:text-blue-800 transition-colors">
              管理
            </button>
            <button class="text-xs text-gray-500 hover:text-gray-700 transition-colors">
              统计
            </button>
          </div>
        </div>
      </div>
      
      <div v-if="contributionModels.length === 0" class="text-center py-8 text-gray-500">
        暂无贡献记录
      </div>
    </div>
  </div>
</template>
