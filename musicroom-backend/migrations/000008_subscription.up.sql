ALTER TABLE users ADD COLUMN subscription_tier       subscription_tier NOT NULL DEFAULT 'free';
ALTER TABLE users ADD COLUMN subscription_expires_at timestamptz;

CREATE TABLE subscription_history (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier       subscription_tier NOT NULL,
    note       text,
    changed_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscription_history_user ON subscription_history(user_id, changed_at DESC);
