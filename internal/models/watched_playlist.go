package models

import "time"

type WatchedPlaylist struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SpotifyID  string    `gorm:"uniqueIndex" json:"spotify_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
