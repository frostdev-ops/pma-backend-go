package energymgr

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/energy"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service provides energy management functionality
type Service struct {
	repo       repositories.EnergyRepository
	entityRepo repositories.EntityRepository
	upsRepo    repositories.UPSRepository
	logger     *logrus.Logger

	// Internal state
	settings       *energy.EnergySettings
	energyHistory  []energy.EnergyHistoryEntry
	lastUpdateTime time.Time
	updateTicker   *time.Ticker
	stopChan       chan bool
	isInitialized  bool
	mutex          sync.RWMutex

	// Entity cache for energy calculations
	entityCache map[string]interface{}
	cacheMutex  sync.RWMutex
	cacheExpiry time.Time
}

// NewService creates a new energy service
func NewService(repo repositories.EnergyRepository, entityRepo repositories.EntityRepository, upsRepo repositories.UPSRepository, logger *logrus.Logger) *Service {
	service := &Service{
		repo:          repo,
		entityRepo:    entityRepo,
		upsRepo:       upsRepo,
		logger:        logger,
		energyHistory: make([]energy.EnergyHistoryEntry, 0),
		stopChan:      make(chan bool),
		entityCache:   make(map[string]interface{}),
	}

	// Initialize asynchronously to avoid blocking constructor
	go service.initialize()

	return service
}

// initialize sets up the energy service
func (s *Service) initialize() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("Initializing Energy Service")

	// Load settings
	if err := s.loadSettings(); err != nil {
		s.logger.WithError(err).Error("Failed to load energy settings")
		return err
	}

	// Load energy history
	if err := s.loadEnergyHistory(); err != nil {
		s.logger.WithError(err).Error("Failed to load energy history")
		return err
	}

	// Start tracking if enabled
	if s.settings.TrackingEnabled {
		s.startTracking()
	}

	s.isInitialized = true
	s.logger.Info("Energy Service initialized successfully")

	return nil
}

// loadSettings loads energy settings from database
func (s *Service) loadSettings() error {
	ctx := context.Background()
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		// Create default settings if none exist
		s.logger.Info("Creating default energy settings")
		s.settings = &energy.EnergySettings{
			ID:               1,
			EnergyRate:       energy.DefaultEnergyRate,
			Currency:         energy.DefaultCurrency,
			TrackingEnabled:  energy.DefaultTrackingEnabled,
			UpdateInterval:   energy.DefaultUpdateInterval,
			HistoricalPeriod: energy.DefaultHistoricalPeriod,
			UpdatedAt:        time.Now(),
		}

		if err := s.repo.UpdateSettings(ctx, s.settings); err != nil {
			return fmt.Errorf("failed to create default settings: %w", err)
		}
	} else {
		s.settings = settings
	}

	s.logger.WithFields(logrus.Fields{
		"energy_rate":      s.settings.EnergyRate,
		"currency":         s.settings.Currency,
		"tracking_enabled": s.settings.TrackingEnabled,
		"update_interval":  s.settings.UpdateInterval,
	}).Info("Energy settings loaded")

	return nil
}

// loadEnergyHistory loads recent energy history from database
func (s *Service) loadEnergyHistory() error {
	ctx := context.Background()

	// Load last 1000 entries
	filter := &energy.EnergyHistoryFilter{
		Limit: 1000,
	}

	histories, err := s.repo.GetEnergyHistory(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to load energy history: %w", err)
	}

	// Convert to simplified history entries
	s.energyHistory = make([]energy.EnergyHistoryEntry, 0, len(histories))
	for _, history := range histories {
		s.energyHistory = append(s.energyHistory, energy.EnergyHistoryEntry{
			Timestamp:        history.Timestamp,
			PowerConsumption: history.PowerConsumption,
			EnergyUsage:      history.EnergyUsage,
			Cost:             history.Cost,
		})
	}

	s.logger.WithField("history_count", len(s.energyHistory)).Info("Energy history loaded")
	return nil
}

// startTracking begins energy monitoring
func (s *Service) startTracking() {
	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}

	interval := time.Duration(s.settings.UpdateInterval) * time.Second
	s.updateTicker = time.NewTicker(interval)

	go func() {
		s.logger.WithField("interval", interval).Info("Starting energy tracking")
		for {
			select {
			case <-s.updateTicker.C:
				if err := s.updateEnergyData(); err != nil {
					s.logger.WithError(err).Error("Failed to update energy data")
				}
			case <-s.stopChan:
				s.logger.Info("Stopping energy tracking")
				return
			}
		}
	}()
}

