package repository

import (
	"silversync-api/internal/models"

	"gorm.io/gorm"
)

type PlaylistRepository interface {
	Create(playlist *models.Playlist) error
	FindAll() ([]models.Playlist, error)
	FindByID(id uint) (*models.Playlist, error)
	Delete(id uint) error
	AddTrack(playlistID uint, trackID uint) error
	RemoveTrack(playlistID uint, trackID uint) error
}

type playlistRepository struct {
	db *gorm.DB
}

func NewPlaylistRepository(db *gorm.DB) PlaylistRepository {
	return &playlistRepository{db: db}
}

func (r *playlistRepository) Create(playlist *models.Playlist) error {
	return r.db.Create(playlist).Error
}

func (r *playlistRepository) FindAll() ([]models.Playlist, error) {
	var playlists []models.Playlist
	err := r.db.Preload("Tracks").Find(&playlists).Error
	return playlists, err
}

func (r *playlistRepository) FindByID(id uint) (*models.Playlist, error) {
	var playlist models.Playlist
	err := r.db.Preload("Tracks").First(&playlist, id).Error
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

func (r *playlistRepository) Delete(id uint) error {
	return r.db.Delete(&models.Playlist{}, id).Error
}

func (r *playlistRepository) AddTrack(playlistID uint, trackID uint) error {
	playlist := models.Playlist{ID: playlistID}
	track := models.Track{ID: trackID}
	return r.db.Model(&playlist).Association("Tracks").Append(&track)
}

func (r *playlistRepository) RemoveTrack(playlistID uint, trackID uint) error {
	playlist := models.Playlist{ID: playlistID}
	track := models.Track{ID: trackID}
	return r.db.Model(&playlist).Association("Tracks").Delete(&track)
}
