<template>
  <div class="min-h-screen bg-background">
    <!-- 顶部标题栏 -->
    <div class="px-6 py-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-bold text-foreground">{{ $t('business.analytics.priceCapsTitle') }}</h1>
          <p class="mt-2 text-muted-foreground">{{ $t('business.analytics.priceCapsDescription') }}</p>
        </div>
        <button
          class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          @click="openAddModal"
        >
          <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          {{ $t('business.analytics.addLimit') }}
        </button>
      </div>
    </div>

    <div class="px-6 pb-6 space-y-6">
      <!-- 说明卡片 -->
      <div class="rounded-xl bg-card border border-blue-500/30 p-5">
        <div class="flex items-start space-x-3">
          <svg class="h-5 w-5 text-blue-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <div class="text-sm text-muted-foreground space-y-1">
            <p><span class="font-medium text-foreground">IPPM_Ceiling</span>: {{ $t('business.analytics.limitInfoIppm') }}</p>
            <p><span class="font-medium text-foreground">OPPM_Ceiling</span>: {{ $t('business.analytics.limitInfoOppm') }}</p>
            <p><span class="font-medium text-foreground">CIPPM_Ceiling</span>: {{ $t('business.analytics.limitInfoCippm') }}</p>
            <p>{{ $t('business.analytics.limitInfoDefault') }}</p>
          </div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="flex justify-center py-16">
        <div class="flex items-center space-x-3 text-[var(--text-secondary)]">
          <div class="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
          <span>{{ $t('business.analytics.loading') }}</span>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else-if="caps.length === 0" class="rounded-xl bg-card border border-border p-16 text-center">
        <div class="w-16 h-16 bg-accent rounded-full flex items-center justify-center mx-auto mb-4">
          <svg class="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
        </div>
        <h3 class="text-lg font-semibold text-foreground mb-2">{{ $t('business.analytics.noLimits') }}</h3>
        <p class="text-muted-foreground mb-5">{{ $t('business.analytics.noLimitsDescription') }}</p>
        <button
          class="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          @click="openAddModal"
        >
          <svg class="mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          {{ $t('business.analytics.addFirstLimit') }}
        </button>
      </div>

      <!-- 限额列表 -->
      <div v-else class="rounded-xl bg-card border border-border overflow-hidden">
        <table class="w-full">
          <thead>
            <tr class="border-b border-border bg-accent">
              <th class="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ $t('business.analytics.model') }}</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ $t('business.analytics.inputLimit') }}</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ $t('business.analytics.outputLimit') }}</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ $t('business.analytics.cachedInputLimit') }}</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ $t('business.analytics.updatedAt') }}</th>
              <th class="px-6 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">{{ $t('business.analytics.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr
              v-for="cap in caps"
              :key="cap.id"
              class="hover:bg-accent transition-colors"
            >
              <td class="px-6 py-4">
                <span class="font-mono text-sm font-medium text-foreground bg-accent px-2 py-1 rounded">
                  {{ cap.model }}
                </span>
              </td>
              <td class="px-6 py-4">
                <span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                  ¥{{ cap.max_ippm.toFixed(2) }} {{ $t('business.analytics.perMillion') }}
                </span>
              </td>
              <td class="px-6 py-4">
                <span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-blue-500/10 text-blue-500 border border-blue-500/20">
                  ¥{{ cap.max_oppm.toFixed(2) }} {{ $t('business.analytics.perMillion') }}
                </span>
              </td>
              <td class="px-6 py-4">
                <span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-orange-500/10 text-orange-500 border border-orange-500/20">
                  ¥{{ (cap.max_cippm || 0).toFixed(2) }} {{ $t('business.analytics.perMillion') }}
                </span>
              </td>
              <td class="px-6 py-4 text-sm text-muted-foreground">
                {{ formatDate(cap.updated_at) }}
              </td>
              <td class="px-6 py-4 text-right space-x-2">
                <button
                  class="text-xs px-3 py-1.5 rounded-lg bg-accent text-foreground border border-border hover:bg-blue-500/10 hover:text-blue-500 hover:border-blue-500/30 transition-colors"
                  @click="openEditModal(cap)"
                >
                  {{ $t('business.analytics.edit') }}
                </button>
                <button
                  class="text-xs px-3 py-1.5 rounded-lg bg-red-500/10 text-red-500 border border-red-500/20 hover:bg-red-500/20 transition-colors"
                  :disabled="deletingModel === cap.model"
                  @click="confirmDelete(cap)"
                >
                  <span v-if="deletingModel === cap.model">{{ $t('business.analytics.deleting') }}</span>
                  <span v-else>{{ $t('business.analytics.delete') }}</span>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 新增 / 编辑 弹窗 -->
    <div
      v-if="showModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      @click.self="closeModal"
    >
      <div class="w-full max-w-md rounded-2xl bg-card border border-border p-6 shadow-2xl">
        <h2 class="text-xl font-bold text-foreground mb-5">
          {{ editingCap ? $t('business.analytics.editLimit') : $t('business.analytics.addLimit') }}
        </h2>

        <div class="space-y-4">
          <!-- 模型名 -->
          <div>
            <label class="block text-sm font-medium text-muted-foreground mb-1.5">{{ $t('business.analytics.modelName') }}</label>

            <!-- 新增：自定义下拉，完全主题适配 -->
            <div v-if="!editingCap" ref="modelDropdownRef" class="relative">
              <!-- 触发按钮 -->
              <button
                type="button"
                class="w-full flex items-center justify-between rounded-lg border bg-accent px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all duration-150"
                :class="[
                  loadingModels ? 'opacity-50 cursor-not-allowed border-border' : 'cursor-pointer border-border hover:border-blue-500/60',
                  modelDropdownOpen ? 'border-blue-500 ring-2 ring-blue-500/30' : '',
                ]"
                :disabled="loadingModels"
                @click="modelDropdownOpen = !modelDropdownOpen"
              >
                <span :class="form.model ? 'font-mono text-foreground' : 'text-muted-foreground'">
                  {{ form.model || (loadingModels ? $t('business.analytics.loadingModels') : availableModels.length === 0 ? $t('business.analytics.noOnlineModels') : $t('business.analytics.selectModel')) }}
                </span>
                <svg v-if="loadingModels" class="animate-spin h-4 w-4 text-muted-foreground flex-shrink-0" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                </svg>
                <svg v-else class="h-4 w-4 text-muted-foreground flex-shrink-0 transition-transform duration-150" :class="modelDropdownOpen ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                </svg>
              </button>

              <!-- 下拉列表 -->
              <Transition
                enter-active-class="transition ease-out duration-100"
                enter-from-class="opacity-0 translate-y-1 scale-[0.98]"
                enter-to-class="opacity-100 translate-y-0 scale-100"
                leave-active-class="transition ease-in duration-75"
                leave-from-class="opacity-100 translate-y-0 scale-100"
                leave-to-class="opacity-0 translate-y-1 scale-[0.98]"
              >
                <div
                  v-show="modelDropdownOpen && !loadingModels"
                  class="absolute z-20 mt-1.5 w-full max-h-52 overflow-y-auto rounded-xl border border-border bg-popover shadow-2xl py-1"
                >
                  <div v-if="availableModels.length === 0" class="px-4 py-3 text-sm text-muted-foreground text-center">
                    {{ $t('business.analytics.noOnlineModels') }}
                  </div>
                  <button
                    v-for="name in availableModels"
                    :key="name"
                    type="button"
                    class="w-full flex items-center justify-between px-4 py-2.5 text-sm transition-colors"
                    :class="[
                      alreadyConfigured(name)
                        ? 'text-muted-foreground opacity-40 cursor-not-allowed'
                        : form.model === name
                          ? 'bg-blue-500/15 text-blue-400 cursor-default'
                          : 'text-foreground hover:bg-accent cursor-pointer',
                    ]"
                    :disabled="alreadyConfigured(name)"
                    @click="!alreadyConfigured(name) && selectModel(name)"
                  >
                    <span class="font-mono truncate">{{ name }}</span>
                    <span v-if="alreadyConfigured(name)" class="ml-2 flex-shrink-0 text-xs font-sans text-muted-foreground">{{ $t('business.analytics.configured') }}</span>
                    <svg v-else-if="form.model === name" class="ml-2 flex-shrink-0 h-3.5 w-3.5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7"/>
                    </svg>
                  </button>
                </div>
              </Transition>
            </div>

            <!-- 编辑：只读展示 -->
            <div v-else class="w-full rounded-lg border border-border bg-accent px-3 py-2.5">
              <span class="font-mono text-sm font-medium text-foreground">{{ form.model }}</span>
            </div>
          </div>

          <!-- IPPM -->
          <div>
            <label class="block text-sm font-medium text-muted-foreground mb-1.5">
              {{ $t('business.analytics.inputPriceLimit') }}
              <span class="text-muted-foreground/60 font-normal ml-1">{{ $t('business.analytics.currencyPerMillionTokens') }}</span>
            </label>
            <input
              v-model.number="form.maxIPPM"
              type="number"
              min="0"
              step="0.01"
              :placeholder="$t('business.analytics.exampleIppm')"
              class="w-full rounded-lg border border-border bg-accent px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <!-- OPPM -->
          <div>
            <label class="block text-sm font-medium text-muted-foreground mb-1.5">
              {{ $t('business.analytics.outputPriceLimit') }}
              <span class="text-muted-foreground/60 font-normal ml-1">{{ $t('business.analytics.currencyPerMillionTokens') }}</span>
            </label>
            <input
              v-model.number="form.maxOPPM"
              type="number"
              min="0"
              step="0.01"
              :placeholder="$t('business.analytics.exampleOppm')"
              class="w-full rounded-lg border border-border bg-accent px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <!-- CIPPM -->
          <div>
            <label class="block text-sm font-medium text-muted-foreground mb-1.5">
              {{ $t('business.analytics.cachedInputPriceLimit') }}
              <span class="text-muted-foreground/60 font-normal ml-1">{{ $t('business.analytics.currencyPerMillionTokens') }}</span>
            </label>
            <input
              v-model.number="form.maxCIPPM"
              type="number"
              min="0"
              step="0.01"
              :placeholder="$t('business.analytics.exampleCippm')"
              class="w-full rounded-lg border border-border bg-accent px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <!-- 错误提示 -->
          <p v-if="formError" class="text-sm text-red-500">{{ formError }}</p>
        </div>

        <div class="mt-6 flex justify-end space-x-3">
          <button
            class="px-4 py-2 rounded-lg text-sm font-medium text-foreground bg-accent border border-border hover:bg-muted transition-colors"
            @click="closeModal"
          >
            {{ $t('business.analytics.cancel') }}
          </button>
          <button
            class="px-4 py-2 rounded-lg text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            :disabled="saving"
            @click="save"
          >
            {{ saving ? $t('business.analytics.saving') : $t('business.analytics.save') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import { onClickOutside } from '@vueuse/core';
import { message } from 'ant-design-vue';
import { requestClient } from '#/api/request';
import { $t } from '#/locales';
import {
  type PriceCap,
  getPriceCapsApi,
  upsertPriceCapApi,
  deletePriceCapApi,
} from '#/api/core/price-cap';

const loading = ref(false);
const saving = ref(false);
const deletingModel = ref('');
const caps = ref<PriceCap[]>([]);

// ── 在线模型列表 ──────────────────────────────────────────
const loadingModels = ref(false);
const availableModels = ref<string[]>([]);

const fetchAvailableModels = async () => {
  try {
    loadingModels.value = true;
    const res = await requestClient.get<any>('/market/models');
    let models: any[] = [];
    if (Array.isArray(res)) {
      models = res;
    } else if (res?.data && Array.isArray(res.data)) {
      models = res.data;
    } else if (res?.data?.models && Array.isArray(res.data.models)) {
      models = res.data.models;
    }
    availableModels.value = models.map((m: any) => m.name as string).filter(Boolean).sort();
  } catch {
    // 静默失败，保留空列表
  } finally {
    loadingModels.value = false;
  }
};

/** 检查某模型是否已有配置（新增时提示） */
const alreadyConfigured = (name: string) => caps.value.some(c => c.model === name);
// ─────────────────────────────────────────────────────────

// ── 自定义下拉状态 ─────────────────────────────────────────
const modelDropdownOpen = ref(false);
const modelDropdownRef = ref<HTMLElement | null>(null);
onClickOutside(modelDropdownRef, () => { modelDropdownOpen.value = false; });
const selectModel = (name: string) => { form.value.model = name; modelDropdownOpen.value = false; };
// ─────────────────────────────────────────────────────────

const showModal = ref(false);
const editingCap = ref<PriceCap | null>(null);
const formError = ref('');
const form = ref({ model: '', maxIPPM: 0, maxOPPM: 0, maxCIPPM: 0 });

const fetchCaps = async () => {
  try {
    loading.value = true;
    caps.value = await getPriceCapsApi();
  } catch (e: any) {
    message.error($t('business.analytics.fetchLimitsFailed', { error: e?.message ?? $t('business.analytics.unknownError') }));
  } finally {
    loading.value = false;
  }
};

const openAddModal = () => {
  editingCap.value = null;
  form.value = { model: '', maxIPPM: 0, maxOPPM: 0, maxCIPPM: 0 };
  formError.value = '';
  modelDropdownOpen.value = false;
  showModal.value = true;
  fetchAvailableModels();
};

const openEditModal = (cap: PriceCap) => {
  editingCap.value = cap;
  form.value = { model: cap.model, maxIPPM: cap.max_ippm, maxOPPM: cap.max_oppm, maxCIPPM: cap.max_cippm || 0 };
  formError.value = '';
  showModal.value = true;
};

const closeModal = () => {
  showModal.value = false;
};

const save = async () => {
  if (!form.value.model.trim()) {
    formError.value = editingCap.value ? $t('business.analytics.modelNameRequired') : $t('business.analytics.selectModelRequired');
    return;
  }
  if (form.value.maxIPPM < 0 || form.value.maxOPPM < 0 || form.value.maxCIPPM < 0) {
    formError.value = $t('business.analytics.limitCannotBeNegative');
    return;
  }
  formError.value = '';
  try {
    saving.value = true;
    await upsertPriceCapApi(form.value.model.trim(), form.value.maxIPPM, form.value.maxOPPM, form.value.maxCIPPM);
    message.success($t('business.analytics.saved'));
    closeModal();
    await fetchCaps();
  } catch (e: any) {
    message.error($t('business.analytics.saveFailed', { error: e?.message ?? $t('business.analytics.unknownError') }));
  } finally {
    saving.value = false;
  }
};

const confirmDelete = async (cap: PriceCap) => {
  if (!window.confirm($t('business.analytics.deleteConfirm', { model: cap.model }))) return;
  try {
    deletingModel.value = cap.model;
    await deletePriceCapApi(cap.model);
    message.success($t('business.analytics.deleted'));
    caps.value = caps.value.filter(c => c.id !== cap.id);
  } catch (e: any) {
    message.error($t('business.analytics.deleteFailed', { error: e?.message ?? $t('business.analytics.unknownError') }));
  } finally {
    deletingModel.value = '';
  }
};

const formatDate = (dateString: string): string => {
  if (!dateString || dateString.startsWith('0001')) return '—';
  return new Date(dateString).toLocaleString('zh-CN');
};

onMounted(fetchCaps);
</script>
