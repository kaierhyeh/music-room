import React, { createContext, useContext, useEffect, useState } from 'react';
import * as WebBrowser from 'expo-web-browser';
import * as Linking from 'expo-linking';
import * as Crypto from 'expo-crypto';
import { getTokens, setTokens, clearTokens, getBaseUrl, type Tokens } from '../storage/storage';
import { authApi } from '../api/auth';
import { usersApi, type User } from '../api/users';
import { setUnauthenticatedHandler } from '../api/client';

WebBrowser.maybeCompleteAuthSession();

type AuthState = {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  loginWithGoogle: () => Promise<void>;
  linkGoogle: () => Promise<void>;
  register: (email: string, password: string, display_name: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
};

async function makePkce(): Promise<{ verifier: string; challenge: string }> {
  const bytes = Crypto.getRandomBytes(32);
  const verifier = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  try {
    const digest = await Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, verifier, {
      encoding: Crypto.CryptoEncoding.BASE64,
    });
    const challenge = digest.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    return { verifier, challenge };
  } catch {
    return { verifier: '', challenge: '' };
  }
}

const AuthContext = createContext<AuthState>(null as unknown as AuthState);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setUnauthenticatedHandler(async () => {
      await clearTokens();
      setUser(null);
    });
    bootstrap();
  }, []);

  async function bootstrap() {
    const tokens = await getTokens();
    if (tokens) {
      try {
        const me = await usersApi.me();
        setUser(me);
      } catch {
        await clearTokens();
      }
    }
    setLoading(false);
  }

  async function login(email: string, password: string) {
    const tokens = await authApi.login(email, password);
    await setTokens(tokens);
    const me = await usersApi.me();
    setUser(me);
  }

  async function loginWithGoogle() {
    const base = (await getBaseUrl()).replace(/\/+$/, '');
    const returnUrl = Linking.createURL('auth');
    const { verifier, challenge } = await makePkce();
    const startUrl = `${base}/auth/google/start?redirect=${encodeURIComponent(returnUrl)}${challenge ? `&code_challenge=${challenge}` : ''}`;
    const result = await WebBrowser.openAuthSessionAsync(startUrl, returnUrl);
    if (result.type !== 'success') return;
    const { queryParams } = Linking.parse(result.url);
    if (typeof queryParams?.error === 'string') {
      throw new Error(queryParams.error.replace(/_/g, ' '));
    }
    if (typeof queryParams?.code !== 'string') {
      throw new Error('no code returned');
    }
    const tokens = await authApi.googleExchange(queryParams.code, verifier);
    await setTokens(tokens);
    setUser(await usersApi.me());
  }

  async function linkGoogle() {
    const base = (await getBaseUrl()).replace(/\/+$/, '');
    const { ticket } = await usersApi.googleLinkTicket();
    const returnUrl = Linking.createURL('auth');
    const { challenge } = await makePkce();
    const startUrl = `${base}/auth/google/start?redirect=${encodeURIComponent(returnUrl)}&link=${encodeURIComponent(ticket)}${challenge ? `&code_challenge=${challenge}` : ''}`;
    const result = await WebBrowser.openAuthSessionAsync(startUrl, returnUrl);
    if (result.type !== 'success') return;
    const { queryParams } = Linking.parse(result.url);
    if (typeof queryParams?.error === 'string') {
      throw new Error(
        queryParams.error === 'already_linked_elsewhere'
          ? 'that google account is already linked to another user'
          : queryParams.error.replace(/_/g, ' '),
      );
    }
    setUser(await usersApi.me());
  }

  async function register(email: string, password: string, display_name: string) {
    await authApi.register(email, password, display_name);
  }

  async function logout() {
    const tokens = await getTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refresh_token);
      } catch {
      }
    }
    await clearTokens();
    setUser(null);
  }

  async function refreshUser() {
    const me = await usersApi.me();
    setUser(me);
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, loginWithGoogle, linkGoogle, register, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
