package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/jmoiron/sqlx"
)

// ControllerRepository implements repositories.ControllerRepository
type ControllerRepository struct {
	db  *sql.DB
	dbx *sqlx.DB
}

// NewControllerRepository creates a new ControllerRepository
func NewControllerRepository(db *sql.DB) repositories.ControllerRepository {
	return &ControllerRepository{
		db:  db,
		dbx: sqlx.NewDb(db, "sqlite3"),
	}
}

// Dashboard CRUD operations

func (r *ControllerRepository) CreateDashboard(ctx context.Context, dashboard *models.ControllerDashboard) error {
	query := `
		INSERT INTO controller_dashboards (
			name, description, category, layout_config, elements_json, 
			style_config, access_config, is_favorite, tags, 
			thumbnail_url, version, user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		dashboard.Name,
		dashboard.Description,
		dashboard.Category,
		dashboard.LayoutConfig,
		dashboard.ElementsJSON,
		dashboard.StyleConfig,
		dashboard.AccessConfig,
		dashboard.IsFavorite,
		dashboard.Tags,
		dashboard.ThumbnailURL,
		dashboard.Version,
		dashboard.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get dashboard ID: %w", err)
	}

	dashboard.ID = int(id)
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	return nil
}

func (r *ControllerRepository) GetDashboardByID(ctx context.Context, id int) (*models.ControllerDashboard, error) {
	query := `
		SELECT id, name, description, category, layout_config, elements_json,
			   style_config, access_config, is_favorite, tags, thumbnail_url,
			   version, user_id, created_at, updated_at, last_accessed
		FROM controller_dashboards
		WHERE id = ?
	`

	dashboard := &models.ControllerDashboard{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&dashboard.ID,
		&dashboard.Name,
		&dashboard.Description,
		&dashboard.Category,
		&dashboard.LayoutConfig,
		&dashboard.ElementsJSON,
		&dashboard.StyleConfig,
		&dashboard.AccessConfig,
		&dashboard.IsFavorite,
		&dashboard.Tags,
		&dashboard.ThumbnailURL,
		&dashboard.Version,
		&dashboard.UserID,
		&dashboard.CreatedAt,
		&dashboard.UpdatedAt,
		&dashboard.LastAccessed,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dashboard not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	return dashboard, nil
}

func (r *ControllerRepository) GetDashboardsByUserID(ctx context.Context, userID *int, includeShared bool) ([]*models.ControllerDashboard, error) {
	var query string
	var args []interface{}

	if includeShared {
		query = `
			SELECT DISTINCT d.id, d.name, d.description, d.category, d.layout_config, 
				   d.elements_json, d.style_config, d.access_config, d.is_favorite, 
				   d.tags, d.thumbnail_url, d.version, d.user_id, d.created_at, 
				   d.updated_at, d.last_accessed
			FROM controller_dashboards d
			LEFT JOIN controller_shares s ON d.id = s.dashboard_id
			WHERE d.user_id = ? OR s.user_id = ?
			ORDER BY d.updated_at DESC
		`
		args = []interface{}{userID, userID}
	} else {
		query = `
			SELECT id, name, description, category, layout_config, elements_json,
				   style_config, access_config, is_favorite, tags, thumbnail_url,
				   version, user_id, created_at, updated_at, last_accessed
			FROM controller_dashboards
			WHERE user_id = ?
			ORDER BY updated_at DESC
		`
		args = []interface{}{userID}
	}

	dashboards := []*models.ControllerDashboard{}
	err := r.dbx.SelectContext(ctx, &dashboards, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user dashboards: %w", err)
	}

	return dashboards, nil
}

func (r *ControllerRepository) GetAllDashboards(ctx context.Context, userID *int) ([]*models.ControllerDashboard, error) {
	// For kiosk-based system: return ALL dashboards regardless of user/access control
	query := `
			SELECT id, name, description, category, layout_config, elements_json,
				   style_config, access_config, is_favorite, tags, thumbnail_url,
				   version, user_id, created_at, updated_at, last_accessed
			FROM controller_dashboards
			ORDER BY updated_at DESC
		`

	dashboards := []*models.ControllerDashboard{}
	err := r.dbx.SelectContext(ctx, &dashboards, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all dashboards: %w", err)
	}

	return dashboards, nil
}

func (r *ControllerRepository) UpdateDashboard(ctx context.Context, dashboard *models.ControllerDashboard) error {
	query := `
		UPDATE controller_dashboards 
		SET name = ?, description = ?, category = ?, layout_config = ?, 
			elements_json = ?, style_config = ?, access_config = ?, 
			is_favorite = ?, tags = ?, thumbnail_url = ?, version = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		dashboard.Name,
		dashboard.Description,
		dashboard.Category,
		dashboard.LayoutConfig,
		dashboard.ElementsJSON,
		dashboard.StyleConfig,
		dashboard.AccessConfig,
		dashboard.IsFavorite,
		dashboard.Tags,
		dashboard.ThumbnailURL,
		dashboard.Version,
		dashboard.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	return nil
}

