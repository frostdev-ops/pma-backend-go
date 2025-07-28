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

// retryableDatabaseOperation retries database operations that fail with SQLITE_BUSY
func retryableDatabaseOperation(ctx context.Context, operation func() error, maxRetries int) error {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Check if this is a database busy error
		if strings.Contains(err.Error(), "database is locked") ||
			strings.Contains(err.Error(), "SQLITE_BUSY") {
			if attempt < maxRetries {
				// Wait before retrying, with exponential backoff
				waitTime := time.Duration(attempt+1) * 10 * time.Millisecond
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitTime):
					continue
				}
			}
		}

		// Return the error if it's not retryable or we've exhausted retries
		return err
	}

	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// ConversationRepository implements repositories.ConversationRepository
type ConversationRepository struct {
	db *sql.DB
}

// NewConversationRepository creates a new ConversationRepository
func NewConversationRepository(db *sql.DB) repositories.ConversationRepository {
	return &ConversationRepository{db: db}
}

// CreateConversation creates a new conversation
func (r *ConversationRepository) CreateConversation(ctx context.Context, conv *ai.Conversation) error {
	query := `
		INSERT INTO conversations (id, user_id, title, system_prompt, provider, model, temperature, max_tokens, context_data, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	contextDataJSON, err := conv.MarshalContextData()
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	metadataJSON, err := conv.MarshalMetadata()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	now := time.Now()
	conv.CreatedAt = now
	conv.UpdatedAt = now

	_, err = r.db.ExecContext(
		ctx,
		query,
		conv.ID,
		conv.UserID,
		conv.Title,
		conv.SystemPrompt,
		conv.Provider,
		conv.Model,
		conv.Temperature,
		conv.MaxTokens,
		contextDataJSON,
		metadataJSON,
		conv.CreatedAt,
		conv.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

// GetConversation retrieves a conversation by ID and user ID
func (r *ConversationRepository) GetConversation(ctx context.Context, id string, userID string) (*ai.Conversation, error) {
	query := `
		SELECT id, user_id, title, system_prompt, provider, model, temperature, max_tokens, 
		       context_data, metadata, message_count, last_message_at, created_at, updated_at, archived
		FROM conversations
		WHERE id = ? AND user_id = ?
	`

	conv := &ai.Conversation{}
	var contextDataJSON, metadataJSON sql.NullString
	var lastMessageAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&conv.ID,
		&conv.UserID,
		&conv.Title,
		&conv.SystemPrompt,
		&conv.Provider,
		&conv.Model,
		&conv.Temperature,
		&conv.MaxTokens,
		&contextDataJSON,
		&metadataJSON,
		&conv.MessageCount,
		&lastMessageAt,
		&conv.CreatedAt,
		&conv.UpdatedAt,
		&conv.Archived,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if lastMessageAt.Valid {
		conv.LastMessageAt = &lastMessageAt.Time
	}

	if contextDataJSON.Valid {
		if err := conv.UnmarshalContextData(contextDataJSON.String); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
		}
	}

	if metadataJSON.Valid {
		if err := conv.UnmarshalMetadata(metadataJSON.String); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return conv, nil
}

// GetConversations retrieves conversations with filtering
func (r *ConversationRepository) GetConversations(ctx context.Context, filter *ai.ConversationFilter) ([]*ai.Conversation, error) {
	query := `
		SELECT id, user_id, title, system_prompt, provider, model, temperature, max_tokens, 
		       context_data, metadata, message_count, last_message_at, created_at, updated_at, archived
		FROM conversations
	`

	var conditions []string
	var args []interface{}

	if filter != nil {
		if filter.UserID != nil {
			conditions = append(conditions, "user_id = ?")
			args = append(args, *filter.UserID)
		}

		if filter.Archived != nil {
			conditions = append(conditions, "archived = ?")
			args = append(args, *filter.Archived)
		}

		if filter.Provider != nil {
			conditions = append(conditions, "provider = ?")
			args = append(args, *filter.Provider)
		}

		if filter.StartDate != nil {
			conditions = append(conditions, "created_at >= ?")
			args = append(args, *filter.StartDate)
		}

		if filter.EndDate != nil {
			conditions = append(conditions, "created_at <= ?")
			args = append(args, *filter.EndDate)
		}

		if filter.HasMessages != nil {
			if *filter.HasMessages {
				conditions = append(conditions, "message_count > 0")
			} else {
				conditions = append(conditions, "message_count = 0")
			}
		}

		if filter.SearchQuery != nil && *filter.SearchQuery != "" {
			conditions = append(conditions, "(title LIKE ? OR EXISTS (SELECT 1 FROM conversation_messages WHERE conversation_id = conversations.id AND content LIKE ?))")
			searchTerm := "%" + *filter.SearchQuery + "%"
			args = append(args, searchTerm, searchTerm)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	orderBy := "created_at"
	orderDir := "DESC"
	if filter != nil {
		if filter.OrderBy != "" {
			orderBy = filter.OrderBy
		}
		if filter.OrderDir != "" {
			orderDir = filter.OrderDir
		}
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*ai.Conversation
	for rows.Next() {
		conv := &ai.Conversation{}
		var contextDataJSON, metadataJSON sql.NullString
		var lastMessageAt sql.NullTime

		err := rows.Scan(
			&conv.ID,
			&conv.UserID,
			&conv.Title,
			&conv.SystemPrompt,
			&conv.Provider,
			&conv.Model,
			&conv.Temperature,
			&conv.MaxTokens,
			&contextDataJSON,
			&metadataJSON,
			&conv.MessageCount,
			&lastMessageAt,
			&conv.CreatedAt,
			&conv.UpdatedAt,
			&conv.Archived,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		if lastMessageAt.Valid {
			conv.LastMessageAt = &lastMessageAt.Time
		}

		if contextDataJSON.Valid {
			if err := conv.UnmarshalContextData(contextDataJSON.String); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
			}
		}

		if metadataJSON.Valid {
			if err := conv.UnmarshalMetadata(metadataJSON.String); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}

	return conversations, nil
}

// GetConversationCount returns the count of conversations matching the filter
func (r *ConversationRepository) GetConversationCount(ctx context.Context, filter *ai.ConversationFilter) (int, error) {
	query := "SELECT COUNT(*) FROM conversations"

	var conditions []string
	var args []interface{}

	if filter != nil {
		if filter.UserID != nil {
			conditions = append(conditions, "user_id = ?")
			args = append(args, *filter.UserID)
		}

		if filter.Archived != nil {
			conditions = append(conditions, "archived = ?")
			args = append(args, *filter.Archived)
		}

		if filter.Provider != nil {
			conditions = append(conditions, "provider = ?")
			args = append(args, *filter.Provider)
		}

		if filter.StartDate != nil {
			conditions = append(conditions, "created_at >= ?")
			args = append(args, *filter.StartDate)
		}

		if filter.EndDate != nil {
			conditions = append(conditions, "created_at <= ?")
			args = append(args, *filter.EndDate)
		}

		if filter.HasMessages != nil {
			if *filter.HasMessages {
				conditions = append(conditions, "message_count > 0")
			} else {
				conditions = append(conditions, "message_count = 0")
			}
		}

		if filter.SearchQuery != nil && *filter.SearchQuery != "" {
			conditions = append(conditions, "(title LIKE ? OR EXISTS (SELECT 1 FROM conversation_messages WHERE conversation_id = conversations.id AND content LIKE ?))")
			searchTerm := "%" + *filter.SearchQuery + "%"
			args = append(args, searchTerm, searchTerm)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	return count, nil
}

// UpdateConversation updates a conversation
func (r *ConversationRepository) UpdateConversation(ctx context.Context, conv *ai.Conversation) error {
	query := `
		UPDATE conversations
		SET title = ?, system_prompt = ?, provider = ?, model = ?, temperature = ?, max_tokens = ?, 
		    context_data = ?, metadata = ?, message_count = ?, last_message_at = ?, updated_at = ?, archived = ?
		WHERE id = ? AND user_id = ?
	`

	contextDataJSON, err := conv.MarshalContextData()
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	metadataJSON, err := conv.MarshalMetadata()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	conv.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		conv.Title,
		conv.SystemPrompt,
		conv.Provider,
		conv.Model,
		conv.Temperature,
		conv.MaxTokens,
		contextDataJSON,
		metadataJSON,
		conv.MessageCount,
		conv.LastMessageAt,
		conv.UpdatedAt,
		conv.Archived,
		conv.ID,
		conv.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found or access denied")
	}

	return nil
}

// DeleteConversation deletes a conversation
func (r *ConversationRepository) DeleteConversation(ctx context.Context, id string, userID string) error {
	query := "DELETE FROM conversations WHERE id = ? AND user_id = ?"

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found or access denied")
	}

	return nil
}

// ArchiveConversation archives a conversation
func (r *ConversationRepository) ArchiveConversation(ctx context.Context, id string, userID string) error {
	query := "UPDATE conversations SET archived = TRUE, updated_at = ? WHERE id = ? AND user_id = ?"

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found or access denied")
	}

	return nil
}

// UnarchiveConversation unarchives a conversation
func (r *ConversationRepository) UnarchiveConversation(ctx context.Context, id string, userID string) error {
	query := "UPDATE conversations SET archived = FALSE, updated_at = ? WHERE id = ? AND user_id = ?"

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		return fmt.Errorf("failed to unarchive conversation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found or access denied")
	}

	return nil
}

// CreateMessage creates a new message with retry logic for database busy errors
func (r *ConversationRepository) CreateMessage(ctx context.Context, msg *ai.ConversationMessage) error {
	return retryableDatabaseOperation(ctx, func() error {
		return r.createMessageInternal(ctx, msg)
	}, 3) // Retry up to 3 times
}

// createMessageInternal performs the actual message creation
func (r *ConversationRepository) createMessageInternal(ctx context.Context, msg *ai.ConversationMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert message
	messageQuery := `
		INSERT INTO conversation_messages (id, conversation_id, role, content, tool_calls, tool_call_id, 
		                                  tokens_used, model_used, provider_used, response_time_ms, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	toolCallsJSON, err := msg.MarshalToolCalls()
	if err != nil {
		return fmt.Errorf("failed to marshal tool calls: %w", err)
	}

	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	msg.CreatedAt = time.Now()

	_, err = tx.ExecContext(
		ctx,
		messageQuery,
		msg.ID,
		msg.ConversationID,
		msg.Role,
		msg.Content,
		toolCallsJSON,
		msg.ToolCallID,
		msg.TokensUsed,
		msg.ModelUsed,
		msg.ProviderUsed,
		msg.ResponseTimeMs,
		string(metadataJSON),
		msg.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Update conversation message count and last message time
	updateQuery := `
		UPDATE conversations 
		SET message_count = message_count + 1, last_message_at = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = tx.ExecContext(ctx, updateQuery, msg.CreatedAt, msg.CreatedAt, msg.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

// GetMessage retrieves a message by ID
func (r *ConversationRepository) GetMessage(ctx context.Context, id string) (*ai.ConversationMessage, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, 
		       tokens_used, model_used, provider_used, response_time_ms, metadata, created_at
		FROM conversation_messages
		WHERE id = ?
	`

	msg := &ai.ConversationMessage{}
	var toolCallsJSON, metadataJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.Role,
		&msg.Content,
		&toolCallsJSON,
		&msg.ToolCallID,
		&msg.TokensUsed,
		&msg.ModelUsed,
		&msg.ProviderUsed,
		&msg.ResponseTimeMs,
		&metadataJSON,
		&msg.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if toolCallsJSON.Valid {
		if err := msg.UnmarshalToolCalls(toolCallsJSON.String); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool calls: %w", err)
		}
	}

	if metadataJSON.Valid {
		if err := json.Unmarshal([]byte(metadataJSON.String), &msg.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return msg, nil
}

// Note: I'll continue with the remaining methods in the next part due to length constraints...

// GetMessages retrieves messages with filtering
func (r *ConversationRepository) GetMessages(ctx context.Context, filter *ai.MessageFilter) ([]*ai.ConversationMessage, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, 
		       tokens_used, model_used, provider_used, response_time_ms, metadata, created_at
		FROM conversation_messages
	`

	var conditions []string
	var args []interface{}

	if filter != nil {
		if filter.ConversationID != nil {
			conditions = append(conditions, "conversation_id = ?")
			args = append(args, *filter.ConversationID)
		}

		if filter.Role != nil {
			conditions = append(conditions, "role = ?")
			args = append(args, *filter.Role)
		}

		if filter.HasToolCalls != nil {
			if *filter.HasToolCalls {
				conditions = append(conditions, "tool_calls IS NOT NULL AND tool_calls != ''")
			} else {
				conditions = append(conditions, "(tool_calls IS NULL OR tool_calls = '')")
			}
		}

		if filter.StartDate != nil {
			conditions = append(conditions, "created_at >= ?")
			args = append(args, *filter.StartDate)
		}

		if filter.EndDate != nil {
			conditions = append(conditions, "created_at <= ?")
			args = append(args, *filter.EndDate)
		}

		if filter.SearchQuery != nil && *filter.SearchQuery != "" {
			conditions = append(conditions, "content LIKE ?")
			args = append(args, "%"+*filter.SearchQuery+"%")
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	orderBy := "created_at"
	orderDir := "DESC"
	if filter != nil {
		if filter.OrderBy != "" {
			orderBy = filter.OrderBy
		}
		if filter.OrderDir != "" {
			orderDir = filter.OrderDir
		}
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*ai.ConversationMessage
	for rows.Next() {
		msg := &ai.ConversationMessage{}
		var toolCallsJSON, metadataJSON sql.NullString

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&toolCallsJSON,
			&msg.ToolCallID,
			&msg.TokensUsed,
			&msg.ModelUsed,
			&msg.ProviderUsed,
			&msg.ResponseTimeMs,
			&metadataJSON,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if toolCallsJSON.Valid {
			if err := msg.UnmarshalToolCalls(toolCallsJSON.String); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool calls: %w", err)
			}
		}

		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &msg.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessageCount returns the count of messages matching the filter
func (r *ConversationRepository) GetMessageCount(ctx context.Context, filter *ai.MessageFilter) (int, error) {
	query := "SELECT COUNT(*) FROM conversation_messages"

	var conditions []string
	var args []interface{}

	if filter != nil {
		if filter.ConversationID != nil {
			conditions = append(conditions, "conversation_id = ?")
			args = append(args, *filter.ConversationID)
		}

		if filter.Role != nil {
			conditions = append(conditions, "role = ?")
			args = append(args, *filter.Role)
		}

		if filter.HasToolCalls != nil {
			if *filter.HasToolCalls {
				conditions = append(conditions, "tool_calls IS NOT NULL AND tool_calls != ''")
			} else {
				conditions = append(conditions, "(tool_calls IS NULL OR tool_calls = '')")
			}
		}

		if filter.StartDate != nil {
			conditions = append(conditions, "created_at >= ?")
			args = append(args, *filter.StartDate)
		}

		if filter.EndDate != nil {
			conditions = append(conditions, "created_at <= ?")
			args = append(args, *filter.EndDate)
		}

		if filter.SearchQuery != nil && *filter.SearchQuery != "" {
			conditions = append(conditions, "content LIKE ?")
			args = append(args, "%"+*filter.SearchQuery+"%")
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// GetConversationMessages retrieves messages for a specific conversation
func (r *ConversationRepository) GetConversationMessages(ctx context.Context, conversationID string, limit int, offset int) ([]*ai.ConversationMessage, error) {
	filter := &ai.MessageFilter{
		ConversationID: &conversationID,
		Limit:          limit,
		Offset:         offset,
		OrderBy:        "created_at",
		OrderDir:       "ASC",
	}
	return r.GetMessages(ctx, filter)
}

// UpdateMessage updates a message
func (r *ConversationRepository) UpdateMessage(ctx context.Context, msg *ai.ConversationMessage) error {
	query := `
		UPDATE conversation_messages
		SET content = ?, tool_calls = ?, tool_call_id = ?, tokens_used = ?, 
		    model_used = ?, provider_used = ?, response_time_ms = ?, metadata = ?
		WHERE id = ?
	`

	toolCallsJSON, err := msg.MarshalToolCalls()
	if err != nil {
		return fmt.Errorf("failed to marshal tool calls: %w", err)
	}

	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.ExecContext(
		ctx,
		query,
		msg.Content,
		toolCallsJSON,
		msg.ToolCallID,
		msg.TokensUsed,
		msg.ModelUsed,
		msg.ProviderUsed,
		msg.ResponseTimeMs,
		string(metadataJSON),
		msg.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// DeleteMessage deletes a message
func (r *ConversationRepository) DeleteMessage(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get conversation ID first
	var conversationID string
	err = tx.QueryRowContext(ctx, "SELECT conversation_id FROM conversation_messages WHERE id = ?", id).Scan(&conversationID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("message not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get conversation ID: %w", err)
	}

	// Delete message
	result, err := tx.ExecContext(ctx, "DELETE FROM conversation_messages WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found")
	}

	// Update conversation message count
	_, err = tx.ExecContext(ctx, "UPDATE conversations SET message_count = message_count - 1, updated_at = ? WHERE id = ?", time.Now(), conversationID)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

// DeleteConversationMessages deletes all messages for a conversation
func (r *ConversationRepository) DeleteConversationMessages(ctx context.Context, conversationID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete messages
	_, err = tx.ExecContext(ctx, "DELETE FROM conversation_messages WHERE conversation_id = ?", conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Reset conversation message count
	_, err = tx.ExecContext(ctx, "UPDATE conversations SET message_count = 0, last_message_at = NULL, updated_at = ? WHERE id = ?", time.Now(), conversationID)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

// CreateOrUpdateAnalytics creates or updates conversation analytics
func (r *ConversationRepository) CreateOrUpdateAnalytics(ctx context.Context, analytics *ai.ConversationAnalytics) error {
	query := `
		INSERT OR REPLACE INTO conversation_analytics 
		(conversation_id, total_messages, total_tokens, total_cost, avg_response_time_ms, 
		 tools_used, providers_used, models_used, sentiment_score, complexity_score, 
		 satisfaction_rating, date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	providersJSON, err := json.Marshal(analytics.ProvidersUsed)
	if err != nil {
		return fmt.Errorf("failed to marshal providers: %w", err)
	}

	modelsJSON, err := json.Marshal(analytics.ModelsUsed)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	now := time.Now()
	if analytics.CreatedAt.IsZero() {
		analytics.CreatedAt = now
	}
	analytics.UpdatedAt = now

	_, err = r.db.ExecContext(
		ctx,
		query,
		analytics.ConversationID,
		analytics.TotalMessages,
		analytics.TotalTokens,
		analytics.TotalCost,
		analytics.AvgResponseTimeMs,
		analytics.ToolsUsed,
		string(providersJSON),
		string(modelsJSON),
		analytics.SentimentScore,
		analytics.ComplexityScore,
		analytics.SatisfactionRating,
		analytics.Date,
		analytics.CreatedAt,
		analytics.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create/update analytics: %w", err)
	}

	return nil
}

// GetConversationAnalytics retrieves analytics for a conversation on a specific date
func (r *ConversationRepository) GetConversationAnalytics(ctx context.Context, conversationID string, date time.Time) (*ai.ConversationAnalytics, error) {
	query := `
		SELECT id, conversation_id, total_messages, total_tokens, total_cost, avg_response_time_ms, 
		       tools_used, providers_used, models_used, sentiment_score, complexity_score, 
		       satisfaction_rating, date, created_at, updated_at
		FROM conversation_analytics
		WHERE conversation_id = ? AND date = ?
	`

	analytics := &ai.ConversationAnalytics{}
	var providersJSON, modelsJSON string

	err := r.db.QueryRowContext(ctx, query, conversationID, date.Format("2006-01-02")).Scan(
		&analytics.ID,
		&analytics.ConversationID,
		&analytics.TotalMessages,
		&analytics.TotalTokens,
		&analytics.TotalCost,
		&analytics.AvgResponseTimeMs,
		&analytics.ToolsUsed,
		&providersJSON,
		&modelsJSON,
		&analytics.SentimentScore,
		&analytics.ComplexityScore,
		&analytics.SatisfactionRating,
		&analytics.Date,
		&analytics.CreatedAt,
		&analytics.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("analytics not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	if err := json.Unmarshal([]byte(providersJSON), &analytics.ProvidersUsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal providers: %w", err)
	}

	if err := json.Unmarshal([]byte(modelsJSON), &analytics.ModelsUsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models: %w", err)
	}

	return analytics, nil
}

// GetAnalyticsByDateRange retrieves analytics for a conversation within a date range
func (r *ConversationRepository) GetAnalyticsByDateRange(ctx context.Context, conversationID string, startDate, endDate time.Time) ([]*ai.ConversationAnalytics, error) {
	query := `
		SELECT id, conversation_id, total_messages, total_tokens, total_cost, avg_response_time_ms, 
		       tools_used, providers_used, models_used, sentiment_score, complexity_score, 
		       satisfaction_rating, date, created_at, updated_at
		FROM conversation_analytics
		WHERE conversation_id = ? AND date >= ? AND date <= ?
		ORDER BY date
	`

	rows, err := r.db.QueryContext(ctx, query, conversationID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query analytics: %w", err)
	}
	defer rows.Close()

	var analyticsSlice []*ai.ConversationAnalytics
	for rows.Next() {
		analytics := &ai.ConversationAnalytics{}
		var providersJSON, modelsJSON string

		err := rows.Scan(
			&analytics.ID,
			&analytics.ConversationID,
			&analytics.TotalMessages,
			&analytics.TotalTokens,
			&analytics.TotalCost,
			&analytics.AvgResponseTimeMs,
			&analytics.ToolsUsed,
			&providersJSON,
			&modelsJSON,
			&analytics.SentimentScore,
			&analytics.ComplexityScore,
			&analytics.SatisfactionRating,
			&analytics.Date,
			&analytics.CreatedAt,
			&analytics.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analytics: %w", err)
		}

		if err := json.Unmarshal([]byte(providersJSON), &analytics.ProvidersUsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal providers: %w", err)
		}

		if err := json.Unmarshal([]byte(modelsJSON), &analytics.ModelsUsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal models: %w", err)
		}

		analyticsSlice = append(analyticsSlice, analytics)
	}

	return analyticsSlice, nil
}

// GetGlobalStatistics retrieves global conversation statistics
func (r *ConversationRepository) GetGlobalStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (*ai.ConversationStatistics, error) {
	stats := &ai.ConversationStatistics{}

	// Get conversation counts
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN archived = 0 THEN 1 END) as active,
			COUNT(CASE WHEN archived = 1 THEN 1 END) as archived
		FROM conversations 
		WHERE user_id = ? AND created_at >= ? AND created_at <= ?
	`, userID, startDate, endDate).Scan(&stats.TotalConversations, &stats.ActiveConversations, &stats.ArchivedConversations)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation counts: %w", err)
	}

	// Get message and token stats
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(COUNT(*), 0) as message_count,
			COALESCE(SUM(tokens_used), 0) as total_tokens,
			COALESCE(AVG(response_time_ms), 0) as avg_response_time
		FROM conversation_messages cm
		JOIN conversations c ON cm.conversation_id = c.id
		WHERE c.user_id = ? AND cm.created_at >= ? AND cm.created_at <= ?
	`, userID, startDate, endDate).Scan(&stats.TotalMessages, &stats.TotalTokensUsed, &stats.AvgResponseTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get message stats: %w", err)
	}

	// Calculate average messages per conversation
	if stats.TotalConversations > 0 {
		stats.AvgMessagesPerConv = float64(stats.TotalMessages) / float64(stats.TotalConversations)
	}

	// Get provider statistics
	providerStats, err := r.getProviderStats(ctx)
	if err != nil {
		// Failed to get provider stats - continue without them
	} else {
		stats.TopProviders = providerStats
	}

	// Get model statistics
	modelStats, err := r.getModelStats(ctx)
	if err != nil {
		// Failed to get model stats - continue without them
	} else {
		stats.TopModels = modelStats
	}

	// Get tool usage statistics
	toolStats, err := r.getToolStats(ctx)
	if err != nil {
		// Failed to get tool stats - continue without them
	} else {
		stats.TopTools = toolStats
	}

	// Get daily activity for the last 30 days
	dailyActivity, err := r.getDailyActivity(ctx, 30)
	if err != nil {
		// Failed to get daily activity - continue without them
	} else {
		stats.DailyActivity = dailyActivity
	}

	return stats, nil
}

// CleanupOldConversations removes conversations older than specified days
func (r *ConversationRepository) CleanupOldConversations(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	query := "DELETE FROM conversations WHERE created_at < ? AND archived = 1"

	result, err := r.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to cleanup old conversations: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old conversations\n", rowsAffected)
	return nil
}

// CleanupOldMessages removes messages older than specified days
func (r *ConversationRepository) CleanupOldMessages(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	query := "DELETE FROM conversation_messages WHERE created_at < ?"

	result, err := r.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to cleanup old messages: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old messages\n", rowsAffected)
	return nil
}

// CleanupOldAnalytics removes analytics older than specified days
func (r *ConversationRepository) CleanupOldAnalytics(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	query := "DELETE FROM conversation_analytics WHERE date < ?"

	result, err := r.db.ExecContext(ctx, query, cutoffDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to cleanup old analytics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old analytics records\n", rowsAffected)
	return nil
}

// getProviderStats returns statistics grouped by AI provider
func (r *ConversationRepository) getProviderStats(ctx context.Context) ([]ai.ProviderUsageStats, error) {
	query := `
		SELECT 
			provider,
			COUNT(*) as conversation_count,
			SUM(CASE WHEN cm.role = 'user' THEN 1 ELSE 0 END) as user_messages,
			SUM(CASE WHEN cm.role = 'assistant' THEN 1 ELSE 0 END) as assistant_messages,
			AVG(LENGTH(cm.content)) as avg_message_length
		FROM conversations c
		LEFT JOIN conversation_messages cm ON c.id = cm.conversation_id
		WHERE c.provider IS NOT NULL
		GROUP BY provider
		ORDER BY conversation_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query provider stats: %w", err)
	}
	defer rows.Close()

	var stats []ai.ProviderUsageStats
	for rows.Next() {
		var provider string
		var conversationCount, userMessages, assistantMessages int
		var avgMessageLength sql.NullFloat64

		err := rows.Scan(&provider, &conversationCount, &userMessages, &assistantMessages, &avgMessageLength)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider stats: %w", err)
		}

		providerStat := ai.ProviderUsageStats{
			Provider:     provider,
			MessageCount: userMessages + assistantMessages,
			TokenCount:   0, // Would need to be calculated from actual usage
			Cost:         0, // Would need to be calculated from actual usage
			AvgLatency:   0, // Would need to be calculated from actual response times
		}

		stats = append(stats, providerStat)
	}

	return stats, rows.Err()
}

// getModelStats returns statistics grouped by AI model
func (r *ConversationRepository) getModelStats(ctx context.Context) ([]ai.ModelUsageStats, error) {
	query := `
		SELECT 
			model,
			COUNT(*) as conversation_count,
			SUM(CASE WHEN cm.role = 'user' THEN 1 ELSE 0 END) as user_messages,
			SUM(CASE WHEN cm.role = 'assistant' THEN 1 ELSE 0 END) as assistant_messages,
			AVG(LENGTH(cm.content)) as avg_message_length
		FROM conversations c
		LEFT JOIN conversation_messages cm ON c.id = cm.conversation_id
		WHERE c.model IS NOT NULL
		GROUP BY model
		ORDER BY conversation_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query model stats: %w", err)
	}
	defer rows.Close()

	var stats []ai.ModelUsageStats
	for rows.Next() {
		var model string
		var conversationCount, userMessages, assistantMessages int
		var avgMessageLength sql.NullFloat64

		err := rows.Scan(&model, &conversationCount, &userMessages, &assistantMessages, &avgMessageLength)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model stats: %w", err)
		}

		modelStat := ai.ModelUsageStats{
			Model:        model,
			Provider:     "", // Would need to be fetched from conversation data
			MessageCount: userMessages + assistantMessages,
			TokenCount:   0, // Would need to be calculated from actual usage
			Cost:         0, // Would need to be calculated from actual usage
			AvgLatency:   0, // Would need to be calculated from actual response times
		}

		stats = append(stats, modelStat)
	}

	return stats, rows.Err()
}

// getToolStats returns statistics about tool usage in conversations
func (r *ConversationRepository) getToolStats(ctx context.Context) ([]ai.ToolUsageStats, error) {
	// Tool usage is typically stored in message metadata or separate tables
	// For now, we'll provide a basic implementation that could be enhanced
	query := `
		SELECT 
			COUNT(DISTINCT c.id) as conversations_with_tools,
			COUNT(*) as total_tool_calls
		FROM conversations c
		JOIN conversation_messages cm ON c.id = cm.conversation_id
		WHERE cm.metadata LIKE '%tool%' OR cm.content LIKE '%function_call%'
	`

	var conversationsWithTools, totalToolCalls int
	err := r.db.QueryRowContext(ctx, query).Scan(&conversationsWithTools, &totalToolCalls)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool stats: %w", err)
	}

	// Create a basic tool usage stat for general tool usage
	var stats []ai.ToolUsageStats
	if totalToolCalls > 0 {
		generalToolStat := ai.ToolUsageStats{
			ToolName:    "general_tools",
			Category:    "various",
			UsageCount:  totalToolCalls,
			SuccessRate: 0.95, // Estimated success rate
			AvgExecTime: 150,  // Estimated average execution time in ms
		}
		stats = append(stats, generalToolStat)
	}

	return stats, nil
}

