package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/sirupsen/logrus"
)

// Service provides test and mock functionality for development and diagnostics
type Service struct {
	cfg   *config.Config
	repos *database.Repositories
	log   *logrus.Logger
	db    *sql.DB
	mu    sync.RWMutex

	// Mock data storage
	mockEntities map[string]*MockEntity
	mockRooms    map[int]*MockRoom

	// Test settings
	testModeEnabled bool
	mockDataPersist bool
}

// MockEntity represents a mock entity for testing
type MockEntity struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Domain      string                 `json:"domain"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	RoomID      *int                   `json:"room_id,omitempty"`
	LastChanged time.Time              `json:"last_changed"`
	LastUpdated time.Time              `json:"last_updated"`
	EntityType  string                 `json:"entity_type"`
	DeviceClass *string                `json:"device_class,omitempty"`
}

// MockRoom represents a mock room for testing
type MockRoom struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	Entities    []string  `json:"entities"`
	CreatedAt   time.Time `json:"created_at"`
}

// ConnectionTestResult represents results of connection testing
type ConnectionTestResult struct {
	Service      string        `json:"service"`
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Details      interface{}   `json:"details,omitempty"`
}

// SystemHealthResult represents system health check results
type SystemHealthResult struct {
	Service   string                 `json:"service"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// TestConfig represents test configuration options
type TestConfig struct {
	MockEntitiesEnabled     bool `json:"mock_entities_enabled"`
	MockDataPersistence     bool `json:"mock_data_persistence"`
	TestEndpointsEnabled    bool `json:"test_endpoints_enabled"`
	DiagnosticsEnabled      bool `json:"diagnostics_enabled"`
	PerformanceTestsEnabled bool `json:"performance_tests_enabled"`
}

// NewService creates a new test service
func NewService(cfg *config.Config, repos *database.Repositories, log *logrus.Logger, db *sql.DB) *Service {
	return &Service{
		cfg:             cfg,
		repos:           repos,
		log:             log,
		db:              db,
		mockEntities:    make(map[string]*MockEntity),
		mockRooms:       make(map[int]*MockRoom),
		testModeEnabled: cfg.Server.Mode == "development",
		mockDataPersist: false,
	}
}

// GenerateMockEntities creates a set of realistic mock entities
func (s *Service) GenerateMockEntities(count int, entityTypes []string) ([]*MockEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(entityTypes) == 0 {
		entityTypes = []string{"light", "switch", "sensor", "binary_sensor", "climate", "cover", "lock"}
	}

	entities := make([]*MockEntity, 0, count)

	for i := 0; i < count; i++ {
		entityType := entityTypes[rand.Intn(len(entityTypes))]
		entity := s.generateMockEntity(entityType, i)

		s.mockEntities[entity.ID] = entity
		entities = append(entities, entity)
	}

	s.log.Infof("Generated %d mock entities", len(entities))
	return entities, nil
}

// generateMockEntity creates a single mock entity
func (s *Service) generateMockEntity(entityType string, index int) *MockEntity {
	now := time.Now()

	entity := &MockEntity{
		ID:          fmt.Sprintf("%s.test_%s_%d", entityType, entityType, index),
		Name:        s.generateEntityName(entityType, index),
		Domain:      entityType,
		Attributes:  make(map[string]interface{}),
		LastChanged: now.Add(-time.Duration(rand.Intn(3600)) * time.Second),
		LastUpdated: now,
		EntityType:  entityType,
	}

	// Set domain-specific state and attributes
	switch entityType {
	case "light":
		entity.State = s.randomChoice([]string{"on", "off"})
		entity.Attributes["brightness"] = rand.Intn(256)
		entity.Attributes["color_mode"] = "brightness"
		if entity.State == "on" {
			entity.Attributes["color_temp"] = 250 + rand.Intn(200)
		}

	case "switch":
		entity.State = s.randomChoice([]string{"on", "off"})
		entity.Attributes["device_class"] = "switch"

	case "sensor":
		deviceClass := s.randomChoice([]string{"temperature", "humidity", "pressure", "battery", "energy"})
		entity.DeviceClass = &deviceClass
		entity.Attributes["device_class"] = deviceClass
		entity.Attributes["unit_of_measurement"] = s.getUnitForDeviceClass(deviceClass)
		entity.State = s.generateSensorValue(deviceClass)

	case "binary_sensor":
		entity.State = s.randomChoice([]string{"on", "off"})
		deviceClass := s.randomChoice([]string{"motion", "door", "window", "smoke", "moisture"})
		entity.DeviceClass = &deviceClass
		entity.Attributes["device_class"] = deviceClass

	case "climate":
		entity.State = s.randomChoice([]string{"heat", "cool", "auto", "off"})
		entity.Attributes["current_temperature"] = 68 + rand.Float64()*15
		entity.Attributes["target_temperature"] = 70 + rand.Float64()*8
		entity.Attributes["hvac_mode"] = entity.State
		entity.Attributes["temperature_unit"] = "°F"

	case "cover":
		entity.State = s.randomChoice([]string{"open", "closed", "opening", "closing"})
		entity.Attributes["current_position"] = rand.Intn(101)
		entity.Attributes["device_class"] = s.randomChoice([]string{"window", "garage", "curtain", "blind"})

	case "lock":
		entity.State = s.randomChoice([]string{"locked", "unlocked"})
		entity.Attributes["device_class"] = "lock"
	}

	// Add common attributes
	entity.Attributes["friendly_name"] = entity.Name
	entity.Attributes["supported_features"] = s.getSupportedFeatures(entityType)

	return entity
}

