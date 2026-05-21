<script lang="ts" setup>
import { ref, computed, inject, watch } from 'vue';
import type { Ref } from 'vue';

interface TokenUsageRecord {
  ID: number;
  RequestID: string;
  UserID: string;
  APIKey: string;
  ClientID: string;
  ClientIP: string;
  Model: string;
  IPPM: number;
  OPPM: number;
  CIPPM: number;
  InputTokens: number;
  OutputTokens: number;
  CachedTokens: number;
  TotalTokens: number;
  Timestamp: string;
}

// 计算收益明细
const calcUncachedInputCost = (r: TokenUsageRecord) => {
  const cached = r.CachedTokens || 0;
  return ((r.InputTokens - cached) * (r.IPPM || 0)) / 1000000;
};
const calcCachedInputCost = (r: TokenUsageRecord) => {
  return ((r.CachedTokens || 0) * (r.CIPPM || 0)) / 1000000;
};
const calcOutputCost = (r: TokenUsageRecord) => {
  return (r.OutputTokens * (r.OPPM || 0)) / 1000000;
};
const calcTotalCost = (r: TokenUsageRecord) => {
  return calcUncachedInputCost(r) + calcCachedInputCost(r) + calcOutputCost(r);
};

const usageRecords = inject<Ref<TokenUsageRecord[]>>(
  'usageRecords',
  ref<TokenUsageRecord[]>([]),
);

const loading = inject<Ref<boolean>>('usageLoading', ref(false));
const currentPage = ref(1);
const pageSize = ref(15);

// 分页数据
const paginatedRecords = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return usageRecords.value.slice(start, end);
});

// 总页数
const totalPages = computed(() => {
  const total = Math.ceil(usageRecords.value.length / pageSize.value);
  return total === 0 ? 1 : total;
});

const totalItems = computed(() => usageRecords.value.length);

const rangeStart = computed(() => {
  if (totalItems.value === 0) {
    return 0;
  }
  return (currentPage.value - 1) * pageSize.value + 1;
});

const rangeEnd = computed(() => {
  if (totalItems.value === 0) {
    return 0;
  }
  return Math.min(currentPage.value * pageSize.value, totalItems.value);
});

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

watch(
  () => usageRecords.value.length,
  () => {
    if (currentPage.value > totalPages.value) {
      currentPage.value = totalPages.value;
    }
    if (usageRecords.value.length === 0) {
      currentPage.value = 1;
    }
  },
);
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
              缓存命中
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              输出Token
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              总Token
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              未命中输入费用
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              缓存命中费用
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              输出费用
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-right text-sm font-medium text-[var(--text-primary)]">
              总费用
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-left text-sm font-medium text-[var(--text-primary)]">
              客户端
            </th>
            <th class="border border-[var(--border-color)] px-3 py-2 text-left text-sm font-medium text-[var(--text-primary)]">
              时间
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="12" class="border border-[var(--border-color)] px-3 py-4 text-center text-[var(--text-secondary)]">
              <div class="flex items-center justify-center">
                <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500"></div>
                <span class="ml-2">加载中...</span>
              </div>
            </td>
          </tr>
          <tr v-else-if="paginatedRecords.length === 0">
            <td colspan="12" class="border border-[var(--border-color)] px-3 py-4 text-center text-[var(--text-secondary)]">
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
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-right" :class="(record.CachedTokens || 0) > 0 ? 'text-orange-500 font-medium' : 'text-[var(--text-primary)]'">
              {{ (record.CachedTokens || 0).toLocaleString() }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)] text-right">
              {{ record.OutputTokens.toLocaleString() }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-primary)] text-right font-medium">
              {{ record.TotalTokens.toLocaleString() }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-right text-blue-500">
              ¥{{ calcUncachedInputCost(record).toFixed(6) }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-right" :class="calcCachedInputCost(record) > 0 ? 'text-orange-500' : 'text-[var(--text-secondary)]'">
              ¥{{ calcCachedInputCost(record).toFixed(6) }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-right text-green-500">
              ¥{{ calcOutputCost(record).toFixed(6) }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-right font-semibold text-emerald-600">
              ¥{{ calcTotalCost(record).toFixed(6) }}
            </td>
            <td class="border border-[var(--border-color)] px-3 py-2 text-sm text-[var(--text-secondary)]">
              <div class="max-w-[100px] truncate" :title="record.ClientID">
                {{ record.ClientID }}
              </div>
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
        显示第 {{ rangeStart }} 到 {{ rangeEnd }} 项，共 {{ totalItems }} 项
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
