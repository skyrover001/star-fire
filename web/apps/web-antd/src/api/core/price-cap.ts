import { requestClient } from '#/api/request';

export interface PriceCap {
  id: string;
  user_id: string;
  model: string;
  max_ippm: number;
  max_oppm: number;
  max_cippm: number;
  created_at: string;
  updated_at: string;
}

/** 获取当前用户所有价格上限配置 */
export async function getPriceCapsApi(): Promise<PriceCap[]> {
  const res = await requestClient.get<{ price_caps: PriceCap[] }>('/user/price-caps');
  return (res as any)?.price_caps ?? [];
}

/** 创建或更新某模型的价格上限 */
export async function upsertPriceCapApi(
  model: string,
  maxIPPM: number,
  maxOPPM: number,
  maxCIPPM: number = 0,
): Promise<PriceCap> {
  return requestClient.put<PriceCap>(`/user/price-caps/${encodeURIComponent(model)}`, {
    max_ippm: maxIPPM,
    max_oppm: maxOPPM,
    max_cippm: maxCIPPM,
  });
}

/** 删除某模型的价格上限（恢复不限价） */
export async function deletePriceCapApi(model: string): Promise<void> {
  return requestClient.delete(`/user/price-caps/${encodeURIComponent(model)}`);
}
