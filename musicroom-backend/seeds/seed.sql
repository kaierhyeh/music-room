BEGIN;

INSERT INTO users (
    email, password_hash, email_verified, display_name,
    profile_data, profile_visibility, music_preferences,
    subscription_tier, subscription_expires_at
) VALUES
  ('alice@musicroom.local', crypt('Password123!', gen_salt('bf', 12)), true, 'Alice Martin',
   '{"bio": "I have a premium account and my profile is fully public. I own the public playlist Electronic Mix, the private playlist Late Night Drafts (Carol edits it, Bob only views it) and the public event Electronic Listening Session. I can edit Bob''s playlist Jazz Standards, I can only view Carol''s playlist Downtempo Sketches, and I am invited to Bob''s event Private Jazz Evening.", "location": "Paris"}',
   '{"bio": "public", "location": "public", "music_preferences": "public"}',
   '{"genres": ["electronic", "house", "disco"], "artists": ["Daft Punk", "Justice", "Moderat"]}',
   'premium', now() + interval '30 days'),

  ('bob@musicroom.local', crypt('Password123!', gen_salt('bf', 12)), true, 'Bob Dupont',
   '{"bio": "I have a free account. My bio and music preferences are public, my location is visible to friends only. I own the invite-only playlist Jazz Standards (Alice and Carol are collaborators) and the private, invite-only event Private Jazz Evening. Alice shared her private playlist Late Night Drafts with me, to view only.", "location": "Lyon"}',
   '{"bio": "public", "location": "friends", "music_preferences": "public"}',
   '{"genres": ["jazz", "soul"], "artists": ["John Coltrane", "Miles Davis", "Nina Simone"]}',
   'free', NULL),

  ('carol@musicroom.local', crypt('Password123!', gen_salt('bf', 12)), true, 'Carol Nguyen',
   '{"bio": "I have a free account. My bio and music preferences are visible to friends only, my location is private. I own the private playlist Trip-Hop Selection, the private playlist Downtempo Sketches (Alice can only view it) and the public event Park Session, where voting is limited by time and location. I can edit Bob''s Jazz Standards and Alice''s Late Night Drafts.", "location": "Nantes"}',
   '{"bio": "friends", "location": "private", "music_preferences": "friends"}',
   '{"genres": ["trip-hop", "downtempo"], "artists": ["Portishead", "Massive Attack", "Air"]}',
   'free', NULL),

  ('dave@musicroom.local', crypt('Password123!', gen_salt('bf', 12)), true, 'Dave Okafor',
   '{"bio": "I have a free account and my profile is fully public. I own the public playlist Hip-Hop and Afrobeat and the public event Hip-Hop Session, whose voting window has closed.", "location": "Marseille"}',
   '{"bio": "public", "location": "public", "music_preferences": "public"}',
   '{"genres": ["hip-hop", "afrobeat"], "artists": ["A Tribe Called Quest", "Fela Kuti"]}',
   'free', NULL),

  ('eve@musicroom.local', crypt('Password123!', gen_salt('bf', 12)), true, 'Eve Lindqvist',
   '{"bio": "I have a free account and my profile is fully private. I do not own any playlists or events.", "location": "Stockholm"}',
   '{"bio": "private", "location": "private", "music_preferences": "private"}',
   '{"genres": ["ambient"], "artists": ["Brian Eno"]}',
   'free', NULL),

  ('frank@musicroom.local', crypt('Password123!', gen_salt('bf', 12)), true, 'Frank Rossi',
   '{"bio": "My premium subscription has expired, so my account is now treated as free. My profile is fully public. I do not own any playlists or events.", "location": "Lille"}',
   '{"bio": "public", "location": "public", "music_preferences": "public"}',
   '{"genres": ["rock"], "artists": ["Tame Impala"]}',
   'premium', now() - interval '2 days')
ON CONFLICT (email) DO UPDATE SET
    password_hash           = EXCLUDED.password_hash,
    email_verified          = EXCLUDED.email_verified,
    display_name            = EXCLUDED.display_name,
    profile_data            = EXCLUDED.profile_data,
    profile_visibility      = EXCLUDED.profile_visibility,
    music_preferences       = EXCLUDED.music_preferences,
    subscription_tier       = EXCLUDED.subscription_tier,
    subscription_expires_at = EXCLUDED.subscription_expires_at;

