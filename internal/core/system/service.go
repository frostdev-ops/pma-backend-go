package system

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"

	"github.com/sirupsen/logrus"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/pkg/version"
)

// ErrorRecord represents a tracked system error
type ErrorRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Details   string    `json:"details,omitempty"`
}

// Service provides system management functionality
type Service struct {
	deviceID      string
	startTime     time.Time
	logger        *logrus.Logger
	logBuffer     []LogEntry
	maxLogEntries int
	mu            sync.RWMutex

	// Service tracking
	trackedServices []ServiceConfig

	// Error tracking
	errorCount      int64
	lastError       time.Time
	errorHistory    []ErrorRecord
	maxErrorHistory int

	// Configuration
	config     SystemConfig
	fullConfig *config.Config
}

// NewService creates a new system service
func NewService(cfg *config.Config, logger *logrus.Logger) *Service {
	maxLogEntries := cfg.System.MaxLogEntries
	if maxLogEntries <= 0 {
		maxLogEntries = 1000 // fallback default
	}

	// Build tracked services from configuration
	var trackedServices []ServiceConfig
	for name, serviceConfig := range cfg.System.Services {
		if serviceConfig.Enabled {
			tracked := ServiceConfig{
				Name:        name,
				Type:        serviceConfig.Type,
				DisplayName: serviceConfig.DisplayName,
			}

			// Set service-specific fields based on type
			switch serviceConfig.Type {
			case "network_service":
				tracked.Host = serviceConfig.Host
				tracked.Port = serviceConfig.Port
			case "file_service":
				tracked.Path = serviceConfig.Path
			case "systemd_service":
				tracked.ServiceName = serviceConfig.ServiceName
			}

			trackedServices = append(trackedServices, tracked)
		}
	}

	// Fallback to default services if none configured
	if len(trackedServices) == 0 {
		trackedServices = []ServiceConfig{
			{Name: "homeAssistant", Type: "network_service", Host: "192.168.100.2", Port: 8123, DisplayName: "Home Assistant"},
			{Name: "database", Type: "file_service", Path: "./data/pma.db", DisplayName: "SQLite Database"},
			{Name: "webSocket", Type: "internal_service", DisplayName: "WebSocket Service"},
			{Name: "pma-backend", Type: "systemd_service", ServiceName: "pma-backend", DisplayName: "PMA Backend"},
			{Name: "nginx", Type: "systemd_service", ServiceName: "nginx", DisplayName: "Nginx Web Server"},
			{Name: "pma-router", Type: "systemd_service", ServiceName: "pma-router", DisplayName: "PMA Router"},
		}
	}

	deviceID := generateDeviceID()

	s := &Service{
		deviceID:        deviceID,
		startTime:       time.Now(),
		logger:          logger,
		maxLogEntries:   maxLogEntries,
		logBuffer:       make([]LogEntry, 0, maxLogEntries),
		trackedServices: trackedServices,
		maxErrorHistory: 100,
		errorHistory:    make([]ErrorRecord, 0, 100),
		config:          SystemConfig{}, // Will be populated with defaults on first access
		fullConfig:      cfg,
	}

	s.logger.Info("System management service initialized", "device_id", s.deviceID)
	return s
}

// GetDeviceInfo returns device information
func (s *Service) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		s.logger.Warn("Failed to get CPU info", "error", err)
	}

	var cpuModel string
	var cpuCount int
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
		cpuCount = len(cpuInfo)
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		s.logger.Warn("Failed to get memory info", "error", err)
	}

	metadata := map[string]interface{}{
		"go_version":     runtime.Version(),
		"total_memory":   int64(0),
		"cpu_count":      cpuCount,
		"cpu_model":      cpuModel,
		"kernel_version": hostInfo.KernelVersion,
		"virtualization": hostInfo.VirtualizationSystem,
	}

	if memInfo != nil {
		metadata["total_memory"] = memInfo.Total
	}

	return &DeviceInfo{
		DeviceID:    s.deviceID,
		Name:        hostInfo.Hostname,
		Version:     version.GetVersion(), // Use GetVersion() instead
		OS:          hostInfo.OS,
		Arch:        runtime.GOARCH,
		Platform:    hostInfo.Platform,
		Hostname:    hostInfo.Hostname,
		LastSeen:    time.Now(),
		Uptime:      int64(hostInfo.Uptime),
		Environment: getEnvironment(), // Use the function instead of config field
		Metadata:    metadata,
	}, nil
}

