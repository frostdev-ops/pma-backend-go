package preferences

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager implements the PreferencesManager interface
type Manager struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewManager creates a new preferences manager
func NewManager(db *sql.DB, logger *logrus.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: logger,
	}
}

// GetUserPreferences retrieves preferences for a user
func (m *Manager) GetUserPreferences(userID string) (*UserPreferences, error) {
	query := `SELECT preferences FROM user_preferences WHERE user_id = ?`

	var prefsJSON string
	err := m.db.QueryRow(query, userID).Scan(&prefsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences for new users
			defaults := DefaultPreferences()
			defaults.UserID = userID

			// Save default preferences to database
			if saveErr := m.UpdateUserPreferences(userID, defaults); saveErr != nil {
				m.logger.WithError(saveErr).Error("Failed to save default preferences")
			}

			return defaults, nil
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	var prefs UserPreferences
	if err := json.Unmarshal([]byte(prefsJSON), &prefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	prefs.UserID = userID
	return &prefs, nil
}

// UpdateUserPreferences saves user preferences to database
func (m *Manager) UpdateUserPreferences(userID string, prefs *UserPreferences) error {
	prefs.UserID = userID
	prefs.UpdatedAt = time.Now()

	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	query := `
		INSERT INTO user_preferences (user_id, preferences, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			preferences = excluded.preferences,
			updated_at = excluded.updated_at
	`

	_, err = m.db.Exec(query, userID, string(prefsJSON), prefs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"action":  "update_preferences",
	}).Info("User preferences updated")

	return nil
}

// GetPreference retrieves a specific preference value by key path
func (m *Manager) GetPreference(userID string, key string) (interface{}, error) {
	prefs, err := m.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}

	return m.getNestedValue(prefs, key)
}

// SetPreference sets a specific preference value by key path
func (m *Manager) SetPreference(userID string, key string, value interface{}) error {
	prefs, err := m.GetUserPreferences(userID)
	if err != nil {
		return err
	}

	if err := m.setNestedValue(prefs, key, value); err != nil {
		return err
	}

	return m.UpdateUserPreferences(userID, prefs)
}

// ResetToDefaults resets user preferences to default values
func (m *Manager) ResetToDefaults(userID string) error {
	defaults := DefaultPreferences()
	defaults.UserID = userID

	return m.UpdateUserPreferences(userID, defaults)
}

// ExportPreferences exports user preferences as JSON
func (m *Manager) ExportPreferences(userID string) ([]byte, error) {
	prefs, err := m.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}

	export := struct {
		Version     string           `json:"version"`
		ExportedAt  time.Time        `json:"exported_at"`
		UserID      string           `json:"user_id"`
		Preferences *UserPreferences `json:"preferences"`
	}{
		Version:     "1.0",
		ExportedAt:  time.Now(),
		UserID:      userID,
		Preferences: prefs,
	}

	return json.MarshalIndent(export, "", "  ")
}

// ImportPreferences imports user preferences from JSON
func (m *Manager) ImportPreferences(userID string, data []byte) error {
	var export struct {
		Version     string           `json:"version"`
		ExportedAt  time.Time        `json:"exported_at"`
		UserID      string           `json:"user_id"`
		Preferences *UserPreferences `json:"preferences"`
	}

	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("failed to unmarshal import data: %w", err)
	}

	if export.Preferences == nil {
		return fmt.Errorf("no preferences found in import data")
	}

	// Validate import version compatibility
	if export.Version != "1.0" {
		m.logger.WithField("version", export.Version).Warn("Unknown import version, proceeding anyway")
	}

	// Override user ID to current user
	export.Preferences.UserID = userID

	return m.UpdateUserPreferences(userID, export.Preferences)
}

// GetUsersByPreference finds users with a specific preference value
func (m *Manager) GetUsersByPreference(key string, value interface{}) ([]string, error) {
	query := `
		SELECT user_id, preferences 
		FROM user_preferences
	`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID, prefsJSON string
		if err := rows.Scan(&userID, &prefsJSON); err != nil {
			m.logger.WithError(err).Error("Failed to scan preference row")
			continue
		}

		var prefs UserPreferences
		if err := json.Unmarshal([]byte(prefsJSON), &prefs); err != nil {
			m.logger.WithError(err).Error("Failed to unmarshal preferences")
			continue
		}

		if prefValue, err := m.getNestedValue(&prefs, key); err == nil {
			if compareValues(prefValue, value) {
				userIDs = append(userIDs, userID)
			}
		}
	}

	return userIDs, nil
}

