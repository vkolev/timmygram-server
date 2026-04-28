ALTER TABLE devices ADD COLUMN last_seen_at DATETIME;
CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_user_device ON devices(user_id, device_id);
