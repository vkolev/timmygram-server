CREATE TABLE IF NOT EXISTS device_watched_videos (
    device_id TEXT NOT NULL,
    video_id INTEGER NOT NULL,
    watched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (device_id, video_id)
);

CREATE INDEX IF NOT EXISTS idx_device_watched_videos_device_video
    ON device_watched_videos (device_id, video_id);

CREATE INDEX IF NOT EXISTS idx_device_watched_videos_device_watched_at
    ON device_watched_videos (device_id, watched_at);

CREATE TABLE IF NOT EXISTS device_video_queue (
    device_id TEXT NOT NULL,
    video_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (device_id, video_id)
);

CREATE INDEX IF NOT EXISTS idx_device_video_queue_device_position
    ON device_video_queue (device_id, position);

DROP TABLE IF EXISTS user_video_queue;
DROP TABLE IF EXISTS user_watched_videos;