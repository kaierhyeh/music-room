ALTER TABLE users ADD COLUMN token_epoch int NOT NULL DEFAULT 0;

CREATE TABLE revoked_access_tokens (
    jti        uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_revoked_access_tokens_expires ON revoked_access_tokens(expires_at);
