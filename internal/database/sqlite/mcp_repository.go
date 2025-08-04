package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// MCPRepository implements repositories.MCPRepository
type MCPRepository struct {
	db *sql.DB
}

// NewMCPRepository creates a new MCPRepository
func NewMCPRepository(db *sql.DB) repositories.MCPRepository {
	return &MCPRepository{db: db}
}

// CreateTool creates a new MCP tool
func (r *MCPRepository) CreateTool(ctx context.Context, tool interface{}) error {
	mcpTool, ok := tool.(*ai.MCPTool)
	if !ok {
		return fmt.Errorf("invalid MCP tool type")
	}
	query := `
		INSERT INTO mcp_tools (id, name, description, schema, handler, category, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	schemaJSON, err := json.Marshal(mcpTool.Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	now := time.Now()
	mcpTool.CreatedAt = now
	mcpTool.UpdatedAt = now

	_, err = r.db.ExecContext(
		ctx,
		query,
		mcpTool.ID,
		mcpTool.Name,
		mcpTool.Description,
		string(schemaJSON),
		mcpTool.Handler,
		mcpTool.Category,
		mcpTool.Enabled,
		mcpTool.CreatedAt,
		mcpTool.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create tool: %w", err)
	}

	return nil
}

// GetTool retrieves a tool by ID
func (r *MCPRepository) GetTool(ctx context.Context, id string) (interface{}, error) {
	query := `
		SELECT id, name, description, schema, handler, category, enabled, usage_count, last_used, created_at, updated_at
		FROM mcp_tools
		WHERE id = ?
	`

	tool := &ai.MCPTool{}
	var schemaJSON string
	var lastUsed sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tool.ID,
		&tool.Name,
		&tool.Description,
		&schemaJSON,
		&tool.Handler,
		&tool.Category,
		&tool.Enabled,
		&tool.UsageCount,
		&lastUsed,
		&tool.CreatedAt,
		&tool.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tool not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tool: %w", err)
	}

	if lastUsed.Valid {
		tool.LastUsed = &lastUsed.Time
	}

	if err := json.Unmarshal([]byte(schemaJSON), &tool.Schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return interface{}(tool), nil
}

// GetToolByName retrieves a tool by name
func (r *MCPRepository) GetToolByName(ctx context.Context, name string) (interface{}, error) {
	query := `
		SELECT id, name, description, schema, handler, category, enabled, usage_count, last_used, created_at, updated_at
		FROM mcp_tools
		WHERE name = ?
	`

	tool := &ai.MCPTool{}
	var schemaJSON string
	var lastUsed sql.NullTime

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&tool.ID,
		&tool.Name,
		&tool.Description,
		&schemaJSON,
		&tool.Handler,
		&tool.Category,
		&tool.Enabled,
		&tool.UsageCount,
		&lastUsed,
		&tool.CreatedAt,
		&tool.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tool not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tool: %w", err)
	}

	if lastUsed.Valid {
		tool.LastUsed = &lastUsed.Time
	}

	if err := json.Unmarshal([]byte(schemaJSON), &tool.Schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return interface{}(tool), nil
}

// GetTools retrieves tools with filtering
func (r *MCPRepository) GetTools(ctx context.Context, filter interface{}) ([]interface{}, error) {
	mcpFilter, ok := filter.(*ai.MCPToolFilter)
	if !ok {
		return nil, fmt.Errorf("invalid MCP tool filter type")
	}
	query := `
		SELECT id, name, description, schema, handler, category, enabled, usage_count, last_used, created_at, updated_at
		FROM mcp_tools
	`

	var conditions []string
	var args []interface{}

	if mcpFilter != nil {
		if mcpFilter.Category != nil {
			conditions = append(conditions, "category = ?")
			args = append(args, *mcpFilter.Category)
		}

		if mcpFilter.Enabled != nil {
			conditions = append(conditions, "enabled = ?")
			args = append(args, *mcpFilter.Enabled)
		}

		if mcpFilter.SearchQuery != nil && *mcpFilter.SearchQuery != "" {
			conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
			searchTerm := "%" + *mcpFilter.SearchQuery + "%"
			args = append(args, searchTerm, searchTerm)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	orderBy := "name"
	orderDir := "ASC"
	if mcpFilter != nil {
		if mcpFilter.OrderBy != "" {
			orderBy = mcpFilter.OrderBy
		}
		if mcpFilter.OrderDir != "" {
			orderDir = mcpFilter.OrderDir
		}
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Add pagination
	if filter != nil && mcpFilter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, mcpFilter.Limit)

		if mcpFilter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, mcpFilter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tools: %w", err)
	}
	defer rows.Close()

	var tools []*ai.MCPTool
	for rows.Next() {
		tool := &ai.MCPTool{}
		var schemaJSON string
		var lastUsed sql.NullTime

		err := rows.Scan(
			&tool.ID,
			&tool.Name,
			&tool.Description,
			&schemaJSON,
			&tool.Handler,
			&tool.Category,
			&tool.Enabled,
			&tool.UsageCount,
			&lastUsed,
			&tool.CreatedAt,
			&tool.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool: %w", err)
		}

		if lastUsed.Valid {
			tool.LastUsed = &lastUsed.Time
		}

		if err := json.Unmarshal([]byte(schemaJSON), &tool.Schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
		}

		tools = append(tools, tool)
	}

	// Convert to []interface{}
	result := make([]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = tool
	}
	return result, nil
}

// GetToolCount returns the count of tools matching the filter
func (r *MCPRepository) GetToolCount(ctx context.Context, filter interface{}) (int, error) {
	mcpFilter, ok := filter.(*ai.MCPToolFilter)
	if !ok {
		return 0, fmt.Errorf("invalid MCP tool filter type")
	}
	query := "SELECT COUNT(*) FROM mcp_tools"

	var conditions []string
	var args []interface{}

	if mcpFilter != nil {
		if mcpFilter.Category != nil {
			conditions = append(conditions, "category = ?")
			args = append(args, *mcpFilter.Category)
		}

		if mcpFilter.Enabled != nil {
			conditions = append(conditions, "enabled = ?")
			args = append(args, *mcpFilter.Enabled)
		}

		if mcpFilter.SearchQuery != nil && *mcpFilter.SearchQuery != "" {
			conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
			searchTerm := "%" + *mcpFilter.SearchQuery + "%"
			args = append(args, searchTerm, searchTerm)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tools: %w", err)
	}

	return count, nil
}

// GetAllTools retrieves all tools from the database
func (r *MCPRepository) GetAllTools(ctx context.Context) ([]interface{}, error) {
	// Use GetTools with no filter to get all tools
	return r.GetTools(ctx, &ai.MCPToolFilter{})
}

// GetEnabledTools retrieves enabled tools, optionally filtered by category
func (r *MCPRepository) GetEnabledTools(ctx context.Context, category string) ([]interface{}, error) {
	filter := &ai.MCPToolFilter{
		Enabled:  &[]bool{true}[0], // Get pointer to true
		OrderBy:  "name",
		OrderDir: "ASC",
	}

	if category != "" {
		filter.Category = &category
	}

	return r.GetTools(ctx, filter)
}

// UpdateTool updates a tool
func (r *MCPRepository) UpdateTool(ctx context.Context, tool interface{}) error {
	mcpTool, ok := tool.(*ai.MCPTool)
	if !ok {
		return fmt.Errorf("invalid MCP tool type")
	}
	query := `
		UPDATE mcp_tools
		SET name = ?, description = ?, schema = ?, handler = ?, category = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`

	schemaJSON, err := json.Marshal(mcpTool.Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	mcpTool.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		mcpTool.Name,
		mcpTool.Description,
		string(schemaJSON),
		mcpTool.Handler,
		mcpTool.Category,
		mcpTool.Enabled,
		mcpTool.UpdatedAt,
		mcpTool.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tool not found")
	}

	return nil
}

// DeleteTool deletes a tool
func (r *MCPRepository) DeleteTool(ctx context.Context, id string) error {
	query := "DELETE FROM mcp_tools WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tool not found")
	}

	return nil
}

// EnableTool enables a tool
func (r *MCPRepository) EnableTool(ctx context.Context, id string) error {
	query := "UPDATE mcp_tools SET enabled = TRUE, updated_at = ? WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to enable tool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tool not found")
	}

	return nil
}

// DisableTool disables a tool
func (r *MCPRepository) DisableTool(ctx context.Context, id string) error {
	query := "UPDATE mcp_tools SET enabled = FALSE, updated_at = ? WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to disable tool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tool not found")
	}

	return nil
}

// CreateToolExecution creates a new tool execution record
func (r *MCPRepository) CreateToolExecution(ctx context.Context, execution interface{}) error {
	mcpExecution, ok := execution.(*ai.MCPToolExecution)
	if !ok {
		return fmt.Errorf("invalid tool execution type")
	}
	query := `
		INSERT INTO mcp_tool_executions (id, conversation_id, message_id, tool_id, tool_name, 
		                                parameters, result, error, execution_time_ms, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	parametersJSON, err := json.Marshal(mcpExecution.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	mcpExecution.CreatedAt = time.Now()

	_, err = r.db.ExecContext(
		ctx,
		query,
		mcpExecution.ID,
		mcpExecution.ConversationID,
		mcpExecution.MessageID,
		mcpExecution.ToolID,
		mcpExecution.ToolName,
		string(parametersJSON),
		mcpExecution.Result,
		mcpExecution.Error,
		mcpExecution.ExecutionTimeMs,
		mcpExecution.Success,
		mcpExecution.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create tool execution: %w", err)
	}

	return nil
}

// GetToolExecution retrieves a tool execution by ID
func (r *MCPRepository) GetToolExecution(ctx context.Context, id string) (interface{}, error) {
	query := `
		SELECT id, conversation_id, message_id, tool_id, tool_name, parameters, result, error, 
		       execution_time_ms, success, created_at
		FROM mcp_tool_executions
		WHERE id = ?
	`

	execution := &ai.MCPToolExecution{}
	var parametersJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&execution.ID,
		&execution.ConversationID,
		&execution.MessageID,
		&execution.ToolID,
		&execution.ToolName,
		&parametersJSON,
		&execution.Result,
		&execution.Error,
		&execution.ExecutionTimeMs,
		&execution.Success,
		&execution.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tool execution not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tool execution: %w", err)
	}

	if err := json.Unmarshal([]byte(parametersJSON), &execution.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return interface{}(execution), nil
}

// GetToolExecutions retrieves tool executions for a conversation
func (r *MCPRepository) GetToolExecutions(ctx context.Context, conversationID string, limit int, offset int) ([]interface{}, error) {
	query := `
		SELECT id, conversation_id, message_id, tool_id, tool_name, parameters, result, error, 
		       execution_time_ms, success, created_at
		FROM mcp_tool_executions
		WHERE conversation_id = ?
		ORDER BY created_at DESC
	`

	args := []interface{}{conversationID}

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)

		if offset > 0 {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool executions: %w", err)
	}
	defer rows.Close()

	var executions []*ai.MCPToolExecution
	for rows.Next() {
		execution := &ai.MCPToolExecution{}
		var parametersJSON string

		err := rows.Scan(
			&execution.ID,
			&execution.ConversationID,
			&execution.MessageID,
			&execution.ToolID,
			&execution.ToolName,
			&parametersJSON,
			&execution.Result,
			&execution.Error,
			&execution.ExecutionTimeMs,
			&execution.Success,
			&execution.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool execution: %w", err)
		}

		if err := json.Unmarshal([]byte(parametersJSON), &execution.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		executions = append(executions, execution)
	}

	// Convert to []interface{}
	result := make([]interface{}, len(executions))
	for i, exec := range executions {
		result[i] = exec
	}
	return result, nil
}

// GetToolExecutionsByTool retrieves tool executions for a specific tool
func (r *MCPRepository) GetToolExecutionsByTool(ctx context.Context, toolID string, limit int, offset int) ([]interface{}, error) {
	query := `
		SELECT id, conversation_id, message_id, tool_id, tool_name, parameters, result, error, 
		       execution_time_ms, success, created_at
		FROM mcp_tool_executions
		WHERE tool_id = ?
		ORDER BY created_at DESC
	`

	args := []interface{}{toolID}

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)

		if offset > 0 {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool executions: %w", err)
	}
	defer rows.Close()

	var executions []*ai.MCPToolExecution
	for rows.Next() {
		execution := &ai.MCPToolExecution{}
		var parametersJSON string

		err := rows.Scan(
			&execution.ID,
			&execution.ConversationID,
			&execution.MessageID,
			&execution.ToolID,
			&execution.ToolName,
			&parametersJSON,
			&execution.Result,
			&execution.Error,
			&execution.ExecutionTimeMs,
			&execution.Success,
			&execution.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool execution: %w", err)
		}

		if err := json.Unmarshal([]byte(parametersJSON), &execution.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		executions = append(executions, execution)
	}

	// Convert to []interface{}
	result := make([]interface{}, len(executions))
	for i, exec := range executions {
		result[i] = exec
	}
	return result, nil
}

// UpdateToolExecution updates a tool execution
func (r *MCPRepository) UpdateToolExecution(ctx context.Context, execution interface{}) error {
	mcpExecution, ok := execution.(*ai.MCPToolExecution)
	if !ok {
		return fmt.Errorf("invalid MCP tool execution type")
	}
	query := `
		UPDATE mcp_tool_executions
		SET result = ?, error = ?, execution_time_ms = ?, success = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		mcpExecution.Result,
		mcpExecution.Error,
		mcpExecution.ExecutionTimeMs,
		mcpExecution.Success,
		mcpExecution.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tool execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tool execution not found")
	}

	return nil
}

// IncrementToolUsage increments the usage count for a tool
func (r *MCPRepository) IncrementToolUsage(ctx context.Context, toolID string) error {
	query := "UPDATE mcp_tools SET usage_count = usage_count + 1, last_used = ? WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, time.Now(), toolID)
	if err != nil {
		return fmt.Errorf("failed to increment tool usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tool not found")
	}

	return nil
}

// GetToolUsageStats retrieves tool usage statistics
func (r *MCPRepository) GetToolUsageStats(ctx context.Context, startDate, endDate time.Time) ([]interface{}, error) {
	query := `
		SELECT 
			t.name,
			t.category,
			COUNT(e.id) as usage_count,
			ROUND(AVG(CASE WHEN e.success THEN 1.0 ELSE 0.0 END) * 100, 2) as success_rate,
			AVG(e.execution_time_ms) as avg_exec_time
		FROM mcp_tools t
		LEFT JOIN mcp_tool_executions e ON t.id = e.tool_id 
			AND e.created_at >= ? AND e.created_at <= ?
		GROUP BY t.id, t.name, t.category
		HAVING usage_count > 0
		ORDER BY usage_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool usage stats: %w", err)
	}
	defer rows.Close()

	var stats []*ai.ToolUsageStats
	for rows.Next() {
		stat := &ai.ToolUsageStats{}

		err := rows.Scan(
			&stat.ToolName,
			&stat.Category,
			&stat.UsageCount,
			&stat.SuccessRate,
			&stat.AvgExecTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool usage stat: %w", err)
		}

		stats = append(stats, stat)
	}

	// Convert to []interface{}
	result := make([]interface{}, len(stats))
	for i, stat := range stats {
		result[i] = stat
	}
	return result, nil
}

// GetToolSuccessRate retrieves the success rate for a tool over the last N days
func (r *MCPRepository) GetToolSuccessRate(ctx context.Context, toolID string, days int) (float64, error) {
	query := `
		SELECT AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) * 100
		FROM mcp_tool_executions
		WHERE tool_id = ? AND created_at >= ?
	`

	cutoffDate := time.Now().AddDate(0, 0, -days)

	var successRate sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, toolID, cutoffDate).Scan(&successRate)
	if err != nil {
		return 0, fmt.Errorf("failed to get tool success rate: %w", err)
	}

	if successRate.Valid {
		return successRate.Float64, nil
	}

	return 0, nil
}

