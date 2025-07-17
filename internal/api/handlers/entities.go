package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/entities"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetEntities retrieves all entities
func (h *Handlers) GetEntities(c *gin.Context) {
	includeRoom := c.Query("include_room") == "true"
	domain := c.Query("domain")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create entity service
	entityService := entities.NewService(h.repos.Entity, h.repos.Room, h.log)

	if domain != "" {
		// Filter by domain
		domainEntities, err := entityService.GetByDomain(ctx, domain)
		if err != nil {
			h.log.WithError(err).Error("Failed to get entities by domain")
			utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
			return
		}

		utils.SendSuccessWithMeta(c, domainEntities, gin.H{
			"count":  len(domainEntities),
			"domain": domain,
		})
		return
	}

	// Get all entities
	entitiesWithRooms, err := entityService.GetAll(ctx, includeRoom)
	if err != nil {
		h.log.WithError(err).Error("Failed to get all entities")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
		return
	}

	utils.SendSuccessWithMeta(c, entitiesWithRooms, gin.H{
		"count":        len(entitiesWithRooms),
		"include_room": includeRoom,
	})
}

// GetEntity retrieves a specific entity
func (h *Handlers) GetEntity(c *gin.Context) {
	entityID := c.Param("id")
	includeRoom := c.Query("include_room") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entityService := entities.NewService(h.repos.Entity, h.repos.Room, h.log)

	entityWithRoom, err := entityService.GetByID(ctx, entityID, includeRoom)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get entity: %s", entityID)
		utils.SendError(c, http.StatusNotFound, "Entity not found")
		return
	}

	utils.SendSuccess(c, entityWithRoom)
}

// UpdateEntityState updates entity state
func (h *Handlers) UpdateEntityState(c *gin.Context) {
	entityID := c.Param("id")

	var request struct {
		State      string                 `json:"state" binding:"required"`
		Attributes map[string]interface{} `json:"attributes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entityService := entities.NewService(h.repos.Entity, h.repos.Room, h.log)

	err := entityService.UpdateState(ctx, entityID, request.State, request.Attributes)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to update entity state: %s", entityID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to update entity state")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Entity state updated successfully",
		"entity_id": entityID,
		"state":     request.State,
	})
}

// AssignEntityToRoom assigns an entity to a room
func (h *Handlers) AssignEntityToRoom(c *gin.Context) {
	entityID := c.Param("id")

	var request struct {
		RoomID *int `json:"room_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entityService := entities.NewService(h.repos.Entity, h.repos.Room, h.log)

	err := entityService.AssignToRoom(ctx, entityID, request.RoomID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to assign entity to room: %s", entityID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to assign entity to room")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Entity assigned to room successfully",
		"entity_id": entityID,
		"room_id":   request.RoomID,
	})
}

// CreateOrUpdateEntity creates or updates an entity
func (h *Handlers) CreateOrUpdateEntity(c *gin.Context) {
	var request struct {
		EntityID     string                 `json:"entity_id" binding:"required"`
		FriendlyName string                 `json:"friendly_name"`
		Domain       string                 `json:"domain" binding:"required"`
		State        string                 `json:"state"`
		Attributes   map[string]interface{} `json:"attributes"`
		RoomID       *int                   `json:"room_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create entity model
	entity := &models.Entity{
		EntityID: request.EntityID,
		Domain:   request.Domain,
	}

	if request.FriendlyName != "" {
		entity.FriendlyName.String = request.FriendlyName
		entity.FriendlyName.Valid = true
	}

	if request.State != "" {
		entity.State.String = request.State
		entity.State.Valid = true
	}

	if request.RoomID != nil {
		entity.RoomID.Int64 = int64(*request.RoomID)
		entity.RoomID.Valid = true
	}

	if request.Attributes != nil {
		attributesJSON, err := json.Marshal(request.Attributes)
		if err != nil {
			utils.SendError(c, http.StatusBadRequest, "Invalid attributes format")
			return
		}
		entity.Attributes = attributesJSON
	}

	entityService := entities.NewService(h.repos.Entity, h.repos.Room, h.log)

	err := entityService.CreateOrUpdate(ctx, entity)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to create/update entity: %s", request.EntityID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to save entity")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Entity saved successfully",
		"entity_id": request.EntityID,
	})
}

// DeleteEntity deletes an entity
func (h *Handlers) DeleteEntity(c *gin.Context) {
	entityID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entityService := entities.NewService(h.repos.Entity, h.repos.Room, h.log)

	err := entityService.Delete(ctx, entityID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to delete entity: %s", entityID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete entity")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Entity deleted successfully",
		"entity_id": entityID,
	})
}
