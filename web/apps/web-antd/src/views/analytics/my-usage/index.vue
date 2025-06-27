<script lang="ts" setup>
import type { TabOption } from '@vben/types';
import { ref, onMounted, computed, nextTick, watch } from 'vue';
import { requestClient } from '#/api/request';
import { useEcharts, EchartsUI } from '@vben/plugins/echarts';

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

const loading = ref(false);
const usageRecords = ref<TokenUsageRecord[]>([]);

// 图表相关
const incomeChartRef = ref();
const { renderEcharts } = useEcharts(incomeChartRef);

// 计算时间段统计数据
const timeStatsData = computed(() => {
  const records = usageRecords.value;
  const now = new Date();
  const today = now.toISOString().split('T')[0];
  const weekStart = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
  
  // 今日数据
  const todayRecords = records.filter(r => r.Timestamp.startsWith(today));
  const todayStats = {
    totalTokens: todayRecords.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: todayRecords.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: todayRecords.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: todayRecords.length,
    models: new Set(todayRecords.map(r => r.Model)).size,
    clients: new Set(todayRecords.map(r => r.ClientID)).size,
    income: todayRecords.reduce((sum, r) => sum + (r.OutputTokens * modelMultiplier.value), 0)
  };
  
  // 本周数据
  const weekRecords = records.filter(r => new Date(r.Timestamp) >= weekStart);
  const weekStats = {
    totalTokens: weekRecords.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: weekRecords.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: weekRecords.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: weekRecords.length,
    models: new Set(weekRecords.map(r => r.Model)).size,
    clients: new Set(weekRecords.map(r => r.ClientID)).size,
    income: weekRecords.reduce((sum, r) => sum + (r.OutputTokens * modelMultiplier.value), 0)
  };
  
  // 本月数据
  const monthRecords = records.filter(r => new Date(r.Timestamp) >= monthStart);
  const monthStats = {
    totalTokens: monthRecords.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: monthRecords.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: monthRecords.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: monthRecords.length,
    models: new Set(monthRecords.map(r => r.Model)).size,
    clients: new Set(monthRecords.map(r => r.ClientID)).size,
    income: monthRecords.reduce((sum, r) => sum + (r.OutputTokens * modelMultiplier.value), 0)
  };
  
  // 总计数据
  const totalStats = {
    totalTokens: records.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: records.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: records.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: records.length,
    models: new Set(records.map(r => r.Model)).size,
    clients: new Set(records.map(r => r.ClientID)).size,
    income: records.reduce((sum, r) => sum + (r.OutputTokens * modelMultiplier.value), 0)
  };
  
  return {
    today: todayStats,
    week: weekStats,
    month: monthStats,
    total: totalStats
  };
});

// 计算输出Token统计
const outputTokenStats = computed(() => {
  const records = usageRecords.value;
  const totalOutputTokens = records.reduce((sum, r) => sum + r.OutputTokens, 0);
  
  const today = new Date().toISOString().split('T')[0] || '';
  const todayRecords = records.filter(r => r.Timestamp && r.Timestamp.startsWith(today));
  const todayOutputTokens = todayRecords.reduce((sum, r) => sum + r.OutputTokens, 0);

  return {
    total: totalOutputTokens,
    today: todayOutputTokens,
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

// 收益设置（默认每个输出Token 0.001元）
const modelMultiplier = ref(0.001);

// 计算收益统计
const incomeStats = computed(() => {
  const records = usageRecords.value;
  const totalIncome = records.reduce((sum, r) => sum + (r.OutputTokens * modelMultiplier.value), 0);
  
  const today = new Date().toISOString().split('T')[0] || '';
  const todayRecords = records.filter(r => r.Timestamp && r.Timestamp.startsWith(today));
  const todayIncome = todayRecords.reduce((sum, r) => sum + (r.OutputTokens * modelMultiplier.value), 0);
  
  return {
    total: totalIncome,
    today: todayIncome,
  };
});

// 计算按模型收益数据（用于柱状图）
const modelIncomeData = computed(() => {
  return modelStats.value.map(model => ({
    name: model.name,
    income: model.outputTokens * modelMultiplier.value,
    outputTokens: model.outputTokens,
    requestCount: model.requestCount,
  })).sort((a, b) => b.income - a.income);
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
    
    if (response && response.data && Array.isArray(response.data)) {
      usageRecords.value = response.data;
      console.log('获取到Token使用记录:', usageRecords.value.length, '条');
      console.log('样本数据:', usageRecords.value.slice(0, 2));
    } else if (Array.isArray(response)) {
      usageRecords.value = response;
      console.log('获取到Token使用记录(直接数组):', usageRecords.value.length, '条');
      console.log('样本数据:', usageRecords.value.slice(0, 2));
    } else {
      console.warn('Token使用数据格式不正确:', response);
      usageRecords.value = [];
    }
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

// 更新收益柱状图
const updateIncomeChart = () => {
  if (!incomeChartRef.value || modelIncomeData.value.length === 0) {
    console.log('图表组件还未挂载或无数据');
    return;
  }

  console.log('更新收益柱状图数据:', modelIncomeData.value);

  const option = {
    title: {
      text: '按模型收益统计',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'bold'
      }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      formatter: function(params: any) {
        const data = params[0];
        const modelData = modelIncomeData.value.find(m => m.name === data.name);
        return `
          <div style="padding: 8px;">
            <div style="font-weight: bold; margin-bottom: 4px;">${data.name}</div>
            <div>收益: ¥${data.value.toFixed(3)}</div>
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
      type: 'category',
      data: modelIncomeData.value.map(item => item.name),
      axisLabel: {
        interval: 0,
        rotate: modelIncomeData.value.length > 3 ? 45 : 0,
        fontSize: 12
      }
    },
    yAxis: {
      type: 'value',
      name: '收益 (¥)',
      axisLabel: {
        formatter: '¥{value}'
      }
    },
    series: [
      {
        name: '收益',
        type: 'bar',
        data: modelIncomeData.value.map(item => ({
          value: item.income,
          name: item.name,
          itemStyle: {
            color: '#10B981'
          }
        })),
        markLine: {
          data: [
            { type: 'average', name: '平均值' }
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
            <p class="text-sm font-medium text-[var(--text-secondary)]">总输出Token</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-20"></span>
              <span v-else>{{ outputTokenStats.total.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)] mt-1">累计生成</p>
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
            <p class="text-sm font-medium text-[var(--text-secondary)]">今日输出Token</p>
            <p class="text-2xl font-semibold text-[var(--text-primary)]">
              <span v-if="loading" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-16"></span>
              <span v-else>{{ outputTokenStats.today.toLocaleString() }}</span>
            </p>
            <p class="text-xs text-[var(--text-tertiary)]">今日生成</p>
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
            <p class="text-xs text-[var(--text-tertiary)]">今日收益</p>
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
    <div class="mt-5 w-full">
      <AnalysisChartCard title="我使用的模型">
        <MyUsageModels />
      </AnalysisChartCard>
    </div>
  </div>
</template>
