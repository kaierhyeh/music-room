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
