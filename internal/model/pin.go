package model

import (
	"errors"
	"time"
)

var ErrDevicePairingPINNotFound = errors.New("device pairing pin not found")

type DevicePairingPIN struct {
	ID        int
	UserID    int
	PIN       string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
