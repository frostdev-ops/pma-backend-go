package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// BackupManager defines the interface for backup and restore operations
type BackupManager interface {
	CreateBackup(options BackupOptions) (*Backup, error)
	RestoreBackup(backupID string, options RestoreOptions) error
	ListBackups() ([]*Backup, error)
	DeleteBackup(backupID string) error
	ScheduleBackup(schedule BackupSchedule) error
	ExportBackup(backupID string, writer io.Writer) error
	ImportBackup(reader io.Reader) (*Backup, error)
	GetBackupInfo(backupID string) (*Backup, error)
	ValidateBackup(backupID string) error
}

// BackupOptions contains options for creating backups
type BackupOptions struct {
	IncludeDatabase bool              `json:"include_database"`
	IncludeConfigs  bool              `json:"include_configs"`
	IncludeMedia    bool              `json:"include_media"`
	IncludeLogs     bool              `json:"include_logs"`
	IncludeFiles    bool              `json:"include_files"`
	Compress        bool              `json:"compress"`
	Encrypt         bool              `json:"encrypt"`
	Description     string            `json:"description"`
	Tags            []string          `json:"tags,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ExcludePaths    []string          `json:"exclude_paths,omitempty"`
}

// RestoreOptions contains options for restoring backups
type RestoreOptions struct {
	RestoreDatabase bool     `json:"restore_database"`
	RestoreConfigs  bool     `json:"restore_configs"`
	RestoreMedia    bool     `json:"restore_media"`
	RestoreLogs     bool     `json:"restore_logs"`
	RestoreFiles    bool     `json:"restore_files"`
	OverwriteFiles  bool     `json:"overwrite_files"`
	TargetPath      string   `json:"target_path,omitempty"`
	ExcludePaths    []string `json:"exclude_paths,omitempty"`
}

// Backup represents a backup record
type Backup struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Size        int64             `json:"size" db:"size"`
	Options     BackupOptions     `json:"options" db:"options"`
	Status      string            `json:"status" db:"status"`
	FilePath    string            `json:"file_path"`
	Checksum    string            `json:"checksum"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	CompletedAt *time.Time        `json:"completed_at" db:"completed_at"`
}

// BackupSchedule represents a scheduled backup
type BackupSchedule struct {
	ID             int           `json:"id" db:"id"`
	Name           string        `json:"name" db:"name"`
	CronExpression string        `json:"cron_expression" db:"cron_expression"`
	Options        BackupOptions `json:"options" db:"options"`
	Enabled        bool          `json:"enabled" db:"enabled"`
	LastRun        *time.Time    `json:"last_run" db:"last_run"`
	NextRun        *time.Time    `json:"next_run" db:"next_run"`
}

