CREATE TABLE IF NOT EXISTS user_video_queue (
    user_id INTEGER NOT NULL,
    video_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, video_id)
);

CREATE INDEX IF NOT EXISTS idx_user_video_queue_user_position
    ON user_video_queue (user_id, position);