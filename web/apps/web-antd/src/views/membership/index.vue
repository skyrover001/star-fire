<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { getMembershipApi, buyMembershipApi } from '#/api/core/membership';
import { $t } from '#/locales';

defineOptions({ name: 'Membership' });

interface MembershipInfo {
  // 消费者会员
  membership: string;
  expire_at: string;
  vip_upgrade_price: number;
  svip_upgrade_price: number;
  // 消费者限流
  rate_limit_rpm: number;
  rate_limit_tpm: number;
  // 贡献者会员
  contributor_membership: string;
  contributor_expire_at: string;
  contributor_vip_upgrade_price: number;
  contributor_svip_upgrade_price: number;
  // 贡献者连接池
  max_connections: number;
  current_connections: number;
  online_clients: number;
  usage_percent: number;
  vip_price: number;
  svip_price: number;
}

type MembershipType = 'consumer' | 'contributor';

const loading = ref(false);
const buying = ref(false);
const activeType = ref<MembershipType>('consumer');

const info = ref<MembershipInfo>({
  membership: 'normal',
  expire_at: '',
  vip_upgrade_price: 20,
  svip_upgrade_price: 50,
  rate_limit_rpm: 120,
  rate_limit_tpm: 500000,
  contributor_membership: 'normal',
  contributor_expire_at: '',
  contributor_vip_upgrade_price: 20,
  contributor_svip_upgrade_price: 50,
  max_connections: 3,
  current_connections: 0,
  online_clients: 0,
  usage_percent: 0,
  vip_price: 20,
  svip_price: 50,
});

const membershipLabels: Record<string, string> = {
  normal: $t('business.membership.normal'),
  vip: $t('business.membership.vip'),
  svip: $t('business.membership.svip'),
};

const typeName = computed(() =>
  activeType.value === 'contributor'
    ? $t('business.membership.contributor')
    : $t('business.membership.consumer'),
);

// 当前激活类型的会员等级
const currentMembership = computed(() =>
  activeType.value === 'contributor' ? info.value.contributor_membership : info.value.membership,
);
const currentExpireAt = computed(() =>
  activeType.value === 'contributor' ? info.value.contributor_expire_at : info.value.expire_at,
);
const currentVipUpgrade = computed(() =>
  activeType.value === 'contributor' ? info.value.contributor_vip_upgrade_price : info.value.vip_upgrade_price,
);
const currentSvipUpgrade = computed(() =>
  activeType.value === 'contributor' ? info.value.contributor_svip_upgrade_price : info.value.svip_upgrade_price,
);

const loadMembership = async () => {
  loading.value = true;
  try {
    const res = await getMembershipApi();
    if (res) {
      info.value = res;
    }
  } catch {
    // ignore
  } finally {
    loading.value = false;
  }
};

// 获取目标等级的应付金额（升级补差价 / 续费全价）
const getPrice = (level: string) => {
  if (level === 'vip') return currentVipUpgrade.value;
  if (level === 'svip') return currentSvipUpgrade.value;
  return 0;
};

// 判断是否为升级（当前等级低于目标等级）
const isUpgrade = (level: string) => {
  const order = { normal: 0, vip: 1, svip: 2 };
  return order[level] > order[currentMembership.value];
};

// 购买前确认
const buy = (level: string) => {
  const price = getPrice(level);
  const levelName = membershipLabels[level];
  const isUp = isUpgrade(level);
  const action = isUp ? $t('business.membership.upgrade') : $t('business.membership.renew');
  const type = typeName.value;

  Modal.confirm({
    title: $t('business.membership.confirmTitle', { action, type, level: levelName }),
    content: $t('business.membership.confirmContent', {
      type,
      current: membershipLabels[currentMembership.value],
      action,
      level: levelName,
      price,
    }),
    okText: $t('business.membership.confirmPay'),
    cancelText: $t('business.membership.cancel'),
    onOk: async () => {
      buying.value = true;
      try {
        await buyMembershipApi(level, activeType.value);
        message.success($t('business.membership.buySuccess', { action }));
        await loadMembership();
      } catch (error: any) {
        message.error(error?.message || $t('business.membership.buyFailed', { action }));
      } finally {
        buying.value = false;
      }
    },
  });
};

const formatExpire = (t: string) => {
  if (!t) return $t('business.membership.notActivated');
  return new Date(t).toLocaleDateString();
};

