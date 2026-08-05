<script lang="ts" setup>
import type { AnalysisOverviewItem } from '@vben/common-ui';
import type { TabOption } from '@vben/types';

import {
  AnalysisChartCard,
  AnalysisChartsTabs,
  AnalysisOverview,
} from '@vben/common-ui';
import {
  SvgBellIcon,
  SvgCakeIcon,
  SvgCardIcon,
  SvgDownloadIcon,
} from '@vben/icons';
import { $t } from '#/locales';

import AnalyticsTrends from './components/analytics-trends.vue';
import AnalyticsVisitsData from './components/analytics-visits-data.vue';
import AnalyticsVisitsSales from './components/analytics-visits-sales.vue';
import AnalyticsVisitsSource from './components/analytics-visits-source.vue';
import AnalyticsVisits from './components/analytics-visits.vue';
import TokenUsage from './components/token-usage.vue';

const overviewItems: AnalysisOverviewItem[] = [
  {
    icon: SvgCardIcon,
    title: $t('business.analytics.overviewUsers'),
    totalTitle: $t('business.analytics.overviewTotalUsers'),
    totalValue: 120_000,
    value: 2000,
  },
  {
    icon: SvgCakeIcon,
    title: $t('business.analytics.overviewVisits'),
    totalTitle: $t('business.analytics.overviewTotalVisits'),
    totalValue: 500_000,
    value: 20_000,
  },
  {
    icon: SvgDownloadIcon,
    title: $t('business.analytics.overviewDownloads'),
    totalTitle: $t('business.analytics.overviewTotalDownloads'),
    totalValue: 120_000,
    value: 8000,
  },
  {
    icon: SvgBellIcon,
    title: $t('business.analytics.overviewTokenUsage'),
    totalTitle: $t('business.analytics.overviewTotalTokenUsage'),
    totalValue: 180_000,
    value: 12000,
  },
];

const chartTabs: TabOption[] = [
  {
    label: $t('business.analytics.tokenUsageAnalysis'),
    value: 'token-usage',
  },
  {
    label: $t('business.analytics.trafficTrend'),
    value: 'trends',
  },
  {
    label: $t('business.analytics.monthlyVisits'),
    value: 'visits',
  },
];
</script>

<template>
  <div>
  <div class="p-5">
    <AnalysisOverview :items="overviewItems" />
    <AnalysisChartsTabs :tabs="chartTabs" class="mt-5">
      <template #token-usage>
        <TokenUsage />
      </template>
      <template #trends>
        <AnalyticsTrends />
      </template>
      <template #visits>
        <AnalyticsVisits />
      </template>
    </AnalysisChartsTabs>

    <div class="mt-5 w-full md:flex">
      <AnalysisChartCard class="mt-5 md:mr-4 md:mt-0 md:w-1/3" :title="$t('business.analytics.visitVolume')">
        <AnalyticsVisitsData />
      </AnalysisChartCard>
      <AnalysisChartCard class="mt-5 md:mr-4 md:mt-0 md:w-1/3" :title="$t('business.analytics.visitSource')">
        <AnalyticsVisitsSource />
      </AnalysisChartCard>
      <AnalysisChartCard class="mt-5 md:mt-0 md:w-1/3" :title="$t('business.analytics.businessShare')">
        <AnalyticsVisitsSales />
      </AnalysisChartCard>
    </div>
  </div>
  </div>
</template>