// stopTracking stops energy monitoring
func (s *Service) stopTracking() {
	if s.updateTicker != nil {
		s.updateTicker.Stop()
		s.updateTicker = nil
	}

	select {
	case s.stopChan <- true:
	default:
	}
}

// updateEnergyData performs the main energy calculation and storage
func (s *Service) updateEnergyData() error {
	s.logger.Debug("Updating energy data")

	// Get current entities
	entities, err := s.getEntities()
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}

	// Calculate energy data
	energyData, err := s.calculateEnergyData(entities)
	if err != nil {
		return fmt.Errorf("failed to calculate energy data: %w", err)
	}

	// Save to database
	if err := s.saveEnergySnapshot(energyData); err != nil {
		return fmt.Errorf("failed to save energy snapshot: %w", err)
	}

	// Update in-memory history
	s.mutex.Lock()
	s.energyHistory = append([]energy.EnergyHistoryEntry{{
		Timestamp:        energyData.Timestamp,
		PowerConsumption: energyData.TotalPowerConsumption,
		EnergyUsage:      energyData.TotalEnergyUsage,
		Cost:             energyData.TotalCost,
	}}, s.energyHistory...)

	// Keep only recent history in memory
	if len(s.energyHistory) > 1000 {
		s.energyHistory = s.energyHistory[:1000]
	}

	s.lastUpdateTime = time.Now()
	s.mutex.Unlock()

	s.logger.WithFields(logrus.Fields{
		"total_power": energyData.TotalPowerConsumption,
		"devices":     len(energyData.DeviceBreakdown),
		"total_cost":  energyData.TotalCost,
	}).Debug("Energy data updated")

	return nil
}

// getEntities retrieves entities with caching
func (s *Service) getEntities() (map[string]interface{}, error) {
	s.cacheMutex.RLock()
	if time.Now().Before(s.cacheExpiry) && len(s.entityCache) > 0 {
		defer s.cacheMutex.RUnlock()
		return s.entityCache, nil
	}
	s.cacheMutex.RUnlock()

	// Cache expired, refresh
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	ctx := context.Background()
	entities, err := s.entityRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}

	// Convert to map and parse attributes
	entityMap := make(map[string]interface{})
	for _, entity := range entities {
		// Parse attributes JSON
		var attributes map[string]interface{}
		if len(entity.Attributes) > 0 {
			json.Unmarshal(entity.Attributes, &attributes)
		}

		entityData := map[string]interface{}{
			"entity_id":     entity.EntityID,
			"friendly_name": entity.FriendlyName.String,
			"domain":        entity.Domain,
			"state":         entity.State.String,
			"attributes":    attributes,
			"room":          "",
		}

		// Add room information if available
		if entity.RoomID.Valid {
			// This would need room lookup - simplified for now
			entityData["room"] = fmt.Sprintf("room_%d", entity.RoomID.Int64)
		}

		entityMap[entity.EntityID] = entityData
	}

	s.entityCache = entityMap
	s.cacheExpiry = time.Now().Add(5 * time.Minute) // Cache for 5 minutes

	s.logger.WithField("entity_count", len(entityMap)).Debug("Entity cache refreshed")
	return entityMap, nil
}

