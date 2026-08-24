import { requestClient } from '#/api/request';

export interface MembershipInfo {
  // 消费者会员
  membership: string;
  expire_at: string;
  vip_upgrade_price: number;
  svip_upgrade_price: number;
  // 消费者限流
  rate_limit_rpm: number;
  rate_limit_tpm: number;
  // 贡献者会员
  contributor_membership: string;
  contributor_expire_at: string;
  contributor_vip_upgrade_price: number;
  contributor_svip_upgrade_price: number;
  // 贡献者连接池
  max_connections: number;
  current_connections: number;
  online_clients: number;
  usage_percent: number;
  vip_price: number;
  svip_price: number;
}

// 查询当前会员状态（消费者 + 贡献者）
export async function getMembershipApi() {
  return requestClient.get<MembershipInfo>('/user/membership');
}

// 购买会员（type: consumer 消费者 / contributor 贡献者）
export async function buyMembershipApi(level: string, type: 'consumer' | 'contributor' = 'consumer') {
  return requestClient.post('/user/membership/buy', { level, type });
}
