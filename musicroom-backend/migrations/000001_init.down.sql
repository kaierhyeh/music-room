DROP FUNCTION IF EXISTS set_updated_at();

DROP TYPE IF EXISTS subscription_tier;
DROP TYPE IF EXISTS friend_request_status;
DROP TYPE IF EXISTS playlist_license;
DROP TYPE IF EXISTS event_license;
DROP TYPE IF EXISTS visibility;
DROP TYPE IF EXISTS oauth_provider;

DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS pgcrypto;
