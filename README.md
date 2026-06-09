# Nexus Forum

Reddit-style forum for fandom communities: posts, nested comments, communities, real-time chat, moderation, notifications, and analytics.

**Stack:** Go (Gin + GORM) backend · React (Vite) frontend · SQLite (default) or PostgreSQL · MinIO (optional) · Prometheus + Grafana + Loki (optional observability)

---

## Project overview

Nexus Forum lets users join communities, publish posts (text, image, video, link, poll), schedule posts, vote, comment in threads, receive in-app notifications (with live WebSocket unread counts), chat in DMs, and moderate content. Admins get analytics (DAU/MAU, retention, engagement) via API.

---

## Architecture overview

```
Browser (React/Vite)
    │  REST + WebSocket
    ▼
Go API (cmd/api)
    ├── middleware/   JWT, rate limits, metrics, logging, Turnstile
    ├── handler/      HTTP controllers
    ├── service/      Business logic
    ├── repository/   GORM data access
    ├── search/       Unicode-aware post/user/community search
    ├── email/        SMTP notifications (optional)
    └── storage/      MinIO or local ./uploads

SQLite / PostgreSQL          MinIO (optional)
Prometheus / Grafana / Loki  (docker-compose)
```

Backend follows layered architecture: **handler → service → repository → model**. Frontend uses a single API client (`src/api/nexusApi.js`) and React Context for auth.

---

## Folder structure

```
nexus_forum/
├── backend/
│   ├── cmd/api/main.go          # Entry point, routes, migrations, seed, schedulers
│   ├── internal/
│   │   ├── handler/             # HTTP handlers
│   │   ├── service/             # Business logic
│   │   ├── repository/          # Database access
│   │   ├── search/              # Unicode search index + queries
│   │   ├── email/               # SMTP mailer + notification emails
│   │   ├── model/               # GORM entities
│   │   ├── middleware/          # Auth, metrics, rate limit
│   │   └── storage/             # MinIO + local upload backends
│   └── scripts/benchperf/       # API latency benchmark
├── src/
│   ├── api/nexusApi.js          # API client (auth, feed, entities)
│   ├── pages/                   # Route pages
│   ├── components/              # UI components
│   └── lib/                     # AuthContext, ChatLayoutContext, helpers
├── monitoring/                  # Prometheus, Grafana, Loki configs
├── docker-compose.yml
└── README.md
```

---

## Local development setup

### Prerequisites

- Go 1.22+
- Node.js 20–22
- npm

### Backend

```bash
cd backend
cp .env.example .env          # optional — edit secrets locally
go run ./cmd/api
```

API: **http://localhost:8080**  
Health: `GET /health`  
Metrics: `GET /metrics`

On first start, SQLite file `backend/nexus_forum.db` is created automatically, tables are migrated, search indexes are built, and demo users are seeded if the DB is empty.

A background worker publishes due **scheduled posts** every 30 seconds.

### Frontend

```bash
npm install
npm run dev
```

App: **http://localhost:5173**

Set `VITE_API_URL=http://localhost:8080/api` in `.env` if the API is not on the default host.

---

## Key features

### Feeds

| Home tab | Endpoint | Content |
|----------|----------|---------|
| Hot / New / Top | `GET /api/posts` | All published posts |
| **Following Users** | `GET /api/posts/following` | Posts from followed users |
| **Following Communities** | `GET /api/posts/following-communities` | Posts from joined communities |

### Notifications

- Created on comment, reply, mention, moderation actions, scheduled publish, etc.
- In-app list: `GET /api/notifications`
- Unread badge: `GET /api/notifications/unread-count`
- Live updates: WebSocket `GET /api/ws/global` (messages `notification`, `unread_count`)
- Optional email via SMTP when user preferences allow (`email_notify_*` fields)

### Search

Unicode-aware search across posts (title, content, tags), users, and communities. Uses FTS5 when the SQLite build supports it; otherwise a Unicode token index (`post_search_tokens` + `search_blob`). Supports Cyrillic and Latin mixed queries.

`GET /api/search?q=...`

### Scheduled posts

