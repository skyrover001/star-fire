/**
 * 解析服务器主机地址，用于构建 OpenAI 兼容 API 的 base_url。
 *
 * 优先级：
 * 1. 配置的 serverHost（.env 或 _app.config.js）
 * 2. 若 serverHost 是占位符/空，回退到当前页面 origin（同源，部署到任何域名都自动正确）
 * 3. 自动补全协议前缀（http:// 或 https://）
 */
export function resolveServerHost(serverHost: string): string {
  const host = (serverHost || '').trim();
  // 占位符（如 your-production-host.com）或空值 → 回退到同源
  if (
    !host ||
    host.includes('your-') ||
    (host === 'localhost' && import.meta.env.PROD)
  ) {
    return window.location.origin;
  }
  // 已带协议 → 原样返回
  if (/^https?:\/\//i.test(host)) {
    return host;
  }
  // 裸 IP/域名 → 根据当前页面协议补全
  const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:';
  return `${protocol}//${host}`;
}

/**
 * 构建 OpenAI 兼容 API 的 base_url（形如 https://<host>/v1）。
 */
export function buildApiBaseUrl(serverHost: string): string {
  return `${resolveServerHost(serverHost)}/v1`;
}
