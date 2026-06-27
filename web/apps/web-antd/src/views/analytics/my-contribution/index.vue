<template>
  <div class="p-5 space-y-5">
    <!-- 收益概览网格 -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
      <!-- 总计收益主卡片 -->
      <div class="lg:col-span-2 flex">
        <div class="relative rounded-2xl bg-gradient-to-br from-green-500 via-emerald-500 to-teal-600 border-2 border-green-300 shadow-2xl p-4 overflow-hidden flex-1">
          <div class="absolute top-0 right-0 w-32 h-32 bg-white/10 rounded-full -translate-y-16 translate-x-16"></div>
          <div class="absolute bottom-0 left-0 w-24 h-24 bg-white/5 rounded-full translate-y-12 -translate-x-12"></div>
          
          <div class="relative z-10">
            <!-- 顶部标题和金额 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2.5">
                <div class="flex items-center justify-center h-10 w-10 rounded-xl bg-white/20 backdrop-blur-sm border border-white/30">
                  <SvgCakeIcon class="h-5 w-5 text-white" />
                </div>
                <div>
                  <h2 class="text-base font-bold text-white leading-tight">💰 总计收益</h2>
                  <p class="text-white/60 text-xs leading-tight">累计收入总额</p>
                </div>
              </div>
              <div class="text-right">
                <p class="text-3xl font-black text-white drop-shadow-lg">
                  <span v-if="loading" class="inline-block animate-pulse bg-white/20 rounded-lg h-9 w-32"></span>
                  <span v-else>¥{{ timeStatsData.total.income.toFixed(4) }}</span>
                </p>
              </div>
            </div>
            
            <!-- 收益指标 -->
            <div class="mb-3">
              <div class="text-xs font-semibold text-white/80 mb-1.5 uppercase tracking-wide">💵 收益指标</div>
              <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">总收益</p>
                  <p class="text-sm font-bold text-white">¥{{ timeStatsData.total.income.toFixed(4) }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">平均单次</p>
                  <p class="text-sm font-bold text-white">¥{{ averageIncomePerCall.toFixed(6) }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">最高单次</p>
                  <p class="text-sm font-bold text-white">¥{{ maxIncomePerCall.toFixed(6) }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">最低单次</p>
                  <p class="text-sm font-bold text-white">¥{{ minIncomePerCall.toFixed(6) }}</p>
                </div>
              </div>
            </div>
            
            <!-- 调用统计 -->
            <div class="mb-3">
              <div class="text-xs font-semibold text-white/80 mb-1.5 uppercase tracking-wide">📞 调用统计</div>
              <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">总调用次数</p>
                  <p class="text-sm font-bold text-white">{{ timeStatsData.total.calls.toLocaleString() }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">成功率</p>
                  <p class="text-sm font-bold text-white">{{ successRate.toFixed(1) }}%</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">活跃客户端</p>
                  <p class="text-sm font-bold text-white">{{ uniqueClientsCount }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">活跃用户数</p>
                  <p class="text-sm font-bold text-white">{{ uniqueUsersCount }}</p>
                </div>
              </div>
            </div>
            
            <!-- Token统计 -->
            <div class="mb-3">
              <div class="text-xs font-semibold text-white/80 mb-1.5 uppercase tracking-wide">🎯 Token统计</div>
              <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">总Token</p>
                  <p class="text-sm font-bold text-white">{{ timeStatsData.total.totalTokens.toLocaleString() }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">输入Token</p>
                  <p class="text-sm font-bold text-white">{{ timeStatsData.total.inputTokens.toLocaleString() }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">输出Token</p>
                  <p class="text-sm font-bold text-white">{{ timeStatsData.total.outputTokens.toLocaleString() }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">平均Token</p>
                  <p class="text-sm font-bold text-white">{{ averageTokensPerCall.toLocaleString() }}</p>
                </div>
              </div>
            </div>
            
            <!-- 模型统计 -->
            <div class="mb-3">
              <div class="text-xs font-semibold text-white/80 mb-1.5 uppercase tracking-wide">🤖 模型统计</div>
              <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">活跃模型数</p>
                  <p class="text-sm font-bold text-white">{{ timeStatsData.total.models }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">最热模型</p>
                  <p class="text-sm font-bold text-white truncate" :title="topModel">{{ topModel }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">最赚钱模型</p>
                  <p class="text-sm font-bold text-white truncate" :title="topIncomeModel">{{ topIncomeModel }}</p>
                </div>
                <div class="bg-white/10 backdrop-blur-sm rounded-lg p-2 border border-white/20">
                  <p class="text-xs font-medium text-white/70 mb-0.5">平均模型价格</p>
                  <p class="text-sm font-bold text-white">¥{{ averageModelPrice.toFixed(6) }}</p>
                </div>
              </div>
            </div>
            
            <!-- 底部按钮 -->
            <div class="flex justify-end">
              <button 
                @click="showDetailTable = !showDetailTable"
                class="px-3 py-1.5 bg-white/15 hover:bg-white/25 rounded-lg border border-white/30 text-white text-sm font-medium transition-all duration-200 backdrop-blur-sm flex items-center gap-2"
              >
                <SvgCardIcon class="h-4 w-4" />
                <span>{{ showDetailTable ? '隐藏详单' : '查看详单' }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 时段收益卡片 -->
      <div class="flex flex-col gap-2.5">
        <!-- 今日收益 -->
        <div class="rounded-xl bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950/30 dark:to-blue-900/20 border border-blue-200 dark:border-blue-800 p-2.5 shadow-md flex-1">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-semibold text-blue-600 dark:text-blue-400 uppercase tracking-wide">今日收益</span>
            <span class="text-xs px-1.5 py-0.5 rounded-full bg-blue-200 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300">Today</span>
          </div>
          <p class="text-xl font-bold text-blue-900 dark:text-blue-100 mb-1.5">
            <span v-if="loading" class="inline-block animate-pulse bg-blue-200 dark:bg-blue-800 rounded h-7 w-24"></span>
            <span v-else>¥{{ timeStatsData.today.income.toFixed(4) }}</span>
          </p>
          <div class="flex items-center justify-between text-xs text-blue-700 dark:text-blue-300">
            <span>{{ timeStatsData.today.calls }} 次调用</span>
            <span>{{ timeStatsData.today.totalTokens.toLocaleString() }} tokens</span>
          </div>
        </div>

        <!-- 本周收益 -->
        <div class="rounded-xl bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950/30 dark:to-purple-900/20 border border-purple-200 dark:border-purple-800 p-2.5 shadow-md flex-1">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-semibold text-purple-600 dark:text-purple-400 uppercase tracking-wide">近7日收益</span>
            <span class="text-xs px-1.5 py-0.5 rounded-full bg-purple-200 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300">7 Days</span>
          </div>
          <p class="text-xl font-bold text-purple-900 dark:text-purple-100 mb-1.5">
            <span v-if="loading" class="inline-block animate-pulse bg-purple-200 dark:bg-purple-800 rounded h-7 w-24"></span>
            <span v-else>¥{{ timeStatsData.week.income.toFixed(4) }}</span>
          </p>
          <div class="flex items-center justify-between text-xs text-purple-700 dark:text-purple-300">
            <span>{{ timeStatsData.week.calls }} 次调用</span>
            <span>{{ timeStatsData.week.totalTokens.toLocaleString() }} tokens</span>
          </div>
        </div>

        <!-- 本月收益 -->
        <div class="rounded-xl bg-gradient-to-br from-emerald-50 to-emerald-100 dark:from-emerald-950/30 dark:to-emerald-900/20 border border-emerald-200 dark:border-emerald-800 p-2.5 shadow-md flex-1">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-semibold text-emerald-600 dark:text-emerald-400 uppercase tracking-wide">本月收益</span>
            <span class="text-xs px-1.5 py-0.5 rounded-full bg-emerald-200 dark:bg-emerald-900/50 text-emerald-700 dark:text-emerald-300">This Month</span>
          </div>
          <p class="text-xl font-bold text-emerald-900 dark:text-emerald-100 mb-1.5">
            <span v-if="loading" class="inline-block animate-pulse bg-emerald-200 dark:bg-emerald-800 rounded h-7 w-24"></span>
            <span v-else>¥{{ timeStatsData.month.income.toFixed(4) }}</span>
          </p>
          <div class="flex items-center justify-between text-xs text-emerald-700 dark:text-emerald-300">
            <span>{{ timeStatsData.month.calls }} 次调用</span>
            <span>{{ timeStatsData.month.totalTokens.toLocaleString() }} tokens</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 收益详单表格 -->
    <div v-if="showDetailTable" class="mt-5">
      <AnalysisChartCard title="收益详单">
        <div class="overflow-x-auto">
          <div v-if="loading" class="p-4">
            <div class="animate-pulse space-y-3">
              <div class="bg-[var(--bg-color-secondary)] rounded h-8"></div>
              <div v-for="i in 10" :key="i" class="bg-[var(--bg-color-secondary)] rounded h-12"></div>
            </div>
          </div>
          <div v-else-if="incomeData.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
            <div class="mb-2">暂无收益详单</div>
            <div class="text-xs text-[var(--text-tertiary)]">请确保已有收益记录</div>
          </div>
          <table v-else class="min-w-full divide-y divide-[var(--border-color)]">
            <thead class="bg-[var(--bg-color-secondary)]">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  时间
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  模型
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  输入Token
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  缓存命中
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  输出Token
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  总Token
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  IPPM
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  OPPM
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  CIPPM
                </th>
                <th class="px-6 py-3 text-right text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  未命中输入收益
                </th>
                <th class="px-6 py-3 text-right text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  缓存命中收益
                </th>
                <th class="px-6 py-3 text-right text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  输出收益
                </th>
                <th class="px-6 py-3 text-right text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  总收益
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  调用者
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  请求ID
                </th>
              </tr>
            </thead>            <tbody class="bg-[var(--content-bg)] divide-y divide-[var(--border-color)]">
              <tr v-for="record in paginatedIncomeData" :key="record.ID" class="hover:bg-[var(--bg-color-secondary)] transition-colors">
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-[var(--text-primary)]">
                    {{ formatTimestamp(record.Timestamp) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium text-[var(--text-primary)]">
                    {{ record.Model }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-blue-600">
                    {{ record.InputTokens.toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm" :class="(record.CachedTokens || 0) > 0 ? 'text-orange-500 font-medium' : 'text-[var(--text-secondary)]'">
                    {{ (record.CachedTokens || 0).toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-green-600 font-medium">
                    {{ record.OutputTokens.toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-indigo-600">
                    {{ record.TotalTokens.toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium text-orange-600">
                    {{ record.IPPM.toFixed(2) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium text-orange-600">
                    {{ record.OPPM.toFixed(2) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium" :class="(record.CIPPM || 0) > 0 ? 'text-orange-500' : 'text-[var(--text-secondary)]'">
                    {{ (record.CIPPM || 0).toFixed(2) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="text-sm text-blue-500">
                    ¥{{ calcUncachedInputIncome(record).toFixed(6) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="text-sm" :class="calcCachedInputIncome(record) > 0 ? 'text-orange-500' : 'text-[var(--text-secondary)]'">
                    ¥{{ calcCachedInputIncome(record).toFixed(6) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="text-sm text-green-500">
                    ¥{{ calcOutputIncome(record).toFixed(6) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="text-sm font-bold text-emerald-600" :title="`(输入${record.InputTokens}-缓存${record.CachedTokens || 0})×IPPM${record.IPPM} + 缓存${record.CachedTokens || 0}×CIPPM${record.CIPPM || 0} + 输出${record.OutputTokens}×OPPM${record.OPPM}) / 1000000`">
                    ¥{{ calculateSingleCallIncome(record).toFixed(6) }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-[var(--text-secondary)] font-mono">
                    {{ record.UserID }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-xs text-[var(--text-tertiary)] font-mono max-w-[120px] truncate" :title="record.RequestID">
                    {{ record.RequestID }}
                  </div>
                </td>
              </tr>
            </tbody>
          </table>          <!-- 分页信息和控件 -->
          <div v-if="sortedIncomeData.length > 0" class="px-6 py-4 bg-[var(--bg-color-secondary)] border-t border-[var(--border-color)]">
            <div class="flex justify-between items-center">
              <div class="text-sm text-[var(--text-secondary)]">
                已加载 {{ sortedIncomeData.length }} / {{ totalRecords }} 条记录（按时间倒序）
              </div>
              <div class="text-sm text-[var(--text-secondary)]">
                收益计算: ((输入tokens - 缓存命中) × IPPM + 缓存命中 × CIPPM + 输出tokens × OPPM) / 1,000,000
              </div>
            </div>
            <!-- 加载更多按钮（懒加载） -->
            <div v-if="currentPage < totalPages" class="flex justify-center items-center mt-4">
              <button 
                @click="loadMoreIncome" 
                :disabled="loadingMore"
                class="px-4 py-2 text-sm rounded-lg bg-blue-500 hover:bg-blue-600 text-white font-medium transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                <div v-if="loadingMore" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                <span>{{ loadingMore ? '加载中...' : '加载更多' }}</span>
              </button>
            </div>
            <div v-else-if="totalRecords > 0" class="flex justify-center items-center mt-4">
              <span class="text-sm text-[var(--text-tertiary)]">已加载全部记录</span>
            </div>
          </div>
        </div>
      </AnalysisChartCard>
    </div>

    <!-- 时间趋势图 -->
    <div class="mt-5">
      <AnalysisChartCard title="收益趋势">
        <div class="flex justify-end mb-3">
          <a-range-picker
            :value="trendDateRange"
            @change="onTrendDateChange"
            :allow-clear="false"
            size="small"
          />
        </div>
        <div class="h-96 p-4 relative">
          <EchartsUI 
            ref="trendChartRef" 
            class="w-full h-full" 
          />
          <div v-if="trendLoading" class="flex items-center justify-center absolute inset-0">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
          <div v-if="!trendLoading && dailyIncomeData.length === 0" class="flex items-center justify-center absolute inset-0 text-[var(--text-secondary)]">
            <p>暂无趋势数据</p>
          </div>
        </div>
      </AnalysisChartCard>
    </div>

    <!-- 按模型收益柱状图 + 模型收益详情（按钮控制） -->
    <div class="mt-5 relative">
      <AnalysisChartCard title="按模型收益统计">
        <!-- 标题行右侧按钮：显示/隐藏模型详情（绝对定位对齐卡片标题高度） -->
        <div class="absolute top-4 right-4 z-10">
          <button 
            @click="toggleModelDetail"
            class="px-3 py-1.5 bg-blue-500 hover:bg-blue-600 rounded-lg text-white text-sm font-medium transition-all duration-200 flex items-center gap-2"
          >
            <SvgCardIcon class="h-4 w-4" />
            <span>{{ showModelDetail ? '隐藏模型详情' : '显示模型详情' }}</span>
          </button>
        </div>
        <!-- 柱状图 -->
        <div class="h-96 p-4 relative">
          <EchartsUI 
            ref="incomeChartRef" 
            class="w-full h-full" 
          />
          <div v-if="modelLoading" class="flex items-center justify-center absolute inset-0">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
          <div v-if="!modelLoading && modelIncomeData.length === 0" class="flex items-center justify-center absolute inset-0 text-[var(--text-secondary)]">
            <p>暂无收益数据</p>
          </div>
        </div>
        <!-- 模型收益详情表格（按钮控制，显示在柱状图下方） -->
        <div v-if="showModelDetail" class="mt-4">
          <div class="overflow-x-auto">
            <div v-if="modelLoading" class="p-4">
              <div class="animate-pulse space-y-3">
                <div class="bg-[var(--bg-color-secondary)] rounded h-8"></div>
                <div v-for="i in 5" :key="i" class="bg-[var(--bg-color-secondary)] rounded h-12"></div>
              </div>
            </div>
            <div v-else-if="modelStatsData.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
              <div class="mb-2">暂无模型统计数据</div>
              <div class="text-xs text-[var(--text-tertiary)]">请确保已有收益记录</div>
            </div>
          <table v-else class="min-w-full divide-y divide-[var(--border-color)]">
            <thead class="bg-[var(--bg-color-secondary)]">
              <tr>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  模型名称
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  输入Tokens
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  输出Tokens
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  总Tokens
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  收益
                </th>                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  客户端数
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  调用次数
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                  成功率
                </th>
              </tr>
            </thead>
            <tbody class="bg-[var(--content-bg)] divide-y divide-[var(--border-color)]">
              <tr v-for="model in modelStats" :key="model.name" class="hover:bg-[var(--bg-color-secondary)] transition-colors">
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="flex items-center">
                    <div class="text-sm font-medium text-[var(--text-primary)]">
                      {{ model.name }}
                    </div>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-blue-600 font-medium">
                    {{ model.inputTokens.toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-green-600 font-medium">
                    {{ model.outputTokens.toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-indigo-600 font-medium">
                    {{ model.totalTokens.toLocaleString() }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-green-600 font-bold">
                    ¥{{ model.income.toFixed(4) }}
                  </div>
                </td>                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-[var(--text-primary)]">
                    {{ model.clientCount }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-[var(--text-primary)]">
                    {{ model.calls }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium" :class="model.successRate >= 95 ? 'text-green-600' : model.successRate >= 80 ? 'text-yellow-600' : 'text-red-600'">
                    {{ model.successRate.toFixed(1) }}%
                  </div>
                </td>
              </tr>            </tbody>
          </table>
          </div>
        </div>
      </AnalysisChartCard>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { requestClient } from '#/api/request'
import { useEcharts, EchartsUI } from '@vben/plugins/echarts'
import {
  AnalysisChartCard,
} from '@vben/common-ui'
import {
  SvgCakeIcon,
  SvgCardIcon,
} from '@vben/icons'
import { RangePicker as ARangePicker } from 'ant-design-vue'
import dayjs, { type Dayjs } from 'dayjs'

interface IncomeRecord {
  ID: number
  RequestID: string
  UserID: string
  APIKey: string
  ClientID: string
  ClientIP: string
  Model: string
  IPPM: number
  OPPM: number
  CIPPM: number
  InputTokens: number
  OutputTokens: number
  CachedTokens: number
  TotalTokens: number
  Timestamp: string
}

interface TotalIncomeStats {
  total_income: number
  total_calls: number
  input_tokens: number
  output_tokens: number
  cached_tokens: number
  total_tokens: number
  models: number
  client_count: number
  unique_users: number
}

// 时段收益统计（来自 /income/stats，服务端聚合，避免详单分页导致今日/近7日/本月数值相同）
interface TimeRangeStats {
  total_income: number
  total_calls: number
  input_tokens: number
  output_tokens: number
  cached_tokens: number
  total_tokens: number
  models: number
}

interface TrendPoint {
  date: string
  income: number
  calls: number
}

interface ModelIncomeStat {
  model: string
  input_tokens: number
  output_tokens: number
  cached_tokens: number
  total_tokens: number
  income: number
  calls: number
  client_count: number
}

const loading = ref(false)
const incomeData = ref<IncomeRecord[]>([])
const showDetailTable = ref(false) // 控制详单表格显示
const currentPage = ref(1) // 当前页
const pageSize = ref(50) // 每页显示数量
const totalRecords = ref(0) // 服务端返回的总记录数
const loadingMore = ref(false) // 加载更多状态

// 总计收益数据（来自 /income/total，真·总计）
const totalStatsData = ref<TotalIncomeStats>({
  total_income: 0,
  total_calls: 0,
  input_tokens: 0,
  output_tokens: 0,
  cached_tokens: 0,
  total_tokens: 0,
  models: 0,
  client_count: 0,
  unique_users: 0,
})

// 时段收益数据（来自 /income/stats，服务端按时间段聚合）
const emptyTimeRangeStats = (): TimeRangeStats => ({
  total_income: 0,
  total_calls: 0,
  input_tokens: 0,
  output_tokens: 0,
  cached_tokens: 0,
  total_tokens: 0,
  models: 0,
})
const todayStatsData = ref<TimeRangeStats>(emptyTimeRangeStats())
const weekStatsData = ref<TimeRangeStats>(emptyTimeRangeStats())
const monthStatsData = ref<TimeRangeStats>(emptyTimeRangeStats())

// 趋势数据（来自 /income/trend，按天聚合）
const trendData = ref<TrendPoint[]>([])
const trendLoading = ref(false)
// 趋势图日期范围（默认最近 90 天，用 dayjs 对象数组供 a-range-picker 使用）
const trendDateRange = ref<[Dayjs, Dayjs]>([
  dayjs().subtract(90, 'day'),
  dayjs(),
])

// 模型收益数据（来自 /income/models，按模型聚合）
const modelStatsData = ref<ModelIncomeStat[]>([])
const modelLoading = ref(false)
const showModelDetail = ref(false) // 控制模型收益详情显示（默认不加载）

// 图表引用
const incomeChartRef = ref()
const trendChartRef = ref()
const { renderEcharts: renderIncomeChart } = useEcharts(incomeChartRef)
const { renderEcharts: renderTrendChart } = useEcharts(trendChartRef)

// 计算单次调用收益（支持缓存命中分离计费）
const calculateSingleCallIncome = (record: IncomeRecord): number => {
  const cachedTokens = record.CachedTokens || 0
  const cippm = record.CIPPM || 0
  const nonCachedInputTokens = record.InputTokens - cachedTokens
  return (nonCachedInputTokens * record.IPPM + cachedTokens * cippm + record.OutputTokens * record.OPPM) / 1000000
}

// 收益分项计算
const calcUncachedInputIncome = (record: IncomeRecord): number => {
  const cached = record.CachedTokens || 0
  return ((record.InputTokens - cached) * record.IPPM) / 1000000
}
const calcCachedInputIncome = (record: IncomeRecord): number => {
  return ((record.CachedTokens || 0) * (record.CIPPM || 0)) / 1000000
}
const calcOutputIncome = (record: IncomeRecord): number => {
  return (record.OutputTokens * record.OPPM) / 1000000
}

// 格式化时间戳
const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 计算时间段统计数据（today/week/month 来自后端 /income/stats 服务端聚合，total 用后端总计接口）
const timeStatsData = computed(() => {
  // 今日数据 — 来自后端 /income/stats（今日 00:00 ~ 23:59:59）
  const todayStats = {
    totalTokens: todayStatsData.value.total_tokens,
    inputTokens: todayStatsData.value.input_tokens,
    outputTokens: todayStatsData.value.output_tokens,
    calls: todayStatsData.value.total_calls,
    models: todayStatsData.value.models,
    income: todayStatsData.value.total_income,
  }

  // 近7日数据 — 来自后端 /income/stats（最近 7 天）
  const weekStats = {
    totalTokens: weekStatsData.value.total_tokens,
    inputTokens: weekStatsData.value.input_tokens,
    outputTokens: weekStatsData.value.output_tokens,
    calls: weekStatsData.value.total_calls,
    models: weekStatsData.value.models,
    income: weekStatsData.value.total_income,
  }

  // 本月数据 — 来自后端 /income/stats（本月 1 日至今）
  const monthStats = {
    totalTokens: monthStatsData.value.total_tokens,
    inputTokens: monthStatsData.value.input_tokens,
    outputTokens: monthStatsData.value.output_tokens,
    calls: monthStatsData.value.total_calls,
    models: monthStatsData.value.models,
    income: monthStatsData.value.total_income,
  }

  // 总计数据 — 来自后端 /income/total 接口（真·总计，不受 30 天限制）
  const totalStats = {
    totalTokens: totalStatsData.value.total_tokens,
    inputTokens: totalStatsData.value.input_tokens,
    outputTokens: totalStatsData.value.output_tokens,
    calls: totalStatsData.value.total_calls,
    models: totalStatsData.value.models,
    income: totalStatsData.value.total_income
  }

  return {
    today: todayStats,
    week: weekStats,
    month: monthStats,
    total: totalStats
  }
})

// 计算属性
const averageIncomePerCall = computed(() => {
  if (incomeData.value.length === 0) return 0
  return timeStatsData.value.total.income / timeStatsData.value.total.calls
})

const maxIncomePerCall = computed(() => {
  if (incomeData.value.length === 0) return 0
  return Math.max(...incomeData.value.map(record => calculateSingleCallIncome(record)))
})

const minIncomePerCall = computed(() => {
  if (incomeData.value.length === 0) return 0
  return Math.min(...incomeData.value.map(record => calculateSingleCallIncome(record)))
})

const averageTokensPerCall = computed(() => {
  if (timeStatsData.value.total.calls === 0) return 0
  return Math.round(timeStatsData.value.total.totalTokens / timeStatsData.value.total.calls)
})

const uniqueClientsCount = computed(() => {
  return new Set(incomeData.value.map(r => r.ClientID)).size
})

const uniqueUsersCount = computed(() => {
  return new Set(incomeData.value.map(r => r.UserID)).size
})

const topModel = computed(() => {
  if (modelStats.value.length === 0) return '-'
  const sorted = [...modelStats.value].sort((a: any, b: any) => b.calls - a.calls)
  return sorted[0]?.name || '-'
})

const topIncomeModel = computed(() => {
  if (modelStats.value.length === 0) return '-'
  return modelStats.value[0]?.name || '-'
})

const averageModelPrice = computed(() => {
  if (incomeData.value.length === 0) return 0
  const totalIPPM = incomeData.value.reduce((sum, r) => sum + r.IPPM, 0)
  const totalOPPM = incomeData.value.reduce((sum, r) => sum + r.OPPM, 0)
  return (totalIPPM + totalOPPM) / (incomeData.value.length * 2000000)
})

const successRate = computed(() => {
  // 假设所有记录都是成功的（因为实际数据中没有status字段）
  return 100
})

// 模型统计数据 — 来自后端 /income/models 接口（需点击按钮加载）
const modelStats = computed(() => {
  return modelStatsData.value.map(stat => ({
    name: stat.model,
    inputTokens: stat.input_tokens,
    outputTokens: stat.output_tokens,
    cachedTokens: stat.cached_tokens,
    totalTokens: stat.total_tokens,
    income: stat.income,
    calls: stat.calls,
    successCalls: stat.calls,
    clients: new Set(),
    clientCount: stat.client_count,
    averageResponseTime: 0,
    successRate: 100,
  }))
})

// 模型收益数据（用于图表）
const modelIncomeData = computed(() => {
  return modelStats.value.map(stat => ({
    name: stat.name,
    income: stat.income
  }))
})

// 每日收益数据 — 来自后端 /income/trend 接口（按天聚合）
const dailyIncomeData = computed(() => {
  return trendData.value.map(point => ({
    date: point.date,
    income: point.income
  }))
})

// 排序后的详单数据（按时间倒序）
const sortedIncomeData = computed(() => {
  return [...incomeData.value].sort((a, b) => 
    new Date(b.Timestamp).getTime() - new Date(a.Timestamp).getTime()
  )
})

// 分页后的详单数据（服务端分页，直接展示已加载数据）
const paginatedIncomeData = computed(() => {
  return sortedIncomeData.value
})

// 总页数（基于服务端返回的总记录数）
const totalPages = computed(() => {
  return Math.ceil(totalRecords.value / pageSize.value)
})

// 获取总计收益数据（真·总计，无时间过滤）
const fetchTotalIncome = async () => {
  try {
    const response = await requestClient.get('/user/income/total')
    if (response && typeof response === 'object') {
      totalStatsData.value = response as TotalIncomeStats
    }
  } catch (error) {
    console.error('Failed to load total income:', error)
  }
}

// 获取指定时间段的收益统计（服务端聚合）
const fetchIncomeStatsByRange = async (
  startDate: string,
  endDate: string,
): Promise<TimeRangeStats> => {
  try {
    const response = await requestClient.get('/user/income/stats', {
      params: { start_date: startDate, end_date: endDate },
    })
    if (response && typeof response === 'object') {
      return response as TimeRangeStats
    }
  } catch (error) {
    console.error('Failed to load income stats by range:', error)
  }
  return emptyTimeRangeStats()
}

// 获取今日/近7日/本月收益统计（并行请求）
const fetchTimeRangeStats = async () => {
  const now = dayjs()
  const todayStart = now.startOf('day').format('YYYY-MM-DD')
  const todayEnd = now.endOf('day').format('YYYY-MM-DD')
  const weekStart = now.subtract(6, 'day').startOf('day').format('YYYY-MM-DD')
  const monthStart = now.startOf('month').format('YYYY-MM-DD')

  const [today, week, month] = await Promise.all([
    fetchIncomeStatsByRange(todayStart, todayEnd),
    fetchIncomeStatsByRange(weekStart, todayEnd),
    fetchIncomeStatsByRange(monthStart, todayEnd),
  ])
  todayStatsData.value = today
  weekStatsData.value = week
  monthStatsData.value = month
}

// 获取收益详单（分页，支持懒加载追加）
const fetchIncomeData = async (page: number = 1, append: boolean = false) => {
  try {
    if (append) {
      loadingMore.value = true
    } else {
      loading.value = true
    }
    const response = await requestClient.get('/user/income', {
      params: { page, size: pageSize.value }
    })
    
    if (response && Array.isArray(response.data)) {
      if (append) {
        incomeData.value = [...incomeData.value, ...response.data]
      } else {
        incomeData.value = response.data
      }
      totalRecords.value = response.total || response.data.length
      currentPage.value = page
    }
  } catch (error) {
    console.error('Failed to load income data:', error)
    if (!append) {
      incomeData.value = []
    }
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

// 加载更多详单（下一页追加）
const loadMoreIncome = () => {
  if (currentPage.value < totalPages.value) {
    fetchIncomeData(currentPage.value + 1, true)
  }
}

// 获取收益趋势数据（按天聚合，支持日期范围滑动）
const fetchTrendData = async (startDate?: string, endDate?: string) => {
  try {
    trendLoading.value = true
    const params: Record<string, string> = {}
    if (startDate) params.start_date = startDate
    if (endDate) params.end_date = endDate
    
    const response = await requestClient.get('/user/income/trend', { params })
    
    if (response && Array.isArray(response.data)) {
      trendData.value = response.data
    } else {
      trendData.value = []
    }
  } catch (error) {
    console.error('Failed to load trend data:', error)
    trendData.value = []
  } finally {
    trendLoading.value = false
  }
}

// 趋势图日期范围变更
const onTrendDateChange = (dates: [Dayjs, Dayjs] | null) => {
  if (dates && dates.length === 2) {
    const start = dates[0].format('YYYY-MM-DD')
    const end = dates[1].format('YYYY-MM-DD')
    trendDateRange.value = [dates[0], dates[1]]
    fetchTrendData(start, end)
  }
}

// 获取模型收益详情（按模型聚合，点击按钮才加载）
const fetchModelStats = async () => {
  try {
    modelLoading.value = true
    const response = await requestClient.get('/user/income/models')
    
    if (response && Array.isArray(response.data)) {
      modelStatsData.value = response.data
    } else {
      modelStatsData.value = []
    }
  } catch (error) {
    console.error('Failed to load model stats:', error)
    modelStatsData.value = []
  } finally {
    modelLoading.value = false
  }
}

// 切换模型收益详情表格显示（数据已在 onMounted 时加载，按钮只控制显隐）
const toggleModelDetail = () => {
  showModelDetail.value = !showModelDetail.value
}

// 更新收益图表
const updateIncomeChart = () => {
  if (modelIncomeData.value.length === 0) return
  
  const option = {
    title: {
      text: '按模型收益统计',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'normal' as const
      }
    },
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: {
        type: 'shadow' as const
      },
      formatter: function(params: any) {
        const data = params[0]
        const modelData = modelStats.value.find(m => m.name === data.name)
        return `
          <div style="font-weight: bold; margin-bottom: 4px;">${data.name}</div>
          <div>收益: ¥${data.value.toFixed(4)}</div>
          <div>输出Token: ${modelData?.outputTokens.toLocaleString()}</div>
          <div>调用次数: ${modelData?.calls}</div>
        `
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category' as const,
      data: modelIncomeData.value.map(item => item.name),
      axisLabel: {
        interval: 0,
        rotate: modelIncomeData.value.length > 3 ? 45 : 0,
        fontSize: 12
      }
    },
    yAxis: {
      type: 'value' as const,
      name: '收益 (¥)',
      axisLabel: {
        formatter: '¥{value}'
      }
    },
    series: [
      {
        name: '收益',
        type: 'bar' as const,
        data: modelIncomeData.value.map(item => ({
          value: item.income,
          name: item.name,
          itemStyle: {
            color: '#10B981'
          }
        })),
        markLine: {
          data: [
            { type: 'average' as const, name: '平均值' }
          ]
        }
      }
    ]
  }
  
  renderIncomeChart(option)
}

// 更新趋势图表
const updateTrendChart = () => {
  if (dailyIncomeData.value.length === 0) return
  
  const option = {
    title: {
      text: '收益趋势',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'normal' as const
      }
    },
    tooltip: {
      trigger: 'axis' as const,
      formatter: function(params: any) {
        const data = params[0]
        return `${data.axisValue}<br/>收益: ¥${data.value.toFixed(4)}`
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category' as const,
      data: dailyIncomeData.value.map(item => item.date),
      axisLabel: {
        formatter: function(value: string) {
          return value.substring(5) // 显示 MM-DD
        }
      }
    },
    yAxis: {
      type: 'value' as const,
      name: '收益 (¥)',
      axisLabel: {
        formatter: '¥{value}'
      }
    },
    series: [
      {
        name: '每日收益',
        type: 'line' as const,
        data: dailyIncomeData.value.map(item => item.income),
        smooth: true,
        lineStyle: {
          color: '#3b82f6'
        },
        areaStyle: {
          color: {
            type: 'linear' as const,
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
              { offset: 1, color: 'rgba(59, 130, 246, 0.1)' }
            ]
          }
        }
      }
    ]
  }
  
  renderTrendChart(option)
}

// 监听模型收益数据变化，自动更新柱状图
watch(modelIncomeData, () => {
  nextTick(() => {
    updateIncomeChart()
  })
}, { deep: true })

// 监听趋势数据变化，自动更新趋势图
watch(dailyIncomeData, () => {
  nextTick(() => {
    updateTrendChart()
  })
}, { deep: true })

onMounted(() => {
  // 并行加载：总计收益 + 时段收益(今日/近7日/本月) + 详单首页 + 趋势图 + 模型收益柱状图
  fetchTotalIncome()
  fetchTimeRangeStats()
  fetchIncomeData(1)
  fetchTrendData(
    trendDateRange.value[0].format('YYYY-MM-DD'),
    trendDateRange.value[1].format('YYYY-MM-DD'),
  )
  fetchModelStats()
})
</script>

<style scoped>
/* 确保表格样式与系统主题一致 */
.hover\:bg-\[var\(--bg-color-secondary\)\]:hover {
  background-color: var(--bg-color-secondary);
}

/* 动画效果 */
.transition-colors {
  transition-property: color, background-color, border-color, text-decoration-color, fill, stroke;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  transition-duration: 150ms;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .grid-cols-1.md\:grid-cols-4 {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }
  
  .grid-cols-1.md\:grid-cols-6 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

/* 表格滚动 */
.overflow-x-auto {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

/* 加载动画 */
.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: .5;
  }
}

.animate-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