// BulkUpdatePreference updates a specific preference for multiple users
func (m *Manager) BulkUpdatePreference(userIDs []string, key string, value interface{}) error {
	for _, userID := range userIDs {
		if err := m.SetPreference(userID, key, value); err != nil {
			m.logger.WithFields(logrus.Fields{
				"user_id": userID,
				"key":     key,
				"error":   err,
			}).Error("Failed to update preference for user")
		}
	}
	return nil
}

// getNestedValue retrieves a value from nested preferences using dot notation
func (m *Manager) getNestedValue(prefs *UserPreferences, key string) (interface{}, error) {
	parts := strings.Split(key, ".")

	// Convert struct to map for easier navigation
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return nil, err
	}

	var prefsMap map[string]interface{}
	if err := json.Unmarshal(prefsJSON, &prefsMap); err != nil {
		return nil, err
	}

	current := prefsMap
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			return current[part], nil
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("key path %s not found", key)
		}
		current = next
	}

	return nil, fmt.Errorf("key path %s not found", key)
}

// setNestedValue sets a value in nested preferences using dot notation
func (m *Manager) setNestedValue(prefs *UserPreferences, key string, value interface{}) error {
	parts := strings.Split(key, ".")

	// Convert struct to map for easier navigation
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	var prefsMap map[string]interface{}
	if err := json.Unmarshal(prefsJSON, &prefsMap); err != nil {
		return err
	}

	// Navigate to parent of target key
	current := prefsMap
	for i, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return fmt.Errorf("key path %s not found at part %d", key, i)
		}
		current = next
	}

	// Set the value
	current[parts[len(parts)-1]] = value

	// Convert back to struct
	updatedJSON, err := json.Marshal(prefsMap)
	if err != nil {
		return err
	}

	return json.Unmarshal(updatedJSON, prefs)
}

// compareValues compares two values for equality
func compareValues(a, b interface{}) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// GetPreferenceStatistics returns statistics about preference usage
func (m *Manager) GetPreferenceStatistics() (map[string]interface{}, error) {
	query := `
		SELECT COUNT(*) as total_users,
			   AVG(LENGTH(preferences)) as avg_size,
			   MIN(updated_at) as oldest_update,
			   MAX(updated_at) as newest_update
		FROM user_preferences
	`

	var stats struct {
		TotalUsers   int       `db:"total_users"`
		AvgSize      float64   `db:"avg_size"`
		OldestUpdate time.Time `db:"oldest_update"`
		NewestUpdate time.Time `db:"newest_update"`
	}

	err := m.db.QueryRow(query).Scan(
		&stats.TotalUsers,
		&stats.AvgSize,
		&stats.OldestUpdate,
		&stats.NewestUpdate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get preference statistics: %w", err)
	}

	result := map[string]interface{}{
		"total_users":    stats.TotalUsers,
		"avg_size_bytes": stats.AvgSize,
		"oldest_update":  stats.OldestUpdate,
		"newest_update":  stats.NewestUpdate,
	}

	return result, nil
}

// CleanupExpiredAPIAccess removes expired API access rules
func (m *Manager) CleanupExpiredAPIAccess() error {
	query := `SELECT user_id, preferences FROM user_preferences`
	rows, err := m.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query preferences for cleanup: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var userID, prefsJSON string
		if err := rows.Scan(&userID, &prefsJSON); err != nil {
			continue
		}

		var prefs UserPreferences
		if err := json.Unmarshal([]byte(prefsJSON), &prefs); err != nil {
			continue
		}

		// Filter out expired API access rules
		var validRules []APIAccessRule
		hasExpired := false

		for _, rule := range prefs.Privacy.APIAccess {
			if rule.ExpiresAt == nil || rule.ExpiresAt.After(now) {
				validRules = append(validRules, rule)
			} else {
				hasExpired = true
			}
		}

		if hasExpired {
			prefs.Privacy.APIAccess = validRules
			if err := m.UpdateUserPreferences(userID, &prefs); err != nil {
				m.logger.WithError(err).WithField("user_id", userID).Error("Failed to update preferences after API access cleanup")
			}
		}
	}

	return nil
}
