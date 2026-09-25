-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name, email_verified)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserDisplayName :one
SELECT display_name FROM users WHERE id = $1;

-- name: UpdateProfile :one
UPDATE users
SET profile_data       = $2,
    profile_visibility = $3,
    music_preferences  = $4
WHERE id = $1
RETURNING *;

-- name: SetEmailVerified :exec
UPDATE users SET email_verified = true WHERE id = $1;

-- name: SetPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: SearchUsers :many
SELECT id, display_name FROM users
WHERE display_name ILIKE '%' || $1 || '%'
ORDER BY display_name
LIMIT $2 OFFSET $3;

-- name: IncrementFailedLogin :one
UPDATE users
SET failed_login_count   = failed_login_count + 1,
    last_failed_login_at = now()
WHERE id = $1
RETURNING failed_login_count, last_failed_login_at, locked_until;

-- name: LockUser :exec
UPDATE users SET locked_until = $2 WHERE id = $1;

-- name: ResetFailedLogin :exec
UPDATE users
SET failed_login_count = 0, last_failed_login_at = NULL, locked_until = NULL
WHERE id = $1;