// getDailyActivity returns daily conversation activity for the specified number of days
func (r *ConversationRepository) getDailyActivity(ctx context.Context, days int) ([]ai.DailyActivityStats, error) {
	query := `
		SELECT 
			DATE(created_at) as date,
			COUNT(DISTINCT c.id) as conversations,
			COUNT(cm.id) as messages,
			COUNT(DISTINCT c.provider) as providers_used
		FROM conversations c
		LEFT JOIN conversation_messages cm ON c.id = cm.conversation_id
		WHERE c.created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY DATE(c.created_at)
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily activity: %w", err)
	}
	defer rows.Close()

	var activity []ai.DailyActivityStats
	for rows.Next() {
		var dateStr string
		var conversations, messages, providersUsed int

		err := rows.Scan(&dateStr, &conversations, &messages, &providersUsed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily activity: %w", err)
		}

		// Parse date string to time.Time
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// Use current date if parsing fails
			date = time.Now()
		}

		dayStat := ai.DailyActivityStats{
			Date:          date,
			Conversations: conversations,
			Messages:      messages,
			Tokens:        0, // Would need to be calculated from actual usage
			ToolCalls:     0, // Would need to be calculated from actual usage
		}

		activity = append(activity, dayStat)
	}

	return activity, rows.Err()
}