// GetMostUsedTools retrieves the most used tools
func (r *MCPRepository) GetMostUsedTools(ctx context.Context, limit int, days int) ([]interface{}, error) {
	query := `
		SELECT t.id, t.name, t.description, t.schema, t.handler, t.category, t.enabled, 
		       t.usage_count, t.last_used, t.created_at, t.updated_at
		FROM mcp_tools t
		WHERE t.last_used >= ?
		ORDER BY t.usage_count DESC
		LIMIT ?
	`

	cutoffDate := time.Now().AddDate(0, 0, -days)

	rows, err := r.db.QueryContext(ctx, query, cutoffDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query most used tools: %w", err)
	}
	defer rows.Close()

	var tools []*ai.MCPTool
	for rows.Next() {
		tool := &ai.MCPTool{}
		var schemaJSON string
		var lastUsed sql.NullTime

		err := rows.Scan(
			&tool.ID,
			&tool.Name,
			&tool.Description,
			&schemaJSON,
			&tool.Handler,
			&tool.Category,
			&tool.Enabled,
			&tool.UsageCount,
			&lastUsed,
			&tool.CreatedAt,
			&tool.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool: %w", err)
		}

		if lastUsed.Valid {
			tool.LastUsed = &lastUsed.Time
		}

		if err := json.Unmarshal([]byte(schemaJSON), &tool.Schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
		}

		tools = append(tools, tool)
	}

	// Convert to []interface{}
	result := make([]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = tool
	}
	return result, nil
}