// calculateEnergyData performs energy calculations for all entities
func (s *Service) calculateEnergyData(entities map[string]interface{}) (*energy.EnergyData, error) {
	energyData := &energy.EnergyData{
		Timestamp:       time.Now(),
		DeviceBreakdown: make([]energy.DeviceEnergyConsumption, 0),
	}

	// Add UPS power consumption if available
	if s.upsRepo != nil {
		if upsConsumption, err := s.getUPSPowerConsumption(); err == nil && upsConsumption > 0 {
			energyData.UPSPowerConsumption = upsConsumption
			energyData.TotalPowerConsumption += upsConsumption
		}
	}

	// Process each entity for power consumption
	for entityID, entityData := range entities {
		entity, ok := entityData.(map[string]interface{})
		if !ok {
			continue
		}

		powerConsumption := s.extractPowerConsumption(entity, entities)
		if powerConsumption > 0 {
			energyUsage := s.calculateEnergyUsage(entity, powerConsumption)
			cost := energyUsage * s.settings.EnergyRate

			// Get comprehensive energy data for Shelly devices
			comprehensiveData := s.getComprehensiveEnergyData(entity, entities)
			sensorsFound := s.getSensorsFound(comprehensiveData)

			deviceConsumption := energy.DeviceEnergyConsumption{
				EntityID:         entityID,
				DeviceName:       s.getStringValue(entity, "friendly_name"),
				Room:             s.getStringValue(entity, "room"),
				PowerConsumption: powerConsumption,
				EnergyUsage:      energyUsage,
				Cost:             cost,
				State:            s.getStringValue(entity, "state"),
				IsOn:             s.getStringValue(entity, "state") == "on",
				Percentage:       0, // Will be calculated after total is known
				Current:          comprehensiveData.Current,
				Voltage:          comprehensiveData.Voltage,
				Frequency:        comprehensiveData.Frequency,
				ReturnedEnergy:   comprehensiveData.ReturnedEnergy,
				HasSensors:       len(sensorsFound) > 0,
				SensorsFound:     sensorsFound,
			}

			energyData.DeviceBreakdown = append(energyData.DeviceBreakdown, deviceConsumption)
			energyData.TotalPowerConsumption += powerConsumption
			energyData.TotalEnergyUsage += energyUsage
			energyData.TotalCost += cost
		}
	}

	// Calculate percentages
	if energyData.TotalPowerConsumption > 0 {
		for i := range energyData.DeviceBreakdown {
			energyData.DeviceBreakdown[i].Percentage = (energyData.DeviceBreakdown[i].PowerConsumption / energyData.TotalPowerConsumption) * 100
		}
	}

	s.logger.WithFields(logrus.Fields{
		"devices_with_power": len(energyData.DeviceBreakdown),
		"total_power":        energyData.TotalPowerConsumption,
	}).Debug("Energy data calculated")

	return energyData, nil
}

