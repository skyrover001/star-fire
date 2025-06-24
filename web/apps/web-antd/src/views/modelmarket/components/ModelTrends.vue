<template>
  <div class="flex flex-col">
    <!-- 动态列表 -->
    <div class="space-y-3 pr-2">
      <div
        v-for="(item, index) in displayItems"
        :key="index"
        class="group relative overflow-hidden bg-[var(--content-bg)] rounded-lg p-4 transition-all duration-300 hover:bg-[var(--hover-bg)] hover:shadow-lg border border-[var(--border-color)] hover:border-purple-300/50"
      >
        <!-- 悬浮效果背景 -->
        <div class="absolute inset-0 bg-gradient-to-r from-purple-500/5 to-pink-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
        
        <div class="relative z-10 flex items-start space-x-2">
          <!-- 头像 -->
          <div class="flex-shrink-0">
            <div :class="getAvatarColor(index)" class="w-10 h-10 rounded-full flex items-center justify-center text-white font-semibold text-sm shadow-lg border-2 border-white/20">
              {{ getAvatarText(item.title) }}
            </div>
          </div>
          
          <!-- 内容 -->
          <div class="flex-1 min-w-0">
            <div class="flex items-start justify-between mb-2">
              <h4 class="text-sm font-semibold text-[var(--text-primary)] group-hover:text-purple-600 transition-colors duration-200 truncate flex-1">
                {{ item.title }}
              </h4>
              <span class="text-xs text-[var(--text-secondary)] flex items-center ml-3 flex-shrink-0">
                <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                {{ item.date }}
              </span>
            </div>
            
            <p class="text-sm text-[var(--text-secondary)] leading-relaxed line-clamp-2 mb-3" v-html="formatContent(item.content)"></p>
            
            <!-- 操作按钮 -->
            <div class="flex items-center space-x-4 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
              <button class="inline-flex items-center text-sm text-purple-500 hover:text-purple-600 transition-colors">
                <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"/>
                </svg>
                点赞
              </button>
              <button class="inline-flex items-center text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors">
                <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                </svg>
                评论
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 加载中指示器 -->
      <div v-if="loading" class="text-center py-4">
        <div class="inline-flex items-center text-xs text-[var(--text-secondary)]">
          <svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          正在加载更多...
        </div>
      </div>
    </div>
    
    <!-- 查看更多按钮 -->
    <div v-if="hasMore && !loading" class="text-center pt-4 border-t border-[var(--border-color)] mt-4">
      <button 
        @click="loadMore" 
        class="inline-flex items-center px-3 py-2 bg-[var(--content-bg)] hover:bg-[var(--hover-bg)] border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] rounded-lg text-xs font-medium transition-all duration-200"
      >
        <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
        </svg>
        查看更多
      </button>
    </div>

    <!-- 没有更多数据提示 -->
    <div v-if="!hasMore && displayItems.length > initialPageSize" class="text-center py-4">
      <span class="text-xs text-[var(--text-secondary)]">已显示全部动态</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
// 导入请求工具
import { requestClient } from '#/api/request';

// 模型动态数据
interface TrendItem {
  title: string;
  content: string;
  date: string;
}

// API 返回的动态数据类型 (根据 /market/trends 接口实际返回结构)
interface ApiTrendItem {
  id?: number | string;
  name?: string;          // 动态名称/标题
  description?: string;   // 动态描述/内容
  author?: string;        // 作者
  created_at?: string;    // 创建时间
  updated_at?: string;    // 更新时间
  type?: string;         // 动态类型
  status?: string;       // 状态
  // 添加其他可能的字段类型保护
  [key: string]: any;
}

// 分页相关状态
const currentPage = ref(0);
const initialPageSize = 5; // 初始显示5条
const pageSize = 3; // 每次懒加载3条
const loading = ref(false);

// 动态数据状态
const allTrendItems = ref<TrendItem[]>([]);
const totalCount = ref(0);

// 数据转换函数：将API数据转换为显示用的动态数据
const transformApiTrend = (apiTrend: ApiTrendItem): TrendItem => {
  // 格式化时间
  const formatTime = (dateString: string | undefined): string => {
    if (!dateString) return '未知时间';
    
    try {
      const date = new Date(dateString);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / (1000 * 60));
      const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
      
      if (diffMins < 1) return '刚刚';
      if (diffMins < 60) return `${diffMins}分钟前`;
      if (diffHours < 24) return `${diffHours}小时前`;
      if (diffDays < 7) return `${diffDays}天前`;
      if (diffDays < 30) return `${Math.floor(diffDays / 7)}周前`;
      
      return date.toLocaleDateString('zh-CN');
    } catch (error) {
      console.warn('时间格式化失败:', dateString, error);
      return '未知时间';
    }
  };

  // 清理和验证文本内容
  const cleanText = (text: string | undefined | null): string => {
    if (!text) return '';
    // 转为字符串并移除可能的危险字符
    return String(text)
      .replace(/[<>]/g, '') // 移除尖括号，防止HTML注入
      .trim();
  };

  const title = cleanText(apiTrend.name) || cleanText(apiTrend.author) || '未知用户';
  const content = cleanText(apiTrend.description) || '暂无内容';
  const date = formatTime(apiTrend.created_at);

  return {
    title,
    content,
    date
  };
};

