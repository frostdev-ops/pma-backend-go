package shelly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/sirupsen/logrus"
)

const (
	shellyServiceType      = "_shelly._tcp"
	shellyDomain           = "local."
	httpTimeout            = 10 * time.Second
	defaultUsername        = "admin"
	discoveryTimeout       = 30 * time.Second
	networkScanConcurrency = 10 // Reduced from 50 to prevent goroutine explosion
	healthCheckTimeout     = 5 * time.Second
	maxConcurrentHealth    = 5                // Limit concurrent health checks
	maxDevicesPerScan      = 100              // Limit devices discovered per scan
	scanInterval           = 10 * time.Minute // Increased from 5 minutes
	healthCheckInterval    = 2 * time.Minute  // Increased from 60 seconds
)

// DeviceGeneration represents the Shelly device generation
type DeviceGeneration int

const (
	Gen1       DeviceGeneration = 1
	Gen2       DeviceGeneration = 2
	Gen3       DeviceGeneration = 3
	Gen4       DeviceGeneration = 4
	GenUnknown DeviceGeneration = 0
)

// DiscoveryMethod represents how a device was discovered
type DiscoveryMethod int

const (
	DiscoveryMDNS DiscoveryMethod = iota
	DiscoveryNetworkScan
	DiscoveryManual
)

// ShellyClient handles communication with Shelly devices with automatic discovery
type ShellyClient struct {
	httpClient      *http.Client
	logger          *logrus.Logger
	defaultUsername string
	defaultPassword string

	// Discovery configuration
	discoveryEnabled   bool
	networkScanEnabled bool
	networkScanRanges  []string
	maxDevices         int
	retryAttempts      int
	retryBackoff       time.Duration
	enableGen1Support  bool
	enableGen2Support  bool
	autoWiFiSetup      bool

	// Auto-detection configuration
	autoDetectSubnets         bool
	autoDetectInterfaceFilter []string
	excludeLoopback           bool
	excludeDockerInterfaces   bool
	minSubnetSize             int

	// Device management
	discoveredDevices map[string]*EnhancedShellyDevice
	devicesMutex      sync.RWMutex
	discoveryChannel  chan *EnhancedShellyDevice
	stopDiscovery     chan bool
	discoveryRunning  bool
	lastDiscoveryTime time.Time
}

// EnhancedShellyDevice represents a discovered Shelly device with extended information
type EnhancedShellyDevice struct {
	// Basic device information
	ID              string           `json:"id"`
	MAC             string           `json:"mac"`
	Name            string           `json:"name"`
	Model           string           `json:"model"`
	Type            string           `json:"type"`
	Generation      DeviceGeneration `json:"generation"`
	FirmwareVersion string           `json:"firmware_version"`

	// Network information
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname"`

	// Discovery information
	DiscoveryMethod DiscoveryMethod `json:"discovery_method"`
	FirstSeen       time.Time       `json:"first_seen"`
	LastSeen        time.Time       `json:"last_seen"`
	LastHealthCheck time.Time       `json:"last_health_check"`
	IsOnline        bool            `json:"is_online"`
	IsConfigured    bool            `json:"is_configured"`

	// Device capabilities and status
	Capabilities     []string `json:"capabilities"`
	SupportedMethods []string `json:"supported_methods"`
	WiFiMode         string   `json:"wifi_mode"` // "ap" or "sta"
	WiFiSSID         string   `json:"wifi_ssid"`
	AuthEnabled      bool     `json:"auth_enabled"`

	// API endpoints based on generation
	Info     interface{} `json:"info"`
	Status   interface{} `json:"status"`
	Settings interface{} `json:"settings"`

	// Error tracking
	ErrorCount    int       `json:"error_count"`
	LastError     string    `json:"last_error"`
	LastErrorTime time.Time `json:"last_error_time"`
}