// extractPowerConsumption extracts power consumption from an entity
func (s *Service) extractPowerConsumption(entity map[string]interface{}, allEntities map[string]interface{}) float64 {
	attributes, ok := entity["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	// Check direct power consumption attributes
	for _, attr := range energy.PowerSensorAttributes {
		if value, exists := attributes[attr]; exists {
			if power := s.parseFloat(value); !math.IsNaN(power) && power > 0 {
				return power
			}
		}
	}

	// Check if this entity is a power sensor itself
	if s.isPowerSensor(entity) {
		if power := s.parseFloat(entity["state"]); !math.IsNaN(power) && power > 0 {
			s.logger.WithFields(logrus.Fields{
				"entity_id": entity["entity_id"],
				"power":     power,
			}).Debug("Direct power sensor found")
			return power
		}
	}

	// Look for associated power sensors (especially for Shelly devices)
	if powerSensorID := s.findAssociatedPowerSensor(entity, allEntities); powerSensorID != "" {
		if powerSensor, exists := allEntities[powerSensorID]; exists {
			if powerEntity, ok := powerSensor.(map[string]interface{}); ok {
				if power := s.parseFloat(powerEntity["state"]); !math.IsNaN(power) && power > 0 {
					s.logger.WithFields(logrus.Fields{
						"entity_id":    entity["entity_id"],
						"power_sensor": powerSensorID,
						"power":        power,
					}).Debug("Associated power sensor found")
					return power
				}
			}
		}
	}

	return 0
}

// isPowerSensor checks if an entity is a power sensor
func (s *Service) isPowerSensor(entity map[string]interface{}) bool {
	entityID := s.getStringValue(entity, "entity_id")
	domain := s.getStringValue(entity, "domain")

	if domain != "sensor" {
		return false
	}

	// Check for power-related keywords in entity ID
	powerKeywords := []string{"power", "watt", "energy", "consumption"}
	entityLower := strings.ToLower(entityID)

	for _, keyword := range powerKeywords {
		if strings.Contains(entityLower, keyword) {
			return true
		}
	}

	// Check unit of measurement
	if attributes, ok := entity["attributes"].(map[string]interface{}); ok {
		if unit, exists := attributes["unit_of_measurement"]; exists {
			unitStr := strings.ToLower(fmt.Sprintf("%v", unit))
			return unitStr == "w" || unitStr == "watt" || unitStr == "watts"
		}
	}

	return false
}

// findAssociatedPowerSensor finds a power sensor associated with an entity
func (s *Service) findAssociatedPowerSensor(entity map[string]interface{}, allEntities map[string]interface{}) string {
	entityID := s.getStringValue(entity, "entity_id")

	// Extract base ID (e.g., "switch.shelly_device" -> "shelly_device")
	parts := strings.Split(entityID, ".")
	if len(parts) < 2 {
		return ""
	}
	baseID := parts[1]

	// Look for Shelly power sensors
	powerSensorPatterns := []string{
		fmt.Sprintf("sensor.%s_power", baseID),
		fmt.Sprintf("sensor.%s_active_power", baseID),
		fmt.Sprintf("sensor.%s_power_w", baseID),
	}

	for _, pattern := range powerSensorPatterns {
		if _, exists := allEntities[pattern]; exists {
			return pattern
		}
	}

	return ""
}

// getComprehensiveEnergyData gets comprehensive energy data for Shelly devices
func (s *Service) getComprehensiveEnergyData(entity map[string]interface{}, allEntities map[string]interface{}) energy.ComprehensiveEnergyData {
	entityID := s.getStringValue(entity, "entity_id")

	// Extract base ID
	parts := strings.Split(entityID, ".")
	if len(parts) < 2 {
		return energy.ComprehensiveEnergyData{}
	}
	baseID := parts[1]

	energyData := energy.ComprehensiveEnergyData{}

	// Look for all Shelly energy sensors
	for sensorType, patterns := range energy.ShellyEnergyPatterns {
		for _, pattern := range patterns {
			sensorID := fmt.Sprintf("sensor.%s%s", baseID, pattern)
			if sensorEntity, exists := allEntities[sensorID]; exists {
				if sensor, ok := sensorEntity.(map[string]interface{}); ok {
					value := s.parseFloat(sensor["state"])
					if !math.IsNaN(value) {
						switch sensorType {
						case "power":
							energyData.Power = value
						case "current":
							energyData.Current = value
						case "energy":
							energyData.Energy = value
						case "voltage":
							energyData.Voltage = value
						case "frequency":
							energyData.Frequency = value
						case "returned_energy":
							energyData.ReturnedEnergy = value
						}
					}
				}
			}
		}
	}

	return energyData
}

// getSensorsFound returns a list of sensors found for comprehensive energy data
func (s *Service) getSensorsFound(data energy.ComprehensiveEnergyData) []string {
	var sensors []string

	if data.Power > 0 {
		sensors = append(sensors, "power")
	}
	if data.Current > 0 {
		sensors = append(sensors, "current")
	}
	if data.Energy > 0 {
		sensors = append(sensors, "energy")
	}
	if data.Voltage > 0 {
		sensors = append(sensors, "voltage")
	}
	if data.Frequency > 0 {
		sensors = append(sensors, "frequency")
	}
	if data.ReturnedEnergy > 0 {
		sensors = append(sensors, "returned_energy")
	}

	return sensors
}

// calculateEnergyUsage calculates energy usage in kWh
func (s *Service) calculateEnergyUsage(entity map[string]interface{}, powerConsumption float64) float64 {
	// Convert watts to kilowatts and calculate usage for the update interval
	intervalHours := float64(s.settings.UpdateInterval) / energy.SecondsPerHour
	return (powerConsumption / energy.WattsToKilowatts) * intervalHours
}

// getUPSPowerConsumption gets power consumption from UPS if available
func (s *Service) getUPSPowerConsumption() (float64, error) {
	// This would integrate with UPS service - placeholder for now
	return 0, nil
}

// saveEnergySnapshot saves energy data to database
func (s *Service) saveEnergySnapshot(energyData *energy.EnergyData) error {
	ctx := context.Background()

	// Save overall energy snapshot
	history := &energy.EnergyHistory{
		Timestamp:        energyData.Timestamp,
		PowerConsumption: energyData.TotalPowerConsumption,
		EnergyUsage:      energyData.TotalEnergyUsage,
		Cost:             energyData.TotalCost,
		DeviceCount:      len(energyData.DeviceBreakdown),
	}

	if err := s.repo.CreateEnergyHistory(ctx, history); err != nil {
		return fmt.Errorf("failed to save energy history: %w", err)
	}

	// Save device-specific data
	var deviceEnergies []*energy.DeviceEnergy
	for _, device := range energyData.DeviceBreakdown {
		deviceEnergy := &energy.DeviceEnergy{
			EntityID:         device.EntityID,
			DeviceName:       device.DeviceName,
			Room:             device.Room,
			PowerConsumption: device.PowerConsumption,
			EnergyUsage:      device.EnergyUsage,
			Cost:             device.Cost,
			State:            device.State,
			IsOn:             device.IsOn,
			Percentage:       device.Percentage,
			Timestamp:        energyData.Timestamp,
		}
		deviceEnergies = append(deviceEnergies, deviceEnergy)
	}

	if len(deviceEnergies) > 0 {
		if err := s.repo.CreateDeviceEnergyBatch(ctx, deviceEnergies); err != nil {
			return fmt.Errorf("failed to save device energies: %w", err)
		}
	}

	return nil
}

// GetCurrentEnergyData returns current energy consumption data
func (s *Service) GetCurrentEnergyData() (*energy.EnergyData, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("energy service not initialized")
	}

	entities, err := s.getEntities()
	if err != nil {
		return nil, err
	}

	return s.calculateEnergyData(entities)
}

