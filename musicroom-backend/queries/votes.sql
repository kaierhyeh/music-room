-- name: SuggestTrack :one
INSERT INTO event_tracks (event_id, track_deezer_id, suggested_by)
VALUES ($1, $2, $3)
ON CONFLICT (event_id, track_deezer_id) DO NOTHING
RETURNING *;

-- name: CastVote :execrows
INSERT INTO votes (event_track_id, user_id)
VALUES ($1, $2)
ON CONFLICT (event_track_id, user_id) DO NOTHING;

-- name: RemoveVote :execrows
DELETE FROM votes WHERE event_track_id = $1 AND user_id = $2;

-- name: MarkTrackPlayed :exec
UPDATE event_tracks SET played_at = now() WHERE id = $1;

-- name: GetEventQueue :many
SELECT
    et.id,
    et.track_deezer_id,
    t.title,
    t.artist_name,
    t.preview_url,
    COUNT(v.user_id) AS vote_count,
    COALESCE(bool_or(v.user_id = sqlc.arg(viewer_id)::uuid), false)::boolean AS voted,
    et.created_at
FROM event_tracks et
JOIN tracks t ON t.deezer_id = et.track_deezer_id
LEFT JOIN votes v ON v.event_track_id = et.id
WHERE et.event_id = sqlc.arg(event_id)::uuid AND et.played_at IS NULL
GROUP BY et.id, t.deezer_id
ORDER BY vote_count DESC, et.created_at ASC;
