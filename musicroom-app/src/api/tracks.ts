import { api } from './client';

export type Track = {
  deezer_id: number;
  title: string;
  artist_name: string;
  album_title?: string;
  duration_sec?: number;
  preview_url?: string;
  cover_url?: string;
};

export const tracksApi = {
  search: (q: string) => api<Track[]>('get', `/tracks/search?q=${encodeURIComponent(q)}`),
  preview: (deezer_id: number) =>
    api<{ preview_url: string }>('get', `/tracks/${deezer_id}/preview`).then((r) => r.preview_url),
};
