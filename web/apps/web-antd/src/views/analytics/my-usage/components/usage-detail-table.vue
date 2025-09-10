<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue';
import { requestClient } from '#/api/request';

interface TokenUsageRecord {
  ID: number;
  RequestID: string;
  UserID: string;
  APIKey: string;
  ClientID: string;
  ClientIP: string;
  Model: string;
  InputTokens: number;
  OutputTokens: number;
  TotalTokens: number;
  Timestamp: string;
  PPM?: number; // 每百万Token价格
}

const loading = ref(false);
const usageRecords = ref<TokenUsageRecord[]>([]);
const currentPage = ref(1);
const pageSize = ref(15);

// 默认PPM值（每百万Token价格）
const defaultPPM = 1000.00;

// 计算单次调用消费
const calculateSingleCallConsumption = (record: TokenUsageRecord) => {
  const ppm = record.PPM || defaultPPM;
  return (ppm / 1000000) * record.TotalTokens;
};

// 分页数据
const paginatedRecords = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return usageRecords.value.slice(start, end);
});

// 总页数
const totalPages = computed(() => {
  return Math.ceil(usageRecords.value.length / pageSize.value);
});

// 获取使用数据
const fetchUsageData = async () => {
  loading.value = true;
  try {
    const response = await requestClient.get('/user/token-usage');
    
    if (response && response.data && Array.isArray(response.data)) {
      usageRecords.value = response.data;
    } else if (Array.isArray(response)) {
      usageRecords.value = response;
    } else {
      usageRecords.value = [];
    }
  } catch (error) {
    console.error('获取使用数据失败:', error);
    usageRecords.value = [];
  } finally {
    loading.value = false;
  }
};

// 格式化时间
const formatTime = (timestamp: string) => {
  return new Date(timestamp).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
};

// 切换页面
const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
  }
};

onMounted(() => {
  fetchUsageData();
});
</script>

<template>
  <div class="space-y-4">
    <!-- 表格 -->
    <div class="overflow-x-auto">
      <table class="w-full border-collapse border border-[var(--border-color)]">
        <thead>
          <tr class="bg-[var(--bg-color-secondary)]">
            <th class="border border-[var(--border-color)] px-3 py-2 text-left text-sm font-medium text-[var(--text-primary)]">
              请求ID
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-left text-sm font-medium text-[var(--text-primary)]">
              模型
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              输入Token
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              输出Token
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              总Token
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              PPM
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              消费金额
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-left text-sm font-medium text-[var(--text-primary)]">
              时间
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="8" class="border border-[var(--border-color)] px-3 py-4 text-center text-[var(--text-secondary)]">
              <div class="flex items-center justify-center">
                <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500"></div>
                <span class="ml-2">加载中...</span>
              </div>
            </td>
          </tr>
          <tr v-else-if="paginatedRecords.length === 0">
            <td colspan="8" class="border border-[var(--border-color)] px-3 py-4 text-center text-[var(--text-secondary)]">
              暂无数据
            </td>
          </tr>
          <tr v-else v-for="record in paginatedRecords" :key="record.ID" class="hover:bg-[var(--bg-color-secondary)]">
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)]">
              <div class="max-w-[120px] truncate" :title="record.RequestID">
                {{ record.RequestID.substring(0, 8) }}...
              </div>
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)]">
              {{ record.Model }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)] text-right">
              {{ record.InputTokens.toLocaleString() }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)] text-right">
              {{ record.OutputTokens.toLocaleString() }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)] text-right font-medium">
              {{ record.TotalTokens.toLocaleString() }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)] text-right">
              {{ (record.PPM || defaultPPM).toFixed(2) }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-green-600 dark:text-green-400 text-right font-medium">
              ¥{{ calculateSingleCallConsumption(record).toFixed(6) }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-secondary)]">
              {{ formatTime(record.Timestamp) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页信息 -->
    <div class="flex items-center justify-between">
      <div class="text-sm text-[var(--text-secondary)]">
        显示第 {{ (currentPage - 1) * pageSize + 1 }} 到 {{ Math.min(currentPage * pageSize, usageRecords.length) }} 项，共 {{ usageRecords.length }} 项
        <span class="ml-2 text-[var(--text-tertiary)]">
          (消费计算：PPM优先取API返回值，默认{{ defaultPPM.toFixed(2) }}，公式为 (PPM/1,000,000) × TotalTokens)
        </span>
      </div>
      
      <div class="flex items-center space-x-2">
        <button 
          @click="goToPage(currentPage - 1)"
          :disabled="currentPage <= 1"
          class="px-3 py-1 text-sm border border-[var(--border-color)] rounded hover:bg-[var(--bg-color-secondary)] disabled:opacity-50 disabled:cursor-not-allowed"
        >
          上一页
        </button>
        
        <span class="text-sm text-[var(--text-secondary)]">
          {{ currentPage }} / {{ totalPages }}
        </span>
        
        <button 
          @click="goToPage(currentPage + 1)"
          :disabled="currentPage >= totalPages"
          class="px-3 py-1 text-sm border border-[var(--border-color)] rounded hover:bg-[var(--bg-color-secondary)] disabled:opacity-50 disabled:cursor-not-allowed"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>
