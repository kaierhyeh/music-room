-- name: CreateEvent :one
INSERT INTO events (
    owner_id, name, description, visibility, license,
    vote_start_at, vote_end_at, vote_latitude, vote_longitude, vote_radius_meters
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetEvent :one
SELECT * FROM events WHERE id = $1;

-- name: ListVisibleEvents :many
SELECT e.id, e.owner_id, u.display_name AS owner_display_name,
       e.name, e.description, e.visibility, e.license,
       e.vote_start_at, e.vote_end_at, e.vote_latitude, e.vote_longitude,
       e.vote_radius_meters, e.created_at
FROM events e
JOIN users u ON u.id = e.owner_id
WHERE e.visibility = 'public'
   OR e.owner_id = sqlc.arg(viewer_id)
   OR EXISTS (
       SELECT 1 FROM event_invitations ei
       WHERE ei.event_id = e.id AND ei.user_id = sqlc.arg(viewer_id)
   )
ORDER BY e.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetEventTrack :one
SELECT * FROM event_tracks WHERE id = $1;

-- name: CreateEventInvitation :exec
INSERT INTO event_invitations (event_id, user_id, invited_by)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: IsEventInvited :one
SELECT EXISTS (
    SELECT 1 FROM event_invitations WHERE event_id = $1 AND user_id = $2
) AS invited;

-- name: ListEventInvitations :many
SELECT u.id, u.display_name
FROM event_invitations ei
JOIN users u ON u.id = ei.user_id
WHERE ei.event_id = $1
ORDER BY u.display_name;