Create with `status: "scheduled"` and `publish_at` (ISO timestamp). View under **Profile → Отложенные**. The backend worker auto-publishes due posts and sends an in-app notification.

### Moderation reports

Users submit: `POST /api/reports`  
Admins view enriched queue: `GET /api/moderation/reports` (reporter username, target type/id, summary, reason, status, date)

---

## Docker setup

```bash
docker compose up -d --build
```

| Service   | URL                         |
|-----------|-----------------------------|
| Frontend  | http://localhost:5173       |
| Backend   | http://localhost:8080       |
| MinIO API | http://localhost:9000       |
| MinIO UI  | http://localhost:9001       |
| Prometheus| http://localhost:9090       |
| Grafana   | http://localhost:3000       |
| Loki      | http://localhost:3100       |

Grafana default login: **admin / admin**

---

## MinIO setup

MinIO runs via docker-compose. Backend env (see `backend/.env.example`):

```env
MINIO_ENDPOINT=minio:9000          # use localhost:9000 for local Go without Docker
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=nexus-forum
MINIO_PUBLIC_URL=http://localhost:9000
MINIO_USE_SSL=false
```

If MinIO is unset or unreachable, uploads fall back to `backend/uploads/` served at `/uploads`.

---

## Prometheus setup

- Backend exposes `GET /metrics` (Prometheus text format).
- Scrape config: `monitoring/prometheus/prometheus.yml` → target `backend:8080`.
- Open **http://localhost:9090** → Status → Targets → `nexus-backend` should be UP.

Example query: `http_requests_total`

---

## Grafana setup

- Container in `docker-compose.yml`, port **3000**.
- Datasources auto-provisioned from `monitoring/grafana/provisioning/datasources/datasources.yml` (Prometheus + Loki).
- Login: **admin / admin**
- Explore → Prometheus for metrics, Explore → Loki for container logs (via Promtail).

---

## Loki setup

- Loki config: `monitoring/loki/loki-config.yml`
- Promtail ships Docker container logs to Loki (`monitoring/promtail/promtail-config.yml`).
- Query logs in Grafana Explore with label `{container=~".*backend.*"}`.

---

## OAuth setup (Google + GitHub)

### Environment variables (backend `.env` — never commit)

```env
FRONTEND_URL=http://localhost:5173

GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...

GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
```

### Callback URLs (register in provider consoles)

| Provider | Authorized redirect URI |
|----------|-------------------------|
| Google   | `http://localhost:5173/auth/callback/google` |
| GitHub   | `http://localhost:5173/auth/callback/github` |

Production: replace host with your domain, e.g. `https://forum.example.com/auth/callback/google`.

### Flow

1. User clicks OAuth button on Login/Register.
2. Frontend calls `GET /api/auth/oauth/{google|github}` → receives redirect URL.
3. Provider redirects to `/auth/callback/{provider}?code=...&state=...`.
4. Frontend posts code to `POST /api/auth/oauth/{provider}/callback`.
5. Backend returns `access_token` + `refresh_token`; frontend stores both.

Check config: `GET /api/auth/oauth/config` → `{ google_enabled, github_enabled, turnstile_site_key }`.

---

## Email & registration

