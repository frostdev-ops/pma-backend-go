package network

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/network"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service provides network management functionality
type Service struct {
	networkClient *network.Client
	networkRepo   repositories.NetworkRepository
	wsHub         WSHub
	logger        *logrus.Logger
}

// WSHub interface for WebSocket broadcasting
type WSHub interface {
	BroadcastToAll(messageType string, data interface{})
	BroadcastToTopic(topic, messageType string, data interface{})
}

// Config represents network service configuration
type Config struct {
	RouterBaseURL   string
	RouterAuthToken string
}

// NewService creates a new network service
func NewService(config Config, networkRepo repositories.NetworkRepository, wsHub WSHub, logger *logrus.Logger) *Service {
	client := network.NewClient(config.RouterBaseURL, config.RouterAuthToken)

	return &Service{
		networkClient: client,
		networkRepo:   networkRepo,
		wsHub:         wsHub,
		logger:        logger,
	}
}

// NetworkStatus represents the overall network status
type NetworkStatus struct {
	RouterStatus  *network.SystemStatus      `json:"router_status"`
	Interfaces    []network.NetworkInterface `json:"interfaces"`
	TrafficStats  *network.TrafficStats      `json:"traffic_stats"`
	DeviceCount   int                        `json:"device_count"`
	OnlineDevices int                        `json:"online_devices"`
	Timestamp     time.Time                  `json:"timestamp"`
}

