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
                共 {{ sortedIncomeData.length }} 条记录（按时间倒序）
              </div>
              <div class="text-sm text-[var(--text-secondary)]">
                收益计算: ((输入tokens - 缓存命中) × IPPM + 缓存命中 × CIPPM + 输出tokens × OPPM) / 1,000,000
              </div>
            </div>
            <!-- 分页控件 -->
            <div v-if="totalPages > 1" class="flex justify-center items-center mt-4 space-x-2">
              <button 
                @click="currentPage = 1" 
                :disabled="currentPage === 1"
                class="px-3 py-1 text-sm rounded bg-[var(--content-bg)] border border-[var(--border-color)] disabled:opacity-50 disabled:cursor-not-allowed hover:bg-[var(--bg-color-secondary)]"
              >
                首页
              </button>
              <button 
                @click="currentPage--" 
                :disabled="currentPage === 1"
                class="px-3 py-1 text-sm rounded bg-[var(--content-bg)] border border-[var(--border-color)] disabled:opacity-50 disabled:cursor-not-allowed hover:bg-[var(--bg-color-secondary)]"
              >
                上一页
              </button>
              <span class="px-3 py-1 text-sm">
                第 {{ currentPage }} / {{ totalPages }} 页
              </span>
              <button 
                @click="currentPage++" 
                :disabled="currentPage === totalPages"
                class="px-3 py-1 text-sm rounded bg-[var(--content-bg)] border border-[var(--border-color)] disabled:opacity-50 disabled:cursor-not-allowed hover:bg-[var(--bg-color-secondary)]"
              >
                下一页
              </button>
              <button 
                @click="currentPage = totalPages" 
                :disabled="currentPage === totalPages"
                class="px-3 py-1 text-sm rounded bg-[var(--content-bg)] border border-[var(--border-color)] disabled:opacity-50 disabled:cursor-not-allowed hover:bg-[var(--bg-color-secondary)]"
              >
                末页
              </button>
            </div>
          </div>
        </div>
      </AnalysisChartCard>
    </div>

    <!-- 按模型收益柱状图 -->
    <div class="mt-5">
      <AnalysisChartCard title="按模型收益统计">
        <div class="h-96 p-4">
          <EchartsUI 
            v-if="modelIncomeData.length > 0 && !loading" 
            ref="incomeChartRef" 
            class="w-full h-full" 
          />
          <div v-else-if="loading" class="flex items-center justify-center h-full">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
          <div v-else class="flex items-center justify-center h-full text-[var(--text-secondary)]">
            <p>暂无收益数据</p>
          </div>
        </div>
      </AnalysisChartCard>
    </div>

    <!-- 时间趋势图 -->
    <div class="mt-5">
      <AnalysisChartCard title="收益趋势">
        <div class="h-96 p-4">
          <EchartsUI 
            v-if="dailyIncomeData.length > 0 && !loading" 
            ref="trendChartRef" 
            class="w-full h-full" 
          />
          <div v-else-if="loading" class="flex items-center justify-center h-full">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
          <div v-else class="flex items-center justify-center h-full text-[var(--text-secondary)]">
            <p>暂无趋势数据</p>
          </div>
        </div>
      </AnalysisChartCard>
    </div>

    <!-- 概览统计（表格形式） -->
    <div class="mt-5">
      <AnalysisChartCard title="模型收益详情">
        <div class="overflow-x-auto">
          <div v-if="loading" class="p-4">
            <div class="animate-pulse space-y-3">
              <div class="bg-[var(--bg-color-secondary)] rounded h-8"></div>
              <div v-for="i in 5" :key="i" class="bg-[var(--bg-color-secondary)] rounded h-12"></div>
            </div>
          </div>
          <div v-else-if="incomeData.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
            <div class="mb-2">暂无收益数据</div>
            <div class="text-xs text-[var(--text-tertiary)]">请确保已有收益记录</div>
          </div>
          <div v-else-if="modelStats.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
            <div class="mb-2">暂无模型统计数据</div>
            <div class="text-xs text-[var(--text-tertiary)]">已获取{{ incomeData.length }}条记录，但无法生成统计</div>
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

const loading = ref(false)
const incomeData = ref<IncomeRecord[]>([])
const showDetailTable = ref(false) // 控制详单表格显示
const currentPage = ref(1) // 当前页
const pageSize = ref(50) // 每页显示数量

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

