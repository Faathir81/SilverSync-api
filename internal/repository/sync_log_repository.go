package repository

import (
	"silversync-api/internal/models"

	"gorm.io/gorm"
)

type SyncLogRepository interface {
	Create(log *models.SyncLog) error
	Update(log *models.SyncLog) error
	FindByID(id uint) (*models.SyncLog, error)
	FindActive() ([]models.SyncLog, error)
	CleanupStaleJobs() error
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

func (r *syncLogRepository) FindActive() ([]models.SyncLog, error) {
	var logs []models.SyncLog
	// Filter by statuses that are not terminal
	err := r.db.Where("status NOT IN ?", []string{"SUCCESS", "FAILED"}).Find(&logs).Error
	return logs, err
}

func (r *syncLogRepository) CleanupStaleJobs() error {
	// Mark any non-terminal jobs as FAILED (Interrupted by server restart)
	return r.db.Model(&models.SyncLog{}).
		Where("status NOT IN ?", []string{"SUCCESS", "FAILED"}).
		Updates(map[string]interface{}{
			"status":  "FAILED",
			"message": "Task interrupted by server restart",
		}).Error
}
