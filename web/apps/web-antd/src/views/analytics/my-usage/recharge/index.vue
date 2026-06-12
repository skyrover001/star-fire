<script lang="ts" setup>
import { ref, computed } from 'vue';
import { message } from 'ant-design-vue';
import {
  createRechargeOrderApi,
  confirmRechargeApi,
  getRechargeHistoryApi,
  getBalanceApi,
} from '#/api/core/balance';
import type { RechargeRecord, BalanceInfo } from '#/api/core/balance';

// 预设充值金额
const presetAmounts = [10, 20, 50, 100, 200, 500];

const balanceInfo = ref<BalanceInfo>({ balance: 0, total_spent: 0 });
const loadingBalance = ref(false);
const loading = ref(false);
const confirming = ref(false);
const historyLoading = ref(false);

// 充值表单
const amount = ref<number | undefined>(undefined);
const customAmount = ref<string>('');
const paymentMethod = ref<'wechat' | 'alipay'>('wechat');
const currentStep = ref<'form' | 'payment' | 'success'>('form');
const currentOrder = ref<{
  order_id: string;
  amount: number;
  qr_code: string;
  created_at: string;
} | null>(null);

// 充值历史
const history = ref<RechargeRecord[]>([]);
const historyTotal = ref(0);

const selectedAmount = computed(() => {
  if (customAmount.value) {
    const val = Number.parseFloat(customAmount.value);
    return Number.isNaN(val) ? undefined : val;
  }
  return amount.value;
});

const selectPreset = (val: number) => {
  amount.value = val;
  customAmount.value = '';
};

const onCustomInput = () => {
  amount.value = undefined;
};

const fetchBalance = async () => {
  loadingBalance.value = true;
  try {
    const info = await getBalanceApi();
    balanceInfo.value = info;
  } catch {
    // ignore
  } finally {
    loadingBalance.value = false;
  }
};

const fetchHistory = async () => {
  historyLoading.value = true;
  try {
    const res = await getRechargeHistoryApi();
    history.value = res.orders || [];
    historyTotal.value = res.total || 0;
  } catch (err: any) {
    console.error('获取充值记录失败:', err);
    message.error(err?.message || '获取充值记录失败');
    history.value = [];
  } finally {
    historyLoading.value = false;
  }
};

const createOrder = async () => {
  const amt = selectedAmount.value;
  if (!amt || amt <= 0) {
    message.error('请输入有效的充值金额');
    return;
  }
  if (amt > 10000) {
    message.error('单次充值金额不能超过 ¥10,000');
    return;
  }

  loading.value = true;
  try {
    const order = await createRechargeOrderApi(amt, paymentMethod.value);
    currentOrder.value = {
      order_id: order.order_id,
      amount: order.amount,
      qr_code: order.qr_code,
      created_at: order.created_at,
    };
    currentStep.value = 'payment';
  } catch (e: any) {
    message.error(e?.message || '创建订单失败');
  } finally {
    loading.value = false;
  }
};

const confirmPayment = async () => {
  if (!currentOrder.value) return;

  confirming.value = true;
  try {
    const res = await confirmRechargeApi(currentOrder.value.order_id);
    message.success(`充值成功！当前余额: ¥${res.balance.toFixed(2)}`);
    currentStep.value = 'success';
    await fetchBalance();
    await fetchHistory();
  } catch (e: any) {
    message.error(e?.message || '确认充值失败');
  } finally {
    confirming.value = false;
  }
};

const resetForm = () => {
  currentStep.value = 'form';
  currentOrder.value = null;
  amount.value = undefined;
  customAmount.value = '';
  paymentMethod.value = 'wechat';
};

const statusLabel = (status: string) => {
  switch (status) {
    case 'pending':
      return '待支付';
    case 'completed':
      return '已支付';
    case 'cancelled':
      return '已取消';
    default:
      return status;
  }
};

const statusColor = (status: string) => {
  switch (status) {
    case 'pending':
      return 'text-yellow-600 bg-yellow-50 border-yellow-200';
    case 'completed':
      return 'text-green-600 bg-green-50 border-green-200';
    case 'cancelled':
      return 'text-gray-500 bg-gray-50 border-gray-200';
    default:
      return 'text-gray-500 bg-gray-50 border-gray-200';
  }
};