// GetSystemHealth returns comprehensive system health information
func (s *Service) GetSystemHealth(ctx context.Context) (*SystemHealth, error) {
	// Gather system information in parallel
	type result struct {
		cpu     *CPUInfo
		memory  *MemoryInfo
		disk    *DiskInfo
		network *NetworkInfo
		err     error
	}

	resultChan := make(chan result, 1)

	go func() {
		var r result

		// Get CPU information
		if cpuInfo, err := s.getCPUInfo(ctx); err != nil {
			r.err = fmt.Errorf("failed to get CPU info: %w", err)
		} else {
			r.cpu = cpuInfo
		}

		// Get memory information
		if memInfo, err := s.getMemoryInfo(ctx); err != nil {
			r.err = fmt.Errorf("failed to get memory info: %w", err)
		} else {
			r.memory = memInfo
		}

		// Get disk information
		if diskInfo, err := s.getDiskInfo(ctx); err != nil {
			r.err = fmt.Errorf("failed to get disk info: %w", err)
		} else {
			r.disk = diskInfo
		}

		// Get network information
		if netInfo, err := s.getNetworkInfo(ctx); err != nil {
			r.err = fmt.Errorf("failed to get network info: %w", err)
		} else {
			r.network = netInfo
		}

		resultChan <- r
	}()

	select {
	case r := <-resultChan:
		if r.err != nil {
			return nil, r.err
		}

		// Get service health
		servicesHealth, err := s.getServicesHealth(ctx)
		if err != nil {
			s.logger.Warn("Failed to get services health", "error", err)
			// Continue with default values
			servicesHealth = &ServicesHealth{}
		}

		return &SystemHealth{
			DeviceID:  s.deviceID,
			Timestamp: time.Now(),
			CPU:       *r.cpu,
			Memory:    *r.memory,
			Disk:      *r.disk,
			Network:   *r.network,
			Services:  *servicesHealth,
		}, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// getCPUInfo returns CPU information and usage
func (s *Service) getCPUInfo(ctx context.Context) (*CPUInfo, error) {
	// Get CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	var usage float64
	if len(cpuPercent) > 0 {
		usage = cpuPercent[0]
	}

	// Get load average
	loadAvg, err := load.Avg()
	if err != nil {
		s.logger.Warn("Failed to get load average", "error", err)
		loadAvg = &load.AvgStat{}
	}

	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	var model string
	var cores int
	if len(cpuInfo) > 0 {
		model = cpuInfo[0].ModelName
		cores = len(cpuInfo)
	}

	// Try to get CPU temperature (Linux only)
	var temperature *float64
	if temp := s.getCPUTemperature(); temp != nil {
		temperature = temp
	}

	return &CPUInfo{
		Usage:       usage,
		LoadAverage: []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15},
		Temperature: temperature,
		Cores:       cores,
		Model:       model,
	}, nil
}

// getCPUTemperature attempts to get CPU temperature (Linux only)
func (s *Service) getCPUTemperature() *float64 {
	// Try different temperature sources
	tempSources := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}

	for _, source := range tempSources {
		if data, err := os.ReadFile(source); err == nil {
			if temp, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
				// Convert from millidegrees to degrees if necessary
				if temp > 1000 {
					temp = temp / 1000.0
				}
				return &temp
			}
		}
	}

	return nil
}

// getMemoryInfo returns memory information
func (s *Service) getMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	return &MemoryInfo{
		Total:       memInfo.Total,
		Available:   memInfo.Available,
		Used:        memInfo.Used,
		UsedPercent: memInfo.UsedPercent,
		Free:        memInfo.Free,
		Cached:      memInfo.Cached,
		Buffers:     memInfo.Buffers,
	}, nil
}

