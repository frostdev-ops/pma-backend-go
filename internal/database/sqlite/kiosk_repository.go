package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// KioskRepository implements repositories.KioskRepository
type KioskRepository struct {
	db *sql.DB
}

// NewKioskRepository creates a new KioskRepository
func NewKioskRepository(db *sql.DB) repositories.KioskRepository {
	return &KioskRepository{db: db}
}

// CreateToken creates a new kiosk token
func (r *KioskRepository) CreateToken(ctx context.Context, token *models.KioskToken) error {
	query := `
		INSERT INTO kiosk_tokens (id, token, name, room_id, allowed_devices, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	token.CreatedAt = time.Now()

	_, err := r.db.ExecContext(
		ctx,
		query,
		token.ID,
		token.Token,
		token.Name,
		token.RoomID,
		token.AllowedDevices,
		token.CreatedAt,
		token.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create kiosk token: %w", err)
	}

	return nil
}

// GetToken retrieves a kiosk token
func (r *KioskRepository) GetToken(ctx context.Context, token string) (*models.KioskToken, error) {
	query := `
		SELECT id, token, name, room_id, allowed_devices, created_at, last_used, expires_at
		FROM kiosk_tokens
		WHERE token = ?
	`

	kioskToken := &models.KioskToken{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&kioskToken.ID,
		&kioskToken.Token,
		&kioskToken.Name,
		&kioskToken.RoomID,
		&kioskToken.AllowedDevices,
		&kioskToken.CreatedAt,
		&kioskToken.LastUsed,
		&kioskToken.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("kiosk token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get kiosk token: %w", err)
	}

	return kioskToken, nil
}

// UpdateTokenLastUsed updates the last used timestamp for a token
func (r *KioskRepository) UpdateTokenLastUsed(ctx context.Context, token string) error {
	query := `UPDATE kiosk_tokens SET last_used = ? WHERE token = ?`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, token)
	if err != nil {
		return fmt.Errorf("failed to update token last used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token not found")
	}

	return nil
}

// DeleteToken removes a kiosk token
func (r *KioskRepository) DeleteToken(ctx context.Context, token string) error {
	query := `DELETE FROM kiosk_tokens WHERE token = ?`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete kiosk token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token not found")
	}

	return nil
}

// GetAllTokens retrieves all kiosk tokens
func (r *KioskRepository) GetAllTokens(ctx context.Context) ([]*models.KioskToken, error) {
	query := `
		SELECT id, token, name, room_id, allowed_devices, created_at, last_used, expires_at
		FROM kiosk_tokens
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query kiosk tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.KioskToken
	for rows.Next() {
		token := &models.KioskToken{}
		err := rows.Scan(
			&token.ID,
			&token.Token,
			&token.Name,
			&token.RoomID,
			&token.AllowedDevices,
			&token.CreatedAt,
			&token.LastUsed,
			&token.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan kiosk token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// CreatePairingSession creates a new kiosk pairing session
func (r *KioskRepository) CreatePairingSession(ctx context.Context, session *models.KioskPairingSession) error {
	query := `
		INSERT INTO kiosk_pairing_sessions (id, pin, room_id, device_info, expires_at, created_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	session.CreatedAt = time.Now()

	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.Pin,
		session.RoomID,
		session.DeviceInfo,
		session.ExpiresAt,
		session.CreatedAt,
		session.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to create pairing session: %w", err)
	}

	return nil
}

// GetPairingSession retrieves a pairing session by PIN
func (r *KioskRepository) GetPairingSession(ctx context.Context, pin string) (*models.KioskPairingSession, error) {
	query := `
		SELECT id, pin, room_id, device_info, expires_at, created_at, status
		FROM kiosk_pairing_sessions
		WHERE pin = ? AND expires_at > datetime('now')
	`

	session := &models.KioskPairingSession{}
	err := r.db.QueryRowContext(ctx, query, pin).Scan(
		&session.ID,
		&session.Pin,
		&session.RoomID,
		&session.DeviceInfo,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.Status,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pairing session not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pairing session: %w", err)
	}

	return session, nil
}

// UpdatePairingSession updates a pairing session
func (r *KioskRepository) UpdatePairingSession(ctx context.Context, session *models.KioskPairingSession) error {
	query := `
		UPDATE kiosk_pairing_sessions 
		SET status = ?, device_info = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, session.Status, session.DeviceInfo, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update pairing session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pairing session not found")
	}

	return nil
}

// DeletePairingSession removes a pairing session
func (r *KioskRepository) DeletePairingSession(ctx context.Context, id string) error {
	query := `DELETE FROM kiosk_pairing_sessions WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pairing session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pairing session not found")
	}

	return nil
}

// CleanupExpiredSessions removes expired pairing sessions
func (r *KioskRepository) CleanupExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM kiosk_pairing_sessions WHERE expires_at <= datetime('now')`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}
