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
