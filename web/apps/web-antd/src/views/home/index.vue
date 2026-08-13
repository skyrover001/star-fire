<script lang="ts" setup>
import type { SupportedLanguagesType } from '@vben/locales';

import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';

import { i18n, loadLocaleMessages } from '@vben/locales';
import { updatePreferences } from '@vben/preferences';
import { $t } from '#/locales';

import { requestClient } from '#/api/request';

interface ModelMarketStat {
  calls: number;
  client_count: number;
  model: string;
  total_tokens: number;
  user_count: number;
}

interface ModelRankEntry {
  calls: number;
  model: string;
  total_tokens: number;
}

interface ContributorRankEntry {
  display_name: string;
  income: number;
}

interface PublicDailyTrend {
  date: string;
  total_tokens: number;
  total_value: number;
}

interface HomepageStats {
  active_users: number;
  daily_trend: PublicDailyTrend[];
  model_stats: ModelMarketStat[];
  total_calls: number;
  total_tokens: number;
  total_value: number;
  model_rank: ModelRankEntry[];
  contributor_rank: ContributorRankEntry[];
}

interface MarketplaceModel {
  client_models?: unknown[];
  name: string;
  quantization?: string;
  size: string;
  type: string;
}

interface HomepageData {
  models: MarketplaceModel[] | null;
  stats: HomepageStats;
}

const router = useRouter();
const homepage = ref<HomepageData>();
const loading = ref(true);
const failed = ref(false);
const navOpen = ref(false);
const scrolled = ref(false);
const heroCanvas = ref<HTMLCanvasElement>();
const trendCanvas = ref<HTMLCanvasElement>();
let refreshTimer: ReturnType<typeof setInterval> | undefined;
let animationFrame = 0;

const currentLang = computed<SupportedLanguagesType>(
  () => i18n.global.locale.value as SupportedLanguagesType,
);

async function switchLang(lang: SupportedLanguagesType) {
  if (lang === currentLang.value) return;
  updatePreferences({ app: { locale: lang } });
  await loadLocaleMessages(lang);
  window.location.reload();
}

// ---- data helpers ----
const modelStats = computed(() =>
  [...(homepage.value?.stats.model_stats ?? [])]
    .sort((a, b) => b.calls - a.calls)
    .slice(0, 7),
);

const maxBarValue = computed(() =>
  Math.max(1, ...modelStats.value.map((m) => m.calls)),
);

// 模型调用次数与 tokens 排名（总的，来自后端聚合）
const modelRank = computed(() => homepage.value?.stats.model_rank ?? []);

// 贡献者收益排名（前10，单位 $）
const contributorRank = computed(
  () => homepage.value?.stats.contributor_rank ?? [],
);

const trendData = computed(() => homepage.value?.stats.daily_trend ?? []);

function getModelStat(name: string) {
  return homepage.value?.stats.model_stats.find((m) => m.model === name);
}

