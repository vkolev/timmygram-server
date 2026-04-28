package repository

import (
	"database/sql"
	"errors"
	"time"
	"timmygram/internal/model"
)

type DeviceRepository interface {
	Upsert(d *model.Device) error
	FindByUserID(userID int) ([]*model.Device, error)
}

type SQLiteDeviceRepository struct {
	db *sql.DB
}

func NewDeviceRepository(db *sql.DB) DeviceRepository {
	return &SQLiteDeviceRepository{db: db}
}

func (r *SQLiteDeviceRepository) Upsert(d *model.Device) error {
	var existingID int
	err := r.db.QueryRow(
		`SELECT id FROM devices WHERE user_id = ? AND device_id = ?`,
		d.UserID, d.DeviceID,
	).Scan(&existingID)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = r.db.Exec(
			`INSERT INTO devices (user_id, device_id, device_name, last_seen_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			d.UserID, d.DeviceID, d.DeviceName,
		)
		return err
	}
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE devices SET device_name = ?, device_os = ?, device_description = ?, last_seen_at = CURRENT_TIMESTAMP WHERE id = ?`,
		d.DeviceName, d.DeviceOs, d.DeviceDescription, existingID,
	)
	return err
}

func (r *SQLiteDeviceRepository) FindByUserID(userID int) ([]*model.Device, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, device_id, device_name, device_os, device_description, created_at, last_seen_at
		 FROM devices WHERE user_id = ?
		 ORDER BY last_seen_at DESC, created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func scanDevice(s scanner) (*model.Device, error) {
	var d model.Device
	var createdAtStr string
	var lastSeenAt sql.NullTime

	err := s.Scan(&d.ID, &d.UserID, &d.DeviceID, &d.DeviceName, &d.DeviceOs, &d.DeviceDescription, &createdAtStr, &lastSeenAt)
	if err != nil {
		return nil, err
	}

	t, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		d.CreatedAt = t
	}

	if lastSeenAt.Valid {
		d.LastSeenAt = &lastSeenAt.Time
	}

	return &d, nil
}