func (r *ControllerRepository) DeleteDashboard(ctx context.Context, id int) error {
	// Foreign key constraints will cascade delete shares and usage logs
	query := `DELETE FROM controller_dashboards WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	return nil
}

func (r *ControllerRepository) DuplicateDashboard(ctx context.Context, id int, userID *int, newName string) (*models.ControllerDashboard, error) {
	// Get original dashboard
	original, err := r.GetDashboardByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get original dashboard: %w", err)
	}

	// Create new dashboard with modified properties
	duplicate := &models.ControllerDashboard{
		Name:         newName,
		Description:  original.Description + " (Copy)",
		Category:     original.Category,
		LayoutConfig: original.LayoutConfig,
		ElementsJSON: original.ElementsJSON,
		StyleConfig:  original.StyleConfig,
		AccessConfig: original.AccessConfig,
		IsFavorite:   false,
		Tags:         original.Tags,
		ThumbnailURL: original.ThumbnailURL,
		Version:      1,
		UserID:       userID,
	}

	err = r.CreateDashboard(ctx, duplicate)
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicate dashboard: %w", err)
	}

	return duplicate, nil
}

// Dashboard searching and filtering

func (r *ControllerRepository) SearchDashboards(ctx context.Context, userID *int, query string, category string, tags []string) ([]*models.ControllerDashboard, error) {
	var sqlQuery strings.Builder
	var args []interface{}

	sqlQuery.WriteString(`
		SELECT DISTINCT d.id, d.name, d.description, d.category, d.layout_config, 
			   d.elements_json, d.style_config, d.access_config, d.is_favorite, 
			   d.tags, d.thumbnail_url, d.version, d.user_id, d.created_at, 
			   d.updated_at, d.last_accessed
		FROM controller_dashboards d
		LEFT JOIN controller_shares s ON d.id = s.dashboard_id
		WHERE 1=1
	`)

	// User access filter
	if userID != nil {
		sqlQuery.WriteString(` AND (d.user_id = ? OR s.user_id = ? OR JSON_EXTRACT(d.access_config, '$.public') = true)`)
		args = append(args, userID, userID)
	} else {
		sqlQuery.WriteString(` AND JSON_EXTRACT(d.access_config, '$.public') = true`)
	}

	// Text search filter
	if query != "" {
		sqlQuery.WriteString(` AND (d.name LIKE ? OR d.description LIKE ?)`)
		searchTerm := "%" + query + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Category filter
	if category != "" {
		sqlQuery.WriteString(` AND d.category = ?`)
		args = append(args, category)
	}

	// Tags filter (simplified - check if any tag matches)
	if len(tags) > 0 {
		tagsConditions := make([]string, len(tags))
		for i, tag := range tags {
			tagsConditions[i] = `d.tags LIKE ?`
			args = append(args, "%\""+tag+"\"%")
		}
		sqlQuery.WriteString(` AND (` + strings.Join(tagsConditions, " OR ") + `)`)
	}

	sqlQuery.WriteString(` ORDER BY d.updated_at DESC`)

	dashboards := []*models.ControllerDashboard{}
	err := r.dbx.SelectContext(ctx, &dashboards, sqlQuery.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search dashboards: %w", err)
	}

	return dashboards, nil
}

func (r *ControllerRepository) GetDashboardsByCategory(ctx context.Context, userID *int, category string) ([]*models.ControllerDashboard, error) {
	return r.SearchDashboards(ctx, userID, "", category, nil)
}

func (r *ControllerRepository) GetFavoriteDashboards(ctx context.Context, userID int) ([]*models.ControllerDashboard, error) {
	query := `
		SELECT id, name, description, category, layout_config, elements_json,
			   style_config, access_config, is_favorite, tags, thumbnail_url,
			   version, user_id, created_at, updated_at, last_accessed
		FROM controller_dashboards
		WHERE user_id = ? AND is_favorite = true
		ORDER BY updated_at DESC
	`

	dashboards := []*models.ControllerDashboard{}
	err := r.dbx.SelectContext(ctx, &dashboards, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorite dashboards: %w", err)
	}

	return dashboards, nil
}

func (r *ControllerRepository) ToggleFavorite(ctx context.Context, dashboardID int, userID int) error {
	query := `
		UPDATE controller_dashboards 
		SET is_favorite = NOT is_favorite 
		WHERE id = ? AND user_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, dashboardID, userID)
	if err != nil {
		return fmt.Errorf("failed to toggle favorite: %w", err)
	}

	return nil
}

