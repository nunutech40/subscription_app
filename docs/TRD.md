# TRD — Technical Requirements Document
**Project:** SAINS API & Admin Dashboard  
**Version:** 1.0  
**Date:** 2026-02-22  
**Ref:** `PRD.md` v1.0 · `../../atomic/docs/BACKEND_PLAN.md` v1.1

---

## 1. Arsitektur Sistem

```
┌────────────────────────────────────────────────────────┐
│                     Clients                             │
│                                                        │
│  ┌──────────┐   ┌──────────┐   ┌────────────────┐    │
│  │ Atomic   │   │ sains.id │   │ Admin Browser  │    │
│  │ (SPA)    │   │ (Landing)│   │ (/admin/*)     │    │
│  └────┬─────┘   └────┬─────┘   └──────┬─────────┘    │
│       │               │                │               │
│       └───────────────┼────────────────┘               │
│                       │                                │
│              ┌────────▼────────┐                       │
│              │   CORS / Rate   │                       │
│              │     Limiter     │                       │
│              └────────┬────────┘                       │
│                       │                                │
│    ┌──────────────────▼──────────────────────┐        │
│    │          Go Binary (Gin)                 │        │
│    │                                          │        │
│    │  /api/*  → JSON handlers (REST API)     │        │
│    │  /admin/* → HTML handlers (HTMX SSR)    │        │
│    │                                          │        │
│    │  middleware: auth, admin, cors, rate     │        │
│    │  service: auth, subscription, anomaly   │        │
│    │  repository: sqlc generated (pgx)       │        │
│    └──────────────────┬──────────────────────┘        │
│                       │                                │
│              ┌────────▼────────┐                       │
│              │  Supabase PG    │                       │
│              │  (via pgx pool) │                       │
│              │  port 6543      │                       │
│              └─────────────────┘                       │
│                                                        │
│  External:  Xendit (payment)  ·  Resend (email)       │
└────────────────────────────────────────────────────────┘
```

---

## 2. Tech Stack

| Layer | Teknologi | Versi | Alasan |
|-------|-----------|-------|--------|
| Language | **Go** | 1.23+ | Performa, single binary, stdlib kuat |
| Framework | **Gin** | 1.10+ | Middleware ecosystem, routing, popular |
| DB Driver | **pgx** | v5 | Pure Go, connection pooling, prepared statements |
| Query Layer | **sqlc** | 1.25+ | SQL → Go code, type-safe, zero reflection |
| Migration | **golang-migrate** | 4.x | Reversible migrations, CLI + library |
| Auth JWT | **golang-jwt** | v5 | Standard JWT implementation |
| Password | **bcrypt** | stdlib | Industry standard password hashing |
| Email | **Resend** | Go SDK | Modern transactional email |
| Payment | **Xendit** | REST API | Indonesian payment gateway |
| Admin UI | **Tabler CSS** | 1.0.0-beta20 | Dark theme, component-rich, Bootstrap-based |
| Admin HTMX | **HTMX** | 1.9.12 | Server-driven interactivity, no JS framework |
| Admin Icons | **Tabler Icons** | 3.3.0 | 5,000+ SVG icons |
| Hosting BE | **Railway** | — | Single binary deploy |
| Hosting DB | **Supabase** | — | Managed Postgres, free tier |

---

## 3. Project Structure

