package filemanager

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/klauspost/compress/zstd"
	"github.com/sirupsen/logrus"
)

// LocalStorage implements the FileManager interface using local filesystem
type LocalStorage struct {
	config          *config.FileManagerConfig
	db              *sql.DB
	logger          *logrus.Logger
	encoder         *zstd.Encoder
	decoder         *zstd.Decoder
	securityManager *SecurityManager
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(cfg *config.FileManagerConfig, db *sql.DB, logger *logrus.Logger, securityManager *SecurityManager) (*LocalStorage, error) {
	// Create necessary directories
	dirs := []string{
		cfg.Storage.BasePath,
		cfg.Storage.TempPath,
		cfg.Media.CachePath,
		cfg.Backup.BackupPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Initialize compression
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}

	return &LocalStorage{
		config:          cfg,
		db:              db,
		logger:          logger,
		encoder:         encoder,
		decoder:         decoder,
		securityManager: securityManager,
	}, nil
}

// Upload implements FileManager.Upload
func (ls *LocalStorage) Upload(filename string, content io.Reader, metadata FileMetadata) (*File, error) {
	fileID := uuid.New().String()

	// Check quota first
	if metadata.UploadedBy > 0 {
		if err := ls.CheckQuota(metadata.UploadedBy, 0); err != nil {
			return nil, fmt.Errorf("quota check failed: %w", err)
		}
	}

	// Create temporary file path
	tempPath := filepath.Join(ls.config.Storage.TempPath, fileID+".tmp")
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Calculate hash and size while copying
	hasher := sha256.New()
	var size int64

	// Use TeeReader to calculate hash while writing
	teeReader := io.TeeReader(content, hasher)
	size, err = io.Copy(tempFile, teeReader)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Check file size limits
	if size > ls.config.Storage.MaxFileSize {
		os.Remove(tempPath)
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", size, ls.config.Storage.MaxFileSize)
	}

	// Security validation
	if ls.securityManager != nil {
		securityConfig := SecurityConfig{
			EncryptionEnabled: false, // Can be made configurable
			VirusScanEnabled:  true,
			AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt", ".doc", ".docx", ".mp4", ".mp3"},
			BlockedExtensions: []string{".exe", ".bat", ".com", ".scr", ".pif", ".cmd"},
			MaxFileSize:       ls.config.Storage.MaxFileSize,
			ScanOnUpload:      true,
		}

		// Reset file pointer for security scanning
		tempFile.Seek(0, 0)
		if err := ls.securityManager.ValidateFileUpload(filename, tempFile, securityConfig); err != nil {
			os.Remove(tempPath)
			ls.logger.WithError(err).WithFields(logrus.Fields{
				"filename": filename,
				"size":     size,
			}).Warn("File upload blocked by security validation")
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
	}

	// Final quota check with actual size
	if metadata.UploadedBy > 0 {
		if err := ls.CheckQuota(metadata.UploadedBy, size); err != nil {
			os.Remove(tempPath)
			return nil, fmt.Errorf("quota exceeded: %w", err)
		}
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Detect file type
	tempFile.Seek(0, 0)
	head := make([]byte, 261)
	tempFile.Read(head)
	kind, err := filetype.Match(head)
	if err != nil {
		ls.logger.Warnf("Failed to detect file type for %s: %v", filename, err)
	}

	mimeType := "application/octet-stream"
	if kind != filetype.Unknown {
		mimeType = kind.MIME.Value
	}

	// Generate final file path based on category and date
	category := metadata.Category
	if category == "" {
		category = CategoryUpload
	}

	now := time.Now()
	relativePath := filepath.Join(
		category,
		now.Format("2006"),
		now.Format("01"),
		fileID+filepath.Ext(filename),
	)
	finalPath := filepath.Join(ls.config.Storage.BasePath, relativePath)

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Atomic move from temp to final location
	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to move file to final location: %w", err)
	}

	// Create file record
	file := &File{
		ID:         fileID,
		Name:       filename,
		Path:       relativePath,
		Size:       size,
		MimeType:   mimeType,
		Checksum:   checksum,
		Category:   category,
		Metadata:   metadata,
		CreatedAt:  now,
		ModifiedAt: now,
	}

	// Store in database
	if err := ls.storeFileRecord(file); err != nil {
		os.Remove(finalPath)
		return nil, fmt.Errorf("failed to store file record: %w", err)
	}

	ls.logger.Infof("Successfully uploaded file %s (ID: %s, Size: %d bytes)", filename, fileID, size)
	return file, nil
}

// Download implements FileManager.Download
func (ls *LocalStorage) Download(fileID string) (io.ReadCloser, error) {
	file, err := ls.GetFileInfo(fileID)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(ls.config.Storage.BasePath, file.Path)

	reader, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return reader, nil
}

// Delete implements FileManager.Delete
func (ls *LocalStorage) Delete(fileID string) error {
	file, err := ls.GetFileInfo(fileID)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(ls.config.Storage.BasePath, file.Path)

	// Remove file from filesystem
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Remove from database
	if err := ls.deleteFileRecord(fileID); err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	ls.logger.Infof("Successfully deleted file %s", fileID)
	return nil
}

// List implements FileManager.List
func (ls *LocalStorage) List(filter FileFilter) ([]*File, error) {
	return ls.listFiles(filter)
}

// GetMetadata implements FileManager.GetMetadata
func (ls *LocalStorage) GetMetadata(fileID string) (*FileMetadata, error) {
	file, err := ls.GetFileInfo(fileID)
	if err != nil {
		return nil, err
	}
	return &file.Metadata, nil
}

// UpdateMetadata implements FileManager.UpdateMetadata
func (ls *LocalStorage) UpdateMetadata(fileID string, metadata FileMetadata) error {
	return ls.updateFileMetadata(fileID, metadata)
}

// GetFileInfo implements FileManager.GetFileInfo
func (ls *LocalStorage) GetFileInfo(fileID string) (*File, error) {
	return ls.getFileRecord(fileID)
}

// CheckQuota implements FileManager.CheckQuota
func (ls *LocalStorage) CheckQuota(userID int, additionalSize int64) error {
	stats, err := ls.GetStorageStats()
	if err != nil {
		return err
	}

	if stats.UsedQuota+additionalSize > stats.TotalQuota {
		return fmt.Errorf("quota exceeded: %d + %d > %d", stats.UsedQuota, additionalSize, stats.TotalQuota)
	}

	return nil
}

// GetStorageStats implements FileManager.GetStorageStats
func (ls *LocalStorage) GetStorageStats() (*StorageStats, error) {
	return ls.calculateStorageStats()
}

// VerifyIntegrity checks file integrity using checksums
func (ls *LocalStorage) VerifyIntegrity(fileID string) error {
	file, err := ls.GetFileInfo(fileID)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(ls.config.Storage.BasePath, file.Path)

	f, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	currentChecksum := hex.EncodeToString(hasher.Sum(nil))
	if currentChecksum != file.Checksum {
		return fmt.Errorf("integrity check failed: expected %s, got %s", file.Checksum, currentChecksum)
	}

	return nil
}

// CompressFile compresses a file using zstd
func (ls *LocalStorage) CompressFile(fileID string) error {
	file, err := ls.GetFileInfo(fileID)
	if err != nil {
		return err
	}

	// Skip if already compressed
	if strings.HasSuffix(file.Path, ".zst") {
		return nil
	}

	fullPath := filepath.Join(ls.config.Storage.BasePath, file.Path)
	compressedPath := fullPath + ".zst"

	// Open source file
	source, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	// Create compressed file
	dest, err := os.Create(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer dest.Close()

	// Compress
	encoder, err := zstd.NewWriter(dest)
	if err != nil {
		return fmt.Errorf("failed to create encoder: %w", err)
	}
	defer encoder.Close()

	if _, err := io.Copy(encoder, source); err != nil {
		os.Remove(compressedPath)
		return fmt.Errorf("failed to compress file: %w", err)
	}

	// Update file record
	file.Path = file.Path + ".zst"
	if err := ls.updateFileRecord(file); err != nil {
		os.Remove(compressedPath)
		return fmt.Errorf("failed to update file record: %w", err)
	}

	// Remove original file
	os.Remove(fullPath)

	ls.logger.Infof("Successfully compressed file %s", fileID)
	return nil
}

// DecompressFile decompresses a zstd compressed file
func (ls *LocalStorage) DecompressFile(fileID string) error {
	file, err := ls.GetFileInfo(fileID)
	if err != nil {
		return err
	}

	// Skip if not compressed
	if !strings.HasSuffix(file.Path, ".zst") {
		return nil
	}

	fullPath := filepath.Join(ls.config.Storage.BasePath, file.Path)
	decompressedPath := strings.TrimSuffix(fullPath, ".zst")

	// Open compressed file
	source, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open compressed file: %w", err)
	}
	defer source.Close()

	// Create decompressed file
	dest, err := os.Create(decompressedPath)
	if err != nil {
		return fmt.Errorf("failed to create decompressed file: %w", err)
	}
	defer dest.Close()

	// Decompress
	decoder, err := zstd.NewReader(source)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}
	defer decoder.Close()

	if _, err := io.Copy(dest, decoder); err != nil {
		os.Remove(decompressedPath)
		return fmt.Errorf("failed to decompress file: %w", err)
	}

	// Update file record
	file.Path = strings.TrimSuffix(file.Path, ".zst")
	if err := ls.updateFileRecord(file); err != nil {
		os.Remove(decompressedPath)
		return fmt.Errorf("failed to update file record: %w", err)
	}

	// Remove compressed file
	os.Remove(fullPath)

	ls.logger.Infof("Successfully decompressed file %s", fileID)
	return nil
}

// Helper methods for database operations
func (ls *LocalStorage) storeFileRecord(file *File) error {
	metadataJSON, err := json.Marshal(file.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO files (id, name, path, size, mime_type, checksum, category, metadata, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = ls.db.Exec(query,
		file.ID, file.Name, file.Path, file.Size, file.MimeType,
		file.Checksum, file.Category, string(metadataJSON),
		file.CreatedAt, file.ModifiedAt,
	)

	return err
}

func (ls *LocalStorage) getFileRecord(fileID string) (*File, error) {
	query := `
		SELECT id, name, path, size, mime_type, checksum, category, metadata, created_at, modified_at
		FROM files WHERE id = ?
	`

	var file File
	var metadataJSON string

	err := ls.db.QueryRow(query, fileID).Scan(
		&file.ID, &file.Name, &file.Path, &file.Size, &file.MimeType,
		&file.Checksum, &file.Category, &metadataJSON,
		&file.CreatedAt, &file.ModifiedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(metadataJSON), &file.Metadata); err != nil {
		ls.logger.Warnf("Failed to unmarshal metadata for file %s: %v", fileID, err)
	}

	return &file, nil
}

func (ls *LocalStorage) deleteFileRecord(fileID string) error {
	_, err := ls.db.Exec("DELETE FROM files WHERE id = ?", fileID)
	return err
}

func (ls *LocalStorage) updateFileRecord(file *File) error {
	metadataJSON, err := json.Marshal(file.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE files 
		SET name = ?, path = ?, size = ?, mime_type = ?, checksum = ?, 
		    category = ?, metadata = ?, modified_at = ?
		WHERE id = ?
	`

	_, err = ls.db.Exec(query,
		file.Name, file.Path, file.Size, file.MimeType,
		file.Checksum, file.Category, string(metadataJSON),
		time.Now(), file.ID,
	)

	return err
}

func (ls *LocalStorage) updateFileMetadata(fileID string, metadata FileMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := "UPDATE files SET metadata = ?, modified_at = ? WHERE id = ?"
	_, err = ls.db.Exec(query, string(metadataJSON), time.Now(), fileID)
	return err
}

func (ls *LocalStorage) listFiles(filter FileFilter) ([]*File, error) {
	query := "SELECT id, name, path, size, mime_type, checksum, category, metadata, created_at, modified_at FROM files WHERE 1=1"
	args := []interface{}{}

	if filter.Category != "" {
		query += " AND category = ?"
		args = append(args, filter.Category)
	}

	if len(filter.MimeTypes) > 0 {
		placeholders := strings.Repeat("?,", len(filter.MimeTypes))
		placeholders = placeholders[:len(placeholders)-1]
		query += " AND mime_type IN (" + placeholders + ")"
		for _, mt := range filter.MimeTypes {
			args = append(args, mt)
		}
	}

	if !filter.StartDate.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		query += " AND created_at <= ?"
		args = append(args, filter.EndDate)
	}

	if filter.MinSize > 0 {
		query += " AND size >= ?"
		args = append(args, filter.MinSize)
	}

	if filter.MaxSize > 0 {
		query += " AND size <= ?"
		args = append(args, filter.MaxSize)
	}

	if filter.NameSearch != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+filter.NameSearch+"%")
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := ls.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var file File
		var metadataJSON string

		err := rows.Scan(
			&file.ID, &file.Name, &file.Path, &file.Size, &file.MimeType,
			&file.Checksum, &file.Category, &metadataJSON,
			&file.CreatedAt, &file.ModifiedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(metadataJSON), &file.Metadata); err != nil {
			ls.logger.Warnf("Failed to unmarshal metadata for file %s: %v", file.ID, err)
		}

		files = append(files, &file)
	}

	return files, nil
}

func (ls *LocalStorage) calculateStorageStats() (*StorageStats, error) {
	stats := &StorageStats{
		TotalQuota:      ls.config.Storage.TotalQuota,
		FilesByType:     make(map[string]int64),
		SizesByType:     make(map[string]int64),
		FilesByCategory: make(map[string]int64),
	}

	query := `
		SELECT 
			COUNT(*) as total_files,
			COALESCE(SUM(size), 0) as total_size,
			mime_type,
			category
		FROM files 
		GROUP BY mime_type, category
	`

	rows, err := ls.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var count, size int64
		var mimeType, category string

		if err := rows.Scan(&count, &size, &mimeType, &category); err != nil {
			return nil, err
		}

		stats.TotalFiles += count
		stats.TotalSize += size
		stats.FilesByType[mimeType] += count
		stats.SizesByType[mimeType] += size
		stats.FilesByCategory[category] += count
	}

	stats.UsedQuota = stats.TotalSize
	stats.AvailableSpace = stats.TotalQuota - stats.UsedQuota

	return stats, nil
}
