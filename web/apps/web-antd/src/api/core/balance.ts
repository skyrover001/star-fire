import { requestClient } from '#/api/request';

export interface BalanceInfo {
  balance: number;
  total_spent: number;
}

export interface RechargeOrder {
  order_id: string;
  amount: number;
  qr_code: string;
  payment_url: string;
  status: string;
  created_at: string;
}

export interface RechargeRecord {
  id: number;
  order_id: string;
  user_id: string;
  amount: number;
  payment_method: string;
  status: string;
  qr_code: string;
  created_at: string;
  completed_at: string;
}

export interface RechargeHistory {
  orders: RechargeRecord[];
  total: number;
}

/** 获取用户余额和总消费 */
export async function getBalanceApi(): Promise<BalanceInfo> {
  return requestClient.get<BalanceInfo>('/user/balance');
}

/** 创建充值订单(模拟) */
export async function createRechargeOrderApi(
  amount: number,
  paymentMethod: string,
): Promise<RechargeOrder> {
  return requestClient.post<RechargeOrder>('/user/recharge', {
    amount,
    payment_method: paymentMethod,
  });
}

/** 确认充值(模拟扫码支付成功) */
export async function confirmRechargeApi(orderId: string): Promise<{
  order_id: string;
  status: string;
  balance: number;
  message: string;
}> {
  return requestClient.post('/user/recharge/confirm', {
    order_id: orderId,
  });
}

/** 获取充值历史 */
export async function getRechargeHistoryApi(): Promise<RechargeHistory> {
  return requestClient.get<RechargeHistory>('/user/recharge/history');
}
