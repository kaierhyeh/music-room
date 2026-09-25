import { api } from './client';

export type FieldVisibility = 'public' | 'friends' | 'private';
export type ProfileData = {
  age?: number;
  location?: string;
  bio?: string;
};

export type MusicPreferences = {
  genres?: string[];
  artists?: string[];
};

export type User = {
  id: string;
  email?: string;
  display_name: string;
  google_linked?: boolean;
  profile_data?: ProfileData;
  profile_visibility?: Record<string, FieldVisibility>;
  music_preferences?: MusicPreferences;
};

export const usersApi = {
  me: () => api<User>('get', '/users/me'),

  update: (
    body: Partial<
      Pick<User, 'display_name' | 'profile_data' | 'profile_visibility' | 'music_preferences'>
    >,
  ) =>
    api<User>('patch', '/users/me', body),

  getById: (id: string) => api<User>('get', `/users/${id}`),

  linkGoogle: (id_token: string) =>
    api<{ message: string }>('post', '/users/me/oauth/google', { id_token }),

  googleLinkTicket: () =>
    api<{ ticket: string }>('post', '/users/me/oauth/google/link-ticket'),
};
