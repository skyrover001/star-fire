<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';
import type { Ref } from 'vue';

import { onMounted, ref, computed, inject, watch } from 'vue';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

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
  PPM?: number;
}

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

// 从父组件注入使用记录数据
const usageRecords = inject<Ref<TokenUsageRecord[]>>('usageRecords', ref([]));

// 监听数据变化，重新渲染图表
watch(usageRecords, () => {
  renderChart();
}, { deep: true });

// 计算按模型分组的调用统计数据
const callStatsData = computed(() => {
  if (!usageRecords.value || usageRecords.value.length === 0) {
    return { models: [], calls: [], tokens: [] };
  }

  // 按模型分组统计
  const modelMap: { [key: string]: { calls: number; tokens: number } } = {};
  
  usageRecords.value.forEach((record: TokenUsageRecord) => {
    if (!record.Model) return;
    
    if (!modelMap[record.Model]) {
      modelMap[record.Model] = { calls: 0, tokens: 0 };
    }
    
    modelMap[record.Model]!.calls += 1;
    modelMap[record.Model]!.tokens += record.TotalTokens || 0;
  });

  // 按调用次数排序，取前10个
  const sortedModels = Object.entries(modelMap)
    .sort(([,a], [,b]) => b.calls - a.calls)
    .slice(0, 10);
  
  return {
    models: sortedModels.map(([model]) => model),
    calls: sortedModels.map(([,data]) => data.calls),
    tokens: sortedModels.map(([,data]) => data.tokens),
  };
});

// 渲染图表
const renderChart = () => {
  if (!chartRef.value) return;
  
  const data = callStatsData.value;
  if (data.models.length === 0) {
    // 如果没有数据，显示空状态
    renderEcharts({
      title: {
        text: '暂无调用数据',
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#999',
          fontSize: 16
        }
      }
    });
    return;
  }

  renderEcharts({
    title: {
      text: '模型调用统计',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'bold'
      }
    },
    grid: {
      bottom: '15%',
      containLabel: true,
      left: '3%',
      right: '4%',
      top: '15%',
    },
    series: [
      {
        name: '调用次数',
        barMaxWidth: 60,
        data: data.calls.map((value, index) => ({
          value,
          itemStyle: {
            color: `hsl(${200 + index * 20}, 70%, 50%)`
          }
        })),
        type: 'bar',
      },
    ],
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      formatter: function(params: any) {
        const data = params[0];
        const index = data.dataIndex;
        const tokens = callStatsData.value.tokens[index] || 0;
        const avgTokens = data.value > 0 ? Math.round(tokens / data.value) : 0;
        return `
          <div style="padding: 8px;">
            <div style="font-weight: bold; margin-bottom: 4px;">${data.name}</div>
            <div>调用次数: ${data.value.toLocaleString()}</div>
            <div>总Token: ${tokens.toLocaleString()}</div>
            <div>平均Token: ${avgTokens.toLocaleString()}</div>
          </div>
        `;
      }
    },
    xAxis: {
      data: data.models,
      type: 'category',
      axisLabel: {
        interval: 0,
        rotate: data.models.length > 5 ? 45 : 0,
        fontSize: 12
      }
    },
    yAxis: {
      name: '调用次数',
      splitNumber: 5,
      type: 'value',
      axisLabel: {
        formatter: function(value: number) {
          if (value >= 1000) {
            return (value / 1000).toFixed(1) + 'K';
          }
          return value.toString();
        }
      }
    },
  });
};

onMounted(() => {
  renderChart();
});
</script>

<template>
  <EchartsUI ref="chartRef" />
</template>
