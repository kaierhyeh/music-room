-- name: CreatePlaylist :one
INSERT INTO playlists (owner_id, name, description, visibility, license)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPlaylist :one
SELECT * FROM playlists WHERE id = $1;

-- name: GetPlaylistForUpdate :one
SELECT * FROM playlists WHERE id = $1 FOR UPDATE;

-- name: ListPublicPlaylists :many
SELECT p.id, p.owner_id, u.display_name AS owner_display_name,
       p.name, p.description, p.visibility, p.license, p.version, p.created_at
FROM playlists p
JOIN users u ON u.id = p.owner_id
WHERE p.visibility = 'public'
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListPlaylistsOwnedBy :many
SELECT p.id, p.owner_id, u.display_name AS owner_display_name,
       p.name, p.description, p.visibility, p.license, p.version, p.created_at
FROM playlists p
JOIN users u ON u.id = p.owner_id
WHERE p.owner_id = $1
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPlaylistsSharedWith :many
SELECT p.id, p.owner_id, u.display_name AS owner_display_name,
       p.name, p.description, p.visibility, p.license, p.version, p.created_at,
       pc.can_edit
FROM playlist_collaborators pc
JOIN playlists p ON p.id = pc.playlist_id
JOIN users u ON u.id = p.owner_id
WHERE pc.user_id = $1
ORDER BY pc.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdatePlaylist :one
UPDATE playlists
SET name = $2, description = $3, visibility = $4, license = $5
WHERE id = $1
RETURNING *;

-- name: GetPlaylistTracks :many
SELECT
    pt.id,
    pt.track_deezer_id,
    t.title,
    t.artist_name,
    t.preview_url,
    pt.position
FROM playlist_tracks pt
JOIN tracks t ON t.deezer_id = pt.track_deezer_id
WHERE pt.playlist_id = $1
ORDER BY pt.position ASC;

-- name: AddPlaylistTrack :one
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: RemovePlaylistTrack :execrows
DELETE FROM playlist_tracks WHERE playlist_id = $1 AND id = $2;

-- name: BumpPlaylistVersion :one
UPDATE playlists SET version = version + 1 WHERE id = $1 RETURNING version;

-- name: ListTrackPositions :many
SELECT id, position FROM playlist_tracks
WHERE playlist_id = $1 ORDER BY position ASC;

-- name: SetTrackPosition :exec
UPDATE playlist_tracks SET position = $2 WHERE id = $1;

-- name: IsPlaylistCollaborator :one
SELECT EXISTS (
    SELECT 1 FROM playlist_collaborators WHERE playlist_id = $1 AND user_id = $2
) AS is_collaborator;

-- name: GetPlaylistCollaboratorRole :one
SELECT can_edit FROM playlist_collaborators WHERE playlist_id = $1 AND user_id = $2;

-- name: AddPlaylistCollaborator :exec
INSERT INTO playlist_collaborators (playlist_id, user_id, invited_by, can_edit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (playlist_id, user_id) DO UPDATE SET can_edit = EXCLUDED.can_edit;

-- name: ListPlaylistCollaborators :many
SELECT u.id, u.display_name, pc.can_edit
FROM playlist_collaborators pc
JOIN users u ON u.id = pc.user_id
WHERE pc.playlist_id = $1
ORDER BY u.display_name;

-- name: RemovePlaylistCollaborator :execrows
DELETE FROM playlist_collaborators WHERE playlist_id = $1 AND user_id = $2;
