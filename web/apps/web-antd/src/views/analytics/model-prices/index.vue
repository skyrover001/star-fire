<template>
  <div class="min-h-screen bg-background">
    <!-- 顶部标题栏 -->
    <div class="px-6 py-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-bold text-foreground">价格配置</h1>
          <p class="mt-2 text-muted-foreground">查看您提供的模型价格（每百万 tokens 价格），价格通过客户端配置设置</p>
        </div>
        <button
          class="inline-flex items-center rounded-lg bg-[var(--bg-color-secondary)] border border-[var(--border-color)] px-4 py-2 text-sm font-medium text-[var(--text-primary)] hover:bg-[var(--bg-color-tertiary)]"
          @click="fetchModels"
        >
          <svg class="mr-2 h-4 w-4" :class="loading ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          刷新
        </button>
      </div>
    </div>

    <div class="px-6 pb-6 space-y-6">
      <!-- 搜索和筛选 -->
      <div class="flex items-center gap-4">
        <div class="relative flex-1 max-w-md">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
          <input
            v-model="searchKeyword"
            type="text"
            placeholder="搜索模型名称或引擎..."
            class="w-full rounded-lg border border-border bg-card pl-10 pr-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <select
          v-model="engineFilter"
          class="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:border-blue-500 focus:outline-none"
        >
          <option value="">全部引擎</option>
          <option v-for="engine in availableEngines" :key="engine" :value="engine">{{ engine }}</option>
        </select>
        <div class="text-sm text-muted-foreground">
          {{ filteredModels.length }} / {{ models.length }} 条
        </div>
      </div>

      <!-- 说明卡片 -->
      <div class="rounded-xl bg-card border border-blue-500/30 p-4">
        <div class="flex items-start space-x-3">
          <svg class="h-5 w-5 text-blue-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <div class="text-sm text-muted-foreground space-y-1">
            <p><span class="font-medium text-foreground">IPPM</span> 输入价格 · <span class="font-medium text-foreground">OPPM</span> 输出价格 · <span class="font-medium text-foreground">CIPPM</span> 缓存命中输入价格（均为 ¥/百万tokens）</p>
            <p>模型价格由客户端配置文件管理，此页面仅供查看。如需修改价格请在客户端设置。</p>
          </div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading && models.length === 0" class="flex justify-center py-16">
        <div class="flex items-center space-x-3 text-muted-foreground">
          <div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
          <span>加载中...</span>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else-if="models.length === 0" class="rounded-xl bg-card border border-border p-16 text-center">
        <div class="w-16 h-16 bg-accent rounded-full flex items-center justify-center mx-auto mb-4">
          <svg class="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
          </svg>
        </div>
        <h3 class="text-lg font-semibold text-foreground mb-2">暂无提供的模型</h3>
        <p class="text-muted-foreground">当前没有客户端连接或没有提供模型。请确保客户端已启动并注册。</p>
      </div>

      <!-- 模型价格列表 -->
      <div v-else class="rounded-xl bg-card border border-border overflow-hidden">
        <table class="w-full">
          <thead>
            <tr class="border-b border-border bg-accent">
              <th class="px-5 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">模型</th>
              <th class="px-5 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">引擎</th>
              <th class="px-5 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">客户端</th>
              <th class="px-5 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">IPPM (¥)</th>
              <th class="px-5 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">OPPM (¥)</th>
              <th class="px-5 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">CIPPM (¥)</th>
              <th class="px-5 py-3 text-center text-xs font-semibold text-muted-foreground uppercase tracking-wider">状态</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr
              v-for="item in filteredModels"
              :key="item.client_id + '|' + item.model_name"
              class="hover:bg-accent transition-colors"
            >
              <td class="px-5 py-3">
                <span class="font-mono text-sm font-medium text-foreground bg-accent px-2 py-0.5 rounded">
                  {{ item.model_name }}
                </span>
              </td>
              <td class="px-5 py-3">
                <span class="text-xs font-medium px-2 py-0.5 rounded-full" :class="getEngineClass(item.engine)">
                  {{ item.engine || 'unknown' }}
                </span>
              </td>
              <td class="px-5 py-3">
                <div class="text-xs font-mono text-muted-foreground">{{ item.client_ip }}</div>
                <div class="text-xs text-muted-foreground truncate max-w-[100px]" :title="item.client_id">{{ item.client_id.substring(0, 8) }}...</div>
              </td>
              <td class="px-5 py-3 text-right">
                <span class="text-sm font-semibold text-blue-500">{{ item.ippm.toFixed(2) }}</span>
              </td>
              <td class="px-5 py-3 text-right">
                <span class="text-sm font-semibold text-green-500">{{ item.oppm.toFixed(2) }}</span>
              </td>
              <td class="px-5 py-3 text-right">
                <span class="text-sm font-semibold" :class="item.cippm > 0 ? 'text-orange-500' : 'text-muted-foreground'">
                  {{ item.cippm.toFixed(2) }}
                </span>
              </td>
              <td class="px-5 py-3 text-center">
                <span v-if="item.online" class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1.5"></span>
                  在线
                </span>
                <span v-else class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-500 border border-red-500/20">
                  <span class="w-1.5 h-1.5 rounded-full bg-red-500 mr-1.5"></span>
                  离线
                </span>
              </td>
            </tr>
          </tbody>
        </table>
        <div class="px-5 py-3 bg-accent border-t border-border">
          <div class="flex justify-between items-center text-xs text-muted-foreground">
            <span>
              共 {{ filteredModels.length }} 个模型实例，{{ onlineCount }} 个在线
            </span>
            <span>价格单位：¥ / 百万 tokens (PPM)</span>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { requestClient } from '#/api/request';

interface UserModel {
  model_name: string;
  engine: string;
  ippm: number;
  oppm: number;
  cippm: number;
  client_id: string;
  client_ip: string;
  online: boolean;
}

const loading = ref(false);
const models = ref<UserModel[]>([]);
const searchKeyword = ref('');
const engineFilter = ref('');

// Available engines for filter
const availableEngines = computed(() => {
  const engines = new Set(models.value.map(m => m.engine));
  return [...engines].sort();
});

// Filtered models
const filteredModels = computed(() => {
  let result = models.value;
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase();
    result = result.filter(m =>
      m.model_name.toLowerCase().includes(kw) ||
      m.engine.toLowerCase().includes(kw)
    );
  }
  if (engineFilter.value) {
    result = result.filter(m => m.engine === engineFilter.value);
  }
  return result;
});

const onlineCount = computed(() => filteredModels.value.filter(m => m.online).length);

const getEngineClass = (engine: string) => {
  switch (engine) {
    case 'ollama':
      return 'bg-purple-500/10 text-purple-500 border border-purple-500/20';
    case 'openai':
      return 'bg-green-500/10 text-green-500 border border-green-500/20';
    case 'vllm':
      return 'bg-blue-500/10 text-blue-500 border border-blue-500/20';
    default:
      return 'bg-accent text-muted-foreground border border-border';
  }
};

// Fetch models
const fetchModels = async () => {
  loading.value = true;
  try {
    const response = await requestClient.get('/user/my-models');
    if (response && Array.isArray(response.models)) {
      models.value = response.models;
    } else if (response && response.data && Array.isArray(response.data.models)) {
      models.value = response.data.models;
    } else {
      models.value = [];
    }
  } catch (error) {
    console.error('获取模型数据失败:', error);
    models.value = [];
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchModels();
});
</script>
