# SAINS API — Developer Guide

**Last updated:** 2026-02-22  
**Stack:** Go 1.23+ · Gin · pgx · sqlc  
**Docs:** [`docs/PRD.md`](docs/PRD.md) · [`docs/TRD.md`](docs/TRD.md)  
**Ref:** `../atomic/docs/BACKEND_PLAN.md` · `../atomic/docs/EXECUTION_PLAN.md`

---

## Quick Start

```bash
# 1. Clone & masuk folder
cd SAINS/api

# 2. Copy env file, isi credentials
cp .env.example .env
# Edit .env → isi DATABASE_URL, JWT_SECRET, dll (lihat bagian Environment di bawah)

# 3. Install deps
go mod tidy

# 4. Run dev server
make dev
# atau: go run cmd/server/main.go

# 5. Test
curl http://localhost:8080/health
# → {"status":"ok","db":"ok","service":"sains-api"}
```

### Admin Dashboard Login

```
URL:      http://localhost:8080/admin/login
Email:    admin@sains.id
Password: SainsAdmin2024!
```

---

## Project Structure

```
api/
├── cmd/server/main.go          ← entry point, Gin setup, graceful shutdown
├── internal/
│   ├── admin/                  ← Admin dashboard handler + embedded templates
│   │   ├── admin_handler.go    ← All admin page handlers (~700 lines)
│   │   └── templates/          ← HTML templates (go:embed)
│   │       ├── layout.html     ← Tabler dark theme layout + sidebar
│   │       ├── dashboard.html  ← Overview stats + recent activity
│   │       ├── users.html      ← User list + search + filter
│   │       ├── user_detail.html← User detail + sessions + anomalies
│   │       ├── anomalies.html  ← Flagged accounts center
│   │       ├── guest_codes.html← Guest code list + generate form
│   │       ├── guest_code_detail.html ← Code detail + login history
│   │       ├── subscriptions.html ← Subscription list + filter
│   │       └── pricing.html    ← Pricing plans + segment groups
│   ├── config/config.go        ← env loader + validation
│   ├── database/postgres.go    ← pgx pool init + close
│   ├── handler/                ← HTTP handler (per resource)
│   ├── middleware/             ← auth, cors, rate limit
│   ├── model/                  ← domain structs (bukan DB model)
│   ├── repository/             ← sqlc generated code (jangan edit manual!)
│   └── service/                ← business logic layer
├── db/
│   ├── migrations/             ← SQL migration files (golang-migrate)
│   └── queries/                ← SQL query files (sqlc)
├── go.mod
├── Makefile
├── .env                        ← ⚠️ JANGAN commit! Ada di .gitignore
└── .env.example                ← Template, BOLEH commit
```

---

## Environment Variables

File: `api/.env` (copy dari `.env.example`)

```bash
# ─── Server ──────────────────────────────────
PORT=8080                          # Port HTTP server
GIN_MODE=debug                     # debug | release

# ─── Database (Supabase) ─────────────────────
# Format: postgresql://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=require
# ⚠️ Password dengan karakter spesial harus URL-encoded:
#    & → %26    ! → %21    # → %23    @ → %40
DATABASE_URL=postgresql://postgres.PROJECT_ID:PASSWORD@aws-1-ap-northeast-2.pooler.supabase.com:6543/postgres?sslmode=require

# ─── Auth ────────────────────────────────────
JWT_SECRET=xxx                     # Generate: openssl rand -hex 32
JWT_EXPIRY=1h                      # Access token lifetime
REFRESH_TOKEN_EXPIRY_DAYS=30       # Refresh token lifetime

# ─── Payment ─────────────────────────────────
XENDIT_API_KEY=xnd_development_xxx
XENDIT_WEBHOOK_TOKEN=xxx
XENDIT_BASE_URL=https://api.xendit.co

# ─── Email ───────────────────────────────────
RESEND_API_KEY=re_xxx
FROM_EMAIL=noreply@sains.id

# ─── CORS ────────────────────────────────────
CORS_ORIGINS=http://localhost:5173

# ─── Admin ───────────────────────────────────
ADMIN_SECRET_KEY=xxx
```

---

## Supabase Setup

### Project Info
- **Dashboard:** https://supabase.com/dashboard/project/pctbsnvklkznxtstavxd
- **Region:** ap-northeast-2 (Seoul)
- **Tier:** Free (500MB storage)

### Connection String
Pakai **Transaction Pooler** (port 6543):
```
postgresql://postgres.pctbsnvklkznxtstavxd:PASSWORD@aws-1-ap-northeast-2.pooler.supabase.com:6543/postgres?sslmode=require
```

**Kenapa pakai pooler?**
- Direct connection (port 5432) butuh IPv4 add-on
- Pooler (port 6543) otomatis proxy lewat IPv4 — gratis
- Cocok untuk Go app yang pakai connection pool sendiri (pgx)

### Kalau Mau Migrasi ke DB Sendiri
```bash
# Export dari Supabase
pg_dump DATABASE_URL_LAMA > backup.sql

# Import ke DB baru
psql DATABASE_URL_BARU < backup.sql

# Update .env
DATABASE_URL=postgresql://user:pass@new-host:5432/sains
```
Nggak perlu ubah code Go — cuma ganti `DATABASE_URL`.

---

## Makefile Commands

```bash
make dev               # Run dev server (GIN_MODE=debug)
make build             # Build production binary → bin/sains-api
make run               # Build + run
make test              # Run all tests
make test-coverage     # Run tests + coverage report

make migrate-up        # Run all pending migrations
make migrate-down      # Rollback 1 migration
make migrate-create    # Create new migration file (prompt for name)

make sqlc              # Regenerate Go code from SQL queries
make deps              # go mod tidy + verify
make lint              # Run golangci-lint
make clean             # Remove build artifacts
```

