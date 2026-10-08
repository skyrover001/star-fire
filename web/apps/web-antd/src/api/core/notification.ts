import { requestClient } from '#/api/request';

export interface NotificationItem {
  id: number;
  user_id: string;
  type: string;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

// 获取通知列表
export async function getNotificationsApi(page = 1, size = 20) {
  return requestClient.get<{ data: NotificationItem[]; total: number }>(
    '/user/notifications',
    { params: { page, size } },
  );
}

// 获取未读通知数
export async function getUnreadCountApi() {
  return requestClient.get<{ unread: number }>('/user/notifications/unread');
}

// 标记单条已读
export async function markNotificationReadApi(id: number) {
  return requestClient.put(`/user/notifications/${id}/read`);
}

// 标记全部已读
export async function markAllNotificationsReadApi() {
  return requestClient.put('/user/notifications/read-all');
}
