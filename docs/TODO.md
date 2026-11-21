<p align="right">
  <a href="#musicroom-zh-todo">
    <img src="https://img.shields.io/badge/%E7%B9%81%E9%AB%94%E4%B8%AD%E6%96%87-sienna?style=for-the-badge" />
  </a>
</p>

# MusicRoom — Progress & Specification Audit Board (TODO) <a id="musicroom-en-todo"></a>

> Maintained per [`docs/SOP.md`](./SOP.md) using the official project specification [`docs/en.subject.pdf`](./en.subject.pdf) (Version 6) as the Single Source of Truth.
> For architecture rationale, mathematical derivations, security threat models, and oral defense drills, see [`docs/walkthrough.md`](./walkthrough.md).

## Table of Contents
- [1. Project Overview & Progress Dashboard](#1-project-overview--progress-dashboard)
- [2. Phase 1 Boundary & Compliance Audit (Forbidden / Constraints / Governance)](#2-phase-1-boundary--compliance-audit-forbidden--constraints--governance)
- [3. Mandatory Specification Checklist (Ch. V)](#3-mandatory-specification-checklist-ch-v)
- [4. Bonus Specification Checklist (Ch. VI)](#4-bonus-specification-checklist-ch-vi)
- [5. Actionable Backlog & Code Hygiene Findings](#5-actionable-backlog--code-hygiene-findings)
- [6. Audit & Review History](#6-audit--review-history)

---

## 1. Project Overview & Progress Dashboard

| Subject Chapter (`en.subject.pdf` v6) | Scope | Status | Completion | Verification Evidence |
| :--- | :--- | :---: | :---: | :--- |
| **Ch. II** AI Instructions | Human technical leadership & independent defense readiness | 🔄 In Progress | 85% | [`walkthrough.md`](./walkthrough.md) + Phase 6 Oral Defense Drill |
| **Ch. IV** General Instructions | Tech justification, zero committed third-party libs, Makefile dependency download | ✅ Done | 100% | [`e2e/repo_test.go`](../musicroom-backend/e2e/repo_test.go) |
| **Ch. V.1** User | Signup, email verification, forgot/reset password, Google OAuth + linking, 4-tier profile, friends | ✅ Done | 100% | [`e2e/user_test.go`](../musicroom-backend/e2e/user_test.go) |
| **Ch. V.2.1** Music Track Vote | Live queue voting, public/private visibility, 3 licenses (incl. time & geo-fencing), concurrency | ✅ Done | 100% | [`e2e/vote_test.go`](../musicroom-backend/e2e/vote_test.go) |
| **Ch. V.2.2** Music Control Delegation | Intentionally omitted under the "at least 2 of 3 services" rule | ➖ Omitted (Valid) | N/A | [`e2e/api_test.go`](../musicroom-backend/e2e/api_test.go#L14-L23) |
| **Ch. V.2.3** Music Playlist Editor | Real-time multi-user editing, public/private visibility, 2 licenses, OCC + fractional indexing | ✅ Done | 100% | [`e2e/playlist_test.go`](../musicroom-backend/e2e/playlist_test.go) |
| **Ch. V.3–V.5** Server / API / Mobile | Backend source of truth, REST/JSON + Swagger, configurable backend URL on mobile | ✅ Done | 100% | [`e2e/api_test.go`](../musicroom-backend/e2e/api_test.go) |
| **Ch. V.6** Securing | Data isolation, dual-layer rate limiting, lockout, token rotation/revocation, audit logs, `.env` ignore | ✅ Done | 100% | [`e2e/security_test.go`](../musicroom-backend/e2e/security_test.go) |
| **Ch. V.7** Ramp-up | `k6` load scripts, hardware specs, and measured concurrency report | ✅ Done | 100% | [`BENCHMARK.md`](../musicroom-backend/docs/BENCHMARK.md) + [`e2e/rampup_test.go`](../musicroom-backend/e2e/rampup_test.go) |
| **Ch. V.8** Agility, Quality & CI | Unit, integration, and E2E suites; migrations and seed datasets | ✅ Done | 100% | 67 Go test functions across Unit + Integration + E2E |
| **Ch. VI.1** Bonus: Multi-platform | React Native + Expo Responsive Web, unified CORS & WebSocket `CheckOrigin` | ✅ Done | 100% | [`e2e/bonus_test.go`](../musicroom-backend/e2e/bonus_test.go#L36-L56) |
| **Ch. VI.3** Bonus: Free vs. Paid | Free/Premium tiers, subscription history, `402` collaboration gate, zero-cron expiry | ✅ Done | 100% | [`e2e/bonus_test.go`](../musicroom-backend/e2e/bonus_test.go#L10-L34) |
| **Ch. VI.2 / VI.4** IoT / Offline Mode | iBeacon & offline sync (optional bonuses) | ➖ Not Implemented | 0% | Optional |
| **Ch. VII** Submission & Hygiene | Clean repo, local `.env` setup, full live test run | ⚠️ Pending Final Run | 85% | See Section 5 Backlog |

---

## 2. Phase 1 Boundary & Compliance Audit (Forbidden / Constraints / Governance)

> [!CAUTION]
> **Critical 42 Evaluation Red Lines (any violation risks immediate 0 / Bonus disqualification):**
> 1. **Publicly Stored Credentials = Instant Fail** (Ch. V.8): All API keys, secrets, and `.env` files MUST remain local and git-ignored.
> 2. **Bonus PERFECT Clause** (Ch. VI, p. 14): Bonuses are evaluated **ONLY** if the Mandatory part is 100% complete and functions without malfunction.
> 3. **SDK Restriction** (Ch. III): External SDKs must not do the core work (Deezer is used strictly for track metadata/preview lookup; Google OAuth solely for identity).
> 4. **AI Responsibility** (Ch. II): The developer remains 100% accountable for every architectural choice and must explain/defend it independently during peer evaluation.

- [x] **Zero Committed Third-Party Code**: Verified by [`TestRepo_DependenciesAreDownloadedNotCommitted`](../musicroom-backend/e2e/repo_test.go#L35-L50) — no `node_modules/` or `vendor/` tracked; dependencies install via `go mod download` and `npm install`.
- [x] **Secrets Git-Ignored**: Verified by [`TestRepo_EnvFilesAreIgnoredAndNeverCommitted`](../musicroom-backend/e2e/repo_test.go#L21-L33) — `.env`, `.env.smtp`, and `musicroom-app/.env` are ignored and untracked.
- [x] **Backend as Single Source of Truth**: Verified by [`TestServer_BackendIsTheSourceOfTruth`](../musicroom-backend/e2e/api_test.go#L25-L47).

---

## 3. Mandatory Specification Checklist (Ch. V)

### V.1 User & Social
- [x] **Email/Password Registration & Login**: [`internal/service/auth.go`](../musicroom-backend/internal/service/auth.go#L75-L150), [`RegisterScreen.tsx`](../musicroom-app/src/screens/auth/RegisterScreen.tsx), [`LoginScreen.tsx`](../musicroom-app/src/screens/auth/LoginScreen.tsx)
- [x] **Mandatory Email Validation**: Unverified users receive `403`; single-use token (24h TTL) delivered via MailHog/SMTP. Verified by [`TestUser_MailPasswordSignupDemandsMailValidation`](../musicroom-backend/e2e/user_test.go#L12-L34).
- [x] **Forgot & Reset Password**: Single-use reset token (1h TTL); resetting atomically bumps `token_epoch` and revokes all refresh tokens. Verified by [`TestUser_ForgottenPasswordIsResetThroughTheMail`](../musicroom-backend/e2e/user_test.go#L36-L54).
- [x] **Social Registration/Login (Google OAuth 2.0 + PKCE S256)**: System browser OAuth flow, deep-link callback, single-use handoff code exchange. Implemented in [`googleflow.go`](../musicroom-backend/internal/api/handlers/googleflow.go) & [`AuthContext.tsx`](../musicroom-app/src/context/AuthContext.tsx#L71-L88).
- [x] **Link & Unlink Social Account**: Authenticated `/users/me/oauth/google/link-ticket` flow; unlink blocks removing the sole login method (`ErrLastLoginMethod`). Verified by [`TestUser_GoogleSignInAndLinking`](../musicroom-backend/e2e/user_test.go#L56-L96).
- [x] **4-Tier Profile & Music Preferences**: Per-field `public` / `friends` / `private` filtering plus `music_preferences` in [`internal/service/user.go`](../musicroom-backend/internal/service/user.go#L34-L121). Verified by [`TestUser_ProfileVisibilityLevelsAndMusicPreferences`](../musicroom-backend/e2e/user_test.go#L98-L151).
- [x] **Friends System**: Trigram user search (`pg_trgm`), requests, accept/reject, auto-accept on reciprocal requests, canonical undirected edges (`CHECK (user_id_a < user_id_b)`). Verified by [`TestUser_FriendRequests`](../musicroom-backend/e2e/user_test.go#L153-L183).

### V.2 Services (2 of 3 Selected)
- [x] **V.2.1 Music Track Vote**
  - [x] Track suggestion (Deezer search + local `tracks` cache), vote/unvote, live deterministic queue ordering (`vote_count DESC, created_at ASC`), WebSocket `queue:updated` push.
  - [x] **Visibility**: `public` by default (discoverable & votable by all); `private` (owner & invited users only).
  - [x] **Licenses**: `open` (default), `invited_only`, and `time_and_location` (server time window + Haversine geo-fence).
  - [x] **Concurrency Control**: Composite primary key `PRIMARY KEY (event_track_id, user_id)` + `ON CONFLICT DO NOTHING`. Verified by [`TestVote_ConcurrentVotesAreAllCountedExactlyOnce`](../musicroom-backend/e2e/vote_test.go#L176-L214).
- [x] **V.2.3 Music Playlist Editor**
  - [x] Real-time add, reorder, and delete tracks with WebSocket `playlist:updated` broadcast.
  - [x] **Visibility**: `public` by default; `private` restricted to owner & collaborators (including WebSocket room guard & `CloseRoom` on visibility downgrade).
  - [x] **Licenses**: `open` (default) and `invited_only` (`can_edit` collaborator role check).
  - [x] **Concurrency Control**: `SELECT ... FOR UPDATE` row lock + optimistic `version` check (`409 Conflict`) + Fractional Indexing with `1e-6` automatic renormalization + client jittered exponential backoff. Verified by [`TestPlaylist_ConcurrentMovesNeverCorruptTheOrder`](../musicroom-backend/e2e/playlist_test.go#L110-L173).

### V.3–V.5 Server / API / Mobile Application
- [x] **REST + JSON + Swagger Documentation**: Served at `/swagger/index.html` and `/swagger/doc.json`. Automated coverage check in [`TestAPI_EveryRouteIsDocumented`](../musicroom-backend/e2e/api_test.go#L49-L94).
- [x] **Configurable Backend Address on Mobile**: `CONFIGURE SERVER URL` in [`LoginScreen.tsx`](../musicroom-app/src/screens/auth/LoginScreen.tsx#L89-L97), persisted in `AsyncStorage` and dynamically applied by Axios.
- [x] **Audio Preview Playback**: [`usePreviewPlayer.ts`](../musicroom-app/src/audio/usePreviewPlayer.ts) fetches fresh Deezer preview URLs via `GET /tracks/:id/preview` and auto-advances to the next track.

### V.6 Securing
- [x] **Strict Data Isolation**: JWT-derived `userID` enforces ownership and visibility boundaries across all endpoints. Verified by [`TestSecurity_NoAccessToAnotherUsersData`](../musicroom-backend/e2e/security_test.go#L15-L48).
- [x] **Brute-Force & Rate-Limit Defenses**: Global `20 req/s/IP` + `/auth/*` `10 req/min/IP` (burst 3) + account lockout after 5 failed logins in 15 min. Verified by [`TestSecurity_PasswordBruteforceLocksTheAccount`](../musicroom-backend/e2e/security_test.go#L88-L100) & [`TestSecurity_RateLimitingPerAddress`](../musicroom-backend/e2e/security_test.go#L102-L135).
- [x] **Session Theft & Replay Defenses**: Access token revocation (`revoked_access_tokens`), `token_epoch` global invalidation on password reset, Refresh Token rotation with family revocation on reuse. Verified by [`TestSecurity_TokensCannotBeForgedReplayedOrKeptAfterLogout`](../musicroom-backend/e2e/security_test.go#L50-L86).
- [x] **Full Mobile Audit Logging**: Axios injects `X-Platform`, `X-Device-Model`, and `X-App-Version`; backend records asynchronously to `audit_logs` for both HTTP and WebSocket ticket holders. Verified by [`TestSecurity_EveryActionIsLoggedWithPlatformDeviceAndVersion`](../musicroom-backend/e2e/security_test.go#L137-L193).

### V.7–V.8 Ramp-up & Quality
- [x] **Load Evaluation Report & Scripts**: [`musicroom-backend/docs/BENCHMARK.md`](../musicroom-backend/docs/BENCHMARK.md) + `benchmarks/{auth,vote-ws,playlist-ws}.js` (`k6`), specifying CPU/RAM and measured throughput across all three scenarios.
- [x] **Multi-Dimensional Test Suite**: Unit tests, DB integration tests, and subject-mapped E2E tests.

---

## 4. Bonus Specification Checklist (Ch. VI)

- [x] **VI.1 Multi-platform Support (Responsive Web)**: Expo Web (`npm run web` on `:8081`), `.web.ts` platform-specific storage/baseURL modules, [`useResponsive.ts`](../musicroom-app/src/theme/useResponsive.ts) (`900px` breakpoint), shared CORS & WebSocket origin whitelist.
- [x] **VI.3 Free vs. Paid Subscription**: Default `free` tier, instant switch to/from `premium` (30-day term, zero-cron read-time expiration), full `subscription_history`, `402 Payment Required` gate on collaborative playlist editing.
- [ ] **VI.2 Reflection on the IoT (iBeacon)**: Not implemented (optional).
- [ ] **VI.4 Offline Mode**: Not implemented (optional).

---

## 5. Actionable Backlog & Code Hygiene Findings

### A. Startup & End-to-End Live Verification (Highest Priority)
- [ ] **Create Local `.env`**: Copy `musicroom-backend/.env.example` to `musicroom-backend/.env` and populate `JWT_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`, `NGROK_AUTHTOKEN`, and `NGROK_DOMAIN`.
- [ ] **Configure Google Cloud Console**: Register `https://<your-ngrok-domain>/auth/google/callback` (and `http://localhost:8080/auth/google/callback`) in Authorized redirect URIs and add test Gmail accounts.
- [ ] **Start Docker & Execute Full Verification Matrix**:
  - Run `make docker-up` and `make seed` in `musicroom-backend/`.
  - Execute `make test`, `make test-integration`, and `make e2e` — confirm 100% `[PASS]`.
  - Test live on physical devices via `./scripts/start-ngrok.sh` + `./scripts/start-cf-tunnel.sh` (registration/MailHog, Google sign-in & link, geo/time-fenced voting, two-device concurrent playlist drag-and-drop).

### B. Code Review & Repository Hygiene Items
- [x] **[Config Fix]** [`musicroom-backend/docker-compose.yml`](../musicroom-backend/docker-compose.yml#L38): Changed hardcoded `JWT_SECRET: dev-secret-change-me` to `${JWT_SECRET:-dev-secret-change-me}` so `.env` overrides take effect inside Docker.
- [x] **[Docs Sync]** Root [`README.md`](../README.md): Replaced stale hardcoded ngrok URL, documented all required `.env` variables (including `JWT_SECRET` and `GOOGLE_REDIRECT_URL`), Google Cloud Console setup, `make seed`, Web launch, and links to `docs/`.
- [ ] **[Docs Minor Fix]** [`musicroom-backend/README.md`](../musicroom-backend/README.md#L9) lists Adminer on `:8081` (actual port in `docker-compose.yml` is `:8090`; `:8081` is reserved for Expo Metro).
- [ ] **[Repo Hygiene / Ch. VII]** Remove duplicate script [`musicroom-backend/t.sh`](../musicroom-backend/t.sh) (byte-identical to [`clean-go.sh`](../musicroom-backend/clean-go.sh)) and `docs/walkthrough_ref.md` before final Git submission.
- [ ] **[Frontend Minor Cleanup]** [`musicroom-app/src/ws/useWebSocket.ts`](../musicroom-app/src/ws/useWebSocket.ts#L33-L35): `.replace(/^http/, 'ws').replace(/^https/, 'wss')` has a dead second `.replace` (the first already converts `https` → `wss`); simplify to `.replace(/^http/, 'ws')` with a brief comment.

### C. Oral Defense Preparation (Phase 6)
- [ ] **Complete Red-Team Drill**: Practice all 10 high-frequency examiner questions in [`docs/walkthrough.md`](./walkthrough.md#8-red-team-oral-defense-drill-phase-6-high-frequency-examiner-questions).

---

## 6. Audit & Review History

| Date | Phase | Summary & Decisions |
| :--- | :--- | :--- |
| 2026-11-11 | Phase 1 Audit | Compared `docs/en.subject.pdf` (v6) vs. legacy `docs/Subject.md`; identified missing Ch. II (AI Instructions), Ch. VI Bonus PERFECT banner, and Ch. VII; audited full backend/frontend/E2E codebase. |
| 2026-11-21 | Phase 3 Artifacts | Removed redundant `docs/Subject.md`; created bilingual (English-first) `docs/TODO.md` and `docs/walkthrough.md`; fixed `JWT_SECRET` env expansion in `docker-compose.yml`; updated root `README.md` with complete startup prerequisites. |

---

<br/>

---

<br/>

<p align="right">
  <a href="#musicroom-en-todo">
    <img src="https://img.shields.io/badge/-TOP-sienna?style=for-the-badge" />
  </a>
</p>

# MusicRoom — 任務進度與規格稽核面板 (TODO & Audit Board) <a id="musicroom-zh-todo"></a>

> 本文件依據 [`docs/SOP.md`](./SOP.md) 維護，並以官方規格書 [`docs/en.subject.pdf`](./en.subject.pdf)（Version 6）作為唯一真實來源（Single Source of Truth）。
> 架構原理解析與答辯推導請參閱技術聖經：[`docs/walkthrough.md`](./walkthrough.md#musicroom-zh-walkthrough)。

## 目錄
- [1. 專案總覽與進度儀表板](#1-專案總覽與進度儀表板)
- [2. Phase 1 專案邊界與紅線稽核 (Forbidden / Constraints / Governance)](#2-phase-1-專案邊界與紅線稽核-forbidden--constraints--governance)
- [3. Mandatory 核心需求逐項核對表 (Ch. V)](#3-mandatory-核心需求逐項核對表-ch-v)
- [4. Bonus 加分項核對表 (Ch. VI)](#4-bonus-加分項核對表-ch-vi)
- [5. 待辦清單、審查發現之問題與改進建議](#5-待辦清單審查發現之問題與改進建議)
- [6. 審查與修復日誌](#6-審查與修復日誌)

---

## 1. 專案總覽與進度儀表板

| 規格章節 (`en.subject.pdf` v6) | 實作範疇 | 狀態 | 完成度 | 驗證依據 |
| :--- | :--- | :---: | :---: | :--- |
| **Ch. II** AI Instructions | 人工決策主導、完整架構掌握與可答辯性 | 🔄 進行中 | 85% | [`walkthrough.md`](./walkthrough.md#musicroom-zh-walkthrough) + Phase 6 演練 |
| **Ch. IV** General Instructions | 技術選型論證、零第三方原始碼提交、Makefile 自動化依賴 | ✅ 完成 | 100% | [`e2e/repo_test.go`](../musicroom-backend/e2e/repo_test.go) |
| **Ch. V.1** User | 註冊、信箱驗證、忘記密碼、Google OAuth + 綁定、4 層隱私 Profile、好友 | ✅ 完成 | 100% | [`e2e/user_test.go`](../musicroom-backend/e2e/user_test.go) |
| **Ch. V.2.1** Music Track Vote | 即時投票、公開/私密、3 種授權（含時間與地理圍欄）、並發防重 | ✅ 完成 | 100% | [`e2e/vote_test.go`](../musicroom-backend/e2e/vote_test.go) |
| **Ch. V.2.2** Music Control Delegation | 依規格「三選二」策略未實作（合法省略） | ➖ 選配省略 | N/A | [`e2e/api_test.go`](../musicroom-backend/e2e/api_test.go#L14-L23) |
| **Ch. V.2.3** Music Playlist Editor | 即時多用戶編輯、公開/私密、2 種授權、樂觀鎖 + 分數索引並發控制 | ✅ 完成 | 100% | [`e2e/playlist_test.go`](../musicroom-backend/e2e/playlist_test.go) |
| **Ch. V.3–V.5** Server / API / Mobile | 後端單一真相來源、REST/JSON + Swagger、App 動態設定 Server URL | ✅ 完成 | 100% | [`e2e/api_test.go`](../musicroom-backend/e2e/api_test.go) |
| **Ch. V.6** Securing | 資料隔離、雙層限流、帳號鎖定、Token 撤銷輪替、行動端審計日誌、`.env` 忽略 | ✅ 完成 | 100% | [`e2e/security_test.go`](../musicroom-backend/e2e/security_test.go) |
| **Ch. V.7** Ramp-up | k6 壓測腳本、伺服器硬體規格與吞吐量報告 | ✅ 完成 | 100% | [`BENCHMARK.md`](../musicroom-backend/docs/BENCHMARK.md) + [`e2e/rampup_test.go`](../musicroom-backend/e2e/rampup_test.go) |
| **Ch. V.8** Agility, Quality & CI | 單元測試、整合測試、E2E 測試、Migration 與 Seed 機制 | ✅ 完成 | 100% | 67 個 Go 測試函式（Unit + Integration + E2E） |
| **Ch. VI.1** Bonus: Multi-platform | React Native + Expo Web 響應式支援、CORS/WS 白名單 | ✅ 完成 | 100% | [`e2e/bonus_test.go`](../musicroom-backend/e2e/bonus_test.go#L36-L56) |
| **Ch. VI.3** Bonus: Free vs Paid | Free/Premium 切換、訂閱歷史、`402` 協同編輯權限門控、自動過期 | ✅ 完成 | 100% | [`e2e/bonus_test.go`](../musicroom-backend/e2e/bonus_test.go#L10-L34) |
| **Ch. VI.2 / VI.4** IoT / Offline Mode | iBeacon 與離線同步（非強制 Bonus） | ➖ 未實作 | 0% | 選配 |
| **Ch. VII** Submission & Hygiene | Git 倉儲整潔、無殘留檔案、全套本地實跑驗證 | ⚠️ 待收尾 | 85% | 見下方第 5 節待辦清單 |

---

## 2. Phase 1 專案邊界與紅線稽核 (Forbidden / Constraints / Governance)

> [!CAUTION]
> **42 官方核心紅線（違反任一條直接判定專案失敗或 Bonus 不計分）：**
> 1. **憑證外洩即 Fail**（Ch. V.8）：所有金鑰、API Key、`.env` 必須本地保存並由 `.gitignore` 忽略，嚴禁提交至 Git。
> 2. **Bonus PERFECT 條款**（Ch. VI, p. 14）：Bonus 只有在 Mandatory **100% 完整且無任何故障**時才會被評估；Mandatory 若有任一缺漏或異常，Bonus 一律不計分。
> 3. **SDK 禁令**（Ch. III）：選用的第三方 SDK 不得取代核心業務實作（僅使用 Deezer API 提供曲目中繼資料與試聽流、Google OAuth 提供身分驗證）。
> 4. **AI 協作與答辯責任**（Ch. II）：開發者對所有架構與技術決策負 100% 責任，必須能於 Peer Evaluation 脫離 AI 獨立深入解釋每項實作。

- [x] **無第三方依賴原始碼提交**：已透過 [`TestRepo_DependenciesAreDownloadedNotCommitted`](../musicroom-backend/e2e/repo_test.go#L35-L50) 驗證未追蹤 `node_modules/` 與 `vendor/`，且依賴透過 `go mod download` / `npm install` 自動安裝。
- [x] **機密檔案 Git 忽略**：已透過 [`TestRepo_EnvFilesAreIgnoredAndNeverCommitted`](../musicroom-backend/e2e/repo_test.go#L21-L33) 驗證 `.env`、`.env.smtp`、`musicroom-app/.env` 均被 `.gitignore` 忽略且未被追蹤。
- [x] **後端為唯一真相來源**：所有狀態持久化於 PostgreSQL 16，已由 [`TestServer_BackendIsTheSourceOfTruth`](../musicroom-backend/e2e/api_test.go#L25-L47) 驗證。

---

## 3. Mandatory 核心需求逐項核對表 (Ch. V)

### V.1 User（用戶與社交）
- [x] **信箱/密碼註冊與登入**：[`internal/service/auth.go`](../musicroom-backend/internal/service/auth.go#L75-L150)、[`RegisterScreen.tsx`](../musicroom-app/src/screens/auth/RegisterScreen.tsx)、[`LoginScreen.tsx`](../musicroom-app/src/screens/auth/LoginScreen.tsx)
- [x] **強制信箱驗證（Mail Validation）**：未驗證帳號登入回傳 `403`，驗證 Token 單次消費（24h TTL），經 MailHog 或 SMTP 寄送。驗證：[`TestUser_MailPasswordSignupDemandsMailValidation`](../musicroom-backend/e2e/user_test.go#L12-L34)
- [x] **忘記密碼與重設密碼**：Token 單次消費（1h TTL），重設後遞增 `token_epoch` 並撤銷所有舊 Refresh Token。驗證：[`TestUser_ForgottenPasswordIsResetThroughTheMail`](../musicroom-backend/e2e/user_test.go#L36-L54)
- [x] **社群登入（Google OAuth 2.0 + PKCE S256）**：支援行動端系統瀏覽器授權、Deep Link 回跳、Handoff Code 兌換 Token Pair。實作：[`googleflow.go`](../musicroom-backend/internal/api/handlers/googleflow.go)、[`AuthContext.tsx`](../musicroom-app/src/context/AuthContext.tsx#L71-L88)
- [x] **社群帳號綁定與解綁**：登入後可透過 `/users/me/oauth/google/link-ticket` 綁定 Google，解綁時檢查是否為唯一登入方式防鎖死（`ErrLastLoginMethod`）。驗證：[`TestUser_GoogleSignInAndLinking`](../musicroom-backend/e2e/user_test.go#L56-L96)
- [x] **個人資料 4 層欄位管理**：支援 public / friends / private 能見度逐欄配置與 `music_preferences`。實作：[`internal/service/user.go`](../musicroom-backend/internal/service/user.go#L34-L121)。驗證：[`TestUser_ProfileVisibilityLevelsAndMusicPreferences`](../musicroom-backend/e2e/user_test.go#L98-L151)
- [x] **好友系統**：支援用戶搜尋（`pg_trgm`）、好友邀請、接受/拒絕、雙向互發自動成為好友，DB 強制 `CHECK (user_id_a < user_id_b)`。驗證：[`TestUser_FriendRequests`](../musicroom-backend/e2e/user_test.go#L153-L183)

### V.2 Services（核心服務 — 三選二）
- [x] **V.2.1 Music Track Vote（即時投票活動）**
  - [x] 推薦歌曲（整合 Deezer API 與本地 `tracks` 快取）、投票與取消投票、即時依票數與建立時間重排佇列並透過 WebSocket 推播 `queue:updated`。
  - [x] **能見度控制**：預設 `public`（所有人可見可投）；`private` 僅主辦人與受邀者可見可投。
  - [x] **授權管理（License）**：預設 `open`（所有人可投）；`invited_only`（僅受邀者）；`time_and_location`（時間區間檢查 + Haversine 球面距離地理圍欄）。
  - [x] **並發競態處理**：DB 複合主鍵 `PRIMARY KEY (event_track_id, user_id)` + `ON CONFLICT DO NOTHING`。驗證：[`TestVote_ConcurrentVotesAreAllCountedExactlyOnce`](../musicroom-backend/e2e/vote_test.go#L176-L214)
- [x] **V.2.3 Music Playlist Editor（即時多用戶協同播放清單）**
  - [x] 新增、移動排序、刪除曲目，並透過 WebSocket 即時推播 `playlist:updated`。
  - [x] **能見度控制**：預設 `public`；`private` 僅擁有者與協作者可存取（含 WS 房間攔截）。
  - [x] **授權管理（License）**：預設 `open`（所有人可編輯）；`invited_only`（僅擁有者與 `can_edit=true` 協作者可編輯）。
  - [x] **並發衝突控制**：PostgreSQL 行鎖 `SELECT ... FOR UPDATE` + 樂觀鎖 `version` 檢查（衝突回傳 `409`）+ 分數索引（Fractional Indexing）與 `1e-6` 自動重均勻化 + 客戶端指數退避重試。驗證：[`TestPlaylist_ConcurrentMovesNeverCorruptTheOrder`](../musicroom-backend/e2e/playlist_test.go#L110-L173)

### V.3–V.5 Server / API / Mobile Application
- [x] **RESTful API + JSON + Swagger 自生成文件**：位於 `/swagger/index.html` 與 `/swagger/doc.json`，由 [`TestAPI_EveryRouteIsDocumented`](../musicroom-backend/e2e/api_test.go#L49-L94) 自動核驗每個路由均已文件化。
- [x] **行動端可動態配置後端位址**：[`LoginScreen.tsx`](../musicroom-app/src/screens/auth/LoginScreen.tsx#L89-L97) 提供 `CONFIGURE SERVER URL`，寫入 `AsyncStorage` 並即時重建 Axios 客戶端。
- [x] **行動端音樂試聽播放**：[`usePreviewPlayer.ts`](../musicroom-app/src/audio/usePreviewPlayer.ts) 透過後端 `/tracks/:id/preview` 即時刷新過期的 Deezer 試聽連結並自動連播下一首。

### V.6 Securing（安全防護與審計）
- [x] **嚴格資料隔離**：所有受保護端點由 JWT 中間件萃取 `userID`，不可越權讀寫他人私密資料、好友清單、私密活動或私密播放清單。驗證：[`TestSecurity_NoAccessToAnotherUsersData`](../musicroom-backend/e2e/security_test.go#L15-L48)
- [x] **防暴力破解與限流**：全域 `20 req/s` + `/auth/*` `10 req/min` (Burst 3) + 連續登入失敗 5 次鎖定 15 分鐘。驗證：[`TestSecurity_PasswordBruteforceLocksTheAccount`](../musicroom-backend/e2e/security_test.go#L88-L100)、[`TestSecurity_RateLimitingPerAddress`](../musicroom-backend/e2e/security_test.go#L102-L135)
- [x] **防 Session 盜用與重放**：Access Token 登出立即進入 `revoked_access_tokens`、密碼重設遞增 `token_epoch`、Refresh Token 單次輪替且重用即撤銷整個 `family_id`。驗證：[`TestSecurity_TokensCannotBeForgedReplayedOrKeptAfterLogout`](../musicroom-backend/e2e/security_test.go#L50-L86)
- [x] **行動端操作全鏈路審計日誌**：Axios 自動附帶 `X-Platform`、`X-Device-Model`、`X-App-Version`，後端非同步寫入 `audit_logs`（含 HTTP 與 WS 票據持有者識別）。驗證：[`TestSecurity_EveryActionIsLoggedWithPlatformDeviceAndVersion`](../musicroom-backend/e2e/security_test.go#L137-L193)

### V.7–V.8 Ramp-up & Quality
- [x] **負載壓測報告與腳本**：[`musicroom-backend/docs/BENCHMARK.md`](../musicroom-backend/docs/BENCHMARK.md) + `benchmarks/{auth,vote-ws,playlist-ws}.js`（k6），載明硬體環境（Ryzen 7 5800H / 13 GiB RAM）與三種場景實測上限。
- [x] **多維度自動化測試**：涵蓋 Unit tests、Integration tests 與 E2E tests。

---

## 4. Bonus 加分項核對表 (Ch. VI)

- [x] **VI.1 Multi-platform support (Responsive Web)**：支援 `npm run web`（Metro `:8081`），具備 `.web.ts` 專屬儲存與 URL 推導模組、[`useResponsive.ts`](../musicroom-app/src/theme/useResponsive.ts) 900px 斷點與桌面寬版佈局、CORS + WS `CheckOrigin` 白名單。
- [x] **VI.3 Free subscription vs. Paid subscription**：新帳號預設 `free`，可於 Profile 切換 `free` ↔ `premium`（30 天效期，讀取時自動降級過期帳號），異動全數記錄於 `subscription_history`；非擁有者協同編輯播放清單或邀請可編輯協作者需 `premium`（否則回傳 `402 Payment Required`）。
- [ ] **VI.2 Reflection on the IoT (iBeacon)**：未實作（選配）。
- [ ] **VI.4 Offline Mode**：未實作（選配）。

---

## 5. 待辦清單、審查發現之問題與改進建議

### A. 啟動與實機驗證（最高優先級）
- [ ] **建立本機 `.env` 設定檔**：於 `musicroom-backend/` 執行 `cp .env.example .env`，填入 `JWT_SECRET`、`GOOGLE_CLIENT_ID`、`GOOGLE_CLIENT_SECRET`、`NGROK_AUTHTOKEN`、`NGROK_DOMAIN`，並將 `GOOGLE_REDIRECT_URL` 改為 `https://<你的NGROK_DOMAIN>/auth/google/callback`。
- [ ] **設定 Google Cloud Console**：於 OAuth 2.0 Client 的 Authorized redirect URIs 加入上述 ngrok callback URL（及本機測試用 `http://localhost:8080/auth/google/callback`）。
- [ ] **啟動環境並執行全量測試矩陣**：
  - 啟動 Docker Daemon 後於 `musicroom-backend/` 執行 `make docker-up` 與 `make seed`。
  - 執行 `make test`、`make test-integration`、`make e2e`，確認 100% `[PASS]`。
  - 實機手機掃描 Expo Go QR Code（`./scripts/start-cf-tunnel.sh`）並搭配 `./scripts/start-ngrok.sh`，端對端實測註冊驗證、Google 登入/綁定、地理時間投票與雙機即時協同拖曳播放清單。

### B. 代碼審查與工程衛生項目
- [x] **[設定修復]** [`musicroom-backend/docker-compose.yml`](../musicroom-backend/docker-compose.yml#L38)：已將寫死的 `JWT_SECRET: dev-secret-change-me` 改為 `${JWT_SECRET:-dev-secret-change-me}`，使 `.env` 的 `JWT_SECRET` 能在 Docker 模式生效。
- [x] **[文件同步]** 根目錄 [`README.md`](../README.md)：已移除舊的硬編碼 ngrok 網域，補齊 `.env` 必設清單（含 `JWT_SECRET`、`GOOGLE_REDIRECT_URL`）、Google Cloud Console 設定、`make seed`、Web 啟動與 `docs/` 索引。
- [ ] **[文件小修正]** [`musicroom-backend/README.md`](../musicroom-backend/README.md#L9) 誤植 Adminer Port 為 `:8081`（實際為 `:8090`），建議統一修正。
- [ ] **[倉儲整潔 / Ch. VII]** 繳交前建議移除重複腳本 [`musicroom-backend/t.sh`](../musicroom-backend/t.sh)（與 [`clean-go.sh`](../musicroom-backend/clean-go.sh) 完全重複）以及參考檔 `docs/walkthrough_ref.md`。
- [ ] **[前端小冗餘]** [`musicroom-app/src/ws/useWebSocket.ts`](../musicroom-app/src/ws/useWebSocket.ts#L33-L35)：`.replace(/^http/, 'ws').replace(/^https/, 'wss')` 第二個 `.replace` 永遠不會命中，建議簡化為 `.replace(/^http/, 'ws')` 並加註解。

### C. 答辯賦能（Phase 6）
- [ ] **完成紅隊模擬答辯**：針對 [`docs/walkthrough.md`](./walkthrough.md#8-紅隊口試答辯高頻考題-phase-6-defense-drill) 第 8 節 10 題高頻考題逐題演練。

---

## 6. 審查與修復日誌

| 日期 | 階段 | 摘要與決議 |
| :--- | :--- | :--- |
| 2026-11-11 | Phase 1 Audit | 完成 `en.subject.pdf` 與舊 `Subject.md` 逐頁比對，識別缺失章節與警語；完成前後端與 E2E 測試全貌盤點。 |
| 2026-11-21 | Phase 3 Artifacts | 刪除冗餘 `docs/Subject.md`；建立英中雙語（英文優先）版 `docs/TODO.md` 與 `docs/walkthrough.md`；修復 `docker-compose.yml` 的 `JWT_SECRET` 環境變數注入；補完根目錄 `README.md` 啟動教學。 |
