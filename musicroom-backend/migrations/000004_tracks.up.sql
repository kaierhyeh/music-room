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