function formatNumber(n = 0) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`;
  return String(n);
}

function formatCurrency(n = 0) {
  if (n >= 1_000) return `¥${(n / 1_000).toFixed(1)}K`;
  return `¥${n.toFixed(0)}`;
}

function formatCompact(n = 0) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

// ---- data loading ----
async function loadHomepage() {
  try {
    const response = await requestClient.get<HomepageData>('/public/homepage');
    // Normalize null models to empty array to prevent downstream errors
    if (response && !response.models) {
      response.models = [];
    }
    homepage.value = response;
    failed.value = false;
    await nextTick();
    drawTrend();
  } catch {
    failed.value = !homepage.value;
  } finally {
    loading.value = false;
  }
}

// ---- toast ----
const toasts = ref<{ id: number; icon: string; msg: string }[]>([]);
let toastId = 0;

function showToast(msg: string, icon = '✨') {
  const id = ++toastId;
  toasts.value.push({ id, icon, msg });
  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  }, 2800);
}

// ---- navigation ----
function scrollTo(id: string) {
  navOpen.value = false;
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' });
}

function goLogin() {
  router.push('/auth/login');
}

function goRegister() {
  router.push('/auth/register');
}

// 客户端下载链接配置（与模型广场保持一致）
const clientDownloadUrls: Record<string, { url: string; filename: string }> = {
  windows: {
    url: '/download/windows/starfire.rar',
    filename: 'starfire.rar',
  },
  macos: {
    url: '/download/macos/starfire.zip',
    filename: 'starfire',
  },
  linux: {
    url: '/download/linux/starfire.tar.gz',
    filename: 'starfire.tar.gz',
  },
};

function downloadClient(os?: string) {
  // 归一化平台名（兼容 'Windows'/'macOS'/'Linux' 与 'windows'/'macos'/'linux'）
  const platform = (os || '').toLowerCase();
  const clientInfo = clientDownloadUrls[platform];

  if (!clientInfo) {
    const msg =
      currentLang.value === 'zh-CN'
        ? '⬇ 客户端下载中...'
        : '⬇ Downloading client...';
    showToast(msg, '⬇');
    return;
  }

  try {
    // 使用同源相对路径构造下载链接（安装包与前端部署在同一 nginx 上），
    // 并附带 filename 参数（匹配 nginx Content-Disposition 配置）。
    // 不依赖 serverHost，确保 dev 与生产环境行为一致。
    const downloadUrl = `${window.location.origin}${clientInfo.url}?filename=${encodeURIComponent(clientInfo.filename)}`;

    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = clientInfo.filename;
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    const msg =
      currentLang.value === 'zh-CN'
        ? `⬇ 正在下载 ${os} 客户端`
        : `⬇ Downloading ${os} client`;
    showToast(msg, '⬇');
  } catch (error) {
    console.error('下载客户端失败:', error);
    const msg =
      currentLang.value === 'zh-CN'
        ? '⬇ 下载失败，请稍后重试'
        : '⬇ Download failed, please try again';
    showToast(msg, '⬇');
  }
}

// ---- hero canvas animation (faithful port from HTML) ----
function initHeroCanvas() {
  const canvas = heroCanvas.value;
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  let W = 0;
  let H = 0;
  let dpr = 1;

  function resize() {
    if (!canvas || !ctx) return;
    const rect = canvas.parentElement!.getBoundingClientRect();
    dpr = window.devicePixelRatio || 1;
    W = canvas.width = rect.width * dpr;
    H = canvas.height = rect.height * dpr;
    canvas.style.width = `${rect.width}px`;
    canvas.style.height = `${rect.height}px`;
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.scale(dpr, dpr);
  }

  const bannerModels = [
    { id: 'DeepSeek-V4', calls: 284, color: '#8b5cf6', logo: '/model-logos/deepseek.png' },
    { id: 'GLM-5.2', calls: 212, color: '#3b82f6', logo: '/model-logos/glm.png' },
    { id: 'Kimi-K3', calls: 187, color: '#22d3ee', logo: '/model-logos/kimi.png' },
    { id: 'Qwen-3.6', calls: 156, color: '#ec4899', logo: '/model-logos/qwen.png' },
    { id: 'MiniMax-2.7', calls: 123, color: '#34d399', logo: '/model-logos/minimax.png' },
    { id: 'Claude Opus 5', calls: 98, color: '#f59e0b', logo: '/model-logos/claude.png' },
    { id: 'GPT-5.6', calls: 72, color: '#f472b6', logo: '/model-logos/gpt.svg' },
  ];
  const maxCalls = Math.max(...bannerModels.map((d) => d.calls));

  // Preload model logo images
  const modelLogoImages: Record<string, HTMLImageElement> = {};
  for (const m of bannerModels) {
    const img = new Image();
    img.src = m.logo;
    modelLogoImages[m.id] = img;
  }

  const centerX = 0.7;
  const centerY = 0.5;
  const spreadX = 0.24;
  const spreadY = 0.22;
  let modelPositions: { x: number; y: number; idx: number }[] = [];

  function initModelPositions() {
    const count = bannerModels.length;
    const positions: { x: number; y: number; idx: number }[] = [];
    const baseSize = 0.028;
    for (let i = 0; i < count; i++) {
      let attempts = 0;
      let placed = false;
      let pos = { x: 0, y: 0 };
      while (!placed && attempts < 500) {
        const angle = Math.random() * Math.PI * 2;
        const r = Math.random() * 0.85;
        const x = centerX + Math.cos(angle) * spreadX * r;
        const y = centerY + Math.sin(angle) * spreadY * r;
        pos = { x, y };
        let overlap = false;
        for (let j = 0; j < positions.length; j++) {
          const dx = pos.x - positions[j]!.x;
          const dy = pos.y - positions[j]!.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          const sizeRatioI = bannerModels[i]!.calls / maxCalls;
          const sizeRatioJ = bannerModels[positions[j]!.idx]!.calls / maxCalls;
          const fontSizeI = baseSize * (0.45 + 0.75 * sizeRatioI);
          const fontSizeJ = baseSize * (0.45 + 0.75 * sizeRatioJ);
          // Account for logo diameter + text height below logo
          const threshold = (fontSizeI + fontSizeJ) * 1.0;
          if (dist < threshold * 1.15) {
            overlap = true;
            break;
          }
        }
        if (!overlap) placed = true;
        attempts++;
      }
      if (placed) {
        positions.push({ x: pos.x, y: pos.y, idx: i });
      } else {
        const angle = Math.random() * Math.PI * 2;
        const r = Math.random() * 0.8;
        positions.push({
          x: centerX + Math.cos(angle) * spreadX * r,
          y: centerY + Math.sin(angle) * spreadY * r,
          idx: i,
        });
      }
    }
    modelPositions = positions;
  }

  const contributorCount = 5;
  const contributors: {
    x: number; y: number; vx: number; vy: number; phase: number;
  }[] = [];
  for (let i = 0; i < contributorCount; i++) {
    contributors.push({
      x: 0.88 + Math.random() * 0.08,
      y: 0.12 + Math.random() * 0.76,
      vx: (Math.random() - 0.5) * 0.0005,
      vy: (Math.random() - 0.5) * 0.0005,
      phase: Math.random() * 100,
    });
  }

  const consumerCount = 5;
  const consumers: {
    x: number; y: number; vx: number; vy: number; phase: number;
  }[] = [];
  for (let i = 0; i < consumerCount; i++) {
    consumers.push({
      x: 0.04 + Math.random() * 0.08,
      y: 0.12 + Math.random() * 0.76,
      vx: (Math.random() - 0.5) * 0.0005,
      vy: (Math.random() - 0.5) * 0.0005,
      phase: Math.random() * 100,
    });
  }

  interface Particle {
    path: { from: string; fromIdx: number; to: string; toIdx: number };
    progress: number;
    speed: number;
    size: number;
    color: string;
    phase: number;
  }

  let particles: Particle[] = [];
  const paths: Particle['path'][] = [];
  for (let ci = 0; ci < contributorCount; ci++) {
    paths.push({ from: 'contributor', fromIdx: ci, to: 'center', toIdx: 0 });
  }
  for (let ci = 0; ci < consumerCount; ci++) {
    paths.push({ from: 'center', fromIdx: 0, to: 'consumer', toIdx: ci });
  }

  function getNode(type: string, idx: number) {
    if (type === 'contributor')
      return contributors[idx % contributors.length] ?? contributors[0]!;
    if (type === 'consumer')
      return consumers[idx % consumers.length] ?? consumers[0]!;
    return { x: centerX, y: centerY };
  }

  function initParticles() {
    particles = [];
    for (const p of paths) {
      const count = p.from === 'contributor' ? 16 : 14;
      for (let i = 0; i < count; i++) {
        const t = i / count;
        const speed = 0.001 + Math.random() * 0.0015;
        const color = p.from === 'contributor' ? '#8b5cf6' : '#f59e0b';
        particles.push({
          path: p,
          progress: t,
          speed,
          size: 2 + Math.random() * 3,
          color,
          phase: Math.random() * 100,
        });
      }
    }
    particles.sort(() => Math.random() - 0.5);
  }

  function getPointOnPath(path: Particle['path'], t: number) {
    const from = getNode(path.from, path.fromIdx);
    const to = getNode(path.to, path.toIdx);
    const mx = (from.x + to.x) / 2;
    const my = (from.y + to.y) / 2 - 0.03;
    const t1 = 1 - t;
    return {
      x: t1 * t1 * from.x + 2 * t1 * t * mx + t * t * to.x,
      y: t1 * t1 * from.y + 2 * t1 * t * my + t * t * to.y,
    };
  }

  function draw() {
    if (!ctx) return;
    const w = W / dpr;
    const h = H / dpr;
    ctx.clearRect(0, 0, w, h);

    const grad = ctx.createRadialGradient(
      w * 0.5, h * 0.5, 0, w * 0.5, h * 0.5, w * 0.7,
    );
    grad.addColorStop(0, '#12121e');
    grad.addColorStop(1, '#07070d');
    ctx.fillStyle = grad;
    ctx.fillRect(0, 0, w, h);

    ctx.globalAlpha = 0.04;
    ctx.lineWidth = 0.5;
    for (const c of contributors) {
      ctx.beginPath();
      ctx.moveTo(c.x * w, c.y * h);
      ctx.lineTo(centerX * w, centerY * h);
      ctx.strokeStyle = '#8b5cf6';
      ctx.stroke();
    }
    for (const c of consumers) {
      ctx.beginPath();
      ctx.moveTo(c.x * w, c.y * h);
      ctx.lineTo(centerX * w, centerY * h);
      ctx.strokeStyle = '#f59e0b';
      ctx.stroke();
    }
    ctx.globalAlpha = 1;

    const glow = ctx.createRadialGradient(
      centerX * w, centerY * h, 0,
      centerX * w, centerY * h, Math.min(w, h) * 0.22,
    );
    glow.addColorStop(0, 'rgba(139,92,246,0.06)');
    glow.addColorStop(1, 'rgba(0,0,0,0)');
    ctx.fillStyle = glow;
    ctx.beginPath();
    ctx.arc(centerX * w, centerY * h, Math.min(w, h) * 0.22, 0, Math.PI * 2);
    ctx.fill();

    const baseSize = Math.min(w, h) * 0.024;
    for (let i = 0; i < modelPositions.length; i++) {
      const mp = modelPositions[i]!;
      const data = bannerModels[i]!;
      const sizeRatio = data.calls / maxCalls;
      const fontSize = baseSize * (0.45 + 0.75 * sizeRatio);
      const px = mp.x * w;
      const py = mp.y * h;
      const r = fontSize * 0.7;

      // Glow halo behind logo
      ctx.shadowColor = data.color;
      ctx.shadowBlur = 18;
      const haloGrad = ctx.createRadialGradient(
        px, py, 0, px, py, r * 1.5,
      );
      haloGrad.addColorStop(0, `${data.color}33`);
      haloGrad.addColorStop(1, `${data.color}00`);
      ctx.fillStyle = haloGrad;
      ctx.beginPath();
      ctx.arc(px, py, r * 1.5, 0, Math.PI * 2);
      ctx.fill();
      ctx.shadowBlur = 0;

      // Draw logo image clipped to circle, or fallback gradient
      const logoImg = modelLogoImages[data.id];
      if (logoImg && logoImg.complete && logoImg.naturalWidth > 0) {
        ctx.save();
        ctx.beginPath();
        ctx.arc(px, py, r, 0, Math.PI * 2);
        ctx.closePath();
        ctx.clip();
        ctx.drawImage(logoImg, px - r, py - r, r * 2, r * 2);
        ctx.restore();
      } else {
        const dotGrad = ctx.createRadialGradient(
          px - r * 0.2, py - r * 0.2, 0, px, py, r,
        );
        dotGrad.addColorStop(0, `${data.color}55`);
        dotGrad.addColorStop(1, `${data.color}08`);
        ctx.beginPath();
        ctx.arc(px, py, r, 0, Math.PI * 2);
        ctx.fillStyle = dotGrad;
        ctx.fill();
      }

      // Colored ring around logo
      ctx.beginPath();
      ctx.arc(px, py, r, 0, Math.PI * 2);
      ctx.strokeStyle = `${data.color}aa`;
      ctx.lineWidth = 1.5;
      ctx.stroke();

      // Model text below the logo
      ctx.fillStyle = '#fff';
      ctx.font = `600 ${fontSize}px Inter, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'top';
      ctx.shadowColor = 'rgba(0,0,0,0.6)';
      ctx.shadowBlur = 4;
      ctx.fillText(data.id, px, py + r + 4);
      ctx.shadowBlur = 0;
    }

    ctx.fillStyle = 'rgba(255,255,255,0.06)';
    ctx.font = `500 ${Math.min(w, h) * 0.011}px Inter, sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'bottom';
    ctx.fillText(`✦ ${$t('page.home.brand')} ${$t('page.home.brandSuffix')}`, centerX * w, (centerY - 0.22) * h);

    for (const c of contributors) {
      const r = Math.min(w, h) * 0.016;
      const g = ctx.createRadialGradient(
        c.x * w - r * 0.3, c.y * h - r * 0.3, 0, c.x * w, c.y * h, r,
      );
      g.addColorStop(0, '#a78bfa');
      g.addColorStop(1, '#6d28d9');
      ctx.shadowColor = '#8b5cf6';
      ctx.shadowBlur = 25;
      ctx.beginPath();
      ctx.arc(c.x * w, c.y * h, r, 0, Math.PI * 2);
      ctx.fillStyle = g;
      ctx.fill();
      ctx.shadowBlur = 0;
      ctx.fillStyle = 'rgba(255,255,255,0.12)';
      ctx.font = `${Math.min(w, h) * 0.012}px Inter, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'bottom';
      ctx.fillText('⚡ Contributors', c.x * w, c.y * h - r - 6);
    }

    for (const c of consumers) {
      const r = Math.min(w, h) * 0.016;
      const g = ctx.createRadialGradient(
        c.x * w - r * 0.3, c.y * h - r * 0.3, 0, c.x * w, c.y * h, r,
      );
      g.addColorStop(0, '#fcd34d');
      g.addColorStop(1, '#b45309');
      ctx.shadowColor = '#f59e0b';
      ctx.shadowBlur = 25;
      ctx.beginPath();
      ctx.arc(c.x * w, c.y * h, r, 0, Math.PI * 2);
      ctx.fillStyle = g;
      ctx.fill();
      ctx.shadowBlur = 0;
      ctx.fillStyle = 'rgba(255,255,255,0.12)';
      ctx.font = `${Math.min(w, h) * 0.012}px Inter, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'bottom';
      ctx.fillText('👤 Consumers', c.x * w, c.y * h - r - 6);
    }

    for (const p of particles) {
      const pos = getPointOnPath(p.path, p.progress);
      const px = pos.x * w;
      const py = pos.y * h;
      const alpha = 0.5 + 0.5 * (1 - Math.abs(p.progress - 0.5) * 2);
      const size = p.size * (0.7 + 0.3 * (1 - Math.abs(p.progress - 0.5) * 2));
      ctx.globalAlpha = alpha * 0.85;
      ctx.shadowColor = p.color;
      ctx.shadowBlur = 16;
      ctx.beginPath();
      ctx.arc(px, py, size * 0.5, 0, Math.PI * 2);
      ctx.fillStyle = p.color;
      ctx.fill();
      ctx.shadowBlur = 0;
      ctx.globalAlpha = alpha * 0.15;
      ctx.beginPath();
      ctx.arc(px - size * 0.15, py - size * 0.15, size * 0.15, 0, Math.PI * 2);
      ctx.fillStyle = '#fff';
      ctx.fill();
    }
    ctx.globalAlpha = 1;
    ctx.shadowBlur = 0;

    ctx.fillStyle = 'rgba(255,255,255,0.03)';
    ctx.fillRect(0, h - 28, w, 28);
    ctx.fillStyle = 'rgba(255,255,255,0.06)';
    ctx.font = '10px Inter, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    const label =
      currentLang.value === 'zh-CN'
        ? '← Token 流入 · 价值流出 →  |  贡献者 → 平台 → 消费者'
        : '← Tokens in · Value out →  |  Contributors → Platform → Consumers';
    ctx.fillText(label, w / 2, h - 14);
  }

  function updateNodes() {
    for (const c of contributors) {
      c.x += c.vx * 0.4;
      c.y += c.vy * 0.4;
      if (c.x < 0.82 || c.x > 0.96) c.vx *= -1;
      if (c.y < 0.1 || c.y > 0.9) c.vy *= -1;
      c.vx += (Math.random() - 0.5) * 0.00015;
      c.vy += (Math.random() - 0.5) * 0.00015;
      c.vx = Math.max(-0.0008, Math.min(0.0008, c.vx));
      c.vy = Math.max(-0.0008, Math.min(0.0008, c.vy));
    }
    for (const c of consumers) {
      c.x += c.vx * 0.4;
      c.y += c.vy * 0.4;
      if (c.x < 0.04 || c.x > 0.18) c.vx *= -1;
      if (c.y < 0.1 || c.y > 0.9) c.vy *= -1;
      c.vx += (Math.random() - 0.5) * 0.00015;
      c.vy += (Math.random() - 0.5) * 0.00015;
      c.vx = Math.max(-0.0008, Math.min(0.0008, c.vx));
      c.vy = Math.max(-0.0008, Math.min(0.0008, c.vy));
    }
  }

  function updateParticles() {
    for (const p of particles) {
      p.progress += p.speed * (0.7 + 0.3 * Math.sin(p.phase + Date.now() * 0.0003));
      if (p.progress > 1) {
        p.progress = 0;
        const pathIdx = Math.floor(Math.random() * paths.length);
        p.path = paths[pathIdx]!;
        p.speed = 0.001 + Math.random() * 0.0015;
        p.color = p.path.from === 'contributor' ? '#8b5cf6' : '#f59e0b';
      }
    }
  }

  function animate() {
    updateNodes();
    updateParticles();
    draw();
    animationFrame = requestAnimationFrame(animate);
  }

  const resizeHandler = () => resize();
  window.addEventListener('resize', resizeHandler);

  initModelPositions();
  initParticles();
  resize();
  animate();

  onBeforeUnmount(() => {
    window.removeEventListener('resize', resizeHandler);
    cancelAnimationFrame(animationFrame);
  });
}

// ---- trend canvas (faithful port) ----
function drawTrend() {
  const canvas = trendCanvas.value;
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const data =
    trendData.value.length > 0
      ? trendData.value.map((d) => d.total_tokens)
      : [0, 0, 0, 0, 0, 0, 0];

  const width = canvas.parentElement!.clientWidth - 32;
  const height = 100;
  canvas.width = width * (window.devicePixelRatio || 1);
  canvas.height = height * (window.devicePixelRatio || 1);
  canvas.style.width = `${width}px`;
  canvas.style.height = `${height}px`;
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.scale(window.devicePixelRatio || 1, window.devicePixelRatio || 1);

  const max = Math.max(...data, 1) * 1.1;
  const min = Math.min(...data) * 0.9;
  const padding = { top: 10, bottom: 10, left: 10, right: 10 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  ctx.clearRect(0, 0, width, height);
  ctx.beginPath();
  ctx.moveTo(
    padding.left,
    padding.top + chartHeight - ((data[0]! - min) / (max - min || 1)) * chartHeight,
  );
  for (let i = 1; i < data.length; i++) {
    const x = padding.left + (i / (data.length - 1)) * chartWidth;
    const y =
      padding.top + chartHeight - ((data[i]! - min) / (max - min || 1)) * chartHeight;
    ctx.lineTo(x, y);
  }
  ctx.strokeStyle = '#8b5cf6';
  ctx.lineWidth = 2;
  ctx.stroke();

  ctx.lineTo(padding.left + chartWidth, padding.top + chartHeight);
  ctx.lineTo(padding.left, padding.top + chartHeight);
  ctx.closePath();
  const grad = ctx.createLinearGradient(0, padding.top, 0, padding.top + chartHeight);
  grad.addColorStop(0, 'rgba(139,92,246,0.2)');
  grad.addColorStop(1, 'rgba(139,92,246,0)');
  ctx.fillStyle = grad;
  ctx.fill();

  for (let i = 0; i < data.length; i++) {
    const x = padding.left + (i / (data.length - 1)) * chartWidth;
    const y =
      padding.top + chartHeight - ((data[i]! - min) / (max - min || 1)) * chartHeight;
    ctx.beginPath();
    ctx.arc(x, y, 3, 0, Math.PI * 2);
    ctx.fillStyle = '#8b5cf6';
    ctx.fill();
  }
}

function onScroll() {
  scrolled.value = window.scrollY > 20;
}

onMounted(() => {
  loadHomepage();
  refreshTimer = setInterval(loadHomepage, 60_000);
  window.addEventListener('scroll', onScroll);
  nextTick(() => initHeroCanvas());
});

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer);
  window.removeEventListener('scroll', onScroll);
  cancelAnimationFrame(animationFrame);
});

watch(currentLang, () => {
  nextTick(() => drawTrend());
});
</script>

<template>
  <div class="sf-home">
    <!-- ===== Navigation ===== -->
    <nav class="navbar" :class="{ scrolled }">
      <div class="container nav-inner">
        <a class="nav-brand" href="#" @click.prevent>
          <span class="logo">✦</span>
          <span class="brand-text">
            <span class="brand-name">{{ $t('page.home.brand') }}</span>
            <span class="brand-suffix">{{ $t('page.home.brandSuffix') }}</span>
          </span>
        </a>
        <ul class="nav-links" :class="{ open: navOpen }">
          <li><a href="#models" @click.prevent="scrollTo('models')">{{ $t('page.home.navModels') }}</a></li>
          <li><a href="#client" @click.prevent="scrollTo('client')">{{ $t('page.home.navClient') }}</a></li>
          <li><a href="#tokens" @click.prevent="scrollTo('tokens')">{{ $t('page.home.navTokens') }}</a></li>
        </ul>
        <div class="nav-actions">
          <div class="lang-toggle">
            <button :class="{ active: currentLang === 'en-US' }" @click="switchLang('en-US')">EN</button>
            <button :class="{ active: currentLang === 'zh-CN' }" @click="switchLang('zh-CN')">中</button>
          </div>
          <button class="btn btn-outline btn-sm" @click="goLogin">{{ $t('page.home.login') }}</button>
          <button class="btn btn-primary btn-sm" @click="goRegister">{{ $t('page.home.signUp') }}</button>
          <button class="nav-toggle" @click="navOpen = !navOpen">
            <span></span><span></span><span></span>
          </button>
        </div>
      </div>
    </nav>

    <!-- ===== Hero ===== -->
    <section id="hero" class="hero">
      <canvas ref="heroCanvas" class="hero-canvas"></canvas>
      <div class="container hero-inner">
        <div class="hero-content">
          <div class="hero-badge">
            <span class="dot"></span>
            <span>{{ $t('page.home.liveNetwork') }}</span>
          </div>
          <h1>
            <span>{{ $t('page.home.headline1') }}</span><br />
            <span class="highlight">{{ $t('page.home.headline2') }}</span>
          </h1>
          <p>{{ $t('page.home.description') }}</p>
          <div class="hero-actions">
            <button class="btn btn-primary" @click="downloadClient()">{{ $t('page.home.downloadClient') }}</button>
            <button class="btn btn-outline" @click="scrollTo('tokens')">{{ $t('page.home.learnMore') }}</button>
          </div>
          <div class="hero-stats">
            <div class="stat">
              <span class="num">{{ formatCompact(homepage?.stats.total_tokens) }}</span>
              <span class="label">{{ $t('page.home.totalTokens') }}</span>
            </div>
            <div class="stat">
              <span class="num">{{ formatCurrency(homepage?.stats.total_value) }}</span>
              <span class="label">{{ $t('page.home.totalValue') }}</span>
            </div>
            <div class="stat">
              <span class="num">{{ formatCompact(homepage?.stats.active_users) }}</span>
              <span class="label">{{ $t('page.home.activeUsers') }}</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== Models ===== -->
    <section id="models" class="section-padding">
      <div class="container">
        <span class="section-label">{{ $t('page.home.modelsLabel') }}</span>
        <h2 class="section-title" v-html="$t('page.home.modelsTitle')"></h2>
        <p class="section-desc">{{ $t('page.home.modelsDesc') }}</p>
        <div v-if="loading" class="loading-msg">{{ $t('page.home.loading') }}</div>
        <div v-else-if="failed" class="loading-msg">{{ $t('page.home.unavailable') }}</div>
        <div v-else-if="homepage?.models?.length" class="models-grid">
          <div
            v-for="model in homepage.models.slice(0, 7)"
            :key="model.name"
            class="model-card"
          >
            <div class="model-header">
              <span class="name">{{ model.name }}</span>
              <span class="type-badge">{{ model.type || 'model' }}</span>
            </div>
            <div class="provider">{{ model.size || 'unknown' }}</div>
            <div class="metrics">
              <div class="metric">
                <span class="icon">🔄</span>
                <span class="value">{{ formatNumber(getModelStat(model.name)?.total_tokens) }}</span>
                <span class="label">{{ $t('page.home.metricTokens') }}</span>
              </div>
              <div class="metric">
                <span class="icon">📞</span>
                <span class="value">{{ formatNumber(getModelStat(model.name)?.calls) }}</span>
                <span class="label">{{ $t('page.home.metricCalls') }}</span>
              </div>
              <div class="metric">
                <span class="icon">💻</span>
                <span class="value">{{ model.client_models?.length ?? 0 }}</span>
                <span class="label">{{ $t('page.home.metricClients') }}</span>
              </div>
            </div>
            <div class="status"><span class="dot"></span> {{ $t('page.home.online') }}</div>
          </div>
        </div>
        <div v-else class="loading-msg">{{ $t('page.home.noModels') }}</div>
      </div>
    </section>

    <!-- ===== Client ===== -->
    <section id="client" class="section-padding client-section">
      <div class="container client-inner">
        <span class="section-label">{{ $t('page.home.downloadLabel') }}</span>
        <h2 class="section-title" v-html="$t('page.home.downloadTitle')"></h2>
        <p class="section-desc">{{ $t('page.home.downloadDesc') }}</p>
        <div class="client-features">
          <span class="feature-tag">{{ $t('page.home.featureZeroGpu') }}</span>
          <span class="feature-tag">{{ $t('page.home.featureStorage') }}</span>
          <span class="feature-tag">{{ $t('page.home.featureShare') }}</span>
          <span class="feature-tag">{{ $t('page.home.featureRewards') }}</span>
        </div>
        <div class="client-grid">
          <div class="client-card">
            <div class="icon">🪟</div>
            <div class="os">{{ $t('page.home.windows') }}</div>
            <div class="ver">v2.1.0 · x64</div>
            <button class="btn btn-primary btn-sm" @click="downloadClient('Windows')">{{ $t('page.home.download') }}</button>
          </div>
          <div class="client-card">
            <div class="icon">🍏</div>
            <div class="os">{{ $t('page.home.macos') }}</div>
            <div class="ver">v2.1.0 · Apple Silicon</div>
            <button class="btn btn-primary btn-sm" @click="downloadClient('macOS')">{{ $t('page.home.download') }}</button>
          </div>
          <div class="client-card">
            <div class="icon">🐧</div>
            <div class="os">{{ $t('page.home.linux') }}</div>
            <div class="ver">v2.1.0 · x64</div>
            <button class="btn btn-primary btn-sm" @click="downloadClient('Linux')">{{ $t('page.home.download') }}</button>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== Tokens Dashboard ===== -->
    <section id="tokens" class="section-padding">
      <div class="container">
        <span class="section-label">{{ $t('page.home.analyticsLabel') }}</span>
        <h2 class="section-title" v-html="$t('page.home.analyticsTitle')"></h2>
        <div class="tokens-grid">
          <div class="dash-card">
            <div class="head">
              <span>{{ $t('page.home.trendHead') }}</span>
              <span class="badge">{{ $t('page.home.last7d') }}</span>
            </div>
            <div class="stat-row">
              <div class="stat-box">
                <div class="num">{{ formatNumber(homepage?.stats.total_tokens) }}</div>
                <div class="lbl">{{ $t('page.home.tokensTraded') }}</div>
              </div>
              <div class="stat-box">
                <div class="num">{{ formatCurrency(homepage?.stats.total_value) }}</div>
                <div class="lbl">{{ $t('page.home.valueCny') }}</div>
              </div>
            </div>
            <canvas ref="trendCanvas" class="trend-canvas"></canvas>
            <div class="trend-labels">
              <span v-for="d in trendData" :key="d.date">{{ d.date.slice(5) }}</span>
            </div>
          </div>
          <div class="dash-card">
            <div class="head">
              <span>{{ $t('page.home.distributionHead') }}</span>
            </div>
            <div class="bar-chart">
              <div v-for="m in modelStats" :key="m.model" class="bar-item">
                <div
                  class="bar"
                  :class="{ highlight: m.calls >= maxBarValue * 0.7 }"
                  :style="{ height: `${(m.calls / maxBarValue) * 100}%` }"
                ></div>
                <div class="label">{{ m.model.length > 7 ? m.model.slice(0, 6) + '…' : m.model }}</div>
              </div>
            </div>
          </div>
        </div>
        <div class="rank-grid">
          <div class="dash-card">
            <div class="head"><span>{{ $t('page.home.modelRankHead') }}</span></div>
            <div class="rank-list">
              <div v-for="(m, i) in modelRank.slice(0, 10)" :key="m.model" class="rank-item rank-item-model">
                <div class="left"><span class="idx">#{{ i + 1 }}</span> {{ m.model }}</div>
                <div class="rank-metrics">
                  <div class="rank-metric">
                    <span class="rank-metric-num">{{ formatNumber(m.calls) }}</span>
                    <span class="rank-metric-label">{{ $t('page.home.metricCalls') }}</span>
                  </div>
                  <div class="rank-metric">
                    <span class="rank-metric-num">{{ formatNumber(m.total_tokens) }}</span>
                    <span class="rank-metric-label">{{ $t('page.home.metricTokens') }}</span>
                  </div>
                </div>
              </div>
              <div v-if="!modelRank.length" class="rank-empty">{{ $t('page.home.noData') }}</div>
            </div>
          </div>
          <div class="dash-card">
            <div class="head"><span>{{ $t('page.home.contributorRankHead') }}</span></div>
            <div class="rank-list">
              <div v-for="(c, i) in contributorRank" :key="`c-${c.display_name}-${i}`" class="rank-item">
                <div class="left"><span class="idx">#{{ i + 1 }}</span> {{ c.display_name }}</div>
                <span class="value">{{ formatCurrency(c.income) }}</span>
              </div>
              <div v-if="!contributorRank.length" class="rank-empty">{{ $t('page.home.noData') }}</div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== Footer ===== -->
    <footer class="footer">
      <div class="container footer-inner">
        <div class="col">
          <div class="brand">
            <span class="logo">✦</span>
            <span class="brand-text">
              <span class="brand-name">{{ $t('page.home.brand') }}</span>
              <span class="brand-suffix">{{ $t('page.home.brandSuffix') }}</span>
            </span>
          </div>
          <p class="footer-tagline">{{ $t('page.home.footerTagline') }}</p>
        </div>
        <div class="col">
          <h4>{{ $t('page.home.contact') }}</h4>
          <a href="mailto:hello@starfire.network" class="email">hello@starfire.network</a>
          <p class="footer-sub">{{ $t('page.home.responseTime') }}</p>
        </div>
        <div class="col">
          <h4>{{ $t('page.home.legal') }}</h4>
          <a href="#" @click.prevent>{{ $t('page.home.privacy') }}</a>
          <a href="#" @click.prevent>{{ $t('page.home.terms') }}</a>
        </div>
      </div>
      <div class="container footer-bottom">
        <span class="copyright">{{ $t('page.home.copyright') }}</span>
        <span>{{ $t('page.home.builtWith') }}</span>
      </div>
    </footer>

    <!-- ===== Toasts ===== -->
    <div class="toast-container">
      <div v-for="t in toasts" :key="t.id" class="toast">
        <span>{{ t.icon }}</span>
        <span class="msg">{{ t.msg }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sf-home {
  --bg: #07070d;
  --card: rgba(255, 255, 255, 0.04);
  --border: rgba(255, 255, 255, 0.06);
  --text: #f0f0f6;
  --text-sec: #8989a0;
  --text-muted: #4a4a5e;
  --purple: #8b5cf6;
  --cyan: #22d3ee;
  --gold: #f59e0b;
  --green: #34d399;
  --grad: linear-gradient(135deg, #8b5cf6, #3b82f6);
  --radius: 16px;
  --radius-sm: 10px;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.6;
  overflow-x: hidden;
  -webkit-font-smoothing: antialiased;
  min-height: 100vh;
}
.sf-home ::-webkit-scrollbar { width: 4px; }
.sf-home ::-webkit-scrollbar-track { background: var(--bg); }
.sf-home ::-webkit-scrollbar-thumb { background: var(--purple); border-radius: 2px; }
.sf-home a { color: inherit; text-decoration: none; }
.container { max-width: 1120px; margin: 0 auto; padding: 0 24px; }
.section-padding { padding: 60px 0; }

/* Buttons */
.btn {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 10px 28px; border-radius: 100px;
  font-weight: 600; font-size: 0.9rem; border: none;
  cursor: pointer; transition: 0.25s; font-family: inherit;
  background: var(--card); color: var(--text); border: 1px solid var(--border);
}
.btn-primary {
  background: var(--grad); color: #fff; border: none;
  box-shadow: 0 4px 20px rgba(139, 92, 246, 0.25);
}
.btn-primary:hover { transform: translateY(-2px); box-shadow: 0 8px 30px rgba(139, 92, 246, 0.35); }
.btn-outline { background: transparent; border: 1px solid rgba(255, 255, 255, 0.1); color: var(--text-sec); }
.btn-outline:hover { background: rgba(255, 255, 255, 0.04); border-color: rgba(255, 255, 255, 0.2); }
.btn-sm { padding: 6px 18px; font-size: 0.8rem; }

/* Navigation */
.navbar {
  position: fixed; top: 0; left: 0; right: 0; z-index: 100;
  padding: 12px 0; background: rgba(7, 7, 13, 0.8);
  backdrop-filter: blur(20px); border-bottom: 1px solid var(--border);
  transition: 0.3s;
}
.navbar.scrolled { padding: 8px 0; background: rgba(7, 7, 13, 0.95); }
.nav-inner { display: flex; align-items: center; justify-content: space-between; }
.nav-brand { display: flex; align-items: center; gap: 10px; font-size: 1.2rem; font-weight: 700; letter-spacing: -0.02em; }
.nav-brand .brand-text { display: flex; flex-direction: column; line-height: 1.15; }
.nav-brand .brand-name { font-size: 1.2rem; font-weight: 700; }
.nav-brand .brand-suffix { font-size: 0.62rem; font-weight: 400; color: var(--text-sec); letter-spacing: 0; }
.nav-brand .logo {
  width: 30px; height: 30px; background: var(--grad); border-radius: 8px;
  display: flex; align-items: center; justify-content: center; font-size: 0.9rem; color: #fff;
}
.nav-links { display: flex; align-items: center; gap: 24px; list-style: none; font-size: 0.85rem; font-weight: 500; }
.nav-links a { color: var(--text-sec); transition: 0.25s; }
.nav-links a:hover { color: var(--text); }
.nav-actions { display: flex; align-items: center; gap: 10px; }
.nav-actions .btn { padding: 6px 18px; font-size: 0.8rem; }
.lang-toggle {
  background: var(--card); border: 1px solid var(--border); border-radius: 100px;
  padding: 4px 8px; display: flex; gap: 4px; font-size: 0.7rem; font-weight: 600;
}
.lang-toggle button {
  background: none; border: none; color: var(--text-muted); cursor: pointer;
  padding: 2px 10px; border-radius: 100px; transition: 0.25s; font-weight: 600; font-family: inherit;
}
.lang-toggle button.active { background: var(--grad); color: #fff; }
.nav-toggle { display: none; flex-direction: column; gap: 4px; background: none; border: none; cursor: pointer; padding: 4px; }
.nav-toggle span { width: 22px; height: 2px; background: var(--text); border-radius: 2px; transition: 0.3s; }

/* Hero */
.hero { position: relative; min-height: 100vh; display: flex; align-items: center; padding-top: 72px; overflow: hidden; }
.hero-canvas { position: absolute; top: 0; left: 0; width: 100%; height: 100%; z-index: 0; pointer-events: none; display: block; }
.hero-inner { position: relative; z-index: 1; display: grid; grid-template-columns: 1fr 1fr; gap: 50px; align-items: center; }
.hero-content { display: flex; flex-direction: column; gap: 16px; }
.hero-badge {
  display: inline-flex; align-items: center; gap: 8px;
  background: rgba(139, 92, 246, 0.1); border: 1px solid rgba(139, 92, 246, 0.12);
  border-radius: 100px; padding: 4px 14px 4px 6px; font-size: 0.75rem; font-weight: 500;
  color: var(--purple); width: fit-content;
}
.hero-badge .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); animation: pulse-dot 2s infinite; }
@keyframes pulse-dot { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }
.hero h1 { font-size: clamp(2.4rem, 5.5vw, 4.2rem); font-weight: 800; letter-spacing: -0.03em; line-height: 1.08; }
.hero h1 .highlight { background: var(--grad); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
.hero p { font-size: 1rem; color: var(--text-sec); max-width: 440px; line-height: 1.8; }
.hero-actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 4px; }
.hero-stats { display: flex; gap: 32px; padding-top: 16px; border-top: 1px solid var(--border); }
.hero-stats .stat .num { font-size: 1.3rem; font-weight: 700; }
.hero-stats .stat .label { font-size: 0.7rem; color: var(--text-muted); }

/* Sections */
.section-label {
  font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em;
  color: var(--purple); background: rgba(139, 92, 246, 0.08); padding: 4px 14px;
  border-radius: 100px; display: inline-block; border: 1px solid rgba(139, 92, 246, 0.08); margin-bottom: 8px;
}
.section-title { font-size: clamp(1.6rem, 3vw, 2.4rem); font-weight: 700; letter-spacing: -0.02em; margin-bottom: 4px; }
.section-title :deep(.highlight) { background: var(--grad); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
.section-desc { color: var(--text-sec); font-size: 0.95rem; max-width: 700px; margin: 0 auto; line-height: 1.7; }
.loading-msg { color: var(--text-muted); font-size: 0.9rem; margin-top: 24px; text-align: center; }

/* Models */
.models-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 16px; margin-top: 24px; }
.model-card { background: var(--card); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 18px 16px 14px; transition: 0.3s; cursor: default; }
.model-card:hover { border-color: rgba(255, 255, 255, 0.08); background: rgba(255, 255, 255, 0.04); }
.model-header { display: flex; justify-content: space-between; align-items: center; }
.model-card .name { font-weight: 600; font-size: 0.95rem; }
.model-card .type-badge {
  font-size: 0.6rem; font-weight: 500; color: var(--cyan);
  background: rgba(34, 211, 238, 0.08); padding: 2px 10px; border-radius: 100px; border: 1px solid rgba(34, 211, 238, 0.08);
}
.model-card .provider { font-size: 0.7rem; color: var(--text-muted); margin: 2px 0 10px; }
.model-card .metrics { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 6px; margin: 8px 0 10px; }
.model-card .metric { background: var(--bg); border-radius: 6px; padding: 6px 4px; text-align: center; border: 1px solid var(--border); }
.model-card .metric .icon { font-size: 0.7rem; display: block; opacity: 0.6; }
.model-card .metric .value { font-weight: 700; font-size: 0.9rem; display: block; color: var(--text); }
.model-card .metric .label { font-size: 0.55rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.04em; display: block; }
.model-card .status { display: flex; align-items: center; gap: 6px; font-size: 0.7rem; color: var(--green); margin-top: 2px; }
.model-card .status .dot { width: 5px; height: 5px; border-radius: 50%; background: var(--green); display: inline-block; }

/* Client */
.client-section { background: var(--card); border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.client-inner { text-align: center; }
.client-features { display: flex; flex-wrap: wrap; gap: 12px; justify-content: center; margin: 16px 0 24px; }
.feature-tag {
  background: var(--card); border: 1px solid var(--border); border-radius: 100px;
  padding: 4px 16px; font-size: 0.75rem; font-weight: 500; color: var(--text-sec); white-space: nowrap;
}
.client-grid { display: flex; flex-wrap: wrap; gap: 20px; margin-top: 24px; justify-content: center; }
.client-card { background: var(--card); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 24px 32px; text-align: center; min-width: 160px; transition: 0.3s; }
.client-card:hover { border-color: rgba(255, 255, 255, 0.08); }
.client-card .icon { font-size: 2rem; }
.client-card .os { font-weight: 600; margin: 6px 0 2px; }
.client-card .ver { font-size: 0.7rem; color: var(--text-muted); }
.client-card .btn { margin-top: 10px; padding: 6px 20px; font-size: 0.8rem; }

/* Tokens Dashboard */
.tokens-grid { display: grid; grid-template-columns: 2fr 1fr; gap: 24px; margin-top: 24px; }
.dash-card { background: var(--card); border: 1px solid var(--border); border-radius: var(--radius); padding: 24px; }
.dash-card .head { display: flex; justify-content: space-between; align-items: center; font-size: 0.8rem; font-weight: 600; color: var(--text-sec); margin-bottom: 16px; }
.dash-card .badge { font-size: 0.65rem; color: var(--text-muted); background: var(--bg); padding: 2px 12px; border-radius: 100px; border: 1px solid var(--border); }
.stat-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px; }
.stat-box { background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 14px; text-align: center; }
.stat-box .num { font-size: 1.4rem; font-weight: 700; }
.stat-box .lbl { font-size: 0.65rem; text-transform: uppercase; color: var(--text-muted); letter-spacing: 0.04em; }
.trend-canvas { width: 100%; height: 100px; border-radius: var(--radius-sm); background: var(--bg); margin-top: 8px; }
.trend-labels { display: flex; justify-content: space-between; font-size: 0.65rem; color: var(--text-muted); padding: 0 4px; margin-top: 4px; }
.bar-chart { display: flex; align-items: flex-end; height: 120px; gap: 8px; padding: 8px 0; justify-content: space-around; }
.bar-item { display: flex; flex-direction: column; align-items: center; flex: 1; }
.bar-item .bar { width: 100%; max-width: 30px; background: var(--purple); border-radius: 2px 2px 0 0; min-height: 8px; transition: 0.3s; opacity: 0.7; }
.bar-item .bar.highlight { background: var(--cyan); opacity: 1; }
.bar-item .label { font-size: 0.55rem; color: var(--text-muted); margin-top: 4px; text-align: center; word-break: break-all; }
.rank-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-top: 24px; }
.rank-list { display: flex; flex-direction: column; gap: 8px; }
.rank-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; background: var(--bg); border-radius: var(--radius-sm); border: 1px solid var(--border); font-size: 0.85rem; }
.rank-item .left { display: flex; align-items: center; gap: 10px; }
.rank-item .left .idx { font-weight: 600; color: var(--text-muted); font-size: 0.7rem; width: 20px; }
.rank-item .value { font-weight: 600; color: var(--cyan); }
.rank-item-model { gap: 12px; }
.rank-metrics { display: flex; gap: 16px; align-items: center; }
.rank-metric { display: flex; flex-direction: column; align-items: flex-end; line-height: 1.2; }
.rank-metric-num { font-weight: 700; font-size: 0.85rem; color: var(--text); }
.rank-metric-label { font-size: 0.55rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.04em; }
.rank-empty { color: var(--text-muted); font-size: 0.8rem; text-align: center; padding: 16px; }

/* Footer */
.footer { border-top: 1px solid var(--border); padding: 40px 0 28px; background: var(--bg); }
.footer-inner { display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 32px; }
.footer .brand { font-weight: 600; color: var(--text); font-size: 1.1rem; display: flex; align-items: center; gap: 8px; }
.footer .brand .brand-text { display: flex; flex-direction: column; line-height: 1.15; }
.footer .brand .brand-name { font-size: 1.1rem; font-weight: 600; }
.footer .brand .brand-suffix { font-size: 0.6rem; font-weight: 400; color: var(--text-sec); letter-spacing: 0; }
.footer .brand .logo { width: 24px; height: 24px; background: var(--grad); border-radius: 6px; display: flex; align-items: center; justify-content: center; font-size: 0.7rem; color: #fff; }
.footer .col h4 { font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-muted); margin-bottom: 10px; }
.footer .col p, .footer .col a { font-size: 0.85rem; color: var(--text-sec); line-height: 1.8; display: block; }
.footer .col a:hover { color: var(--text); }
.footer .col .email { color: var(--purple); }
.footer-tagline { color: var(--text-muted); font-size: 0.85rem; margin-top: 6px; max-width: 260px; }
.footer-sub { color: var(--text-muted); font-size: 0.75rem; margin-top: 4px; }
.footer-bottom { border-top: 1px solid var(--border); padding-top: 20px; margin-top: 12px; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; font-size: 0.75rem; color: var(--text-muted); }

/* Toast */
.toast-container { position: fixed; bottom: 24px; right: 24px; z-index: 9999; display: flex; flex-direction: column; gap: 8px; max-width: 300px; width: 100%; }
.toast {
  background: var(--card); backdrop-filter: blur(16px); border: 1px solid var(--border);
  border-radius: var(--radius-sm); padding: 10px 16px; display: flex; align-items: center; gap: 10px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5); animation: slideIn 0.35s ease; font-size: 0.85rem;
}
.toast .msg { color: var(--text-sec); }
@keyframes slideIn { from { opacity: 0; transform: translateX(30px); } to { opacity: 1; transform: translateX(0); } }

/* Responsive */
@media (max-width: 1024px) {
  .hero-inner { grid-template-columns: 1fr; gap: 30px; text-align: center; }
  .hero-content { align-items: center; }
  .hero p { max-width: 100%; }
  .hero-stats { justify-content: center; }
  .tokens-grid { grid-template-columns: 1fr; }
  .models-grid { grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); }
  .footer-inner { flex-direction: column; gap: 24px; }
  .footer-bottom { flex-direction: column; text-align: center; }
}
@media (max-width: 768px) {
  .nav-links {
    display: none; flex-direction: column; background: var(--bg); padding: 20px;
    border-radius: var(--radius); border: 1px solid var(--border);
    position: absolute; top: 56px; left: 16px; right: 16px; gap: 12px;
  }
  .nav-links.open { display: flex; }
  .nav-toggle { display: flex; }
  .nav-actions .btn { padding: 4px 14px; font-size: 0.7rem; }
  .hero h1 { font-size: 2rem; }
  .models-grid { grid-template-columns: 1fr 1fr; }
  .stat-row { grid-template-columns: 1fr; }
  .client-grid { flex-direction: column; align-items: center; }
  .footer-inner { flex-direction: column; text-align: center; }
  .rank-grid { grid-template-columns: 1fr; }
}
@media (max-width: 480px) {
  .models-grid { grid-template-columns: 1fr; }
  .hero-actions .btn { width: 100%; justify-content: center; }
  .stat-row { grid-template-columns: 1fr; }
}
</style>