func (r *ControllerRepository) UpdateLastAccessed(ctx context.Context, dashboardID int) error {
	query := `UPDATE controller_dashboards SET last_accessed = CURRENT_TIMESTAMP WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, dashboardID)
	if err != nil {
		return fmt.Errorf("failed to update last accessed: %w", err)
	}

	return nil
}

// Template operations

func (r *ControllerRepository) CreateTemplate(ctx context.Context, template *models.ControllerTemplate) error {
	query := `
		INSERT INTO controller_templates (
			name, description, category, template_json, variables_json,
			thumbnail_url, usage_count, rating, is_public, user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		template.Name,
		template.Description,
		template.Category,
		template.TemplateJSON,
		template.VariablesJSON,
		template.ThumbnailURL,
		template.UsageCount,
		template.Rating,
		template.IsPublic,
		template.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get template ID: %w", err)
	}

	template.ID = int(id)
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	return nil
}

func (r *ControllerRepository) GetTemplateByID(ctx context.Context, id int) (*models.ControllerTemplate, error) {
	query := `
		SELECT id, name, description, category, template_json, variables_json,
			   thumbnail_url, usage_count, rating, is_public, user_id,
			   created_at, updated_at
		FROM controller_templates
		WHERE id = ?
	`

	template := &models.ControllerTemplate{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.Category,
		&template.TemplateJSON,
		&template.VariablesJSON,
		&template.ThumbnailURL,
		&template.UsageCount,
		&template.Rating,
		&template.IsPublic,
		&template.UserID,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return template, nil
}

func (r *ControllerRepository) GetTemplatesByUserID(ctx context.Context, userID *int, includePublic bool) ([]*models.ControllerTemplate, error) {
	var query string
	var args []interface{}

	if includePublic {
		query = `
			SELECT id, name, description, category, template_json, variables_json,
				   thumbnail_url, usage_count, rating, is_public, user_id,
				   created_at, updated_at
			FROM controller_templates
			WHERE user_id = ? OR is_public = true
			ORDER BY usage_count DESC, updated_at DESC
		`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT id, name, description, category, template_json, variables_json,
				   thumbnail_url, usage_count, rating, is_public, user_id,
				   created_at, updated_at
			FROM controller_templates
			WHERE user_id = ?
			ORDER BY updated_at DESC
		`
		args = []interface{}{userID}
	}

	templates := []*models.ControllerTemplate{}
	err := r.dbx.SelectContext(ctx, &templates, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user templates: %w", err)
	}

	return templates, nil
}

func (r *ControllerRepository) GetPublicTemplates(ctx context.Context) ([]*models.ControllerTemplate, error) {
	query := `
		SELECT id, name, description, category, template_json, variables_json,
			   thumbnail_url, usage_count, rating, is_public, user_id,
			   created_at, updated_at
		FROM controller_templates
		WHERE is_public = true
		ORDER BY usage_count DESC, rating DESC, updated_at DESC
	`

	templates := []*models.ControllerTemplate{}
	err := r.dbx.SelectContext(ctx, &templates, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get public templates: %w", err)
	}

	return templates, nil
}

func (r *ControllerRepository) UpdateTemplate(ctx context.Context, template *models.ControllerTemplate) error {
	query := `
		UPDATE controller_templates 
		SET name = ?, description = ?, category = ?, template_json = ?, 
			variables_json = ?, thumbnail_url = ?, rating = ?, is_public = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		template.Name,
		template.Description,
		template.Category,
		template.TemplateJSON,
		template.VariablesJSON,
		template.ThumbnailURL,
		template.Rating,
		template.IsPublic,
		template.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	return nil
}

func (r *ControllerRepository) DeleteTemplate(ctx context.Context, id int) error {
	query := `DELETE FROM controller_templates WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

func (r *ControllerRepository) IncrementTemplateUsage(ctx context.Context, id int) error {
	query := `UPDATE controller_templates SET usage_count = usage_count + 1 WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment template usage: %w", err)
	}

	return nil
}

// Sharing operations

func (r *ControllerRepository) CreateShare(ctx context.Context, share *models.ControllerShare) error {
	query := `
		INSERT INTO controller_shares (
			dashboard_id, user_id, permissions, shared_by, expires_at
		) VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		share.DashboardID,
		share.UserID,
		share.Permissions,
		share.SharedBy,
		share.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get share ID: %w", err)
	}

	share.ID = int(id)
	share.CreatedAt = time.Now()

	return nil
}

func (r *ControllerRepository) GetSharesByDashboardID(ctx context.Context, dashboardID int) ([]*models.ControllerShare, error) {
	query := `
		SELECT id, dashboard_id, user_id, permissions, shared_by, expires_at, created_at
		FROM controller_shares
		WHERE dashboard_id = ?
		ORDER BY created_at DESC
	`

	shares := []*models.ControllerShare{}
	err := r.dbx.SelectContext(ctx, &shares, query, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard shares: %w", err)
	}

	return shares, nil
}

func (r *ControllerRepository) GetSharesByUserID(ctx context.Context, userID int) ([]*models.ControllerShare, error) {
	query := `
		SELECT id, dashboard_id, user_id, permissions, shared_by, expires_at, created_at
		FROM controller_shares
		WHERE user_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY created_at DESC
	`

	shares := []*models.ControllerShare{}
	err := r.dbx.SelectContext(ctx, &shares, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user shares: %w", err)
	}

	return shares, nil
}

func (r *ControllerRepository) UpdateSharePermissions(ctx context.Context, id int, permissions string) error {
	query := `UPDATE controller_shares SET permissions = ? WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, permissions, id)
	if err != nil {
		return fmt.Errorf("failed to update share permissions: %w", err)
	}

	return nil
}