```
api/
├── cmd/server/
│   └── main.go                     ← Entry point, router setup, graceful shutdown
│
├── internal/
│   ├── admin/                      ← Admin dashboard (HTMX SSR)
│   │   ├── admin_handler.go        ← All admin page handlers (~700 lines)
│   │   └── templates/              ← HTML templates (go:embed)
│   │       ├── layout.html         ← Tabler dark theme + sidebar nav
│   │       ├── dashboard.html      ← Stats, quota, recent activity
│   │       ├── users.html          ← User list + search + filter
│   │       ├── user_detail.html    ← User detail + sessions + anomalies
│   │       ├── anomalies.html      ← Flagged accounts center
│   │       ├── guest_codes.html    ← Guest code management
│   │       ├── guest_code_detail.html ← Code detail + login history
│   │       ├── subscriptions.html  ← Subscription list
│   │       └── pricing.html        ← Pricing plans by segment
│   │
│   ├── config/config.go            ← Environment loader + validation
│   ├── database/postgres.go        ← pgx pool init + graceful close
│   ├── handler/                    ← JSON API handlers
│   │   ├── auth_handler.go         ← Register, login, logout, me, guest-login
│   │   ├── plan_handler.go         ← Pricing plans CRUD
│   │   ├── subscription_handler.go ← Checkout, webhook, access-check
│   │   └── guest_handler.go        ← Guest code CRUD (API endpoints)
│   ├── middleware/                  ← HTTP middleware
│   │   ├── auth.go                 ← JWT verification
│   │   ├── admin.go                ← Admin role gate
│   │   ├── cors.go                 ← CORS configuration
│   │   └── rate_limit.go           ← Token bucket rate limiter
│   ├── model/                      ← Domain structs
│   ├── repository/                 ← sqlc generated (DO NOT EDIT)
│   └── service/                    ← Business logic
│       ├── token_service.go        ← JWT create/verify/refresh
│       └── email_service.go        ← Resend integration
│
├── db/
│   ├── migrations/
│   │   ├── 000001_init_schema.up.sql    ← All tables + indexes + seeds
│   │   └── 000001_init_schema.down.sql  ← Drop all
│   └── queries/                    ← SQL source for sqlc
│       ├── users.sql
│       ├── sessions.sql
│       ├── subscriptions.sql
│       ├── pricing_plans.sql
│       ├── guest_codes.sql
│       ├── anomaly_logs.sql
│       ├── products.sql
│       └── system_config.sql
│
├── sqlc.yaml                       ← sqlc configuration
├── go.mod / go.sum
├── Makefile
├── .env.example
└── .gitignore
```

---

## 4. Database Schema

### 4.1 Tables Overview (10 tables)

| Table | Purpose | Key Relations |
|-------|---------|---------------|
| `products` | Produk SAINS (atomic, energi, dll) | → pricing_plans, subscriptions |
| `pricing_plans` | Harga per segment × durasi | → products |
| `users` | User accounts (admin, subscriber, guest) | → sessions, subscriptions |
| `sessions` | Active JWT sessions | → users |
| `subscriptions` | Langganan user per produk | → users, pricing_plans |
| `guest_codes` | Kode akses trial | → products |
| `guest_logins` | Log login per email per code | → guest_codes |
| `anomaly_logs` | Event mencurigakan per user | → users |
| `access_logs` | Request access log | → users |
| `system_config` | Key-value config (quota, limits) | — |

### 4.2 Key Indexes

```sql
-- Performance critical
CREATE UNIQUE INDEX ON users (email);
CREATE INDEX ON sessions (user_id, is_active);
CREATE INDEX ON subscriptions (user_id, product_id, status);
CREATE INDEX ON anomaly_logs (user_id, created_at DESC);
CREATE INDEX ON guest_logins (guest_code_id, email);
```

### 4.3 Connection Config

```
PostgreSQL: Supabase (ap-northeast-2, Seoul)
Connection: Transaction pooler (port 6543)
Pool: min=2, max=10
SSL: require
```

---

## 5. API Architecture

### 5.1 Two Response Types

```
/api/*    → JSON (REST API for SPA clients)
/admin/*  → HTML (server-rendered via HTMX)
```

### 5.2 JSON Response Format

```json
// Success
{ "data": { ... }, "message": "optional" }

// Error
{ "error": { "code": "MACHINE_READABLE", "message": "Human readable" } }

// List with pagination
{ "data": [...], "meta": { "page": 1, "per_page": 20, "total": 156 } }
```

### 5.3 Error Codes

| Code | HTTP | When |
|------|------|------|
| `VALIDATION_ERROR` | 400 | Invalid input |
| `UNAUTHORIZED` | 401 | No/invalid token |
| `INVALID_CREDENTIALS` | 401 | Wrong email/password |
| `TOKEN_EXPIRED` | 401 | JWT expired |
| `FORBIDDEN` | 403 | Not admin |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Duplicate (email exists) |
| `RATE_LIMITED` | 429 | Too many requests |
| `QUOTA_FULL` | 503 | Subscriber quota full |

---

## 6. Auth Architecture

### 6.1 Token Flow

```
Login → JWT access token (1h) + refresh token (30d, httpOnly cookie)
         ↓
Request → Authorization: Bearer <jwt> → middleware verifies → handler
         ↓
Refresh → POST /api/auth/refresh → new access token (via cookie)
         ↓
Logout → Revoke session + clear cookie
```

### 6.2 Single Session Rule

```
User logs in from Device B:
  1. Check: any active session for this user?
  2. YES → revoke old session + log anomaly event (score +5)
  3. Create new session for Device B
  4. Return JWT for Device B
```

### 6.3 Guest Auth Flow

```
Guest enters: email + code
  1. Validate: code active & not expired?
  2. Check: email login count < max_logins_per_email?
  3. YES → create session (24h), increment login count
  4. NO → reject ("Trial sudah habis")
```

