-- name: AreFriends :one
SELECT EXISTS (
    SELECT 1 FROM friendships
    WHERE (user_id_a = sqlc.arg(user_a) AND user_id_b = sqlc.arg(user_b))
       OR (user_id_a = sqlc.arg(user_b) AND user_id_b = sqlc.arg(user_a))
) AS are_friends;

-- name: CreateFriendRequest :one
INSERT INTO friend_requests (requester_id, addressee_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetFriendRequest :one
SELECT * FROM friend_requests WHERE id = $1;

-- name: GetPendingRequestBetween :one
SELECT * FROM friend_requests
WHERE status = 'pending'
  AND ((requester_id = sqlc.arg(u1) AND addressee_id = sqlc.arg(u2))
    OR (requester_id = sqlc.arg(u2) AND addressee_id = sqlc.arg(u1)))
LIMIT 1;

-- name: SetFriendRequestStatus :exec
UPDATE friend_requests SET status = $2, responded_at = now() WHERE id = $1;

-- name: CreateFriendship :exec
INSERT INTO friendships (user_id_a, user_id_b)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListFriends :many
SELECT u.id, u.display_name
FROM friendships f
JOIN users u ON u.id = CASE WHEN f.user_id_a = $1 THEN f.user_id_b ELSE f.user_id_a END
WHERE f.user_id_a = $1 OR f.user_id_b = $1
ORDER BY u.display_name;

-- name: ListIncomingRequests :many
SELECT fr.id, fr.requester_id, u.display_name, fr.created_at
FROM friend_requests fr
JOIN users u ON u.id = fr.requester_id
WHERE fr.addressee_id = $1 AND fr.status = 'pending'
ORDER BY fr.created_at DESC;