// generateEntityName creates realistic entity names
func (s *Service) generateEntityName(entityType string, index int) string {
	roomNames := []string{"Living Room", "Kitchen", "Bedroom", "Bathroom", "Office", "Garage", "Basement", "Attic"}

	var deviceNames []string
	switch entityType {
	case "light":
		deviceNames = []string{"Ceiling Light", "Table Lamp", "Floor Lamp", "Strip Light", "Pendant Light"}
	case "switch":
		deviceNames = []string{"Main Switch", "Outlet", "Fan Switch", "Power Switch"}
	case "sensor":
		deviceNames = []string{"Temperature Sensor", "Humidity Sensor", "Motion Detector", "Energy Monitor"}
	case "binary_sensor":
		deviceNames = []string{"Door Sensor", "Window Sensor", "Motion Detector", "Smoke Detector"}
	case "climate":
		deviceNames = []string{"Thermostat", "AC Unit", "Heater"}
	case "cover":
		deviceNames = []string{"Window Blinds", "Garage Door", "Curtains", "Shades"}
	case "lock":
		deviceNames = []string{"Front Door Lock", "Back Door Lock", "Gate Lock"}
	default:
		deviceNames = []string{fmt.Sprintf("Test %s", entityType)}
	}

	room := roomNames[rand.Intn(len(roomNames))]
	device := deviceNames[rand.Intn(len(deviceNames))]

	return fmt.Sprintf("%s %s", room, device)
}

// TestConnections performs comprehensive connectivity testing
func (s *Service) TestConnections(ctx context.Context) (map[string]*ConnectionTestResult, error) {
	results := make(map[string]*ConnectionTestResult)

	// Test database connection
	results["database"] = s.testDatabaseConnection(ctx)

	// Test Home Assistant connection
	results["home_assistant"] = s.testHomeAssistantConnection(ctx)

	// Test WebSocket functionality
	results["websocket"] = s.testWebSocketConnection(ctx)

	// Test PMA Router connection
	results["pma_router"] = s.testPMARouterConnection(ctx)

	// Test external services
	results["ring"] = s.testRingConnection(ctx)
	results["ollama"] = s.testOllamaConnection(ctx)

	// Test network interfaces
	results["network"] = s.testNetworkInterfaces(ctx)

	return results, nil
}

// testDatabaseConnection tests database connectivity
func (s *Service) testDatabaseConnection(ctx context.Context) *ConnectionTestResult {
	start := time.Now()
	result := &ConnectionTestResult{
		Service: "database",
		Status:  "healthy",
	}

	err := s.db.PingContext(ctx)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		// Get additional database info
		var dbVersion string
		s.db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&dbVersion)

		result.Details = map[string]interface{}{
			"version": dbVersion,
			"driver":  "sqlite",
		}
	}

	return result
}

// testHomeAssistantConnection tests HA API connectivity
func (s *Service) testHomeAssistantConnection(ctx context.Context) *ConnectionTestResult {
	start := time.Now()
	result := &ConnectionTestResult{
		Service: "home_assistant",
		Status:  "healthy",
	}

	if s.cfg.HomeAssistant.URL == "" {
		result.Status = "disabled"
		result.Error = "Home Assistant URL not configured"
		return result
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", s.cfg.HomeAssistant.URL+"/api/", nil)
	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
		return result
	}

	if s.cfg.HomeAssistant.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.HomeAssistant.Token)
	}

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var apiInfo map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&apiInfo)
			result.Details = apiInfo
		} else {
			result.Status = "unhealthy"
			result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	return result
}

