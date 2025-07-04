<script lang="ts" setup>
import { computed, watch } from 'vue';

import { AuthenticationLoginExpiredModal } from '@vben/common-ui';
import { useWatermark } from '@vben/hooks';
import { BookOpenText, CircleHelp, MdiGithub } from '@vben/icons';
import {
  BasicLayout,
  LockScreen,
  UserDropdown,
} from '@vben/layouts';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { openWindow } from '@vben/utils';

import { $t } from '#/locales';
import { useAuthStore } from '#/store';
import LoginForm from '#/views/_core/authentication/login.vue';

const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const { destroyWatermark, updateWatermark } = useWatermark();

const GITHUB_BASE_URL = 'https://github.com/skyrover001/star-fire';

const openGitHub = () => {
  openWindow(GITHUB_BASE_URL, { target: '_blank' });
};

const openGitHubIssues = () => {
  openWindow(`${GITHUB_BASE_URL}/issues`, { target: '_blank' });
};

const menus = computed(() => [
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
