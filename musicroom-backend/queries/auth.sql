-- name: GetOAuthAccount :one
SELECT * FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2;

-- name: LinkOAuthAccount :one
INSERT INTO oauth_accounts (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UnlinkOAuthAccount :execrows
DELETE FROM oauth_accounts WHERE user_id = $1 AND provider = $2;

-- name: HasOAuthProvider :one
SELECT EXISTS (
    SELECT 1 FROM oauth_accounts WHERE user_id = $1 AND provider = $2
);

-- name: CountUserLoginMethods :one
SELECT
    (SELECT count(*) FROM oauth_accounts WHERE user_id = $1) AS oauth_count,
    (SELECT password_hash IS NOT NULL FROM users WHERE id = $1) AS has_password;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at, user_agent, ip_address)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: RotateRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now(), replaced_by = $2 WHERE id = $1;

-- name: RevokeTokenFamily :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreateEmailVerification :one
INSERT INTO email_verifications (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ConsumeEmailVerification :one
UPDATE email_verifications
SET consumed_at = now()
WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
RETURNING user_id;

-- name: CreatePasswordReset :one
INSERT INTO password_resets (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ConsumePasswordReset :one
UPDATE password_resets
SET consumed_at = now()
WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
RETURNING user_id;

-- name: CheckAccessToken :one
SELECT u.token_epoch,
       EXISTS (SELECT 1 FROM revoked_access_tokens r WHERE r.jti = $2) AS revoked
FROM users u
WHERE u.id = $1;

-- name: GetTokenEpoch :one
SELECT token_epoch FROM users WHERE id = $1;

-- name: BumpTokenEpoch :exec
UPDATE users SET token_epoch = token_epoch + 1 WHERE id = $1;

-- name: RevokeAccessToken :exec
INSERT INTO revoked_access_tokens (jti, user_id, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (jti) DO NOTHING;

-- name: PurgeExpiredRevokedTokens :exec
DELETE FROM revoked_access_tokens WHERE expires_at < now();
