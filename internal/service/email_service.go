package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmailService handles sending emails via Resend API.
type EmailService struct {
	apiKey     string
	fromEmail  string
	httpClient *http.Client
}

// NewEmailService creates a new EmailService.
func NewEmailService(apiKey, fromEmail string) *EmailService {
	return &EmailService{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendEmailInput holds data for sending an email.
type SendEmailInput struct {
	To      string
	Subject string
	HTML    string
}

// SendEmail sends an email via Resend API.
func (s *EmailService) SendEmail(ctx context.Context, input SendEmailInput) error {
	if s.apiKey == "" {
		// Skip sending if no API key (dev mode)
		fmt.Printf("📧 [DEV] Email skipped (no API key): to=%s subject=%s\n", input.To, input.Subject)
		return nil
	}

	body := map[string]interface{}{
		"from":    s.fromEmail,
		"to":      []string{input.To},
		"subject": input.Subject,
		"html":    input.HTML,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ── Email Templates ─────────────────────────────────────────────────

// SendWelcomeEmail sends a welcome email after subscription activation.
func (s *EmailService) SendWelcomeEmail(ctx context.Context, to, name, productName, expiresAt string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 0; background: #f4f4f5; }
    .container { max-width: 560px; margin: 0 auto; padding: 40px 20px; }
    .card { background: #fff; border-radius: 12px; padding: 40px 32px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
    .logo { text-align: center; margin-bottom: 24px; }
    .logo span { font-size: 28px; font-weight: 700; background: linear-gradient(135deg, #6366f1, #8b5cf6); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    h1 { font-size: 22px; font-weight: 600; color: #18181b; margin: 0 0 16px; }
    p { font-size: 15px; line-height: 1.6; color: #3f3f46; margin: 0 0 12px; }
    .highlight { background: #f0fdf4; border-left: 3px solid #22c55e; padding: 16px; border-radius: 8px; margin: 20px 0; }
    .highlight p { margin: 0; color: #15803d; font-weight: 500; }
    .btn { display: inline-block; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; padding: 14px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 15px; margin: 20px 0; }
    .footer { text-align: center; padding: 24px 0; color: #a1a1aa; font-size: 13px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="card">
      <div class="logo"><span>⚛️ SAINS</span></div>
      <h1>Selamat datang, %s! 🎉</h1>
      <p>Subscription kamu untuk <strong>%s</strong> sudah aktif.</p>
      <div class="highlight">
        <p>✅ Akses aktif hingga: <strong>%s</strong></p>
      </div>
      <p>Kamu sekarang bisa mengakses semua fitur premium termasuk:</p>
      <p>• Model atom 3D interaktif (Bohr + Orbital)<br>
         • Anatomi elemen lengkap<br>
         • Molecule Builder<br>
         • Kimia Lab<br>
         • Modul belajar tanpa batas</p>
      <a href="https://sains.id" class="btn">Mulai Belajar →</a>
      <p style="color:#71717a; font-size:13px; margin-top:24px;">Butuh bantuan? Balas email ini atau hubungi support@sains.id</p>
    </div>
    <div class="footer">
      © 2026 SAINS — Interactive Science Learning<br>
      Email ini dikirim karena kamu berlangganan di sains.id
    </div>
  </div>
</body>
</html>`, name, productName, expiresAt)

	return s.SendEmail(ctx, SendEmailInput{
		To:      to,
		Subject: fmt.Sprintf("🎉 Selamat datang di %s!", productName),
		HTML:    html,
	})
}

// SendExpiryReminderEmail sends a subscription expiry reminder.
func (s *EmailService) SendExpiryReminderEmail(ctx context.Context, to, name, productName, expiresAt string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 0; background: #f4f4f5; }
    .container { max-width: 560px; margin: 0 auto; padding: 40px 20px; }
    .card { background: #fff; border-radius: 12px; padding: 40px 32px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
    .logo { text-align: center; margin-bottom: 24px; }
    .logo span { font-size: 28px; font-weight: 700; background: linear-gradient(135deg, #6366f1, #8b5cf6); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    h1 { font-size: 22px; font-weight: 600; color: #18181b; margin: 0 0 16px; }
    p { font-size: 15px; line-height: 1.6; color: #3f3f46; margin: 0 0 12px; }
    .warning { background: #fffbeb; border-left: 3px solid #f59e0b; padding: 16px; border-radius: 8px; margin: 20px 0; }
    .warning p { margin: 0; color: #92400e; font-weight: 500; }
    .btn { display: inline-block; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; padding: 14px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 15px; margin: 20px 0; }
    .footer { text-align: center; padding: 24px 0; color: #a1a1aa; font-size: 13px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="card">
      <div class="logo"><span>⚛️ SAINS</span></div>
      <h1>Hai %s, subscription-mu segera berakhir ⏰</h1>
      <div class="warning">
        <p>⚠️ Subscription %s akan berakhir pada <strong>%s</strong></p>
      </div>
      <p>Perpanjang sekarang agar tetap bisa mengakses semua fitur premium tanpa gangguan.</p>
      <a href="https://sains.id/pricing" class="btn">Perpanjang Sekarang →</a>
      <p style="color:#71717a; font-size:13px; margin-top:24px;">Jika kamu tidak ingin menerima email ini lagi, abaikan saja.</p>
    </div>
    <div class="footer">
      © 2026 SAINS — Interactive Science Learning
    </div>
  </div>
</body>
</html>`, name, productName, expiresAt)

	return s.SendEmail(ctx, SendEmailInput{
		To:      to,
		Subject: fmt.Sprintf("⏰ Subscription %s segera berakhir", productName),
		HTML:    html,
	})
}

// SendGuestOTPEmail sends a 6-digit OTP code for guest email verification.
func (s *EmailService) SendGuestOTPEmail(ctx context.Context, to, otpCode string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 0; background: #f4f4f5; }
    .container { max-width: 560px; margin: 0 auto; padding: 40px 20px; }
    .card { background: #fff; border-radius: 12px; padding: 40px 32px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
    .logo { text-align: center; margin-bottom: 24px; }
    .logo span { font-size: 28px; font-weight: 700; background: linear-gradient(135deg, #6366f1, #8b5cf6); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    h1 { font-size: 22px; font-weight: 600; color: #18181b; margin: 0 0 16px; text-align: center; }
    p { font-size: 15px; line-height: 1.6; color: #3f3f46; margin: 0 0 12px; text-align: center; }
    .otp-box { background: #f0f0ff; border: 2px solid #6366f1; border-radius: 12px; padding: 24px; margin: 24px 0; text-align: center; }
    .otp-code { font-size: 36px; font-weight: 700; letter-spacing: 8px; color: #4f46e5; font-family: 'Courier New', monospace; margin: 0; }
    .warning { color: #f59e0b; font-size: 13px; margin-top: 8px; }
    .footer { text-align: center; padding: 24px 0; color: #a1a1aa; font-size: 13px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="card">
      <div class="logo"><span>⚛️ Atomic</span></div>
      <h1>Verifikasi Email Kamu 🔐</h1>
      <p>Masukkan kode berikut untuk login sebagai guest di Atomic:</p>
      <div class="otp-box">
        <p class="otp-code">%s</p>
        <p class="warning">⏱ Kode berlaku 5 menit</p>
      </div>
      <p>Jika kamu tidak meminta kode ini, abaikan email ini.</p>
      <p style="color:#71717a; font-size:13px; margin-top:24px;">Kode ini hanya bisa dipakai 1 kali dan akan expired otomatis.</p>
    </div>
    <div class="footer">
      © 2026 SAINS — Interactive Science Learning
    </div>
  </div>
</body>
</html>`, otpCode)

	return s.SendEmail(ctx, SendEmailInput{
		To:      to,
		Subject: "🔐 Kode Verifikasi Atomic — " + otpCode,
		HTML:    html,
	})
}