// GetRecentToolExecutions retrieves recent tool executions
func (r *MCPRepository) GetRecentToolExecutions(ctx context.Context, limit int) ([]interface{}, error) {
	query := `
		SELECT id, conversation_id, message_id, tool_id, tool_name, parameters, result, error, 
		       execution_time_ms, success, created_at
		FROM mcp_tool_executions
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent tool executions: %w", err)
	}
	defer rows.Close()

	var executions []*ai.MCPToolExecution
	for rows.Next() {
		execution := &ai.MCPToolExecution{}
		var parametersJSON string

		err := rows.Scan(
			&execution.ID,
			&execution.ConversationID,
			&execution.MessageID,
			&execution.ToolID,
			&execution.ToolName,
			&parametersJSON,
			&execution.Result,
			&execution.Error,
			&execution.ExecutionTimeMs,
			&execution.Success,
			&execution.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool execution: %w", err)
		}

		if err := json.Unmarshal([]byte(parametersJSON), &execution.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		executions = append(executions, execution)
	}

	// Convert to []interface{}
	result := make([]interface{}, len(executions))
	for i, exec := range executions {
		result[i] = exec
	}
	return result, nil
}

// CleanupOldExecutions removes tool executions older than specified days
func (r *MCPRepository) CleanupOldExecutions(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	query := "DELETE FROM mcp_tool_executions WHERE created_at < ?"

	result, err := r.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to cleanup old tool executions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old tool executions\n", rowsAffected)
	return nil
}
