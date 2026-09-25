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
