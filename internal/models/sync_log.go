package models

import (
	"time"
)

type SyncLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SpotifyURL string    `json:"spotify_url"`
	SyncType   string    `json:"sync_type"` // e.g., "MANUAL", "SMART"
	Status     string    `json:"status"` // e.g., "PENDING", "COMPLETED", "FAILED"
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
