import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  // 模型广场（原控制台）
  {
    meta: {
      affixTab: true,
      icon: 'lucide:shopping-cart',
      order: 0,
      title: '模型广场',
    },
    name: 'ModelMarketplace',
    path: '/model-marketplace',
    component: () => import('#/views/modelmarket/index.vue'),
  },
  // 模型详情页
  {
    meta: {
      hideInMenu: true,
      title: '模型详情',
    },
    name: 'ModelMarketplaceDetail',
    path: '/model-marketplace-detail',
    component: () => import('#/views/model-marketplace-detail/index.vue'),
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
  // API Key 管理
  {
    meta: {
      icon: 'lucide:key',
      order: 1,
      title: 'API Key 管理',
    },
    name: 'MyKeys',
    path: '/my-keys',
    component: () => import('#/views/apikey/index.vue'),
  },
  // 使用分析 - 改为父菜单
  {
    meta: {
      icon: 'lucide:area-chart',
      order: 2,
      title: '使用分析',
    },
    name: 'UsageAnalytics',
    path: '/usage-analytics',
    redirect: '/usage-analytics/my-contribution',
    children: [
      {
        meta: {
          title: '我的收益',
        },
        name: 'MyContribution',
        path: '/usage-analytics/my-contribution',
        component: () => import('#/views/analytics/my-contribution/index.vue'),
      },
      {
        meta: {
          title: '我的使用',
        },
        name: 'MyUsage',
        path: '/usage-analytics/my-usage',
        component: () => import('#/views/analytics/my-usage/index.vue'),
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
