package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/backup"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// BackupHandler handles backup-related API requests
type BackupHandler struct {
	backupManager backup.BackupManager
	logger        *logrus.Logger
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(backupManager backup.BackupManager, logger *logrus.Logger) *BackupHandler {
	return &BackupHandler{
		backupManager: backupManager,
		logger:        logger,
	}
}

// CreateBackup creates a new backup
func (bh *BackupHandler) CreateBackup(c *gin.Context) {
	var options backup.BackupOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid backup options")
		return
	}

	// Set default options if not provided
	if options.Compress == false && options.Encrypt == false {
		options.Compress = true // Default to compression
	}

	backup, err := bh.backupManager.CreateBackup(options)
	if err != nil {
		bh.logger.WithError(err).Error("Failed to create backup")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to create backup: %v", err))
		return
	}

	bh.logger.WithField("backup_id", backup.ID).Info("Backup created successfully")
	utils.SendSuccess(c, backup)
}

// RestoreBackup restores a backup
func (bh *BackupHandler) RestoreBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		utils.SendError(c, http.StatusBadRequest, "Backup ID is required")
		return
	}

	var options backup.RestoreOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid restore options")
		return
	}

	err := bh.backupManager.RestoreBackup(backupID, options)
	if err != nil {
		bh.logger.WithError(err).WithField("backup_id", backupID).Error("Failed to restore backup")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to restore backup: %v", err))
		return
	}

	bh.logger.WithField("backup_id", backupID).Info("Backup restored successfully")
	utils.SendSuccess(c, gin.H{
		"message":   "Backup restored successfully",
		"backup_id": backupID,
	})
}

// ListBackups lists all available backups
func (bh *BackupHandler) ListBackups(c *gin.Context) {
	backups, err := bh.backupManager.ListBackups()
	if err != nil {
		bh.logger.WithError(err).Error("Failed to list backups")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list backups: %v", err))
		return
	}

	utils.SendSuccess(c, gin.H{
		"backups": backups,
		"count":   len(backups),
	})
}

// GetBackup retrieves information about a specific backup
func (bh *BackupHandler) GetBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		utils.SendError(c, http.StatusBadRequest, "Backup ID is required")
		return
	}

	backup, err := bh.backupManager.GetBackupInfo(backupID)
	if err != nil {
		bh.logger.WithError(err).WithField("backup_id", backupID).Error("Failed to get backup info")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Backup not found: %v", err))
		return
	}

	utils.SendSuccess(c, backup)
}

// DeleteBackup deletes a backup
func (bh *BackupHandler) DeleteBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		utils.SendError(c, http.StatusBadRequest, "Backup ID is required")
		return
	}

	err := bh.backupManager.DeleteBackup(backupID)
	if err != nil {
		bh.logger.WithError(err).WithField("backup_id", backupID).Error("Failed to delete backup")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete backup: %v", err))
		return
	}

	bh.logger.WithField("backup_id", backupID).Info("Backup deleted successfully")
	utils.SendSuccess(c, gin.H{
		"message":   "Backup deleted successfully",
		"backup_id": backupID,
	})
}

// ValidateBackup validates the integrity of a backup
func (bh *BackupHandler) ValidateBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		utils.SendError(c, http.StatusBadRequest, "Backup ID is required")
		return
	}

	err := bh.backupManager.ValidateBackup(backupID)
	if err != nil {
		bh.logger.WithError(err).WithField("backup_id", backupID).Error("Backup validation failed")
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Backup validation failed: %v", err))
		return
	}

	bh.logger.WithField("backup_id", backupID).Info("Backup validation successful")
	utils.SendSuccess(c, gin.H{
		"message":   "Backup is valid",
		"backup_id": backupID,
		"status":    "valid",
	})
}