func (r *ControllerRepository) DeleteShare(ctx context.Context, id int) error {
	query := `DELETE FROM controller_shares WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete share: %w", err)
	}

	return nil
}

func (r *ControllerRepository) CheckUserAccess(ctx context.Context, dashboardID int, userID int) (string, error) {
	// Check if user owns the dashboard
	ownerQuery := `SELECT 1 FROM controller_dashboards WHERE id = ? AND user_id = ?`
	var owner int
	err := r.db.QueryRowContext(ctx, ownerQuery, dashboardID, userID).Scan(&owner)
	if err == nil {
		return "admin", nil
	}

	// Check if dashboard is shared with user
	shareQuery := `
		SELECT permissions 
		FROM controller_shares 
		WHERE dashboard_id = ? AND user_id = ? 
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`
	var permissions string
	err = r.db.QueryRowContext(ctx, shareQuery, dashboardID, userID).Scan(&permissions)
	if err == nil {
		return permissions, nil
	}

	// Check if dashboard is public
	publicQuery := `
		SELECT JSON_EXTRACT(access_config, '$.public') 
		FROM controller_dashboards 
		WHERE id = ?
	`
	var isPublic bool
	err = r.db.QueryRowContext(ctx, publicQuery, dashboardID).Scan(&isPublic)
	if err == nil && isPublic {
		return "view", nil
	}

	return "", fmt.Errorf("access denied")
}

// Usage analytics

func (r *ControllerRepository) LogUsage(ctx context.Context, log *models.ControllerUsageLog) error {
	query := `
		INSERT INTO controller_usage_logs (
			dashboard_id, user_id, action, element_id, element_type,
			session_id, ip_address, user_agent, duration_ms, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.DashboardID,
		log.UserID,
		log.Action,
		log.ElementID,
		log.ElementType,
		log.SessionID,
		log.IPAddress,
		log.UserAgent,
		log.DurationMS,
		log.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to log usage: %w", err)
	}

	return nil
}

func (r *ControllerRepository) GetUsageStats(ctx context.Context, dashboardID int, timeRange string) (map[string]interface{}, error) {
	var timeFilter string
	switch timeRange {
	case "hour":
		timeFilter = "created_at >= datetime('now', '-1 hour')"
	case "day":
		timeFilter = "created_at >= datetime('now', '-1 day')"
	case "week":
		timeFilter = "created_at >= datetime('now', '-7 days')"
	case "month":
		timeFilter = "created_at >= datetime('now', '-30 days')"
	default:
		timeFilter = "1=1" // All time
	}

	stats := make(map[string]interface{})

	// Total views
	var totalViews int
	viewQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM controller_usage_logs 
		WHERE dashboard_id = ? AND action = 'view' AND %s
	`, timeFilter)
	err := r.db.QueryRowContext(ctx, viewQuery, dashboardID).Scan(&totalViews)
	if err != nil {
		return nil, fmt.Errorf("failed to get total views: %w", err)
	}
	stats["total_views"] = totalViews

	// Unique users
	var uniqueUsers int
	userQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT user_id) FROM controller_usage_logs 
		WHERE dashboard_id = ? AND %s AND user_id IS NOT NULL
	`, timeFilter)
	err = r.db.QueryRowContext(ctx, userQuery, dashboardID).Scan(&uniqueUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique users: %w", err)
	}
	stats["unique_users"] = uniqueUsers

	// Element interactions
	elementQuery := fmt.Sprintf(`
		SELECT element_type, COUNT(*) as count
		FROM controller_usage_logs 
		WHERE dashboard_id = ? AND action = 'element_action' AND %s
		GROUP BY element_type
	`, timeFilter)

	rows, err := r.db.QueryContext(ctx, elementQuery, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get element stats: %w", err)
	}
	defer rows.Close()

	elementStats := make(map[string]int)
	for rows.Next() {
		var elementType string
		var count int
		err := rows.Scan(&elementType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan element stats: %w", err)
		}
		elementStats[elementType] = count
	}
	stats["element_interactions"] = elementStats

	return stats, nil
}

