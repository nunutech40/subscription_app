# PRD — Product Requirements Document
**Project:** SAINS API & Admin Dashboard  
**Version:** 1.0  
**Date:** 2026-02-22  
**Status:** Phase BE-1 ✅ · Phase BE-2 ✅ · Phase BE-3 ✅ · Phase BE-4 ✅ (partial) · Phase BE-5 ⏳

---

## 1. Latar Belakang & Tujuan

SAINS adalah **platform multi-produk pembelajaran sains**. Atomic adalah produk pertama. Backend ini adalah satu server terpusat yang mengelola autentikasi, langganan, pembayaran, dan administrasi untuk semua produk SAINS.

**Tujuan utama:**
1. Menyediakan **auth system** yang aman dengan JWT + single session rule
2. Mengelola **subscription & payment** via Xendit secara otomatis
3. Memberikan **guest access** melalui guest code system yang viral-friendly
4. Menyediakan **admin dashboard** untuk monitoring dan manajemen real-time
5. Mendeteksi dan mencegah **account sharing** dengan anomaly scoring

---

## 2. User Personas

### 2.1 End User (Subscriber)
- Pelajar SMA/mahasiswa yang berlangganan produk SAINS
- Mendaftar via email → bayar via Xendit → akses konten
- **Batasan:** 1 session aktif per akun (anti account sharing)

### 2.2 Guest User
- Calon user yang mendapat guest code dari admin
- Login via email + guest code → akses trial terbatas
- **Batasan:** 2x login per email per code, durasi sesuai konfigurasi

### 2.3 Admin
- Pengelola platform SAINS
- Akses admin dashboard untuk monitoring dan manajemen
- **Capabilities:** manage users, codes, subscriptions, pricing, anomalies

---

## 3. Fitur & Requirements

### 3.1 Authentication System ✅

| Requirement | Detail | Status |
|-------------|--------|--------|
| Register | Email + password → bcrypt hash → user created | ✅ |
| Login | Email + password → JWT access + refresh cookie | ✅ |
| Guest Login | Email + guest code → validasi → session 24 jam | ✅ |
| Logout | Revoke session + clear cookie | ✅ |
| Single Session | Login baru → revoke session lama + log anomaly | ✅ |
| JWT | Access token (1h) + refresh token (30d, httpOnly cookie) | ✅ |

### 3.2 Subscription & Payment ✅

| Requirement | Detail | Status |
|-------------|--------|--------|
| Pricing Plans | Multi-segment pricing (global/student/parent) × durasi | ✅ |
| Checkout | Create Xendit invoice → redirect ke payment page | ✅ |
| Webhook | Terima callback Xendit → activate subscription | ✅ |
| Access Check | Cek apakah user punya langganan aktif per produk | ✅ |
| Quota Control | Max subscriber + max guest (configurable via DB) | ✅ |
| Email | Welcome email + renewal reminder via Resend | ✅ |

### 3.3 Guest Code System ✅

| Requirement | Detail | Status |
|-------------|--------|--------|
| Generate Code | Admin create code (label, max logins, durasi) | ✅ |
| Code Format | `ATOM-XXXX` (4 char, human-friendly) | ✅ |
| Login Limit | Max 2 login per email per code (configurable) | ✅ |
| Expiry | Adjustable (1–168 jam) | ✅ |
| Tracking | Catat email, login count, last login per code | ✅ |
| Revocation | Admin bisa revoke code kapan saja | ✅ |

### 3.4 Anomaly Detection ✅

| Requirement | Detail | Status |
|-------------|--------|--------|
| Score System | Setiap event mencurigakan tambah skor anomaly | ✅ |
| Events Tracked | Multi-session, IP switch, device switch, rapid login | ✅ |
| Flagging | User dengan skor > threshold ditampilkan di dashboard | ✅ |
| Admin Action | Lock/unlock user + revoke semua session | ✅ |

### 3.5 Admin Dashboard ✅

| Requirement | Detail | Status |
|-------------|--------|--------|
| Dashboard Overview | Revenue, subscribers, guests, users, recent activity | ✅ |
| Quota Widget | Progress bar subscriber (x/200) + guest (x/50) | ✅ |
| User Management | Search, filter role, detail, lock/unlock | ✅ |
| User Detail | Sessions aktif, anomaly logs, subscription history | ✅ |
| Anomaly Center | Flagged users sorted by score, last event | ✅ |
| Guest Code Mgmt | Generate, list, detail (login history), revoke | ✅ |
| Subscription List | Filter by status (active/pending/expired) | ✅ |
| Pricing View | Plans grouped by segment, IDR formatting | ✅ |
| Revenue Analytics | Chart.js trends, segment breakdown | ✅ |
| Inline Price Edit | Edit harga langsung di dashboard | ✅ |
| System Config | Edit config (quota, limits) dari dashboard | ⏳ |
| Admin Auth | Cookie-based JWT admin authentication | ✅ |

---

## 4. Non-Functional Requirements

### 4.1 Performance
- API response time < 200ms (p95)
- Dashboard page load < 1s
- Database connection pool: min 2, max 10

### 4.2 Security
- Password: bcrypt (cost 10)
- JWT: signed with HMAC-SHA256
- Single active session per user
- Rate limiting: 100 req/min global, 5 req/min auth endpoints
- CORS whitelist: production origins only
- Webhook: HMAC signature verification

### 4.3 Reliability
- Graceful shutdown dengan context propagation
- Database health check endpoint
- Anomaly logging untuk semua event mencurigakan

### 4.4 Scalability
- Multi-product architecture (pricing_plans.product_id)
- Configurable quotas via system_config table
- Stateless JWT auth (horizontal scalable)

---

## 5. User Flows

### 5.1 Subscriber Flow
```
Register → Login → Checkout → Bayar (Xendit) → Webhook activates subscription → Access content
```

### 5.2 Guest Flow
```
Admin generates code → Share code → Guest enters email + code → Validate → Session 24h → Expire
```

### 5.3 Admin Flow
```
Login (admin role) → Dashboard overview → Manage users/codes/subscriptions/pricing → Monitor anomalies
```

---

## 6. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Uptime | > 99.5% | Health check monitoring |
| Auth Latency | < 100ms | JWT verification time |
| Payment Success | > 95% | Xendit webhook success rate |
| Guest Conversion | > 10% | Guest → subscriber conversion |
| Account Sharing | < 5% | Anomaly flagged accounts |

---

## 7. Dependencies

| Dependency | Purpose | Status |
|------------|---------|--------|
| Supabase Postgres | Primary database | ✅ Active |
| Xendit | Payment gateway (Indonesia) | ✅ Integrated |
| Resend | Transactional email | ✅ Integrated |
| Tabler CSS | Admin dashboard UI framework | ✅ CDN |
| HTMX | Admin dashboard interactivity | ✅ CDN |

---

## 8. Roadmap

```
Phase BE-1: Foundation        ✅  Auth, JWT, sessions, migrations
Phase BE-2: Subscription      ✅  Xendit, checkout, webhook, email
Phase BE-3: Guest + Security  ✅  Guest codes, anomaly scoring
Phase BE-4: Admin Dashboard   ✅  Tabler + HTMX, 7 pages, management UI
Phase BE-5: Hardening         ⏳  Rate limit prod, audit logs, Docker, monitoring
```

---

## 9. Out of Scope (v1.0)

- Mobile app (hanya web-based)
- Multi-currency (IDR only for now)
- Auto-renewal subscription (manual renewal via new checkout)
- Real-time notifications (WebSocket)
- A/B testing infrastructure