// 计算时间段统计数据
const timeStatsData = computed(() => {
  const records = incomeData.value
  const now = new Date()
  const today = now.toISOString().split('T')[0]
  const weekStart = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)
  
  // 今日数据
  const todayRecords = records.filter(r => r.Timestamp && r.Timestamp.startsWith(today || ''))
  const todayStats = {
    totalTokens: todayRecords.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: todayRecords.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: todayRecords.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: todayRecords.length,
    models: new Set(todayRecords.map(r => r.Model)).size,
    income: todayRecords.reduce((sum, r) => sum + calculateSingleCallIncome(r), 0)
  }
  
  // 本周数据
  const weekRecords = records.filter(r => new Date(r.Timestamp) >= weekStart)
  const weekStats = {
    totalTokens: weekRecords.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: weekRecords.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: weekRecords.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: weekRecords.length,
    models: new Set(weekRecords.map(r => r.Model)).size,
    income: weekRecords.reduce((sum, r) => sum + calculateSingleCallIncome(r), 0)
  }
  
  // 本月数据
  const monthRecords = records.filter(r => new Date(r.Timestamp) >= monthStart)
  const monthStats = {
    totalTokens: monthRecords.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: monthRecords.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: monthRecords.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: monthRecords.length,
    models: new Set(monthRecords.map(r => r.Model)).size,
    income: monthRecords.reduce((sum, r) => sum + calculateSingleCallIncome(r), 0)
  }
  
  // 总计数据
  const totalStats = {
    totalTokens: records.reduce((sum, r) => sum + r.TotalTokens, 0),
    inputTokens: records.reduce((sum, r) => sum + r.InputTokens, 0),
    outputTokens: records.reduce((sum, r) => sum + r.OutputTokens, 0),
    calls: records.length,
    models: new Set(records.map(r => r.Model)).size,
    income: records.reduce((sum, r) => sum + calculateSingleCallIncome(r), 0)
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

// 模型统计数据
const modelStats = computed(() => {
  const stats: Record<string, any> = {}
  
  incomeData.value.forEach(record => {
    if (!stats[record.Model]) {
      stats[record.Model] = {
        name: record.Model,
        inputTokens: 0,
        outputTokens: 0,
        cachedTokens: 0,
        totalTokens: 0,
        income: 0,
        calls: 0,
        successCalls: 0,
        clients: new Set()
      }
    }
    
    const stat = stats[record.Model]
    stat.inputTokens += record.InputTokens
    stat.outputTokens += record.OutputTokens
    stat.cachedTokens += record.CachedTokens || 0
    stat.totalTokens += record.TotalTokens
    stat.income += calculateSingleCallIncome(record)
    stat.calls += 1
    stat.successCalls += 1
    stat.clients.add(record.ClientID)
  })
  
  return Object.values(stats).map((stat: any) => ({
    ...stat,
    averageResponseTime: 0, // API中没有响应时间数据
    successRate: 100, // 假设100%成功率
    clientCount: stat.clients.size
  })).sort((a: any, b: any) => b.income - a.income)
})

// 模型收益数据（用于图表）
const modelIncomeData = computed(() => {
  return modelStats.value.map(stat => ({
    name: stat.name,
    income: stat.income
  }))
})

// 每日收益数据（用于趋势图）
const dailyIncomeData = computed(() => {
  const daily: Record<string, number> = {}
  
  incomeData.value.forEach(record => {
    const date = record.Timestamp?.split('T')[0]
    if (date) {
      daily[date] = (daily[date] || 0) + calculateSingleCallIncome(record)
    }
  })
  
  return Object.entries(daily)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([date, income]) => ({ date, income }))
})

// 排序后的详单数据（按时间倒序）
const sortedIncomeData = computed(() => {
  return [...incomeData.value].sort((a, b) => 
    new Date(b.Timestamp).getTime() - new Date(a.Timestamp).getTime()
  )
})

// 分页后的详单数据
const paginatedIncomeData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return sortedIncomeData.value.slice(start, end)
})

// 总页数
const totalPages = computed(() => {
  return Math.ceil(sortedIncomeData.value.length / pageSize.value)
})

// 获取收益数据
const fetchIncomeData = async () => {
  try {
    loading.value = true
    console.log('正在获取收益数据...')
    const response = await requestClient.get('/user/income')
    console.log('收益API响应:', response)
    
    if (response && response.data && Array.isArray(response.data)) {
      incomeData.value = response.data
      console.log('获取到收益记录:', incomeData.value.length, '条')
      console.log('样本数据:', incomeData.value.slice(0, 2))
    } else if (Array.isArray(response)) {
      incomeData.value = response
      console.log('获取到收益记录(直接数组):', incomeData.value.length, '条')
    } else {
      console.warn('收益数据格式不正确:', response)
      incomeData.value = []
    }
  } catch (error) {
    console.error('Failed to load income data:', error)
    incomeData.value = []
  } finally {
    loading.value = false
    console.log('最终incomeData:', incomeData.value.length, '条')
  }
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

// 监听数据变化，自动更新图表
watch(modelIncomeData, () => {
  nextTick(() => {
    updateIncomeChart()
  })
}, { deep: true })

watch(dailyIncomeData, () => {
  nextTick(() => {
    updateTrendChart()
  })
}, { deep: true })

onMounted(() => {
  fetchIncomeData().then(() => {
    // 数据加载完成后，延迟渲染图表
    nextTick(() => {
      setTimeout(() => {
        console.log('timeStatsData:', timeStatsData.value)
        console.log('modelStats:', modelStats.value)
        console.log('dailyIncomeData:', dailyIncomeData.value)
        updateIncomeChart()
        updateTrendChart()
      }, 500)
    })
  })
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