func (r *ControllerRepository) GetDashboardAnalytics(ctx context.Context, userID *int) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var userFilter string
	var args []interface{}
	if userID != nil {
		userFilter = "WHERE d.user_id = ?"
		args = append(args, *userID)
	} else {
		userFilter = ""
	}

	// Dashboard count
	var dashboardCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM controller_dashboards d %s", userFilter)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&dashboardCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard count: %w", err)
	}
	analytics["dashboard_count"] = dashboardCount

	// Most viewed dashboards
	viewQuery := fmt.Sprintf(`
		SELECT d.id, d.name, COUNT(l.id) as views
		FROM controller_dashboards d
		LEFT JOIN controller_usage_logs l ON d.id = l.dashboard_id AND l.action = 'view'
		%s
		GROUP BY d.id, d.name
		ORDER BY views DESC
		LIMIT 10
	`, userFilter)

	rows, err := r.db.QueryContext(ctx, viewQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get most viewed dashboards: %w", err)
	}
	defer rows.Close()

	type DashboardView struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Views int    `json:"views"`
	}

	var mostViewed []DashboardView
	for rows.Next() {
		var dv DashboardView
		err := rows.Scan(&dv.ID, &dv.Name, &dv.Views)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dashboard views: %w", err)
		}
		mostViewed = append(mostViewed, dv)
	}
	analytics["most_viewed"] = mostViewed

	return analytics, nil
}

func (r *ControllerRepository) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	query := `
		DELETE FROM controller_usage_logs 
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`

	_, err := r.db.ExecContext(ctx, query, retentionDays)
	if err != nil {
		return fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	return nil
}

// Import/Export

func (r *ControllerRepository) ExportDashboard(ctx context.Context, id int) (map[string]interface{}, error) {
	dashboard, err := r.GetDashboardByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard for export: %w", err)
	}

	export := map[string]interface{}{
		"version":     "1.0",
		"exported_at": time.Now().UTC(),
		"dashboard": map[string]interface{}{
			"name":        dashboard.Name,
			"description": dashboard.Description,
			"category":    dashboard.Category,
			"layout":      json.RawMessage(dashboard.LayoutConfig),
			"elements":    json.RawMessage(dashboard.ElementsJSON),
			"style":       json.RawMessage(dashboard.StyleConfig),
			"access":      json.RawMessage(dashboard.AccessConfig),
			"tags":        json.RawMessage(dashboard.Tags),
		},
	}

	return export, nil
}

func (r *ControllerRepository) ImportDashboard(ctx context.Context, data map[string]interface{}, userID *int) (*models.ControllerDashboard, error) {
	dashboardData, ok := data["dashboard"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dashboard data structure")
	}

	// Convert interface{} to JSON strings
	layoutJSON, _ := json.Marshal(dashboardData["layout"])
	elementsJSON, _ := json.Marshal(dashboardData["elements"])
	styleJSON, _ := json.Marshal(dashboardData["style"])
	accessJSON, _ := json.Marshal(dashboardData["access"])
	tagsJSON, _ := json.Marshal(dashboardData["tags"])

	dashboard := &models.ControllerDashboard{
		Name:         dashboardData["name"].(string),
		Description:  dashboardData["description"].(string),
		Category:     dashboardData["category"].(string),
		LayoutConfig: string(layoutJSON),
		ElementsJSON: string(elementsJSON),
		StyleConfig:  string(styleJSON),
		AccessConfig: string(accessJSON),
		Tags:         string(tagsJSON),
		IsFavorite:   false,
		Version:      1,
		UserID:       userID,
	}

	err := r.CreateDashboard(ctx, dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to import dashboard: %w", err)
	}

	return dashboard, nil
}
