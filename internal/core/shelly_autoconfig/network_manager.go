package shelly_autoconfig

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NetworkManagerImpl implements the NetworkManager interface
type NetworkManagerImpl struct {
	logger                *logrus.Logger
	config                NetworkManagerConfig
	requiredExternalConns int
}

// NetworkManagerConfig holds configuration for network operations
type NetworkManagerConfig struct {
	WiFiInterface           string        `json:"wifi_interface"`
	RequiredExternalConns   int           `json:"required_external_connections"`
	ConnectionTimeout       time.Duration `json:"connection_timeout"`
	EnableNetworkCommands   bool          `json:"enable_network_commands"`
	PreferredExternalIfaces []string      `json:"preferred_external_interfaces"`
	ExcludeInterfaces       []string      `json:"exclude_interfaces"`
}

// NewNetworkManagerImpl creates a new network manager implementation
func NewNetworkManagerImpl(logger *logrus.Logger, config NetworkManagerConfig) *NetworkManagerImpl {
	// Set defaults
	if config.WiFiInterface == "" {
		config.WiFiInterface = "wlan0"
	}
	if config.RequiredExternalConns == 0 {
		config.RequiredExternalConns = 1
	}
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 30 * time.Second
	}
	if len(config.PreferredExternalIfaces) == 0 {
		config.PreferredExternalIfaces = []string{"eth0", "eth1", "en0", "en1"}
	}
	if len(config.ExcludeInterfaces) == 0 {
		config.ExcludeInterfaces = []string{"lo", "docker0", "br-"}
	}

	return &NetworkManagerImpl{
		logger:                logger,
		config:                config,
		requiredExternalConns: config.RequiredExternalConns,
	}
}

// GetActiveInterfaces returns all active network interfaces
func (nm *NetworkManagerImpl) GetActiveInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var result []NetworkInterface

	for _, iface := range interfaces {
		// Skip excluded interfaces
		if nm.isExcludedInterface(iface.Name) {
			continue
		}

		// Get interface addresses
		addrs, err := iface.Addrs()
		if err != nil {
			nm.logger.WithError(err).WithField("interface", iface.Name).Warn("Failed to get interface addresses")
			continue
		}

		// Determine if interface is active (has IP addresses)
		var ipAddress string
		isActive := false
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					ipAddress = ipNet.IP.String()
					isActive = true
					break
				}
			}
		}

		// Determine interface type
		interfaceType := nm.determineInterfaceType(iface.Name)

		// Get WiFi SSID if it's a WiFi interface
		var ssid string
		if interfaceType == "wifi" && isActive {
			ssid = nm.getWiFiSSID(iface.Name)
		}

		netInterface := NetworkInterface{
			Name:      iface.Name,
			Type:      interfaceType,
			IsActive:  isActive,
			IPAddress: ipAddress,
			SSID:      ssid,
		}

		result = append(result, netInterface)
	}

	return result, nil
}

// ConnectToWiFi connects to a WiFi network
func (nm *NetworkManagerImpl) ConnectToWiFi(ctx context.Context, ssid, password string) error {
	if !nm.config.EnableNetworkCommands {
		return fmt.Errorf("network commands are disabled in configuration")
	}

	nm.logger.WithFields(logrus.Fields{
		"ssid":      ssid,
		"interface": nm.config.WiFiInterface,
	}).Info("Connecting to WiFi network")

	// Create connection context with timeout
	connectCtx, cancel := context.WithTimeout(ctx, nm.config.ConnectionTimeout)
	defer cancel()

	// Disconnect from current WiFi if connected
	if err := nm.disconnectCurrentWiFi(connectCtx); err != nil {
		nm.logger.WithError(err).Warn("Failed to disconnect from current WiFi, continuing anyway")
	}

	// Scan for the target SSID
	if err := nm.scanForSSID(connectCtx, ssid); err != nil {
		return fmt.Errorf("failed to find SSID %s: %w", ssid, err)
	}

	// Connect to the WiFi network
	if err := nm.connectWiFi(connectCtx, ssid, password); err != nil {
		return fmt.Errorf("failed to connect to WiFi: %w", err)
	}

	// Verify connection
	if err := nm.verifyWiFiConnection(connectCtx, ssid); err != nil {
		return fmt.Errorf("WiFi connection verification failed: %w", err)
	}

	nm.logger.WithField("ssid", ssid).Info("Successfully connected to WiFi")
	return nil
}