// getDiskInfo returns disk information for the root filesystem
func (s *Service) getDiskInfo(ctx context.Context) (*DiskInfo, error) {
	diskUsage, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	return &DiskInfo{
		Total:       diskUsage.Total,
		Free:        diskUsage.Free,
		Used:        diskUsage.Used,
		UsedPercent: diskUsage.UsedPercent,
		Path:        diskUsage.Path,
		Filesystem:  diskUsage.Fstype,
	}, nil
}

// getNetworkInfo returns network information
func (s *Service) getNetworkInfo(ctx context.Context) (*NetworkInfo, error) {
	interfaces, err := psnet.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var netInterfaces []NetworkInterface
	for _, iface := range interfaces {
		// Get interface statistics
		stats, err := psnet.IOCounters(true)
		var netStats *NetworkStats
		for _, stat := range stats {
			if stat.Name == iface.Name && err == nil {
				netStats = &NetworkStats{
					BytesSent:   stat.BytesSent,
					BytesRecv:   stat.BytesRecv,
					PacketsSent: stat.PacketsSent,
					PacketsRecv: stat.PacketsRecv,
					Errors:      stat.Errin + stat.Errout,
					Drops:       stat.Dropin + stat.Dropout,
				}
				break
			}
		}

		// Convert addresses to string slice
		var addresses []string
		for _, addr := range iface.Addrs {
			addresses = append(addresses, addr.Addr)
		}

		netInterface := NetworkInterface{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr,
			Addresses:    addresses,
			IsUp:         false, // Will be determined below
			IsLoopback:   false, // Will be determined below
			MTU:          int(iface.MTU),
			Stats:        netStats,
		}

		// Check flags
		for _, flag := range iface.Flags {
			if strings.Contains(flag, "up") {
				netInterface.IsUp = true
			}
			if strings.Contains(flag, "loopback") {
				netInterface.IsLoopback = true
			}
		}

		// Determine interface type
		if netInterface.IsLoopback {
			netInterface.Type = "loopback"
		} else if strings.HasPrefix(iface.Name, "eth") {
			netInterface.Type = "ethernet"
		} else if strings.HasPrefix(iface.Name, "wlan") || strings.HasPrefix(iface.Name, "wifi") {
			netInterface.Type = "wireless"
		} else {
			netInterface.Type = "unknown"
		}

		netInterfaces = append(netInterfaces, netInterface)
	}

	// Get connectivity information
	connectivity := s.getNetworkConnectivity(ctx)

	return &NetworkInfo{
		Interfaces:   netInterfaces,
		Connectivity: connectivity,
	}, nil
}

// getNetworkConnectivity checks network connectivity
func (s *Service) getNetworkConnectivity(ctx context.Context) NetworkConnectivity {
	connectivity := NetworkConnectivity{
		HasInternet: false,
		DNS:         []string{"8.8.8.8", "1.1.1.1"},
	}

	// Test internet connectivity
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Use configured external service URLs
	primaryURL := s.fullConfig.ExternalServices.IPCheckServices.Primary
	fallbackURL := s.fullConfig.ExternalServices.IPCheckServices.Fallback

	if resp, err := client.Get(primaryURL); err == nil {
		resp.Body.Close()
		connectivity.HasInternet = true

		// Try to get external IP using fallback service
		if resp, err := client.Get(fallbackURL); err == nil {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				connectivity.ExternalIP = strings.TrimSpace(string(body))
			}
		}
	}

	// Get default gateway
	if gateway := s.getDefaultGateway(); gateway != "" {
		connectivity.Gateway = gateway
	}

	return connectivity
}

// getDefaultGateway returns the default gateway IP
func (s *Service) getDefaultGateway() string {
	s.logger.Debug("Getting default gateway using 'ip route show default'")

	// Use timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ip", "route", "show", "default")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "default") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					s.logger.Debug("Default gateway found:", fields[2])
					return fields[2]
				}
			}
		}
	}
	s.logger.Debug("No default gateway found")
	return ""
}