onMounted(() => {
  loadMembership();
});
</script>

<template>
  <div class="p-5">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-[var(--text-primary)]">{{ $t('business.membership.title') }}</h1>
      <p class="mt-1 text-sm text-[var(--text-secondary)]">
        {{ $t('business.membership.subtitle') }}
      </p>
    </div>

    <!-- 类型切换 -->
    <div class="mb-6 inline-flex rounded-lg bg-[var(--bg-color-secondary)] p-1">
      <button
        class="rounded-md px-4 py-1.5 text-sm font-medium transition-all"
        :class="activeType === 'consumer' ? 'bg-[var(--content-bg)] text-[var(--primary-color)] shadow' : 'text-[var(--text-secondary)]'"
        @click="activeType = 'consumer'"
      >{{ $t('business.membership.consumer') }}</button>
      <button
        class="rounded-md px-4 py-1.5 text-sm font-medium transition-all"
        :class="activeType === 'contributor' ? 'bg-[var(--content-bg)] text-[var(--primary-color)] shadow' : 'text-[var(--text-secondary)]'"
        @click="activeType = 'contributor'"
      >{{ $t('business.membership.contributor') }}</button>
    </div>

    <!-- 当前状态 -->
    <div class="mb-6 rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-5">
      <p class="text-sm text-[var(--text-secondary)]">
        {{ $t('business.membership.currentLevel', { type: typeName }) }}
        <span class="font-semibold text-[var(--primary-color)]">{{ membershipLabels[currentMembership] || currentMembership }}</span>
        <span v-if="currentMembership !== 'normal'" class="ml-2 text-xs text-[var(--text-tertiary)]">
          {{ $t('business.membership.expireAt', { date: formatExpire(currentExpireAt) }) }}
        </span>
      </p>
      <template v-if="activeType === 'contributor'">
        <p class="mt-1 text-sm text-[var(--text-secondary)]">
          {{ $t('business.membership.connLimit') }}<span class="font-semibold">{{ info.max_connections === -1 ? $t('business.membership.unlimited') : info.max_connections }}</span>
        </p>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">
          {{ $t('business.membership.currentConnections') }}<span class="font-semibold">{{ info.current_connections }}</span>
          <span class="ml-2 text-xs text-[var(--text-tertiary)]">{{ $t('business.membership.onlineClients', { count: info.online_clients }) }}</span>
        </p>
        <div v-if="info.max_connections > 0" class="mt-2 flex items-center gap-2">
          <div class="h-2 w-48 rounded-full bg-[var(--bg-color-secondary)] overflow-hidden">
            <div
              class="h-full rounded-full transition-all"
              :class="info.usage_percent >= 90 ? 'bg-red-500' : info.usage_percent >= 70 ? 'bg-orange-500' : 'bg-green-500'"
              :style="`width: ${info.usage_percent}%`"
            ></div>
          </div>
          <span class="text-xs text-[var(--text-tertiary)]">{{ $t('business.membership.connPoolUsage', { percent: info.usage_percent }) }}</span>
        </div>
        <!-- SVIP 无限连接池：固定宽度蓝色/紫色条表示"无限容量"，永不填满，绝不显示红色（满） -->
        <div v-else-if="info.max_connections === -1" class="mt-2 flex items-center gap-2">
          <div class="h-2 w-48 rounded-full bg-[var(--bg-color-secondary)] overflow-hidden">
            <div
              class="h-full rounded-full transition-all"
              style="width: 30%; background: linear-gradient(90deg, #3b82f6, #8b5cf6);"
            ></div>
          </div>
          <span class="text-xs text-[var(--text-tertiary)]">{{ $t('business.membership.connPoolUnlimited', { count: info.current_connections }) }}</span>
        </div>
      </template>
      <template v-else>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">
          {{ $t('business.membership.rateLimit') }}<span class="font-semibold">{{ $t('business.membership.rateLimitValue', { rpm: info.rate_limit_rpm, tpm: info.rate_limit_tpm }) }}</span>
        </p>
      </template>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- 普通会员 -->
      <div class="rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-6 flex flex-col">
        <h3 class="text-lg font-bold text-[var(--text-primary)]">{{ $t('business.membership.normal') }}</h3>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">{{ $t('business.membership.free') }}</p>
        <ul class="mt-4 space-y-2 text-sm text-[var(--text-secondary)] flex-1">
          <li v-if="activeType === 'contributor'">• {{ $t('business.membership.normalConn') }}</li>
          <li v-else>• {{ $t('business.membership.normalRateLimit') }}</li>
          <li>• {{ $t('business.membership.basicBenefits') }}</li>
        </ul>
        <div class="mt-4">
          <span v-if="currentMembership === 'normal'" class="inline-block w-full text-center rounded-lg bg-[var(--bg-color-secondary)] px-4 py-2 text-sm font-medium text-[var(--text-secondary)]">
            {{ $t('business.membership.currentLevelBadge') }}
          </span>
        </div>
      </div>

      <!-- VIP -->
      <div class="rounded-xl border-2 border-[var(--primary-color)] bg-[var(--content-bg)] p-6 flex flex-col shadow-lg">
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-bold text-[var(--primary-color)]">{{ $t('business.membership.vip') }}</h3>
          <span class="rounded-full bg-[var(--primary-color)]/10 px-2 py-0.5 text-xs font-medium text-[var(--primary-color)]">{{ $t('business.membership.recommended') }}</span>
        </div>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">{{ $t('business.membership.perYear', { price: info.vip_price }) }}</p>
        <p v-if="currentMembership !== 'vip'" class="mt-1 text-xs text-[var(--text-tertiary)]">
          {{ currentMembership === 'normal' ? $t('business.membership.newPrice', { price: currentVipUpgrade }) : $t('business.membership.upgradePrice', { price: currentVipUpgrade }) }}
        </p>
        <ul class="mt-4 space-y-2 text-sm text-[var(--text-secondary)] flex-1">
          <li v-if="activeType === 'contributor'">• {{ $t('business.membership.vipConn') }}</li>
          <li v-else>• {{ $t('business.membership.vipRateLimit') }}</li>
          <li>• {{ $t('business.membership.higherBenefits') }}</li>
        </ul>
        <div class="mt-4">
          <button
            v-if="currentMembership === 'vip'"
            disabled
            class="w-full rounded-lg bg-[var(--bg-color-secondary)] px-4 py-2 text-sm font-medium text-[var(--text-secondary)]"
          >{{ $t('business.membership.currentLevelBadge') }}</button>
          <button
            v-else
            :disabled="buying"
            class="w-full rounded-lg bg-[var(--primary-color)] px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
            @click="buy('vip')"
          >{{ currentMembership === 'svip' ? $t('business.membership.upgradeToVip') : $t('business.membership.buyNow') }}</button>
        </div>
      </div>

      <!-- SVIP -->
      <div class="rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-6 flex flex-col">
        <h3 class="text-lg font-bold text-[var(--text-primary)]">{{ $t('business.membership.svip') }}</h3>
        <p class="mt-1 text-sm text-[var(--text-secondary)]">{{ $t('business.membership.perYear', { price: info.svip_price }) }}</p>
        <p v-if="currentMembership !== 'svip'" class="mt-1 text-xs text-[var(--text-tertiary)]">
          {{ currentMembership === 'normal' ? $t('business.membership.newPrice', { price: currentSvipUpgrade }) : $t('business.membership.upgradePrice', { price: currentSvipUpgrade }) }}
        </p>
        <ul class="mt-4 space-y-2 text-sm text-[var(--text-secondary)] flex-1">
          <li v-if="activeType === 'contributor'">• {{ $t('business.membership.svipConn') }}</li>
          <li v-else>• {{ $t('business.membership.svipRateLimit') }}</li>
          <li>• {{ $t('business.membership.highestBenefits') }}</li>
        </ul>
        <div class="mt-4">
          <button
            v-if="currentMembership === 'svip'"
            disabled
            class="w-full rounded-lg bg-[var(--bg-color-secondary)] px-4 py-2 text-sm font-medium text-[var(--text-secondary)]"
          >{{ $t('business.membership.currentLevelBadge') }}</button>
          <button
            v-else
            :disabled="buying"
            class="w-full rounded-lg bg-[var(--primary-color)] px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
            @click="buy('svip')"
          >{{ currentMembership === 'vip' ? $t('business.membership.upgradeToSvip') : $t('business.membership.buyNow') }}</button>
        </div>
      </div>
    </div>

    <div class="mt-6 rounded-xl border border-[var(--border-color)] bg-[var(--content-bg)] p-4 text-sm text-[var(--text-secondary)]">
      {{ $t('business.membership.tip') }}
    </div>
  </div>
</template>
