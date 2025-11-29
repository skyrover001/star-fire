<script lang="ts" setup>
import type { TabOption } from '@vben/types';
import { ref, onMounted, computed, nextTick, watch, provide } from 'vue';
import { requestClient } from '#/api/request';
import { useEcharts } from '@vben/plugins/echarts';

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
  IPPM: number; // 输入Token价格
  OPPM: number; // 输出Token价格
  InputTokens: number;
  OutputTokens: number;
  TotalTokens: number;
  Timestamp: string;
}

const loading = ref(false);
const usageRecords = ref<TokenUsageRecord[]>([]);

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


// 计算单次调用消费（收益）
const calculateSingleCallConsumption = (record: TokenUsageRecord) => {
  // 收益 = (输入tokens数 * IPPM + 输出tokens数 * OPPM) / 1,000,000
  return (record.InputTokens * record.IPPM + record.OutputTokens * record.OPPM) / 1000000;
};

// 图表相关
const incomeChartRef = ref();
const { renderEcharts } = useEcharts(incomeChartRef);

// 计算总Token统计（包括输入和输出）
const totalTokenStats = computed(() => {
  const records = usageRecords.value;
  const totalTokens = records.reduce((sum, r) => sum + r.TotalTokens, 0);
  
  const today = new Date().toISOString().split('T')[0] || '';
  const todayRecords = records.filter(r => r.Timestamp && r.Timestamp.startsWith(today));
  const todayTotalTokens = todayRecords.reduce((sum, r) => sum + r.TotalTokens, 0);

  return {
    total: totalTokens,
    today: todayTotalTokens,
  };
});