// BackupProgress represents backup operation progress
type BackupProgress struct {
	BackupID    string  `json:"backup_id"`
	Stage       string  `json:"stage"`
	Progress    float64 `json:"progress"`
	CurrentFile string  `json:"current_file,omitempty"`
	Message     string  `json:"message,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// LocalBackupManager implements BackupManager for local file system
type LocalBackupManager struct {
	config     *config.FileManagerConfig
	repos      *database.Repositories
	logger     *logrus.Logger
	cron       *cron.Cron
	encryptKey []byte
	db         *sql.DB
}

// Constants for backup status
const (
	BackupStatusPending    = "pending"
	BackupStatusInProgress = "in_progress"
	BackupStatusCompleted  = "completed"
	BackupStatusFailed     = "failed"
	BackupStatusCorrupted  = "corrupted"
)

// NewLocalBackupManager creates a new local backup manager
func NewLocalBackupManager(cfg *config.FileManagerConfig, repos *database.Repositories, db *sql.DB, logger *logrus.Logger, encryptKey []byte) *LocalBackupManager {
	// Create backup directory
	os.MkdirAll(cfg.Backup.BackupPath, 0755)

	return &LocalBackupManager{
		config:     cfg,
		repos:      repos,
		logger:     logger,
		cron:       cron.New(),
		encryptKey: encryptKey,
		db:         db,
	}
}

// CreateBackup implements BackupManager.CreateBackup
func (lbm *LocalBackupManager) CreateBackup(options BackupOptions) (*Backup, error) {
	backupID := uuid.New().String()

	backup := &Backup{
		ID:          backupID,
		Name:        fmt.Sprintf("backup_%s", time.Now().Format("20060102_150405")),
		Description: options.Description,
		Options:     options,
		Status:      BackupStatusPending,
		CreatedAt:   time.Now(),
		Metadata:    options.Metadata,
	}

	if backup.Metadata == nil {
		backup.Metadata = make(map[string]string)
	}
	backup.Metadata["version"] = "1.0"
	backup.Metadata["created_by"] = "pma-backend-go"

	// Store initial backup record
	if err := lbm.storeBackupRecord(backup); err != nil {
		return nil, fmt.Errorf("failed to store backup record: %w", err)
	}

	// Start backup process in background
	go lbm.performBackup(backup)

	lbm.logger.Infof("Started backup %s", backupID)
	return backup, nil
}

// performBackup performs the actual backup operation
func (lbm *LocalBackupManager) performBackup(backup *Backup) {
	backup.Status = BackupStatusInProgress
	lbm.updateBackupRecord(backup)

	// Create backup file path
	filename := fmt.Sprintf("%s.tar", backup.ID)
	if backup.Options.Compress {
		filename += ".gz"
	}
	if backup.Options.Encrypt {
		filename += ".enc"
	}

	backup.FilePath = filepath.Join(lbm.config.Backup.BackupPath, filename)

	// Create backup file
	backupFile, err := os.Create(backup.FilePath)
	if err != nil {
		lbm.markBackupFailed(backup, fmt.Errorf("failed to create backup file: %w", err))
		return
	}
	defer backupFile.Close()

	var writer io.Writer = backupFile

	// Add encryption if enabled
	if backup.Options.Encrypt {
		if len(lbm.encryptKey) == 0 {
			lbm.markBackupFailed(backup, fmt.Errorf("encryption enabled but no key provided"))
			return
		}

		encWriter, err := lbm.createEncryptedWriter(writer)
		if err != nil {
			lbm.markBackupFailed(backup, fmt.Errorf("failed to create encrypted writer: %w", err))
			return
		}
		writer = encWriter
	}

	// Add compression if enabled
	if backup.Options.Compress {
		if backup.Options.Encrypt {
			// Use zstd for encrypted files
			zstdWriter, err := zstd.NewWriter(writer)
			if err != nil {
				lbm.markBackupFailed(backup, fmt.Errorf("failed to create zstd writer: %w", err))
				return
			}
			defer zstdWriter.Close()
			writer = zstdWriter
		} else {
			// Use gzip for unencrypted files
			gzipWriter := gzip.NewWriter(writer)
			defer gzipWriter.Close()
			writer = gzipWriter
		}
	}

	// Create tar writer
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Backup components based on options
	if err := lbm.backupComponents(tarWriter, backup.Options); err != nil {
		lbm.markBackupFailed(backup, fmt.Errorf("backup failed: %w", err))
		return
	}

	// Get file size
	fileInfo, err := backupFile.Stat()
	if err == nil {
		backup.Size = fileInfo.Size()
	}

	// Mark backup as completed
	now := time.Now()
	backup.Status = BackupStatusCompleted
	backup.CompletedAt = &now

	if err := lbm.updateBackupRecord(backup); err != nil {
		lbm.logger.Errorf("Failed to update backup record: %v", err)
	}

	lbm.logger.Infof("Backup %s completed successfully, size: %d bytes", backup.ID, backup.Size)

	// Clean up old backups if needed
	lbm.cleanupOldBackups()
}

// backupComponents backs up different system components
func (lbm *LocalBackupManager) backupComponents(tarWriter *tar.Writer, options BackupOptions) error {
	if options.IncludeDatabase {
		if err := lbm.backupDatabase(tarWriter); err != nil {
			return fmt.Errorf("database backup failed: %w", err)
		}
	}

	if options.IncludeConfigs {
		if err := lbm.backupConfigs(tarWriter); err != nil {
			return fmt.Errorf("config backup failed: %w", err)
		}
	}

	if options.IncludeFiles {
		if err := lbm.backupFiles(tarWriter, options.ExcludePaths); err != nil {
			return fmt.Errorf("files backup failed: %w", err)
		}
	}

	if options.IncludeLogs {
		if err := lbm.backupLogs(tarWriter); err != nil {
			return fmt.Errorf("logs backup failed: %w", err)
		}
	}

	return nil
}

// backupDatabase backs up the database
func (lbm *LocalBackupManager) backupDatabase(tarWriter *tar.Writer) error {
	lbm.logger.Debug("Backing up database")

	// For SQLite, we can copy the database file
	// In production, you might want to use VACUUM INTO or similar
	dbPath := "./data/pma.db" // This should come from config

	return lbm.addFileToTar(tarWriter, dbPath, "database/pma.db")
}

// backupConfigs backs up configuration files
func (lbm *LocalBackupManager) backupConfigs(tarWriter *tar.Writer) error {
	lbm.logger.Debug("Backing up configs")

	configPaths := []string{
		"./configs/config.yaml",
		"./configs/config.toml",
		// Add other config files as needed
	}

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue // Skip non-existent files
		}

		relativePath := filepath.Join("configs", filepath.Base(configPath))
		if err := lbm.addFileToTar(tarWriter, configPath, relativePath); err != nil {
			return err
		}
	}

	return nil
}

// backupFiles backs up user files
func (lbm *LocalBackupManager) backupFiles(tarWriter *tar.Writer, excludePaths []string) error {
	lbm.logger.Debug("Backing up files")

	filesPath := lbm.config.Storage.BasePath
	return lbm.addDirectoryToTar(tarWriter, filesPath, "files", excludePaths)
}

// backupLogs backs up log files
func (lbm *LocalBackupManager) backupLogs(tarWriter *tar.Writer) error {
	lbm.logger.Debug("Backing up logs")

	logPaths := []string{
		"./logs",
		"./data/logs",
		"/var/log/pma", // System logs if accessible
	}

	for _, logPath := range logPaths {
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue // Skip non-existent directories
		}

		relativePath := filepath.Join("logs", filepath.Base(logPath))
		if err := lbm.addDirectoryToTar(tarWriter, logPath, relativePath, nil); err != nil {
			lbm.logger.Warnf("Failed to backup logs from %s: %v", logPath, err)
		}
	}

	return nil
}

// addFileToTar adds a single file to the tar archive
func (lbm *LocalBackupManager) addFileToTar(tarWriter *tar.Writer, filePath, tarPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    tarPath,
		Size:    fileInfo.Size(),
		Mode:    int64(fileInfo.Mode()),
		ModTime: fileInfo.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	return err
}

// addDirectoryToTar adds a directory recursively to the tar archive
func (lbm *LocalBackupManager) addDirectoryToTar(tarWriter *tar.Writer, dirPath, tarPath string, excludePaths []string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if path should be excluded
		for _, excludePath := range excludePaths {
			if matched, _ := filepath.Match(excludePath, path); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		fullTarPath := filepath.Join(tarPath, relativePath)

		if info.IsDir() {
			header := &tar.Header{
				Name:     fullTarPath + "/",
				Mode:     int64(info.Mode()),
				ModTime:  info.ModTime(),
				Typeflag: tar.TypeDir,
			}
			return tarWriter.WriteHeader(header)
		}

		return lbm.addFileToTar(tarWriter, path, fullTarPath)
	})
}

// RestoreBackup implements BackupManager.RestoreBackup
func (lbm *LocalBackupManager) RestoreBackup(backupID string, options RestoreOptions) error {
	backup, err := lbm.GetBackupInfo(backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup info: %w", err)
	}

	if backup.Status != BackupStatusCompleted {
		return fmt.Errorf("backup %s is not in completed status", backupID)
	}

	lbm.logger.Infof("Starting restore of backup %s", backupID)

	// Open backup file
	backupFile, err := os.Open(backup.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()

	var reader io.Reader = backupFile

	// Handle decryption if needed
	if backup.Options.Encrypt {
		decReader, err := lbm.createDecryptedReader(reader)
		if err != nil {
			return fmt.Errorf("failed to create decrypted reader: %w", err)
		}
		reader = decReader
	}

	// Handle decompression if needed
	if backup.Options.Compress {
		if backup.Options.Encrypt {
			// Use zstd for encrypted files
			zstdReader, err := zstd.NewReader(reader)
			if err != nil {
				return fmt.Errorf("failed to create zstd reader: %w", err)
			}
			defer zstdReader.Close()
			reader = zstdReader
		} else {
			// Use gzip for unencrypted files
			gzipReader, err := gzip.NewReader(reader)
			if err != nil {
				return fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzipReader.Close()
			reader = gzipReader
		}
	}

	// Create tar reader
	tarReader := tar.NewReader(reader)

	// Restore components
	return lbm.restoreComponents(tarReader, options)
}

// restoreComponents restores different system components
func (lbm *LocalBackupManager) restoreComponents(tarReader *tar.Reader, options RestoreOptions) error {
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Determine component type and restore accordingly
		if lbm.shouldRestoreFile(header.Name, options) {
			if err := lbm.restoreFile(tarReader, header, options); err != nil {
				return fmt.Errorf("failed to restore %s: %w", header.Name, err)
			}
		}
	}

	return nil
}

// shouldRestoreFile determines if a file should be restored based on options
func (lbm *LocalBackupManager) shouldRestoreFile(filename string, options RestoreOptions) bool {
	if strings.HasPrefix(filename, "database/") && !options.RestoreDatabase {
		return false
	}
	if strings.HasPrefix(filename, "configs/") && !options.RestoreConfigs {
		return false
	}
	if strings.HasPrefix(filename, "files/") && !options.RestoreFiles {
		return false
	}
	if strings.HasPrefix(filename, "logs/") && !options.RestoreLogs {
		return false
	}

	// Check excluded paths
	for _, excludePath := range options.ExcludePaths {
		if matched, _ := filepath.Match(excludePath, filename); matched {
			return false
		}
	}

	return true
}

// restoreFile restores a single file from the backup
func (lbm *LocalBackupManager) restoreFile(tarReader *tar.Reader, header *tar.Header, options RestoreOptions) error {
	targetPath := header.Name
	if options.TargetPath != "" {
		targetPath = filepath.Join(options.TargetPath, header.Name)
	}

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// Handle directories
	if header.Typeflag == tar.TypeDir {
		return os.MkdirAll(targetPath, os.FileMode(header.Mode))
	}

	// Check if file exists and if we should overwrite
	if _, err := os.Stat(targetPath); err == nil && !options.OverwriteFiles {
		lbm.logger.Warnf("Skipping existing file: %s", targetPath)
		return nil
	}

	// Create and write file
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, tarReader); err != nil {
		return err
	}

	// Set file permissions and modification time
	if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
		lbm.logger.Warnf("Failed to set permissions for %s: %v", targetPath, err)
	}

	if err := os.Chtimes(targetPath, time.Now(), header.ModTime); err != nil {
		lbm.logger.Warnf("Failed to set modification time for %s: %v", targetPath, err)
	}

	return nil
}

// ListBackups implements BackupManager.ListBackups
func (lbm *LocalBackupManager) ListBackups() ([]*Backup, error) {
	query := `
		SELECT id, name, description, size, options, status, created_at, completed_at
		FROM backups
		ORDER BY created_at DESC
	`

	rows, err := lbm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []*Backup
	for rows.Next() {
		var backup Backup
		var optionsJSON string
		var completedAt sql.NullTime

		err := rows.Scan(
			&backup.ID, &backup.Name, &backup.Description, &backup.Size,
			&optionsJSON, &backup.Status, &backup.CreatedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			backup.CompletedAt = &completedAt.Time
		}

		if err := json.Unmarshal([]byte(optionsJSON), &backup.Options); err != nil {
			lbm.logger.Warnf("Failed to unmarshal options for backup %s: %v", backup.ID, err)
		}

		backups = append(backups, &backup)
	}

	return backups, nil
}

// DeleteBackup implements BackupManager.DeleteBackup
func (lbm *LocalBackupManager) DeleteBackup(backupID string) error {
	backup, err := lbm.GetBackupInfo(backupID)
	if err != nil {
		return err
	}

	// Remove backup file
	if backup.FilePath != "" {
		if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
			lbm.logger.Warnf("Failed to remove backup file %s: %v", backup.FilePath, err)
		}
	}

	// Remove from database
	_, err = lbm.db.Exec("DELETE FROM backups WHERE id = ?", backupID)
	return err
}

// GetBackupInfo implements BackupManager.GetBackupInfo
func (lbm *LocalBackupManager) GetBackupInfo(backupID string) (*Backup, error) {
	query := `
		SELECT id, name, description, size, options, status, created_at, completed_at
		FROM backups
		WHERE id = ?
	`

	var backup Backup
	var optionsJSON string
	var completedAt sql.NullTime

	err := lbm.db.QueryRow(query, backupID).Scan(
		&backup.ID, &backup.Name, &backup.Description, &backup.Size,
		&optionsJSON, &backup.Status, &backup.CreatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		backup.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal([]byte(optionsJSON), &backup.Options); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	// Set file path
	filename := fmt.Sprintf("%s.tar", backup.ID)
	if backup.Options.Compress {
		filename += ".gz"
	}
	if backup.Options.Encrypt {
		filename += ".enc"
	}
	backup.FilePath = filepath.Join(lbm.config.Backup.BackupPath, filename)

	return &backup, nil
}

// ValidateBackup implements BackupManager.ValidateBackup
func (lbm *LocalBackupManager) ValidateBackup(backupID string) error {
	backup, err := lbm.GetBackupInfo(backupID)
	if err != nil {
		return err
	}

	// Check if backup file exists
	if _, err := os.Stat(backup.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backup.FilePath)
	}

	// TODO: Add checksum validation, archive integrity checks, etc.

	return nil
}

// ScheduleBackup implements BackupManager.ScheduleBackup
func (lbm *LocalBackupManager) ScheduleBackup(schedule BackupSchedule) error {
	// Store schedule in database
	if err := lbm.storeBackupSchedule(&schedule); err != nil {
		return fmt.Errorf("failed to store backup schedule: %w", err)
	}

	// Add to cron
	_, err := lbm.cron.AddFunc(schedule.CronExpression, func() {
		lbm.logger.Infof("Running scheduled backup: %s", schedule.Name)

		backup, err := lbm.CreateBackup(schedule.Options)
		if err != nil {
			lbm.logger.Errorf("Scheduled backup failed: %v", err)
			return
		}

		lbm.logger.Infof("Scheduled backup created: %s", backup.ID)

		// Update last run time
		now := time.Now()
		schedule.LastRun = &now
		lbm.updateBackupSchedule(&schedule)
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	lbm.cron.Start()
	return nil
}

// ExportBackup implements BackupManager.ExportBackup
func (lbm *LocalBackupManager) ExportBackup(backupID string, writer io.Writer) error {
	backup, err := lbm.GetBackupInfo(backupID)
	if err != nil {
		return err
	}

	file, err := os.Open(backup.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}

// ImportBackup implements BackupManager.ImportBackup
func (lbm *LocalBackupManager) ImportBackup(reader io.Reader) (*Backup, error) {
	backupID := uuid.New().String()

	// Create temporary file
	tempPath := filepath.Join(lbm.config.Backup.BackupPath, backupID+".tmp")
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Copy data
	size, err := io.Copy(tempFile, reader)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to copy backup data: %w", err)
	}

	// Create backup record
	backup := &Backup{
		ID:          backupID,
		Name:        fmt.Sprintf("imported_%s", time.Now().Format("20060102_150405")),
		Description: "Imported backup",
		Size:        size,
		Status:      BackupStatusCompleted,
		CreatedAt:   time.Now(),
		FilePath:    tempPath,
	}

	now := time.Now()
	backup.CompletedAt = &now

	// Store backup record
	if err := lbm.storeBackupRecord(backup); err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to store backup record: %w", err)
	}

	// Rename temp file to final name
	finalPath := filepath.Join(lbm.config.Backup.BackupPath, backupID+".tar")
	if err := os.Rename(tempPath, finalPath); err != nil {
		return nil, fmt.Errorf("failed to rename backup file: %w", err)
	}

	backup.FilePath = finalPath
	return backup, nil
}

// Helper methods for encryption
func (lbm *LocalBackupManager) createEncryptedWriter(writer io.Writer) (io.Writer, error) {
	block, err := aes.NewCipher(lbm.encryptKey)
	if err != nil {
		return nil, err
	}

	// Generate a random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Write IV to the beginning of the stream
	if _, err := writer.Write(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	return cipher.StreamWriter{S: stream, W: writer}, nil
}

func (lbm *LocalBackupManager) createDecryptedReader(reader io.Reader) (io.Reader, error) {
	block, err := aes.NewCipher(lbm.encryptKey)
	if err != nil {
		return nil, err
	}

	// Read IV from the beginning of the stream
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	return cipher.StreamReader{S: stream, R: reader}, nil
}

// Helper methods for database operations
func (lbm *LocalBackupManager) storeBackupRecord(backup *Backup) error {
	optionsJSON, err := json.Marshal(backup.Options)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO backups (id, name, description, size, options, status, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = lbm.db.Exec(query,
		backup.ID, backup.Name, backup.Description, backup.Size,
		string(optionsJSON), backup.Status, backup.CreatedAt, backup.CompletedAt,
	)

	return err
}

func (lbm *LocalBackupManager) updateBackupRecord(backup *Backup) error {
	optionsJSON, err := json.Marshal(backup.Options)
	if err != nil {
		return err
	}

	query := `
		UPDATE backups 
		SET name = ?, description = ?, size = ?, options = ?, status = ?, completed_at = ?
		WHERE id = ?
	`

	_, err = lbm.db.Exec(query,
		backup.Name, backup.Description, backup.Size,
		string(optionsJSON), backup.Status, backup.CompletedAt,
		backup.ID,
	)

	return err
}

func (lbm *LocalBackupManager) storeBackupSchedule(schedule *BackupSchedule) error {
	optionsJSON, err := json.Marshal(schedule.Options)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO backup_schedules (name, cron_expression, options, enabled, last_run, next_run)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = lbm.db.Exec(query,
		schedule.Name, schedule.CronExpression, string(optionsJSON),
		schedule.Enabled, schedule.LastRun, schedule.NextRun,
	)

	return err
}

func (lbm *LocalBackupManager) updateBackupSchedule(schedule *BackupSchedule) error {
	query := `
		UPDATE backup_schedules 
		SET last_run = ?, next_run = ?
		WHERE id = ?
	`

	_, updateErr := lbm.db.Exec(query, schedule.LastRun, schedule.NextRun, schedule.ID)
	return updateErr
}

func (lbm *LocalBackupManager) markBackupFailed(backup *Backup, backupErr error) {
	backup.Status = BackupStatusFailed
	lbm.logger.Errorf("Backup %s failed: %v", backup.ID, backupErr)
	if updateErr := lbm.updateBackupRecord(backup); updateErr != nil {
		lbm.logger.Errorf("Failed to update backup record after failure: %v", updateErr)
	}
}

func (lbm *LocalBackupManager) cleanupOldBackups() {
	if lbm.config.Backup.MaxBackups <= 0 {
		return
	}

	backups, err := lbm.ListBackups()
	if err != nil {
		lbm.logger.Errorf("Failed to list backups for cleanup: %v", err)
		return
	}

	if len(backups) <= lbm.config.Backup.MaxBackups {
		return
	}

	// Delete oldest backups
	for i := lbm.config.Backup.MaxBackups; i < len(backups); i++ {
		if err := lbm.DeleteBackup(backups[i].ID); err != nil {
			lbm.logger.Errorf("Failed to delete old backup %s: %v", backups[i].ID, err)
		} else {
			lbm.logger.Infof("Deleted old backup %s", backups[i].ID)
		}
	}
}