// ExportBackup exports a backup for download
func (bh *BackupHandler) ExportBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		utils.SendError(c, http.StatusBadRequest, "Backup ID is required")
		return
	}

	// Get backup info for filename
	backupInfo, err := bh.backupManager.GetBackupInfo(backupID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Backup not found")
		return
	}

	filename := fmt.Sprintf("%s.backup", backupInfo.Name)

	// Set headers for file download
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	err = bh.backupManager.ExportBackup(backupID, c.Writer)
	if err != nil {
		bh.logger.WithError(err).WithField("backup_id", backupID).Error("Failed to export backup")
		// Note: Can't send JSON error here since we've already started writing the response
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	bh.logger.WithField("backup_id", backupID).Info("Backup exported successfully")
}

// ImportBackup imports a backup from uploaded file
func (bh *BackupHandler) ImportBackup(c *gin.Context) {
	file, header, err := c.Request.FormFile("backup")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	bh.logger.WithField("filename", header.Filename).Info("Starting backup import")

	backup, err := bh.backupManager.ImportBackup(file)
	if err != nil {
		bh.logger.WithError(err).WithField("filename", header.Filename).Error("Failed to import backup")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to import backup: %v", err))
		return
	}

	bh.logger.WithField("backup_id", backup.ID).Info("Backup imported successfully")
	utils.SendSuccess(c, gin.H{
		"message":   "Backup imported successfully",
		"backup":    backup,
		"backup_id": backup.ID,
	})
}

// ScheduleBackup schedules automatic backups
func (bh *BackupHandler) ScheduleBackup(c *gin.Context) {
	var schedule backup.BackupSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid schedule configuration")
		return
	}

	err := bh.backupManager.ScheduleBackup(schedule)
	if err != nil {
		bh.logger.WithError(err).Error("Failed to schedule backup")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to schedule backup: %v", err))
		return
	}

	bh.logger.WithField("schedule", schedule.CronExpression).Info("Backup scheduled successfully")
	utils.SendSuccess(c, gin.H{
		"message":  "Backup scheduled successfully",
		"schedule": schedule,
	})
}

// GetBackupStatistics returns statistics about backups
func (bh *BackupHandler) GetBackupStatistics(c *gin.Context) {
	backups, err := bh.backupManager.ListBackups()
	if err != nil {
		bh.logger.WithError(err).Error("Failed to get backup statistics")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get backup statistics")
		return
	}

	var totalSize int64
	var completedCount int
	var failedCount int
	var encryptedCount int
	var compressedCount int

	for _, backup := range backups {
		totalSize += backup.Size
		switch backup.Status {
		case "completed":
			completedCount++
		case "failed":
			failedCount++
		}
		if backup.Options.Encrypt {
			encryptedCount++
		}
		if backup.Options.Compress {
			compressedCount++
		}
	}

	stats := gin.H{
		"total_backups":    len(backups),
		"completed_count":  completedCount,
		"failed_count":     failedCount,
		"encrypted_count":  encryptedCount,
		"compressed_count": compressedCount,
		"total_size":       totalSize,
		"total_size_mb":    float64(totalSize) / (1024 * 1024),
	}

	utils.SendSuccess(c, stats)
}

// CleanupOldBackups removes old backups based on retention policy
func (bh *BackupHandler) CleanupOldBackups(c *gin.Context) {
	// Get retention days from query parameter, default to 30
	retentionDaysStr := c.DefaultQuery("retention_days", "30")
	retentionDays, err := strconv.Atoi(retentionDaysStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid retention_days parameter")
		return
	}

	backups, err := bh.backupManager.ListBackups()
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to list backups")
		return
	}

	var deletedCount int
	var deletedSize int64
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	for _, backup := range backups {
		// Only delete completed backups older than retention period
		if backup.Status == "completed" && backup.CreatedAt.Before(cutoffDate) {
			if err := bh.backupManager.DeleteBackup(backup.ID); err != nil {
				bh.logger.WithError(err).WithField("backup_id", backup.ID).Warn("Failed to delete old backup")
				continue
			}
			deletedCount++
			deletedSize += backup.Size
			bh.logger.WithField("backup_id", backup.ID).Info("Deleted old backup")
		}
	}

	utils.SendSuccess(c, gin.H{
		"message":        "Cleanup completed",
		"deleted_count":  deletedCount,
		"deleted_size":   deletedSize,
		"retention_days": retentionDays,
	})
}
