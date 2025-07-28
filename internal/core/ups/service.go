package ups

import (
	"context"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/ups"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service provides UPS monitoring functionality
type Service struct {
	nutClient  *ups.NUTClient
	upsRepo    repositories.UPSRepository
	wsHub      WSHub
	logger     *logrus.Logger
	monitoring bool
	config     Config
	stopChan   chan struct{}
}

// WSHub interface for WebSocket broadcasting
type WSHub interface {
	BroadcastToAll(messageType string, data interface{})
	BroadcastToTopic(topic, messageType string, data interface{})
}

// Config represents UPS service configuration
type Config struct {
	NUTHost            string
	NUTPort            int
	UPSName            string
	MonitoringInterval time.Duration
	HistoryRetention   int // days
	AlertThresholds    AlertThresholds
}

// AlertThresholds defines UPS alert thresholds
type AlertThresholds struct {
	LowBattery      float64 // percentage
	CriticalBattery float64 // percentage
	HighTemperature float64 // celsius
	HighLoad        float64 // percentage
}

// UPSStatus represents the current UPS status with additional computed fields
type UPSStatus struct {
	*models.UPSStatus
	ConnectionStatus string                 `json:"connection_status"`
	AlertLevel       string                 `json:"alert_level"`
	Alerts           []Alert                `json:"alerts"`
	UPSInfo          *ups.UPSData           `json:"ups_info,omitempty"`
	Variables        map[string]interface{} `json:"variables,omitempty"`
}

// Alert represents a UPS alert
type Alert struct {
	Type         string    `json:"type"`
	Level        string    `json:"level"` // info, warning, critical
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

// NewService creates a new UPS service
func NewService(config Config, upsRepo repositories.UPSRepository, wsHub WSHub, logger *logrus.Logger) *Service {
	// Set default values
	if config.MonitoringInterval == 0 {
		config.MonitoringInterval = 30 * time.Second
	}
	if config.HistoryRetention == 0 {
		config.HistoryRetention = 30 // days
	}
	if config.AlertThresholds.LowBattery == 0 {
		config.AlertThresholds.LowBattery = 20.0
	}
	if config.AlertThresholds.CriticalBattery == 0 {
		config.AlertThresholds.CriticalBattery = 10.0
	}
	if config.AlertThresholds.HighTemperature == 0 {
		config.AlertThresholds.HighTemperature = 40.0
	}
	if config.AlertThresholds.HighLoad == 0 {
		config.AlertThresholds.HighLoad = 80.0
	}

	nutClient := ups.NewNUTClient(config.NUTHost, config.NUTPort, logger)

	return &Service{
		nutClient: nutClient,
		upsRepo:   upsRepo,
		wsHub:     wsHub,
		logger:    logger,
		config:    config,
		stopChan:  make(chan struct{}),
	}
}

// GetCurrentStatus retrieves the current UPS status from NUT and database
func (s *Service) GetCurrentStatus(ctx context.Context) (*UPSStatus, error) {
	// Get latest database status
	dbStatus, err := s.upsRepo.GetLatestStatus(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get latest UPS status from database")
		// Continue with live data if database is empty
	}

	// Try to get live data from NUT
	liveStatus, err := s.getLiveUPSStatus(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get live UPS status")

		// If we have database status, return it with connection error
		if dbStatus != nil {
			upsStatus := &UPSStatus{
				UPSStatus:        dbStatus,
				ConnectionStatus: "disconnected",
				AlertLevel:       s.calculateAlertLevel(dbStatus),
				Alerts:           s.generateAlerts(dbStatus),
			}
			return upsStatus, nil
		}

		return nil, fmt.Errorf("no UPS data available: %w", err)
	}

	// Generate alerts and calculate alert level
	alerts := s.generateAlerts(liveStatus)
	alertLevel := s.calculateAlertLevel(liveStatus)

	upsStatus := &UPSStatus{
		UPSStatus:        liveStatus,
		ConnectionStatus: "connected",
		AlertLevel:       alertLevel,
		Alerts:           alerts,
	}

	return upsStatus, nil
}

// getLiveUPSStatus gets current UPS status from NUT server
func (s *Service) getLiveUPSStatus(ctx context.Context) (*models.UPSStatus, error) {
	if err := s.nutClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to NUT server: %w", err)
	}
	defer s.nutClient.Close()

	// Get UPS data from NUT
	upsData, err := s.nutClient.GetUPSData(ctx, s.config.UPSName)
	if err != nil {
		return nil, fmt.Errorf("failed to get UPS data: %w", err)
	}

	// Convert to database model
	status := &models.UPSStatus{
		BatteryCharge:  getFloatValue(upsData.BatteryCharge),
		BatteryRuntime: int(getFloatValue(upsData.BatteryRuntime)),
		InputVoltage:   getFloatValue(upsData.InputVoltage),
		OutputVoltage:  getFloatValue(upsData.OutputVoltage),
		Load:           getFloatValue(upsData.LoadPercent),
		Status:         upsData.Status,
		Temperature:    getFloatValue(upsData.Temperature),
		LastUpdated:    time.Now(),
	}

	return status, nil
}

// getFloatValue safely extracts float value from pointer
func getFloatValue(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}

// StartMonitoring begins continuous UPS monitoring
func (s *Service) StartMonitoring(ctx context.Context) error {
	if s.monitoring {
		return fmt.Errorf("monitoring already started")
	}

	s.monitoring = true
	s.logger.Info("Starting UPS monitoring")

	// Start background monitoring goroutine
	go s.monitoringLoop(ctx)

	// Start cleanup routine
	go s.cleanupLoop(ctx)

	return nil
}

// StopMonitoring stops continuous UPS monitoring
func (s *Service) StopMonitoring() error {
	if !s.monitoring {
		return fmt.Errorf("monitoring not started")
	}

	s.monitoring = false
	close(s.stopChan)
	s.logger.Info("Stopping UPS monitoring")

	return nil
}

// monitoringLoop continuously monitors UPS status
func (s *Service) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.collectAndStoreStatus(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to collect UPS status")
			}
		}
	}
}

