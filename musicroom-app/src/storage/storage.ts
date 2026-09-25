import AsyncStorage from '@react-native-async-storage/async-storage';
import { tokenStore } from './tokenStore';
import { DEFAULT_BASE_URL } from './defaultBaseUrl';

const KEY_ACCESS = 'mr_access';
const KEY_REFRESH = 'mr_refresh';
const KEY_BASE_URL = 'mr_base_url';

export type Tokens = { access_token: string; refresh_token: string };

export async function getTokens(): Promise<Tokens | null> {
  const [access, refresh] = await Promise.all([
    tokenStore.get(KEY_ACCESS),
    tokenStore.get(KEY_REFRESH),
  ]);
  if (!access || !refresh) return null;
  return { access_token: access, refresh_token: refresh };
}

export async function setTokens(t: Tokens) {
  await Promise.all([
    tokenStore.set(KEY_ACCESS, t.access_token),
    tokenStore.set(KEY_REFRESH, t.refresh_token),
  ]);
}

export async function clearTokens() {
  await Promise.all([
    tokenStore.remove(KEY_ACCESS),
    tokenStore.remove(KEY_REFRESH),
  ]);
}

export async function getBaseUrl(): Promise<string> {
  return (await AsyncStorage.getItem(KEY_BASE_URL)) ?? DEFAULT_BASE_URL;
}

export async function setBaseUrl(url: string) {
  await AsyncStorage.setItem(KEY_BASE_URL, url);
}
