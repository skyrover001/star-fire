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
  IPPM: number;
  OPPM: number;
  CIPPM: number;
  InputTokens: number;
  OutputTokens: number;
  CachedTokens: number;
  TotalTokens: number;
  Timestamp: string;
}

interface ModelUsageSummary {
  model: string;
  totalCalls: number;
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  cachedTokens: number;
  totalCost: number;
  uncachedInputCost: number;
  cachedInputCost: number;
  outputCost: number;
  lastUsed: string;
}

const usageRecords = inject<Ref<TokenUsageRecord[]>>(
  'usageRecords',
  ref<TokenUsageRecord[]>([]),
);
const loading = inject<Ref<boolean>>('usageLoading', ref(false));

// 当前展开详情的模型
const expandedModel = ref<string | null>(null);

// 详情分页
const detailPage = ref(1);
const detailPageSize = 15;

// 费用计算
const calcRecordCost = (r: TokenUsageRecord) => {
  const cached = r.CachedTokens || 0;
  const uncachedInput = ((r.InputTokens - cached) * (r.IPPM || 0)) / 1000000;
  const cachedInput = (cached * (r.CIPPM || 0)) / 1000000;
  const output = (r.OutputTokens * (r.OPPM || 0)) / 1000000;
  return { uncachedInput, cachedInput, output, total: uncachedInput + cachedInput + output };
};

// 模型汇总
const modelUsageSummary = computed<ModelUsageSummary[]>(() => {
  const stats: Record<string, {
    calls: number;
    totalTokens: number;
    inputTokens: number;
    outputTokens: number;
    cachedTokens: number;
    uncachedInputCost: number;
    cachedInputCost: number;
    outputCost: number;
    totalCost: number;
    lastTs: number;
  }> = {};

  usageRecords.value.forEach(record => {
    const model = record.Model;
    if (!stats[model]) {
      stats[model] = {
        calls: 0, totalTokens: 0, inputTokens: 0, outputTokens: 0, cachedTokens: 0,
        uncachedInputCost: 0, cachedInputCost: 0, outputCost: 0, totalCost: 0, lastTs: 0,
      };
    }
    const s = stats[model]!;
    s.calls += 1;
    s.totalTokens += record.TotalTokens;
    s.inputTokens += record.InputTokens;
    s.outputTokens += record.OutputTokens;
    s.cachedTokens += record.CachedTokens || 0;

    const cost = calcRecordCost(record);
    s.uncachedInputCost += cost.uncachedInput;
    s.cachedInputCost += cost.cachedInput;
    s.outputCost += cost.output;
    s.totalCost += cost.total;

    const ts = new Date(record.Timestamp).getTime();
    if (ts > s.lastTs) s.lastTs = ts;
  });

  return Object.entries(stats).map(([model, s]) => ({
    model,
    totalCalls: s.calls,
    totalTokens: s.totalTokens,
    inputTokens: s.inputTokens,
    outputTokens: s.outputTokens,
    cachedTokens: s.cachedTokens,
    totalCost: s.totalCost,
    uncachedInputCost: s.uncachedInputCost,
    cachedInputCost: s.cachedInputCost,
    outputCost: s.outputCost,
    lastUsed: s.lastTs > 0 ? new Date(s.lastTs).toLocaleString('zh-CN') : 'N/A',
  })).sort((a, b) => b.totalCost - a.totalCost);
});

// 展开模型的详情记录
const expandedRecords = computed(() => {
  if (!expandedModel.value) return [];
  return usageRecords.value
    .filter(r => r.Model === expandedModel.value)
    .sort((a, b) => new Date(b.Timestamp).getTime() - new Date(a.Timestamp).getTime());
});

const paginatedDetailRecords = computed(() => {
  const start = (detailPage.value - 1) * detailPageSize;
  return expandedRecords.value.slice(start, start + detailPageSize);
});

const detailTotalPages = computed(() => {
  return Math.max(1, Math.ceil(expandedRecords.value.length / detailPageSize));
});

const toggleDetail = (model: string) => {
  if (expandedModel.value === model) {
    expandedModel.value = null;
  } else {
    expandedModel.value = model;
    detailPage.value = 1;
  }
};

const formatTokens = (tokens: number) => {
  if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(1)}M`;
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}K`;
  return tokens.toString();
};

const formatTime = (timestamp: string) => {
  return new Date(timestamp).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
  });
};
</script>