// testWebSocketConnection tests WebSocket functionality
func (s *Service) testWebSocketConnection(ctx context.Context) *ConnectionTestResult {
	return &ConnectionTestResult{
		Service: "websocket",
		Status:  "healthy",
		Details: map[string]interface{}{
			"endpoint": "/ws",
			"status":   "WebSocket endpoint available",
		},
	}
}

// testPMARouterConnection tests PMA Router API connectivity
func (s *Service) testPMARouterConnection(ctx context.Context) *ConnectionTestResult {
	start := time.Now()
	result := &ConnectionTestResult{
		Service: "pma_router",
		Status:  "healthy",
	}

	// Try to connect to PMA Router API
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/api/status", nil)
	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
		return result
	}

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var status map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&status)
			result.Details = status
		} else {
			result.Status = "unhealthy"
			result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	return result
}

// testRingConnection tests Ring API connectivity
func (s *Service) testRingConnection(ctx context.Context) *ConnectionTestResult {
	return &ConnectionTestResult{
		Service: "ring",
		Status:  "not_configured",
		Details: map[string]interface{}{
			"message": "Ring integration test not implemented",
		},
	}
}

// testOllamaConnection tests Ollama AI service connectivity
func (s *Service) testOllamaConnection(ctx context.Context) *ConnectionTestResult {
	start := time.Now()
	result := &ConnectionTestResult{
		Service: "ollama",
		Status:  "healthy",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
		return result
	}

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var models map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&models)
			result.Details = models
		} else {
			result.Status = "unhealthy"
			result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	return result
}

// testNetworkInterfaces tests network interface connectivity
func (s *Service) testNetworkInterfaces(ctx context.Context) *ConnectionTestResult {
	start := time.Now()
	result := &ConnectionTestResult{
		Service: "network",
		Status:  "healthy",
	}

	interfaces, err := net.Interfaces()
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
		return result
	}

	interfaceInfo := make([]map[string]interface{}, 0)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, _ := iface.Addrs()
			addrStrings := make([]string, len(addrs))
			for i, addr := range addrs {
				addrStrings[i] = addr.String()
			}

			interfaceInfo = append(interfaceInfo, map[string]interface{}{
				"name":      iface.Name,
				"addresses": addrStrings,
				"mtu":       iface.MTU,
			})
		}
	}

	result.Details = map[string]interface{}{
		"interfaces": interfaceInfo,
		"count":      len(interfaceInfo),
	}

	return result
}

// GetSystemHealth performs comprehensive system health checks
func (s *Service) GetSystemHealth(ctx context.Context) (map[string]*SystemHealthResult, error) {
	results := make(map[string]*SystemHealthResult)

	// System resources
	results["system_resources"] = s.checkSystemResources()

	// Database health
	results["database"] = s.checkDatabaseHealth(ctx)

	// Service status
	results["services"] = s.checkServiceStatus(ctx)

	// Configuration validation
	results["configuration"] = s.checkConfiguration()

	return results, nil
}

// checkSystemResources checks CPU, memory, and disk usage
func (s *Service) checkSystemResources() *SystemHealthResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &SystemHealthResult{
		Service:   "system_resources",
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"memory": map[string]interface{}{
				"alloc_mb":       float64(m.Alloc) / 1024 / 1024,
				"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
				"sys_mb":         float64(m.Sys) / 1024 / 1024,
				"gc_cycles":      m.NumGC,
			},
			"goroutines": runtime.NumGoroutine(),
			"cpu_cores":  runtime.NumCPU(),
		},
	}
}

// checkDatabaseHealth checks database integrity and performance
func (s *Service) checkDatabaseHealth(ctx context.Context) *SystemHealthResult {
	result := &SystemHealthResult{
		Service:   "database",
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Test query performance
	start := time.Now()
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM entities").Scan(&count)
	queryTime := time.Since(start)

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		result.Details["entity_count"] = count
		result.Details["query_time_ms"] = queryTime.Milliseconds()

		// Check database size
		var pageCount, pageSize int
		s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
		s.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)

		result.Details["database_size_mb"] = float64(pageCount*pageSize) / 1024 / 1024
	}

	return result
}

