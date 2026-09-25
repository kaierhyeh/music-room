import Constants from 'expo-constants';

const API_PORT = '8080';
const FALLBACK_BASE_URL = `http://localhost:${API_PORT}`;

const host = Constants.expoConfig?.hostUri?.split('/')[0].split(':')[0];

export const DEFAULT_BASE_URL =
  process.env.EXPO_PUBLIC_API_URL || (host ? `http://${host}:${API_PORT}` : FALLBACK_BASE_URL);