---

## 7. Admin Dashboard Architecture

### 7.1 Rendering Strategy

```
Go html/template + embed.FS → per-page template parsing
                             → layout.html + page.html
                             → no "content" name collision

Template loading per-render (not pre-compiled all):
  render("users", data) → parse(layout.html + users.html) → execute("layout", data)
```

**Why per-page parsing?**
All page templates define `{{define "content"}}`. Go's template parser uses the *last* definition when all are parsed together. Per-page parsing ensures each page's content block is unique.

### 7.2 UI Stack

| Component | Source | Purpose |
|-----------|--------|---------|
| Tabler CSS | CDN (beta20) | Dark theme, cards, tables, forms |
| Tabler Icons | CDN (3.3.0) | Navigation + action icons |
| HTMX | CDN (1.9.12) | Search, filters, inline actions |

### 7.3 Pages Implemented

| Page | Route | Template | Features |
|------|-------|----------|----------|
| Dashboard | `/admin/` | `dashboard.html` | 4 stat cards, quota bars, recent subs, anomalies, guest codes |
| Users | `/admin/users` | `users.html` | Search, role filter, pagination, status badges |
| User Detail | `/admin/users/:id` | `user_detail.html` | Profile, sessions, anomaly log, subscription history |
| Anomalies | `/admin/anomalies` | `anomalies.html` | Flagged users, score, last event, lock/unlock |
| Guest Codes | `/admin/guest-codes` | `guest_codes.html` | Generate form, list, usage stats |
| Code Detail | `/admin/guest-codes/:id` | `guest_code_detail.html` | Login history per email |
| Subscriptions | `/admin/subscriptions` | `subscriptions.html` | Status filter, user links |
| Pricing | `/admin/pricing` | `pricing.html` | Segment groups, IDR format |

### 7.4 Embedded Assets

```go
//go:embed templates/*.html
var templateFS embed.FS
```

All templates compiled into the Go binary. Zero external file dependencies at runtime.

---

## 8. External Integrations

### 8.1 Xendit (Payment)

```
Checkout Flow:
  1. POST /api/checkout → create Xendit invoice (REST API)
  2. Redirect user to Xendit payment page
  3. User pays → Xendit sends webhook → POST /api/xendit/webhook
  4. Verify HMAC signature → activate subscription
```

### 8.2 Resend (Email)

```
Trigger: subscription activated (via Xendit webhook)
  → Send welcome email with subscription details
  → Send renewal reminder 7 days before expiry (future)
```

---

## 9. Security Controls

| Control | Implementation | Status |
|---------|----------------|--------|
| Password Hashing | bcrypt (cost 10) | ✅ |
| JWT Signing | HMAC-SHA256 | ✅ |
| Single Session | Revoke on new login | ✅ |
| Rate Limiting | Token bucket (100/min global, 5/min auth) | ✅ |
| CORS | Whitelist origins | ✅ |
| Webhook Auth | HMAC signature verification | ✅ |
| SQL Injection | sqlc parameterized queries | ✅ |
| XSS (Admin) | Go html/template auto-escaping | ✅ |
| Admin Auth | Cookie-based JWT (admin_token httpOnly) | ✅ |

---

## 10. Deployment

### 10.1 Current (Development)

```bash
# Local dev
go run cmd/server/main.go

# With Makefile
make dev
```

### 10.2 Production (Planned)

```
Build:   go build -o bin/sains-api cmd/server/main.go
Deploy:  Railway (single binary, auto-deploy from git)
DB:      Supabase Postgres (managed, free tier)
Domain:  api.sains.id (planned)
```

### 10.3 Environment Variables

See `api/.env.example` for full list. Critical vars:
- `DATABASE_URL` — Supabase pooler connection string
- `JWT_SECRET` — HMAC signing key (openssl rand -hex 32)
- `XENDIT_API_KEY` — Payment gateway key
- `XENDIT_WEBHOOK_TOKEN` — Webhook HMAC verification
- `RESEND_API_KEY` — Email service key

---

## 11. Known Limitations & TODO

| Item | Priority | Detail |
|------|----------|--------|
| Chart.js revenue trends | Medium | Revenue line chart not yet implemented |
| Inline price editing | Medium | Template ready, handler not wired |
| System config UI | Low | Currently managed via DB directly |
| Docker build | Low | Not yet containerized |
| Load testing | Low | No benchmarks yet |
| Monitoring | Low | No Sentry/Grafana integration |

---

## 12. Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-02-22 | Initial TRD — covers Phase BE-1 through BE-4 |
