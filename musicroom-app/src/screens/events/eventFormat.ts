import { Platform } from 'react-native';
import * as Location from 'expo-location';
import { MusicEvent } from '../../api/events';

const DAYS = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'];
const MONTHS = ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'];
const pad = (n: number) => String(n).padStart(2, '0');

const hhmm = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}`;
const day = (d: Date) => `${DAYS[d.getDay()]} ${d.getDate()} ${MONTHS[d.getMonth()]}`;

export function formatWindow(startIso: string, endIso: string): string {
  const s = new Date(startIso);
  const e = new Date(endIso);
  const sameDay = s.toDateString() === e.toDateString();
  return `${day(s)} ${hhmm(s)} → ${sameDay ? '' : `${day(e)} `}${hhmm(e)}`;
}

export type VoteWindow = 'upcoming' | 'open' | 'closed';

export function voteWindow(ev: MusicEvent, now = new Date()): VoteWindow | null {
  if (!ev.vote_start_at || !ev.vote_end_at) return null;
  if (now < new Date(ev.vote_start_at)) return 'upcoming';
  if (now > new Date(ev.vote_end_at)) return 'closed';
  return 'open';
}

export function toLocalInput(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${hhmm(d)}`;
}

export function parseLocalInput(text: string): Date | null {
  const m = text.trim().match(/^(\d{4})-(\d{1,2})-(\d{1,2})[ T](\d{1,2}):(\d{2})$/);
  if (!m) return null;
  const [y, mo, d, h, mi] = m.slice(1).map(Number);
  const date = new Date(y, mo - 1, d, h, mi);
  if (date.getMonth() !== mo - 1 || date.getDate() !== d || h > 23) return null;
  return date;
}

export function distanceMeters(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const rad = Math.PI / 180;
  const dLat = (lat2 - lat1) * rad;
  const dLng = (lng2 - lng1) * rad;
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(lat1 * rad) * Math.cos(lat2 * rad) * Math.sin(dLng / 2) ** 2;
  return 6371000 * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

export function formatDistance(m: number): string {
  return m < 1000 ? `${Math.round(m)} M` : `${(m / 1000).toFixed(m < 10000 ? 1 : 0)} KM`;
}

const NOMINATIM = 'https://nominatim.openstreetmap.org';
const NOMINATIM_HEADERS: Record<string, string> = Platform.OS === 'web' ? {} : { 'User-Agent': 'MusicRoom/1.0' };

type Point = { latitude: number; longitude: number };

export async function describePlace(latitude: number, longitude: number): Promise<string | null> {
  try {
    const res = await fetch(`${NOMINATIM}/reverse?format=json&lat=${latitude}&lon=${longitude}`, {
      headers: NOMINATIM_HEADERS,
    });
    const a = (await res.json())?.address;
    if (!a) return null;
    const street = [a.house_number, a.road].filter(Boolean).join(' ');
    const parts = [street || a.amenity, a.city ?? a.town ?? a.village, a.country].filter(Boolean);
    return parts.length ? parts.join(', ') : null;
  } catch {
    return null;
  }
}

export async function findPlace(query: string): Promise<Point | null> {
  const m = query.trim().match(/^(-?\d+(?:\.\d+)?)\s*[,;\s]\s*(-?\d+(?:\.\d+)?)$/);
  if (m) {
    const [latitude, longitude] = [Number(m[1]), Number(m[2])];
    return Math.abs(latitude) <= 90 && Math.abs(longitude) <= 180 ? { latitude, longitude } : null;
  }
  const res = await fetch(`${NOMINATIM}/search?format=json&limit=1&q=${encodeURIComponent(query.trim())}`, {
    headers: NOMINATIM_HEADERS,
  });
  if (!res.ok) throw new Error(`nominatim ${res.status}`);
  const [hit] = await res.json();
  return hit ? { latitude: Number(hit.lat), longitude: Number(hit.lon) } : null;
}

export const formatCoords = (lat: number, lng: number) => `${lat.toFixed(5)}, ${lng.toFixed(5)}`;

export async function currentPosition(): Promise<{ latitude: number; longitude: number }> {
  const { status } = await Location.requestForegroundPermissionsAsync();
  if (status !== 'granted') {
    throw new Error('LOCATION ACCESS IS OFF. ALLOW IT IN YOUR SETTINGS TO USE THIS.');
  }
  try {
    const loc = await Location.getCurrentPositionAsync({});
    return { latitude: loc.coords.latitude, longitude: loc.coords.longitude };
  } catch {
    throw new Error('COULD NOT FIND WHERE YOU ARE. CHECK THAT LOCATION IS ON AND TRY AGAIN.');
  }
}
