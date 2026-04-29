package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"timmygram/internal/model"
)

type VideoRepository interface {
	Create(v *model.Video) (int64, error)
	FindByID(id int) (*model.Video, error)
	FindByUserID(userID int) ([]*model.Video, error)
	FindRandomByUserID(userID, limit int) ([]*model.Video, error)
	FindPageByUserID(userID, limit, offset int) ([]*model.Video, error)
	GetRandomUnwatchedVideo(userID int, deviceID string) (*model.Video, error)
	LikeVideo(videoID int, deviceID string) (int, error)
	CountLikes(videoID int) (int, error)
	UpdateTranscoded(id, duration int, aspectRatio, thumbnail string) error
	UpdateTitle(id int, title string) error
	Delete(id int) error
}

type SQLiteVideoRepository struct {
	db *sql.DB
}

func NewVideoRepository(db *sql.DB) VideoRepository {
	return &SQLiteVideoRepository{db: db}
}

func (r *SQLiteVideoRepository) Create(v *model.Video) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO videos (user_id, title, filename, duration, aspect_ratio, output_ratio, is_public)
		 VALUES (?, ?, ?, 0, '', ?, false)`,
		v.UserID, v.Title, v.Filename, v.OutputRatio,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *SQLiteVideoRepository) FindByID(id int) (*model.Video, error) {
	row := r.db.QueryRow(
		`SELECT v.id, v.user_id, v.title, v.filename, v.duration, v.aspect_ratio, v.output_ratio,
		        COALESCE(v.thumbnail, ''), v.is_public,
		        COUNT(vl.id) AS likes_count,
		        v.created_at, v.transcoded_at
		 FROM videos v
		 LEFT JOIN video_likes vl ON vl.video_id = v.id
		 WHERE v.id = ?
		 GROUP BY v.id`, id,
	)
	return scanVideo(row)
}

func (r *SQLiteVideoRepository) FindByUserID(userID int) ([]*model.Video, error) {
	rows, err := r.db.Query(
		`SELECT v.id, v.user_id, v.title, v.filename, v.duration, v.aspect_ratio, v.output_ratio,
		        COALESCE(v.thumbnail, ''), v.is_public,
		        COUNT(vl.id) AS likes_count,
		        v.created_at, v.transcoded_at
		 FROM videos v
		 LEFT JOIN video_likes vl ON vl.video_id = v.id
		 WHERE v.user_id = ?
		 GROUP BY v.id
		 ORDER BY v.created_at DESC`, userID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

func (r *SQLiteVideoRepository) FindPageByUserID(userID, limit, offset int) ([]*model.Video, error) {
	rows, err := r.db.Query(
		`SELECT v.id, v.user_id, v.title, v.filename, v.duration, v.aspect_ratio, v.output_ratio,
		        COALESCE(v.thumbnail, ''), v.is_public,
		        COUNT(vl.id) AS likes_count,
		        v.created_at, v.transcoded_at
		 FROM videos v
		 LEFT JOIN video_likes vl ON vl.video_id = v.id
		 WHERE v.user_id = ?
		 GROUP BY v.id
		 ORDER BY v.created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