const formatTime = (t: string) => {
  if (!t) return '-';
  return new Date(t).toLocaleString('zh-CN');
};

// 初始化
fetchBalance();
fetchHistory();
</script>

<template>
  <div class="min-h-screen">
    <!-- 顶部标题栏 -->
    <div class="px-6 py-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-bold text-[var(--text-primary)]">账户充值</h1>
          <p class="mt-2 text-[var(--text-secondary)]">充值余额用于模型调用计费</p>
        </div>
      </div>
    </div>

    <div class="px-6 pb-6 space-y-6">
      <!-- 余额卡片 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-green-100 dark:bg-green-900/20">
                <svg class="h-6 w-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-[var(--text-secondary)]">账户余额</p>
              <p class="text-2xl font-semibold text-[var(--text-primary)]">
                <span v-if="loadingBalance" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-24"></span>
                <span v-else class="text-green-600">¥{{ balanceInfo.balance?.toFixed(4) || '0.0000' }}</span>
              </p>
            </div>
          </div>
        </div>

        <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <div class="flex items-center justify-center h-12 w-12 rounded-lg bg-blue-100 dark:bg-blue-900/20">
                <svg class="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
              </div>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-[var(--text-secondary)]">累计消费</p>
              <p class="text-2xl font-semibold text-[var(--text-primary)]">
                <span v-if="loadingBalance" class="inline-block animate-pulse bg-[var(--bg-color-secondary)] rounded h-8 w-24"></span>
                <span v-else>¥{{ balanceInfo.total_spent?.toFixed(4) || '0.0000' }}</span>
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- 充值区域 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] p-6">
        <!-- 步骤1: 选择金额和方式 -->
        <div v-if="currentStep === 'form'">
          <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-4">选择充值金额</h3>

          <!-- 预设金额 -->
          <div class="grid grid-cols-3 md:grid-cols-6 gap-3 mb-4">
            <button
              v-for="amt in presetAmounts"
              :key="amt"
              type="button"
              class="relative rounded-lg border-2 p-3 text-center transition-all"
              :class="amount === amt ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-[var(--border-color)] hover:border-blue-300'"
              @click="selectPreset(amt)"
            >
              <span class="text-lg font-bold text-[var(--text-primary)]">¥{{ amt }}</span>
            </button>
          </div>

          <!-- 自定义金额 -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">自定义金额</label>
            <div class="relative max-w-xs">
              <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-secondary)]">¥</span>
              <input
                v-model="customAmount"
                type="number"
                min="0.01"
                max="10000"
                step="0.01"
                placeholder="输入自定义金额"
                class="w-full rounded-lg border border-[var(--border-color)] bg-[var(--bg-color)] py-2 pl-8 pr-3 text-[var(--text-primary)] focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                @input="onCustomInput"
              />
            </div>
          </div>

          <!-- 支付方式 -->
          <div class="mb-6">
            <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">支付方式</label>
            <div class="flex gap-4">
              <button
                type="button"
                class="flex items-center gap-2 rounded-lg border-2 px-4 py-3 transition-all"
                :class="paymentMethod === 'wechat' ? 'border-green-500 bg-green-50 dark:bg-green-900/20' : 'border-[var(--border-color)]'"
                @click="paymentMethod = 'wechat'"
              >
                <span class="text-green-600 font-bold text-xl">💬</span>
                <span class="font-medium text-[var(--text-primary)]">微信支付</span>
              </button>
              <button
                type="button"
                class="flex items-center gap-2 rounded-lg border-2 px-4 py-3 transition-all"
                :class="paymentMethod === 'alipay' ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-[var(--border-color)]'"
                @click="paymentMethod = 'alipay'"
              >
                <span class="text-blue-600 font-bold text-xl">💳</span>
                <span class="font-medium text-[var(--text-primary)]">支付宝</span>
              </button>
            </div>
          </div>

          <button
            type="button"
            :disabled="!selectedAmount || selectedAmount <= 0 || loading"
            class="inline-flex items-center rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            @click="createOrder"
          >
            <svg v-if="loading" class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            {{ loading ? '创建中...' : `立即充值 ¥${selectedAmount?.toFixed(2) || '0.00'}` }}
          </button>
        </div>

        <!-- 步骤2: 模拟支付 -->
        <div v-else-if="currentStep === 'payment' && currentOrder">
          <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">扫码支付（模拟）</h3>
          <p class="text-sm text-[var(--text-secondary)] mb-6">
            这是模拟支付环境，点击下方"确认支付"按钮即可完成充值
          </p>

          <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-8 text-center mb-6 max-w-sm mx-auto">
            <div class="bg-white dark:bg-gray-700 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-8 mb-4">
              <div class="text-6xl mb-4">📱</div>
              <p class="text-sm text-[var(--text-secondary)] mb-2">
                {{ paymentMethod === 'wechat' ? '微信' : '支付宝' }}扫码支付
              </p>
              <p class="text-2xl font-bold text-[var(--text-primary)]">¥{{ currentOrder.amount.toFixed(2) }}</p>
              <p class="text-xs text-[var(--text-tertiary)] mt-2">订单号: {{ currentOrder.order_id }}</p>
              <p class="text-xs text-[var(--text-tertiary)]">创建时间: {{ formatTime(currentOrder.created_at) }}</p>
            </div>
          </div>

          <div class="flex justify-center gap-4">
            <button
              type="button"
              class="rounded-lg border border-[var(--border-color)] px-4 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-color-secondary)] transition-colors"
              @click="resetForm"
            >
              取消
            </button>
            <button
              type="button"
              :disabled="confirming"
              class="inline-flex items-center rounded-lg bg-green-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              @click="confirmPayment"
            >
              <svg v-if="confirming" class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {{ confirming ? '确认中...' : '确认支付（模拟）' }}
            </button>
          </div>
        </div>

        <!-- 步骤3: 充值成功 -->
        <div v-else-if="currentStep === 'success'" class="text-center py-8">
          <div class="text-6xl mb-4">✅</div>
          <h3 class="text-xl font-bold text-[var(--text-primary)] mb-2">充值成功！</h3>
          <p class="text-[var(--text-secondary)] mb-6">
            充值金额已添加到您的账户余额
          </p>
          <button
            type="button"
            class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
            @click="resetForm"
          >
            继续充值
          </button>
        </div>
      </div>

      <!-- 充值历史 -->
      <div class="rounded-xl bg-[var(--content-bg)] border border-[var(--border-color)] overflow-hidden">
        <div class="px-6 py-4 border-b border-[var(--border-color)]">
          <h3 class="text-lg font-semibold text-[var(--text-primary)]">充值记录</h3>
        </div>

        <div v-if="historyLoading" class="flex justify-center py-8">
          <div class="flex items-center space-x-3 text-[var(--text-secondary)]">
            <div class="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
            <span>加载中...</span>
          </div>
        </div>

        <div v-else-if="history.length === 0" class="p-8 text-center text-[var(--text-secondary)]">
          暂无充值记录
        </div>

        <table v-else class="w-full">
          <thead>
            <tr class="border-b border-[var(--border-color)] bg-[var(--bg-color-secondary)]">
              <th class="px-6 py-3 text-left text-xs font-semibold text-[var(--text-secondary)] uppercase">订单号</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-[var(--text-secondary)] uppercase">金额</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-[var(--text-secondary)] uppercase">支付方式</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-[var(--text-secondary)] uppercase">状态</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-[var(--text-secondary)] uppercase">创建时间</th>
              <th class="px-6 py-3 text-left text-xs font-semibold text-[var(--text-secondary)] uppercase">完成时间</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border-color)]">
            <tr v-for="record in history" :key="record.id" class="hover:bg-[var(--bg-color-secondary)] transition-colors">
              <td class="px-6 py-4 font-mono text-xs text-[var(--text-primary)]">{{ record.order_id }}</td>
              <td class="px-6 py-4">
                <span class="font-semibold text-green-600">¥{{ record.amount?.toFixed(2) }}</span>
              </td>
              <td class="px-6 py-4 text-[var(--text-secondary)]">
                {{ record.payment_method === 'wechat' ? '微信支付' : record.payment_method === 'alipay' ? '支付宝' : record.payment_method }}
              </td>
              <td class="px-6 py-4">
                <span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium border" :class="statusColor(record.status)">
                  {{ statusLabel(record.status) }}
                </span>
              </td>
              <td class="px-6 py-4 text-sm text-[var(--text-secondary)]">{{ formatTime(record.created_at) }}</td>
              <td class="px-6 py-4 text-sm text-[var(--text-secondary)]">{{ formatTime(record.completed_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
