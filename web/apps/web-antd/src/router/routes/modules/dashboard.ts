import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  // 模型广场（原控制台）
  {
    meta: {
      affixTab: true,
      icon: 'lucide:shopping-cart',
      order: 0,
      title: $t('business.navigation.marketplace'),
    },
    name: 'ModelMarketplace',
    path: '/model-marketplace',
    component: () => import('#/views/modelmarket/index.vue'),
  },
  // 模型详情页
  {
    meta: {
      hideInMenu: true,
      title: $t('business.navigation.modelDetail'),
    },
    name: 'ModelMarketplaceDetail',
    path: '/model-marketplace-detail',
    component: () => import('#/views/model-marketplace-detail/index.vue'),
  },
  // 聊天页面
  {
    meta: {
      hideInMenu: true,
      title: $t('business.navigation.chat'),
    },
    name: 'Chat',
    path: '/chat',
    component: () => import('#/views/chat/index.vue'),
  },
  // 我的模型
  // {
  //   meta: {
  //     icon: 'lucide:brain-circuit',
  //     order: 1,
  //     title: '我的模型',
  //   },
  //   name: 'MyModels',
  //   path: '/my-models',
  //   component: () => import('#/views/mymodels/index.vue'),
  // },
  // 模型收益 - 一级菜单
  {
    meta: {
      icon: 'lucide:trending-up',
      order: 1,
      title: $t('business.navigation.modelIncome'),
    },
    name: 'ModelIncome',
    path: '/model-income',
    redirect: '/model-income/overview',
    children: [
      {
        meta: {
          title: $t('business.navigation.incomeOverview'),
        },
        name: 'IncomeOverview',
        path: '/model-income/overview',
        component: () => import('#/views/analytics/my-contribution/index.vue'),
      },
      {
        meta: {
          title: $t('business.navigation.modelPrices'),
        },
        name: 'ModelPrices',
        path: '/model-income/model-prices',
        component: () => import('#/views/analytics/model-prices/index.vue'),
      },
    ],
  },
  // 模型使用 - 一级菜单
  {
    meta: {
      icon: 'lucide:area-chart',
      order: 2,
      title: $t('business.navigation.modelUsage'),
    },
    name: 'ModelUsage',
    path: '/model-usage',
    redirect: '/model-usage/overview',
    children: [
      {
        meta: {
          title: $t('business.navigation.usageOverview'),
        },
        name: 'UsageOverview',
        path: '/model-usage/overview',
        component: () => import('#/views/analytics/my-usage/index.vue'),
      },
      {
        meta: {
          title: $t('business.navigation.priceCaps'),
        },
        name: 'PriceCaps',
        path: '/model-usage/price-caps',
        component: () => import('#/views/analytics/price-caps/index.vue'),
      },
      {
        meta: {
          title: $t('business.navigation.apiKeys'),
        },
        name: 'MyKeys',
        path: '/model-usage/api-keys',
        component: () => import('#/views/apikey/index.vue'),
      },
      {
        meta: {
          title: $t('business.navigation.recharge'),
        },
        name: 'Recharge',
        path: '/model-usage/recharge',
        component: () => import('#/views/analytics/my-usage/recharge/index.vue'),
      },
    ],
  },
  // 保留工作台作为一级菜单 - 暂时隐藏
  // {
  //   meta: {
  //     icon: 'carbon:workspace',
  //     order: 1,
  //     title: $t('page.dashboard.workspace'),
  //   },
  //   name: 'Workspace',
  //   path: '/workspace',
  //   component: () => import('#/views/dashboard/workspace/index.vue'),
  // },
];

export default routes;
