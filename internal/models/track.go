package models

import (
	"time"
)

type Track struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SpotifyID    string    `gorm:"uniqueIndex;not null" json:"spotify_id"`
	Title        string    `json:"title"`
	Artist       string    `json:"artist"`
	DriveFileID  string    `json:"drive_file_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
