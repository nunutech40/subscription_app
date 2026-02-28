# 🚀 SAINS Atomic — Deployment Guide

> **Last deployed:** 2026-02-27
> **Server:** IDCloudHost VPS
> **Domain:** https://sains-atomic.web.id

---

## 📡 Server Info

| Item | Value |
|------|-------|
| **Provider** | IDCloudHost |
| **IP** | `103.181.143.73` (tersembunyi di balik Cloudflare) |
| **OS** | Ubuntu 24.04 LTS |
| **Spek** | 2 vCPU, 2GB RAM, 40GB Disk |
| **Username** | `nunuadmin` |
| **Domain** | `sains-atomic.web.id` ✅ Active |
| **Monthly Cost** | Rp 100.000 (VPS) + Rp 50.000/tahun (domain) |

---

## 🌐 Domain & Cloudflare

| Item | Value |
|------|-------|
| **Domain Registrar** | DomainNesia |
| **DNS Provider** | Cloudflare (Free plan) |
| **Nameservers** | `beau.ns.cloudflare.com`, `coraline.ns.cloudflare.com` |
| **SSL** | Cloudflare Universal SSL (auto-managed) |
| **SSL Mode** | Full |
| **Proxy** | ON — IP asli tersembunyi |
| **Cloudflare Account** | Nunutech4.0@gmail.com |

### DNS Records (di Cloudflare)
| Type | Name | Value | Proxy |
|------|------|-------|-------|
| A | `@` | `103.181.143.73` | ☁️ Proxied |
| A | `www` | `103.181.143.73` | ☁️ Proxied |
| A | `app` | `103.181.143.73` | ☁️ Proxied |
| A | `mail` | `36.50.77.96` | DNS only |
| MX | `@` | `mx2.mailspace.id` | DNS only |

### URL Structure
| URL | Fungsi |
|-----|--------|
| `https://sains-atomic.web.id` | Landing page (marketing) |
| `https://app.sains-atomic.web.id` | Atomic App (3D periodic table) |
| `https://sains-atomic.web.id/api/` | Backend API |

### Kenapa Cloudflare?
- ✅ SSL/HTTPS gratis & otomatis (gak perlu Certbot)
- ✅ IP VPS tersembunyi (anti DDoS)
- ✅ CDN global (website lebih cepat)
- ✅ DDoS protection bawaan

---

## 🔑 SSH Access

```bash
# Login ke VPS
ssh nunuadmin@103.181.143.73

# SSH key yang dipakai: nugraha17 (sudah di-register di IDCloudHost)
# Key location di Mac: ~/.ssh/id_ed25519 (atau ~/.ssh/id_rsa)
```

---

## 📂 Struktur File di VPS

```
/home/nunuadmin/
├── sains-api/
│   ├── sains-api          # Go binary (compiled for linux/amd64)
│   ├── .env               # Environment variables
│   ├── templates/         # Admin dashboard HTML templates
│   └── static/            # Admin dashboard static files
├── sains-landing/
│   ├── index.html          # Landing page
│   ├── app.js              # Frontend JS
│   ├── style.css           # Styles
│   ├── privacy.html        # Privacy policy
│   ├── terms.html          # Terms of service
│   └── img/                # Images
├── sains-atomic-app/
│   ├── index.html          # Atomic SPA entry
│   ├── assets/             # JS/CSS bundles (Vite build)
│   ├── onboarding/         # Onboarding assets
│   └── textures/           # 3D textures
```

---

## 🗄️ Database

| Item | Value |
|------|-------|
| **Engine** | PostgreSQL 16 |
| **Database** | `sains_db` |
| **User** | `sains_user` |
| **Password** | `SainsAtomic2026!` |
| **Connection** | `postgresql://sains_user:SainsAtomic2026!@localhost:5432/sains_db?sslmode=disable` |

### Akses Database Manual
```bash
# Via SSH
ssh nunuadmin@103.181.143.73

# Login ke psql
sudo -u postgres psql -d sains_db

# Atau dengan user sains_user
psql "postgresql://sains_user:SainsAtomic2026!@localhost:5432/sains_db?sslmode=disable"
```

### Backup Database
```bash
# Backup
ssh nunuadmin@103.181.143.73 "sudo -u postgres pg_dump sains_db" > backup_sains_$(date +%Y%m%d).sql

# Restore (di VPS baru)
cat backup_sains_YYYYMMDD.sql | ssh user@NEW_VPS "sudo -u postgres psql -d sains_db"
```

---

## 🌐 Nginx Config

**File:** `/etc/nginx/sites-available/sains-atomic`

```nginx
# Block direct IP access (return nothing)
server {
    listen 80 default_server;
    listen 443 default_server ssl;
    server_name _;
    ssl_certificate /etc/nginx/ssl/selfsigned.crt;
    ssl_certificate_key /etc/nginx/ssl/selfsigned.key;
    return 444;
}

# Main site (via Cloudflare only)
server {
    listen 80;
    listen 443 ssl;
    server_name sains-atomic.web.id www.sains-atomic.web.id;

    ssl_certificate /etc/nginx/ssl/selfsigned.crt;
    ssl_certificate_key /etc/nginx/ssl/selfsigned.key;

    root /home/nunuadmin/sains-landing;
    index index.html;

    # API → Go backend
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $http_cf_connecting_ip;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Landing page
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    server_tokens off;
}
```

