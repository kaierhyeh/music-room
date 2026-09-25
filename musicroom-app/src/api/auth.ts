import { api } from './client';

export type Tokens = { access_token: string; refresh_token: string };

export const authApi = {
  register: (email: string, password: string, display_name: string) =>
    api<{ message: string }>('post', '/auth/register', { email, password, display_name }),

  login: (email: string, password: string) =>
    api<Tokens>('post', '/auth/login', { email, password }),

  verifyEmail: (token: string) =>
    api<void>('post', '/auth/verify-email', { token }),

  forgotPassword: (email: string) =>
    api<{ message: string }>('post', '/auth/forgot-password', { email }),

  resetPassword: (token: string, password: string) =>
    api<{ message: string }>('post', '/auth/reset-password', { token, password }),

  google: (id_token: string) =>
    api<Tokens>('post', '/auth/google', { id_token }),

  googleExchange: (code: string, code_verifier: string) =>
    api<Tokens>('post', '/auth/google/exchange', code_verifier ? { code, code_verifier } : { code }),

  logout: (refresh_token: string) =>
    api<void>('post', '/auth/logout', { refresh_token }),

  refresh: (refresh_token: string) =>
    api<Tokens>('post', '/auth/refresh', { refresh_token }),
};