// GetNetworkStatus retrieves comprehensive network status
func (s *Service) GetNetworkStatus(ctx context.Context) (*NetworkStatus, error) {
	// Get router status
	routerStatus, err := s.networkClient.GetSystemStatus(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get router status")
		return nil, fmt.Errorf("failed to get router status: %w", err)
	}

	// Get network interfaces
	interfaces, err := s.networkClient.GetNetworkInterfaces(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get network interfaces")
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Get traffic statistics
	trafficStats, err := s.networkClient.GetTrafficStats(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get traffic statistics")
		return nil, fmt.Errorf("failed to get traffic statistics: %w", err)
	}

	// Get discovered devices
	devicesResp, err := s.networkClient.GetNetworkDevices(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get network devices")
		return nil, fmt.Errorf("failed to get network devices: %w", err)
	}

	// Count online devices
	onlineDevices := 0
	for _, device := range devicesResp.Devices {
		if device.IsOnline {
			onlineDevices++
		}
	}

	status := &NetworkStatus{
		RouterStatus:  routerStatus,
		Interfaces:    interfaces,
		TrafficStats:  trafficStats,
		DeviceCount:   devicesResp.Count,
		OnlineDevices: onlineDevices,
		Timestamp:     time.Now(),
	}

	return status, nil
}

// GetDiscoveredDevices retrieves all discovered network devices
func (s *Service) GetDiscoveredDevices(ctx context.Context) (*network.NetworkDevicesResponse, error) {
	devices, err := s.networkClient.GetNetworkDevices(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get discovered devices")
		return nil, fmt.Errorf("failed to get discovered devices: %w", err)
	}

	// Update local cache if needed
	if err := s.cacheNetworkDevices(ctx, devices.Devices); err != nil {
		s.logger.WithError(err).Warn("Failed to cache network devices")
	}

	return devices, nil
}

// GetPortForwardingRules retrieves all port forwarding rules
func (s *Service) GetPortForwardingRules(ctx context.Context) (map[string]network.PortForwardingRule, error) {
	rules, err := s.networkClient.GetPortForwardingRules(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get port forwarding rules")
		return nil, fmt.Errorf("failed to get port forwarding rules: %w", err)
	}

	return rules, nil
}

// CreatePortForwardingRule creates a new port forwarding rule
func (s *Service) CreatePortForwardingRule(ctx context.Context, req network.CreatePortForwardingRequest) (*network.PortForwardingRule, error) {
	rule, err := s.networkClient.CreatePortForwardingRule(ctx, req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create port forwarding rule")
		return nil, fmt.Errorf("failed to create port forwarding rule: %w", err)
	}

	// Broadcast WebSocket event
	if s.wsHub != nil {
		s.wsHub.BroadcastToTopic("network", "port_forwarding_rule_created", map[string]interface{}{
			"rule": rule,
		})
	}

	s.logger.WithFields(logrus.Fields{
		"rule_id":       rule.ID,
		"internal_ip":   rule.InternalIP,
		"external_port": rule.ExternalPort,
	}).Info("Port forwarding rule created")

	return rule, nil
}

// UpdatePortForwardingRule updates an existing port forwarding rule
func (s *Service) UpdatePortForwardingRule(ctx context.Context, ruleID string, req network.CreatePortForwardingRequest) (*network.PortForwardingRule, error) {
	rule, err := s.networkClient.UpdatePortForwardingRule(ctx, ruleID, req)
	if err != nil {
		s.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to update port forwarding rule")
		return nil, fmt.Errorf("failed to update port forwarding rule: %w", err)
	}

	// Broadcast WebSocket event
	if s.wsHub != nil {
		s.wsHub.BroadcastToTopic("network", "port_forwarding_rule_updated", map[string]interface{}{
			"rule": rule,
		})
	}

	s.logger.WithFields(logrus.Fields{
		"rule_id":       rule.ID,
		"internal_ip":   rule.InternalIP,
		"external_port": rule.ExternalPort,
	}).Info("Port forwarding rule updated")

	return rule, nil
}

// DeletePortForwardingRule deletes a port forwarding rule
func (s *Service) DeletePortForwardingRule(ctx context.Context, ruleID string) error {
	err := s.networkClient.DeletePortForwardingRule(ctx, ruleID)
	if err != nil {
		s.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to delete port forwarding rule")
		return fmt.Errorf("failed to delete port forwarding rule: %w", err)
	}

	// Broadcast WebSocket event
	if s.wsHub != nil {
		s.wsHub.BroadcastToTopic("network", "port_forwarding_rule_deleted", map[string]interface{}{
			"rule_id": ruleID,
		})
	}

	s.logger.WithField("rule_id", ruleID).Info("Port forwarding rule deleted")

	return nil
}

// GetTrafficStatistics retrieves network traffic statistics
func (s *Service) GetTrafficStatistics(ctx context.Context) (*network.TrafficStats, error) {
	stats, err := s.networkClient.GetTrafficStats(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get traffic statistics")
		return nil, fmt.Errorf("failed to get traffic statistics: %w", err)
	}

	return stats, nil
}

// GetNetworkInterfaces retrieves network interface information
func (s *Service) GetNetworkInterfaces(ctx context.Context) ([]network.NetworkInterface, error) {
	interfaces, err := s.networkClient.GetNetworkInterfaces(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get network interfaces")
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	return interfaces, nil
}

// cacheNetworkDevices stores discovered network devices in the local database
func (s *Service) cacheNetworkDevices(ctx context.Context, devices []network.NetworkDevice) error {
	for _, device := range devices {
		// Convert open ports to JSON for services field
		var services []map[string]interface{}
		for _, port := range device.OpenPorts {
			services = append(services, map[string]interface{}{
				"port":     port.Port,
				"protocol": port.Protocol,
				"service":  port.Service,
				"version":  port.Version,
				"state":    port.State,
			})
		}
		servicesJSON, _ := json.Marshal(services)

		// Create metadata
		metadata := map[string]interface{}{
			"user_label":    device.UserLabel,
			"response_time": device.ResponseTime,
			"discovered_by": device.DiscoveredBy,
		}
		metadataJSON, _ := json.Marshal(metadata)

		dbDevice := &models.NetworkDevice{
			IPAddress:    device.IP,
			MACAddress:   sql.NullString{String: device.MAC, Valid: device.MAC != ""},
			Hostname:     sql.NullString{String: device.Hostname, Valid: device.Hostname != ""},
			Manufacturer: sql.NullString{String: device.Manufacturer, Valid: device.Manufacturer != ""},
			DeviceType:   sql.NullString{String: device.DeviceType, Valid: device.DeviceType != ""},
			IsOnline:     device.IsOnline,
			LastSeen:     time.Now(),
			FirstSeen:    time.Now(), // Will be updated if device already exists
			Services:     servicesJSON,
			Metadata:     metadataJSON,
		}

		// Try to get existing device first
		existingDevice, err := s.networkRepo.GetDevice(ctx, device.IP)
		if err != nil {
			// Device doesn't exist, create it
			if err := s.networkRepo.CreateDevice(ctx, dbDevice); err != nil {
				s.logger.WithError(err).WithField("device_ip", device.IP).Warn("Failed to create network device")
				continue
			}
		} else {
			// Device exists, update it but preserve FirstSeen
			dbDevice.ID = existingDevice.ID
			dbDevice.FirstSeen = existingDevice.FirstSeen
			if err := s.networkRepo.UpdateDevice(ctx, dbDevice); err != nil {
				s.logger.WithError(err).WithField("device_ip", device.IP).Warn("Failed to update network device")
				continue
			}
		}
	}

	return nil
}

// ScanNetworkDevices triggers a network scan and returns discovered devices
func (s *Service) ScanNetworkDevices(ctx context.Context) (*network.NetworkDevicesResponse, error) {
	// For now, just return the current discovered devices
	// In the future, we could trigger an active scan via PMA-Router API
	devices, err := s.GetDiscoveredDevices(ctx)
	if err != nil {
		return nil, err
	}

	// Broadcast WebSocket event
	if s.wsHub != nil {
		s.wsHub.BroadcastToTopic("network", "device_scan_completed", map[string]interface{}{
			"device_count":   devices.Count,
			"online_devices": s.countOnlineDevices(devices.Devices),
		})
	}

	s.logger.WithField("device_count", devices.Count).Info("Network device scan completed")

	return devices, nil
}

// countOnlineDevices counts how many devices are currently online
func (s *Service) countOnlineDevices(devices []network.NetworkDevice) int {
	count := 0
	for _, device := range devices {
		if device.IsOnline {
			count++
		}
	}
	return count
}

// DeviceWithPorts represents a network device with suggested port forwarding options
type DeviceWithPorts struct {
	Device         network.NetworkDevice        `json:"device"`
	SuggestedPorts []SuggestedPortRule          `json:"suggested_ports"`
	ExistingRules  []network.PortForwardingRule `json:"existing_rules"`
}

// SuggestedPortRule represents a suggested port forwarding rule
type SuggestedPortRule struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Service     string `json:"service"`
	Description string `json:"description"`
	Priority    int    `json:"priority"` // Higher = more important
}

// GetDevicesWithPortSuggestions returns devices with suggested port forwarding rules
func (s *Service) GetDevicesWithPortSuggestions(ctx context.Context) ([]DeviceWithPorts, error) {
	// Get discovered devices
	devicesResp, err := s.networkClient.GetNetworkDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get network devices: %w", err)
	}

	// Get existing port forwarding rules
	existingRules, err := s.networkClient.GetPortForwardingRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get port forwarding rules: %w", err)
	}

	var result []DeviceWithPorts

	for _, device := range devicesResp.Devices {
		deviceWithPorts := DeviceWithPorts{
			Device:         device,
			SuggestedPorts: s.generatePortSuggestions(device),
			ExistingRules:  s.findExistingRulesForDevice(device.IP, existingRules),
		}
		result = append(result, deviceWithPorts)
	}

	return result, nil
}