// collectAndStoreStatus collects current UPS status and stores it
func (s *Service) collectAndStoreStatus(ctx context.Context) error {
	status, err := s.getLiveUPSStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get live UPS status: %w", err)
	}

	// Store in database
	if err := s.upsRepo.CreateStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to store UPS status: %w", err)
	}

	// Generate alerts
	alerts := s.generateAlerts(status)
	alertLevel := s.calculateAlertLevel(status)

	// Broadcast WebSocket event if there are alerts or status changes
	if len(alerts) > 0 || alertLevel != "normal" {
		upsStatus := &UPSStatus{
			UPSStatus:        status,
			ConnectionStatus: "connected",
			AlertLevel:       alertLevel,
			Alerts:           alerts,
		}

		if s.wsHub != nil {
			s.wsHub.BroadcastToTopic("ups", "ups_status_update", upsStatus)
		}
	}

	// Log critical alerts
	for _, alert := range alerts {
		if alert.Level == "critical" {
			s.logger.WithFields(logrus.Fields{
				"alert_type": alert.Type,
				"message":    alert.Message,
			}).Warn("Critical UPS alert")
		}
	}

	return nil
}

// cleanupLoop periodically cleans up old UPS status records
func (s *Service) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Daily cleanup
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.upsRepo.CleanupOldStatus(ctx, s.config.HistoryRetention); err != nil {
				s.logger.WithError(err).Error("Failed to cleanup old UPS status records")
			} else {
				s.logger.Debug("Cleaned up old UPS status records")
			}
		}
	}
}

// generateAlerts checks UPS status and generates alerts
func (s *Service) generateAlerts(status *models.UPSStatus) []Alert {
	var alerts []Alert
	now := time.Now()

	// Low battery alert
	if status.BatteryCharge <= s.config.AlertThresholds.CriticalBattery {
		alerts = append(alerts, Alert{
			Type:      "battery_critical",
			Level:     "critical",
			Message:   fmt.Sprintf("Battery critically low: %.1f%%", status.BatteryCharge),
			Timestamp: now,
		})
	} else if status.BatteryCharge <= s.config.AlertThresholds.LowBattery {
		alerts = append(alerts, Alert{
			Type:      "battery_low",
			Level:     "warning",
			Message:   fmt.Sprintf("Battery low: %.1f%%", status.BatteryCharge),
			Timestamp: now,
		})
	}

	// High temperature alert
	if status.Temperature >= s.config.AlertThresholds.HighTemperature {
		alerts = append(alerts, Alert{
			Type:      "temperature_high",
			Level:     "warning",
			Message:   fmt.Sprintf("High temperature: %.1f°C", status.Temperature),
			Timestamp: now,
		})
	}

	// High load alert
	if status.Load >= s.config.AlertThresholds.HighLoad {
		alerts = append(alerts, Alert{
			Type:      "load_high",
			Level:     "warning",
			Message:   fmt.Sprintf("High load: %.1f%%", status.Load),
			Timestamp: now,
		})
	}

	// UPS status alerts
	if status.Status != "OL" && status.Status != "OL CHRG" { // Not online or online charging
		level := "warning"
		if status.Status == "OB" || status.Status == "OB DISCHRG" { // On battery
			level = "critical"
		}

		alerts = append(alerts, Alert{
			Type:      "ups_status",
			Level:     level,
			Message:   fmt.Sprintf("UPS status: %s", status.Status),
			Timestamp: now,
		})
	}

	return alerts
}

// calculateAlertLevel determines the overall alert level
func (s *Service) calculateAlertLevel(status *models.UPSStatus) string {
	if status.BatteryCharge <= s.config.AlertThresholds.CriticalBattery {
		return "critical"
	}

	if status.Status == "OB" || status.Status == "OB DISCHRG" {
		return "critical"
	}

	if status.BatteryCharge <= s.config.AlertThresholds.LowBattery ||
		status.Temperature >= s.config.AlertThresholds.HighTemperature ||
		status.Load >= s.config.AlertThresholds.HighLoad {
		return "warning"
	}

	return "normal"
}