<template>
  <div class="space-y-4">
    <!-- 加载状态 -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>

    <div v-else-if="modelUsageSummary.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
      暂无使用记录
    </div>

    <template v-else>
      <!-- 模型汇总表格 -->
      <div class="overflow-x-auto">
        <table class="w-full border-collapse border border-[var(--border-color)]">
          <thead>
            <tr class="bg-[var(--bg-color-secondary)]">
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-left text-sm font-medium text-[var(--text-primary)]">模型</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">调用次数</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">输入Tokens</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">缓存命中</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">输出Tokens</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">总Tokens</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">未命中输入费用</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">缓存命中费用</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">输出费用</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">总费用</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-left text-sm font-medium text-[var(--text-primary)]">最后使用</th>
              <th class="border border-[var(--border-color)] px-4 py-2.5 text-center text-sm font-medium text-[var(--text-primary)]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="model in modelUsageSummary"
              :key="model.model"
              class="hover:bg-[var(--bg-color-secondary)] transition-colors"
              :class="expandedModel === model.model ? 'bg-blue-500/5' : ''"
            >
              <td class="border border-[var(--border-color)] px-4 py-2.5">
                <span class="font-mono text-sm font-medium text-[var(--text-primary)]">{{ model.model }}</span>
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-[var(--text-primary)]">
                {{ model.totalCalls.toLocaleString() }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-blue-500">
                {{ model.inputTokens.toLocaleString() }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm" :class="model.cachedTokens > 0 ? 'text-orange-500 font-medium' : 'text-[var(--text-secondary)]'">
                {{ model.cachedTokens.toLocaleString() }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-green-500">
                {{ model.outputTokens.toLocaleString() }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-medium text-[var(--text-primary)]">
                {{ model.totalTokens.toLocaleString() }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-blue-500">
                ¥{{ model.uncachedInputCost.toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm" :class="model.cachedInputCost > 0 ? 'text-orange-500' : 'text-[var(--text-secondary)]'">
                ¥{{ model.cachedInputCost.toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-green-500">
                ¥{{ model.outputCost.toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-semibold text-emerald-600">
                ¥{{ model.totalCost.toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-sm text-[var(--text-secondary)]">
                {{ model.lastUsed }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-center">
                <button
                  class="text-xs px-2.5 py-1 rounded transition-colors"
                  :class="expandedModel === model.model
                    ? 'bg-blue-500 text-white hover:bg-blue-600'
                    : 'text-blue-500 hover:bg-blue-500/10 border border-blue-500/30'"
                  @click="toggleDetail(model.model)"
                >
                  {{ expandedModel === model.model ? '收起详情' : '查看详情' }}
                </button>
              </td>
            </tr>
          </tbody>
          <!-- 汇总行 -->
          <tfoot>
            <tr class="bg-[var(--bg-color-secondary)] font-semibold">
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-sm text-[var(--text-primary)]">合计（{{ modelUsageSummary.length }} 个模型）</td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-[var(--text-primary)]">
                {{ modelUsageSummary.reduce((s, m) => s + m.totalCalls, 0).toLocaleString() }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-blue-500">
                {{ formatTokens(modelUsageSummary.reduce((s, m) => s + m.inputTokens, 0)) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-orange-500">
                {{ formatTokens(modelUsageSummary.reduce((s, m) => s + m.cachedTokens, 0)) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-green-500">
                {{ formatTokens(modelUsageSummary.reduce((s, m) => s + m.outputTokens, 0)) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-[var(--text-primary)]">
                {{ formatTokens(modelUsageSummary.reduce((s, m) => s + m.totalTokens, 0)) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-blue-500">
                ¥{{ modelUsageSummary.reduce((s, m) => s + m.uncachedInputCost, 0).toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-orange-500">
                ¥{{ modelUsageSummary.reduce((s, m) => s + m.cachedInputCost, 0).toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm text-green-500">
                ¥{{ modelUsageSummary.reduce((s, m) => s + m.outputCost, 0).toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5 text-right text-sm font-bold text-emerald-600">
                ¥{{ modelUsageSummary.reduce((s, m) => s + m.totalCost, 0).toFixed(4) }}
              </td>
              <td class="border border-[var(--border-color)] px-4 py-2.5"></td>
              <td class="border border-[var(--border-color)] px-4 py-2.5"></td>
            </tr>
          </tfoot>
        </table>
      </div>

      <!-- 展开的模型详情 -->
      <div v-if="expandedModel" class="rounded-lg border border-blue-500/30 bg-blue-500/5 overflow-hidden">
        <div class="px-4 py-3 border-b border-blue-500/20 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-[var(--text-primary)]">
              <span class="font-mono">{{ expandedModel }}</span> 使用详单
            </span>
            <span class="text-xs text-[var(--text-secondary)]">
              共 {{ expandedRecords.length }} 条记录
            </span>
          </div>
          <button
            class="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] px-2 py-1 rounded hover:bg-[var(--bg-color-secondary)]"
            @click="expandedModel = null"
          >
            收起
          </button>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-[var(--bg-color-secondary)]">
                <th class="px-3 py-2 text-left text-xs font-medium text-[var(--text-secondary)]">请求ID</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">输入Token</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">缓存命中</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">输出Token</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">未命中输入费用</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">缓存命中费用</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">输出费用</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-[var(--text-secondary)]">总费用</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-[var(--text-secondary)]">客户端</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-[var(--text-secondary)]">时间</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border-color)]">
              <tr v-for="record in paginatedDetailRecords" :key="record.ID" class="hover:bg-[var(--bg-color-secondary)] transition-colors">
                <td class="px-3 py-2 text-sm text-[var(--text-primary)]">
                  <div class="max-w-[100px] truncate font-mono text-xs" :title="record.RequestID">{{ record.RequestID.substring(0, 8) }}...</div>
                </td>
                <td class="px-3 py-2 text-sm text-right text-blue-500">{{ record.InputTokens.toLocaleString() }}</td>
                <td class="px-3 py-2 text-sm text-right" :class="(record.CachedTokens || 0) > 0 ? 'text-orange-500 font-medium' : 'text-[var(--text-secondary)]'">
                  {{ (record.CachedTokens || 0).toLocaleString() }}
                </td>
                <td class="px-3 py-2 text-sm text-right text-green-500">{{ record.OutputTokens.toLocaleString() }}</td>
                <td class="px-3 py-2 text-sm text-right text-blue-500">¥{{ calcRecordCost(record).uncachedInput.toFixed(6) }}</td>
                <td class="px-3 py-2 text-sm text-right" :class="calcRecordCost(record).cachedInput > 0 ? 'text-orange-500' : 'text-[var(--text-secondary)]'">
                  ¥{{ calcRecordCost(record).cachedInput.toFixed(6) }}
                </td>
                <td class="px-3 py-2 text-sm text-right text-green-500">¥{{ calcRecordCost(record).output.toFixed(6) }}</td>
                <td class="px-3 py-2 text-sm text-right font-semibold text-emerald-600">¥{{ calcRecordCost(record).total.toFixed(6) }}</td>
                <td class="px-3 py-2 text-xs text-[var(--text-secondary)]">
                  <div class="max-w-[80px] truncate" :title="record.ClientID">{{ record.ClientID }}</div>
                </td>
                <td class="px-3 py-2 text-xs text-[var(--text-secondary)]">{{ formatTime(record.Timestamp) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- 分页 -->
        <div v-if="detailTotalPages > 1" class="px-4 py-2 border-t border-blue-500/20 flex items-center justify-between">
          <span class="text-xs text-[var(--text-secondary)]">
            第 {{ (detailPage - 1) * detailPageSize + 1 }}-{{ Math.min(detailPage * detailPageSize, expandedRecords.length) }} 条，共 {{ expandedRecords.length }} 条
          </span>
          <div class="flex items-center gap-2">
            <button
              :disabled="detailPage <= 1"
              class="px-2 py-1 text-xs border border-[var(--border-color)] rounded hover:bg-[var(--bg-color-secondary)] disabled:opacity-50 disabled:cursor-not-allowed"
              @click="detailPage--"
            >上一页</button>
            <span class="text-xs text-[var(--text-secondary)]">{{ detailPage }} / {{ detailTotalPages }}</span>
            <button
              :disabled="detailPage >= detailTotalPages"
              class="px-2 py-1 text-xs border border-[var(--border-color)] rounded hover:bg-[var(--bg-color-secondary)] disabled:opacity-50 disabled:cursor-not-allowed"
              @click="detailPage++"
            >下一页</button>
          </div>
        </div>
        <!-- 费用说明 -->
        <div class="px-4 py-2 border-t border-blue-500/20 text-xs text-[var(--text-secondary)]">
          费用 = (输入tokens - 缓存命中) × IPPM + 缓存命中 × CIPPM + 输出tokens × OPPM）/ 1,000,000
        </div>
      </div>
    </template>
  </div>
</template>
