package model

import (
	"errors"
	"time"
)

var (
	ErrVideoNotFound     = errors.New("video not found")
	ErrVideoForbidden    = errors.New("forbidden")
	ErrVideoAlreadyLiked = errors.New("video already liked today")
)

type Video struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	Title        string     `json:"title"`
	Filename     string     `json:"filename"`
	Duration     int        `json:"duration"`
	AspectRatio  string     `json:"aspect_ratio"`
	OutputRatio  string     `json:"output_ratio"`
	Thumbnail    string     `json:"thumbnail"`
	IsPublic     bool       `json:"is_public"`
	LikesCount   int        `json:"likes_count"`
	CreatedAt    time.Time  `json:"created_at"`
	TranscodedAt *time.Time `json:"transcoded_at"`
}

func (v *Video) IsProcessing() bool {
	return v.TranscodedAt == nil
}
