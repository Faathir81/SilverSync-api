package repository

import (
	"silversync-api/internal/models"

	"gorm.io/gorm"
)

type SyncLogRepository interface {
	Create(log *models.SyncLog) error
	Update(log *models.SyncLog) error
	FindByID(id uint) (*models.SyncLog, error)
}

type syncLogRepository struct {
	db *gorm.DB
}

func NewSyncLogRepository(db *gorm.DB) SyncLogRepository {
	return &syncLogRepository{db: db}
}

func (r *syncLogRepository) Create(log *models.SyncLog) error {
	return r.db.Create(log).Error
}

func (r *syncLogRepository) Update(log *models.SyncLog) error {
	return r.db.Save(log).Error
}

func (r *syncLogRepository) FindByID(id uint) (*models.SyncLog, error) {
	var syncLog models.SyncLog
	err := r.db.First(&syncLog, id).Error
	if err != nil {
		return nil, err
	}
	return &syncLog, nil
}
