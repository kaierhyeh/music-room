-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
    user_id, action, resource_type, resource_id,
    platform, device_model, app_version, ip_address, user_agent, status_code
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
