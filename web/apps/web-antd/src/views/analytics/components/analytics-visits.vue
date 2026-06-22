<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';

import { onMounted, ref, watch, inject } from 'vue';
import type { Ref } from 'vue';
import { requestClient } from '#/api/request';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

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

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const loading = ref(false);
const modelStats = ref<ModelUsageStat[]>([]);

// 获取模型调用统计（调 /usage/models 接口）
const fetchModelStats = async () => {
  try {
    loading.value = true;
    const response = await requestClient.get('/user/usage/models');
    if (response && Array.isArray(response.data)) {
      // 按调用次数排序，取前10
      modelStats.value = (response.data as ModelUsageStat[])
        .sort((a, b) => b.calls - a.calls)
        .slice(0, 10);
    } else {
      modelStats.value = [];
    }
  } catch (error) {
    console.error('获取模型调用统计失败:', error);
    modelStats.value = [];
  } finally {
    loading.value = false;
  }
};

// 监听数据变化，重新渲染图表
watch(modelStats, () => {
  renderChart();
}, { deep: true });

// 渲染图表
const renderChart = () => {
  if (!chartRef.value) return;
  
  if (modelStats.value.length === 0) {
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

  const models = modelStats.value.map(s => s.model);
  const calls = modelStats.value.map(s => s.calls);
  const tokens = modelStats.value.map(s => s.total_tokens);

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
        data: calls.map((value, index) => ({
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
        const t = tokens[index] || 0;
        const avgTokens = data.value > 0 ? Math.round(t / data.value) : 0;
        return `
          <div style="padding: 8px;">
            <div style="font-weight: bold; margin-bottom: 4px;">${data.name}</div>
            <div>调用次数: ${data.value.toLocaleString()}</div>
            <div>总Token: ${t.toLocaleString()}</div>
            <div>平均Token: ${avgTokens.toLocaleString()}</div>
          </div>
        `;
      }
    },
    xAxis: {
      data: models,
      type: 'category',
      axisLabel: {
        interval: 0,
        rotate: models.length > 5 ? 45 : 0,
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
  fetchModelStats();
});
</script>

<template>
  <EchartsUI ref="chartRef" />
</template>
