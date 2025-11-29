<script lang="ts" setup>
import { ref, computed, inject } from 'vue';
import type { Ref } from 'vue';

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

const usageRecords = inject<Ref<TokenUsageRecord[]>>(
  'usageRecords',
  ref<TokenUsageRecord[]>([]),
);

const loading = inject<Ref<boolean>>('usageLoading', ref(false));

// 计算单次调用收益（输入tokens数 * IPPM + 输出tokens数 * OPPM）
const calculateSingleCallConsumption = (record: TokenUsageRecord) => {
  // 收益 = (输入tokens数 * IPPM + 输出tokens数 * OPPM) / 1,000,000
  return (record.InputTokens * record.IPPM + record.OutputTokens * record.OPPM) / 1000000;
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

const formatTokens = (tokens: number) => {
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}K`;
  }
  return tokens.toString();
};
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
            <span class="text-[var(--text-secondary)]">总收益:</span>
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
