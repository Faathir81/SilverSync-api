package repository

import (
	"silversync-api/internal/models"

	"gorm.io/gorm"
)

type TrackRepository interface {
	Save(track *models.Track) error
	UpdateFavorite(id uint, isFavorite bool) error
	FindBySpotifyID(spotifyID string) (*models.Track, error)
	FindByID(id uint) (*models.Track, error)
	FindAll(query string, sort string, limit int, offset int) ([]models.Track, int64, error)
	Update(track *models.Track) error
	Delete(id uint) error
}

type trackRepository struct {
	db *gorm.DB
}

func NewTrackRepository(db *gorm.DB) TrackRepository {
	return &trackRepository{db: db}
}

func (r *trackRepository) Save(track *models.Track) error {
	return r.db.Save(track).Error
}

func (r *trackRepository) UpdateFavorite(id uint, isFavorite bool) error {
	return r.db.Model(&models.Track{}).Where("id = ?", id).Update("is_favorite", isFavorite).Error
}

func (r *trackRepository) FindBySpotifyID(spotifyID string) (*models.Track, error) {
	var track models.Track
	err := r.db.Where("spotify_id = ?", spotifyID).First(&track).Error
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (r *trackRepository) FindAll(query string, sort string, limit int, offset int) ([]models.Track, int64, error) {
	var tracks []models.Track
	var count int64

	db := r.db.Model(&models.Track{})

	if query != "" {
		search := "%" + query + "%"
		db = db.Where("title ILIKE ? OR artist ILIKE ?", search, search)
	}

	err := db.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	// Default sort if empty
	if sort == "" {
		sort = "created_at desc"
	}

	err = db.Order(sort).Limit(limit).Offset(offset).Find(&tracks).Error
	return tracks, count, err
}

func (r *trackRepository) Update(track *models.Track) error {
	return r.db.Save(track).Error
}

func (r *trackRepository) Delete(id uint) error {
	return r.db.Delete(&models.Track{}, id).Error
}

func (r *trackRepository) FindByID(id uint) (*models.Track, error) {
	var track models.Track
	err := r.db.First(&track, id).Error
	return &track, err
}
