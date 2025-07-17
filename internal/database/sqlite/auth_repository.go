package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// AuthRepository implements repositories.AuthRepository
type AuthRepository struct {
	db *sql.DB
}

// NewAuthRepository creates a new AuthRepository
func NewAuthRepository(db *sql.DB) repositories.AuthRepository {
	return &AuthRepository{db: db}
}

// GetSettings retrieves the authentication settings
func (r *AuthRepository) GetSettings(ctx context.Context) (*models.AuthSetting, error) {
	query := `
		SELECT id, pin_code, session_timeout, max_failed_attempts, lockout_duration, last_updated
		FROM auth_settings
		WHERE id = 1
	`

	settings := &models.AuthSetting{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&settings.ID,
		&settings.PinCode,
		&settings.SessionTimeout,
		&settings.MaxFailedAttempts,
		&settings.LockoutDuration,
		&settings.LastUpdated,
	)

	if err == sql.ErrNoRows {
		// Return default settings if none exist
		return &models.AuthSetting{
			ID:                1,
			SessionTimeout:    300,
			MaxFailedAttempts: 3,
			LockoutDuration:   300,
			LastUpdated:       time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get auth settings: %w", err)
	}

	return settings, nil
}

// SetSettings creates or updates the authentication settings
func (r *AuthRepository) SetSettings(ctx context.Context, settings *models.AuthSetting) error {
	query := `
		INSERT INTO auth_settings (id, pin_code, session_timeout, max_failed_attempts, lockout_duration, last_updated)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			pin_code = excluded.pin_code,
			session_timeout = excluded.session_timeout,
			max_failed_attempts = excluded.max_failed_attempts,
			lockout_duration = excluded.lockout_duration,
			last_updated = excluded.last_updated
	`

	settings.LastUpdated = time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		settings.PinCode,
		settings.SessionTimeout,
		settings.MaxFailedAttempts,
		settings.LockoutDuration,
		settings.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to set auth settings: %w", err)
	}

	return nil
}

// CreateSession creates a new authentication session
func (r *AuthRepository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (id, token, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
		session.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (r *AuthRepository) GetSession(ctx context.Context, token string) (*models.Session, error) {
	query := `
		SELECT id, token, expires_at, created_at, updated_at
		FROM sessions
		WHERE token = ? AND expires_at > datetime('now')
	`

	session := &models.Session{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// DeleteSession removes a session by token
func (r *AuthRepository) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = ?`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// DeleteExpiredSessions removes all expired sessions
func (r *AuthRepository) DeleteExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at <= datetime('now')`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// RecordFailedAttempt records a failed authentication attempt
func (r *AuthRepository) RecordFailedAttempt(ctx context.Context, attempt *models.FailedAuthAttempt) error {
	query := `
		INSERT INTO failed_auth_attempts (client_id, ip_address, attempt_at, attempt_type)
		VALUES (?, ?, ?, ?)
	`

	attempt.AttemptAt = time.Now()

	_, err := r.db.ExecContext(
		ctx,
		query,
		attempt.ClientID,
		attempt.IPAddress,
		attempt.AttemptAt,
		attempt.AttemptType,
	)

	if err != nil {
		return fmt.Errorf("failed to record failed attempt: %w", err)
	}

	return nil
}

// GetFailedAttempts retrieves failed attempts for a client since a given time
func (r *AuthRepository) GetFailedAttempts(ctx context.Context, clientID string, since int64) ([]*models.FailedAuthAttempt, error) {
	query := `
		SELECT id, client_id, ip_address, attempt_at, attempt_type
		FROM failed_auth_attempts
		WHERE client_id = ? AND attempt_at > datetime(?, 'unixepoch')
		ORDER BY attempt_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, clientID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed attempts: %w", err)
	}
	defer rows.Close()

	var attempts []*models.FailedAuthAttempt
	for rows.Next() {
		attempt := &models.FailedAuthAttempt{}
		err := rows.Scan(
			&attempt.ID,
			&attempt.ClientID,
			&attempt.IPAddress,
			&attempt.AttemptAt,
			&attempt.AttemptType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan failed attempt: %w", err)
		}
		attempts = append(attempts, attempt)
	}

	return attempts, nil
}

// CleanupFailedAttempts removes failed attempts older than the specified time
func (r *AuthRepository) CleanupFailedAttempts(ctx context.Context, before int64) error {
	query := `DELETE FROM failed_auth_attempts WHERE attempt_at < datetime(?, 'unixepoch')`

	_, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return fmt.Errorf("failed to cleanup failed attempts: %w", err)
	}

	return nil
}
