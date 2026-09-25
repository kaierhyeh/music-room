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