// getServicesHealth returns health information for tracked services
func (s *Service) getServicesHealth(ctx context.Context) (*ServicesHealth, error) {
	healthChan := make(chan map[string]ServiceHealth, 1)

	go func() {
		health := make(map[string]ServiceHealth)
		var wg sync.WaitGroup
		mu := sync.Mutex{}

		for _, service := range s.trackedServices {
			if service.Name == "homeAssistant" || service.Name == "database" || service.Name == "webSocket" {
				wg.Add(1)
				go func(svc ServiceConfig) {
					defer wg.Done()
					serviceHealth := s.checkServiceHealth(ctx, svc)
					mu.Lock()
					health[svc.Name] = serviceHealth
					mu.Unlock()
				}(service)
			}
		}

		wg.Wait()
		healthChan <- health
	}()

	select {
	case health := <-healthChan:
		return &ServicesHealth{
			HomeAssistant: health["homeAssistant"],
			Database:      health["database"],
			WebSocket:     health["webSocket"],
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// checkServiceHealth checks the health of a specific service
func (s *Service) checkServiceHealth(ctx context.Context, service ServiceConfig) ServiceHealth {
	startTime := time.Now()
	details := make(map[string]interface{})

	var status string = "unhealthy"
	var uptime int64 = 0

	switch service.Type {
	case "network_service":
		if s.checkNetworkService(ctx, service, details) {
			status = "healthy"
		}
	case "file_service":
		if s.checkFileService(ctx, service, details) {
			status = "healthy"
		}
	case "systemd_service":
		if healthy, svcUptime := s.checkSystemdService(ctx, service, details); healthy {
			status = "healthy"
			uptime = svcUptime
		}
	case "internal_service":
		if s.checkInternalService(ctx, service, details) {
			status = "healthy"
		}
	}

	responseTime := time.Since(startTime).Milliseconds()

	return ServiceHealth{
		Status:       status,
		Uptime:       uptime,
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		ErrorCount:   int(s.GetErrorCount()),
		Details:      details,
	}
}

// checkNetworkService checks if a network service is healthy
func (s *Service) checkNetworkService(ctx context.Context, service ServiceConfig, details map[string]interface{}) bool {
	address := fmt.Sprintf("%s:%d", service.Host, service.Port)

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		details["error"] = err.Error()
		details["address"] = address
		return false
	}
	defer conn.Close()

	details["address"] = address
	details["connected"] = true
	return true
}

// checkFileService checks if a file service is healthy
func (s *Service) checkFileService(ctx context.Context, service ServiceConfig, details map[string]interface{}) bool {
	info, err := os.Stat(service.Path)
	if err != nil {
		details["error"] = err.Error()
		details["path"] = service.Path
		return false
	}

	details["path"] = service.Path
	details["size"] = info.Size()
	details["modified"] = info.ModTime()
	details["accessible"] = true
	return true
}

// checkSystemdService checks if a systemd service is healthy
func (s *Service) checkSystemdService(ctx context.Context, service ServiceConfig, details map[string]interface{}) (bool, int64) {
	s.logger.Debug("Checking systemd service:", service.ServiceName)
	if service.ServiceName == "" {
		details["error"] = "No service name specified"
		return false, 0
	}

	cmd := exec.CommandContext(ctx, "systemctl", "show", service.ServiceName,
		"--property=ActiveState,LoadState,SubState,MainPID")

	output, err := cmd.Output()
	if err != nil {
		s.logger.Debug("Failed to check systemd service:", service.ServiceName, "error:", err)
		return false, 0
	}

	s.logger.Debug("Systemd service check output for", service.ServiceName, ":", string(output))

	properties := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			properties[parts[0]] = parts[1]
		}
	}

	isActive := properties["ActiveState"] == "active"
	isLoaded := properties["LoadState"] == "loaded"
	subState := properties["SubState"]
	mainPID := properties["MainPID"]

	details["active_state"] = properties["ActiveState"]
	details["sub_state"] = subState
	details["load_state"] = properties["LoadState"]
	details["unit_file_state"] = properties["UnitFileState"]
	details["main_pid"] = mainPID
	details["loaded"] = isLoaded

	// Calculate uptime
	var uptime int64 = 0
	if startTime := properties["ExecMainStartTimestamp"]; startTime != "" && startTime != "0" {
		if parsed, err := time.Parse("Mon 2006-01-02 15:04:05 MST", startTime); err == nil {
			uptime = time.Since(parsed).Milliseconds()
		}
	}

	return isActive && isLoaded, uptime
}

