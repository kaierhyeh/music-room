-- name: GetEffectiveTier :one
SELECT CASE
         WHEN subscription_tier = 'premium'
          AND (subscription_expires_at IS NULL OR subscription_expires_at > now())
         THEN 'premium'::subscription_tier
         ELSE 'free'::subscription_tier
       END AS tier
FROM users
WHERE id = $1;

-- name: GetSubscription :one
SELECT subscription_tier, subscription_expires_at FROM users WHERE id = $1;

-- name: SetSubscription :one
UPDATE users SET subscription_tier = $2, subscription_expires_at = $3
WHERE id = $1
RETURNING subscription_tier, subscription_expires_at;

-- name: RecordSubscriptionChange :exec
INSERT INTO subscription_history (user_id, tier, note) VALUES ($1, $2, $3);

-- name: ListSubscriptionHistory :many
SELECT tier, note, changed_at FROM subscription_history
WHERE user_id = $1
ORDER BY changed_at DESC
LIMIT 20;
