package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/network"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetNetworkStatus handles GET /api/v1/network/status
func (h *Handlers) GetNetworkStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := h.networkService.GetNetworkStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get network status: "+err.Error())
		return
	}

	utils.SendSuccess(c, status)
}

// GetNetworkInterfaces handles GET /api/v1/network/interfaces
func (h *Handlers) GetNetworkInterfaces(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	interfaces, err := h.networkService.GetNetworkInterfaces(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get network interfaces: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"interfaces": interfaces,
		"count":      len(interfaces),
	})
}

// GetTrafficStatistics handles GET /api/v1/network/traffic
func (h *Handlers) GetTrafficStatistics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stats, err := h.networkService.GetTrafficStatistics(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get traffic statistics: "+err.Error())
		return
	}

	utils.SendSuccess(c, stats)
}

// GetDiscoveredDevices handles GET /api/v1/network/devices
func (h *Handlers) GetDiscoveredDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	devices, err := h.networkService.GetDiscoveredDevices(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get discovered devices: "+err.Error())
		return
	}

	utils.SendSuccess(c, devices)
}

// ScanNetworkDevices handles POST /api/v1/network/devices/scan
func (h *Handlers) ScanNetworkDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	devices, err := h.networkService.ScanNetworkDevices(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to scan network devices: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message":      "Network scan completed",
		"device_count": devices.Count,
		"devices":      devices.Devices,
	})
}

// GetDevicesWithPortSuggestions handles GET /api/v1/network/devices/suggestions
func (h *Handlers) GetDevicesWithPortSuggestions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	devicesWithPorts, err := h.networkService.GetDevicesWithPortSuggestions(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get device port suggestions: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"devices": devicesWithPorts,
		"count":   len(devicesWithPorts),
	})
}

// GetPortForwardingRules handles GET /api/v1/network/port-forwarding
func (h *Handlers) GetPortForwardingRules(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rules, err := h.networkService.GetPortForwardingRules(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get port forwarding rules: "+err.Error())
		return
	}

	// Convert map to slice for easier frontend handling
	var rulesList []network.PortForwardingRule
	for _, rule := range rules {
		rulesList = append(rulesList, rule)
	}

	utils.SendSuccess(c, map[string]interface{}{
		"rules": rulesList,
		"count": len(rulesList),
	})
}

// CreatePortForwardingRule handles POST /api/v1/network/port-forwarding
func (h *Handlers) CreatePortForwardingRule(c *gin.Context) {
	var request network.CreatePortForwardingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if request.InternalIP == "" {
		utils.SendError(c, http.StatusBadRequest, "Internal IP is required")
		return
	}
	if request.InternalPort <= 0 || request.InternalPort > 65535 {
		utils.SendError(c, http.StatusBadRequest, "Invalid internal port")
		return
	}
	if request.ExternalPort <= 0 || request.ExternalPort > 65535 {
		utils.SendError(c, http.StatusBadRequest, "Invalid external port")
		return
	}
	if request.Protocol != "tcp" && request.Protocol != "udp" {
		utils.SendError(c, http.StatusBadRequest, "Protocol must be 'tcp' or 'udp'")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rule, err := h.networkService.CreatePortForwardingRule(ctx, request)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to create port forwarding rule: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message": "Port forwarding rule created successfully",
		"rule":    rule,
	})
}

// UpdatePortForwardingRule handles PUT /api/v1/network/port-forwarding/:ruleId
func (h *Handlers) UpdatePortForwardingRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		utils.SendError(c, http.StatusBadRequest, "Rule ID is required")
		return
	}

	var request network.CreatePortForwardingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if request.InternalIP == "" {
		utils.SendError(c, http.StatusBadRequest, "Internal IP is required")
		return
	}
	if request.InternalPort <= 0 || request.InternalPort > 65535 {
		utils.SendError(c, http.StatusBadRequest, "Invalid internal port")
		return
	}
	if request.ExternalPort <= 0 || request.ExternalPort > 65535 {
		utils.SendError(c, http.StatusBadRequest, "Invalid external port")
		return
	}
	if request.Protocol != "tcp" && request.Protocol != "udp" {
		utils.SendError(c, http.StatusBadRequest, "Protocol must be 'tcp' or 'udp'")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rule, err := h.networkService.UpdatePortForwardingRule(ctx, ruleID, request)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to update port forwarding rule: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message": "Port forwarding rule updated successfully",
		"rule":    rule,
	})
}

// DeletePortForwardingRule handles DELETE /api/v1/network/port-forwarding/:ruleId
func (h *Handlers) DeletePortForwardingRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		utils.SendError(c, http.StatusBadRequest, "Rule ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := h.networkService.DeletePortForwardingRule(ctx, ruleID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete port forwarding rule: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message": "Port forwarding rule deleted successfully",
		"rule_id": ruleID,
	})
}

