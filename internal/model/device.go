package model

import (
	"errors"
	"time"
)

type Device struct {
	ID                int
	UserID            int
	DeviceID          string
	DeviceName        string
	DeviceOs          string
	DeviceDescription string
	CreatedAt         time.Time
	LastSeenAt        *time.Time
}

var ErrDeviceNotFound = errors.New("device not found")
