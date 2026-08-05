<script lang="ts" setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { AuthPageLayout } from '@vben/layouts';
import { $t } from '#/locales';

const appName = computed(() => $t('page.home.brand'));
const appNameSuffix = computed(() => $t('page.home.brandSuffix'));
const logo = computed(() => '');
const copyrightYear = new Date().getFullYear();

const router = useRouter();

function goHome() {
  router.push('/home');
}
</script>

<template>
  <AuthPageLayout
    :app-name="appName"
    :app-name-suffix="appNameSuffix"
    :click-logo="goHome"
    :logo="logo"
    :page-description="$t('authentication.pageDesc')"
    :page-title="$t('authentication.pageTitle')"
  >
    <template #copyright>
      <div>Copyright © {{ copyrightYear }} StarFire</div>
    </template>
    <!-- 自定义工具栏 -->
    <!-- <template #toolbar></template> -->
  </AuthPageLayout>
</template>
