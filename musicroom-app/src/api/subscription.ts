import { api } from './client';

export type Tier = 'free' | 'premium';

export type Subscription = {
  tier: Tier;
  expires_at?: string;
};

export type SubscriptionChange = {
  tier: Tier;
  note?: string;
  changed_at: string;
};

export const subscriptionApi = {
  get: () => api<Subscription>('get', '/users/me/subscription'),
  set: (tier: Tier, note?: string) =>
    api<Subscription>('put', '/users/me/subscription', { tier, note }),
  history: () => api<SubscriptionChange[]>('get', '/users/me/subscription/history'),
};