### Nginx Commands
```bash
sudo nginx -t                          # Test config
sudo systemctl reload nginx            # Reload
sudo tail -f /var/log/nginx/error.log  # Error logs
sudo tail -f /var/log/nginx/access.log # Access logs
```

---

## ⚙️ Systemd Service

**File:** `/etc/systemd/system/sains-api.service`

### Service Commands
```bash
sudo systemctl status sains-api       # Status
sudo systemctl restart sains-api      # Restart
sudo systemctl stop sains-api         # Stop
sudo journalctl -u sains-api -f       # Live log
sudo journalctl -u sains-api -n 50    # 50 baris terakhir
```

---

## 🔥 Firewall (UFW)

```bash
sudo ufw status

# Ports yang dibuka:
# - 22 (SSH)
# - 80 (HTTP)
# - 443 (HTTPS)
```

---

## 🔄 Deploy Ulang (Update Code)

### Update API
```bash
# 1. Build di Mac (dari folder SAINS/api)
cd api
GOOS=linux GOARCH=amd64 go build -o sains-api-linux ./cmd/server/

# 2. Upload ke VPS
scp sains-api-linux nunuadmin@103.181.143.73:/home/nunuadmin/sains-api/sains-api

# 3. Restart service
ssh nunuadmin@103.181.143.73 "chmod +x /home/nunuadmin/sains-api/sains-api && sudo systemctl restart sains-api"

# 4. Clean up
rm sains-api-linux
```

### Update Landing Page
```bash
# Upload semua file landing
scp -r landing/student-kimia-v1/* nunuadmin@103.181.143.73:/home/nunuadmin/sains-landing/

# Nginx auto-serve, no restart needed
```

### Update Atomic App
```bash
# 1. Build di Mac (dari folder SAINS/atomic)
cd atomic
npm run build

# 2. Upload ke VPS
scp -r dist/* nunuadmin@103.181.143.73:/home/nunuadmin/sains-atomic-app/

# Nginx auto-serve, no restart needed
```

### Update .env
```bash
# Edit langsung di VPS
ssh nunuadmin@103.181.143.73 "nano /home/nunuadmin/sains-api/.env"

# Restart setelah edit
ssh nunuadmin@103.181.143.73 "sudo systemctl restart sains-api"
```

---

## 🔒 Security Notes

- **IP tersembunyi:** Cloudflare proxy menyembunyikan IP VPS. Orang hanya lihat IP Cloudflare.
- **Akses IP langsung ditolak:** Nginx return 444 (connection drop) untuk akses via IP.
- **CORS:** Hanya izinkan `https://sains-atomic.web.id`, `https://www.sains-atomic.web.id`, dan `https://app.sains-atomic.web.id`.
- **SSH:** Key-based authentication only (nugraha17).
- **Server tokens:** Hidden (`server_tokens off`).

---

## 🚚 Migrasi ke VPS Baru

```bash
# 1. Backup database
ssh nunuadmin@103.181.143.73 "sudo -u postgres pg_dump sains_db" > backup.sql

# 2. Copy semua files
scp -r nunuadmin@103.181.143.73:/home/nunuadmin/sains-api ./backup-api
scp -r nunuadmin@103.181.143.73:/home/nunuadmin/sains-landing ./backup-landing

# 3. Di VPS baru: install dependencies
sudo apt update && sudo apt install -y nginx postgresql postgresql-contrib ufw

# 4. Create database & restore
sudo -u postgres psql -c "CREATE USER sains_user WITH PASSWORD 'SainsAtomic2026!';"
sudo -u postgres psql -c "CREATE DATABASE sains_db OWNER sains_user;"
cat backup.sql | sudo -u postgres psql -d sains_db

# 5. Upload files, setup systemd & nginx (same as initial setup)
# 6. Update Cloudflare DNS → IP baru (SSL otomatis)
```

---

## 📦 GitHub Repositories

| Project | Repository | Branch |
|---------|-----------|--------|
| **API (Go)** | `nunutech40/subscription_app` | main |
| **Landing Page** | `nunutech40/landingpage-marketing` | main |
| **Atomic App** | `nunutech40/atomic` | main |

---

## 📋 Credentials Quick Reference

| Credential | Value | Catatan |
|-----------|-------|---------|
| SSH | `ssh nunuadmin@103.181.143.73` | Key: nugraha17 |
| DB User | `sains_user` | Password: `SainsAtomic2026!` |
| DB Name | `sains_db` | PostgreSQL 16 |
| JWT Secret | Auto-generated | Lihat di `.env` |
| Admin Key | Auto-generated | Lihat di `.env` |
| Cloudflare | Nunutech4.0@gmail.com | Free plan |
| Domain | DomainNesia (Rizka Fajar Nugraha) | Expires 2027-02-27 |
| Email Hosting | DomainNesia Mailspace (Rp 60.000/thn) | Company ID: sainsato |
| Midtrans | `SB-Mid-server-GANTI_NANTI` | ⚠️ Belum diisi |

