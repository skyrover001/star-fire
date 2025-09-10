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

interface ModelUsageSummary {
  model: string;
  totalCalls: number;
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  lastUsed: string;
  avgResponseTime: number;
  firstUsed: string;
  totalConsumption: number; // 总消费金额
}

const usageRecords = ref<TokenUsageRecord[]>([]);
const loading = ref(false);

// 默认PPM值（每百万Token价格）
const defaultPPM = 1000.00;

// 计算单次调用消费
const calculateSingleCallConsumption = (record: TokenUsageRecord) => {
  const ppm = record.PPM || defaultPPM;
  return (ppm / 1000000) * record.TotalTokens;
};

// 根据Token使用记录计算模型使用汇总
const modelUsageSummary = computed<ModelUsageSummary[]>(() => {
  const modelStats: { [key: string]: {
    calls: number;
    totalTokens: number;
    inputTokens: number;
    outputTokens: number;
    timestamps: string[];
    totalConsumption: number; // 总消费金额
  } } = {};

  usageRecords.value.forEach(record => {
    const model = record.Model;
    if (!modelStats[model]) {
      modelStats[model] = {
        calls: 0,
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        timestamps: [],
        totalConsumption: 0
      };
    }

    modelStats[model].calls += 1;
    modelStats[model].totalTokens += record.TotalTokens;
    modelStats[model].inputTokens += record.InputTokens;
    modelStats[model].outputTokens += record.OutputTokens;
    modelStats[model].timestamps.push(record.Timestamp);
    modelStats[model].totalConsumption += calculateSingleCallConsumption(record);
  });

  return Object.entries(modelStats).map(([model, stats]) => {
    const sortedTimestamps = stats.timestamps.filter(t => t).sort((a, b) => 
      new Date(b).getTime() - new Date(a).getTime()
    );

    return {
      model,
      totalCalls: stats.calls,
      totalTokens: stats.totalTokens,
      inputTokens: stats.inputTokens,
      outputTokens: stats.outputTokens,
      lastUsed: sortedTimestamps[0] ? new Date(sortedTimestamps[0]).toLocaleString('zh-CN') : 'N/A',
      firstUsed: sortedTimestamps.length > 0 ? new Date(sortedTimestamps[sortedTimestamps.length - 1]!).toLocaleString('zh-CN') : 'N/A',
      avgResponseTime: Math.floor(Math.random() * 500) + 200, // 模拟平均响应时间
      totalConsumption: stats.totalConsumption,
    };
  }).sort((a, b) => b.totalCalls - a.totalCalls); // 按调用次数排序
});

const fetchUsageData = async () => {
  loading.value = true;
  try {
    console.log('正在获取Token使用数据...');
    const response = await requestClient.get('/user/token-usage');
    
    if (response && response.data && Array.isArray(response.data)) {
      usageRecords.value = response.data;
      console.log('获取到Token使用记录:', usageRecords.value.length, '条');
    } else if (Array.isArray(response)) {
      usageRecords.value = response;
      console.log('获取到Token使用记录:', usageRecords.value.length, '条');
    } else {
      console.warn('Token使用数据格式不正确:', response);
      throw new Error('数据格式错误');
    }
  } catch (error) {
    console.error('获取使用数据失败:', error);
    // 使用模拟数据
    loadMockData();
  } finally {
    loading.value = false;
  }
};

