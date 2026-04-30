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
	FindByID(id int) (*model.Device, error)
	FindByDeviceID(userID int, deviceID string) (*model.Device, error)
	Block(id, userID int) error
	Unblock(id, userID int) error
	Delete(id, userID int) error
	DeleteExpiredPairingPINs() error
	FindActivePairingPINByUserID(userID int) (*model.DevicePairingPIN, error)
	FindActivePairingPIN(pin string) (*model.DevicePairingPIN, error)
	CreatePairingPIN(pairingPIN *model.DevicePairingPIN) error
	DeletePairingPIN(id int) error
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
		`UPDATE devices
		 SET device_name = ?, device_os = ?, device_description = ?, last_seen_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND blocked_at IS NULL`,
		d.DeviceName, d.DeviceOs, d.DeviceDescription, existingID,
	)
	return err
}

func (r *SQLiteDeviceRepository) FindByUserID(userID int) ([]*model.Device, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, device_id, device_name, device_os, device_description, created_at, last_seen_at, blocked_at
				 FROM devices WHERE user_id = ?
				 ORDER BY blocked_at IS NOT NULL , last_seen_at DESC, created_at DESC`,
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

func (r *SQLiteDeviceRepository) FindByID(id int) (*model.Device, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, device_id, device_name, device_os, device_description, created_at, last_seen_at, blocked_at
		 FROM devices
		 WHERE id = ?`,
		id,
	)

	return scanDevice(row)
}

func (r *SQLiteDeviceRepository) FindByDeviceID(userID int, deviceID string) (*model.Device, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, device_id, device_name, device_os, device_description, created_at, last_seen_at, blocked_at
		 FROM devices
		 WHERE user_id = ? AND device_id = ?`,
		userID, deviceID,
	)

	return scanDevice(row)
}

func (r *SQLiteDeviceRepository) Block(id, userID int) error {
	result, err := r.db.Exec(
		`UPDATE devices SET blocked_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if err != nil {
		return err
	}

	return requireAffectedDevice(result)
}

func (r *SQLiteDeviceRepository) Unblock(id, userID int) error {
	result, err := r.db.Exec(
		`UPDATE devices SET blocked_at = NULL WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if err != nil {
		return err
	}

	return requireAffectedDevice(result)
}

func (r *SQLiteDeviceRepository) Delete(id, userID int) error {
	result, err := r.db.Exec(`DELETE FROM devices WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}

	return requireAffectedDevice(result)
}

func requireAffectedDevice(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return model.ErrDeviceNotFound
	}
	return nil
}

func scanDevice(s scanner) (*model.Device, error) {
	var d model.Device
	var createdAtStr string
	var lastSeenAt sql.NullTime
	var blockedAt sql.NullTime

	err := s.Scan(
		&d.ID,
		&d.UserID,
		&d.DeviceID,
		&d.DeviceName,
		&d.DeviceOs,
		&d.DeviceDescription,
		&createdAtStr,
		&lastSeenAt,
		&blockedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrDeviceNotFound
		}
		return nil, err
	}

	t, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		d.CreatedAt = t
	}

	if lastSeenAt.Valid {
		d.LastSeenAt = &lastSeenAt.Time
	}

	if blockedAt.Valid {
		d.BlockedAt = &blockedAt.Time
	}

	return &d, nil
}

func (r *SQLiteDeviceRepository) DeleteExpiredPairingPINs() error {
	_, err := r.db.Exec(`DELETE FROM device_pairing_pins WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}

func (r *SQLiteDeviceRepository) FindActivePairingPINByUserID(userID int) (*model.DevicePairingPIN, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, pin, token, expires_at, created_at
		 FROM device_pairing_pins
		 WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	)

	return scanDevicePairingPIN(row)
}

func (r *SQLiteDeviceRepository) FindActivePairingPIN(pin string) (*model.DevicePairingPIN, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, pin, token, expires_at, created_at
		 FROM device_pairing_pins
		 WHERE pin = ? AND expires_at > CURRENT_TIMESTAMP
		 LIMIT 1`,
		pin,
	)

	return scanDevicePairingPIN(row)
}

func (r *SQLiteDeviceRepository) CreatePairingPIN(pairingPIN *model.DevicePairingPIN) error {
	_, err := r.db.Exec(
		`INSERT INTO device_pairing_pins (user_id, pin, token, expires_at)
		 VALUES (?, ?, ?, ?)`,
		pairingPIN.UserID,
		pairingPIN.PIN,
		pairingPIN.Token,
		pairingPIN.ExpiresAt,
	)
	return err
}

func (r *SQLiteDeviceRepository) DeletePairingPIN(id int) error {
	_, err := r.db.Exec(`DELETE FROM device_pairing_pins WHERE id = ?`, id)
	return err
}

func scanDevicePairingPIN(s scanner) (*model.DevicePairingPIN, error) {
	var pairingPIN model.DevicePairingPIN
	var expiresAtStr string
	var createdAtStr string

	err := s.Scan(
		&pairingPIN.ID,
		&pairingPIN.UserID,
		&pairingPIN.PIN,
		&pairingPIN.Token,
		&expiresAtStr,
		&createdAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrDevicePairingPINNotFound
		}
		return nil, err
	}

	if expiresAt, err := parseSQLiteTime(expiresAtStr); err == nil {
		pairingPIN.ExpiresAt = expiresAt
	}

	if createdAt, err := parseSQLiteTime(createdAtStr); err == nil {
		pairingPIN.CreatedAt = createdAt
	}

	return &pairingPIN, nil
}

func parseSQLiteTime(value string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return t, nil
	}

	return time.Parse(time.RFC3339Nano, value)
}
