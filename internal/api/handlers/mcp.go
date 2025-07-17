package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// MCPServerConfig represents MCP server configuration
type MCPServerConfig struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	CWD         string            `json:"cwd"`
	Timeout     int               `json:"timeout"`
	Enabled     bool              `json:"enabled"`
	AutoRestart bool              `json:"autoRestart"`
	MaxRestarts int               `json:"maxRestarts"`
}

// MCPServerStatus represents the status of an MCP server
type MCPServerStatus struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Status       string              `json:"status"` // connecting, connected, disconnected, error, disabled
	PID          int                 `json:"pid,omitempty"`
	Uptime       int64               `json:"uptime,omitempty"`
	LastError    string              `json:"lastError,omitempty"`
	RestartCount int                 `json:"restartCount"`
	Tools        []MCPTool           `json:"tools"`
	Resources    []MCPResource       `json:"resources"`
	Statistics   MCPServerStatistics `json:"statistics"`
	LastActivity time.Time           `json:"lastActivity"`
}

// MCPTool represents an MCP tool
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerID    string                 `json:"serverId"`
}

// MCPResource represents an MCP resource
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
	ServerID    string `json:"serverId"`
}

// MCPServerStatistics represents server statistics
type MCPServerStatistics struct {
	ToolCalls        int64     `json:"toolCalls"`
	ResourceReads    int64     `json:"resourceReads"`
	Errors           int64     `json:"errors"`
	Uptime           int64     `json:"uptime"`
	LastToolCall     time.Time `json:"lastToolCall"`
	LastResourceRead time.Time `json:"lastResourceRead"`
}

