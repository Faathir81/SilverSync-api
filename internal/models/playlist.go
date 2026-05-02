package models

import (
	"time"
)

type Playlist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"unique;not null" json:"name"`
	Tracks    []Track   `gorm:"many2many:playlist_tracks;" json:"tracks,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