// 获取动态数据
const fetchTrends = async (page: number = 0, size: number = initialPageSize) => {
  try {
    loading.value = true;
    const response = await requestClient.get('/market/trends?start_date=2025-01-01&end_date=2025-12-31', {
      params: {
        page: page,
        size: size
      }
    });
    
    console.log('Trends API 响应:', response);
    
    let apiTrends: ApiTrendItem[] = [];
    let hasMoreData = false;
    
    if (response && Array.isArray(response)) {
      // 直接处理数组响应
      apiTrends = response.filter((item: any) => 
        item && typeof item === 'object' && 
        (item.name || item.title) && 
        (item.description || item.content)
      );
      hasMoreData = apiTrends.length === size;
    } else if (response && response.success && response.data) {
      // 处理包装的响应格式
      const dataArray = response || [];
      if (Array.isArray(dataArray)) {
        apiTrends = dataArray.filter((item: any) => 
          item && typeof item === 'object' && 
          (item.name || item.title) && 
          (item.description || item.content)
        );
        hasMoreData = response.data.hasMore || (apiTrends.length === size);
        totalCount.value = response.data.total || apiTrends.length;
      }
    } else if (response && response.data && Array.isArray(response.data)) {
      // 处理另一种可能的格式
      apiTrends = response.data.filter((item: any) => 
        item && typeof item === 'object' && 
        (item.name || item.title) && 
        (item.description || item.content)
      );
      hasMoreData = apiTrends.length === size;
    }
    
    if (apiTrends.length > 0) {
      // 转换数据前先验证
      console.log('API 原始数据:', apiTrends);
      
      try {
        const transformedTrends = apiTrends.map((trend, index) => {
          try {
            return transformApiTrend(trend);
          } catch (error) {
            console.error(`转换第 ${index} 条数据失败:`, trend, error);
            // 返回一个安全的默认对象
            return {
              title: '数据异常',
              content: '该动态数据存在异常',
              date: '未知时间'
            };
          }
        });
        
        if (page === 0) {
          // 首次加载，替换所有数据
          allTrendItems.value = transformedTrends;
        } else {
          // 追加加载，添加到现有数据
          allTrendItems.value.push(...transformedTrends);
        }
        
        console.log(`成功获取 ${transformedTrends.length} 条动态数据`);
        
        return {
          trends: transformedTrends,
          hasMore: hasMoreData
        };
      } catch (error) {
        console.error('数据转换过程中出现错误:', error);
        throw error;
      }
    } else {
      console.warn('API 返回数据为空或格式不正确:', response);
      // 使用默认数据作为备用
      if (page === 0) {
        allTrendItems.value = getDefaultTrends();
        console.log('使用默认动态数据');
      }
      return {
        trends: [],
        hasMore: false
      };
    }
  } catch (error) {
    console.error('获取动态数据失败:', error);
    // 使用默认数据作为备用
    if (page === 0) {
      allTrendItems.value = getDefaultTrends();
      console.log('API 请求失败，使用默认动态数据');
    }
    return {
      trends: [],
      hasMore: false
    };
  } finally {
    loading.value = false;
  }
};

// 默认数据作为备用
const getDefaultTrends = (): TrendItem[] => [
  {
    title: 'KEG实验室',
    content: '发布了新模型 <strong>ChatGLM3-6B</strong> 到模型广场，支持多轮对话和代码生成',
    date: '刚刚',
  },
  {
    title: '通义千问团队',
    content: '更新了模型 <strong>Qwen2-7B-Instruct</strong> 的配置，提升了推理性能',
    date: '1小时前',
  },
  {
    title: '上海AI实验室',
    content: '分享了模型优化技巧 <strong>大模型推理加速与内存优化</strong>',
    date: '2小时前',
  },
  {
    title: '百川智能',
    content: '发布了技术文档 <strong>如何部署私有化MaaS服务</strong>',
    date: '4小时前',
  },
  {
    title: 'Hugging Face',
    content: '新增模型 <strong>Meta-Llama-3-8B-Instruct</strong> 支持中文对话',
    date: '6小时前',
  },
  {
    title: 'OpenAI',
    content: '发布了 <strong>GPT-4 Turbo</strong> 新版本，降低了API调用成本',
    date: '1天前',
  },
  {
    title: '智谱AI',
    content: '优化了 <strong>GLM-4</strong> 的函数调用能力，支持更复杂的工具使用',
    date: '1天前',
  },
  {
    title: '阿里云',
    content: '推出了 <strong>通义千问Plus</strong> 企业版，支持私有部署',
    date: '2天前',
  }
];