DELETE FROM playlists WHERE owner_id IN (SELECT id FROM users WHERE email LIKE '%@musicroom.local');
DELETE FROM events    WHERE owner_id IN (SELECT id FROM users WHERE email LIKE '%@musicroom.local');
DELETE FROM subscription_history WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@musicroom.local');
DELETE FROM friend_requests
 WHERE requester_id IN (SELECT id FROM users WHERE email LIKE '%@musicroom.local')
    OR addressee_id IN (SELECT id FROM users WHERE email LIKE '%@musicroom.local');

INSERT INTO friendships (user_id_a, user_id_b)
SELECT LEAST(a.id, b.id), GREATEST(a.id, b.id)
FROM (VALUES
    ('alice@musicroom.local', 'bob@musicroom.local'),
    ('alice@musicroom.local', 'carol@musicroom.local'),
    ('bob@musicroom.local',   'dave@musicroom.local'),
    ('carol@musicroom.local', 'dave@musicroom.local')
) AS pair(ea, eb)
JOIN users a ON a.email = pair.ea
JOIN users b ON b.email = pair.eb
ON CONFLICT DO NOTHING;

INSERT INTO friend_requests (requester_id, addressee_id, status, responded_at)
SELECT r.id, s.id, req.status::friend_request_status,
       CASE WHEN req.status = 'pending' THEN NULL ELSE now() - interval '3 days' END
FROM (VALUES
    ('eve@musicroom.local',   'alice@musicroom.local', 'pending'),
    ('frank@musicroom.local', 'bob@musicroom.local',   'pending'),
    ('dave@musicroom.local',  'eve@musicroom.local',   'rejected')
) AS req(requester, addressee, status)
JOIN users r ON r.email = req.requester
JOIN users s ON s.email = req.addressee;

INSERT INTO subscription_history (user_id, tier, note, changed_at)
SELECT u.id, h.tier::subscription_tier, h.note, now() - h.ago
FROM (VALUES
    ('alice@musicroom.local', 'free',    'seed: account created',      interval '90 days'),
    ('alice@musicroom.local', 'premium', 'seed: upgraded to premium',  interval '10 days'),
    ('frank@musicroom.local', 'premium', 'seed: upgraded to premium',  interval '40 days')
) AS h(email, tier, note, ago)
JOIN users u ON u.email = h.email;

WITH owner AS (SELECT id FROM users WHERE email = 'alice@musicroom.local'),
ins AS (
    INSERT INTO playlists (owner_id, name, description, visibility, license)
    SELECT id, 'Electronic Mix', 'A selection of electronic and house tracks.', 'public', 'open' FROM owner
    RETURNING id
)
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
SELECT ins.id, t.deezer_id,
       row_number() OVER (ORDER BY t.deezer_id) * 1024.0,
       (SELECT id FROM owner)
FROM ins, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Daft Punk', 'Justice', 'LCD Soundsystem', 'Moderat', 'Bonobo', 'Caribou',
        'Four Tet')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

WITH owner AS (SELECT id FROM users WHERE email = 'bob@musicroom.local'),
ins AS (
    INSERT INTO playlists (owner_id, name, description, visibility, license)
    SELECT id, 'Jazz Standards', 'A selection of jazz and soul standards.', 'public', 'invited_only' FROM owner
    RETURNING id
),
collab AS (
    INSERT INTO playlist_collaborators (playlist_id, user_id, invited_by, can_edit)
    SELECT ins.id, u.id, (SELECT id FROM owner), true
    FROM ins, users u
    WHERE u.email IN ('alice@musicroom.local', 'carol@musicroom.local')
    RETURNING 1
)
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
SELECT ins.id, t.deezer_id,
       row_number() OVER (ORDER BY t.deezer_id) * 1024.0,
       (SELECT id FROM owner)
FROM ins, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Nina Simone', 'John Coltrane', 'Miles Davis', 'Khruangbin')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

WITH owner AS (SELECT id FROM users WHERE email = 'carol@musicroom.local'),
ins AS (
    INSERT INTO playlists (owner_id, name, description, visibility, license)
    SELECT id, 'Trip-Hop Selection', 'A personal selection of trip-hop and downtempo tracks.', 'private', 'open' FROM owner
    RETURNING id
)
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
SELECT ins.id, t.deezer_id,
       row_number() OVER (ORDER BY t.deezer_id) * 1024.0,
       (SELECT id FROM owner)