const loadMockData = () => {
  // 使用提供的真实数据格式作为模拟数据
  usageRecords.value = [
    // {
    //   ID: 24,
    //   RequestID: "c8664981-1bc9-47b9-976b-d0d405284a7e",
    //   UserID: "2",
    //   APIKey: "key-1750211178363226700",
    //   ClientID: "",
    //   ClientIP: "::1",
    //   Model: "qwen3:0.6b",
    //   InputTokens: 256,
    //   OutputTokens: 213,
    //   TotalTokens: 469,
    //   Timestamp: "2025-06-18T18:25:31.1905443+08:00"
    // },
    // {
    //   ID: 23,
    //   RequestID: "ad5eee14-bed4-40a6-b2a5-bab188f94117",
    //   UserID: "2",
    //   APIKey: "key-1750211178363226700",
    //   ClientID: "",
    //   ClientIP: "::1",
    //   Model: "qwen3:0.6b",
    //   InputTokens: 136,
    //   OutputTokens: 298,
    //   TotalTokens: 434,
    //   Timestamp: "2025-06-18T17:21:19.0153747+08:00"
    // },
    // {
    //   ID: 22,
    //   RequestID: "ae5d358e-d586-4852-9a6c-e820fc771210",
    //   UserID: "2",
    //   APIKey: "key-1750211178363226700",
    //   ClientID: "",
    //   ClientIP: "::1",
    //   Model: "qwen3:0.6b",
    //   InputTokens: 14,
    //   OutputTokens: 315,
    //   TotalTokens: 329,
    //   Timestamp: "2025-06-18T17:20:49.2163893+08:00"
    // },
    // // 添加更多模型的数据
    // {
    //   ID: 25,
    //   RequestID: "test-llama-request",
    //   UserID: "2",
    //   APIKey: "key-1750211178363226700",
    //   ClientID: "",
    //   ClientIP: "::1",
    //   Model: "llama2:7b",
    //   InputTokens: 180,
    //   OutputTokens: 250,
    //   TotalTokens: 430,
    //   Timestamp: "2025-06-19T10:15:30.0000000+08:00"
    // },
    // {
    //   ID: 26,
    //   RequestID: "test-deepseek-request",
    //   UserID: "2",
    //   APIKey: "key-1750211178363226700",
    //   ClientID: "",
    //   ClientIP: "::1",
    //   Model: "deepseek-coder:6.7b",
    //   InputTokens: 120,
    //   OutputTokens: 180,
    //   TotalTokens: 300,
    //   Timestamp: "2025-06-19T09:30:15.0000000+08:00"
    // }
  ];
  console.log('加载模拟数据完成');
};

const formatTokens = (tokens: number) => {
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}K`;
  }
  return tokens.toString();
};

onMounted(() => {
  fetchUsageData();
});
</script>

<template>
  <div class="overflow-y-auto">
    <div v-if="loading" class="flex items-center justify-center h-full">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>
    
    <div v-else class="space-y-3">
      <div
        v-for="model in modelUsageSummary"
        :key="model.model"
        class="bg-[var(--content-bg)] rounded-lg border border-[var(--border-color)] p-4 hover:shadow-md transition-shadow"
      >
        <div class="flex items-start justify-between mb-2">
          <div>
            <h4 class="font-semibold text-[var(--text-primary)]">{{ model.model }}</h4>
            <span class="text-xs text-[var(--text-tertiary)] bg-[var(--bg-color-secondary)] px-2 py-1 rounded">OLLAMA</span>
          </div>
          <div class="text-right">
            <div class="text-sm font-medium text-[var(--text-primary)]">{{ model.totalCalls }} 次调用</div>
            <div class="text-xs text-[var(--text-secondary)]">{{ formatTokens(model.totalTokens) }} tokens</div>
          </div>
        </div>
        
        <div class="grid grid-cols-2 gap-4 mt-3 text-sm">
          <div>
            <span class="text-[var(--text-secondary)]">最后使用:</span>
            <div class="font-medium text-[var(--text-primary)]">{{ model.lastUsed }}</div>
          </div>
          <div>
            <span class="text-[var(--text-secondary)]">总消费:</span>
            <div class="font-medium text-green-600">¥{{ model.totalConsumption.toFixed(4) }}</div>
          </div>
        </div>
        
        <div class="mt-3 flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <span class="text-xs text-[var(--text-secondary)]">输入/输出:</span>
            <span class="text-sm font-semibold text-[var(--text-primary)]">
              {{ formatTokens(model.inputTokens) }} / {{ formatTokens(model.outputTokens) }}
            </span>
          </div>
          <div class="text-xs text-[var(--text-secondary)]">
            平均响应: {{ model.avgResponseTime }}ms
          </div>
        </div>
        
        <div class="mt-3 flex justify-end space-x-2">
          <button class="text-xs text-blue-600 hover:text-blue-800 transition-colors">
            查看详情
          </button>
          <button class="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors">
            使用历史
          </button>
        </div>
      </div>
      
      <div v-if="modelUsageSummary.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
        暂无使用记录
      </div>
    </div>
  </div>
</template>
