<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';

import { AuthenticationLoginExpiredModal } from '@vben/common-ui';
import { useWatermark } from '@vben/hooks';
import { BookOpenText, CircleHelp, MdiGithub, VipCrown } from '@vben/icons';
import {
  BasicLayout,
  LockScreen,
  Notification,
  UserDropdown,
} from '@vben/layouts';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { openWindow } from '@vben/utils';

import {
  getNotificationsApi,
  getUnreadCountApi,
  markAllNotificationsReadApi,
  markNotificationReadApi,
} from '#/api/core/notification';
import { $t } from '#/locales';
import { useAuthStore } from '#/store';
import LoginForm from '#/views/_core/authentication/login.vue';

const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const router = useRouter();
const { destroyWatermark, updateWatermark } = useWatermark();

const GITHUB_BASE_URL = 'https://github.com/skyrover001/star-fire';

const openGitHub = () => {
  openWindow(GITHUB_BASE_URL, { target: '_blank' });
};

const openGitHubIssues = () => {
  openWindow(`${GITHUB_BASE_URL}/issues`, { target: '_blank' });
};

const openMembership = () => {
  router.push('/membership');
};

const menus = computed(() => [
  {
    handler: openMembership,
    icon: VipCrown,
    text: $t('business.navigation.membership'),
  },
  {
    handler: openGitHub,
    icon: BookOpenText,
    text: $t('ui.widgets.document'),
  },
  {
    handler: openGitHub,
    icon: MdiGithub,
    text: 'GitHub',
  },
  {
    handler: openGitHubIssues,
    icon: CircleHelp,
    text: $t('ui.widgets.qa'),
  },
]);

const avatar = computed(() => {
  return userStore.userInfo?.avatar ?? preferences.app.defaultAvatar;
});

// 计算用户信息显示
const userDescription = computed(() => {
  return userStore.userInfo?.email || 'user@example.com';
});

const userTagText = computed(() => {
  return userStore.userInfo?.roles?.[0] || 'User';
});

const userName = computed(() => {
  return userStore.userInfo?.realName || userStore.userInfo?.username || '用户';
});

async function handleLogout() {
  await authStore.logout(false);
}

// ===== 通知逻辑 =====
const notifications = ref<any[]>([]);
const unreadCount = ref(0);
let notifTimer: ReturnType<typeof setInterval> | null = null;

const loadNotifications = async () => {
  try {
    const res = await getNotificationsApi(1, 20);
    if (res?.data) {
      notifications.value = res.data.map((n: any) => ({
        avatar: preferences.app.defaultAvatar,
        date: n.created_at ? new Date(n.created_at).toLocaleString() : '',
        isRead: n.is_read,
        message: n.content,
        title: n.title,
        id: n.id,
      }));
    }
  } catch {
    // ignore
  }
};

const loadUnreadCount = async () => {
  try {
    const res = await getUnreadCountApi();
    unreadCount.value = res?.unread ?? 0;
  } catch {
    // ignore
  }
};

const refreshNotifications = () => {
  loadNotifications();
  loadUnreadCount();
};

const handleNotificationRead = async (item: any) => {
  if (!item.isRead && item.id) {
    await markNotificationReadApi(item.id);
    item.isRead = true;
    unreadCount.value = Math.max(0, unreadCount.value - 1);
  }
};

const handleMarkAllRead = async () => {
  await markAllNotificationsReadApi();
  notifications.value.forEach((n) => (n.isRead = true));
  unreadCount.value = 0;
};

const handleViewAll = () => {
  // 暂无通知中心页面，跳转到会员中心或保持现状
};

onMounted(() => {
  refreshNotifications();
  notifTimer = setInterval(refreshNotifications, 30000);
});

onUnmounted(() => {
  if (notifTimer) {
    clearInterval(notifTimer);
  }
});

watch(
  () => preferences.app.watermark,
  async (enable) => {
    if (enable) {
      await updateWatermark({
        content: `${userStore.userInfo?.username} - ${userStore.userInfo?.realName}`,
      });
    } else {
      destroyWatermark();
    }
  },
  {
    immediate: true,
  },
);
</script>

<template>
  <BasicLayout @clear-preferences-and-logout="handleLogout">
    <template #logo>
      <div class="flex h-full items-center gap-2 px-3">
        <span
          class="flex size-8 shrink-0 items-center justify-center rounded-lg text-sm text-white"
          style="background: linear-gradient(135deg, #8b5cf6, #3b82f6)"
        >
          ✦
        </span>
        <span v-if="!preferences.sidebar.collapsed" class="flex flex-col text-foreground truncate text-nowrap font-semibold leading-tight">
          {{ $t('page.home.brand') }}
          <span class="text-[10px] font-normal opacity-70">
            {{ $t('page.home.brandSuffix') }}
          </span>
        </span>
      </div>
    </template>
    <template #side-extra-title>
      <span class="text-foreground truncate text-nowrap font-semibold">
        {{ $t('page.home.brand') }}
      </span>
    </template>
    <template #user-dropdown>
      <UserDropdown
        :avatar
        :menus
        :text="userName"
        :description="userDescription"
        :tag-text="userTagText"
        @logout="handleLogout"
      />
    </template>
    <template #notification>
      <Notification
        :dot="unreadCount > 0"
        :notifications="notifications"
        @clear="handleMarkAllRead"
        @make-all="handleMarkAllRead"
        @read="handleNotificationRead"
        @view-all="handleViewAll"
      />
    </template>
    <template #extra>
      <AuthenticationLoginExpiredModal
        v-model:open="accessStore.loginExpired"
        :avatar
      >
        <LoginForm />
      </AuthenticationLoginExpiredModal>
    </template>
    <template #lock-screen>
      <LockScreen :avatar @to-login="handleLogout" />
    </template>
  </BasicLayout>
</template>