// MCPToolExecutionRequest represents a tool execution request
type MCPToolExecutionRequest struct {
	ServerID  string                 `json:"serverId"`
	ToolName  string                 `json:"toolName"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolExecutionResponse represents a tool execution response
type MCPToolExecutionResponse struct {
	Success       bool        `json:"success"`
	Result        interface{} `json:"result,omitempty"`
	Error         string      `json:"error,omitempty"`
	ExecutionTime int64       `json:"executionTime"`
	ServerID      string      `json:"serverId"`
	ToolName      string      `json:"toolName"`
}

// MCPHandler handles MCP operations
type MCPHandler struct {
	servers    map[string]*MCPServerStatus
	configs    map[string]*MCPServerConfig
	processes  map[string]*exec.Cmd
	serversMux sync.RWMutex
	log        *log.Logger
	enabled    bool
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(logger *log.Logger) *MCPHandler {
	return &MCPHandler{
		servers:   make(map[string]*MCPServerStatus),
		configs:   make(map[string]*MCPServerConfig),
		processes: make(map[string]*exec.Cmd),
		log:       logger,
		enabled:   true,
	}
}

// GetMCPStatus returns the overall MCP status
func (h *MCPHandler) GetMCPStatus(c *gin.Context) {
	h.serversMux.RLock()
	defer h.serversMux.RUnlock()

	connected := 0
	total := len(h.servers)
	errors := []string{}

	for _, server := range h.servers {
		if server.Status == "connected" {
			connected++
		}
		if server.LastError != "" {
			errors = append(errors, fmt.Sprintf("%s: %s", server.Name, server.LastError))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":          h.enabled,
			"connectedServers": connected,
			"totalServers":     total,
			"lastActivity":     time.Now().Format(time.RFC3339),
			"errors":           errors,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetMCPServers returns all MCP servers
func (h *MCPHandler) GetMCPServers(c *gin.Context) {
	h.serversMux.RLock()
	defer h.serversMux.RUnlock()

	servers := make([]gin.H, 0, len(h.servers))
	for _, server := range h.servers {
		servers = append(servers, gin.H{
			"id":            server.ID,
			"name":          server.Name,
			"status":        server.Status,
			"toolCount":     len(server.Tools),
			"resourceCount": len(server.Resources),
			"uptime":        server.Uptime,
			"lastError":     server.LastError,
			"restartCount":  server.RestartCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      servers,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// AddMCPServer adds a new MCP server
func (h *MCPHandler) AddMCPServer(c *gin.Context) {
	var config MCPServerConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid server configuration: " + err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Validate required fields
	if config.ID == "" || config.Name == "" || config.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "ID, name, and command are required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	h.serversMux.Lock()
	defer h.serversMux.Unlock()

	// Check if server already exists
	if _, exists := h.servers[config.ID]; exists {
		c.JSON(http.StatusConflict, gin.H{
			"success":   false,
			"error":     "Server with this ID already exists",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.MaxRestarts == 0 {
		config.MaxRestarts = 3
	}

	// Create server status
	status := &MCPServerStatus{
		ID:           config.ID,
		Name:         config.Name,
		Status:       "disconnected",
		Tools:        []MCPTool{},
		Resources:    []MCPResource{},
		Statistics:   MCPServerStatistics{},
		LastActivity: time.Now(),
	}

	h.configs[config.ID] = &config
	h.servers[config.ID] = status

	h.log.Printf("Added MCP server: %s (%s)", config.Name, config.ID)

	// Auto-connect if enabled
	if config.Enabled {
		go h.connectServer(config.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"serverId": config.ID,
			"message":  "Server added successfully",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RemoveMCPServer removes an MCP server
func (h *MCPHandler) RemoveMCPServer(c *gin.Context) {
	serverID := c.Param("serverId")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Server ID is required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	h.serversMux.Lock()
	defer h.serversMux.Unlock()

	// Check if server exists
	if _, exists := h.servers[serverID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Server not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Stop the server if running
	if process, exists := h.processes[serverID]; exists {
		if process.Process != nil {
			process.Process.Kill()
		}
		delete(h.processes, serverID)
	}

	// Remove from maps
	delete(h.servers, serverID)
	delete(h.configs, serverID)

	h.log.Printf("Removed MCP server: %s", serverID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Server removed successfully",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ConnectMCPServer connects to an MCP server
func (h *MCPHandler) ConnectMCPServer(c *gin.Context) {
	serverID := c.Param("serverId")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Server ID is required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	h.serversMux.RLock()
	_, configExists := h.configs[serverID]
	server, serverExists := h.servers[serverID]
	h.serversMux.RUnlock()

	if !configExists || !serverExists {
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Server not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	if server.Status == "connected" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"message": "Server already connected",
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	go h.connectServer(serverID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Connection initiated",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// DisconnectMCPServer disconnects from an MCP server
func (h *MCPHandler) DisconnectMCPServer(c *gin.Context) {
	serverID := c.Param("serverId")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Server ID is required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	h.serversMux.Lock()
	defer h.serversMux.Unlock()

	// Check if server exists
	server, exists := h.servers[serverID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Server not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Stop the process if running
	if process, exists := h.processes[serverID]; exists {
		if process.Process != nil {
			process.Process.Kill()
		}
		delete(h.processes, serverID)
	}

	// Update status
	server.Status = "disconnected"
	server.PID = 0
	server.Uptime = 0

	h.log.Printf("Disconnected MCP server: %s", serverID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Server disconnected successfully",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetMCPTools returns all available tools
func (h *MCPHandler) GetMCPTools(c *gin.Context) {
	serverID := c.Query("serverId")

	h.serversMux.RLock()
	defer h.serversMux.RUnlock()

	var tools []MCPTool

	if serverID != "" {
		// Get tools for specific server
		if server, exists := h.servers[serverID]; exists {
			tools = server.Tools
		}
	} else {
		// Get tools from all servers
		for _, server := range h.servers {
			if server.Status == "connected" {
				tools = append(tools, server.Tools...)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      tools,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ExecuteMCPTools executes MCP tools
func (h *MCPHandler) ExecuteMCPTools(c *gin.Context) {
	var request MCPToolExecutionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid execution request: " + err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	if request.ServerID == "" || request.ToolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "ServerID and ToolName are required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	start := time.Now()

	h.serversMux.RLock()
	server, exists := h.servers[request.ServerID]
	h.serversMux.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Server not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	if server.Status != "connected" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success":   false,
			"error":     "Server not connected",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Mock execution for now - in a real implementation, this would
	// communicate with the actual MCP server process
	executionTime := time.Since(start).Milliseconds()

	response := MCPToolExecutionResponse{
		Success: true,
		Result: map[string]interface{}{
			"message":   fmt.Sprintf("Tool %s executed successfully", request.ToolName),
			"arguments": request.Arguments,
		},
		ExecutionTime: executionTime,
		ServerID:      request.ServerID,
		ToolName:      request.ToolName,
	}

	// Update statistics
	h.serversMux.Lock()
	server.Statistics.ToolCalls++
	server.Statistics.LastToolCall = time.Now()
	server.LastActivity = time.Now()
	h.serversMux.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      response,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// connectServer connects to an MCP server (internal method)
func (h *MCPHandler) connectServer(serverID string) {
	h.serversMux.RLock()
	config := h.configs[serverID]
	server := h.servers[serverID]
	h.serversMux.RUnlock()

	if config == nil || server == nil {
		return
	}

	h.serversMux.Lock()
	server.Status = "connecting"
	h.serversMux.Unlock()

	h.log.Printf("Connecting to MCP server: %s", config.Name)

	// Mock connection for now - in a real implementation, this would
	// start the actual MCP server process and establish communication
	time.Sleep(2 * time.Second)

	h.serversMux.Lock()
	server.Status = "connected"
	server.PID = 12345 // Mock PID
	server.Uptime = time.Now().Unix()
	server.LastActivity = time.Now()

	// Mock some tools and resources
	server.Tools = []MCPTool{
		{
			Name:        "file_reader",
			Description: "Read files from the filesystem",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
			ServerID: serverID,
		},
		{
			Name:        "web_search",
			Description: "Search the web for information",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
			ServerID: serverID,
		},
	}

	server.Resources = []MCPResource{
		{
			URI:         "file:///tmp/example.txt",
			Name:        "example.txt",
			Description: "Example text file",
			MimeType:    "text/plain",
			ServerID:    serverID,
		},
	}

	h.serversMux.Unlock()

	h.log.Printf("Connected to MCP server: %s", config.Name)
}

// Shutdown gracefully shuts down the MCP handler
func (h *MCPHandler) Shutdown() {
	h.serversMux.Lock()
	defer h.serversMux.Unlock()

	// Stop all processes
	for serverID, process := range h.processes {
		if process.Process != nil {
			h.log.Printf("Stopping MCP server process: %s", serverID)
			process.Process.Kill()
		}
	}

	h.log.Println("MCP handler shutdown complete")
}