// checkInternalService checks if an internal service is healthy
func (s *Service) checkInternalService(ctx context.Context, service ServiceConfig, details map[string]interface{}) bool {
	// For internal services, we assume they're healthy if we're running
	// In a real implementation, you'd check specific service instances
	details["type"] = "internal"
	details["status"] = "running"
	return true
}

// AddLog adds a log entry to the buffer
func (s *Service) AddLog(level, service, message string, data map[string]interface{}, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Service:   service,
		Message:   message,
		Data:      data,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Add to buffer
	s.logBuffer = append(s.logBuffer, entry)

	// Trim buffer if too large
	if len(s.logBuffer) > s.maxLogEntries {
		s.logBuffer = s.logBuffer[len(s.logBuffer)-s.maxLogEntries:]
	}

	// Log to logger as well
	switch level {
	case "debug":
		s.logger.Debug(message, "service", service, "data", data, "error", err)
	case "info":
		s.logger.Info(message, "service", service, "data", data, "error", err)
	case "warn":
		s.logger.Warn(message, "service", service, "data", data, "error", err)
	case "error":
		s.logger.Error(message, "service", service, "data", data, "error", err)
	}
}

// GetSystemLogs returns system logs based on the request
func (s *Service) GetSystemLogs(ctx context.Context, req LogsRequest) (*SystemLogs, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter logs based on request
	var filteredLogs []LogEntry
	for _, log := range s.logBuffer {
		// Filter by level
		if req.Level != "" && log.Level != req.Level {
			continue
		}

		// Filter by service
		if req.Service != "" && log.Service != req.Service {
			continue
		}

		// Filter by time range
		if !req.StartTime.IsZero() && log.Timestamp.Before(req.StartTime) {
			continue
		}
		if !req.EndTime.IsZero() && log.Timestamp.After(req.EndTime) {
			continue
		}

		filteredLogs = append(filteredLogs, log)
	}

	// Apply limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	hasMore := len(filteredLogs) > limit
	if hasMore {
		filteredLogs = filteredLogs[:limit]
	}

	// Calculate time range
	var timeRange TimeRange
	if len(filteredLogs) > 0 {
		timeRange.Start = filteredLogs[0].Timestamp
		timeRange.End = filteredLogs[len(filteredLogs)-1].Timestamp
	}

	return &SystemLogs{
		Logs:       filteredLogs,
		TotalCount: len(s.logBuffer),
		HasMore:    hasMore,
		TimeRange:  timeRange,
	}, nil
}

// RebootSystem initiates a system reboot
func (s *Service) RebootSystem(ctx context.Context, action PowerAction) error {
	s.logger.Info("System reboot requested", "reason", action.Reason, "requested_by", action.RequestBy)

	s.AddLog("info", "system", "System reboot initiated", map[string]interface{}{
		"reason":       action.Reason,
		"requested_by": action.RequestBy,
		"delay":        action.Delay,
	}, nil)

	// Add delay if specified
	if action.Delay > 0 {
		time.Sleep(time.Duration(action.Delay) * time.Second)
	}

	var cmd *exec.Cmd
	if action.Force {
		cmd = exec.CommandContext(ctx, "sudo", "reboot", "-f")
	} else {
		cmd = exec.CommandContext(ctx, "sudo", "reboot")
	}

	return cmd.Run()
}

