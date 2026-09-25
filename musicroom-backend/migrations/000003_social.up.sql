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
