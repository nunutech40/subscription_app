-- Guest OTP verification table
CREATE TABLE IF NOT EXISTS guest_otps (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    guest_code_id UUID NOT NULL REFERENCES guest_codes(id),
    otp_code      TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    verified      BOOLEAN NOT NULL DEFAULT FALSE,
    ip            INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_guest_otps_email_code ON guest_otps(email, guest_code_id);
CREATE INDEX idx_guest_otps_expires ON guest_otps(expires_at);
