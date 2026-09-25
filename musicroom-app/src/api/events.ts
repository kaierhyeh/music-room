import { api } from './client';

export type EventLicense = 'open' | 'invited_only' | 'time_and_location';

export type MusicEvent = {
  id: string;
  owner_id: string;
  owner_display_name?: string;
  name: string;
  description?: string;
  visibility: 'public' | 'private';
  license: EventLicense;
  vote_start_at?: string;
  vote_end_at?: string;
  vote_latitude?: number;
  vote_longitude?: number;
  vote_radius_meters?: number;
  created_at: string;
  queue?: EventTrack[];
};

export type EventTrack = {
  id: string;
  event_id: string;
  track_deezer_id: number;
  suggested_by?: string;
  vote_count: number;
  voted: boolean;
  title: string;
  artist_name: string;
  preview_url?: string;
  cover_url?: string;
};

export type CreateEventInput = {
  name: string;
  description?: string;
  visibility?: 'public' | 'private';
  license?: EventLicense;
  vote_start_at?: string;
  vote_end_at?: string;
  vote_latitude?: number;
  vote_longitude?: number;
  vote_radius_meters?: number;
};

export const eventsApi = {
  list: () => api<MusicEvent[]>('get', '/events'),
  get: (id: string) => api<MusicEvent>('get', `/events/${id}`),
  create: (body: CreateEventInput) => api<MusicEvent>('post', '/events', body),
  tracks: (id: string) =>
    api<{ queue?: EventTrack[] }>('get', `/events/${id}`).then((e) => e.queue ?? []),
  suggestTrack: (id: string, deezer_id: number) =>
    api<void>('post', `/events/${id}/tracks`, { deezer_id }),
  vote: (id: string, trackId: string, coords?: { latitude: number; longitude: number }) =>
    api<{ message: string }>('post', `/events/${id}/tracks/${trackId}/vote`, coords ?? {}),
  invite: (id: string, user_id: string) =>
    api<void>('post', `/events/${id}/invitations`, { user_id }),
  invitations: (id: string) =>
    api<{ id: string; display_name: string }[]>('get', `/events/${id}/invitations`),
  wsTicket: () => api<{ ticket: string }>('post', '/ws-tickets', {}),
};
