package cover

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service handles cover business logic
type Service struct {
	entityRepo repositories.EntityRepository
	haClient   HAClient
	logger     *logrus.Logger
}

// HAClient interface for Home Assistant operations
type HAClient interface {
	CallService(ctx context.Context, domain, service string, data map[string]interface{}) error
}

// NewService creates a new cover service
func NewService(entityRepo repositories.EntityRepository, haClient HAClient, logger *logrus.Logger) *Service {
	return &Service{
		entityRepo: entityRepo,
		haClient:   haClient,
		logger:     logger,
	}
}

// CoverState represents the current state of a cover
type CoverState struct {
	State        string   `json:"state"`         // open, closed, opening, closing, stopped
	Position     *int     `json:"position"`      // 0-100, null if not supported
	TiltPosition *int     `json:"tilt_position"` // 0-100, null if not supported
	Features     []string `json:"features"`      // supported features
}

// CoverCapabilities represents what operations a cover supports
type CoverCapabilities struct {
	SupportsPosition  bool `json:"supports_position"`
	SupportsTilt      bool `json:"supports_tilt"`
	SupportsStop      bool `json:"supports_stop"`
	SupportsOpenClose bool `json:"supports_open_close"`
}

// OpenCover opens a cover fully
func (s *Service) OpenCover(ctx context.Context, entityID string) error {
	s.logger.WithField("entity_id", entityID).Info("Opening cover")

	// Validate entity is a cover
	entity, err := s.validateCoverEntity(ctx, entityID)
	if err != nil {
		return err
	}

	// Call Home Assistant service
	serviceData := map[string]interface{}{
		"entity_id": entityID,
	}

	if err := s.haClient.CallService(ctx, "cover", "open_cover", serviceData); err != nil {
		s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to open cover")
		return fmt.Errorf("failed to open cover: %w", err)
	}

	// Update local state
	position := 100
	if err := s.updateCoverState(ctx, entity, "opening", &position, nil); err != nil {
		s.logger.WithError(err).Warn("Failed to update local cover state")
	}

	s.logger.WithField("entity_id", entityID).Info("Cover opened successfully")
	return nil
}

// CloseCover closes a cover fully
func (s *Service) CloseCover(ctx context.Context, entityID string) error {
	s.logger.WithField("entity_id", entityID).Info("Closing cover")

	// Validate entity is a cover
	entity, err := s.validateCoverEntity(ctx, entityID)
	if err != nil {
		return err
	}

	// Call Home Assistant service
	serviceData := map[string]interface{}{
		"entity_id": entityID,
	}

	if err := s.haClient.CallService(ctx, "cover", "close_cover", serviceData); err != nil {
		s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to close cover")
		return fmt.Errorf("failed to close cover: %w", err)
	}

	// Update local state
	position := 0
	if err := s.updateCoverState(ctx, entity, "closing", &position, nil); err != nil {
		s.logger.WithError(err).Warn("Failed to update local cover state")
	}

	s.logger.WithField("entity_id", entityID).Info("Cover closed successfully")
	return nil
}

// StopCover stops cover movement
func (s *Service) StopCover(ctx context.Context, entityID string) error {
	s.logger.WithField("entity_id", entityID).Info("Stopping cover")

	// Validate entity is a cover
	entity, err := s.validateCoverEntity(ctx, entityID)
	if err != nil {
		return err
	}

	// Check if cover supports stop
	capabilities, err := s.getCoverCapabilities(entity)
	if err != nil {
		return err
	}

	if !capabilities.SupportsStop {
		return fmt.Errorf("cover does not support stop operation")
	}

	// Call Home Assistant service
	serviceData := map[string]interface{}{
		"entity_id": entityID,
	}

	if err := s.haClient.CallService(ctx, "cover", "stop_cover", serviceData); err != nil {
		s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to stop cover")
		return fmt.Errorf("failed to stop cover: %w", err)
	}

	// Update local state to stopped
	if err := s.updateCoverState(ctx, entity, "stopped", nil, nil); err != nil {
		s.logger.WithError(err).Warn("Failed to update local cover state")
	}

	s.logger.WithField("entity_id", entityID).Info("Cover stopped successfully")
	return nil
}