// DisconnectFromWiFi disconnects from a WiFi network
func (nm *NetworkManagerImpl) DisconnectFromWiFi(ctx context.Context, interfaceName string) error {
	if !nm.config.EnableNetworkCommands {
		return fmt.Errorf("network commands are disabled in configuration")
	}

	if interfaceName == "" {
		interfaceName = nm.config.WiFiInterface
	}

	nm.logger.WithField("interface", interfaceName).Info("Disconnecting from WiFi")

	// Use NetworkManager if available
	if nm.isNetworkManagerAvailable() {
		return nm.disconnectWiFiNetworkManager(ctx, interfaceName)
	}

	// Use wpa_supplicant if available
	if nm.isWpaSupplicantAvailable() {
		return nm.disconnectWiFiWpaSupplicant(ctx, interfaceName)
	}

	// Fallback to basic interface down/up
	return nm.disconnectWiFiBasic(ctx, interfaceName)
}

// BackupNetworkConfig backs up current network configuration
func (nm *NetworkManagerImpl) BackupNetworkConfig() (*NetworkBackup, error) {
	backup := &NetworkBackup{
		BackupTime:      time.Now(),
		RestoreRequired: false,
	}

	// Get current WiFi interface and SSID
	currentSSID := nm.getWiFiSSID(nm.config.WiFiInterface)
	if currentSSID != "" {
		backup.OriginalInterface = nm.config.WiFiInterface
		backup.OriginalSSID = currentSSID
		backup.RestoreRequired = true
	}

	nm.logger.WithFields(logrus.Fields{
		"interface":        backup.OriginalInterface,
		"ssid":             backup.OriginalSSID,
		"restore_required": backup.RestoreRequired,
	}).Info("Network configuration backed up")

	return backup, nil
}

// RestoreNetworkConfig restores network configuration from backup
func (nm *NetworkManagerImpl) RestoreNetworkConfig(backup *NetworkBackup) error {
	if !backup.RestoreRequired {
		nm.logger.Info("No network restore required")
		return nil
	}

	if !nm.config.EnableNetworkCommands {
		return fmt.Errorf("network commands are disabled in configuration")
	}

	nm.logger.WithFields(logrus.Fields{
		"interface": backup.OriginalInterface,
		"ssid":      backup.OriginalSSID,
	}).Info("Restoring network configuration")

	ctx, cancel := context.WithTimeout(context.Background(), nm.config.ConnectionTimeout)
	defer cancel()

	// Attempt to reconnect to original network
	// Note: This would require stored credentials, which is complex
	// For now, we'll just disconnect from current and let system auto-reconnect
	if err := nm.DisconnectFromWiFi(ctx, backup.OriginalInterface); err != nil {
		nm.logger.WithError(err).Warn("Failed to disconnect during restore")
	}

	// Wait a moment for auto-reconnection
	time.Sleep(5 * time.Second)

	nm.logger.Info("Network configuration restore attempted")
	return nil
}

// IsNetworkSafe checks if it's safe to modify network connections
func (nm *NetworkManagerImpl) IsNetworkSafe() (bool, error) {
	interfaces, err := nm.GetActiveInterfaces()
	if err != nil {
		return false, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Count active external connections (non-WiFi, non-loopback)
	externalConnections := 0
	wifiConnections := 0

	for _, iface := range interfaces {
		if !iface.IsActive {
			continue
		}

		if iface.Type == "wifi" {
			wifiConnections++
		} else if iface.Type == "ethernet" {
			externalConnections++
		}
	}

	// We need at least one external connection if we're going to modify WiFi
	isSafe := externalConnections >= nm.requiredExternalConns

	nm.logger.WithFields(logrus.Fields{
		"external_connections": externalConnections,
		"wifi_connections":     wifiConnections,
		"required_external":    nm.requiredExternalConns,
		"is_safe":              isSafe,
	}).Debug("Network safety check")

	return isSafe, nil
}

// Helper methods

func (nm *NetworkManagerImpl) isExcludedInterface(name string) bool {
	for _, excluded := range nm.config.ExcludeInterfaces {
		if strings.Contains(name, excluded) {
			return true
		}
	}
	return false
}

func (nm *NetworkManagerImpl) determineInterfaceType(name string) string {
	// Common WiFi interface patterns
	wifiPatterns := []string{"wlan", "wifi", "wl"}
	for _, pattern := range wifiPatterns {
		if strings.Contains(name, pattern) {
			return "wifi"
		}
	}

	// Common Ethernet interface patterns
	ethernetPatterns := []string{"eth", "en", "em", "p"}
	for _, pattern := range ethernetPatterns {
		if strings.HasPrefix(name, pattern) {
			return "ethernet"
		}
	}

	// Default to unknown
	return "unknown"
}

func (nm *NetworkManagerImpl) getWiFiSSID(interfaceName string) string {
	if !nm.config.EnableNetworkCommands {
		return ""
	}

	// Try iwgetid first
	if cmd := exec.Command("iwgetid", "-r", interfaceName); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			ssid := strings.TrimSpace(string(output))
			if ssid != "" {
				return ssid
			}
		}
	}

	// Try iw command
	if cmd := exec.Command("iw", "dev", interfaceName, "link"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			// Parse output for SSID
			re := regexp.MustCompile(`SSID:\s*(.+)`)
			if matches := re.FindStringSubmatch(string(output)); len(matches) > 1 {
				return strings.TrimSpace(matches[1])
			}
		}
	}

	return ""
}

