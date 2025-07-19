package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// DisplaySettingsRepository implements repositories.DisplayRepository
type DisplaySettingsRepository struct {
	db *sql.DB
}

// NewDisplaySettingsRepository creates a new DisplaySettingsRepository
func NewDisplaySettingsRepository(db *sql.DB) repositories.DisplayRepository {
	return &DisplaySettingsRepository{db: db}
}

// GetSettings retrieves the current display settings
func (r *DisplaySettingsRepository) GetSettings(ctx context.Context) (*models.DisplaySettings, error) {
	query := `
		SELECT id, brightness, timeout, orientation, darkMode, screensaver, 
		       screensaverType, screensaverShowClock, screensaverRotationSpeed,
		       screensaverPictureFrameImage, screensaverUploadEnabled, 
		       dimBeforeSleep, dimLevel, dimTimeout, created_at, updated_at
		FROM display_settings 
		WHERE id = 1
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query)

	var settings models.DisplaySettings
	var createdAtStr, updatedAtStr string

	var screensaverPictureFrameImage sql.NullString

	err := row.Scan(
		&settings.ID,
		&settings.Brightness,
		&settings.Timeout,
		&settings.Orientation,
		&settings.DarkMode,
		&settings.Screensaver,
		&settings.ScreensaverType,
		&settings.ScreensaverShowClock,
		&settings.ScreensaverRotationSpeed,
		&screensaverPictureFrameImage,
		&settings.ScreensaverUploadEnabled,
		&settings.DimBeforeSleep,
		&settings.DimLevel,
		&settings.DimTimeout,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No settings exist, create defaults
			defaultSettings := &models.DisplaySettings{
				ID:                           1,
				Brightness:                   100,
				Timeout:                      300,
				Orientation:                  "landscape",
				DarkMode:                     "auto",
				Screensaver:                  true,
				ScreensaverType:              "clock",
				ScreensaverShowClock:         true,
				ScreensaverRotationSpeed:     5,
				ScreensaverPictureFrameImage: "",
				ScreensaverUploadEnabled:     true,
				DimBeforeSleep:               true,
				DimLevel:                     30,
				DimTimeout:                   60,
				CreatedAt:                    time.Now(),
				UpdatedAt:                    time.Now(),
			}

			// Insert defaults into database
			if err := r.UpdateSettings(ctx, defaultSettings); err != nil {
				return nil, fmt.Errorf("failed to create default settings: %w", err)
			}

			return defaultSettings, nil
		}
		return nil, fmt.Errorf("failed to get display settings: %w", err)
	}

	// Handle nullable fields
	if screensaverPictureFrameImage.Valid {
		settings.ScreensaverPictureFrameImage = screensaverPictureFrameImage.String
	}

	// Parse timestamps
	if createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
		settings.CreatedAt = createdAt
	} else {
		settings.CreatedAt = time.Now()
	}

	if updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr); err == nil {
		settings.UpdatedAt = updatedAt
	} else {
		settings.UpdatedAt = time.Now()
	}

	return &settings, nil
}

// UpdateSettings updates the display settings
func (r *DisplaySettingsRepository) UpdateSettings(ctx context.Context, settings *models.DisplaySettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	// Validate settings
	if err := r.validateSettings(settings); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO display_settings (
			id, brightness, timeout, orientation, darkMode, screensaver, 
			screensaverType, screensaverShowClock, screensaverRotationSpeed,
			screensaverPictureFrameImage, screensaverUploadEnabled, 
			dimBeforeSleep, dimLevel, dimTimeout, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	var screensaverPictureFrameImage sql.NullString
	if settings.ScreensaverPictureFrameImage != "" {
		screensaverPictureFrameImage = sql.NullString{String: settings.ScreensaverPictureFrameImage, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		settings.Brightness,
		settings.Timeout,
		settings.Orientation,
		settings.DarkMode,
		settings.Screensaver,
		settings.ScreensaverType,
		settings.ScreensaverShowClock,
		settings.ScreensaverRotationSpeed,
		screensaverPictureFrameImage,
		settings.ScreensaverUploadEnabled,
		settings.DimBeforeSleep,
		settings.DimLevel,
		settings.DimTimeout,
	)

	if err != nil {
		return fmt.Errorf("failed to update display settings: %w", err)
	}

	// Update the UpdatedAt field
	settings.UpdatedAt = time.Now()

	return nil
}

// validateSettings validates the display settings values
func (r *DisplaySettingsRepository) validateSettings(settings *models.DisplaySettings) error {
	// Validate brightness (0-100)
	if settings.Brightness < 0 || settings.Brightness > 100 {
		return fmt.Errorf("brightness must be between 0 and 100, got %d", settings.Brightness)
	}

	// Validate timeout (must be non-negative)
	if settings.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative, got %d", settings.Timeout)
	}

	// Validate orientation
	validOrientations := map[string]bool{
		"portrait":          true,
		"landscape":         true,
		"portrait_flipped":  true,
		"landscape_flipped": true,
	}
	if !validOrientations[settings.Orientation] {
		return fmt.Errorf("invalid orientation: %s", settings.Orientation)
	}

	// Validate dark mode
	validDarkModes := map[string]bool{
		"light": true,
		"dark":  true,
		"auto":  true,
	}
	if !validDarkModes[settings.DarkMode] {
		return fmt.Errorf("invalid dark mode: %s", settings.DarkMode)
	}

	// Validate screensaver type
	validTypes := map[string]bool{
		"none":         true,
		"clock":        true,
		"slideshow":    true,
		"pictureframe": true,
	}
	if !validTypes[settings.ScreensaverType] {
		return fmt.Errorf("invalid screensaver type: %s", settings.ScreensaverType)
	}

	// Validate screensaver rotation speed (1-60)
	if settings.ScreensaverRotationSpeed < 1 || settings.ScreensaverRotationSpeed > 60 {
		return fmt.Errorf("screensaver rotation speed must be between 1 and 60, got %d", settings.ScreensaverRotationSpeed)
	}

	// Validate dim level (1-99)
	if settings.DimLevel < 1 || settings.DimLevel > 99 {
		return fmt.Errorf("dim level must be between 1 and 99, got %d", settings.DimLevel)
	}

	// Validate dim timeout (must be positive)
	if settings.DimTimeout < 1 {
		return fmt.Errorf("dim timeout must be positive, got %d", settings.DimTimeout)
	}

	return nil
}

// InitializeDefaults ensures default settings exist in the database
func (r *DisplaySettingsRepository) InitializeDefaults(ctx context.Context) error {
	// Check if settings already exist
	_, err := r.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize default display settings: %w", err)
	}
	return nil
}