// SetCoverPosition sets a specific position for the cover
func (s *Service) SetCoverPosition(ctx context.Context, entityID string, position int) error {
	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"position":  position,
	}).Info("Setting cover position")

	// Validate position
	if position < 0 || position > 100 {
		return fmt.Errorf("position must be between 0 and 100, got %d", position)
	}

	// Validate entity is a cover
	entity, err := s.validateCoverEntity(ctx, entityID)
	if err != nil {
		return err
	}

	// Check if cover supports position control
	capabilities, err := s.getCoverCapabilities(entity)
	if err != nil {
		return err
	}

	if !capabilities.SupportsPosition {
		return fmt.Errorf("cover does not support position control")
	}

	// Call Home Assistant service
	serviceData := map[string]interface{}{
		"entity_id": entityID,
		"position":  position,
	}

	if err := s.haClient.CallService(ctx, "cover", "set_cover_position", serviceData); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"entity_id": entityID,
			"position":  position,
		}).Error("Failed to set cover position")
		return fmt.Errorf("failed to set cover position: %w", err)
	}

	// Determine state based on position
	state := "moving"
	if position == 0 {
		state = "closing"
	} else if position == 100 {
		state = "opening"
	}

	// Update local state
	if err := s.updateCoverState(ctx, entity, state, &position, nil); err != nil {
		s.logger.WithError(err).Warn("Failed to update local cover state")
	}

	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"position":  position,
	}).Info("Cover position set successfully")
	return nil
}

// SetCoverTilt sets the tilt position for venetian blinds
func (s *Service) SetCoverTilt(ctx context.Context, entityID string, tiltPosition int) error {
	s.logger.WithFields(logrus.Fields{
		"entity_id":     entityID,
		"tilt_position": tiltPosition,
	}).Info("Setting cover tilt position")

	// Validate tilt position
	if tiltPosition < 0 || tiltPosition > 100 {
		return fmt.Errorf("tilt position must be between 0 and 100, got %d", tiltPosition)
	}

	// Validate entity is a cover
	entity, err := s.validateCoverEntity(ctx, entityID)
	if err != nil {
		return err
	}

	// Check if cover supports tilt control
	capabilities, err := s.getCoverCapabilities(entity)
	if err != nil {
		return err
	}

	if !capabilities.SupportsTilt {
		return fmt.Errorf("cover does not support tilt control")
	}

	// Call Home Assistant service
	serviceData := map[string]interface{}{
		"entity_id":     entityID,
		"tilt_position": tiltPosition,
	}

	if err := s.haClient.CallService(ctx, "cover", "set_cover_tilt_position", serviceData); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"entity_id":     entityID,
			"tilt_position": tiltPosition,
		}).Error("Failed to set cover tilt position")
		return fmt.Errorf("failed to set cover tilt position: %w", err)
	}

	// Update local state
	if err := s.updateCoverState(ctx, entity, "", nil, &tiltPosition); err != nil {
		s.logger.WithError(err).Warn("Failed to update local cover state")
	}

	s.logger.WithFields(logrus.Fields{
		"entity_id":     entityID,
		"tilt_position": tiltPosition,
	}).Info("Cover tilt position set successfully")
	return nil
}

// GetCoverStatus retrieves the current status of a cover
func (s *Service) GetCoverStatus(ctx context.Context, entityID string) (*CoverState, error) {
	entity, err := s.validateCoverEntity(ctx, entityID)
	if err != nil {
		return nil, err
	}

	// Parse current attributes
	var attributes map[string]interface{}
	if entity.Attributes != nil {
		if err := json.Unmarshal(entity.Attributes, &attributes); err != nil {
			s.logger.WithError(err).Warn("Failed to parse entity attributes")
			attributes = make(map[string]interface{})
		}
	}

	state := &CoverState{
		State: entity.State.String,
	}

	// Extract position if available
	if pos, ok := attributes["current_position"]; ok {
		if posInt, ok := pos.(float64); ok {
			position := int(posInt)
			state.Position = &position
		}
	}

	// Extract tilt position if available
	if tilt, ok := attributes["current_tilt_position"]; ok {
		if tiltInt, ok := tilt.(float64); ok {
			tiltPosition := int(tiltInt)
			state.TiltPosition = &tiltPosition
		}
	}

	// Get capabilities
	capabilities, err := s.getCoverCapabilities(entity)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get cover capabilities")
	} else {
		// Add supported features to state
		var features []string
		if capabilities.SupportsPosition {
			features = append(features, "position")
		}
		if capabilities.SupportsTilt {
			features = append(features, "tilt")
		}
		if capabilities.SupportsStop {
			features = append(features, "stop")
		}
		if capabilities.SupportsOpenClose {
			features = append(features, "open_close")
		}
		state.Features = features
	}

	return state, nil
}