---

## API Endpoints — REST API (JSON)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check + DB ping | None |
| GET | `/api/ping` | Simple ping/pong | None |
| POST | `/api/auth/register` | Registrasi user baru | None |
| POST | `/api/auth/login` | Login → JWT + refresh cookie | None |
| POST | `/api/auth/guest-login` | Guest login (code + email) | None |
| POST | `/api/auth/logout` | Logout + revoke session | JWT |
| GET | `/api/auth/me` | Get current user info | JWT |
| GET | `/api/plans` | List pricing plans (?product&segment) | None |
| GET | `/api/plans/:id` | Get plan detail | None |
| GET | `/api/quota-status` | Current subscriber + guest counts | None |
| POST | `/api/checkout` | Create Xendit invoice → checkout URL | JWT |
| GET | `/api/subscriptions/me` | List user's subscriptions | JWT |
| GET | `/api/access-check` | Check product access (?product) | JWT |
| POST | `/api/xendit/webhook` | Xendit payment callback | Token |
| POST | `/api/admin/pricing-plans` | Create pricing plan | Admin |
| PUT | `/api/admin/pricing-plans/:id` | Update pricing plan | Admin |
| POST | `/api/admin/guest-codes` | Create guest code | Admin |
| GET | `/api/admin/guest-codes` | List guest codes | Admin |
| GET | `/api/admin/guest-codes/:id` | Guest code detail + logins | Admin |
| DELETE | `/api/admin/guest-codes/:id` | Revoke guest code | Admin |

## Admin Dashboard — HTML Routes (HTMX)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/admin/login` | Login page | None |
| POST | `/admin/login` | Login submit (form) | None |
| GET | `/admin/logout` | Logout + clear cookie | None |
| GET | `/admin/` | Dashboard overview — stats, quota, recent activity | Cookie |
| GET | `/admin/users` | User list — search, filter by role, pagination | Cookie |
| GET | `/admin/users/:id` | User detail — sessions, anomaly logs, subscriptions | Cookie |
| POST | `/admin/users/:id/lock` | Lock user + revoke all sessions | Cookie |
| POST | `/admin/users/:id/unlock` | Unlock user | Cookie |
| GET | `/admin/anomalies` | Flagged accounts — sorted by anomaly score | Cookie |
| GET | `/admin/guest-codes` | Guest code list + usage stats | Cookie |
| GET | `/admin/guest-codes/:id` | Guest code detail + login history | Cookie |
| POST | `/admin/guest-codes/create` | Generate new guest code | Cookie |
| DELETE | `/admin/guest-codes/:id/revoke` | Revoke guest code | Cookie |
| GET | `/admin/subscriptions` | Subscription list — filter by status | Cookie |
| GET | `/admin/pricing` | Pricing plans — grouped by segment | Cookie |

> `/api/*` routes return **JSON** — untuk frontend app (Atomic, sains.id)
> `/admin/*` routes return **HTML** — server-rendered admin dashboard via HTMX

---

## API Response Format

Semua endpoint `/api/*` menggunakan format JSON yang konsisten.

### Success Response
```json
{
  "data": { ... },
  "message": "optional human-readable message"
}
```

### Error Response
```json
{
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Pesan untuk ditampilkan ke user"
  }
}
```

### List Response (with pagination)
```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 156
  }
}
```

### Error Codes

| Code | HTTP | Kapan |
|------|------|-------|
| `VALIDATION_ERROR` | 400 | Input nggak valid (email, password, dll) |
| `UNAUTHORIZED` | 401 | Token nggak ada |
| `INVALID_CREDENTIALS` | 401 | Email/password salah |
| `TOKEN_EXPIRED` | 401 | JWT sudah expired |
| `TOKEN_INVALID` | 401 | JWT rusak/tampered |
| `FORBIDDEN` | 403 | Bukan admin |
| `NOT_FOUND` | 404 | Resource nggak ketemu |
| `CONFLICT` | 409 | Duplicate (email sudah ada) |
| `RATE_LIMITED` | 429 | Terlalu banyak request |
| `QUOTA_FULL` | 503 | Kuota subscriber penuh |
| `INTERNAL_ERROR` | 500 | Generic server error |

### Frontend Usage Pattern
```javascript
const res = await fetch('/api/auth/login', { method: 'POST', body: ... });
const json = await res.json();

if (json.error) {
  // Error path — show json.error.message to user
  // Use json.error.code for programmatic handling
  if (json.error.code === 'TOKEN_EXPIRED') { await refreshToken(); }
  showToast(json.error.message);
} else {
  // Success path — use json.data
  setUser(json.data.user);
  setToken(json.data.access_token);
}
```

### HTMX (Admin Dashboard Only)
Endpoint `/admin/*` (Phase BE-4) akan return **HTML fragments** bukan JSON.
HTMX swap partial HTML langsung ke DOM — nggak perlu parsing JSON.
```
/api/*   → JSON  → untuk frontend app (Atomic, sains.id)
/admin/* → HTML  → untuk admin dashboard (HTMX)
```

---

## Troubleshooting

### "no route to host" saat connect DB
→ Supabase direct connection butuh IPv4. Pakai **pooler URL** (port 6543):
```
aws-1-ap-northeast-2.pooler.supabase.com:6543
```

### "Tenant or user not found"
→ Region salah di pooler URL. Pastikan prefix `aws-1` (bukan `aws-0`).

### Port 8080 already in use
```bash
lsof -ti:8080 | xargs kill -9
```

### Password encoding issues
Karakter spesial dalam password harus URL-encoded di `DATABASE_URL`:
```
&  →  %26
!  →  %21
#  →  %23
@  →  %40
=  →  %3D
```
