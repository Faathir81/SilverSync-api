package repository

import (
	"silversync-api/internal/models"

	"gorm.io/gorm"
)

type WatchedPlaylistRepository interface {
	Create(wp *models.WatchedPlaylist) error
	FindAll() ([]models.WatchedPlaylist, error)
	Delete(id uint) error
}

type watchedPlaylistRepository struct {
	db *gorm.DB
}

func NewWatchedPlaylistRepository(db *gorm.DB) WatchedPlaylistRepository {
	return &watchedPlaylistRepository{db: db}
}

func (r *watchedPlaylistRepository) Create(wp *models.WatchedPlaylist) error {
	return r.db.Create(wp).Error
}

func (r *watchedPlaylistRepository) FindAll() ([]models.WatchedPlaylist, error) {
	var playlists []models.WatchedPlaylist
	err := r.db.Find(&playlists).Error
	return playlists, err
}

func (r *watchedPlaylistRepository) Delete(id uint) error {
	return r.db.Delete(&models.WatchedPlaylist{}, id).Error
}
