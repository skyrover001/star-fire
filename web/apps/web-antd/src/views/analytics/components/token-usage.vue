<template>
  <div class="space-y-6">
    <!-- Token 使用情况概览卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
      <!-- 今日Token使用量 -->
      <div class="bg-[var(--content-bg)] rounded-2xl shadow-lg border border-[var(--border-color)] p-6 hover:shadow-xl transition-all duration-300">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-14 w-14 rounded-xl bg-gradient-to-br from-blue-500/20 to-blue-600/30 border border-blue-500/30">
              <SvgBellIcon class="h-7 w-7 text-blue-400" />
            </div>
          </div>
          <div class="ml-4 flex-1">
            <p class="text-sm font-semibold text-[var(--text-secondary)] mb-1">{{ $t('business.analytics.todayUsage') }}</p>
            <p class="text-2xl font-bold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else class="text-blue-400">{{ formatNumber(tokenUsage.todayUsage) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] font-medium">{{ $t('business.analytics.todayConsumed') }}</p>
          </div>
        </div>
      </div>

      <!-- 本月Token使用量 -->
      <div class="bg-[var(--content-bg)] rounded-2xl shadow-lg border border-[var(--border-color)] p-6 hover:shadow-xl transition-all duration-300">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-14 w-14 rounded-xl bg-gradient-to-br from-green-500/20 to-green-600/30 border border-green-500/30">
              <SvgCardIcon class="h-7 w-7 text-green-400" />
            </div>
          </div>
          <div class="ml-4 flex-1">
            <p class="text-sm font-semibold text-[var(--text-secondary)] mb-1">{{ $t('business.analytics.monthlyUsage') }}</p>
            <p class="text-2xl font-bold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else class="text-green-400">{{ formatNumber(tokenUsage.monthlyUsage) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] font-medium">{{ $t('business.analytics.cumulativeConsumed') }}</p>
          </div>
        </div>
      </div>

      <!-- 总Token使用量 -->
      <div class="bg-[var(--content-bg)] rounded-2xl shadow-lg border border-[var(--border-color)] p-6 hover:shadow-xl transition-all duration-300">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-14 w-14 rounded-xl bg-gradient-to-br from-purple-500/20 to-purple-600/30 border border-purple-500/30">
              <SvgCakeIcon class="h-7 w-7 text-purple-400" />
            </div>
          </div>
          <div class="ml-4 flex-1">
            <p class="text-sm font-semibold text-[var(--text-secondary)] mb-1">{{ $t('business.analytics.totalUsage') }}</p>
            <p class="text-2xl font-bold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-20"></span>
              <span v-else class="text-purple-400">{{ formatNumber(tokenUsage.totalUsage) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] font-medium">{{ $t('business.analytics.historicalTotal') }}</p>
          </div>
        </div>
      </div>

      <!-- 平均每日使用量 -->
      <div class="bg-[var(--content-bg)] rounded-2xl shadow-lg border border-[var(--border-color)] p-6 hover:shadow-xl transition-all duration-300">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-14 w-14 rounded-xl bg-gradient-to-br from-orange-500/20 to-orange-600/30 border border-orange-500/30">
              <SvgDownloadIcon class="h-7 w-7 text-orange-400" />
            </div>
          </div>
          <div class="ml-4 flex-1">
            <p class="text-sm font-semibold text-[var(--text-secondary)] mb-1">{{ $t('business.analytics.averageDailyUsage') }}</p>
            <p class="text-2xl font-bold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else class="text-orange-400">{{ formatNumber(tokenUsage.averageDailyUsage) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] font-medium">{{ $t('business.analytics.monthlyAverage') }}</p>
          </div>
        </div>
      </div>
    </div>



    <!-- 按模型统计 -->
    <div class="bg-[var(--content-bg)] rounded-2xl shadow-lg border border-[var(--border-color)] overflow-hidden">
      <div class="px-8 py-6 border-b border-[var(--border-color)] bg-gradient-to-r from-[var(--bg-color-secondary)] to-[var(--content-bg)]">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <div class="w-3 h-3 rounded-full bg-[var(--primary-color)]"></div>
            <h3 class="text-xl font-bold text-[var(--text-primary)]">{{ $t('business.analytics.byModel') }}</h3>
          </div>
          <div class="flex items-center space-x-2">
            <button
              v-for="period in timePeriods"
              :key="period.value"
              class="px-4 py-2 text-sm font-semibold rounded-lg transition-all duration-200 transform hover:scale-105"
              :class="selectedPeriod === period.value 
                ? 'bg-[var(--primary-color)] text-white shadow-lg' 
                : 'bg-[var(--content-bg)] text-[var(--text-secondary)] border border-[var(--border-color)] hover:bg-[var(--bg-color-secondary)] hover:text-[var(--text-primary)]'"
              @click="selectedPeriod = period.value; fetchTokenUsage()"
            >
              {{ period.label }}
            </button>
          </div>
        </div>
      </div>
      <div class="overflow-x-auto">
        <!-- 模型统计表格 -->
        <table class="w-full">
          <thead class="bg-[var(--bg-color-secondary)] border-b border-[var(--border-color)]">
            <tr>
              <th class="px-6 py-4 text-left text-xs font-bold text-[var(--text-primary)] uppercase tracking-wider">{{ $t('business.analytics.modelName') }}</th>
              <th class="px-6 py-4 text-left text-xs font-bold text-[var(--text-primary)] uppercase tracking-wider">{{ $t('business.analytics.requestCount') }}</th>
              <th class="px-6 py-4 text-left text-xs font-bold text-[var(--text-primary)] uppercase tracking-wider">{{ $t('business.analytics.inputToken') }}</th>
              <th class="px-6 py-4 text-left text-xs font-bold text-[var(--text-primary)] uppercase tracking-wider">{{ $t('business.analytics.outputToken') }}</th>
              <th class="px-6 py-4 text-left text-xs font-bold text-[var(--text-primary)] uppercase tracking-wider">{{ $t('business.analytics.totalToken') }}</th>
              <th class="px-6 py-4 text-left text-xs font-bold text-[var(--text-primary)] uppercase tracking-wider">{{ $t('business.analytics.usageRatio') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border-color)] bg-[var(--content-bg)]">
            <tr v-if="loading" v-for="i in 5" :key="i">
              <td v-for="j in 6" :key="j" class="px-6 py-4">
                <div class="animate-pulse bg-[var(--bg-color-secondary)] rounded h-4 w-16"></div>
              </td>
            </tr>
            <tr v-else-if="modelStats.length === 0">
              <td colspan="6" class="px-6 py-12 text-center text-[var(--text-secondary)]">
                <div class="flex flex-col items-center">
                  <div class="w-12 h-12 bg-[var(--bg-color-secondary)] rounded-full flex items-center justify-center mb-3">
                    <svg class="w-6 h-6 text-[var(--text-tertiary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path>
                    </svg>
                  </div>
                  <p class="text-[var(--text-secondary)] font-semibold">{{ $t('business.analytics.noModelUsage') }}</p>
                </div>
              </td>
            </tr>
            <tr v-else v-for="(stat, index) in modelStats" :key="stat.model" class="hover:bg-[var(--bg-color-secondary)] transition-colors">
              <td class="px-6 py-4">
                <div class="flex items-center">
                  <div class="w-2 h-2 rounded-full mr-3" :class="getModelColor(index)"></div>
                  <span class="text-sm font-bold text-[var(--text-primary)]">{{ stat.model }}</span>
                </div>
              </td>
              <td class="px-6 py-4 text-sm font-semibold text-[var(--text-primary)]">{{ formatNumber(stat.requestCount) }}</td>
              <td class="px-6 py-4 text-sm font-semibold text-[var(--text-secondary)]">{{ formatNumber(stat.inputTokens) }}</td>
              <td class="px-6 py-4 text-sm font-semibold text-[var(--text-secondary)]">{{ formatNumber(stat.outputTokens) }}</td>
              <td class="px-6 py-4 text-sm font-bold text-[var(--primary-color)]">{{ formatNumber(stat.totalTokens) }}</td>
              <td class="px-6 py-4">
                <div class="flex items-center">
                  <div class="w-16 bg-[var(--bg-color-secondary)] rounded-full h-2 mr-3">
                    <div 
                      class="h-2 rounded-full" 
                      :class="getModelColor(index)"
                      :style="`width: ${stat.percentage}%`"
                    ></div>
                  </div>
                  <span class="text-sm font-bold text-[var(--text-primary)]">{{ stat.percentage.toFixed(1) }}%</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, nextTick, onUnmounted, watch, inject } from 'vue';
