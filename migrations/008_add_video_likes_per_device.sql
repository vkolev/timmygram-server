CREATE TABLE IF NOT EXISTS video_likes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    video_id INTEGER NOT NULL,
    device_id TEXT NOT NULL,
    liked_on DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
    UNIQUE (video_id, device_id, liked_on)
);

CREATE INDEX IF NOT EXISTS idx_video_likes_video_id
    ON video_likes(video_id);

CREATE INDEX IF NOT EXISTS idx_video_likes_device_id
    ON video_likes(device_id);