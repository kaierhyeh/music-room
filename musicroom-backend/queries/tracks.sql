-- name: UpsertTrack :one
INSERT INTO tracks (deezer_id, title, artist_name, album_title, duration_sec, preview_url, cover_url, raw)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (deezer_id) DO UPDATE
SET title        = EXCLUDED.title,
    artist_name  = EXCLUDED.artist_name,
    album_title  = EXCLUDED.album_title,
    duration_sec = EXCLUDED.duration_sec,
    preview_url  = EXCLUDED.preview_url,
    cover_url    = EXCLUDED.cover_url,
    raw          = EXCLUDED.raw,
    cached_at    = now()
RETURNING *;

-- name: GetTrack :one
SELECT * FROM tracks WHERE deezer_id = $1;
