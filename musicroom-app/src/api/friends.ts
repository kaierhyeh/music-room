import { api } from './client';
import { User } from './users';

export type FriendRequest = {
  id: string;
  requester_id: string;
  display_name: string;
  created_at: string;
};

export const friendsApi = {
  list: () => api<User[]>('get', '/friends'),
  requests: () => api<FriendRequest[]>('get', '/friends/requests'),
  send: (user_id: string) => api<{ status: string }>('post', '/friends/requests', { addressee_id: user_id }),
  accept: (id: string) => api<void>('post', `/friends/requests/${id}/accept`, {}),
  reject: (id: string) => api<void>('post', `/friends/requests/${id}/reject`, {}),
};
