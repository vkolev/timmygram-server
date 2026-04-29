package model

import (
	"database/sql"
	"errors"
	"time"
)

type Device struct {
	ID                int
	UserID            int            `json:"user_id"`
	DeviceID          string         `json:"device_id"`
	DeviceName        string         `json:"device_name"`
	DeviceOs          sql.NullString `json:"device_os"`
	DeviceDescription sql.NullString `json:"device_description"`
	CreatedAt         time.Time      `json:"created_at"`
	LastSeenAt        *time.Time     `json:"last_seen_at"`
}

var ErrDeviceNotFound = errors.New("device not found")