func (nm *NetworkManagerImpl) disconnectCurrentWiFi(ctx context.Context) error {
	return nm.DisconnectFromWiFi(ctx, nm.config.WiFiInterface)
}

func (nm *NetworkManagerImpl) scanForSSID(ctx context.Context, ssid string) error {
	// Trigger WiFi scan
	if cmd := exec.CommandContext(ctx, "iwlist", nm.config.WiFiInterface, "scan"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			// Check if SSID is in scan results
			if strings.Contains(string(output), fmt.Sprintf(`ESSID:"%s"`, ssid)) {
				return nil
			}
		}
	}

	// Try with iw command
	if cmd := exec.CommandContext(ctx, "iw", "dev", nm.config.WiFiInterface, "scan"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			if strings.Contains(string(output), ssid) {
				return nil
			}
		}
	}

	return fmt.Errorf("SSID %s not found in scan results", ssid)
}

func (nm *NetworkManagerImpl) connectWiFi(ctx context.Context, ssid, password string) error {
	// Try NetworkManager first
	if nm.isNetworkManagerAvailable() {
		return nm.connectWiFiNetworkManager(ctx, ssid, password)
	}

	// Try wpa_supplicant
	if nm.isWpaSupplicantAvailable() {
		return nm.connectWiFiWpaSupplicant(ctx, ssid, password)
	}

	return fmt.Errorf("no supported WiFi management tools available")
}

func (nm *NetworkManagerImpl) isNetworkManagerAvailable() bool {
	_, err := exec.LookPath("nmcli")
	return err == nil
}

func (nm *NetworkManagerImpl) isWpaSupplicantAvailable() bool {
	_, err := exec.LookPath("wpa_supplicant")
	return err == nil
}

func (nm *NetworkManagerImpl) connectWiFiNetworkManager(ctx context.Context, ssid, password string) error {
	// Use nmcli to connect
	args := []string{"device", "wifi", "connect", ssid}
	if password != "" {
		args = append(args, "password", password)
	}
	args = append(args, "ifname", nm.config.WiFiInterface)

	cmd := exec.CommandContext(ctx, "nmcli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nmcli connect failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (nm *NetworkManagerImpl) disconnectWiFiNetworkManager(ctx context.Context, interfaceName string) error {
	cmd := exec.CommandContext(ctx, "nmcli", "device", "disconnect", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nmcli disconnect failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (nm *NetworkManagerImpl) connectWiFiWpaSupplicant(ctx context.Context, ssid, password string) error {
	// This is a simplified implementation
	// In practice, you'd need to manage wpa_supplicant configuration files
	return fmt.Errorf("wpa_supplicant connection not implemented yet")
}

func (nm *NetworkManagerImpl) disconnectWiFiWpaSupplicant(ctx context.Context, interfaceName string) error {
	// This is a simplified implementation
	return fmt.Errorf("wpa_supplicant disconnect not implemented yet")
}

func (nm *NetworkManagerImpl) disconnectWiFiBasic(ctx context.Context, interfaceName string) error {
	// Bring interface down then up
	if cmd := exec.CommandContext(ctx, "ip", "link", "set", interfaceName, "down"); cmd != nil {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to bring interface down: %w", err)
		}
	}

	time.Sleep(2 * time.Second)

	if cmd := exec.CommandContext(ctx, "ip", "link", "set", interfaceName, "up"); cmd != nil {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to bring interface up: %w", err)
		}
	}

	return nil
}

func (nm *NetworkManagerImpl) verifyWiFiConnection(ctx context.Context, expectedSSID string) error {
	// Wait for connection to establish
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for WiFi connection")
		case <-ticker.C:
			currentSSID := nm.getWiFiSSID(nm.config.WiFiInterface)
			if currentSSID == expectedSSID {
				// Verify we have an IP address
				interfaces, err := nm.GetActiveInterfaces()
				if err != nil {
					continue
				}

				for _, iface := range interfaces {
					if iface.Name == nm.config.WiFiInterface && iface.IsActive {
						return nil
					}
				}
			}
		}
	}
}