// validateCoverEntity validates that the entity exists and is a cover
func (s *Service) validateCoverEntity(ctx context.Context, entityID string) (*models.Entity, error) {
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}

	if entity.Domain != "cover" {
		return nil, fmt.Errorf("entity %s is not a cover (domain: %s)", entityID, entity.Domain)
	}

	return entity, nil
}

// getCoverCapabilities determines what operations a cover supports
func (s *Service) getCoverCapabilities(entity *models.Entity) (*CoverCapabilities, error) {
	capabilities := &CoverCapabilities{
		SupportsOpenClose: true, // All covers support basic open/close
	}

	// Parse attributes to determine capabilities
	var attributes map[string]interface{}
	if entity.Attributes != nil {
		if err := json.Unmarshal(entity.Attributes, &attributes); err != nil {
			return capabilities, nil // Return basic capabilities if parsing fails
		}
	}

	// Check for supported features bitmask (HA standard)
	if supportedFeatures, ok := attributes["supported_features"]; ok {
		if features, ok := supportedFeatures.(float64); ok {
			featureInt := int(features)
			capabilities.SupportsStop = (featureInt & 2) != 0     // SUPPORT_STOP = 2
			capabilities.SupportsPosition = (featureInt & 4) != 0 // SUPPORT_SET_POSITION = 4
			capabilities.SupportsTilt = (featureInt & 128) != 0   // SUPPORT_SET_TILT_POSITION = 128
		}
	}

	// Fallback: check for position/tilt attributes
	if _, ok := attributes["current_position"]; ok {
		capabilities.SupportsPosition = true
	}
	if _, ok := attributes["current_tilt_position"]; ok {
		capabilities.SupportsTilt = true
	}

	return capabilities, nil
}

// updateCoverState updates the local entity state
func (s *Service) updateCoverState(ctx context.Context, entity *models.Entity, state string, position *int, tiltPosition *int) error {
	// Parse current attributes
	var attributes map[string]interface{}
	if entity.Attributes != nil {
		if err := json.Unmarshal(entity.Attributes, &attributes); err != nil {
			attributes = make(map[string]interface{})
		}
	} else {
		attributes = make(map[string]interface{})
	}

	// Update attributes
	if position != nil {
		attributes["current_position"] = *position
	}
	if tiltPosition != nil {
		attributes["current_tilt_position"] = *tiltPosition
	}

	// Marshal updated attributes
	updatedAttributes, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Create updated entity
	updatedEntity := &models.Entity{
		EntityID:     entity.EntityID,
		Domain:       entity.Domain,
		FriendlyName: entity.FriendlyName,
		RoomID:       entity.RoomID,
		Attributes:   updatedAttributes,
		LastUpdated:  time.Now(),
	}

	// Update state if provided
	if state != "" {
		updatedEntity.State.String = state
		updatedEntity.State.Valid = true
	} else {
		updatedEntity.State = entity.State
	}

	// Save to database
	return s.entityRepo.Update(ctx, updatedEntity)
}

// OperateCoversInGroup operates multiple covers simultaneously
func (s *Service) OperateCoversInGroup(ctx context.Context, entityIDs []string, operation string, data map[string]interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"entity_ids": entityIDs,
		"operation":  operation,
	}).Info("Operating covers in group")

	// Validate all entities are covers
	for _, entityID := range entityIDs {
		if _, err := s.validateCoverEntity(ctx, entityID); err != nil {
			return fmt.Errorf("invalid entity %s: %w", entityID, err)
		}
	}

	// Prepare service data
	serviceData := make(map[string]interface{})
	serviceData["entity_id"] = entityIDs

	for k, v := range data {
		serviceData[k] = v
	}

	// Call appropriate service
	var service string
	switch operation {
	case "open":
		service = "open_cover"
	case "close":
		service = "close_cover"
	case "stop":
		service = "stop_cover"
	case "set_position":
		service = "set_cover_position"
	case "set_tilt":
		service = "set_cover_tilt_position"
	default:
		return fmt.Errorf("unsupported group operation: %s", operation)
	}

	if err := s.haClient.CallService(ctx, "cover", service, serviceData); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"entity_ids": entityIDs,
			"operation":  operation,
		}).Error("Failed to operate covers in group")
		return fmt.Errorf("failed to operate covers in group: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"entity_ids": entityIDs,
		"operation":  operation,
	}).Info("Group cover operation completed successfully")
	return nil
}
