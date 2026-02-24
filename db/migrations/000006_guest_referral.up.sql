-- Add referral_source to guest_logins for tracking where guests found the code
ALTER TABLE guest_logins ADD COLUMN referral_source TEXT DEFAULT '';