// 计算按模型统计的数据
const modelStats = computed(() => {
  const records = usageRecords.value;
  console.log('modelStats计算中，records数量:', records.length);
  
  if (records.length === 0) {
    console.log('无数据，返回空数组');
    return [];
  }
  
  const modelData: { [key: string]: {
    name: string;
    totalTokens: number;
    inputTokens: number;
    outputTokens: number;
    requestCount: number;
    clientCount: number;
    clients: Set<string>;
  } } = {};
  
  records.forEach((record, index) => {
    if (!record || !record.Model) {
      console.warn(`记录${index}缺少Model字段:`, record);
      return;
    }
    
    const model = record.Model;
    
    if (!modelData[model]) {
      modelData[model] = {
        name: model,
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        requestCount: 0,
        clientCount: 0,
        clients: new Set(),
      };
    }
    
    modelData[model].totalTokens += record.TotalTokens || 0;
    modelData[model].inputTokens += record.InputTokens || 0;
    modelData[model].outputTokens += record.OutputTokens || 0;
    modelData[model].requestCount += 1;
    if (record.ClientID) {
      modelData[model].clients.add(record.ClientID);
    }
  });
  
  const result = Object.values(modelData).map(item => ({
    name: item.name,
    totalTokens: item.totalTokens,
    inputTokens: item.inputTokens,
    outputTokens: item.outputTokens,
    requestCount: item.requestCount,
    clientCount: item.clients.size,
  })).sort((a, b) => b.totalTokens - a.totalTokens);
  
  console.log('modelStats计算结果:', result);
  return result;
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

// 计算消费统计
const incomeStats = computed(() => {
  const records = usageRecords.value;
  const totalIncome = records.reduce((sum, r) => sum + calculateSingleCallConsumption(r), 0);
  
  const today = new Date().toISOString().split('T')[0] || '';
  const todayRecords = records.filter(r => r.Timestamp && r.Timestamp.startsWith(today));
  const todayIncome = todayRecords.reduce((sum, r) => sum + calculateSingleCallConsumption(r), 0);
  
  return {
    total: totalIncome,
    today: todayIncome,
  };
});

// 计算按模型消费数据（用于柱状图）
const modelIncomeData = computed(() => {
  return modelStats.value.map(model => {
    // 模拟计算该模型的总消费（基于PPM）
    const modelRecords = usageRecords.value.filter(r => r.Model === model.name);
    const totalConsumption = modelRecords.reduce((sum, r) => sum + calculateSingleCallConsumption(r), 0);
    
    return {
      name: model.name,
      income: totalConsumption,
      outputTokens: model.outputTokens,
      requestCount: model.requestCount,
    };
  }).sort((a, b) => b.income - a.income);
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
    console.log('正在获取Token使用数据...');
    const response = await requestClient.get('/user/token-usage');
    console.log('API响应:', response);

    const records = normalizeUsageRecords(response);

    if (records.length === 0) {
      console.warn('Token使用数据为空或格式不正确:', response);
    }

    usageRecords.value = records;
    console.log('获取到Token使用记录:', usageRecords.value.length, '条');
    console.log('样本数据:', usageRecords.value.slice(0, 2));
  } catch (error) {
    console.error('获取使用数据失败:', error);
    usageRecords.value = [];
  } finally {
    loading.value = false;
    console.log('最终usageRecords:', usageRecords.value.length, '条');
    // 延迟打印计算结果，确保reactive数据已更新
    setTimeout(() => {
      console.log('modelStats计算结果:', modelStats.value);
    }, 100);
  }
};

// 更新消费柱状图
const updateIncomeChart = () => {
  if (!incomeChartRef.value || modelIncomeData.value.length === 0) {
    console.log('图表组件还未挂载或无数据');
    return;
  }

  console.log('更新消费柱状图数据:', modelIncomeData.value);

  const option = {
    title: {
      text: '按模型消费统计',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'bold' as const
      }
    },
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: {
        type: 'shadow' as const
      },
      formatter: function(params: any) {
        const data = params[0];
        const modelData = modelIncomeData.value.find(m => m.name === data.name);
        return `
          <div style="padding: 8px;">
            <div style="font-weight: bold; margin-bottom: 4px;">${data.name}</div>
            <div>消费: ¥${data.value.toFixed(3)}</div>
            <div>输出Token: ${modelData?.outputTokens.toLocaleString()}</div>
            <div>调用次数: ${modelData?.requestCount}</div>
          </div>
        `;
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category' as const,
      data: modelIncomeData.value.map(item => item.name),
      axisLabel: {
        interval: 0,
        rotate: modelIncomeData.value.length > 3 ? 45 : 0,
        fontSize: 12
      }
    },
    yAxis: {
      type: 'value' as const,
      name: '消费 (¥)',
      axisLabel: {
        formatter: '¥{value}'
      }
    },
    series: [
      {
        name: '消费',
        type: 'bar' as const,
        data: modelIncomeData.value.map(item => ({
          value: item.income,
          name: item.name,
          itemStyle: {
            color: '#10B981'
          }
        })),
        markLine: {
          data: [
            { type: 'average' as const, name: '平均值' }
          ]
        }
      }
    ]
  };

  renderEcharts(option);
};

// 监听数据变化，自动更新图表
watch(modelIncomeData, () => {
  nextTick(() => {
    updateIncomeChart();
  });
}, { deep: true });

onMounted(() => {
  fetchUsageData().then(() => {
    // 数据加载完成后，延迟渲染图表
    nextTick(() => {
      setTimeout(() => {
        updateIncomeChart();
      }, 500);
    });
  });
});
</script>

<template>
  <div class="p-5">
    
    <!-- 输出Token和收益统计卡片 -->
    <div class="mt-5 grid grid-cols-1 md:grid-cols-6 gap-6">
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
            <p class="text-xs text-[var(--text-tertiary)] mt-1">累计使用</p>
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
            <p class="text-xs text-[var(--text-tertiary)]">今日使用</p>
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
            <p class="text-sm font-medium text-[var(--text-secondary)]">客户端数</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-12"></span>
              <span v-else>{{ clientStats.totalClients }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">独立客户端</p>
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
            <p class="text-sm font-medium text-[var(--text-secondary)]">API密钥数</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-12"></span>
              <span v-else>{{ clientStats.totalApiKeys }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">使用的密钥</p>
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
            <p class="text-sm font-medium text-[var(--text-secondary)]">总消费</p>
            <p class="text-2xl font-semibold text-green-600 dark:text-green-400">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>¥{{ incomeStats.total.toFixed(3) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">累计消费</p>
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-yellow-100 dark:bg-yellow-900/20">
              <SvgCakeIcon class="h-6 w-6 text-yellow-600 dark:text-yellow-400" />
            </div>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-[var(--text-secondary)]">今日消费</p>
            <p class="text-2xl font-semibold text-green-600 dark:text-green-400">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>¥{{ incomeStats.today.toFixed(3) }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">今日消费</p>
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

    <!-- 详情卡片 -->
    <div class="mt-5 grid grid-cols-1 lg:grid-cols-2 gap-5">
      <AnalysisChartCard title="我使用的模型">
        <MyUsageModels />
      </AnalysisChartCard>
      
      <AnalysisChartCard title="使用详单">
        <UsageDetailTable />
      </AnalysisChartCard>
    </div>
  </div>
</template>
