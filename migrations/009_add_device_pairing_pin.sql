CREATE TABLE IF NOT EXISTS device_pairing_pins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    pin TEXT NOT NULL UNIQUE,
    token TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_pairing_pins_user_id
    ON device_pairing_pins(user_id);

CREATE INDEX IF NOT EXISTS idx_device_pairing_pins_expires_at
    ON device_pairing_pins(expires_at);