// ShutdownSystem initiates a system shutdown
func (s *Service) ShutdownSystem(ctx context.Context, action PowerAction) error {
	s.logger.Info("System shutdown requested", "reason", action.Reason, "requested_by", action.RequestBy)

	s.AddLog("info", "system", "System shutdown initiated", map[string]interface{}{
		"reason":       action.Reason,
		"requested_by": action.RequestBy,
		"delay":        action.Delay,
	}, nil)

	// Add delay if specified
	if action.Delay > 0 {
		time.Sleep(time.Duration(action.Delay) * time.Second)
	}

	var cmd *exec.Cmd
	if action.Force {
		cmd = exec.CommandContext(ctx, "sudo", "shutdown", "-h", "-f", "now")
	} else {
		cmd = exec.CommandContext(ctx, "sudo", "shutdown", "-h", "now")
	}

	return cmd.Run()
}

// GetConfig returns the system configuration
func (s *Service) GetConfig() SystemConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If config is empty, return defaults
	if s.config.System.Name == "" {
		return s.getDefaultConfig()
	}

	return s.config
}

// UpdateConfig updates the system configuration
func (s *Service) UpdateConfig(config SystemConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate required fields
	if config.System.Name == "" {
		config.System.Name = "PMA System"
	}
	if config.System.Timezone == "" {
		config.System.Timezone = "UTC"
	}
	if config.System.Locale == "" {
		config.System.Locale = "en"
	}

	// Set metadata
	config.UpdatedAt = time.Now().Format(time.RFC3339)
	if config.CreatedAt == "" {
		config.CreatedAt = time.Now().Format(time.RFC3339)
	}
	config.Version++

	s.config = config
	s.logger.Info("System configuration updated")

	return nil
}

