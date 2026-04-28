package repository

import (
	"database/sql"
	"errors"
	"time"
	"timmygram/internal/model"
)

type VideoRepository interface {
	Create(v *model.Video) (int64, error)
	FindByID(id int) (*model.Video, error)
	FindByUserID(userID int) ([]*model.Video, error)
	FindRandomByUserID(userID, limit int) ([]*model.Video, error)
	UpdateTranscoded(id, duration int, aspectRatio, thumbnail string) error
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
		`SELECT id, user_id, title, filename, duration, aspect_ratio, output_ratio,
		        COALESCE(thumbnail, ''), is_public, created_at, transcoded_at
		 FROM videos WHERE id = ?`, id,
	)
	return scanVideo(row)
}

func (r *SQLiteVideoRepository) FindByUserID(userID int) ([]*model.Video, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, title, filename, duration, aspect_ratio, output_ratio,
		        COALESCE(thumbnail, ''), is_public, created_at, transcoded_at
		 FROM videos WHERE user_id = ? ORDER BY created_at DESC`, userID,
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
		`SELECT id, user_id, title, filename, duration, aspect_ratio, output_ratio,
		        COALESCE(thumbnail, ''), is_public, created_at, transcoded_at
		 FROM videos WHERE user_id = ? AND transcoded_at IS NOT NULL
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

func (r *SQLiteVideoRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM videos WHERE id = ?`, id)
	return err
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