FROM ins, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Portishead', 'Massive Attack', 'Air', 'Radiohead')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

WITH owner AS (SELECT id FROM users WHERE email = 'dave@musicroom.local'),
ins AS (
    INSERT INTO playlists (owner_id, name, description, visibility, license)
    SELECT id, 'Hip-Hop and Afrobeat', 'A selection of hip-hop and afrobeat tracks.', 'public', 'open' FROM owner
    RETURNING id
)
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
SELECT ins.id, t.deezer_id,
       row_number() OVER (ORDER BY t.deezer_id) * 1024.0,
       (SELECT id FROM owner)
FROM ins, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Kendrick Lamar', 'A Tribe Called Quest', 'Fela Kuti', 'Tame Impala')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

WITH owner AS (SELECT id FROM users WHERE email = 'alice@musicroom.local'),
ins AS (
    INSERT INTO playlists (owner_id, name, description, visibility, license)
    SELECT id, 'Late Night Drafts', 'Private work in progress. Carol can edit it, Bob can only listen.', 'private', 'invited_only' FROM owner
    RETURNING id
),
collab AS (
    INSERT INTO playlist_collaborators (playlist_id, user_id, invited_by, can_edit)
    SELECT ins.id, u.id, (SELECT id FROM owner), inv.can_edit
    FROM ins, (VALUES
        ('carol@musicroom.local', true),
        ('bob@musicroom.local',   false)
    ) AS inv(email, can_edit)
    JOIN users u ON u.email = inv.email
    RETURNING 1
)
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
SELECT ins.id, t.deezer_id,
       row_number() OVER (ORDER BY t.deezer_id) * 1024.0,
       (SELECT id FROM owner)
FROM ins, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Moderat', 'Bonobo', 'Massive Attack')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

WITH owner AS (SELECT id FROM users WHERE email = 'carol@musicroom.local'),
ins AS (
    INSERT INTO playlists (owner_id, name, description, visibility, license)
    SELECT id, 'Downtempo Sketches', 'Private, shared with Alice to listen only.', 'private', 'invited_only' FROM owner
    RETURNING id
),
collab AS (
    INSERT INTO playlist_collaborators (playlist_id, user_id, invited_by, can_edit)
    SELECT ins.id, u.id, (SELECT id FROM owner), false
    FROM ins, users u
    WHERE u.email = 'alice@musicroom.local'
    RETURNING 1
)
INSERT INTO playlist_tracks (playlist_id, track_deezer_id, position, added_by)
SELECT ins.id, t.deezer_id,
       row_number() OVER (ORDER BY t.deezer_id) * 1024.0,
       (SELECT id FROM owner)
FROM ins, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Caribou', 'Khruangbin', 'Four Tet')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

WITH owner AS (SELECT id FROM users WHERE email = 'alice@musicroom.local'),
ev AS (
    INSERT INTO events (owner_id, name, description, visibility, license)
    SELECT id, 'Electronic Listening Session', 'Public event. Anyone can suggest and vote for tracks.', 'public', 'open' FROM owner
    RETURNING id
),
et AS (
    INSERT INTO event_tracks (event_id, track_deezer_id, suggested_by, played_at)
    SELECT ev.id, t.deezer_id, (SELECT id FROM owner),
           CASE WHEN row_number() OVER (ORDER BY t.deezer_id) = 1
                THEN now() - interval '20 minutes' END
    FROM ev, (
        SELECT DISTINCT ON (artist_name) deezer_id
        FROM tracks WHERE artist_name IN (
            'Daft Punk', 'Justice', 'LCD Soundsystem', 'Moderat', 'Bonobo')
        ORDER BY artist_name, cached_at DESC, deezer_id
    ) t
    RETURNING id, track_deezer_id
)
INSERT INTO votes (event_track_id, user_id)
SELECT r.id, v.id
FROM (SELECT id, row_number() OVER (ORDER BY track_deezer_id) AS rn FROM et) r
CROSS JOIN (
    SELECT id, row_number() OVER (ORDER BY email) AS vn FROM users
    WHERE email IN ('bob@musicroom.local', 'carol@musicroom.local', 'dave@musicroom.local')
) v
WHERE v.vn <= GREATEST(4 - r.rn, 0);