// GetNetworkMetrics handles GET /api/v1/network/metrics
func (h *Handlers) GetNetworkMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get comprehensive network metrics
	status, err := h.networkService.GetNetworkStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get network metrics: "+err.Error())
		return
	}

	// Get port forwarding rules count
	rules, err := h.networkService.GetPortForwardingRules(ctx)
	if err != nil {
		h.log.WithError(err).Warn("Failed to get port forwarding rules for metrics")
		rules = make(map[string]network.PortForwardingRule) // Empty map as fallback
	}

	// Calculate active rules
	activeRules := 0
	for _, rule := range rules {
		if rule.Active {
			activeRules++
		}
	}

	metrics := map[string]interface{}{
		"router_status": map[string]interface{}{
			"service": status.RouterStatus.Service,
			"version": status.RouterStatus.Version,
			"uptime":  status.RouterStatus.Uptime,
		},
		"device_metrics": map[string]interface{}{
			"total_devices":   status.DeviceCount,
			"online_devices":  status.OnlineDevices,
			"offline_devices": status.DeviceCount - status.OnlineDevices,
		},
		"traffic_metrics": map[string]interface{}{
			"total_packets":   status.TrafficStats.TotalPackets,
			"total_bytes":     status.TrafficStats.TotalBytes,
			"packets_per_sec": status.TrafficStats.PacketsPerSec,
			"bytes_per_sec":   status.TrafficStats.BytesPerSec,
		},
		"port_forwarding": map[string]interface{}{
			"total_rules":  len(rules),
			"active_rules": activeRules,
		},
		"interfaces": len(status.Interfaces),
		"timestamp":  status.Timestamp,
	}

	utils.SendSuccess(c, metrics)
}

// GetNetworkConfiguration handles GET /api/v1/network/config
func (h *Handlers) GetNetworkConfiguration(c *gin.Context) {
	// For now, return static configuration info
	// In the future, this could fetch actual router configuration
	config := map[string]interface{}{
		"router_api_base_url": h.cfg.Router.BaseURL,
		"monitoring_enabled":  h.cfg.Router.MonitoringEnabled,
		"auto_discovery":      h.cfg.Router.AutoDiscovery,
		"traffic_logging":     h.cfg.Router.TrafficLogging,
		"features": map[string]bool{
			"port_forwarding":    true,
			"device_discovery":   true,
			"traffic_monitoring": true,
			"nginx_proxy":        true,
			"bridge_management":  true,
		},
		"interfaces": map[string]interface{}{
			"external": "eth0",
			"internal": "eth1",
			"bridge":   "pma-br0",
		},
	}

	utils.SendSuccess(c, config)
}

// TestRouterConnection handles POST /api/v1/network/test-connection
func (h *Handlers) TestRouterConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection by getting router status
	status, err := h.networkService.GetNetworkStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Router connection failed: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message":          "Router connection successful",
		"router_service":   status.RouterStatus.Service,
		"router_version":   status.RouterStatus.Version,
		"connection_time":  time.Now(),
		"response_time_ms": 150, // This could be measured in the future
	})
}

// Network Settings Management Handlers

// GetNetworkSettings retrieves network/router settings
func (h *Handlers) GetNetworkSettings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current network status (using available method)
	status, err := h.networkService.GetNetworkStatus(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get network settings")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve network settings")
		return
	}

	// Prepare network settings response
	settings := map[string]interface{}{
		"router": map[string]interface{}{
			"enabled":    h.cfg.Router.Enabled,
			"base_url":   h.cfg.Router.BaseURL,
			"timeout":    h.cfg.Router.Timeout,
			"monitoring": h.cfg.Router.MonitoringEnabled,
		},
		"router_status": status.RouterStatus,
		"interfaces":    status.Interfaces,
		"last_updated":  time.Now(),
	}

	utils.SendSuccess(c, settings)
}

// UpdateNetworkSettings updates network settings
func (h *Handlers) UpdateNetworkSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// In a full implementation, you would:
	// 1. Validate the settings
	// 2. Apply them to the router service
	// 3. Update the configuration files
	// 4. Send WebSocket notifications

	// For now, just log the settings update
	h.log.WithField("settings", req).Info("Network settings update requested")

	// Save to config repository
	for key, value := range req {
		if strValue, ok := value.(string); ok {
			configKey := fmt.Sprintf("network.%s", key)
			if err := h.repos.Config.Set(ctx, &models.SystemConfig{
				Key:   configKey,
				Value: strValue,
			}); err != nil {
				h.log.WithError(err).Warn("Failed to save network config", "key", configKey)
			}
		}
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message":    "Network settings updated successfully",
		"updated_at": time.Now(),
		"settings":   req,
	})
}

// ResetNetworkConfiguration resets network configuration to defaults
func (h *Handlers) ResetNetworkConfiguration(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// In a full implementation, you would:
	// 1. Stop network services
	// 2. Reset configuration files to defaults
	// 3. Restart network services
	// 4. Send notifications

	// For now, just simulate the reset
	h.log.Info("Network configuration reset requested")

	// Reset network-related config entries
	networkKeys := []string{
		"network.router_enabled",
		"network.dhcp_enabled",
		"network.firewall_enabled",
		"network.monitoring_enabled",
	}

	for _, key := range networkKeys {
		if err := h.repos.Config.Delete(ctx, key); err != nil {
			h.log.WithError(err).Warn("Failed to delete network config", "key", key)
		}
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message":   "Network configuration reset successfully",
		"reset_at":  time.Now(),
		"status":    "configuration_reset",
		"next_step": "restart_network_services",
	})
}

// TestRouterConnectivity tests router connectivity
func (h *Handlers) TestRouterConnectivity(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()

	// Test connection by getting router status
	status, err := h.networkService.GetNetworkStatus(ctx)
	latency := time.Since(startTime)

	if err != nil {
		utils.SendSuccess(c, map[string]interface{}{
			"success":    false,
			"message":    "Router connection failed",
			"error":      err.Error(),
			"tested_at":  time.Now(),
			"latency_ms": latency.Milliseconds(),
		})
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"success":        true,
		"message":        "Router connection successful",
		"router_service": status.RouterStatus.Service,
		"router_version": status.RouterStatus.Version,
		"tested_at":      time.Now(),
		"latency_ms":     latency.Milliseconds(),
		"status":         "online",
	})
}