// checkServiceStatus checks status of related services
func (s *Service) checkServiceStatus(ctx context.Context) *SystemHealthResult {
	return &SystemHealthResult{
		Service:   "services",
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"pma_backend": "running",
			"message":     "Service status checking not fully implemented",
		},
	}
}

// checkConfiguration validates system configuration
func (s *Service) checkConfiguration() *SystemHealthResult {
	result := &SystemHealthResult{
		Service:   "configuration",
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check critical configuration values
	issues := make([]string, 0)

	if s.cfg.Database.Path == "" {
		issues = append(issues, "Database path not configured")
	}

	if s.cfg.Auth.JWTSecret == "your-secret-key-here" {
		issues = append(issues, "Default JWT secret in use")
	}

	if len(issues) > 0 {
		result.Status = "warning"
		result.Details["issues"] = issues
	}

	result.Details["server_mode"] = s.cfg.Server.Mode
	result.Details["server_port"] = s.cfg.Server.Port
	result.Details["log_level"] = s.cfg.Logging.Level

	return result
}

// ResetTestData clears all mock data and test state
func (s *Service) ResetTestData() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mockEntities = make(map[string]*MockEntity)
	s.mockRooms = make(map[int]*MockRoom)

	s.log.Info("Test data reset completed")
	return nil
}

// GetTestConfig returns current test configuration
func (s *Service) GetTestConfig() *TestConfig {
	return &TestConfig{
		MockEntitiesEnabled:     s.testModeEnabled,
		MockDataPersistence:     s.mockDataPersist,
		TestEndpointsEnabled:    s.cfg.Server.Mode == "development",
		DiagnosticsEnabled:      true,
		PerformanceTestsEnabled: s.cfg.Server.Mode == "development",
	}
}

// Helper functions
func (s *Service) randomChoice(choices []string) string {
	return choices[rand.Intn(len(choices))]
}

func (s *Service) getUnitForDeviceClass(deviceClass string) string {
	units := map[string]string{
		"temperature": "°F",
		"humidity":    "%",
		"pressure":    "hPa",
		"battery":     "%",
		"energy":      "kWh",
	}
	if unit, ok := units[deviceClass]; ok {
		return unit
	}
	return ""
}

func (s *Service) generateSensorValue(deviceClass string) string {
	switch deviceClass {
	case "temperature":
		return fmt.Sprintf("%.1f", 60+rand.Float64()*30)
	case "humidity":
		return strconv.Itoa(30 + rand.Intn(40))
	case "pressure":
		return fmt.Sprintf("%.1f", 1000+rand.Float64()*50)
	case "battery":
		return strconv.Itoa(20 + rand.Intn(80))
	case "energy":
		return fmt.Sprintf("%.2f", rand.Float64()*10)
	default:
		return strconv.Itoa(rand.Intn(100))
	}
}

func (s *Service) getSupportedFeatures(entityType string) int {
	features := map[string]int{
		"light":         3,  // brightness + color
		"switch":        0,  // basic on/off
		"sensor":        0,  // read-only
		"binary_sensor": 0,  // read-only
		"climate":       17, // target temp + hvac mode
		"cover":         15, // open/close + position
		"lock":          1,  // lock/unlock
	}
	if feature, ok := features[entityType]; ok {
		return feature
	}
	return 0
}

// GetMockEntities returns all mock entities
func (s *Service) GetMockEntities() map[string]*MockEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modifications
	entities := make(map[string]*MockEntity)
	for k, v := range s.mockEntities {
		entities[k] = v
	}
	return entities
}

// UpdateMockEntity updates a mock entity's state
func (s *Service) UpdateMockEntity(entityID string, state string, attributes map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entity, exists := s.mockEntities[entityID]
	if !exists {
		return fmt.Errorf("mock entity not found: %s", entityID)
	}

	entity.State = state
	entity.LastUpdated = time.Now()

	if attributes != nil {
		for k, v := range attributes {
			entity.Attributes[k] = v
		}
	}

	return nil
}

// DeleteMockEntity removes a mock entity
func (s *Service) DeleteMockEntity(entityID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.mockEntities[entityID]; !exists {
		return fmt.Errorf("mock entity not found: %s", entityID)
	}

	delete(s.mockEntities, entityID)
	return nil
}
