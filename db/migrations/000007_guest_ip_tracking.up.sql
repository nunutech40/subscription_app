-- Add IP address tracking to guest_logins for geolocation
ALTER TABLE guest_logins ADD COLUMN ip_address INET;
