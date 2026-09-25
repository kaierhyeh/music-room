import { api } from './client';

export type PlaylistLicense = 'open' | 'invited_only';

export type PlaylistVisibility = 'public' | 'private';

export type Playlist = {
  id: string;
  owner_id: string;
  owner_display_name: string;
  name: string;
  description?: string;
  visibility: PlaylistVisibility;
  license: PlaylistLicense;
  version: number;
  created_at: string;
  can_edit?: boolean;
};

export type Collaborator = { id: string; display_name: string; can_edit: boolean };

export type PlaylistTrack = {
  id: string;
  playlist_id: string;
  track_deezer_id: number;
  position: number;
  title: string;
  artist_name: string;
  preview_url?: string;
  cover_url?: string;
};

export const playlistsApi = {
  list: () => api<Playlist[]>('get', '/playlists'),
  listMine: () => api<Playlist[]>('get', '/playlists/mine'),
  listShared: () => api<Playlist[]>('get', '/playlists/shared'),
  get: (id: string) => api<Playlist>('get', `/playlists/${id}`),
  create: (body: {
    name: string;
    description?: string;
    visibility?: string;
    license?: string;
    collaborators?: { user_id: string; can_edit: boolean }[];
  }) =>
    api<Playlist>('post', '/playlists', body),
  update: (
    id: string,
    body: { name?: string; description?: string; visibility?: PlaylistVisibility; license?: PlaylistLicense },
  ) => api<Playlist>('patch', `/playlists/${id}`, body),
  tracks: (id: string) =>
    api<{ tracks?: PlaylistTrack[] }>('get', `/playlists/${id}`).then((p) => p.tracks ?? []),
  addTrack: (id: string, deezer_id: number) =>
    api<{ version: number }>('post', `/playlists/${id}/tracks`, { deezer_id }),
  moveTrack: (id: string, trackId: string, body: { after_track_id?: string | null; version: number }) =>
    api<{ version: number }>('patch', `/playlists/${id}/tracks/${trackId}`, body),
  deleteTrack: (id: string, trackId: string) =>
    api<{ message: string }>('delete', `/playlists/${id}/tracks/${trackId}`),
  collaborators: (id: string) => api<Collaborator[]>('get', `/playlists/${id}/collaborators`),
  addCollaborator: (id: string, user_id: string, can_edit: boolean) =>
    api<{ message: string }>('post', `/playlists/${id}/collaborators`, { user_id, can_edit }),
  removeCollaborator: (id: string, user_id: string) =>
    api<void>('delete', `/playlists/${id}/collaborators/${user_id}`),
  wsTicket: () => api<{ ticket: string }>('post', '/ws-tickets', {}),
};
