package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostdev-ops/pma-backend-go/internal/core/screensaver"
	"github.com/jmoiron/sqlx"
)

// ScreensaverRepository implements the screensaver repository interface
type ScreensaverRepository struct {
	db *sqlx.DB
}

// NewScreensaverRepository creates a new screensaver repository
func NewScreensaverRepository(db *sqlx.DB) *ScreensaverRepository {
	return &ScreensaverRepository{db: db}
}

// CreateImage creates a new screensaver image record
func (r *ScreensaverRepository) CreateImage(ctx context.Context, image *screensaver.ScreensaverImage) error {
	query := `
		INSERT INTO screensaver_images (
			filename, original_name, content_type, file_size, width, height,
			checksum, uploaded_at, uploaded_by, tags, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		image.Filename, image.OriginalName, image.ContentType, image.FileSize,
		image.Width, image.Height, image.Checksum, image.UploadedAt,
		image.UploadedBy, image.Tags, image.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create screensaver image: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted ID: %w", err)
	}

	image.ID = int(id)
	return nil
}

// GetImageByID retrieves a screensaver image by ID
func (r *ScreensaverRepository) GetImageByID(ctx context.Context, id int) (*screensaver.ScreensaverImage, error) {
	query := `
		SELECT id, filename, original_name, content_type, file_size, width, height,
			   checksum, uploaded_at, uploaded_by, tags, active
		FROM screensaver_images WHERE id = ?
	`

	var image screensaver.ScreensaverImage
	err := r.db.GetContext(ctx, &image, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("screensaver image not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get screensaver image: %w", err)
	}

	return &image, nil
}

// GetImageByFilename retrieves a screensaver image by filename
func (r *ScreensaverRepository) GetImageByFilename(ctx context.Context, filename string) (*screensaver.ScreensaverImage, error) {
	query := `
		SELECT id, filename, original_name, content_type, file_size, width, height,
			   checksum, uploaded_at, uploaded_by, tags, active
		FROM screensaver_images WHERE filename = ?
	`

	var image screensaver.ScreensaverImage
	err := r.db.GetContext(ctx, &image, query, filename)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("screensaver image not found: %s", filename)
		}
		return nil, fmt.Errorf("failed to get screensaver image: %w", err)
	}

	return &image, nil
}

// GetImageByChecksum retrieves a screensaver image by checksum
func (r *ScreensaverRepository) GetImageByChecksum(ctx context.Context, checksum string) (*screensaver.ScreensaverImage, error) {
	query := `
		SELECT id, filename, original_name, content_type, file_size, width, height,
			   checksum, uploaded_at, uploaded_by, tags, active
		FROM screensaver_images WHERE checksum = ?
	`

	var image screensaver.ScreensaverImage
	err := r.db.GetContext(ctx, &image, query, checksum)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("screensaver image not found: %s", checksum)
		}
		return nil, fmt.Errorf("failed to get screensaver image: %w", err)
	}

	return &image, nil
}

// GetAllImages retrieves all screensaver images
func (r *ScreensaverRepository) GetAllImages(ctx context.Context) ([]screensaver.ScreensaverImage, error) {
	query := `
		SELECT id, filename, original_name, content_type, file_size, width, height,
			   checksum, uploaded_at, uploaded_by, tags, active
		FROM screensaver_images ORDER BY uploaded_at DESC
	`

	var images []screensaver.ScreensaverImage
	err := r.db.SelectContext(ctx, &images, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get screensaver images: %w", err)
	}

	return images, nil
}

// GetActiveImages retrieves all active screensaver images
func (r *ScreensaverRepository) GetActiveImages(ctx context.Context) ([]screensaver.ScreensaverImage, error) {
	query := `
		SELECT id, filename, original_name, content_type, file_size, width, height,
			   checksum, uploaded_at, uploaded_by, tags, active
		FROM screensaver_images WHERE active = 1 ORDER BY uploaded_at DESC
	`

	var images []screensaver.ScreensaverImage
	err := r.db.SelectContext(ctx, &images, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active screensaver images: %w", err)
	}

	return images, nil
}

// UpdateImage updates a screensaver image record
func (r *ScreensaverRepository) UpdateImage(ctx context.Context, image *screensaver.ScreensaverImage) error {
	query := `
		UPDATE screensaver_images SET
			filename = ?, original_name = ?, content_type = ?, file_size = ?,
			width = ?, height = ?, checksum = ?, uploaded_by = ?, tags = ?, active = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		image.Filename, image.OriginalName, image.ContentType, image.FileSize,
		image.Width, image.Height, image.Checksum, image.UploadedBy,
		image.Tags, image.Active, image.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update screensaver image: %w", err)
	}

	return nil
}

