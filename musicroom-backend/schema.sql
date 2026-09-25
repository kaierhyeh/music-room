CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE oauth_provider        AS ENUM ('google');
CREATE TYPE visibility            AS ENUM ('public', 'private');
CREATE TYPE event_license         AS ENUM ('open', 'invited_only', 'time_and_location');
CREATE TYPE playlist_license      AS ENUM ('open', 'invited_only');
CREATE TYPE friend_request_status AS ENUM ('pending', 'accepted', 'rejected');
CREATE TYPE subscription_tier     AS ENUM ('free', 'premium');

CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email                citext NOT NULL UNIQUE,
    password_hash        text,
    email_verified       boolean NOT NULL DEFAULT false,
    display_name         text NOT NULL,
    profile_data         jsonb NOT NULL DEFAULT '{}'::jsonb,
    profile_visibility   jsonb NOT NULL DEFAULT '{}'::jsonb,
    music_preferences    jsonb NOT NULL DEFAULT '{}'::jsonb,
    failed_login_count   int NOT NULL DEFAULT 0,
    last_failed_login_at timestamptz,
    locked_until         timestamptz,
    token_epoch          int NOT NULL DEFAULT 0,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_display_name_trgm ON users USING gin (display_name gin_trgm_ops);
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE oauth_accounts (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         oauth_provider NOT NULL,
    provider_user_id text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider)
);

CREATE TABLE email_verifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_verifications_user ON email_verifications(user_id);

CREATE TABLE password_resets (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_password_resets_user ON password_resets(user_id);

CREATE TABLE refresh_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    family_id   uuid NOT NULL,
    replaced_by uuid REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    user_agent  text,
    ip_address  inet,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user   ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);

CREATE TABLE revoked_access_tokens (
    jti        uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_revoked_access_tokens_expires ON revoked_access_tokens(expires_at);

CREATE TABLE friend_requests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       friend_request_status NOT NULL DEFAULT 'pending',
    responded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (requester_id <> addressee_id)
);
CREATE UNIQUE INDEX uq_friend_requests_pending
    ON friend_requests(requester_id, addressee_id) WHERE status = 'pending';
CREATE INDEX idx_friend_requests_addressee ON friend_requests(addressee_id, status);

CREATE TABLE friendships (
    user_id_a  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_id_b  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id_a, user_id_b),
    CHECK (user_id_a < user_id_b)
);
CREATE INDEX idx_friendships_b ON friendships(user_id_b);

CREATE TABLE tracks (
    deezer_id    bigint PRIMARY KEY,
    title        text NOT NULL,
    artist_name  text NOT NULL,
    album_title  text,
    duration_sec int,
    preview_url  text,
    cover_url    text,
    raw          jsonb,
    cached_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE events (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text NOT NULL,
    description text,
    visibility  visibility NOT NULL DEFAULT 'public',
    license     event_license NOT NULL DEFAULT 'open',
    vote_start_at      timestamptz,
    vote_end_at        timestamptz,
    vote_latitude      double precision,
    vote_longitude     double precision,
    vote_radius_meters int,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_event_time_location CHECK (
        license <> 'time_and_location' OR (
            vote_start_at IS NOT NULL AND vote_end_at IS NOT NULL
            AND vote_latitude IS NOT NULL AND vote_longitude IS NOT NULL
            AND vote_radius_meters IS NOT NULL AND vote_radius_meters > 0
            AND vote_end_at > vote_start_at
        )
    )
);
CREATE INDEX idx_events_public ON events(created_at DESC) WHERE visibility = 'public';
CREATE INDEX idx_events_owner  ON events(owner_id);
CREATE TRIGGER trg_events_updated_at BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE event_invitations (
    event_id   uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invited_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);
CREATE INDEX idx_event_invitations_user ON event_invitations(user_id);

CREATE TABLE event_tracks (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    track_deezer_id bigint NOT NULL REFERENCES tracks(deezer_id),
    suggested_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    played_at       timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, track_deezer_id)
);
CREATE INDEX idx_event_tracks_queue ON event_tracks(event_id) WHERE played_at IS NULL;

CREATE TABLE votes (
    event_track_id uuid NOT NULL REFERENCES event_tracks(id) ON DELETE CASCADE,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_track_id, user_id)
);

CREATE TABLE playlists (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text NOT NULL,
    description text,
    visibility  visibility NOT NULL DEFAULT 'public',
    license     playlist_license NOT NULL DEFAULT 'open',
    version     integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_playlists_public ON playlists(created_at DESC) WHERE visibility = 'public';
CREATE INDEX idx_playlists_owner  ON playlists(owner_id);
CREATE TRIGGER trg_playlists_updated_at BEFORE UPDATE ON playlists
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE playlist_collaborators (
    playlist_id uuid NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invited_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    can_edit    boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (playlist_id, user_id)
);
CREATE INDEX idx_playlist_collaborators_user ON playlist_collaborators(user_id);

CREATE TABLE playlist_tracks (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    playlist_id     uuid NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_deezer_id bigint NOT NULL REFERENCES tracks(deezer_id),
    position        double precision NOT NULL,
    added_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (playlist_id, track_deezer_id)
);
CREATE INDEX idx_playlist_tracks_order ON playlist_tracks(playlist_id, position);

CREATE TABLE audit_logs (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id       uuid REFERENCES users(id) ON DELETE SET NULL,
    action        text NOT NULL,
    resource_type text,
    resource_id   text,
    platform      text,
    device_model  text,
    app_version   text,
    ip_address    inet,
    user_agent    text,
    status_code   int,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_logs_user    ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

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