// DiscoveryConfig holds discovery configuration
type DiscoveryConfig struct {
	Enabled            bool          `json:"enabled"`
	Interval           time.Duration `json:"interval"`
	Timeout            time.Duration `json:"timeout"`
	NetworkScanEnabled bool          `json:"network_scan_enabled"`
	NetworkScanRanges  []string      `json:"network_scan_ranges"`
	MaxDevices         int           `json:"max_devices"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryBackoff       time.Duration `json:"retry_backoff"`
	EnableGen1Support  bool          `json:"enable_gen1_support"`
	EnableGen2Support  bool          `json:"enable_gen2_support"`
	AutoWiFiSetup      bool          `json:"auto_wifi_setup"`
	DefaultUsername    string        `json:"default_username"`
	DefaultPassword    string        `json:"default_password"`

	// Auto-detection configuration
	AutoDetectSubnets         bool     `json:"auto_detect_subnets"`
	AutoDetectInterfaceFilter []string `json:"auto_detect_interface_filter"`
	ExcludeLoopback           bool     `json:"exclude_loopback"`
	ExcludeDockerInterfaces   bool     `json:"exclude_docker_interfaces"`
	MinSubnetSize             int      `json:"min_subnet_size"`
}

// Gen2RPCRequest represents a JSON-RPC 2.0 request for Gen2+ devices
type Gen2RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Src     string      `json:"src"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Gen2RPCResponse represents a JSON-RPC 2.0 response for Gen2+ devices
type Gen2RPCResponse struct {
	ID     int         `json:"id"`
	Src    string      `json:"src"`
	Dst    string      `json:"dst"`
	Result interface{} `json:"result,omitempty"`
	Error  *RPCError   `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ShellySettings represents Shelly device settings
type ShellySettings struct {
	Device       ShellyDeviceSettings `json:"device"`
	WiFi         ShellyWiFiSettings   `json:"wifi_sta"`
	AP           ShellyAPSettings     `json:"wifi_ap"`
	MQTT         ShellyMQTTSettings   `json:"mqtt"`
	CoIoT        ShellyCoIoTSettings  `json:"coiot"`
	Sntp         ShellySntp           `json:"sntp"`
	LoginEnabled bool                 `json:"login_enabled"`
	PinCode      string               `json:"pin_code"`
	Name         string               `json:"name"`
	FwUpdate     ShellyFwUpdate       `json:"fw_update"`
	Discoverable bool                 `json:"discoverable"`
}

type ShellyDeviceSettings struct {
	Type       string `json:"type"`
	MAC        string `json:"mac"`
	Hostname   string `json:"hostname"`
	NumOutputs int    `json:"num_outputs"`
	NumMeters  int    `json:"num_meters"`
}

type ShellyWiFiSettings struct {
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid"`
	IP      string `json:"ip"`
	GW      string `json:"gw"`
	Mask    string `json:"mask"`
	DNS     string `json:"dns"`
}

type ShellyAPSettings struct {
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid"`
	Key     string `json:"key"`
}

type ShellyMQTTSettings struct {
	Enable              bool    `json:"enable"`
	Server              string  `json:"server"`
	User                string  `json:"user"`
	ID                  string  `json:"id"`
	ReconnectTimeout    float64 `json:"reconnect_timeout_max"`
	ReconnectTimeoutMin float64 `json:"reconnect_timeout_min"`
	CleanSession        bool    `json:"clean_session"`
	KeepAlive           int     `json:"keep_alive"`
	MaxQoS              int     `json:"max_qos"`
	Retain              bool    `json:"retain"`
	UpdatePeriod        int     `json:"update_period"`
}

type ShellyCoIoTSettings struct {
	Enabled      bool   `json:"enabled"`
	UpdatePeriod int    `json:"update_period"`
	Peer         string `json:"peer"`
}

type ShellySntp struct {
	Server string `json:"server"`
}

type ShellyFwUpdate struct {
	Strategy string `json:"strategy"`
	Server   string `json:"server"`
}

// ShellyStatus represents Shelly device status
type ShellyStatus struct {
	WiFi          ShellyWiFiStatus     `json:"wifi_sta"`
	Cloud         ShellyCloudStatus    `json:"cloud"`
	MQTT          ShellyMQTTStatus     `json:"mqtt"`
	Time          string               `json:"time"`
	Unixtime      int64                `json:"unixtime"`
	Serial        int                  `json:"serial"`
	HasUpdate     bool                 `json:"has_update"`
	MAC           string               `json:"mac"`
	CfgChangedCnt int                  `json:"cfg_changed_cnt"`
	ActionsStats  ShellyActionsStats   `json:"actions_stats"`
	Relays        []ShellyRelayStatus  `json:"relays,omitempty"`
	Meters        []ShellyMeterStatus  `json:"meters,omitempty"`
	Inputs        []ShellyInputStatus  `json:"inputs,omitempty"`
	Lights        []ShellyLightStatus  `json:"lights,omitempty"`
	Dimmers       []ShellyDimmerStatus `json:"dimmers,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	Humidity      *float64             `json:"humidity,omitempty"`
	Pressure      *float64             `json:"pressure,omitempty"`
	Voltage       *float64             `json:"voltage,omitempty"`
	Battery       *int                 `json:"battery,omitempty"`
}

type ShellyWiFiStatus struct {
	Connected bool   `json:"connected"`
	SSID      string `json:"ssid"`
	IP        string `json:"ip"`
	RSSI      int    `json:"rssi"`
}

type ShellyCloudStatus struct {
	Enabled   bool `json:"enabled"`
	Connected bool `json:"connected"`
}

type ShellyMQTTStatus struct {
	Connected bool `json:"connected"`
}

type ShellyActionsStats struct {
	Skipped int `json:"skipped"`
}

type ShellyRelayStatus struct {
	IsOn            bool   `json:"ison"`
	HasTimer        bool   `json:"has_timer"`
	TimerStarted    int64  `json:"timer_started"`
	TimerDuration   int    `json:"timer_duration"`
	TimerRemaining  int    `json:"timer_remaining"`
	Source          string `json:"source"`
	Overpower       bool   `json:"overpower"`
	OverTemperature bool   `json:"overtemperature"`
}

type ShellyMeterStatus struct {
	Power     float64   `json:"power"`
	Total     float64   `json:"total"`
	Counters  []float64 `json:"counters"`
	IsValid   bool      `json:"is_valid"`
	Timestamp int64     `json:"timestamp"`
}

type ShellyInputStatus struct {
	Input    int    `json:"input"`
	Event    string `json:"event"`
	EventCnt int    `json:"event_cnt"`
}

type ShellyLightStatus struct {
	IsOn           bool  `json:"ison"`
	Brightness     int   `json:"brightness"`
	Red            int   `json:"red"`
	Green          int   `json:"green"`
	Blue           int   `json:"blue"`
	White          int   `json:"white"`
	Gain           int   `json:"gain"`
	Temp           int   `json:"temp"`
	Effect         int   `json:"effect"`
	HasTimer       bool  `json:"has_timer"`
	TimerStarted   int64 `json:"timer_started"`
	TimerDuration  int   `json:"timer_duration"`
	TimerRemaining int   `json:"timer_remaining"`
}

type ShellyDimmerStatus struct {
	IsOn           bool  `json:"ison"`
	Brightness     int   `json:"brightness"`
	HasTimer       bool  `json:"has_timer"`
	TimerStarted   int64 `json:"timer_started"`
	TimerDuration  int   `json:"timer_duration"`
	TimerRemaining int   `json:"timer_remaining"`
}

// NewEnhancedShellyClient creates a new enhanced Shelly client with discovery capabilities
func NewEnhancedShellyClient(config DiscoveryConfig, logger *logrus.Logger) *ShellyClient {
	if config.DefaultUsername == "" {
		config.DefaultUsername = defaultUsername
	}

	return &ShellyClient{
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:                    logger,
		defaultUsername:           config.DefaultUsername,
		defaultPassword:           config.DefaultPassword,
		discoveryEnabled:          config.Enabled,
		networkScanEnabled:        config.NetworkScanEnabled,
		networkScanRanges:         config.NetworkScanRanges,
		maxDevices:                config.MaxDevices,
		retryAttempts:             config.RetryAttempts,
		retryBackoff:              config.RetryBackoff,
		enableGen1Support:         config.EnableGen1Support,
		enableGen2Support:         config.EnableGen2Support,
		autoWiFiSetup:             config.AutoWiFiSetup,
		autoDetectSubnets:         config.AutoDetectSubnets,
		autoDetectInterfaceFilter: config.AutoDetectInterfaceFilter,
		excludeLoopback:           config.ExcludeLoopback,
		excludeDockerInterfaces:   config.ExcludeDockerInterfaces,
		minSubnetSize:             config.MinSubnetSize,
		discoveredDevices:         make(map[string]*EnhancedShellyDevice),
		discoveryChannel:          make(chan *EnhancedShellyDevice, 100),
		stopDiscovery:             make(chan bool),
		discoveryRunning:          false,
	}
}

// StartDiscovery starts the automatic device discovery process
func (c *ShellyClient) StartDiscovery(ctx context.Context) error {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	if c.discoveryRunning {
		return fmt.Errorf("discovery already running")
	}

	c.discoveryRunning = true
	c.logger.Info("Starting enhanced Shelly device discovery")

	// Auto-detect local subnets if enabled
	if c.autoDetectSubnets {
		detectedSubnets, err := c.detectLocalSubnets()
		if err != nil {
			c.logger.WithError(err).Warn("Auto-detection failed, falling back to manual ranges")
		} else if len(detectedSubnets) > 0 {
			// Combine auto-detected with manual ranges
			originalRanges := c.networkScanRanges
			c.networkScanRanges = c.combineNetworkRanges(detectedSubnets, originalRanges)
			c.logger.Infof("Combined scan ranges: %v", c.networkScanRanges)
		}
	}

	// Start mDNS discovery goroutine
	go c.runMDNSDiscovery(ctx)

	// Start network scan discovery goroutine if enabled
	if c.networkScanEnabled {
		go c.runNetworkScanDiscovery(ctx)
	}

	// Start device health monitoring goroutine
	go c.runHealthMonitoring(ctx)

	return nil
}

// StopDiscovery stops the automatic device discovery process
func (c *ShellyClient) StopDiscovery() {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	if !c.discoveryRunning {
		return
	}

	c.logger.Info("Stopping device discovery")
	c.discoveryRunning = false

	// Signal all goroutines to stop
	close(c.stopDiscovery)

	// Close the discovery channel to prevent channel leaks
	if c.discoveryChannel != nil {
		close(c.discoveryChannel)
		c.discoveryChannel = nil
	}

	// Wait a bit for goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	// Recreate the stop channel for next discovery session
	c.stopDiscovery = make(chan bool)

	// Clear discovered devices to prevent memory accumulation
	c.discoveredDevices = make(map[string]*EnhancedShellyDevice)

	c.logger.Info("Device discovery stopped and cleaned up")
}

// GetDiscoveredDevices returns all currently discovered devices
func (c *ShellyClient) GetDiscoveredDevices() []*EnhancedShellyDevice {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	devices := make([]*EnhancedShellyDevice, 0, len(c.discoveredDevices))
	for _, device := range c.discoveredDevices {
		devices = append(devices, device)
	}

	return devices
}

// GetOnlineDevices returns only online devices
func (c *ShellyClient) GetOnlineDevices() []*EnhancedShellyDevice {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	devices := make([]*EnhancedShellyDevice, 0, len(c.discoveredDevices))
	for _, device := range c.discoveredDevices {
		if device.IsOnline {
			devices = append(devices, device)
		}
	}

	return devices
}

// runMDNSDiscovery runs continuous mDNS discovery
func (c *ShellyClient) runMDNSDiscovery(ctx context.Context) {
	ticker := time.NewTicker(discoveryTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopDiscovery:
			return
		case <-ticker.C:
			c.performMDNSDiscovery(ctx)
		}
	}
}

// performMDNSDiscovery performs a single mDNS discovery scan
func (c *ShellyClient) performMDNSDiscovery(ctx context.Context) {
	c.logger.Debug("Starting mDNS discovery scan")

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		c.logger.WithError(err).Error("Failed to create mDNS resolver")
		return
	}

	entries := make(chan *zeroconf.ServiceEntry, 10)
	discoveryCtx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	go func() {
		defer close(entries)
		err := resolver.Browse(discoveryCtx, shellyServiceType, shellyDomain, entries)
		if err != nil {
			c.logger.WithError(err).Error("mDNS discovery failed")
		}
	}()

	for entry := range entries {
		if strings.Contains(strings.ToLower(entry.Instance), "shelly") {
			device := c.processMDNSEntry(entry)
			if device != nil {
				c.addOrUpdateDevice(device)
			}
		}
	}

	c.lastDiscoveryTime = time.Now()
}

// processMDNSEntry processes an mDNS service entry into a device
func (c *ShellyClient) processMDNSEntry(entry *zeroconf.ServiceEntry) *EnhancedShellyDevice {
	if len(entry.AddrIPv4) == 0 {
		return nil
	}

	device := &EnhancedShellyDevice{
		IP:              entry.AddrIPv4[0].String(),
		Port:            entry.Port,
		Hostname:        entry.HostName,
		Name:            entry.Instance,
		DiscoveryMethod: DiscoveryMDNS,
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		IsOnline:        true,
	}

	// Parse TXT records for additional information
	for _, txt := range entry.Text {
		parts := strings.SplitN(txt, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			switch key {
			case "gen":
				if gen, err := strconv.Atoi(value); err == nil {
					device.Generation = DeviceGeneration(gen)
				}
			case "app":
				device.Model = value
			case "ver":
				device.FirmwareVersion = value
			case "mac":
				device.MAC = value
			case "id":
				device.ID = value
			}
		}
	}

	// Generate ID if not provided
	if device.ID == "" {
		device.ID = fmt.Sprintf("shelly_%s", device.MAC)
	}

	// Determine device type from instance name
	device.Type = extractShellyType(entry.Instance)

	return device
}

// runNetworkScanDiscovery runs network range scanning discovery
func (c *ShellyClient) runNetworkScanDiscovery(ctx context.Context) {
	ticker := time.NewTicker(scanInterval) // Scan less frequently than mDNS
	defer ticker.Stop()

	// Perform initial scan
	c.performNetworkScan(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopDiscovery:
			return
		case <-ticker.C:
			c.performNetworkScan(ctx)
		}
	}
}

// performNetworkScan performs network range scanning for Shelly devices
func (c *ShellyClient) performNetworkScan(ctx context.Context) {
	c.logger.Debug("Starting network scan discovery")

	for _, networkRange := range c.networkScanRanges {
		c.scanNetworkRange(ctx, networkRange)
	}
}

// scanNetworkRange scans a specific network range for Shelly devices
func (c *ShellyClient) scanNetworkRange(ctx context.Context, networkRange string) {
	_, ipNet, err := net.ParseCIDR(networkRange)
	if err != nil {
		c.logger.WithError(err).Errorf("Invalid network range: %s", networkRange)
		return
	}

	// Generate IP addresses to scan
	ips := generateIPsFromCIDR(ipNet)

	// Limit the number of IPs to scan to prevent goroutine explosion
	if len(ips) > maxDevicesPerScan {
		c.logger.WithField("total_ips", len(ips)).WithField("max_scan", maxDevicesPerScan).Warn("Limiting network scan to prevent memory leak")
		ips = ips[:maxDevicesPerScan]
	}

	// Create semaphore for concurrency control
	sem := make(chan struct{}, networkScanConcurrency)
	var wg sync.WaitGroup

	// Add timeout to prevent hanging scans
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	for _, ip := range ips {
		select {
		case <-scanCtx.Done():
			c.logger.Debug("Network scan cancelled due to timeout or context cancellation")
			return
		default:
		}

		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-scanCtx.Done():
				return
			}

			c.scanSingleIP(scanCtx, ip)
		}(ip)
	}

	// Wait for all scans to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Debug("Network scan completed successfully")
	case <-scanCtx.Done():
		c.logger.Warn("Network scan timed out, some IPs may not have been scanned")
	}
}

// scanSingleIP scans a single IP address for Shelly device
func (c *ShellyClient) scanSingleIP(ctx context.Context, ip string) {
	// Try to detect if it's a Shelly device
	device := c.detectShellyDevice(ctx, ip)
	if device != nil {
		device.DiscoveryMethod = DiscoveryNetworkScan
		c.addOrUpdateDevice(device)
	}
}

// detectShellyDevice attempts to detect if an IP hosts a Shelly device
func (c *ShellyClient) detectShellyDevice(ctx context.Context, ip string) *EnhancedShellyDevice {
	// Try Gen1 detection first (HTTP /shelly endpoint)
	if c.enableGen1Support {
		if device := c.detectGen1Device(ctx, ip); device != nil {
			return device
		}
	}

	// Try Gen2+ detection (HTTP /shelly endpoint or RPC)
	if c.enableGen2Support {
		if device := c.detectGen2Device(ctx, ip); device != nil {
			return device
		}
	}

	return nil
}

// detectGen1Device detects Gen1 Shelly devices
func (c *ShellyClient) detectGen1Device(ctx context.Context, ip string) *EnhancedShellyDevice {
	url := fmt.Sprintf("http://%s/shelly", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	client := &http.Client{Timeout: healthCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var info ShellyDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil
	}

	// Verify it's actually a Shelly device
	if !strings.Contains(strings.ToLower(info.Type), "shelly") {
		return nil
	}

	device := &EnhancedShellyDevice{
		ID:              fmt.Sprintf("shelly_%s", info.MAC),
		MAC:             info.MAC,
		Name:            info.Name,
		Model:           info.Model,
		Type:            info.Type,
		Generation:      DeviceGeneration(info.Gen),
		FirmwareVersion: info.Version,
		IP:              ip,
		Port:            80,
		Hostname:        info.Hostname,
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		IsOnline:        true,
		AuthEnabled:     info.AuthEn,
		Info:            &info,
	}

	// Get additional device information
	c.enrichDeviceInformation(ctx, device)

	return device
}

// detectGen2Device detects Gen2+ Shelly devices
func (c *ShellyClient) detectGen2Device(ctx context.Context, ip string) *EnhancedShellyDevice {
	// Try RPC method first
	device := c.detectGen2ViaRPC(ctx, ip)
	if device != nil {
		return device
	}

	// Fallback to HTTP /shelly endpoint
	return c.detectGen2ViaHTTP(ctx, ip)
}

// detectGen2ViaRPC detects Gen2+ devices via JSON-RPC
func (c *ShellyClient) detectGen2ViaRPC(ctx context.Context, ip string) *EnhancedShellyDevice {
	rpcReq := Gen2RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Src:     "pma_discovery",
		Method:  "Shelly.GetDeviceInfo",
	}

	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return nil
	}

	url := fmt.Sprintf("http://%s/rpc", ip)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: healthCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var rpcResp Gen2RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil
	}

	if rpcResp.Error != nil {
		return nil
	}

	// Parse device info from result
	resultData, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return nil
	}

	var deviceInfo map[string]interface{}
	if err := json.Unmarshal(resultData, &deviceInfo); err != nil {
		return nil
	}

	device := &EnhancedShellyDevice{
		IP:         ip,
		Port:       80,
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		IsOnline:   true,
		Generation: Gen2, // Assume Gen2+ if RPC works
	}

	// Extract device information
	if id, ok := deviceInfo["id"].(string); ok {
		device.ID = id
	}
	if mac, ok := deviceInfo["mac"].(string); ok {
		device.MAC = mac
	}
	if model, ok := deviceInfo["model"].(string); ok {
		device.Model = model
	}
	if gen, ok := deviceInfo["gen"].(float64); ok {
		device.Generation = DeviceGeneration(int(gen))
	}
	if ver, ok := deviceInfo["ver"].(string); ok {
		device.FirmwareVersion = ver
	}
	if authEn, ok := deviceInfo["auth_en"].(bool); ok {
		device.AuthEnabled = authEn
	}

	// Set default values if missing
	if device.ID == "" && device.MAC != "" {
		device.ID = fmt.Sprintf("shelly_%s", device.MAC)
	}
	if device.Name == "" {
		device.Name = device.Model
	}

	return device
}

// detectGen2ViaHTTP detects Gen2+ devices via HTTP /shelly endpoint
func (c *ShellyClient) detectGen2ViaHTTP(ctx context.Context, ip string) *EnhancedShellyDevice {
	// This is similar to Gen1 detection but handles Gen2+ response format
	url := fmt.Sprintf("http://%s/shelly", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	client := &http.Client{Timeout: healthCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var deviceInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&deviceInfo); err != nil {
		return nil
	}

	// Check if generation indicates Gen2+
	gen, hasGen := deviceInfo["gen"].(float64)
	if !hasGen || gen < 2 {
		return nil
	}

	device := &EnhancedShellyDevice{
		IP:         ip,
		Port:       80,
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		IsOnline:   true,
		Generation: DeviceGeneration(int(gen)),
	}

	// Extract device information
	if id, ok := deviceInfo["id"].(string); ok {
		device.ID = id
	}
	if mac, ok := deviceInfo["mac"].(string); ok {
		device.MAC = mac
	}
	if model, ok := deviceInfo["model"].(string); ok {
		device.Model = model
	}
	if ver, ok := deviceInfo["ver"].(string); ok {
		device.FirmwareVersion = ver
	}
	if authEn, ok := deviceInfo["auth_en"].(bool); ok {
		device.AuthEnabled = authEn
	}

	// Set default values if missing
	if device.ID == "" && device.MAC != "" {
		device.ID = fmt.Sprintf("shelly_%s", device.MAC)
	}
	if device.Name == "" {
		device.Name = device.Model
	}

	return device
}

// enrichDeviceInformation fetches additional device information
func (c *ShellyClient) enrichDeviceInformation(ctx context.Context, device *EnhancedShellyDevice) {
	// Get device status and settings based on generation
	switch device.Generation {
	case Gen1:
		c.enrichGen1Device(ctx, device)
	case Gen2, Gen3, Gen4:
		c.enrichGen2Device(ctx, device)
	}
}

// enrichGen1Device enriches Gen1 device information
func (c *ShellyClient) enrichGen1Device(ctx context.Context, device *EnhancedShellyDevice) {
	// Get device status
	if status, err := c.getGen1DeviceStatus(ctx, device.IP); err == nil {
		device.Status = status

		// Extract WiFi information
		if wifiStatus, ok := status["wifi_sta"].(map[string]interface{}); ok {
			if connected, ok := wifiStatus["connected"].(bool); ok && connected {
				device.WiFiMode = "sta"
				if ssid, ok := wifiStatus["ssid"].(string); ok {
					device.WiFiSSID = ssid
				}
			} else {
				device.WiFiMode = "ap"
			}
		}
	}

	// Get device settings
	if settings, err := c.getGen1DeviceSettings(ctx, device.IP); err == nil {
		device.Settings = settings
	}
}

// enrichGen2Device enriches Gen2+ device information
func (c *ShellyClient) enrichGen2Device(ctx context.Context, device *EnhancedShellyDevice) {
	// Get device status via RPC
	if status, err := c.getGen2DeviceStatus(ctx, device.IP); err == nil {
		device.Status = status

		// Extract WiFi information
		if wifiStatus, ok := status["wifi"].(map[string]interface{}); ok {
			if staIP, ok := wifiStatus["sta_ip"]; ok && staIP != nil {
				device.WiFiMode = "sta"
				if ssid, ok := wifiStatus["ssid"].(string); ok {
					device.WiFiSSID = ssid
				}
			} else {
				device.WiFiMode = "ap"
			}
		}
	}

	// Get device config via RPC
	if config, err := c.getGen2DeviceConfig(ctx, device.IP); err == nil {
		device.Settings = config
	}

	// Get supported methods
	if methods, err := c.getGen2SupportedMethods(ctx, device.IP); err == nil {
		device.SupportedMethods = methods
	}
}

// getGen1DeviceStatus gets Gen1 device status
func (c *ShellyClient) getGen1DeviceStatus(ctx context.Context, ip string) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/status", ip)
	return c.makeJSONRequest(ctx, "GET", url, nil)
}

// getGen1DeviceSettings gets Gen1 device settings
func (c *ShellyClient) getGen1DeviceSettings(ctx context.Context, ip string) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/settings", ip)
	return c.makeJSONRequest(ctx, "GET", url, nil)
}

// getGen2DeviceStatus gets Gen2+ device status via RPC
func (c *ShellyClient) getGen2DeviceStatus(ctx context.Context, ip string) (map[string]interface{}, error) {
	return c.makeGen2RPCCall(ctx, ip, "Shelly.GetStatus", nil)
}

// getGen2DeviceConfig gets Gen2+ device config via RPC
func (c *ShellyClient) getGen2DeviceConfig(ctx context.Context, ip string) (map[string]interface{}, error) {
	return c.makeGen2RPCCall(ctx, ip, "Shelly.GetConfig", nil)
}

// getGen2SupportedMethods gets Gen2+ supported methods via RPC
func (c *ShellyClient) getGen2SupportedMethods(ctx context.Context, ip string) ([]string, error) {
	result, err := c.makeGen2RPCCall(ctx, ip, "Shelly.ListMethods", nil)
	if err != nil {
		return nil, err
	}

	if methods, ok := result["methods"].([]interface{}); ok {
		strMethods := make([]string, len(methods))
		for i, method := range methods {
			if methodStr, ok := method.(string); ok {
				strMethods[i] = methodStr
			}
		}
		return strMethods, nil
	}

	return nil, fmt.Errorf("invalid methods response")
}

// makeGen2RPCCall makes a JSON-RPC call to a Gen2+ device
func (c *ShellyClient) makeGen2RPCCall(ctx context.Context, ip, method string, params interface{}) (map[string]interface{}, error) {
	rpcReq := Gen2RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Src:     "pma_client",
		Method:  method,
		Params:  params,
	}

	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s/rpc", ip)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if needed
	if c.defaultPassword != "" {
		req.SetBasicAuth(c.defaultUsername, c.defaultPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var rpcResp Gen2RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Convert result to map
	resultData, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// makeJSONRequest makes a JSON HTTP request and returns the parsed response
func (c *ShellyClient) makeJSONRequest(ctx context.Context, method, url string, body []byte) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Add authentication if needed
	if c.defaultPassword != "" {
		req.SetBasicAuth(c.defaultUsername, c.defaultPassword)
	}

	req.Header.Set("User-Agent", "PMA-Shelly-Enhanced/1.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// addOrUpdateDevice adds or updates a device in the discovered devices map
func (c *ShellyClient) addOrUpdateDevice(device *EnhancedShellyDevice) {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	// Check if we've reached the maximum number of devices
	if len(c.discoveredDevices) >= c.maxDevices {
		c.logger.WithField("max_devices", c.maxDevices).Warn("Maximum devices reached, removing oldest device")

		// Remove the oldest device to make room
		var oldestDevice string
		var oldestTime time.Time
		for id, dev := range c.discoveredDevices {
			if oldestDevice == "" || dev.FirstSeen.Before(oldestTime) {
				oldestDevice = id
				oldestTime = dev.FirstSeen
			}
		}
		if oldestDevice != "" {
			delete(c.discoveredDevices, oldestDevice)
		}
	}

	// Generate device ID if not present
	if device.ID == "" {
		device.ID = fmt.Sprintf("shelly_%s", device.MAC)
	}

	// Update existing device or add new one
	if existing, exists := c.discoveredDevices[device.ID]; exists {
		// Update existing device
		existing.LastSeen = time.Now()
		existing.IsOnline = device.IsOnline
		existing.IP = device.IP
		existing.Port = device.Port
		existing.Hostname = device.Hostname
		existing.Name = device.Name
		existing.Model = device.Model
		existing.Type = device.Type
		existing.Generation = device.Generation
		existing.FirmwareVersion = device.FirmwareVersion
		existing.Capabilities = device.Capabilities
		existing.SupportedMethods = device.SupportedMethods
		existing.WiFiMode = device.WiFiMode
		existing.WiFiSSID = device.WiFiSSID
		existing.AuthEnabled = device.AuthEnabled
		existing.Info = device.Info
		existing.Status = device.Status
		existing.Settings = device.Settings
	} else {
		// Add new device
		device.FirstSeen = time.Now()
		device.LastSeen = time.Now()
		c.discoveredDevices[device.ID] = device
	}

	// Send to discovery channel if available
	if c.discoveryChannel != nil {
		select {
		case c.discoveryChannel <- device:
		default:
			// Channel is full, skip sending
		}
	}
}

// runHealthMonitoring runs continuous health monitoring for discovered devices
func (c *ShellyClient) runHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopDiscovery:
			return
		case <-ticker.C:
			c.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs health checks on all discovered devices
func (c *ShellyClient) performHealthCheck(ctx context.Context) {
	c.devicesMutex.Lock()
	devices := make([]*EnhancedShellyDevice, 0, len(c.discoveredDevices))
	for _, device := range c.discoveredDevices {
		devices = append(devices, device)
	}
	c.devicesMutex.Unlock()

	// Limit the number of devices to check to prevent memory leaks
	if len(devices) > maxDevicesPerScan {
		c.logger.WithField("total_devices", len(devices)).WithField("max_check", maxDevicesPerScan).Warn("Limiting health check to prevent memory leak")
		devices = devices[:maxDevicesPerScan]
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentHealth) // Limit concurrent health checks

	// Add timeout to prevent hanging health checks
	healthCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	for _, device := range devices {
		select {
		case <-healthCtx.Done():
			c.logger.Debug("Health check cancelled due to timeout or context cancellation")
			return
		default:
		}

		wg.Add(1)
		go func(device *EnhancedShellyDevice) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-healthCtx.Done():
				return
			}

			c.checkDeviceHealth(healthCtx, device)
		}(device)
	}

	// Wait for all health checks to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Debug("Health check completed successfully")
	case <-healthCtx.Done():
		c.logger.Warn("Health check timed out, some devices may not have been checked")
	}
}

// checkDeviceHealth checks the health of a single device
func (c *ShellyClient) checkDeviceHealth(ctx context.Context, device *EnhancedShellyDevice) {
	isOnline := c.isDeviceReachable(ctx, device.IP)

	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	device.LastHealthCheck = time.Now()

	if isOnline {
		device.IsOnline = true
		device.LastSeen = time.Now()
		device.ErrorCount = 0
		device.LastError = ""
	} else {
		device.ErrorCount++
		if device.ErrorCount >= 3 {
			device.IsOnline = false
		}
		device.LastError = "Device unreachable"
		device.LastErrorTime = time.Now()
	}
}

// isDeviceReachable checks if a device is reachable
func (c *ShellyClient) isDeviceReachable(ctx context.Context, ip string) bool {
	url := fmt.Sprintf("http://%s/shelly", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: healthCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// generateIPsFromCIDR generates a list of IP addresses from a CIDR notation
func generateIPsFromCIDR(ipNet *net.IPNet) []string {
	var ips []string

	// Get the network and broadcast addresses
	networkAddr := ipNet.IP.Mask(ipNet.Mask)

	// Calculate the number of host bits
	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones

	// Limit scanning to reasonable subnet sizes
	if hostBits > 16 {
		return ips // Too many IPs to scan
	}

	// Generate IPs
	numIPs := 1 << hostBits
	for i := 1; i < numIPs-1; i++ { // Skip network and broadcast addresses
		ip := make(net.IP, len(networkAddr))
		copy(ip, networkAddr)

		// Add the host part
		for j := len(ip) - 1; j >= 0 && i > 0; j-- {
			ip[j] += byte(i & 0xff)
			i >>= 8
		}

		ips = append(ips, ip.String())
	}

	return ips
}

// extractShellyType extracts the Shelly device type from the mDNS instance name
func extractShellyType(instance string) string {
	parts := strings.Split(strings.ToLower(instance), "-")
	for _, part := range parts {
		if strings.HasPrefix(part, "shelly") {
			return part
		}
	}
	return "unknown"
}

// detectLocalSubnets discovers local network interfaces and returns their subnets
func (c *ShellyClient) detectLocalSubnets() ([]string, error) {
	if !c.autoDetectSubnets {
		return nil, nil
	}

	c.logger.Info("Auto-detecting local network subnets...")

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var detectedSubnets []string

	for _, iface := range interfaces {
		if c.isExcludedInterface(iface) {
			c.logger.Debugf("Excluding interface: %s (%s)", iface.Name, c.getExclusionReason(iface))
			continue
		}

		subnets, err := c.interfaceToSubnets(iface)
		if err != nil {
			c.logger.WithError(err).Warnf("Failed to extract subnets from interface %s", iface.Name)
			continue
		}

		for _, subnet := range subnets {
			// Validate subnet size
			_, ipNet, err := net.ParseCIDR(subnet)
			if err != nil {
				c.logger.WithError(err).Warnf("Invalid subnet detected: %s", subnet)
				continue
			}

			ones, _ := ipNet.Mask.Size()
			if c.minSubnetSize > 0 && ones > c.minSubnetSize {
				c.logger.Debugf("Skipping subnet %s (/%d smaller than minimum /%d)", subnet, ones, c.minSubnetSize)
				continue
			}

			detectedSubnets = append(detectedSubnets, subnet)
			c.logger.Infof("Found network interface: %s (%s)", iface.Name, subnet)
		}
	}

	if len(detectedSubnets) > 0 {
		c.logger.Infof("Auto-detected subnets: %v", detectedSubnets)
	} else {
		c.logger.Warn("No suitable subnets auto-detected")
	}

	return detectedSubnets, nil
}

// isExcludedInterface checks if an interface should be excluded from scanning
func (c *ShellyClient) isExcludedInterface(iface net.Interface) bool {
	// Check if interface is down
	if iface.Flags&net.FlagUp == 0 {
		return true
	}

	// Exclude loopback interfaces if configured
	if c.excludeLoopback && iface.Flags&net.FlagLoopback != 0 {
		return true
	}

	// Exclude docker interfaces if configured
	if c.excludeDockerInterfaces && strings.HasPrefix(iface.Name, "docker") {
		return true
	}

	// Exclude common virtual interfaces
	if c.excludeDockerInterfaces {
		virtualPrefixes := []string{"br-", "veth", "lo", "tun", "tap"}
		for _, prefix := range virtualPrefixes {
			if strings.HasPrefix(iface.Name, prefix) {
				return true
			}
		}
	}

	// Check interface filter if specified
	if len(c.autoDetectInterfaceFilter) > 0 {
		found := false
		for _, filterName := range c.autoDetectInterfaceFilter {
			if strings.Contains(iface.Name, filterName) {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	return false
}

// getExclusionReason returns a human-readable reason why an interface was excluded
func (c *ShellyClient) getExclusionReason(iface net.Interface) string {
	if iface.Flags&net.FlagUp == 0 {
		return "interface down"
	}
	if c.excludeLoopback && iface.Flags&net.FlagLoopback != 0 {
		return "loopback interface"
	}
	if c.excludeDockerInterfaces && strings.HasPrefix(iface.Name, "docker") {
		return "docker interface"
	}
	if c.excludeDockerInterfaces {
		virtualPrefixes := []string{"br-", "veth", "lo", "tun", "tap"}
		for _, prefix := range virtualPrefixes {
			if strings.HasPrefix(iface.Name, prefix) {
				return "virtual interface"
			}
		}
	}
	if len(c.autoDetectInterfaceFilter) > 0 {
		return "not in interface filter"
	}
	return "unknown"
}

// interfaceToSubnets extracts subnet information from a network interface
func (c *ShellyClient) interfaceToSubnets(iface net.Interface) ([]string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
	}

	var subnets []string
	for _, addr := range addrs {
		var ip net.IP
		var network *net.IPNet

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
			network = v
		case *net.IPAddr:
			ip = v.IP
			// For single IP addresses, we need to determine the network
			if ip.To4() != nil {
				// IPv4 - assume /24 for private networks
				if isPrivateIPv4(ip) {
					_, network, _ = net.ParseCIDR(fmt.Sprintf("%s/24", ip.String()))
				} else {
					continue // Skip public IPs without explicit subnet
				}
			} else {
				continue // Skip IPv6 for now
			}
		default:
			continue
		}

		// Only process IPv4 addresses
		if ip.To4() == nil {
			continue
		}

		// Only process private network ranges for security
		if !isPrivateIPv4(ip) {
			c.logger.Debugf("Skipping public IP address: %s", ip.String())
			continue
		}

		if network != nil {
			subnets = append(subnets, network.String())
		}
	}

	return subnets, nil
}

// isPrivateIPv4 checks if an IPv4 address is in a private range
func isPrivateIPv4(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}

	// Check private IPv4 ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// combineNetworkRanges combines auto-detected subnets with manually configured ranges
func (c *ShellyClient) combineNetworkRanges(detectedSubnets, manualRanges []string) []string {
	combined := make([]string, 0, len(detectedSubnets)+len(manualRanges))
	seenRanges := make(map[string]bool)

	// Add detected subnets first
	for _, subnet := range detectedSubnets {
		if !seenRanges[subnet] {
			combined = append(combined, subnet)
			seenRanges[subnet] = true
		}
	}

	// Add manual ranges, avoiding duplicates
	for _, subnet := range manualRanges {
		if !seenRanges[subnet] {
			combined = append(combined, subnet)
			seenRanges[subnet] = true
		}
	}

	return combined
}

// Legacy methods for backward compatibility

// DiscoverDevices legacy method for backward compatibility
func (c *ShellyClient) DiscoverDevices(ctx context.Context, timeout time.Duration) ([]DiscoveredDevice, error) {
	c.logger.Info("Using legacy DiscoverDevices method")

	devices := c.GetOnlineDevices()
	legacyDevices := make([]DiscoveredDevice, len(devices))

	for i, device := range devices {
		legacyDevices[i] = DiscoveredDevice{
			IP:       device.IP,
			Port:     device.Port,
			Hostname: device.Hostname,
			Name:     device.Name,
			Type:     device.Type,
		}
	}

	return legacyDevices, nil
}

// GetDeviceInfo legacy method - now enhanced with generation support
func (c *ShellyClient) GetDeviceInfo(ctx context.Context, deviceIP string) (*ShellyDeviceInfo, error) {
	// Try to find device in discovered devices first
	c.devicesMutex.RLock()
	for _, device := range c.discoveredDevices {
		if device.IP == deviceIP {
			c.devicesMutex.RUnlock()

			// Convert to legacy format
			info := &ShellyDeviceInfo{
				ID:       device.ID,
				Type:     device.Type,
				MAC:      device.MAC,
				Hostname: device.Hostname,
				Name:     device.Name,
				Model:    device.Model,
				Gen:      int(device.Generation),
				Version:  device.FirmwareVersion,
				AuthEn:   device.AuthEnabled,
			}

			return info, nil
		}
	}
	c.devicesMutex.RUnlock()

	// Fallback to direct detection
	device := c.detectShellyDevice(ctx, deviceIP)
	if device == nil {
		return nil, fmt.Errorf("device not found or not a Shelly device")
	}

	// Convert to legacy format
	info := &ShellyDeviceInfo{
		ID:       device.ID,
		Type:     device.Type,
		MAC:      device.MAC,
		Hostname: device.Hostname,
		Name:     device.Name,
		Model:    device.Model,
		Gen:      int(device.Generation),
		Version:  device.FirmwareVersion,
		AuthEn:   device.AuthEnabled,
	}

	return info, nil
}

// GetDeviceSettings gets device settings with generation support
func (c *ShellyClient) GetDeviceSettings(ctx context.Context, deviceIP string) (*ShellySettings, error) {
	// Find device to determine generation
	c.devicesMutex.RLock()
	var deviceGen DeviceGeneration = Gen1 // Default to Gen1 for backward compatibility
	for _, device := range c.discoveredDevices {
		if device.IP == deviceIP {
			deviceGen = device.Generation
			break
		}
	}
	c.devicesMutex.RUnlock()

	switch deviceGen {
	case Gen1:
		return c.getGen1Settings(ctx, deviceIP)
	case Gen2, Gen3, Gen4:
		return c.getGen2SettingsAsGen1(ctx, deviceIP)
	default:
		// Try both and see which works
		if settings, err := c.getGen1Settings(ctx, deviceIP); err == nil {
			return settings, nil
		}
		return c.getGen2SettingsAsGen1(ctx, deviceIP)
	}
}

// getGen1Settings gets Gen1 device settings
func (c *ShellyClient) getGen1Settings(ctx context.Context, deviceIP string) (*ShellySettings, error) {
	result, err := c.makeJSONRequest(ctx, "GET", fmt.Sprintf("http://%s/settings", deviceIP), nil)
	if err != nil {
		return nil, err
	}

	// Convert map to ShellySettings struct
	// This is a simplified conversion - in practice, you'd want to properly map all fields
	settings := &ShellySettings{}

	// Convert the map to JSON and back to struct for easy conversion
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// getGen2SettingsAsGen1 gets Gen2+ settings and converts to Gen1 format
func (c *ShellyClient) getGen2SettingsAsGen1(ctx context.Context, deviceIP string) (*ShellySettings, error) {
	result, err := c.makeGen2RPCCall(ctx, deviceIP, "Shelly.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	// Convert Gen2 config format to Gen1 settings format
	// This is a simplified conversion - you'd need to map specific fields
	settings := &ShellySettings{}

	// Basic conversion
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetDeviceStatus gets device status with generation support
func (c *ShellyClient) GetDeviceStatus(ctx context.Context, deviceIP string) (*ShellyStatus, error) {
	// Find device to determine generation
	c.devicesMutex.RLock()
	var deviceGen DeviceGeneration = Gen1 // Default to Gen1 for backward compatibility
	for _, device := range c.discoveredDevices {
		if device.IP == deviceIP {
			deviceGen = device.Generation
			break
		}
	}
	c.devicesMutex.RUnlock()

	switch deviceGen {
	case Gen1:
		return c.getGen1Status(ctx, deviceIP)
	case Gen2, Gen3, Gen4:
		return c.getGen2StatusAsGen1(ctx, deviceIP)
	default:
		// Try both and see which works
		if status, err := c.getGen1Status(ctx, deviceIP); err == nil {
			return status, nil
		}
		return c.getGen2StatusAsGen1(ctx, deviceIP)
	}
}

// getGen1Status gets Gen1 device status
func (c *ShellyClient) getGen1Status(ctx context.Context, deviceIP string) (*ShellyStatus, error) {
	result, err := c.makeJSONRequest(ctx, "GET", fmt.Sprintf("http://%s/status", deviceIP), nil)
	if err != nil {
		return nil, err
	}

	// Convert map to ShellyStatus struct
	status := &ShellyStatus{}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, status); err != nil {
		return nil, err
	}

	return status, nil
}

// getGen2StatusAsGen1 gets Gen2+ status and converts to Gen1 format
func (c *ShellyClient) getGen2StatusAsGen1(ctx context.Context, deviceIP string) (*ShellyStatus, error) {
	result, err := c.makeGen2RPCCall(ctx, deviceIP, "Shelly.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	// Convert Gen2 status format to Gen1 status format
	status := &ShellyStatus{}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, status); err != nil {
		return nil, err
	}

	return status, nil
}

// Control methods with generation support

// SetRelay controls a relay with generation support
func (c *ShellyClient) SetRelay(ctx context.Context, deviceIP string, relayIndex int, state bool, timer *int) error {
	// Determine device generation
	c.devicesMutex.RLock()
	var deviceGen DeviceGeneration = Gen1
	for _, device := range c.discoveredDevices {
		if device.IP == deviceIP {
			deviceGen = device.Generation
			break
		}
	}
	c.devicesMutex.RUnlock()

	switch deviceGen {
	case Gen1:
		return c.setGen1Relay(ctx, deviceIP, relayIndex, state, timer)
	case Gen2, Gen3, Gen4:
		return c.setGen2Relay(ctx, deviceIP, relayIndex, state, timer)
	default:
		// Try Gen1 first, then Gen2
		if err := c.setGen1Relay(ctx, deviceIP, relayIndex, state, timer); err == nil {
			return nil
		}
		return c.setGen2Relay(ctx, deviceIP, relayIndex, state, timer)
	}
}

// setGen1Relay sets Gen1 relay state
func (c *ShellyClient) setGen1Relay(ctx context.Context, deviceIP string, relayIndex int, state bool, timer *int) error {
	params := url.Values{}
	params.Set("turn", map[bool]string{true: "on", false: "off"}[state])

	if timer != nil {
		params.Set("timer", strconv.Itoa(*timer))
	}

	url := fmt.Sprintf("http://%s/relay/%d?%s", deviceIP, relayIndex, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if c.defaultPassword != "" {
		req.SetBasicAuth(c.defaultUsername, c.defaultPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// setGen2Relay sets Gen2+ relay state via RPC
func (c *ShellyClient) setGen2Relay(ctx context.Context, deviceIP string, relayIndex int, state bool, timer *int) error {
	params := map[string]interface{}{
		"id": relayIndex,
		"on": state,
	}

	if timer != nil {
		params["toggle_after"] = *timer
	}

	_, err := c.makeGen2RPCCall(ctx, deviceIP, "Switch.Set", params)
	return err
}

// SetLight controls a light with generation support
func (c *ShellyClient) SetLight(ctx context.Context, deviceIP string, lightIndex int, params map[string]interface{}) error {
	// Determine device generation
	c.devicesMutex.RLock()
	var deviceGen DeviceGeneration = Gen1
	for _, device := range c.discoveredDevices {
		if device.IP == deviceIP {
			deviceGen = device.Generation
			break
		}
	}
	c.devicesMutex.RUnlock()

	switch deviceGen {
	case Gen1:
		return c.setGen1Light(ctx, deviceIP, lightIndex, params)
	case Gen2, Gen3, Gen4:
		return c.setGen2Light(ctx, deviceIP, lightIndex, params)
	default:
		// Try Gen1 first, then Gen2
		if err := c.setGen1Light(ctx, deviceIP, lightIndex, params); err == nil {
			return nil
		}
		return c.setGen2Light(ctx, deviceIP, lightIndex, params)
	}
}

// setGen1Light sets Gen1 light state
func (c *ShellyClient) setGen1Light(ctx context.Context, deviceIP string, lightIndex int, params map[string]interface{}) error {
	urlParams := url.Values{}

	for key, value := range params {
		switch key {
		case "turn":
			if turn, ok := value.(bool); ok {
				urlParams.Set("turn", map[bool]string{true: "on", false: "off"}[turn])
			}
		case "brightness":
			if brightness, ok := value.(int); ok {
				urlParams.Set("brightness", strconv.Itoa(brightness))
			}
		case "red", "green", "blue", "white", "temp", "effect":
			if intVal, ok := value.(int); ok {
				urlParams.Set(key, strconv.Itoa(intVal))
			}
		}
	}

	requestURL := fmt.Sprintf("http://%s/light/%d?%s", deviceIP, lightIndex, urlParams.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return err
	}

	if c.defaultPassword != "" {
		req.SetBasicAuth(c.defaultUsername, c.defaultPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// setGen2Light sets Gen2+ light state via RPC
func (c *ShellyClient) setGen2Light(ctx context.Context, deviceIP string, lightIndex int, params map[string]interface{}) error {
	rpcParams := map[string]interface{}{
		"id": lightIndex,
	}

	// Convert params to Gen2 format
	for key, value := range params {
		switch key {
		case "turn":
			rpcParams["on"] = value
		case "brightness":
			rpcParams["brightness"] = value
		case "red":
			if rgb, ok := rpcParams["rgb"].([]int); ok {
				rgb[0] = value.(int)
			} else {
				rpcParams["rgb"] = []int{value.(int), 0, 0}
			}
		case "green":
			if rgb, ok := rpcParams["rgb"].([]int); ok {
				rgb[1] = value.(int)
			} else {
				rpcParams["rgb"] = []int{0, value.(int), 0}
			}
		case "blue":
			if rgb, ok := rpcParams["rgb"].([]int); ok {
				rgb[2] = value.(int)
			} else {
				rpcParams["rgb"] = []int{0, 0, value.(int)}
			}
		default:
			rpcParams[key] = value
		}
	}

	_, err := c.makeGen2RPCCall(ctx, deviceIP, "Light.Set", rpcParams)
	return err
}

// SetDimmer controls a dimmer with generation support
func (c *ShellyClient) SetDimmer(ctx context.Context, deviceIP string, dimmerIndex int, brightness int, turn *bool) error {
	// For Gen2+, dimmers are typically controlled via Light.Set
	params := map[string]interface{}{
		"brightness": brightness,
	}

	if turn != nil {
		params["turn"] = *turn
	}

	return c.SetLight(ctx, deviceIP, dimmerIndex, params)
}

// IsDeviceReachable checks if a Shelly device is reachable
func (c *ShellyClient) IsDeviceReachable(ctx context.Context, deviceIP string) bool {
	return c.isDeviceReachable(ctx, deviceIP)
}

// Additional utility methods

// GetDeviceByIP returns a discovered device by IP address
func (c *ShellyClient) GetDeviceByIP(ip string) *EnhancedShellyDevice {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	for _, device := range c.discoveredDevices {
		if device.IP == ip {
			return device
		}
	}

	return nil
}

// GetDevicesByGeneration returns devices filtered by generation
func (c *ShellyClient) GetDevicesByGeneration(gen DeviceGeneration) []*EnhancedShellyDevice {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	var devices []*EnhancedShellyDevice
	for _, device := range c.discoveredDevices {
		if device.Generation == gen {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetDevicesByModel returns devices filtered by model
func (c *ShellyClient) GetDevicesByModel(model string) []*EnhancedShellyDevice {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	var devices []*EnhancedShellyDevice
	for _, device := range c.discoveredDevices {
		if strings.Contains(strings.ToLower(device.Model), strings.ToLower(model)) {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetDiscoveryStats returns discovery statistics
func (c *ShellyClient) GetDiscoveryStats() map[string]interface{} {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	stats := map[string]interface{}{
		"total_devices":     len(c.discoveredDevices),
		"online_devices":    0,
		"gen1_devices":      0,
		"gen2_devices":      0,
		"discovery_running": c.discoveryRunning,
		"last_discovery":    c.lastDiscoveryTime,
	}

	for _, device := range c.discoveredDevices {
		if device.IsOnline {
			stats["online_devices"] = stats["online_devices"].(int) + 1
		}

		switch device.Generation {
		case Gen1:
			stats["gen1_devices"] = stats["gen1_devices"].(int) + 1
		case Gen2, Gen3, Gen4:
			stats["gen2_devices"] = stats["gen2_devices"].(int) + 1
		}
	}

	return stats
}

// RemoveDevice removes a device from the discovered devices list
func (c *ShellyClient) RemoveDevice(deviceID string) bool {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	if _, exists := c.discoveredDevices[deviceID]; exists {
		delete(c.discoveredDevices, deviceID)
		c.logger.Infof("Removed device: %s", deviceID)
		return true
	}

	return false
}

// ClearDevices clears all discovered devices
func (c *ShellyClient) ClearDevices() {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	c.discoveredDevices = make(map[string]*EnhancedShellyDevice)
	c.logger.Info("Cleared all discovered devices")
}

// GetDiscoveryChannel returns the discovery channel for real-time device updates
func (c *ShellyClient) GetDiscoveryChannel() <-chan *EnhancedShellyDevice {
	return c.discoveryChannel
}

// Legacy compatibility - keep existing type definitions but mark as deprecated

// DiscoveredDevice represents a discovered Shelly device (legacy)
type DiscoveredDevice struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

// ShellyDeviceInfo represents basic Shelly device information (legacy but enhanced)
type ShellyDeviceInfo struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	MAC        string `json:"mac"`
	Hostname   string `json:"hostname"`
	Name       string `json:"name"`
	Model      string `json:"model"`
	Gen        int    `json:"gen"`
	FwID       string `json:"fw_id"`
	Version    string `json:"ver"`
	App        string `json:"app"`
	AuthEn     bool   `json:"auth_en"`
	AuthDomain string `json:"auth_domain"`
}

// Keep existing ShellySettings, ShellyStatus, and related types for backward compatibility