WITH owner AS (SELECT id FROM users WHERE email = 'bob@musicroom.local'),
ev AS (
    INSERT INTO events (owner_id, name, description, visibility, license)
    SELECT id, 'Private Jazz Evening', 'Private event. Only invited users can join and vote.', 'private', 'invited_only' FROM owner
    RETURNING id
),
inv AS (
    INSERT INTO event_invitations (event_id, user_id, invited_by)
    SELECT ev.id, u.id, (SELECT id FROM owner)
    FROM ev, users u
    WHERE u.email IN ('alice@musicroom.local', 'carol@musicroom.local')
    RETURNING 1
),
et AS (
    INSERT INTO event_tracks (event_id, track_deezer_id, suggested_by)
    SELECT ev.id, t.deezer_id, (SELECT id FROM owner)
    FROM ev, (
        SELECT DISTINCT ON (artist_name) deezer_id
        FROM tracks WHERE artist_name IN (
            'Nina Simone', 'John Coltrane', 'Miles Davis')
        ORDER BY artist_name, cached_at DESC, deezer_id
    ) t
    RETURNING id, track_deezer_id
)
INSERT INTO votes (event_track_id, user_id)
SELECT r.id, v.id
FROM (SELECT id, row_number() OVER (ORDER BY track_deezer_id) AS rn FROM et) r
CROSS JOIN (
    SELECT id, row_number() OVER (ORDER BY email) AS vn FROM users
    WHERE email IN ('alice@musicroom.local', 'carol@musicroom.local')
) v
WHERE v.vn <= GREATEST(3 - r.rn, 0);

WITH owner AS (SELECT id FROM users WHERE email = 'carol@musicroom.local'),
ev AS (
    INSERT INTO events (owner_id, name, description, visibility, license,
                        vote_start_at, vote_end_at,
                        vote_latitude, vote_longitude, vote_radius_meters)
    SELECT id, 'Park Session', 'Voting is open for a limited time and only within 1 km of the venue.', 'public', 'time_and_location',
           now() - interval '1 hour', now() + interval '6 hours',
           48.8674, 2.3636, 1000
    FROM owner
    RETURNING id
),
et AS (
    INSERT INTO event_tracks (event_id, track_deezer_id, suggested_by)
    SELECT ev.id, t.deezer_id, (SELECT id FROM owner)
    FROM ev, (
        SELECT DISTINCT ON (artist_name) deezer_id
        FROM tracks WHERE artist_name IN (
            'Caribou', 'Four Tet', 'Khruangbin', 'Tame Impala')
        ORDER BY artist_name, cached_at DESC, deezer_id
    ) t
    RETURNING id, track_deezer_id
)
INSERT INTO votes (event_track_id, user_id)
SELECT r.id, v.id
FROM (SELECT id, row_number() OVER (ORDER BY track_deezer_id) AS rn FROM et) r
CROSS JOIN (
    SELECT id, row_number() OVER (ORDER BY email) AS vn FROM users
    WHERE email IN ('alice@musicroom.local', 'dave@musicroom.local')
) v
WHERE v.vn <= GREATEST(3 - r.rn, 0);

WITH owner AS (SELECT id FROM users WHERE email = 'dave@musicroom.local'),
ev AS (
    INSERT INTO events (owner_id, name, description, visibility, license,
                        vote_start_at, vote_end_at,
                        vote_latitude, vote_longitude, vote_radius_meters)
    SELECT id, 'Hip-Hop Session', 'The voting window for this event has closed.', 'public', 'time_and_location',
           now() - interval '2 days', now() - interval '1 day',
           48.8674, 2.3636, 500
    FROM owner
    RETURNING id
)
INSERT INTO event_tracks (event_id, track_deezer_id, suggested_by)
SELECT ev.id, t.deezer_id, (SELECT id FROM owner)
FROM ev, (
    SELECT DISTINCT ON (artist_name) deezer_id
    FROM tracks WHERE artist_name IN (
        'Kendrick Lamar', 'A Tribe Called Quest', 'Fela Kuti')
    ORDER BY artist_name, cached_at DESC, deezer_id
) t;

COMMIT;