// GetEnergyStats returns energy statistics
func (s *Service) GetEnergyStats() (*energy.EnergyStats, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("energy service not initialized")
	}

	ctx := context.Background()

	// Get stats for the last 24 hours
	endDate := time.Now()
	startDate := endDate.Add(-24 * time.Hour)

	return s.repo.GetEnergyStats(ctx, startDate, endDate)
}

// UpdateSettings updates energy settings
func (s *Service) UpdateSettings(newSettings *energy.EnergySettingsRequest) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ctx := context.Background()

	// Update settings
	if newSettings.EnergyRate != nil {
		s.settings.EnergyRate = *newSettings.EnergyRate
	}
	if newSettings.Currency != nil {
		s.settings.Currency = *newSettings.Currency
	}
	if newSettings.TrackingEnabled != nil {
		s.settings.TrackingEnabled = *newSettings.TrackingEnabled
	}
	if newSettings.UpdateInterval != nil {
		s.settings.UpdateInterval = *newSettings.UpdateInterval
	}
	if newSettings.HistoricalPeriod != nil {
		s.settings.HistoricalPeriod = *newSettings.HistoricalPeriod
	}

	s.settings.UpdatedAt = time.Now()

	// Save to database
	if err := s.repo.UpdateSettings(ctx, s.settings); err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	// Restart tracking if interval changed or tracking enabled/disabled
	if newSettings.UpdateInterval != nil || newSettings.TrackingEnabled != nil {
		s.stopTracking()
		if s.settings.TrackingEnabled {
			s.startTracking()
		}
	}

	s.logger.WithFields(logrus.Fields{
		"energy_rate":      s.settings.EnergyRate,
		"tracking_enabled": s.settings.TrackingEnabled,
		"update_interval":  s.settings.UpdateInterval,
	}).Info("Energy settings updated")

	return nil
}

// GetSettings returns current energy settings
func (s *Service) GetSettings() *energy.EnergySettings {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.settings
}

// GetDevicePowerConsumption returns power consumption for a specific device
func (s *Service) GetDevicePowerConsumption(entityID string) (float64, error) {
	entities, err := s.getEntities()
	if err != nil {
		return 0, err
	}

	if entityData, exists := entities[entityID]; exists {
		if entity, ok := entityData.(map[string]interface{}); ok {
			return s.extractPowerConsumption(entity, entities), nil
		}
	}

	return 0, fmt.Errorf("entity not found: %s", entityID)
}

// GetDeviceComprehensiveEnergyData returns comprehensive energy data for a device
func (s *Service) GetDeviceComprehensiveEnergyData(entityID string) (*energy.ComprehensiveEnergyData, error) {
	entities, err := s.getEntities()
	if err != nil {
		return nil, err
	}

	if entityData, exists := entities[entityID]; exists {
		if entity, ok := entityData.(map[string]interface{}); ok {
			data := s.getComprehensiveEnergyData(entity, entities)
			return &data, nil
		}
	}

	return nil, fmt.Errorf("entity not found: %s", entityID)
}

