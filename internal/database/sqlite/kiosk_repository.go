package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// KioskRepository implements the full repositories.KioskRepository interface
type KioskRepository struct {
	db *sql.DB
}

// NewKioskRepository creates a new KioskRepository
func NewKioskRepository(db *sql.DB) repositories.KioskRepository {
	return &KioskRepository{db: db}
}

// ======== TOKEN MANAGEMENT ========

// CreateToken creates a new kiosk token
func (r *KioskRepository) CreateToken(ctx context.Context, token *models.KioskToken) error {
	query := `
		INSERT INTO kiosk_tokens (id, token, name, room_id, allowed_devices, active, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	token.CreatedAt = time.Now()
	if !token.Active {
		token.Active = true // default to active
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		token.ID,
		token.Token,
		token.Name,
		token.RoomID,
		token.AllowedDevices,
		token.Active,
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
		SELECT id, token, name, room_id, allowed_devices, active, created_at, last_used, expires_at
		FROM kiosk_tokens
		WHERE token = ? AND active = 1
	`

	kioskToken := &models.KioskToken{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&kioskToken.ID,
		&kioskToken.Token,
		&kioskToken.Name,
		&kioskToken.RoomID,
		&kioskToken.AllowedDevices,
		&kioskToken.Active,
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
	query := `UPDATE kiosk_tokens SET last_used = ? WHERE token = ? AND active = 1`

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

// GetAllTokens retrieves all active kiosk tokens
func (r *KioskRepository) GetAllTokens(ctx context.Context) ([]*models.KioskToken, error) {
	query := `
		SELECT id, token, name, room_id, allowed_devices, active, created_at, last_used, expires_at
		FROM kiosk_tokens
		WHERE active = 1
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
			&token.Active,
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

// GetTokensByRoom retrieves kiosk tokens for a specific room
func (r *KioskRepository) GetTokensByRoom(ctx context.Context, roomID string) ([]*models.KioskToken, error) {
	query := `
		SELECT id, token, name, room_id, allowed_devices, active, created_at, last_used, expires_at
		FROM kiosk_tokens
		WHERE room_id = ? AND active = 1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query kiosk tokens by room: %w", err)
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
			&token.Active,
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

// UpdateTokenStatus updates the active status of a token
func (r *KioskRepository) UpdateTokenStatus(ctx context.Context, tokenID string, active bool) error {
	query := `UPDATE kiosk_tokens SET active = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, active, tokenID)
	if err != nil {
		return fmt.Errorf("failed to update token status: %w", err)
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

// ======== PAIRING SESSION MANAGEMENT ========

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

// ======== CONFIGURATION MANAGEMENT ========

// CreateConfig creates a new kiosk configuration
func (r *KioskRepository) CreateConfig(ctx context.Context, config *models.KioskConfig) error {
	query := `
		INSERT INTO kiosk_configs (room_id, theme, layout, quick_actions, update_interval, 
			display_timeout, brightness, screensaver_enabled, screensaver_type, screensaver_timeout,
			auto_hide_navigation, fullscreen_mode, voice_control_enabled, gesture_control_enabled,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx,
		query,
		config.RoomID,
		config.Theme,
		config.Layout,
		config.QuickActions,
		config.UpdateInterval,
		config.DisplayTimeout,
		config.Brightness,
		config.ScreensaverEnabled,
		config.ScreensaverType,
		config.ScreensaverTimeout,
		config.AutoHideNavigation,
		config.FullscreenMode,
		config.VoiceControlEnabled,
		config.GestureControlEnabled,
		config.CreatedAt,
		config.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create kiosk config: %w", err)
	}

	return nil
}

// GetConfig retrieves kiosk configuration for a room
func (r *KioskRepository) GetConfig(ctx context.Context, roomID string) (*models.KioskConfig, error) {
	query := `
		SELECT id, room_id, theme, layout, quick_actions, update_interval, display_timeout,
			brightness, screensaver_enabled, screensaver_type, screensaver_timeout,
			auto_hide_navigation, fullscreen_mode, voice_control_enabled, gesture_control_enabled,
			created_at, updated_at
		FROM kiosk_configs
		WHERE room_id = ?
	`

	config := &models.KioskConfig{}
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(
		&config.ID,
		&config.RoomID,
		&config.Theme,
		&config.Layout,
		&config.QuickActions,
		&config.UpdateInterval,
		&config.DisplayTimeout,
		&config.Brightness,
		&config.ScreensaverEnabled,
		&config.ScreensaverType,
		&config.ScreensaverTimeout,
		&config.AutoHideNavigation,
		&config.FullscreenMode,
		&config.VoiceControlEnabled,
		&config.GestureControlEnabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("kiosk config not found for room")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get kiosk config: %w", err)
	}

	return config, nil
}

// UpdateConfig updates kiosk configuration
func (r *KioskRepository) UpdateConfig(ctx context.Context, config *models.KioskConfig) error {
	query := `
		UPDATE kiosk_configs 
		SET theme = ?, layout = ?, quick_actions = ?, update_interval = ?, display_timeout = ?,
			brightness = ?, screensaver_enabled = ?, screensaver_type = ?, screensaver_timeout = ?,
			auto_hide_navigation = ?, fullscreen_mode = ?, voice_control_enabled = ?, 
			gesture_control_enabled = ?, updated_at = ?
		WHERE room_id = ?
	`

	config.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		config.Theme,
		config.Layout,
		config.QuickActions,
		config.UpdateInterval,
		config.DisplayTimeout,
		config.Brightness,
		config.ScreensaverEnabled,
		config.ScreensaverType,
		config.ScreensaverTimeout,
		config.AutoHideNavigation,
		config.FullscreenMode,
		config.VoiceControlEnabled,
		config.GestureControlEnabled,
		config.UpdatedAt,
		config.RoomID,
	)

	if err != nil {
		return fmt.Errorf("failed to update kiosk config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("kiosk config not found")
	}

	return nil
}

// DeleteConfig removes kiosk configuration for a room
func (r *KioskRepository) DeleteConfig(ctx context.Context, roomID string) error {
	query := `DELETE FROM kiosk_configs WHERE room_id = ?`

	result, err := r.db.ExecContext(ctx, query, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete kiosk config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("kiosk config not found")
	}

	return nil
}

// ======== DEVICE GROUP MANAGEMENT ========

// CreateDeviceGroup creates a new device group
func (r *KioskRepository) CreateDeviceGroup(ctx context.Context, group *models.KioskDeviceGroup) error {
	query := `
		INSERT INTO kiosk_device_groups (id, name, description, color, icon, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx,
		query,
		group.ID,
		group.Name,
		group.Description,
		group.Color,
		group.Icon,
		group.CreatedAt,
		group.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create device group: %w", err)
	}

	return nil
}

// GetDeviceGroup retrieves a device group by ID
func (r *KioskRepository) GetDeviceGroup(ctx context.Context, groupID string) (*models.KioskDeviceGroup, error) {
	query := `
		SELECT id, name, description, color, icon, created_at, updated_at
		FROM kiosk_device_groups
		WHERE id = ?
	`

	group := &models.KioskDeviceGroup{}
	err := r.db.QueryRowContext(ctx, query, groupID).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.Color,
		&group.Icon,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device group not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device group: %w", err)
	}

	return group, nil
}

// GetAllDeviceGroups retrieves all device groups
func (r *KioskRepository) GetAllDeviceGroups(ctx context.Context) ([]*models.KioskDeviceGroup, error) {
	query := `
		SELECT id, name, description, color, icon, created_at, updated_at
		FROM kiosk_device_groups
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query device groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.KioskDeviceGroup
	for rows.Next() {
		group := &models.KioskDeviceGroup{}
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.Color,
			&group.Icon,
			&group.CreatedAt,
			&group.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// UpdateDeviceGroup updates a device group
func (r *KioskRepository) UpdateDeviceGroup(ctx context.Context, group *models.KioskDeviceGroup) error {
	query := `
		UPDATE kiosk_device_groups 
		SET name = ?, description = ?, color = ?, icon = ?, updated_at = ?
		WHERE id = ?
	`

	group.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		group.Name,
		group.Description,
		group.Color,
		group.Icon,
		group.UpdatedAt,
		group.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update device group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device group not found")
	}

	return nil
}

// DeleteDeviceGroup removes a device group
func (r *KioskRepository) DeleteDeviceGroup(ctx context.Context, groupID string) error {
	query := `DELETE FROM kiosk_device_groups WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete device group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device group not found")
	}

	return nil
}

// AddTokenToGroup adds a kiosk token to a device group
func (r *KioskRepository) AddTokenToGroup(ctx context.Context, tokenID, groupID string) error {
	query := `
		INSERT OR IGNORE INTO kiosk_group_memberships (kiosk_token_id, group_id, added_at)
		VALUES (?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, tokenID, groupID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add token to group: %w", err)
	}

	return nil
}

// RemoveTokenFromGroup removes a kiosk token from a device group
func (r *KioskRepository) RemoveTokenFromGroup(ctx context.Context, tokenID, groupID string) error {
	query := `DELETE FROM kiosk_group_memberships WHERE kiosk_token_id = ? AND group_id = ?`

	result, err := r.db.ExecContext(ctx, query, tokenID, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove token from group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("group membership not found")
	}

	return nil
}

// GetTokenGroups retrieves all groups for a kiosk token
func (r *KioskRepository) GetTokenGroups(ctx context.Context, tokenID string) ([]*models.KioskDeviceGroup, error) {
	query := `
		SELECT g.id, g.name, g.description, g.color, g.icon, g.created_at, g.updated_at
		FROM kiosk_device_groups g
		JOIN kiosk_group_memberships m ON g.id = m.group_id
		WHERE m.kiosk_token_id = ?
		ORDER BY g.name
	`

	rows, err := r.db.QueryContext(ctx, query, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to query token groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.KioskDeviceGroup
	for rows.Next() {
		group := &models.KioskDeviceGroup{}
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.Color,
			&group.Icon,
			&group.CreatedAt,
			&group.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetGroupTokens retrieves all kiosk tokens in a device group
func (r *KioskRepository) GetGroupTokens(ctx context.Context, groupID string) ([]*models.KioskToken, error) {
	query := `
		SELECT t.id, t.token, t.name, t.room_id, t.allowed_devices, t.active, t.created_at, t.last_used, t.expires_at
		FROM kiosk_tokens t
		JOIN kiosk_group_memberships m ON t.id = m.kiosk_token_id
		WHERE m.group_id = ? AND t.active = 1
		ORDER BY t.name
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group tokens: %w", err)
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
			&token.Active,
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

// ======== LOGGING ========

// CreateLog creates a new kiosk log entry
func (r *KioskRepository) CreateLog(ctx context.Context, log *models.KioskLog) error {
	query := `
		INSERT INTO kiosk_logs (kiosk_token_id, level, category, message, details, device_id,
			user_action, error_code, stack_trace, ip_address, user_agent, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	log.Timestamp = time.Now()

	_, err := r.db.ExecContext(
		ctx,
		query,
		log.KioskTokenID,
		log.Level,
		log.Category,
		log.Message,
		log.Details,
		log.DeviceID,
		log.UserAction,
		log.ErrorCode,
		log.StackTrace,
		log.IPAddress,
		log.UserAgent,
		log.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to create kiosk log: %w", err)
	}

	return nil
}

// GetLogs retrieves kiosk logs with filtering
func (r *KioskRepository) GetLogs(ctx context.Context, tokenID string, query *models.KioskLogQuery) ([]*models.KioskLog, error) {
	sqlQuery := `
		SELECT id, kiosk_token_id, level, category, message, details, device_id,
			user_action, error_code, stack_trace, ip_address, user_agent, timestamp
		FROM kiosk_logs
		WHERE kiosk_token_id = ?
	`

	args := []interface{}{tokenID}

	if query != nil {
		if query.Level != "" {
			sqlQuery += " AND level = ?"
			args = append(args, query.Level)
		}
		if query.Category != "" {
			sqlQuery += " AND category = ?"
			args = append(args, query.Category)
		}
		if !query.StartTime.IsZero() {
			sqlQuery += " AND timestamp >= ?"
			args = append(args, query.StartTime)
		}
		if !query.EndTime.IsZero() {
			sqlQuery += " AND timestamp <= ?"
			args = append(args, query.EndTime)
		}
		if query.DeviceID != "" {
			sqlQuery += " AND device_id = ?"
			args = append(args, query.DeviceID)
		}
	}

	sqlQuery += " ORDER BY timestamp DESC"

	if query != nil && query.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.Limit)

		if query.Offset > 0 {
			sqlQuery += " OFFSET ?"
			args = append(args, query.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query kiosk logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.KioskLog
	for rows.Next() {
		log := &models.KioskLog{}
		err := rows.Scan(
			&log.ID,
			&log.KioskTokenID,
			&log.Level,
			&log.Category,
			&log.Message,
			&log.Details,
			&log.DeviceID,
			&log.UserAction,
			&log.ErrorCode,
			&log.StackTrace,
			&log.IPAddress,
			&log.UserAgent,
			&log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan kiosk log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// DeleteOldLogs removes logs older than the specified number of days
func (r *KioskRepository) DeleteOldLogs(ctx context.Context, olderThanDays int) error {
	query := `DELETE FROM kiosk_logs WHERE timestamp < datetime('now', '-' || ? || ' days')`

	_, err := r.db.ExecContext(ctx, query, olderThanDays)
	if err != nil {
		return fmt.Errorf("failed to delete old kiosk logs: %w", err)
	}

	return nil
}

// ======== DEVICE STATUS MANAGEMENT ========

// CreateOrUpdateDeviceStatus creates or updates device status
func (r *KioskRepository) CreateOrUpdateDeviceStatus(ctx context.Context, status *models.KioskDeviceStatus) error {
	query := `
		INSERT OR REPLACE INTO kiosk_device_status (
			kiosk_token_id, status, last_heartbeat, device_info, performance_metrics,
			error_count_24h, uptime_seconds, network_quality, battery_level, temperature,
			memory_usage_percent, cpu_usage_percent, storage_usage_percent, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	status.LastHeartbeat = now
	status.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx,
		query,
		status.KioskTokenID,
		status.Status,
		status.LastHeartbeat,
		status.DeviceInfo,
		status.PerformanceMetrics,
		status.ErrorCount24h,
		status.UptimeSeconds,
		status.NetworkQuality,
		status.BatteryLevel,
		status.Temperature,
		status.MemoryUsagePercent,
		status.CPUUsagePercent,
		status.StorageUsagePercent,
		status.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create/update device status: %w", err)
	}

	return nil
}

// GetDeviceStatus retrieves device status for a kiosk token
func (r *KioskRepository) GetDeviceStatus(ctx context.Context, tokenID string) (*models.KioskDeviceStatus, error) {
	query := `
		SELECT id, kiosk_token_id, status, last_heartbeat, device_info, performance_metrics,
			error_count_24h, uptime_seconds, network_quality, battery_level, temperature,
			memory_usage_percent, cpu_usage_percent, storage_usage_percent, created_at, updated_at
		FROM kiosk_device_status
		WHERE kiosk_token_id = ?
	`

	status := &models.KioskDeviceStatus{}
	err := r.db.QueryRowContext(ctx, query, tokenID).Scan(
		&status.ID,
		&status.KioskTokenID,
		&status.Status,
		&status.LastHeartbeat,
		&status.DeviceInfo,
		&status.PerformanceMetrics,
		&status.ErrorCount24h,
		&status.UptimeSeconds,
		&status.NetworkQuality,
		&status.BatteryLevel,
		&status.Temperature,
		&status.MemoryUsagePercent,
		&status.CPUUsagePercent,
		&status.StorageUsagePercent,
		&status.CreatedAt,
		&status.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device status not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device status: %w", err)
	}

	return status, nil
}

// GetAllDeviceStatuses retrieves all device statuses
func (r *KioskRepository) GetAllDeviceStatuses(ctx context.Context) ([]*models.KioskDeviceStatus, error) {
	query := `
		SELECT id, kiosk_token_id, status, last_heartbeat, device_info, performance_metrics,
			error_count_24h, uptime_seconds, network_quality, battery_level, temperature,
			memory_usage_percent, cpu_usage_percent, storage_usage_percent, created_at, updated_at
		FROM kiosk_device_status
		ORDER BY last_heartbeat DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query device statuses: %w", err)
	}
	defer rows.Close()

	var statuses []*models.KioskDeviceStatus
	for rows.Next() {
		status := &models.KioskDeviceStatus{}
		err := rows.Scan(
			&status.ID,
			&status.KioskTokenID,
			&status.Status,
			&status.LastHeartbeat,
			&status.DeviceInfo,
			&status.PerformanceMetrics,
			&status.ErrorCount24h,
			&status.UptimeSeconds,
			&status.NetworkQuality,
			&status.BatteryLevel,
			&status.Temperature,
			&status.MemoryUsagePercent,
			&status.CPUUsagePercent,
			&status.StorageUsagePercent,
			&status.CreatedAt,
			&status.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device status: %w", err)
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// UpdateHeartbeat updates the last heartbeat timestamp for a device
func (r *KioskRepository) UpdateHeartbeat(ctx context.Context, tokenID string) error {
	query := `
		UPDATE kiosk_device_status 
		SET last_heartbeat = ?, updated_at = ?
		WHERE kiosk_token_id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, now, tokenID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Create initial status if it doesn't exist
		status := &models.KioskDeviceStatus{
			KioskTokenID:  tokenID,
			Status:        "online",
			LastHeartbeat: now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		return r.CreateOrUpdateDeviceStatus(ctx, status)
	}

	return nil
}

// ======== COMMAND MANAGEMENT ========

// CreateCommand creates a new kiosk command
func (r *KioskRepository) CreateCommand(ctx context.Context, command *models.KioskCommand) error {
	query := `
		INSERT INTO kiosk_commands (id, kiosk_token_id, command_type, command_data, status, 
			created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	command.CreatedAt = time.Now()

	_, err := r.db.ExecContext(
		ctx,
		query,
		command.ID,
		command.KioskTokenID,
		command.CommandType,
		command.CommandData,
		command.Status,
		command.CreatedAt,
		command.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create kiosk command: %w", err)
	}

	return nil
}

// GetCommand retrieves a kiosk command by ID
func (r *KioskRepository) GetCommand(ctx context.Context, commandID string) (*models.KioskCommand, error) {
	query := `
		SELECT id, kiosk_token_id, command_type, command_data, status, created_at,
			sent_at, acknowledged_at, completed_at, expires_at, result_data, error_message
		FROM kiosk_commands
		WHERE id = ?
	`

	command := &models.KioskCommand{}
	err := r.db.QueryRowContext(ctx, query, commandID).Scan(
		&command.ID,
		&command.KioskTokenID,
		&command.CommandType,
		&command.CommandData,
		&command.Status,
		&command.CreatedAt,
		&command.SentAt,
		&command.AcknowledgedAt,
		&command.CompletedAt,
		&command.ExpiresAt,
		&command.ResultData,
		&command.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("command not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get command: %w", err)
	}

	return command, nil
}

// GetPendingCommands retrieves pending commands for a kiosk token
func (r *KioskRepository) GetPendingCommands(ctx context.Context, tokenID string) ([]*models.KioskCommand, error) {
	query := `
		SELECT id, kiosk_token_id, command_type, command_data, status, created_at,
			sent_at, acknowledged_at, completed_at, expires_at, result_data, error_message
		FROM kiosk_commands
		WHERE kiosk_token_id = ? AND status IN ('pending', 'sent') 
		AND expires_at > datetime('now')
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending commands: %w", err)
	}
	defer rows.Close()

	var commands []*models.KioskCommand
	for rows.Next() {
		command := &models.KioskCommand{}
		err := rows.Scan(
			&command.ID,
			&command.KioskTokenID,
			&command.CommandType,
			&command.CommandData,
			&command.Status,
			&command.CreatedAt,
			&command.SentAt,
			&command.AcknowledgedAt,
			&command.CompletedAt,
			&command.ExpiresAt,
			&command.ResultData,
			&command.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, command)
	}

	return commands, nil
}

// UpdateCommandStatus updates the status of a command
func (r *KioskRepository) UpdateCommandStatus(ctx context.Context, commandID, status string) error {
	var query string
	var args []interface{}

	now := time.Now()

	switch status {
	case "sent":
		query = `UPDATE kiosk_commands SET status = ?, sent_at = ? WHERE id = ?`
		args = []interface{}{status, now, commandID}
	case "acknowledged":
		query = `UPDATE kiosk_commands SET status = ?, acknowledged_at = ? WHERE id = ?`
		args = []interface{}{status, now, commandID}
	case "completed", "failed":
		query = `UPDATE kiosk_commands SET status = ?, completed_at = ? WHERE id = ?`
		args = []interface{}{status, now, commandID}
	default:
		query = `UPDATE kiosk_commands SET status = ? WHERE id = ?`
		args = []interface{}{status, commandID}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update command status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("command not found")
	}

	return nil
}

// CompleteCommand marks a command as completed with result data
func (r *KioskRepository) CompleteCommand(ctx context.Context, commandID string, resultData []byte, errorMsg string) error {
	query := `
		UPDATE kiosk_commands 
		SET status = ?, completed_at = ?, result_data = ?, error_message = ?
		WHERE id = ?
	`

	status := "completed"
	if errorMsg != "" {
		status = "failed"
	}

	result, err := r.db.ExecContext(
		ctx,
		query,
		status,
		time.Now(),
		resultData,
		errorMsg,
		commandID,
	)

	if err != nil {
		return fmt.Errorf("failed to complete command: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("command not found")
	}

	return nil
}

// CleanupExpiredCommands removes expired commands
func (r *KioskRepository) CleanupExpiredCommands(ctx context.Context) error {
	query := `DELETE FROM kiosk_commands WHERE expires_at <= datetime('now')`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired commands: %w", err)
	}

	return nil
}

// ======== STATISTICS ========

// GetKioskStats retrieves kiosk system statistics
func (r *KioskRepository) GetKioskStats(ctx context.Context) (*models.KioskStatsResponse, error) {
	stats := &models.KioskStatsResponse{}

	// Total and active devices
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN active = 1 THEN 1 END) as active
		FROM kiosk_tokens
	`).Scan(&stats.TotalDevices, &stats.ActiveDevices)
	if err != nil {
		return nil, fmt.Errorf("failed to get device counts: %w", err)
	}

	// Device status counts
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(CASE WHEN status = 'offline' THEN 1 END) as offline,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as error_devices
		FROM kiosk_device_status s
		JOIN kiosk_tokens t ON s.kiosk_token_id = t.id
		WHERE t.active = 1
	`).Scan(&stats.OfflineDevices, &stats.ErrorDevices)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	// Total groups
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM kiosk_device_groups`).Scan(&stats.TotalGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to get group count: %w", err)
	}

	// Pending pairing sessions
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM kiosk_pairing_sessions 
		WHERE status = 'pending' AND expires_at > datetime('now')
	`).Scan(&stats.PendingSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending sessions count: %w", err)
	}

	// Logs today
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_logs,
			COUNT(CASE WHEN level IN ('error', 'critical') THEN 1 END) as error_logs
		FROM kiosk_logs 
		WHERE date(timestamp) = date('now')
	`).Scan(&stats.LogsToday, &stats.ErrorsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get log counts: %w", err)
	}

	return stats, nil
}
