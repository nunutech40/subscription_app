#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════
# SAINS — VPS Deployment Script
# Jalankan di VPS (Ubuntu 22.04+)
# Usage: bash deploy.sh
# ═══════════════════════════════════════════════════════════════════════

set -e

DOMAIN="${1:-_}"  # Pass domain as arg, or use _ for IP-only
APP_DIR="/opt/sains"
API_DIR="$APP_DIR/api"
FRONTEND_DIR="/var/www/atomic"
REPO="https://github.com/nunutech40/subscription_app.git"

echo "═══════════════════════════════════════════"
echo "  🚀 SAINS Deployment"
echo "═══════════════════════════════════════════"

# ── 1. System packages ───────────────────────────────────────────────
echo "📦 [1/7] Installing system packages..."
sudo apt update -y
sudo apt install -y nginx certbot python3-certbot-nginx git curl

# ── 2. Install Go (if not installed) ─────────────────────────────────
if ! command -v go &> /dev/null; then
    echo "📦 [2/7] Installing Go 1.23..."
    curl -OL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.23.6.linux-amd64.tar.gz
    rm go1.23.6.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin
else
    echo "✅ [2/7] Go already installed: $(go version)"
fi

# ── 3. Install Node.js (for frontend build) ──────────────────────────
if ! command -v node &> /dev/null; then
    echo "📦 [3/7] Installing Node.js 20..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt install -y nodejs
else
    echo "✅ [3/7] Node.js already installed: $(node -v)"
fi

# ── 4. Clone/Pull repo ───────────────────────────────────────────────
echo "📥 [4/7] Setting up project..."
sudo mkdir -p "$APP_DIR" "$FRONTEND_DIR"
sudo chown -R $USER:$USER "$APP_DIR"

if [ -d "$API_DIR/.git" ]; then
    cd "$API_DIR"
    git pull origin main
else
    git clone "$REPO" "$API_DIR"
fi

# ── 5. Build API ─────────────────────────────────────────────────────
echo "🔨 [5/7] Building API..."
cd "$API_DIR"

# Check .env exists
if [ ! -f .env ]; then
    cp .env.example .env
    echo ""
    echo "⚠️  PENTING: Edit file .env dulu!"
    echo "   nano $API_DIR/.env"
    echo ""
    echo "   Yang WAJIB diisi:"
    echo "   - DATABASE_URL"
    echo "   - JWT_SECRET (generate: openssl rand -hex 32)"
    echo "   - SMTP_HOST, SMTP_USER, SMTP_PASS"
    echo "   - GIN_MODE=release"
    echo "   - CORS_ORIGINS=https://$DOMAIN"
    echo "   - FRONTEND_URL=https://$DOMAIN"
    echo ""
    echo "   Setelah edit, jalankan: bash deploy.sh $DOMAIN"
    exit 1
fi

CGO_ENABLED=0 go build -ldflags="-s -w" -o sains-api ./cmd/server/

# ── 6. Build Frontend ────────────────────────────────────────────────
echo "🎨 [6/7] Building frontend..."

# Cek apakah ada folder atomic di level atas
ATOMIC_DIR=""
if [ -d "$APP_DIR/../atomic" ]; then
    ATOMIC_DIR="$APP_DIR/../atomic"
elif [ -d "$APP_DIR/../../SAINS/atomic" ]; then
    ATOMIC_DIR="$APP_DIR/../../SAINS/atomic"
fi

if [ -n "$ATOMIC_DIR" ]; then
    cd "$ATOMIC_DIR"
    npm ci
    npm run build
    sudo cp -r dist/* "$FRONTEND_DIR/"
    echo "✅ Frontend built & deployed to $FRONTEND_DIR"
else
    echo "⚠️  Frontend (atomic) not found. Upload manually:"
    echo "   1. Di lokal: cd atomic && npm run build"
    echo "   2. SCP:     scp -r dist/* user@vps:$FRONTEND_DIR/"
fi

# ── 7. Setup services ────────────────────────────────────────────────
echo "⚙️  [7/7] Setting up services..."
cd "$API_DIR"

# Systemd service
sudo cp deploy/sains-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable sains-api
sudo systemctl restart sains-api

# Nginx
sudo cp deploy/nginx/sains.conf /etc/nginx/sites-available/sains
sudo ln -sf /etc/nginx/sites-available/sains /etc/nginx/sites-enabled/sains
sudo rm -f /etc/nginx/sites-enabled/default

# Update domain in nginx config
if [ "$DOMAIN" != "_" ]; then
    sudo sed -i "s/server_name _;/server_name $DOMAIN www.$DOMAIN;/" /etc/nginx/sites-available/sains
fi

sudo nginx -t && sudo systemctl reload nginx

# ── SSL (if domain provided) ─────────────────────────────────────────
if [ "$DOMAIN" != "_" ]; then
    echo "🔒 Setting up SSL for $DOMAIN..."
    sudo certbot --nginx -d "$DOMAIN" -d "www.$DOMAIN" --non-interactive --agree-tos --email admin@$DOMAIN || true
fi

# ── Done ──────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════"
echo "  ✅ SAINS Deployed Successfully!"
echo "═══════════════════════════════════════════"
echo ""
echo "  🌐 Frontend:  http://$DOMAIN"
echo "  🔧 Admin:     http://$DOMAIN/admin"
echo "  📡 API:       http://$DOMAIN/api/health"
echo ""
echo "  📋 Commands:"
echo "  sudo systemctl status sains-api   # Check API status"
echo "  sudo journalctl -u sains-api -f   # View API logs"
echo "  sudo systemctl restart sains-api  # Restart API"
echo ""
