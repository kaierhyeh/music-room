# MusicRoom — Backend

REST + WebSocket API in Go (Echo, sqlc, PostgreSQL).

## Quick start

```bash
cp .env.example .env          # set JWT_SECRET and GOOGLE_CLIENT_ID
make docker-up                # postgres + app + adminer on :5432/:8080/:8081
```

Check:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

Swagger UI: http://localhost:8080/swagger/index.html

## Make targets

| Command | Description |
|---|---|
| `make docker-up` | Full stack (postgres, app, adminer) |
| `make run` | Run server locally (requires `.env`) |
| `make migrate-up` | Apply all migrations |
| `make migrate-down` | Rollback last migration |
| `make test` | Run tests |
| `make e2e` | End-to-end suite against the running stack |
| `make sqlc` | Regenerate Go code from SQL queries |
| `make swagger` | Regenerate `docs/` |
| `make tools` | Install sqlc, migrate, swag |