func (r *SQLiteVideoRepository) FindRandomByUserID(userID, limit int) ([]*model.Video, error) {
	rows, err := r.db.Query(
		`SELECT v.id, v.user_id, v.title, v.filename, v.duration, v.aspect_ratio, v.output_ratio,
		        COALESCE(v.thumbnail, ''), v.is_public,
		        COUNT(vl.id) AS likes_count,
		        v.created_at, v.transcoded_at
		 FROM videos v
		 LEFT JOIN video_likes vl ON vl.video_id = v.id
		 WHERE v.user_id = ? AND v.transcoded_at IS NOT NULL
		 GROUP BY v.id
		 ORDER BY RANDOM() LIMIT ?`, userID, limit,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*model.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

func (r *SQLiteVideoRepository) UpdateTranscoded(id, duration int, aspectRatio, thumbnail string) error {
	_, err := r.db.Exec(
		`UPDATE videos SET duration = ?, aspect_ratio = ?, thumbnail = ?, transcoded_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		duration, aspectRatio, thumbnail, id,
	)
	return err
}

func (r *SQLiteVideoRepository) UpdateTitle(id int, title string) error {
	_, err := r.db.Exec(`UPDATE videos SET title = ? WHERE id = ?`, title, id)
	return err
}

func (r *SQLiteVideoRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM videos WHERE id = ?`, id)
	return err
}

func (r *SQLiteVideoRepository) GetRandomUnwatchedVideo(userID int, deviceID string) (*model.Video, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	video, err := r.popNextQueuedVideo(tx, deviceID)
	if err != nil {
		if !errors.Is(err, model.ErrVideoNotFound) {
			return nil, err
		}

		lastVideoID, _ := r.findLastWatchedVideoID(tx, deviceID)

		if err := r.refillVideoQueue(tx, userID, deviceID, lastVideoID); err != nil {
			return nil, err
		}

		video, err = r.popNextQueuedVideo(tx, deviceID)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(
		`INSERT OR REPLACE INTO device_watched_videos (device_id, video_id, watched_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)`,
		userID,
		video.ID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return video, nil
}

func (r *SQLiteVideoRepository) popNextQueuedVideo(tx *sql.Tx, deviceID string) (*model.Video, error) {
	var videoID int

	err := tx.QueryRow(
		`SELECT video_id
		 FROM device_video_queue
		 WHERE device_id = ?
		 ORDER BY position ASC
		 LIMIT 1`,
		deviceID,
	).Scan(&videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVideoNotFound
		}
		return nil, err
	}

	if _, err := tx.Exec(
		`DELETE FROM device_video_queue WHERE device_id = ? AND video_id = ?`,
		deviceID,
		videoID,
	); err != nil {
		return nil, err
	}

	return findVideoByIDInTx(tx, videoID)
}

func (r *SQLiteVideoRepository) refillVideoQueue(tx *sql.Tx, userID int, deviceID string, lastVideoID int) error {
	if _, err := tx.Exec(`DELETE FROM device_video_queue WHERE device_id = ?`, deviceID); err != nil {
		return err
	}

	rows, err := tx.Query(
		`SELECT id
		 FROM videos
		 WHERE user_id = ?
		   AND transcoded_at IS NOT NULL
		 ORDER BY RANDOM()`,
		userID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var videoIDs []int
	for rows.Next() {
		var videoID int
		if err := rows.Scan(&videoID); err != nil {
			return err
		}
		videoIDs = append(videoIDs, videoID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(videoIDs) == 0 {
		return model.ErrVideoNotFound
	}

	// Avoid showing the same video as the first item of the new cycle when possible.
	if len(videoIDs) > 1 && lastVideoID != 0 && videoIDs[0] == lastVideoID {
		videoIDs[0], videoIDs[1] = videoIDs[1], videoIDs[0]
	}

	stmt, err := tx.Prepare(
		`INSERT INTO device_video_queue (device_id, video_id, position)
		 VALUES (?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for position, videoID := range videoIDs {
		if _, err := stmt.Exec(deviceID, videoID, position); err != nil {
			return err
		}
	}

	return nil
}

func findVideoByIDInTx(tx *sql.Tx, videoID int) (*model.Video, error) {
	row := tx.QueryRow(
		`SELECT v.id, v.user_id, v.title, v.filename, v.duration, v.aspect_ratio, v.output_ratio,
		        COALESCE(v.thumbnail, ''), v.is_public,
		        COUNT(vl.id) AS likes_count,
		        v.created_at, v.transcoded_at
		 FROM videos v
		 LEFT JOIN video_likes vl ON vl.video_id = v.id
		 WHERE v.id = ?
		 GROUP BY v.id`,
		videoID,
	)

	return scanVideo(row)
}

func (r *SQLiteVideoRepository) findLastWatchedVideoID(tx *sql.Tx, deviceID string) (int, error) {
	var videoID int
	err := tx.QueryRow(
		`SELECT video_id
		 FROM device_watched_videos
		 WHERE device_id = ?
		 ORDER BY watched_at DESC
		 LIMIT 1`,
		deviceID,
	).Scan(&videoID)
	if err != nil {
		return 0, err
	}

	return videoID, nil
}

func (r *SQLiteVideoRepository) LikeVideo(videoID int, deviceID string) (int, error) {
	_, err := r.db.Exec(
		`INSERT INTO video_likes (video_id, device_id, liked_on)
		 VALUES (?, ?, CURRENT_DATE)`,
		videoID,
		deviceID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
			return r.CountLikes(videoID)
		}
		return 0, err
	}

	return r.CountLikes(videoID)
}

func (r *SQLiteVideoRepository) CountLikes(videoID int) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM video_likes WHERE video_id = ?`,
		videoID,
	).Scan(&count)
	return count, err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanVideo(s scanner) (*model.Video, error) {
	var v model.Video
	var transcodedAt sql.NullTime
	var createdAtStr string

	err := s.Scan(
		&v.ID, &v.UserID, &v.Title, &v.Filename, &v.Duration,
		&v.AspectRatio, &v.OutputRatio, &v.Thumbnail, &v.IsPublic,
		&v.LikesCount,
		&createdAtStr, &transcodedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVideoNotFound
		}
		return nil, err
	}

	// SQLite stores DATETIME as text; parse it.
	t, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		v.CreatedAt = t
	}

	if transcodedAt.Valid {
		v.TranscodedAt = &transcodedAt.Time
	}

	return &v, nil
}