// GetStatusHistory retrieves UPS status history
func (s *Service) GetStatusHistory(ctx context.Context, limit int) ([]*models.UPSStatus, error) {
	return s.upsRepo.GetStatusHistory(ctx, limit)
}

// GetBatteryTrends retrieves battery trends for analytics
func (s *Service) GetBatteryTrends(ctx context.Context, hours int) ([]*models.UPSStatus, error) {
	return s.upsRepo.GetBatteryTrends(ctx, hours)
}

// GetUPSInfo retrieves detailed UPS information from NUT
func (s *Service) GetUPSInfo(ctx context.Context) (*ups.UPSData, error) {
	if err := s.nutClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to NUT server: %w", err)
	}
	defer s.nutClient.Close()

	return s.nutClient.GetUPSData(ctx, s.config.UPSName)
}

// GetUPSVariables retrieves all UPS variables from NUT
func (s *Service) GetUPSVariables(ctx context.Context) (map[string]ups.UPSVariable, error) {
	if err := s.nutClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to NUT server: %w", err)
	}
	defer s.nutClient.Close()

	return s.nutClient.GetUPSVariables(ctx, s.config.UPSName)
}

// TestConnection tests the connection to the NUT server
func (s *Service) TestConnection(ctx context.Context) error {
	if err := s.nutClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to NUT server: %w", err)
	}
	defer s.nutClient.Close()

	return s.nutClient.Ping(ctx)
}

// GetConnectionInfo returns information about the NUT connection
func (s *Service) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	info := map[string]interface{}{
		"nut_host":   s.config.NUTHost,
		"nut_port":   s.config.NUTPort,
		"ups_name":   s.config.UPSName,
		"monitoring": s.monitoring,
		"config":     s.config,
	}

	// Try to get NUT version
	if err := s.nutClient.Connect(ctx); err == nil {
		defer s.nutClient.Close()

		if version, err := s.nutClient.GetNUTVersion(ctx); err == nil {
			info["nut_version"] = version
		}

		if upsList, err := s.nutClient.ListUPS(ctx); err == nil {
			info["available_ups"] = upsList
		}
	}

	return info, nil
}

// IsMonitoring returns whether UPS monitoring is currently active
func (s *Service) IsMonitoring() bool {
	return s.monitoring
}

// GetAlertThresholds returns the current alert thresholds
func (s *Service) GetAlertThresholds(ctx context.Context) (*AlertThresholds, error) {
	return &s.config.AlertThresholds, nil
}

// UpdateAlertThresholds updates the UPS alert thresholds and persists them
func (s *Service) UpdateAlertThresholds(ctx context.Context, thresholds *AlertThresholds) error {
	// Validate thresholds
	if thresholds.LowBattery < 0 || thresholds.LowBattery > 100 {
		return fmt.Errorf("low battery threshold must be between 0 and 100")
	}
	if thresholds.CriticalBattery < 0 || thresholds.CriticalBattery > 100 {
		return fmt.Errorf("critical battery threshold must be between 0 and 100")
	}
	if thresholds.HighTemperature < -20 || thresholds.HighTemperature > 100 {
		return fmt.Errorf("high temperature threshold must be between -20 and 100")
	}
	if thresholds.HighLoad < 0 || thresholds.HighLoad > 100 {
		return fmt.Errorf("high load threshold must be between 0 and 100")
	}

	// Update the in-memory config
	s.config.AlertThresholds = *thresholds

	s.logger.WithFields(map[string]interface{}{
		"low_battery":      thresholds.LowBattery,
		"critical_battery": thresholds.CriticalBattery,
		"high_temperature": thresholds.HighTemperature,
		"high_load":        thresholds.HighLoad,
	}).Info("UPS alert thresholds updated")

	return nil
}

// CheckAlerts checks current UPS status against configured thresholds
func (s *Service) CheckAlerts(ctx context.Context, status *UPSStatus) []string {
	var alerts []string
	thresholds := s.config.AlertThresholds

	if status.BatteryCharge <= thresholds.CriticalBattery {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: Battery charge is %.1f%%, below critical threshold of %.1f%%", status.BatteryCharge, thresholds.CriticalBattery))
	} else if status.BatteryCharge <= thresholds.LowBattery {
		alerts = append(alerts, fmt.Sprintf("WARNING: Battery charge is %.1f%%, below low threshold of %.1f%%", status.BatteryCharge, thresholds.LowBattery))
	}

	if status.Load >= thresholds.HighLoad {
		alerts = append(alerts, fmt.Sprintf("WARNING: UPS load is %.1f%%, above high load threshold of %.1f%%", status.Load, thresholds.HighLoad))
	}

	// Temperature alert (if temperature data is available)
	if status.Temperature > 0 && float64(status.Temperature) >= thresholds.HighTemperature {
		alerts = append(alerts, fmt.Sprintf("WARNING: UPS temperature is %.1f°C, above high temperature threshold of %.1f°C", status.Temperature, thresholds.HighTemperature))
	}

	return alerts
}