import type { Ref } from 'vue';
import { message } from 'ant-design-vue';
import { requestClient } from '#/api/request';
import { $t } from '#/locales';
import {
  SvgBellIcon,
  SvgCakeIcon,
  SvgCardIcon,
  SvgDownloadIcon,
} from '@vben/icons';

// 接口类型定义
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
}

interface UsageStats {
  total_calls: number;
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  total_tokens: number;
  total_cost: number;
  client_count: number;
  model_count: number;
}

interface ModelUsageStat {
  model: string;
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  total_tokens: number;
  total_cost: number;
  calls: number;
  client_count: number;
  last_used: string;
}

// 从父组件注入聚合统计数据（替代全量 usageRecords）
const usageTotalStats = inject<Ref<UsageStats>>('usageTotalStats', ref<UsageStats>({
  total_calls: 0, input_tokens: 0, output_tokens: 0, cached_tokens: 0,
  total_tokens: 0, total_cost: 0, client_count: 0, model_count: 0,
}));
const usageStats = inject<Ref<UsageStats>>('usageStats', ref<UsageStats>({
  total_calls: 0, input_tokens: 0, output_tokens: 0, cached_tokens: 0,
  total_tokens: 0, total_cost: 0, client_count: 0, model_count: 0,
}));
const usageLoading = inject<Ref<boolean>>('usageLoading', ref(false));