| Variable | Purpose |
|----------|---------|
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM` | Enable outbound email |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | SMTP auth (e.g. Gmail app password) |

Registration sends a confirmation link (`/confirm-email?token=...`) when SMTP is configured; otherwise dev OTP **`123456`** is accepted.

Password reset and notification emails also use SMTP when enabled.

---

## Environment variables

| Variable | Location | Purpose |
|----------|----------|---------|
| `JWT_SECRET` | backend `.env` | JWT signing |
| `DB_TYPE` | backend | `sqlite` or `postgres` |
| `SQLITE_DB` | backend | SQLite filename |
| `DATABASE_URL` | backend | Postgres DSN |
| `FRONTEND_URL` | backend | OAuth redirects, email links |
| `GOOGLE_CLIENT_ID/SECRET` | backend | Google OAuth |
| `GITHUB_CLIENT_ID/SECRET` | backend | GitHub OAuth |
| `SMTP_*` | backend | Email delivery |
| `CLOUDFLARE_TURNSTILE_SECRET` | backend | Server-side CAPTCHA |
| `CLOUDFLARE_TURNSTILE_SITE_KEY` | backend | Public site key (returned via `/auth/oauth/config`) |
| `MINIO_*` | backend | Object storage |
| `VITE_API_URL` | frontend `.env` | API base URL |
| `VITE_TURNSTILE_SITE_KEY` | frontend | Optional CAPTCHA fallback |

**Secrets belong in `backend/.env` and root `.env` only.** Both are listed in `.gitignore`.

---

## Database initialization

1. Delete `backend/nexus_forum.db` (optional fresh start).
2. Start backend → GORM `AutoMigrate` creates all tables.
3. If no users exist, `seedDemoData()` inserts demo accounts.

Demo password for all seeded users: **`password123`**

| Email | Role |
|-------|------|
| amira@example.com | admin |
| moderator@example.com | moderator |
| kai@example.com | user |

---

## Running tests

```bash
cd backend
go test ./...
```

Covers auth, posts, comments, moderation, analytics, search normalization, and resilience helpers.

---

## Backup strategy

**Current status:** no automated backup job is implemented.

**Recommended for production:**

```bash
# SQLite
cp backend/nexus_forum.db "backups/nexus_forum-$(date +%F).db"

# Postgres
pg_dump "$DATABASE_URL" > "backups/nexus_forum-$(date +%F).sql"

# MinIO
mc mirror local/nexus-forum /backups/minio/
```

Schedule daily cron + off-site copy. For HA, use managed Postgres with replicas (not included in this repo).

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `address already in use :8080` | Stop other backend process or change `PORT` |
| OAuth returns 503 | Set `GOOGLE_CLIENT_ID` / `GITHUB_CLIENT_ID` in `.env` |
| Upload fails | Check MinIO running or use local fallback (`LOCAL_UPLOAD_DIR`) |
| Following Users feed empty | Follow users via profile; uses `GET /api/posts/following` |
| Following Communities feed empty | Join communities; uses `GET /api/posts/following-communities` |
| Search misses Cyrillic | Restart API to rebuild indexes; ensure post is `published` |
| Notifications not live | Check WebSocket `GET /api/ws/global` and JWT in `Sec-WebSocket-Protocol` |
| CAPTCHA modal empty | Set `CLOUDFLARE_TURNSTILE_SITE_KEY` + `CLOUDFLARE_TURNSTILE_SECRET` |
| 401 after login | Re-login (JWT requires `sid` session id); clear localStorage if stuck |
| Prometheus target DOWN | Ensure backend container name matches `prometheus.yml` target |

---

## Deployment instructions

1. Build images: `docker compose build`
2. Set production `.env` (strong `JWT_SECRET`, Postgres, MinIO, OAuth URLs, SMTP).
3. Point domain to frontend; reverse-proxy `/api` and `/uploads` to backend.
4. Register production OAuth callback URLs.
5. Enable HTTPS; set `MINIO_USE_SSL=true` if using TLS MinIO.
6. Start stack: `docker compose up -d`
7. Verify: `/health`, `/metrics`, login, feed sorts, notifications.

---

## Production checklist

- [ ] Change `JWT_SECRET` to a long random value
- [ ] Use PostgreSQL instead of SQLite
- [ ] Configure MinIO with non-default credentials
- [ ] Set real OAuth client IDs and production callback URLs
- [ ] Configure SMTP for email confirmation and notifications
- [ ] Enable Turnstile CAPTCHA for suspicious users
- [ ] Configure automated DB + object storage backups
- [ ] Restrict Grafana admin password
- [ ] Put API behind reverse proxy with TLS
- [ ] Set `FRONTEND_URL` and `VITE_API_URL` to production domains

---

## API benchmark (local)

Run with backend on port 8080:

```bash
cd backend && go run ./scripts/benchperf/main.go
```

Target: all listed routes **< 200 ms** average.

---

## Demo users

See [Database initialization](#database-initialization). Admin panel: `/admin` (admin role only). Moderation reports: `/admin/reports`.
