import axios, { AxiosInstance } from 'axios';
import Constants from 'expo-constants';
import * as Device from 'expo-device';
import { Platform } from 'react-native';
import { getBaseUrl, getTokens, setTokens, clearTokens } from '../storage/storage';

let client: AxiosInstance | null = null;
let onUnauthenticated: (() => void) | null = null;

const DEVICE_MODEL = (() => {
  const model = Device.modelName ?? Device.modelId;
  if (!model) return Device.osName ? `${Device.osName} browser` : 'unknown';
  return Device.manufacturer && !model.startsWith(Device.manufacturer)
    ? `${Device.manufacturer} ${model}`
    : model;
})();

function clientHeaders() {
  return {
    'X-Platform': Platform.OS,
    'X-Device-Model': DEVICE_MODEL,
    'X-App-Version': Constants.expoConfig?.version ?? '1.0.0',
  };
}

export function setUnauthenticatedHandler(cb: () => void) {
  onUnauthenticated = cb;
}

async function buildClient(): Promise<AxiosInstance> {
  const baseURL = await getBaseUrl();
  const instance = axios.create({ baseURL });

  instance.interceptors.request.use(async (config) => {
    const tokens = await getTokens();
    if (tokens) config.headers.Authorization = `Bearer ${tokens.access_token}`;
    Object.assign(config.headers, clientHeaders());
    return config;
  });

  let refreshing = false;
  let queue: Array<() => void> = [];

  instance.interceptors.response.use(
    (res) => res,
    async (error) => {
      const original = error.config;
      if (error.response?.status !== 401 || original._retry) {
        return Promise.reject(error);
      }
      if (refreshing) {
        return new Promise((resolve) => {
          queue.push(() => resolve(instance(original)));
        });
      }
      original._retry = true;
      refreshing = true;
      try {
        const tokens = await getTokens();
        if (!tokens) throw new Error('no tokens');
        const res = await axios.post(
          `${baseURL}/auth/refresh`,
          { refresh_token: tokens.refresh_token },
          { headers: clientHeaders() },
        );
        await setTokens(res.data);
        original.headers.Authorization = `Bearer ${res.data.access_token}`;
        queue.forEach((r) => r());
        queue = [];
        return instance(original);
      } catch {
        await clearTokens();
        onUnauthenticated?.();
        return Promise.reject(error);
      } finally {
        refreshing = false;
      }
    },
  );

  return instance;
}

let _baseUrlSnapshot = '';

export async function getClient(): Promise<AxiosInstance> {
  const baseURL = await getBaseUrl();
  if (!client || _baseUrlSnapshot !== baseURL) {
    client = await buildClient();
    _baseUrlSnapshot = baseURL;
  }
  return client;
}

export class NotApiError extends Error {
  constructor(public baseURL: string) {
    super(`${baseURL} is not the MusicRoom API`);
  }
}

export async function api<T = unknown>(
  method: 'get' | 'post' | 'put' | 'patch' | 'delete',
  path: string,
  data?: unknown,
): Promise<T> {
  const c = await getClient();
  const res = await c[method]<T>(path, data);
  const type = String(res.headers['content-type'] ?? '');
  if (res.status !== 204 && !type.includes('application/json')) {
    throw new NotApiError(String(c.defaults.baseURL));
  }
  return res.data;
}