// 响应式数据
const loading = ref(false);
const selectedPeriod = ref('30d');

// 今日使用量（来自 /usage/stats 按天接口）
const todayUsage = ref(0);

const tokenUsage = reactive({
  todayUsage: 0,
  monthlyUsage: 0,
  totalUsage: 0,
  averageDailyUsage: 0,
});



// 模型统计数据
const modelStats = ref<Array<{
  model: string;
  requestCount: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  percentage: number;
}>>([]);

// 时间周期选项
const timePeriods = [
  { label: $t('business.analytics.period7d'), value: '7d' },
  { label: $t('business.analytics.period30d'), value: '30d' },
  { label: $t('business.analytics.period90d'), value: '90d' },
];

// 获取模型颜色类
const getModelColor = (index: number): string => {
  const colors = [
    'bg-blue-500',
    'bg-green-500', 
    'bg-purple-500',
    'bg-orange-500',
    'bg-red-500',
    'bg-yellow-500',
    'bg-indigo-500',
    'bg-pink-500'
  ];
  return colors[index % colors.length];
};
const formatNumber = (num: number): string => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toLocaleString();
};



// 格式化日期
const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
};

// 格式化日期时间
const formatDateTime = (dateTimeStr: string): string => {
  const date = new Date(dateTimeStr);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
};





const resetTokenUsageStats = () => {
  Object.assign(tokenUsage, {
    todayUsage: 0,
    monthlyUsage: 0,
    totalUsage: 0,
    averageDailyUsage: 0,
  });
  modelStats.value = [];
};

// 从注入的聚合数据计算统计（替代原 fetchTokenUsage 拉全量记录）
const calculateStatisticsFromStats = () => {
  const total = usageTotalStats.value.total_tokens;
  const monthly = usageStats.value.total_tokens; // 30天窗口近似月度
  const today = todayUsage.value; // 今日数据来自按天接口
  const daysInMonth = 30;
  const averageDailyUsage = Math.round(monthly / daysInMonth);

  Object.assign(tokenUsage, {
    todayUsage: today,
    monthlyUsage: monthly,
    totalUsage: total,
    averageDailyUsage,
  });
};

// 获取今日使用统计（start_date=end_date=今天）
const fetchTodayUsage = async () => {
  try {
    const todayStr = new Date().toISOString().slice(0, 10);
    const response = await requestClient.get('/user/usage/stats', {
      params: { start_date: todayStr, end_date: todayStr },
    });
    if (response && typeof response === 'object' && response.total_tokens !== undefined) {
      todayUsage.value = response.total_tokens;
    } else {
      todayUsage.value = 0;
    }
  } catch (error) {
    console.error('获取今日使用统计失败:', error);
    todayUsage.value = 0;
  }
  // 将最新今日数据同步到展示用的 tokenUsage
  calculateStatisticsFromStats();
};

// 获取模型统计（调 /usage/models 接口，替代客户端聚合）
const fetchModelStats = async () => {
  try {
    const response = await requestClient.get('/user/usage/models');
    if (response && Array.isArray(response.data)) {
      const stats = response.data as ModelUsageStat[];
      const totalTokensAll = stats.reduce((sum, item) => sum + item.total_tokens, 0);
      modelStats.value = stats.map(item => ({
        model: item.model,
        requestCount: item.calls,
        inputTokens: item.input_tokens,
        outputTokens: item.output_tokens,
        totalTokens: item.total_tokens,
        percentage: totalTokensAll > 0 ? (item.total_tokens / totalTokensAll) * 100 : 0,
      })).sort((a, b) => b.totalTokens - a.totalTokens);
    } else {
      modelStats.value = [];
    }
  } catch (error) {
    console.error('获取模型统计失败:', error);
    modelStats.value = [];
  }
};

// 监听注入数据变化，重新计算
watch([usageTotalStats, usageStats], () => {
  calculateStatisticsFromStats();
}, { deep: true });

// 导出刷新方法
const refreshData = () => {
  calculateStatisticsFromStats();
  fetchTodayUsage();
  fetchModelStats();
};

// 组件挂载时加载数据（用注入的聚合数据 + 拉取模型统计）
onMounted(() => {
  calculateStatisticsFromStats();
  fetchTodayUsage();
  fetchModelStats();
});

// 组件卸载时清理资源
onUnmounted(() => {
  // 清理资源
});

// 导出方法供父组件调用
defineExpose({
  refreshData,
});
</script>
