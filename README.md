# music-room

Collaborative music app: shared playlists, voting events, friends.

| Directory | Stack |
|---|---|
| `musicroom-backend/` | Go 1.25 + Echo + PostgreSQL 16 (sqlc, golang-migrate) |
| `musicroom-app/` | React Native / Expo **SDK 57** (Android, iOS, web) |

---

## Quick start

Terminal 1:
```bash
# 1. backend
cd musicroom-backend
cp .env.example .env
# add to the env vars needed !
make docker-up
```

Terminal 2:
```bash
cd musicroom-backend
./scripts/start-ngrok.sh
```

Terminal 3:
```bash
cd musicroom-app
npm install
./scripts/start-cf-tunnel.sh
```

On the phone:
1. Scan the QR code from terminal 3 with Expo Go.
2. Login screen → `CONFIGURE SERVER URL` →
   `https://golf-iodize-bartender.ngrok-free.dev` → `SAVE`. The app remembers
   it, and since the address is fixed, once per phone is enough.
3. 
    a. log in as alice@musicroom.local / Password123!
    b. `SIGN IN WITH GOOGLE`. On ngrok's free plan a warning page may come first: tap **Visit Site**.
    c. register new account (mail will go to mailhog : http://localhost:8025)

---

### Services

| Service | URL | Credentials |
|---|---|---|
| API | http://localhost:8080 | — |
| Swagger UI | http://localhost:8080/swagger/index.html | — |
| Adminer | http://localhost:8090 | server `postgres`, user/pass/db `musicroom` |
| MailHog | http://localhost:8025 | catches every outgoing email |
| PostgreSQL | `localhost:5432` | `musicroom` / `musicroom` / `musicroom` |

### End-to-end tests

```bash
make e2e          # the stack must be up
```

