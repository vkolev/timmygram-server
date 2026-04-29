CREATE TABLE IF NOT EXISTS user_watched_videos (
    user_id INTEGER NOT NULL,
    video_id INTEGER NOT NULL,
    watched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, video_id)
);

CREATE INDEX IF NOT EXISTS idx_user_watched_videos ON user_watched_videos (user_id, video_id);