// generatePortSuggestions creates suggested port forwarding rules based on open ports
func (s *Service) generatePortSuggestions(device network.NetworkDevice) []SuggestedPortRule {
	var suggestions []SuggestedPortRule

	for _, openPort := range device.OpenPorts {
		suggestion := SuggestedPortRule{
			Port:     openPort.Port,
			Protocol: openPort.Protocol,
			Service:  openPort.Service,
			Priority: s.getServicePriority(openPort.Service),
		}

		// Generate description based on service
		switch openPort.Service {
		case "http":
			suggestion.Description = fmt.Sprintf("Web server access to %s", device.Hostname)
		case "https":
			suggestion.Description = fmt.Sprintf("Secure web server access to %s", device.Hostname)
		case "ssh":
			suggestion.Description = fmt.Sprintf("SSH access to %s", device.Hostname)
		case "ftp":
			suggestion.Description = fmt.Sprintf("FTP access to %s", device.Hostname)
		default:
			suggestion.Description = fmt.Sprintf("%s service access to %s", openPort.Service, device.Hostname)
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// getServicePriority returns priority for common services
func (s *Service) getServicePriority(service string) int {
	priorities := map[string]int{
		"http":  90,
		"https": 95,
		"ssh":   80,
		"ftp":   70,
		"smtp":  60,
		"pop3":  50,
		"imap":  50,
	}

	if priority, exists := priorities[service]; exists {
		return priority
	}
	return 30 // Default priority
}

// findExistingRulesForDevice finds existing port forwarding rules for a device
func (s *Service) findExistingRulesForDevice(deviceIP string, allRules map[string]network.PortForwardingRule) []network.PortForwardingRule {
	var deviceRules []network.PortForwardingRule

	for _, rule := range allRules {
		if rule.InternalIP == deviceIP {
			deviceRules = append(deviceRules, rule)
		}
	}

	return deviceRules
}
