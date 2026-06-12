<script lang="ts" setup>
import type { TabOption } from '@vben/types';
import { ref, onMounted, computed, provide } from 'vue';
import { requestClient } from '#/api/request';
import { getBalanceApi } from '#/api/core/balance';
import type { BalanceInfo } from '#/api/core/balance';

import {
  AnalysisChartCard,
  AnalysisChartsTabs,
} from '@vben/common-ui';
import {
  SvgBellIcon,
  SvgCakeIcon,
  SvgCardIcon,
  SvgDownloadIcon,
} from '@vben/icons';

import AnalyticsTrends from '../components/analytics-trends.vue';
import AnalyticsVisits from '../components/analytics-visits.vue';
import TokenUsage from '../components/token-usage.vue';
// @ts-ignore
import MyUsageModels from './components/my-usage-models.vue';
// @ts-ignore
import UsageDetailTable from './components/usage-detail-table.vue';

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

const loading = ref(false);
const usageRecords = ref<TokenUsageRecord[]>([]);
const balanceInfo = ref<BalanceInfo>({ balance: 0, total_spent: 0 });
const balanceLoading = ref(false);

provide('usageRecords', usageRecords);
provide('usageLoading', loading);

const normalizeUsageRecords = (payload: unknown): TokenUsageRecord[] => {
  if (!payload) {
    return [];
  }

  if (Array.isArray(payload)) {
    return payload as TokenUsageRecord[];
  }

  if (typeof payload === 'object') {
    const body = payload as Record<string, unknown>;

    if (Array.isArray(body.data)) {
      return body.data as TokenUsageRecord[];
    }

    if (
      body.data &&
      typeof body.data === 'object' &&
      Array.isArray((body.data as Record<string, unknown>).data)
    ) {
      return (body.data as Record<string, unknown>).data as TokenUsageRecord[];
    }

    if (Array.isArray(body.records)) {
      return body.records as TokenUsageRecord[];
    }

    if (Array.isArray(body.items)) {
      return body.items as TokenUsageRecord[];
    }
  }

  return [];
};

// 计算总Token统计（包括输入和输出）
const totalTokenStats = computed(() => {
  const records = usageRecords.value;
  const totalTokens = records.reduce((sum, r) => sum + r.TotalTokens, 0);
  const totalInput = records.reduce((sum, r) => sum + r.InputTokens, 0);
  const totalOutput = records.reduce((sum, r) => sum + r.OutputTokens, 0);
  
  const today = new Date().toISOString().split('T')[0] || '';
  const todayRecords = records.filter(r => r.Timestamp && r.Timestamp.startsWith(today));
  const todayTotalTokens = todayRecords.reduce((sum, r) => sum + r.TotalTokens, 0);

  return {
    total: totalTokens,
    today: todayTotalTokens,
    totalInput,
    totalOutput,
    totalCalls: records.length,
  };
});

// 计算客户端统计
const clientStats = computed(() => {
  const records = usageRecords.value;
  const uniqueClients = new Set(records.map(r => r.ClientID));
  const uniqueApiKeys = new Set(records.map(r => r.APIKey));
  
  return {
    totalClients: uniqueClients.size,
    totalApiKeys: uniqueApiKeys.size,
  };
});

const chartTabs: TabOption[] = [
  {
    label: 'Token使用分析',
    value: 'token-usage',
  },
  {
    label: '使用趋势',
    value: 'trends',
  },
  {
    label: '调用统计',
    value: 'visits',
  },
];

// 获取Token使用数据
const fetchUsageData = async () => {
  loading.value = true;
  try {
    const response = await requestClient.get('/user/token-usage');
    const records = normalizeUsageRecords(response);
    usageRecords.value = records;
  } catch (error) {
    console.error('获取使用数据失败:', error);
    usageRecords.value = [];
  } finally {
    loading.value = false;
  }
};

// 获取余额
const fetchBalance = async () => {
  balanceLoading.value = true;
  try {
    const info = await getBalanceApi();
    balanceInfo.value = info;
  } catch {
    // ignore
  } finally {
    balanceLoading.value = false;
  }
};

onMounted(() => {
  fetchUsageData();
  fetchBalance();
});
</script>

<template>
  <div class="p-5">
    
    <!-- Token使用统计卡片 -->
    <div class="mt-5 grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
      <!-- 账户余额卡片 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-green-100 dark:bg-green-900/20">
              <svg class="h-6 w-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">账户余额</p>
            <p class="text-2xl font-semibold text-green-600">
              <span v-if="balanceLoading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-24"></span>
              <span v-else>¥{{ balanceInfo.balance?.toFixed(4) || '0.0000' }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] mt-1">可用余额</p>
          </div>
        </div>
      </div>

      <!-- 累计消费卡片 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-blue-100 dark:bg-blue-900/20">
              <svg class="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
              </svg>
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">累计消费</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="balanceLoading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-24"></span>
              <span v-else>¥{{ balanceInfo.total_spent?.toFixed(4) || '0.0000' }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] mt-1">历史消费总额</p>
          </div>
        </div>
      </div>
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-purple-100 dark:bg-purple-900/20">
              <SvgDownloadIcon class="h-6 w-6 text-purple-600 dark:text-purple-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">总Tokens</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-20"></span>
              <span v-else>{{ totalTokenStats.total.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] mt-1">累计消耗</p>
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-green-100 dark:bg-green-900/20">
              <SvgCakeIcon class="h-6 w-6 text-green-600 dark:text-green-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">今日Tokens</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>{{ totalTokenStats.today.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">今日消耗</p>
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-blue-100 dark:bg-blue-900/20">
              <SvgCardIcon class="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">输入Tokens</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>{{ totalTokenStats.totalInput.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">累计输入</p>
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-orange-100 dark:bg-orange-900/20">
              <SvgBellIcon class="h-6 w-6 text-orange-600 dark:text-orange-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">输出Tokens</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>{{ totalTokenStats.totalOutput.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">累计输出</p>
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-indigo-100 dark:bg-indigo-900/20">
              <SvgCardIcon class="h-6 w-6 text-indigo-600 dark:text-indigo-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">调用次数</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-12"></span>
              <span v-else>{{ totalTokenStats.totalCalls.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">累计调用</p>
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-emerald-100 dark:bg-emerald-900/20">
              <SvgDownloadIcon class="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">客户端数</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-12"></span>
              <span v-else>{{ clientStats.totalClients }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">独立客户端</p>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 图表标签页 -->
    <AnalysisChartsTabs :tabs="chartTabs" class="mt-5">
      <template #token-usage>
        <TokenUsage />
      </template>
      <template #trends>
        <AnalyticsTrends />
      </template>
      <template #visits>
        <AnalyticsVisits />
      </template>
    </AnalysisChartsTabs>

    <!-- 模型使用统计 -->
    <div class="mt-5">
      <AnalysisChartCard title="我使用的模型">
        <MyUsageModels />
      </AnalysisChartCard>
    </div>

    <!-- 全部使用详单 -->
    <div class="mt-5">
      <AnalysisChartCard title="全部使用详单">
        <UsageDetailTable />
      </AnalysisChartCard>
    </div>
  </div>
</template>
