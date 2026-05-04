package model

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	IsOwner  bool   `json:"is_owner"`
}