// getDefaultConfig returns a default configuration that matches frontend expectations
func (s *Service) getDefaultConfig() SystemConfig {
	now := time.Now().Format(time.RFC3339)

	return SystemConfig{
		System: SystemSectionConfig{
			Name:                "PMA System",
			Description:         "Personal Management Assistant Home Control System",
			Timezone:            "UTC",
			Locale:              "en",
			LogLevel:            "info",
			DebugMode:           false,
			MaintenanceMode:     false,
			AutoBackup:          true,
			BackupRetentionDays: 30,
			HubUser: HubUser{
				Name:    "Hub",
				Email:   "hub@pma.local",
				Enabled: true,
			},
		},
		Server: ServerSectionConfig{
			Host:           "0.0.0.0",
			Port:           3001,
			HTTPSEnabled:   false,
			CORSEnabled:    true,
			CORSOrigins:    []string{"*"},
			RateLimiting:   false,
			MaxRequestSize: 10485760, // 10MB
			TimeoutSeconds: 30,
		},
		Database: DatabaseSectionConfig{
			Type:               "sqlite",
			Name:               "./data/pma.db",
			ConnectionPoolSize: 10,
			MaxIdleConnections: 5,
			ConnectionTimeout:  30,
		},
		Auth: AuthSectionConfig{
			Enabled:             true,
			Method:              "pin",
			PinLength:           4,
			SessionTimeout:      1800, // 30 minutes
			JWTSecret:           "pma-jwt-secret-change-in-production",
			RefreshTokenEnabled: true,
			MaxSessionsPerUser:  5,
			LockoutThreshold:    5,
			LockoutDuration:     300, // 5 minutes
		},
		WebSocket: WebSocketSectionConfig{
			Enabled:            true,
			MaxConnections:     100,
			HeartbeatInterval:  30,
			MessageBufferSize:  1000,
			CompressionEnabled: true,
		},
		Services: ServicesSectionConfig{
			HomeAssistant: &HomeAssistantConfig{
				Enabled:           false,
				URL:               "http://192.168.100.2:8123",
				Token:             "",
				VerifySSL:         false,
				Timeout:           30,
				ReconnectInterval: 60,
				SyncInterval:      300,
				IncludedDomains:   []string{},
				ExcludedDomains:   []string{},
			},
			AI: &AIConfig{
				Enabled:         "false",
				DefaultProvider: "ollama",
				Providers: AIProvidersConfig{
					Ollama: &OllamaConfig{
						Enabled: false,
						URL:     "http://localhost:11434",
						Models:  []string{},
						Timeout: 30,
					},
					OpenAI: &OpenAIConfig{
						Enabled:     false,
						APIKey:      "",
						Model:       "gpt-3.5-turbo",
						MaxTokens:   2048,
						Temperature: 0.7,
					},
					Claude: &ClaudeConfig{
						Enabled:   false,
						APIKey:    "",
						Model:     "claude-3-sonnet-20240229",
						MaxTokens: 2048,
					},
					Gemini: &GeminiConfig{
						Enabled:        false,
						APIKey:         "",
						Model:          "gemini-pro",
						SafetySettings: make(map[string]string),
					},
				},
			},
			Energy: &EnergyConfig{
				Enabled:                false,
				UpdateInterval:         60,
				CostPerKWH:             0.12,
				Currency:               "USD",
				RetentionDays:          365,
				TrackIndividualDevices: true,
			},
			Network: &NetworkConfig{
				Enabled:            false,
				MonitorInterfaces:  []string{"eth0", "wlan0"},
				ScanInterval:       300,
				PortScanEnabled:    false,
				IntrusionDetection: false,
			},
		},
		Monitoring: MonitoringSectionConfig{
			Enabled:              true,
			MetricsRetentionDays: 30,
			HealthCheckInterval:  60,
			AlertThresholds: AlertThresholds{
				CPUUsage:     80.0,
				MemoryUsage:  90.0,
				DiskUsage:    85.0,
				ResponseTime: 5000,
				ErrorRate:    5.0,
			},
			Notifications: NotificationConfig{
				EmailEnabled:    false,
				EmailRecipients: []string{},
				WebhookEnabled:  false,
				SlackEnabled:    false,
			},
		},
		Security: SecuritySectionConfig{
			EncryptionEnabled:     false,
			AuditLogging:          true,
			FailedLoginTracking:   true,
			IPWhitelist:           []string{},
			IPBlacklist:           []string{},
			APIKeyRequired:        false,
			CSRFProtection:        true,
			ContentSecurityPolicy: false,
		},
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

// generateDeviceID generates a unique device ID
func generateDeviceID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("pma-device-%s-%d", hostname, time.Now().Unix())
}

// TrackError records a system error
func (s *Service) TrackError(errorType, source, message, details string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.errorCount++
	s.lastError = time.Now()

	// Add to error history
	errorRecord := ErrorRecord{
		Timestamp: time.Now(),
		Message:   message,
		Type:      errorType,
		Source:    source,
		Details:   details,
	}

	// Add to history, maintaining max size
	s.errorHistory = append(s.errorHistory, errorRecord)
	if len(s.errorHistory) > s.maxErrorHistory {
		// Remove oldest error
		s.errorHistory = s.errorHistory[1:]
	}

	// Log the error
	s.logger.WithFields(logrus.Fields{
		"error_type": errorType,
		"source":     source,
		"details":    details,
	}).Error(message)
}

// GetErrorCount returns the total error count
func (s *Service) GetErrorCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errorCount
}

// GetLastError returns the timestamp of the last error
func (s *Service) GetLastError() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

// GetErrorHistory returns recent error history
func (s *Service) GetErrorHistory(limit int) []ErrorRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.errorHistory) {
		limit = len(s.errorHistory)
	}

	// Return the most recent errors
	start := len(s.errorHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]ErrorRecord, limit)
	copy(result, s.errorHistory[start:])
	return result
}

// ClearErrorHistory clears the error history and resets counters
func (s *Service) ClearErrorHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.errorCount = 0
	s.errorHistory = s.errorHistory[:0]
	s.lastError = time.Time{}
}

// getEnvironment returns the current environment
func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	return "production"
}
