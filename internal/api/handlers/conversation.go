package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/gin-gonic/gin"
)

// Conversation management handlers

// CreateConversation creates a new conversation
func (h *Handlers) CreateConversation(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ai.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	conversation, err := conversationService.CreateConversation(c.Request.Context(), userID, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to create conversation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"conversation": conversation,
		"message":      "Conversation created successfully",
	})
}

// GetConversation retrieves a specific conversation
func (h *Handlers) GetConversation(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	conversation, err := conversationService.GetConversation(c.Request.Context(), userID, conversationID)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to get conversation")
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": conversation,
	})
}

// GetConversations retrieves conversations for the user
func (h *Handlers) GetConversations(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse query parameters
	filter := &ai.ConversationFilter{}

	if archived := c.Query("archived"); archived != "" {
		if archivedBool, err := strconv.ParseBool(archived); err == nil {
			filter.Archived = &archivedBool
		}
	}

	if provider := c.Query("provider"); provider != "" {
		filter.Provider = &provider
	}

	if searchQuery := c.Query("search"); searchQuery != "" {
		filter.SearchQuery = &searchQuery
	}

	if limit := c.Query("limit"); limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil {
			filter.Limit = limitInt
		}
	} else {
		filter.Limit = 20 // Default limit
	}

	if offset := c.Query("offset"); offset != "" {
		if offsetInt, err := strconv.Atoi(offset); err == nil {
			filter.Offset = offsetInt
		}
	}

	if orderBy := c.Query("order_by"); orderBy != "" {
		filter.OrderBy = orderBy
	} else {
		filter.OrderBy = "last_message_at"
	}

	if orderDir := c.Query("order_dir"); orderDir != "" {
		filter.OrderDir = orderDir
	} else {
		filter.OrderDir = "DESC"
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	conversations, err := conversationService.GetConversations(c.Request.Context(), userID, filter)
	if err != nil {
		h.log.WithError(err).Error("Failed to get conversations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
		"count":         len(conversations),
		"filter":        filter,
	})
}

// UpdateConversation updates conversation settings
func (h *Handlers) UpdateConversation(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	var req ai.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	conversation, err := conversationService.UpdateConversation(c.Request.Context(), userID, conversationID, &req)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to update conversation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": conversation,
		"message":      "Conversation updated successfully",
	})
}

// DeleteConversation deletes a conversation
func (h *Handlers) DeleteConversation(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	err := conversationService.DeleteConversation(c.Request.Context(), userID, conversationID)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to delete conversation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation deleted successfully",
	})
}

// GetConversationMessages retrieves messages for a conversation
func (h *Handlers) GetConversationMessages(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Parse pagination parameters
	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limitInt, err := strconv.Atoi(limitStr); err == nil && limitInt > 0 {
			limit = limitInt
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offsetInt, err := strconv.Atoi(offsetStr); err == nil && offsetInt >= 0 {
			offset = offsetInt
		}
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	messages, err := conversationService.GetConversationMessages(c.Request.Context(), userID, conversationID, limit, offset)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to get conversation messages")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
		"limit":    limit,
		"offset":   offset,
	})
}

// SendMessage sends a message in a conversation
func (h *Handlers) SendMessage(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	var req ai.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	response, err := conversationService.SendMessage(c.Request.Context(), userID, conversationID, &req)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to send message")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
		"message":  "Message sent successfully",
	})
}

// ArchiveConversation archives a conversation
func (h *Handlers) ArchiveConversation(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	err := conversationService.ArchiveConversation(c.Request.Context(), userID, conversationID)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to archive conversation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation archived successfully",
	})
}

// UnarchiveConversation unarchives a conversation
func (h *Handlers) UnarchiveConversation(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	err := conversationService.UnarchiveConversation(c.Request.Context(), userID, conversationID)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to unarchive conversation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unarchive conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation unarchived successfully",
	})
}

// GetConversationStatistics retrieves conversation statistics
func (h *Handlers) GetConversationStatistics(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse date range parameters
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30) // Default to last 30 days

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	stats, err := conversationService.GetConversationStatistics(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		h.log.WithError(err).Error("Failed to get conversation statistics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
		"date_range": gin.H{
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
		},
	})
}

// GenerateConversationTitle generates a title for a conversation
func (h *Handlers) GenerateConversationTitle(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	// Verify user has access to conversation
	_, err := conversationService.GetConversation(c.Request.Context(), userID, conversationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	title, err := conversationService.GenerateConversationTitle(c.Request.Context(), conversationID)
	if err != nil {
		h.log.WithError(err).WithField("conversation_id", conversationID).Error("Failed to generate conversation title")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate title"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"title":   title,
		"message": "Title generated successfully",
	})
}

// CleanupConversations cleans up old conversation data
func (h *Handlers) CleanupConversations(c *gin.Context) {
	// This endpoint might be admin-only or require special permissions
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse days parameter
	days := 90 // Default to 90 days
	if daysStr := c.Query("days"); daysStr != "" {
		if daysInt, err := strconv.Atoi(daysStr); err == nil && daysInt > 0 {
			days = daysInt
		}
	}

	conversationService := h.getConversationService()
	if conversationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Conversation service not available"})
		return
	}

	err := conversationService.CleanupOldData(c.Request.Context(), days)
	if err != nil {
		h.log.WithError(err).Error("Failed to cleanup conversation data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cleanup completed successfully",
		"days":    days,
	})
}

// Helper function to get conversation service
func (h *Handlers) getConversationService() *ai.ConversationService {
	// Return the properly wired conversation service with MCP integration
	return h.conversationService
}

// Helper function to get user ID from context
func getUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	// Return default user ID when authentication is disabled
	// This allows conversation functionality to work without authentication
	return "1"
}
