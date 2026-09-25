# MusicRoom — Load benchmark report

## Environment

| Item | Value |
|---|---|
| Machine | Lenovo Yoga Slim 7 Pro 16ACH6 (Arch Linux) |
| CPU | AMD Ryzen 7 5800H — 8 cores / 16 threads |
| RAM | 13 GiB available |
| Stack | Docker Compose 29.7.2 — app + postgres:16-alpine on same host |
| Tool | [k6](https://k6.io) v1.8.1 |
| Run on | 23 Aug 2026, all three scenarios from a single host, limiters lifted; scenario 3 re-run 24 Sep 2026 |

> **How these numbers were obtained.** Every route sits behind a global
> 20 req/s per-IP rate limiter, and `/auth/*` behind a second one at 10 req/min
> with a burst of 3. Driving load from one machine therefore measures *the rate
> limiter*, not the backend — which is exactly what an earlier round of this
> document did, reporting rejection rates of 57 % to 99 % as if they described
> the server. The thresholds are configurable for this reason; every figure
> below comes from a run with them lifted. See *Measuring the backend rather
> than the limiter*.

> PostgreSQL runs in a container and shares host resources — it is the primary bottleneck on concurrent writes (votes, playlist mutations). On a dedicated DB server, REST throughput would increase significantly.

## How to run

```bash
# Install k6 (any platform with Go — same as `make tools`)
go install go.k6.io/k6@latest

# Start the full stack
make docker-up

# Create a test user (needed by auth.js)
curl -s -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"bench@example.com","password":"bench_password_123","display_name":"Bench"}'

# Auth throughput
k6 run benchmarks/auth.js

# Concurrent votes (set env vars first)
export TOKEN=<access_token>
export EVENT_ID=<event_uuid>
export TRACK_ID=<event_track_uuid>
k6 run benchmarks/vote-ws.js

# Concurrent playlist edits
export PLAYLIST_ID=<playlist_uuid>
export TRACK_ID=<playlist_track_uuid>
k6 run benchmarks/playlist-ws.js
```

## Scenario 1 — Auth throughput (`/auth/login`)

bcrypt (cost 12) is CPU-bound, and that is exactly what the measurement shows.

Run with the limiter lifted (see below), so these numbers describe the server.
Every run: **0 % errors**, every login answered with a token.

| Virtual users | Logins/s | p95 latency | Errors |
|---|---|---|---|
| 50 | 35.9 | 513 ms | 0 % |
| 60 | 38.7 | 845 ms | 0 % |
| 100 | 48.0 | 1.48 s | 0 % |
| 200 | **56.3** | 3.48 s | 0 % |

**Reading:** throughput plateaus at **≈56 logins/s** — the CPU spending itself on
bcrypt-12, nothing else. Past that point extra load buys queueing, not
throughput: from 50 to 200 users the rate rises 57 % while p95 latency grows
almost sevenfold. The usable operating point on this machine is **≈36 logins/s**,
where p95 sits at about half a second.

The staged profile the script ships with (ramp to 50) passes both of its own
thresholds: p95 238 ms against a 500 ms budget, 0 % errors against a 1 % budget.

An earlier run of this scenario reported 417 ms per login and inferred "2.4
logins/s per busy core". That figure came from ten requests that slipped through
the limiter's burst while 99.28 % were rejected — it measured one bcrypt hash in
isolation, not the server, and it understated real throughput by more than
twentyfold. Concurrency is what the measurement was missing.

## Scenario 2 — Concurrent votes (WebSocket)

100 clients connect to the same event and vote simultaneously. The `UNIQUE(user_id, event_track_id)` constraint + `ON CONFLICT DO NOTHING` guarantees at-most-one vote per user regardless of concurrency.

| Concurrent VUs | Sessions | WS connect p95 | HTTP p95 | Checks passed | Errors |
|---|---|---|---|---|---|
| 100 | 529 | 1.15 ms | 1.71 ms | 1 058 / 1 058 | 0 % |
| 500 | 4 000 | 24.3 ms | 15.8 ms | 8 000 / 8 000 | 0 % |
| 1 500 | 12 000 | 98.9 ms | 61.3 ms | 24 000 / 24 000 | 0 % |

**Reading:** **1 500 concurrent clients on a single event, not one failure.** The
hub was never pushed to a limit — connection latency degrades gracefully and
stays under 100 ms at p95. Duplicate votes remain at zero throughout: the
composite primary key handles concurrency, as the integration tests assert
independently.

This is the run the limiter had hidden most completely. The previous attempt
accepted 197 handshakes out of 1 439 and called the rest an error rate.

## Scenario 3 — Concurrent playlist edits

20 clients move the same track in the same playlist simultaneously. Optimistic
locking (version field) answers 409 to every editor that read a stale version;
the client re-reads the version and retries, up to five times, after a random
delay that doubles with each attempt (20–50 ms, 40–100 ms, …).

Re-run on 24 Sep 2026, limiters lifted, three consecutive runs:

| Metric | Run 1 | Run 2 | Run 3 |
|---|---|---|---|
| Iterations (30 s) | 604 | 601 | 605 |
| **409 conflicts detected** | 18 | 35 | 22 |
| Retries | 22 | 45 | 30 |
| **Moves still refused after retrying** | **0** | **0** | **0** |
| HTTP p95 | 8.08 ms | 7.87 ms | 7.90 ms |
| Checks passed | 1 230 / 1 230 | 1 247 / 1 247 | 1 240 / 1 240 |

The playlist's version went from 563 to 2 373 across the three runs: exactly
1 810 moves, one per iteration. No update was lost and none was applied twice.
The 2–3.5 % `http_req_failed` is those 409s — the lock working, not a fault.

**Reading:** the earlier version of this scenario retried once, and reported
"134 conflicts, all resolved" next to 1 340 / 1 344 checks — the four failed
checks were retries that hit a second 409. Re-running that script on 24 Sep
2026 gave 213 conflicts and 44 moves refused for good (7 % of iterations).
One retry is not enough when twenty editors race for the same row: the
losers all re-read at the same instant and collide again. Spacing the retries
at random breaks that lockstep, which is also why far fewer first attempts
conflict at all. The mobile app (`PlaylistDetailScreen.moveTrack`) uses the
same five-retry backoff.

The script enforces it: `playlist_unresolved` has a `count==0` threshold, so a
move that runs out of retries fails the run instead of hiding in a ratio.

## Measuring the backend rather than the limiter

The limiter defends the server; during a load test from one host it is the only
thing being measured. Four environment variables move it out of the way, all
read by `internal/config`:

| Variable | Default | Effect |
|---|---|---|
| `RATE_LIMIT_RPS` | `20` | Global ceiling, per source IP |
| `RATE_LIMIT_AUTH_PER_MIN` | `10` | `/auth/*`, per source IP |
| `RATE_LIMIT_AUTH_BURST` | `3` | Burst allowed on `/auth/*` |
| `RATE_LIMIT_DISABLED` | unset | `true` removes every limiter |

```bash
# Raise the ceiling for one run
RATE_LIMIT_RPS=5000 RATE_LIMIT_AUTH_PER_MIN=10000 RATE_LIMIT_AUTH_BURST=500 \
  docker compose up -d app

# Or remove the limiters entirely
RATE_LIMIT_DISABLED=true docker compose up -d app

# ... run the scenarios ...

# Put the defences back
docker compose up -d app
```

Two behaviours are deliberate. Disabling logs a warning at startup, so a server
left in that state announces it in its own logs. And a threshold that is absent,
zero, negative or unparseable falls back to its default instead of turning the
limiter off — a typo in a benchmark script must not silently remove a defence.

Measured effect, 300 requests to `/health` from one source address:

| Configuration | 200 | 429 |
|---|---|---|
| Default | 50 | 250 |
| `RATE_LIMIT_DISABLED=true` | 300 | 0 |
| `RATE_LIMIT_RPS=0` (typo) | 32 | 88 — default still enforced |

Re-running the three scenarios under a raised ceiling is what would turn the
estimates below into measurements. That run has not been done.

## Capacity

Measured, not extrapolated — with one exception, marked as such.

| Service | Capacity on this machine | Basis |
|---|---|---|
| Auth (`/auth/login`, bcrypt) | **56 logins/s** saturated, **36/s** at p95 ≈ 500 ms | measured, 200 VUs |
| WebSocket (events) | **≥1 500** concurrent, 0 failures, connect p95 99 ms | measured — no ceiling found |
| WebSocket (playlists) | same hub, same path | measured via the events run |
| Playlist writes under contention | 20 concurrent editors, 0 moves lost over 1 810 (3 runs) | measured |
| REST reads | not pushed to saturation | the limiter was the only ceiling previously; no dedicated read scenario exists |

Auth is the one hard ceiling, and it is bcrypt by design: cost 12 is a deliberate
~200 ms of CPU per login. Everything else has headroom left at the loads these
scenarios could generate from a single host.

These figures assume a single-instance deployment on the test machine, with
PostgreSQL in a container sharing its resources. A dedicated database server
and/or connection pooling (PgBouncer) would improve write throughput further.

## Optimization levers

| Bottleneck | Mitigation |
|---|---|
| bcrypt on login | The measured ceiling, 56 logins/s. Cost 12→10 would roughly quadruple it, at the price of a weaker hash; a session cache avoids paying it twice |
| DB write contention (votes) | Increase `max_connections`; `pgxpool` max already 4×cores |
| WS memory | Hub stores one `*websocket.Conn` per client — ~10 KB/conn; 1000 conns ≈ 10 MB |
| Slow queries | `EXPLAIN ANALYZE` on hot paths; indexes already on FK + vote unique key |
