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

// 计算按日期分组的使用趋势数据
const trendData = computed(() => {
  if (!usageRecords.value || usageRecords.value.length === 0) {
    return { dates: [], inputTokens: [], outputTokens: [], totalTokens: [] };
  }

  // 按日期分组统计
  const dateMap: { [key: string]: { input: number; output: number; total: number } } = {};
  
  usageRecords.value.forEach((record: TokenUsageRecord) => {
    if (!record.Timestamp) return;
    
    const date = record.Timestamp.split('T')[0] || record.Timestamp.split(' ')[0];
    if (!date) return;
    
    if (!dateMap[date]) {
      dateMap[date] = { input: 0, output: 0, total: 0 };
    }
    
    dateMap[date].input += record.InputTokens || 0;
    dateMap[date].output += record.OutputTokens || 0;
    dateMap[date].total += record.TotalTokens || 0;
  });

  // 获取最近30天的数据
  const sortedDates = Object.keys(dateMap).sort().slice(-30);
  
  return {
    dates: sortedDates.map(date => {
      const d = new Date(date);
      return `${d.getMonth() + 1}/${d.getDate()}`;
    }),
    inputTokens: sortedDates.map(date => dateMap[date]?.input || 0),
    outputTokens: sortedDates.map(date => dateMap[date]?.output || 0),
    totalTokens: sortedDates.map(date => dateMap[date]?.total || 0),
  };
});

// 渲染图表
const renderChart = () => {
  if (!chartRef.value) return;
  
  const data = trendData.value;
  if (data.dates.length === 0) {
    // 如果没有数据，显示空状态
    renderEcharts({
      title: {
        text: '暂无使用数据',
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
      text: 'Token使用趋势',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'bold'
      }
    },
    legend: {
      data: ['输入Token', '输出Token', '总Token'],
      top: 30
    },
    grid: {
      bottom: '10%',
      containLabel: true,
      left: '3%',
      right: '4%',
      top: '20%',
    },
    series: [
      {
        name: '输入Token',
        areaStyle: {
          opacity: 0.3
        },
        data: data.inputTokens,
        itemStyle: {
          color: '#5ab1ef',
        },
        smooth: true,
        type: 'line',
      },
      {
        name: '输出Token',
        areaStyle: {
          opacity: 0.3
        },
        data: data.outputTokens,
        itemStyle: {
          color: '#019680',
        },
        smooth: true,
        type: 'line',
      },
      {
        name: '总Token',
        areaStyle: {
          opacity: 0.2
        },
        data: data.totalTokens,
        itemStyle: {
          color: '#ff9800',
        },
        smooth: true,
        type: 'line',
      },
    ],
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        lineStyle: {
          color: '#019680',
          width: 1,
        },
      },
      formatter: function(params: any) {
        let result = `<div style="padding: 8px;"><div style="font-weight: bold; margin-bottom: 4px;">${params[0].axisValue}</div>`;
        params.forEach((param: any) => {
          result += `<div><span style="display:inline-block;margin-right:5px;border-radius:10px;width:10px;height:10px;background-color:${param.color};"></span>${param.seriesName}: ${param.value.toLocaleString()}</div>`;
        });
        result += '</div>';
        return result;
      }
    },
    xAxis: {
      axisTick: {
        show: false,
      },
      boundaryGap: false,
      data: data.dates,
      splitLine: {
        lineStyle: {
          type: 'dashed',
          width: 1,
        },
        show: true,
      },
      type: 'category',
    },
    yAxis: [
      {
        axisTick: {
          show: false,
        },
        name: 'Token数量',
        splitArea: {
          show: true,
        },
        splitNumber: 5,
        type: 'value',
        axisLabel: {
          formatter: function(value: number) {
            if (value >= 1000000) {
              return (value / 1000000).toFixed(1) + 'M';
            } else if (value >= 1000) {
              return (value / 1000).toFixed(1) + 'K';
            }
            return value.toString();
          }
        }
      },
    ],
  });
};

onMounted(() => {
  renderChart();
});
</script>

<template>
  <EchartsUI ref="chartRef" />
</template>
