# MusicRoom

Collaborative music platform: live voting events (**Music Track Vote**), real-time multi-user playlists (**Music Playlist Editor**), friends, Google OAuth, and Free/Premium subscriptions.

| Directory | Stack |
| :--- | :--- |
| `musicroom-backend/` | Go 1.25 + Echo v4 + PostgreSQL 16 (`sqlc`, `golang-migrate`) + Gorilla WebSocket |
| `musicroom-app/` | React Native / Expo **SDK 57** + TypeScript (Android, iOS, Responsive Web) |

---

## Quick Start

### 1. Configure Environment Variables (`musicroom-backend/.env`)

```bash
cd musicroom-backend
cp .env.example .env
```

Edit `musicroom-backend/.env` before starting the stack:

```dotenv
# Required for signing JWT access tokens (generate with: openssl rand -hex 32)
JWT_SECRET=<your-strong-random-jwt-secret>

# Required for Google OAuth sign-in & account linking (Google Cloud Console -> Web OAuth Client)
GOOGLE_CLIENT_ID=<your-google-oauth-web-client-id>
GOOGLE_CLIENT_SECRET=<your-google-oauth-client-secret>
# Use your ngrok domain for mobile testing, or http://localhost:8080/auth/google/callback for local web only
GOOGLE_REDIRECT_URL=https://<your-ngrok-domain>/auth/google/callback

# Required for exposing the local API over HTTPS to physical phones via start-ngrok.sh
NGROK_AUTHTOKEN=<your-ngrok-authtoken>
NGROK_DOMAIN=<your-static-ngrok-domain, e.g. your-domain.ngrok-free.dev>
```

> **Google Cloud Console setup (for Google OAuth):**
> 1. Add `https://<your-ngrok-domain>/auth/google/callback` (and `http://localhost:8080/auth/google/callback` if testing locally on Web) under **Authorized redirect URIs** in your OAuth 2.0 Web Client.
> 2. If your OAuth consent screen is in **Testing** mode, add the Gmail accounts you will test with under **Test users**.
>
> **Local-only / E2E mode:** If you only want to run the backend, MailHog, Web client, or `make e2e` locally without Google Sign-In or a phone, you can leave `GOOGLE_*` and `NGROK_*` empty (Google routes return `503` by design and `make e2e` skips the Google OAuth check automatically).
>
> **Security:** Never commit `.env` or `.env.smtp` — both are git-ignored.

---

### 2. Start Backend & Seed Demo Data

**Terminal 1 (Backend + PostgreSQL + MailHog + Adminer):**
```bash
cd musicroom-backend
make docker-up
```

Once the API is healthy (`curl http://localhost:8080/health` returns `{"status":"ok"}`), load the demo dataset in another terminal (required for the demo accounts such as `alice@musicroom.local`):

```bash
cd musicroom-backend
make seed
```

---

### 3. Run the Client (Mobile or Web)

#### Option A — Physical Phone via Expo Go (Android / iOS)

**Terminal 2 (HTTPS tunnel to backend API):**
```bash
cd musicroom-backend
./scripts/start-ngrok.sh
```

**Terminal 3 (Expo Metro + Cloudflare tunnel):**
```bash
cd musicroom-app
npm install
./scripts/start-cf-tunnel.sh
```

On your phone:
1. Scan the QR code from Terminal 3 with **Expo Go**.
2. `start-cf-tunnel.sh` automatically reads `NGROK_DOMAIN` from `musicroom-backend/.env`. You can also set or change the API address anytime on the Login screen via **`CONFIGURE SERVER URL`** → `https://<your-ngrok-domain>` → **`SAVE`**.
3. Sign in with any of the following:
   - **Demo account** (after `make seed`): `alice@musicroom.local` / `Password123!`
   - **Google OAuth**: tap **`SIGN IN WITH GOOGLE`** (on ngrok's free tier, tap **Visit Site** if the interstitial page appears)
   - **New account**: tap **`REGISTER NEW ACCOUNT`** and retrieve the verification token from MailHog at http://localhost:8025

#### Option B — Responsive Web Client (Local)

```bash
cd musicroom-app
npm install
npm run web        # opens Metro Web on http://localhost:8081 (allowed in CORS_ORIGINS)
```

---

## Services

| Service | URL | Credentials / Notes |
| :--- | :--- | :--- |
| API | http://localhost:8080 | Health check: `/health` |
| Swagger UI | http://localhost:8080/swagger/index.html | Auto-generated OpenAPI 2.0 reference |
| Adminer | http://localhost:8090 | Server `postgres`, user/pass/db `musicroom` |
| MailHog | http://localhost:8025 | Catches all outgoing verification & password-reset emails |
| PostgreSQL | `localhost:5432` | `musicroom` / `musicroom` / `musicroom` |

---

## Testing & Load Benchmarks

```bash
cd musicroom-backend

make test              # Unit tests
make test-integration  # Integration tests against a throwaway DB (stack must be up)
make e2e               # Full end-to-end suite against the running stack
```

Load benchmark scripts (`k6`) and measured capacity results live in [`musicroom-backend/docs/BENCHMARK.md`](musicroom-backend/docs/BENCHMARK.md).

---

## Documentation

- **Architecture, Concurrency & Defense Guide**: [`docs/walkthrough.md`](docs/walkthrough.md)
- **Specification Audit & Progress Board**: [`docs/TODO.md`](docs/TODO.md)
- **Load Benchmark Report**: [`musicroom-backend/docs/BENCHMARK.md`](musicroom-backend/docs/BENCHMARK.md)
- **Collaboration SOP**: [`docs/SOP.md`](docs/SOP.md)