---

## 📧 SMTP Email

| Setting | Value |
|---------|-------|
| **Email** | `noreply@sains-atomic.web.id` |
| **Password** | `45Shc3ahW7N0Y` |
| **Mail Server** | `mail.sains-atomic.web.id` |
| **Mailspace Login** | `login.mailspace.id` |
| **SMTP Host** | `mx2.mailspace.id` (⚠️ bukan `mail.sains-atomic.web.id` — SSL cert mismatch) |
| **SMTP Port** | `465` (SSL) / `587` (TLS) |
| **Webmail** | `login.mailspace.id` |

### ✅ SMTP — SOLVED (2026-02-28)
Port 465/587 sudah dibuka oleh IDCloudHost + DomainNesia whitelist IP VPS.

**Catatan penting:**
- SMTP_HOST harus pakai `mx2.mailspace.id` (bukan `mail.sains-atomic.web.id`) karena SSL cert terdaftar atas nama `mx2.mailspace.id`
- Kalau pakai `mail.sains-atomic.web.id` akan error: `x509: certificate is valid for mx2.mailspace.id`
- Email terkirim dari `noreply@sains-atomic.web.id` via port 465 (SSL)
- **HARUS** ada header `Date` dan `Message-ID` di email, tanpa ini Gmail drop email tanpa jejak
- SMTP dial punya timeout 15 detik + 30 detik deadline agar goroutine tidak hang jika port blocked
- DomainNesia punya rate limiting — jangan spam SMTP connection, bisa kena temporary block

### ⚠️ DNS — Catatan Penting
- **DMARC**: `_dmarc.sains-atomic.web.id` → `v=DMARC1; p=none; ...`
  - ❌ JANGAN pakai `p=reject` sampai SPF/DKIM 100% verified, Gmail akan buang email tanpa bekas
  - ✅ Pakai `p=none` dulu untuk monitoring
- **SPF**: `v=spf1 a mx include:relay.mailchannels.net ~all` (DomainNesia pakai MailChannels relay)
- **DKIM**: Sudah configured via `default._domainkey` TXT record

### 🚀 Deployment Paths (PENTING!)
| Komponen | Path di VPS | Nginx Config |
|----------|-------------|--------------|
| **API** | `/home/nunuadmin/sains-api/sains-api` | reverse proxy :8080 |
| **Atomic App** | `/home/nunuadmin/sains-atomic-app/` | `sains-atomic-app` |
| **Landing Page** | `/home/nunuadmin/sains-landing/` | `sains-atomic` |

⚠️ Hati-hati: `sains-atomic` dan `sains-atomic-app` adalah folder BERBEDA!
- `sains-atomic-app` → `app.sains-atomic.web.id` (Atomic App)
- `sains-landing` → `sains-atomic.web.id` (Landing Page)

### 🧹 Cloudflare Cache
Setelah deploy atomic app, **WAJIB** purge Cloudflare cache:
1. Buka `dash.cloudflare.com` → domain `sains-atomic.web.id`
2. **Caching** → **Configuration** → **Purge Everything**
Tanpa purge, user akan tetap load JS bundle lama.

### 👤 Admin Login
| Setting | Value |
|---------|-------|
| **URL** | `https://sains-atomic.web.id/admin/login` |
| **Email** | `nunutech4.0@gmail.com` |
| **Password** | `Admin1234!` |

---

## ✅ Status Checklist

- [x] VPS created & running
- [x] SSH access working
- [x] PostgreSQL installed & configured
- [x] Database migrations applied
- [x] Go API built & deployed
- [x] Nginx configured as reverse proxy
- [x] Landing page deployed
- [x] Atomic App deployed (`app.sains-atomic.web.id`)
- [x] Firewall configured (UFW)
- [x] Systemd service (auto-restart)
- [x] Domain DNS configured (Cloudflare)
- [x] SSL/HTTPS via Cloudflare (auto-managed)
- [x] IP tersembunyi (Cloudflare proxy)
- [x] Direct IP access blocked
- [x] CORS configured (3 origins)
- [x] GitHub repos up-to-date
- [x] Email account created (`noreply@sains-atomic.web.id`)
- [x] SMTP port unblocked ✅ (IDCloudHost + DomainNesia whitelist)
- [x] Email OTP working ✅ (Date + Message-ID headers added)
- [x] DMARC fixed ✅ (p=none, was p=reject causing Gmail drops)
- [x] SMTP timeout ✅ (15s dial + 30s deadline, no more goroutine hangs)
- [x] Onboarding flow working ✅ (hash check fix + correct deploy path)
- [ ] Midtrans keys configured
- [ ] Midtrans webhook URL set