// 计算当前显示的动态列表
const displayItems = computed(() => {
  const totalDisplayed = currentPage.value === 0 
    ? initialPageSize 
    : initialPageSize + (currentPage.value * pageSize);
  return allTrendItems.value.slice(0, totalDisplayed);
});

// 计算是否还有更多数据
const hasMore = computed(() => {
  return displayItems.value.length < allTrendItems.value.length;
});

// 加载更多数据
const loadMore = async () => {
  if (loading.value || !hasMore.value) return;
  
  loading.value = true;
  
  try {
    // 计算下一页的页码
    const nextPage = Math.floor(allTrendItems.value.length / pageSize);
    const result = await fetchTrends(nextPage, pageSize);
    
    if (result.trends.length > 0) {
      currentPage.value++;
    }
  } catch (error) {
    console.error('加载更多动态失败:', error);
  } finally {
    loading.value = false;
  }
};

// 初始化数据
const initializeData = async () => {
  currentPage.value = 0;
  allTrendItems.value = [];
  await fetchTrends(0, initialPageSize);
};

// 组件挂载时获取数据
onMounted(() => {
  initializeData();
});

// 获取头像颜色
const getAvatarColor = (index: number) => {
  const colors = [
    'bg-gradient-to-br from-blue-500 to-indigo-600',
    'bg-gradient-to-br from-purple-500 to-pink-600',
    'bg-gradient-to-br from-green-500 to-emerald-600',
    'bg-gradient-to-br from-yellow-500 to-orange-600',
    'bg-gradient-to-br from-red-500 to-rose-600',
    'bg-gradient-to-br from-indigo-500 to-purple-600',
    'bg-gradient-to-br from-teal-500 to-cyan-600',
    'bg-gradient-to-br from-orange-500 to-red-600',
  ];
  return colors[index % colors.length];
};

// 获取头像文字
const getAvatarText = (title: string) => {
  // 优先匹配具体机构/团队
  if (title.includes('KEG') || title.includes('实验室')) return '实';
  if (title.includes('通义') || title.includes('阿里')) return '阿';
  if (title.includes('百川') || title.includes('智能')) return '智';
  if (title.includes('智谱')) return '智';
  if (title.includes('团队')) return '团';
  if (title.includes('上海') || title.includes('中心')) return '中';
  
  // 匹配国外知名AI公司
  if (title.includes('OpenAI')) return 'O';
  if (title.includes('Meta')) return 'M';
  if (title.includes('Anthropic')) return 'A';
  if (title.includes('Hugging') || title.includes('Face')) return 'H';
  if (title.includes('Google')) return 'G';
  if (title.includes('Microsoft')) return 'M';
  
  // 匹配国内公司
  if (title.includes('腾讯')) return '腾';
  if (title.includes('百度')) return '百';
  if (title.includes('字节') || title.includes('豆包')) return '字';
  if (title.includes('讯飞')) return '讯';
  if (title.includes('商汤')) return '商';
  if (title.includes('旷视')) return '旷';
  
  // 通用AI关键词
  if (title.includes('AI') || title.includes('ai')) return 'AI';
  
  // 默认取第一个字符
  return title.charAt(0);
};

// 格式化内容，安全地处理HTML标签
const formatContent = (content: string) => {
  if (!content || typeof content !== 'string') {
    return '';
  }
  
  try {
    // 只允许特定的安全标签，并转换为安全的样式
    return content
      .replace(/<strong>(.*?)<\/strong>/g, '<span class="text-purple-500 font-semibold">$1</span>')
      .replace(/<b>(.*?)<\/b>/g, '<span class="font-semibold">$1</span>')
      .replace(/<em>(.*?)<\/em>/g, '<span class="italic">$1</span>')
      .replace(/<i>(.*?)<\/i>/g, '<span class="italic">$1</span>')
      // 移除其他所有HTML标签
      .replace(/<[^>]*>/g, '');
  } catch (error) {
    console.warn('内容格式化失败:', content, error);
    return String(content).replace(/<[^>]*>/g, ''); // 移除所有HTML标签作为后备
  }
};

// 暴露方法给父组件
defineExpose({
  initializeData,
});
</script>

<style scoped>
.line-clamp-2 {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
</style>
