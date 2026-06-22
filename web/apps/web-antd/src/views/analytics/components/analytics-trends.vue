<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';

import { onMounted, ref, watch } from 'vue';
import { requestClient } from '#/api/request';
import { RangePicker as ARangePicker } from 'ant-design-vue';
import dayjs, { type Dayjs } from 'dayjs';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

interface UsageTrendPoint {
  date: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  calls: number;
}

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const loading = ref(false);
const trendPoints = ref<UsageTrendPoint[]>([]);

// 日期范围（默认最近 90 天，用 dayjs 对象供 a-range-picker 使用）
const dateRange = ref<[Dayjs, Dayjs]>([
  dayjs().subtract(90, 'day'),
  dayjs(),
]);

// 获取趋势数据（调 /usage/trend 接口）
const fetchTrendData = async (startDate?: string, endDate?: string) => {
  try {
    loading.value = true;
    const params: Record<string, string> = {};
    if (startDate) params.start_date = startDate;
    if (endDate) params.end_date = endDate;
    
    const response = await requestClient.get('/user/usage/trend', { params });
    if (response && Array.isArray(response.data)) {
      trendPoints.value = response.data;
    } else {
      trendPoints.value = [];
    }
  } catch (error) {
    console.error('获取使用趋势失败:', error);
    trendPoints.value = [];
  } finally {
    loading.value = false;
  }
};

// 日期范围变更
const onDateChange = (dates: [Dayjs, Dayjs] | null) => {
  if (dates && dates.length === 2) {
    const start = dates[0].format('YYYY-MM-DD');
    const end = dates[1].format('YYYY-MM-DD');
    dateRange.value = [dates[0], dates[1]];
    fetchTrendData(start, end);
  }
};

// 监听趋势数据变化，重新渲染图表
watch(trendPoints, () => {
  renderChart();
}, { deep: true });

// 渲染图表
const renderChart = () => {
  if (!chartRef.value) return;
  
  if (trendPoints.value.length === 0) {
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
        data: trendPoints.value.map(p => p.input_tokens),
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
        data: trendPoints.value.map(p => p.output_tokens),
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
        data: trendPoints.value.map(p => p.total_tokens),
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
      data: trendPoints.value.map(p => {
        const d = new Date(p.date);
        return `${d.getMonth() + 1}/${d.getDate()}`;
      }),
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
  fetchTrendData(
    dateRange.value[0].format('YYYY-MM-DD'),
    dateRange.value[1].format('YYYY-MM-DD'),
  );
});
</script>

<template>
  <div class="space-y-3">
    <div class="flex justify-end">
      <a-range-picker
        :value="dateRange"
        @change="onDateChange"
        :allow-clear="false"
        size="small"
      />
    </div>
    <EchartsUI ref="chartRef" />
  </div>
</template>
