package repositories

import (
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"gorm.io/gorm"
)

// AISelectionLogRepository defines the interface for AI selection log operations
type AISelectionLogRepository interface {
	CreateLog(log *models.AISelectionLog) error
	GetLogByID(id uint) (*models.AISelectionLog, error)
	GetAllLogs() ([]models.AISelectionLog, error)
	// Add more query methods as needed for analysis
}

// GormAISelectionLogRepository implements AISelectionLogRepository using GORM
type GormAISelectionLogRepository struct {
	db *gorm.DB
}

// NewGormAISelectionLogRepository creates a new GormAISelectionLogRepository
func NewGormAISelectionLogRepository(db *gorm.DB) *GormAISelectionLogRepository {
	return &GormAISelectionLogRepository{db: db}
}

// CreateLog creates a new AI selection log entry
func (r *GormAISelectionLogRepository) CreateLog(log *models.AISelectionLog) error {
	return r.db.Create(log).Error
}

// GetLogByID retrieves an AI selection log entry by ID
func (r *GormAISelectionLogRepository) GetLogByID(id uint) (*models.AISelectionLog, error) {
	var log models.AISelectionLog
	if err := r.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetAllLogs retrieves all AI selection log entries
func (r *GormAISelectionLogRepository) GetAllLogs() ([]models.AISelectionLog, error) {
	var logs []models.AISelectionLog
	if err := r.db.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
