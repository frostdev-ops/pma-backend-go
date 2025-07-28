package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a client for the PMA-Router API
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClient creates a new PMA-Router API client
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest performs an HTTP request to the PMA-Router API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/v1%s", c.baseURL, endpoint)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}

	return resp, nil
}

// SystemStatus represents the system status response
type SystemStatus struct {
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Uptime    int64  `json:"uptime"`
}

// GetSystemStatus retrieves the system status
func (c *Client) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", "/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var status SystemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
	State string `json:"state"`
	MTU   int    `json:"mtu"`
}

// GetNetworkInterfaces retrieves all network interfaces
func (c *Client) GetNetworkInterfaces(ctx context.Context) ([]NetworkInterface, error) {
	resp, err := c.makeRequest(ctx, "GET", "/interfaces", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var interfaces []NetworkInterface
	if err := json.NewDecoder(resp.Body).Decode(&interfaces); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return interfaces, nil
}

// TrafficStats represents traffic statistics
type TrafficStats struct {
	TotalPackets   int64                            `json:"total_packets"`
	TotalBytes     int64                            `json:"total_bytes"`
	PacketsPerSec  int64                            `json:"packets_per_sec"`
	BytesPerSec    int64                            `json:"bytes_per_sec"`
	InterfaceStats map[string]InterfaceTrafficStats `json:"interface_stats"`
	LastUpdated    string                           `json:"last_updated"`
}

// InterfaceTrafficStats represents traffic stats for a specific interface
type InterfaceTrafficStats struct {
	Name        string `json:"name"`
	RxPackets   int64  `json:"rx_packets"`
	TxPackets   int64  `json:"tx_packets"`
	RxBytes     int64  `json:"rx_bytes"`
	TxBytes     int64  `json:"tx_bytes"`
	RxErrors    int64  `json:"rx_errors"`
	TxErrors    int64  `json:"tx_errors"`
	RxDropped   int64  `json:"rx_dropped"`
	TxDropped   int64  `json:"tx_dropped"`
	LastUpdated string `json:"last_updated"`
}

// GetTrafficStats retrieves comprehensive traffic statistics
func (c *Client) GetTrafficStats(ctx context.Context) (*TrafficStats, error) {
	resp, err := c.makeRequest(ctx, "GET", "/traffic", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var stats TrafficStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// PortForwardingRule represents a port forwarding rule
type PortForwardingRule struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	InternalIP    string `json:"internal_ip"`
	InternalMAC   string `json:"internal_mac"`
	InternalPort  int    `json:"internal_port"`
	ExternalPort  int    `json:"external_port"`
	Protocol      string `json:"protocol"`
	Description   string `json:"description"`
	Hostname      string `json:"hostname"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Active        bool   `json:"active"`
	AutoSuggested bool   `json:"auto_suggested"`
}

// GetPortForwardingRules retrieves all port forwarding rules
func (c *Client) GetPortForwardingRules(ctx context.Context) (map[string]PortForwardingRule, error) {
	resp, err := c.makeRequest(ctx, "GET", "/port-forwarding/rules", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var rules map[string]PortForwardingRule
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return rules, nil
}

// CreatePortForwardingRequest represents a request to create a port forwarding rule
type CreatePortForwardingRequest struct {
	Name         string `json:"name,omitempty"`
	InternalIP   string `json:"internal_ip"`
	InternalMAC  string `json:"internal_mac,omitempty"`
	InternalPort int    `json:"internal_port"`
	ExternalPort int    `json:"external_port"`
	Protocol     string `json:"protocol"`
	Description  string `json:"description,omitempty"`
	Active       bool   `json:"active"`
}

// CreatePortForwardingRule creates a new port forwarding rule
func (c *Client) CreatePortForwardingRule(ctx context.Context, rule CreatePortForwardingRequest) (*PortForwardingRule, error) {
	resp, err := c.makeRequest(ctx, "POST", "/port-forwarding/rules", rule)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var createdRule PortForwardingRule
	if err := json.NewDecoder(resp.Body).Decode(&createdRule); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdRule, nil
}

// NetworkDevice represents a discovered network device
type NetworkDevice struct {
	IP           string     `json:"ip"`
	MAC          string     `json:"mac"`
	Hostname     string     `json:"hostname"`
	Manufacturer string     `json:"manufacturer"`
	DeviceType   string     `json:"device_type"`
	IsOnline     bool       `json:"is_online"`
	FirstSeen    string     `json:"first_seen"`
	LastSeen     string     `json:"last_seen"`
	ResponseTime int64      `json:"response_time"`
	OpenPorts    []OpenPort `json:"open_ports"`
	UserLabel    string     `json:"user_label"`
	DiscoveredBy []string   `json:"discovered_by"`
}

// OpenPort represents an open port on a device
type OpenPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	State    string `json:"state"`
}

// NetworkDevicesResponse represents the response for discovered devices
type NetworkDevicesResponse struct {
	Devices []NetworkDevice `json:"devices"`
	Count   int             `json:"count"`
}

// GetNetworkDevices retrieves discovered network devices
func (c *Client) GetNetworkDevices(ctx context.Context) (*NetworkDevicesResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/port-forwarding/devices", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var devices NetworkDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &devices, nil
}

// DeletePortForwardingRule deletes a port forwarding rule
func (c *Client) DeletePortForwardingRule(ctx context.Context, ruleID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/port-forwarding/rules/%s", ruleID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// UpdatePortForwardingRule updates an existing port forwarding rule
func (c *Client) UpdatePortForwardingRule(ctx context.Context, ruleID string, rule CreatePortForwardingRequest) (*PortForwardingRule, error) {
	resp, err := c.makeRequest(ctx, "PUT", fmt.Sprintf("/port-forwarding/rules/%s", ruleID), rule)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var updatedRule PortForwardingRule
	if err := json.NewDecoder(resp.Body).Decode(&updatedRule); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedRule, nil
}