// DeleteImage deletes a screensaver image by ID
func (r *ScreensaverRepository) DeleteImage(ctx context.Context, id int) error {
	query := `DELETE FROM screensaver_images WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete screensaver image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("screensaver image not found: %d", id)
	}

	return nil
}

// DeleteImageByFilename deletes a screensaver image by filename
func (r *ScreensaverRepository) DeleteImageByFilename(ctx context.Context, filename string) error {
	query := `DELETE FROM screensaver_images WHERE filename = ?`

	result, err := r.db.ExecContext(ctx, query, filename)
	if err != nil {
		return fmt.Errorf("failed to delete screensaver image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("screensaver image not found: %s", filename)
	}

	return nil
}

// GetImagesByIDs retrieves multiple screensaver images by IDs
func (r *ScreensaverRepository) GetImagesByIDs(ctx context.Context, ids []int) ([]screensaver.ScreensaverImage, error) {
	if len(ids) == 0 {
		return []screensaver.ScreensaverImage{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, filename, original_name, content_type, file_size, width, height,
			   checksum, uploaded_at, uploaded_by, tags, active
		FROM screensaver_images WHERE id IN (?) ORDER BY uploaded_at DESC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	query = r.db.Rebind(query)

	var images []screensaver.ScreensaverImage
	err = r.db.SelectContext(ctx, &images, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get screensaver images: %w", err)
	}

	return images, nil
}

// DeleteImagesByIDs deletes multiple screensaver images by IDs
func (r *ScreensaverRepository) DeleteImagesByIDs(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM screensaver_images WHERE id IN (?)`, ids)
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	query = r.db.Rebind(query)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete screensaver images: %w", err)
	}

	return nil
}

// SetImagesActive sets the active status of multiple images
func (r *ScreensaverRepository) SetImagesActive(ctx context.Context, ids []int, active bool) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`UPDATE screensaver_images SET active = ? WHERE id IN (?)`, active, ids)
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	query = r.db.Rebind(query)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update screensaver images: %w", err)
	}

	return nil
}

// GetTotalImageCount returns the total number of images
func (r *ScreensaverRepository) GetTotalImageCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM screensaver_images`

	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get total image count: %w", err)
	}

	return count, nil
}

// GetActiveImageCount returns the number of active images
func (r *ScreensaverRepository) GetActiveImageCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM screensaver_images WHERE active = 1`

	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get active image count: %w", err)
	}

	return count, nil
}

// GetTotalImageSize returns the total size of all images
func (r *ScreensaverRepository) GetTotalImageSize(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(SUM(file_size), 0) FROM screensaver_images`

	var size int64
	err := r.db.GetContext(ctx, &size, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get total image size: %w", err)
	}

	return size, nil
}

// GetActiveImageSize returns the total size of active images
func (r *ScreensaverRepository) GetActiveImageSize(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(SUM(file_size), 0) FROM screensaver_images WHERE active = 1`

	var size int64
	err := r.db.GetContext(ctx, &size, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get active image size: %w", err)
	}

	return size, nil
}

// GetImageCountByContentType returns the number of images by content type
func (r *ScreensaverRepository) GetImageCountByContentType(ctx context.Context, contentType string) (int, error) {
	query := `SELECT COUNT(*) FROM screensaver_images WHERE content_type = ?`

	var count int
	err := r.db.GetContext(ctx, &count, query, contentType)
	if err != nil {
		return 0, fmt.Errorf("failed to get image count by content type: %w", err)
	}

	return count, nil
}

// DeleteInactiveImages deletes all inactive images
func (r *ScreensaverRepository) DeleteInactiveImages(ctx context.Context) error {
	query := `DELETE FROM screensaver_images WHERE active = 0`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete inactive images: %w", err)
	}

	return nil
}

// GetOrphanedImages finds images in the database that don't exist on disk
func (r *ScreensaverRepository) GetOrphanedImages(ctx context.Context, imageDirectory string) ([]screensaver.ScreensaverImage, error) {
	images, err := r.GetAllImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all images: %w", err)
	}

	var orphaned []screensaver.ScreensaverImage
	for _, image := range images {
		fullPath := filepath.Join(imageDirectory, image.Filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			orphaned = append(orphaned, image)
		}
	}

	return orphaned, nil
}