// GetEnergyMetrics returns energy monitoring metrics
func (s *Service) GetEnergyMetrics() (*energy.EnergyMetrics, error) {
	ctx := context.Background()
	return s.repo.GetDeviceEnergyMetrics(ctx)
}

// CleanupOldData removes old energy data based on retention settings
func (s *Service) CleanupOldData() error {
	if !s.isInitialized {
		return fmt.Errorf("energy service not initialized")
	}

	ctx := context.Background()
	days := s.settings.HistoricalPeriod

	s.logger.WithField("retention_days", days).Info("Cleaning up old energy data")

	// Cleanup old history
	if err := s.repo.CleanupOldHistory(ctx, days); err != nil {
		return fmt.Errorf("failed to cleanup old history: %w", err)
	}

	// Cleanup old device energy data
	if err := s.repo.CleanupOldDeviceEnergy(ctx, days); err != nil {
		return fmt.Errorf("failed to cleanup old device energy: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the energy service
func (s *Service) Shutdown() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("Shutting down Energy Service")
	s.stopTracking()
	return nil
}

// GetEnergyHistory retrieves energy history with filtering
func (s *Service) GetEnergyHistory(filter *energy.EnergyHistoryFilter) ([]energy.EnergyHistory, error) {
	ctx := context.Background()
	historyPtrs, err := s.repo.GetEnergyHistory(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert from []*energy.EnergyHistory to []energy.EnergyHistory
	history := make([]energy.EnergyHistory, len(historyPtrs))
	for i, ptr := range historyPtrs {
		if ptr != nil {
			history[i] = *ptr
		}
	}

	return history, nil
}

// GetDeviceEnergyHistory retrieves device energy history with filtering
func (s *Service) GetDeviceEnergyHistory(filter *energy.DeviceEnergyFilter) ([]energy.DeviceEnergy, error) {
	ctx := context.Background()
	deviceEnergyPtrs, err := s.repo.GetDeviceEnergy(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert from []*energy.DeviceEnergy to []energy.DeviceEnergy
	deviceEnergy := make([]energy.DeviceEnergy, len(deviceEnergyPtrs))
	for i, ptr := range deviceEnergyPtrs {
		if ptr != nil {
			deviceEnergy[i] = *ptr
		}
	}

	return deviceEnergy, nil
}

// GetEnergyStatistics retrieves energy statistics for a time period
func (s *Service) GetEnergyStatistics(startTime, endTime *time.Time) (*energy.EnergyStats, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.isInitialized {
		return nil, fmt.Errorf("energy service not initialized")
	}

	// Get current energy data
	currentData, err := s.GetCurrentEnergyData()
	if err != nil {
		return nil, fmt.Errorf("failed to get current energy data: %w", err)
	}

	// Create energy stats
	stats := &energy.EnergyStats{
		CurrentPower: currentData.TotalPowerConsumption,
		TotalEnergy:  currentData.TotalEnergyUsage,
		TotalCost:    currentData.TotalCost,
		TopConsumers: currentData.DeviceBreakdown,
		History:      s.energyHistory,
	}

	// Calculate statistics from history
	if len(s.energyHistory) > 0 {
		var totalPower, maxPower float64
		for _, entry := range s.energyHistory {
			totalPower += entry.PowerConsumption
			if entry.PowerConsumption > maxPower {
				maxPower = entry.PowerConsumption
			}
		}
		stats.AveragePower = totalPower / float64(len(s.energyHistory))
		stats.PeakPower = maxPower
	}

	// Calculate savings (placeholder logic)
	stats.Savings = energy.EnergySavings{
		TotalSavings:        stats.TotalCost * 0.1, // 10% assumed savings
		AutomationSavings:   stats.TotalCost * 0.05,
		OptimizationSavings: stats.TotalCost * 0.03,
		SchedulingSavings:   stats.TotalCost * 0.02,
		PeriodDays:          s.settings.HistoricalPeriod,
	}

	return stats, nil
}

// GetEnergyDeviceBreakdown retrieves current device energy breakdown
func (s *Service) GetEnergyDeviceBreakdown() ([]energy.DeviceEnergyConsumption, error) {
	currentData, err := s.GetCurrentEnergyData()
	if err != nil {
		return nil, fmt.Errorf("failed to get current energy data: %w", err)
	}

	return currentData.DeviceBreakdown, nil
}

// GetDeviceEnergyData retrieves energy data for a specific device
func (s *Service) GetDeviceEnergyData(entityID string) (*energy.DeviceEnergyConsumption, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.isInitialized {
		return nil, fmt.Errorf("energy service not initialized")
	}

	// Get entities
	entities, err := s.getEntities()
	if err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}

	// Find the specific entity
	entityData, exists := entities[entityID]
	if !exists {
		return nil, fmt.Errorf("entity not found: %s", entityID)
	}

	entity, ok := entityData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid entity data for: %s", entityID)
	}

	// Calculate power consumption for this device
	powerConsumption := s.extractPowerConsumption(entity, entities)
	energyUsage := s.calculateEnergyUsage(entity, powerConsumption)
	cost := energyUsage * s.settings.EnergyRate

	// Get comprehensive energy data
	comprehensiveData := s.getComprehensiveEnergyData(entity, entities)

	deviceData := &energy.DeviceEnergyConsumption{
		EntityID:         entityID,
		DeviceName:       s.getStringValue(entity, "friendly_name"),
		Room:             s.getStringValue(entity, "area_id"),
		PowerConsumption: powerConsumption,
		EnergyUsage:      energyUsage,
		Cost:             cost,
		State:            s.getStringValue(entity, "state"),
		IsOn:             s.getStringValue(entity, "state") == "on",
		Current:          comprehensiveData.Current,
		Voltage:          comprehensiveData.Voltage,
		Frequency:        comprehensiveData.Frequency,
		ReturnedEnergy:   comprehensiveData.ReturnedEnergy,
		HasSensors:       comprehensiveData.Power > 0 || comprehensiveData.Current > 0,
		SensorsFound:     s.getSensorsFound(comprehensiveData),
	}

	return deviceData, nil
}

// StartTracking starts energy tracking (public method)
func (s *Service) StartTracking() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isInitialized {
		return fmt.Errorf("energy service not initialized")
	}

	if s.updateTicker != nil {
		return fmt.Errorf("tracking already started")
	}

	s.startTracking()

	// Update settings in database
	s.settings.TrackingEnabled = true
	ctx := context.Background()
	if err := s.repo.UpdateSettings(ctx, s.settings); err != nil {
		s.logger.WithError(err).Error("Failed to update tracking enabled setting")
		return fmt.Errorf("failed to update settings: %w", err)
	}

	s.logger.Info("Energy tracking started")
	return nil
}

// StopTracking stops energy tracking (public method)
func (s *Service) StopTracking() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isInitialized {
		return fmt.Errorf("energy service not initialized")
	}

	if s.updateTicker == nil {
		return fmt.Errorf("tracking not currently running")
	}

	s.stopTracking()

	// Update settings in database
	s.settings.TrackingEnabled = false
	ctx := context.Background()
	if err := s.repo.UpdateSettings(ctx, s.settings); err != nil {
		s.logger.WithError(err).Error("Failed to update tracking enabled setting")
		return fmt.Errorf("failed to update settings: %w", err)
	}

	s.logger.Info("Energy tracking stopped")
	return nil
}

// GetServiceStatus retrieves the current status of the energy service
func (s *Service) GetServiceStatus() (map[string]interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status := map[string]interface{}{
		"initialized":      s.isInitialized,
		"tracking_enabled": s.settings != nil && s.settings.TrackingEnabled,
		"tracking_active":  s.updateTicker != nil,
		"last_update_time": s.lastUpdateTime,
		"history_entries":  len(s.energyHistory),
		"cached_entities":  len(s.entityCache),
		"cache_expiry":     s.cacheExpiry,
		"service_uptime":   time.Since(s.lastUpdateTime),
	}

	if s.settings != nil {
		status["update_interval"] = s.settings.UpdateInterval
		status["energy_rate"] = s.settings.EnergyRate
		status["currency"] = s.settings.Currency
		status["historical_period"] = s.settings.HistoricalPeriod
	}

	return status, nil
}

// Helper functions

func (s *Service) getStringValue(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func (s *Service) parseFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return math.NaN()
}
