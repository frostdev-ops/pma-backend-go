package filemanager

import (
	"io"
	"time"
)

// FileManager defines the interface for file management operations
type FileManager interface {
	Upload(filename string, content io.Reader, metadata FileMetadata) (*File, error)
	Download(fileID string) (io.ReadCloser, error)
	Delete(fileID string) error
	List(filter FileFilter) ([]*File, error)
	GetMetadata(fileID string) (*FileMetadata, error)
	UpdateMetadata(fileID string, metadata FileMetadata) error
	GetFileInfo(fileID string) (*File, error)
	CheckQuota(userID int, additionalSize int64) error
	GetStorageStats() (*StorageStats, error)
}

// File represents a file in the system
type File struct {
	ID         string       `json:"id" db:"id"`
	Name       string       `json:"name" db:"name"`
	Path       string       `json:"path" db:"path"`
	Size       int64        `json:"size" db:"size"`
	MimeType   string       `json:"mime_type" db:"mime_type"`
	Checksum   string       `json:"checksum" db:"checksum"`
	Category   string       `json:"category" db:"category"`
	Metadata   FileMetadata `json:"metadata" db:"metadata"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	ModifiedAt time.Time    `json:"modified_at" db:"modified_at"`
}

// FileMetadata contains additional information about a file
type FileMetadata struct {
	Category    string                 `json:"category,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	UploadedBy  int                    `json:"uploaded_by,omitempty"`
}

// FileFilter contains options for filtering file lists
type FileFilter struct {
	Category   string    `json:"category,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	MimeTypes  []string  `json:"mime_types,omitempty"`
	StartDate  time.Time `json:"start_date,omitempty"`
	EndDate    time.Time `json:"end_date,omitempty"`
	MinSize    int64     `json:"min_size,omitempty"`
	MaxSize    int64     `json:"max_size,omitempty"`
	UserID     int       `json:"user_id,omitempty"`
	NameSearch string    `json:"name_search,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

// StorageStats contains storage usage statistics
type StorageStats struct {
	TotalFiles      int64            `json:"total_files"`
	TotalSize       int64            `json:"total_size"`
	UsedQuota       int64            `json:"used_quota"`
	TotalQuota      int64            `json:"total_quota"`
	AvailableSpace  int64            `json:"available_space"`
	FilesByType     map[string]int64 `json:"files_by_type"`
	SizesByType     map[string]int64 `json:"sizes_by_type"`
	FilesByCategory map[string]int64 `json:"files_by_category"`
}

// FilePermission represents access control for files
type FilePermission struct {
	ID         int       `json:"id" db:"id"`
	FileID     string    `json:"file_id" db:"file_id"`
	UserID     *int      `json:"user_id" db:"user_id"`
	Permission string    `json:"permission" db:"permission"`
	GrantedAt  time.Time `json:"granted_at" db:"granted_at"`
}

// UploadResult contains the result of a file upload operation
type UploadResult struct {
	File     *File    `json:"file"`
	Success  bool     `json:"success"`
	Message  string   `json:"message,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// FileChunk represents a chunk of a large file
type FileChunk struct {
	ID       string `json:"id"`
	FileID   string `json:"file_id"`
	ChunkNum int    `json:"chunk_num"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Path     string `json:"path"`
}

// Constants for file categories
const (
	CategoryBackup = "backup"
	CategoryMedia  = "media"
	CategoryLog    = "log"
	CategoryConfig = "config"
	CategoryUpload = "upload"
	CategoryTemp   = "temp"
)

// Constants for file permissions
const (
	PermissionRead   = "read"
	PermissionWrite  = "write"
	PermissionDelete = "delete"
	PermissionAdmin  = "admin"
)
