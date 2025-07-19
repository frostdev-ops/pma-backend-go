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
)

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

	s := &Service{
		deviceID:        generateDeviceID(),
		startTime:       time.Now(),
		logger:          logger,
		maxLogEntries:   maxLogEntries,
		logBuffer:       make([]LogEntry, 0, maxLogEntries),
		trackedServices: trackedServices,
		config: SystemConfig{
			Environment:     getEnvironment(),
			Debug:           false,
			LogLevel:        "info",
			MaxLogEntries:   maxLogEntries,
			UpdateChannel:   "stable",
			AutoUpdate:      false,
			MaintenanceMode: false,
			Services:        make(map[string]interface{}),
			Features:        make(map[string]bool),
		},
		fullConfig: cfg,
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
		Version:     "1.0.0", // TODO: Get from build info
		OS:          fmt.Sprintf("%s %s", hostInfo.OS, hostInfo.PlatformVersion),
		Arch:        runtime.GOARCH,
		Platform:    hostInfo.Platform,
		Hostname:    hostInfo.Hostname,
		LastSeen:    time.Now(),
		Uptime:      int64(hostInfo.Uptime),
		Environment: s.config.Environment,
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
	if runtime.GOOS == "linux" {
		cmd := exec.Command("ip", "route", "show", "default")
		if output, err := cmd.Output(); err == nil {
			fields := strings.Fields(string(output))
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					return fields[i+1]
				}
			}
		}
	}
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
		ErrorCount:   0, // TODO: Track error counts
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
	if service.ServiceName == "" {
		details["error"] = "No service name specified"
		return false, 0
	}

	cmd := exec.CommandContext(ctx, "systemctl", "show", service.ServiceName,
		"--property=ActiveState,SubState,MainPID,ExecMainStartTimestamp,LoadState,UnitFileState")

	output, err := cmd.Output()
	if err != nil {
		details["error"] = err.Error()
		details["service_name"] = service.ServiceName
		return false, 0
	}

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
	return s.config
}

// UpdateConfig updates the system configuration
func (s *Service) UpdateConfig(config SystemConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
	s.logger.Info("System configuration updated")

	return nil
}

// generateDeviceID generates a unique device ID
func generateDeviceID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("pma-device-%s-%d", hostname, time.Now().Unix())
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
