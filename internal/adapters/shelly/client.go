package shelly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/sirupsen/logrus"
)

const (
	shellyServiceType = "_http._tcp"
	shellyDomain      = "local."
	httpTimeout       = 10 * time.Second
	defaultUsername   = "admin"
)

// ShellyClient handles communication with Shelly devices
type ShellyClient struct {
	httpClient *http.Client
	logger     *logrus.Logger
	username   string
	password   string
}

// ShellyDevice represents basic Shelly device information
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

// DiscoveredDevice represents a discovered Shelly device
type DiscoveredDevice struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

// NewShellyClient creates a new Shelly client
func NewShellyClient(username, password string, logger *logrus.Logger) *ShellyClient {
	if username == "" {
		username = defaultUsername
	}

	return &ShellyClient{
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:   logger,
		username: username,
		password: password,
	}
}

// DiscoverDevices discovers Shelly devices on the local network using mDNS
func (c *ShellyClient) DiscoverDevices(ctx context.Context, timeout time.Duration) ([]DiscoveredDevice, error) {
	c.logger.Info("Starting Shelly device discovery via mDNS...")

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mDNS resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	var devices []DiscoveredDevice

	go func() {
		for entry := range entries {
			// Filter Shelly devices
			if strings.Contains(strings.ToLower(entry.Instance), "shelly") {
				device := DiscoveredDevice{
					IP:       entry.AddrIPv4[0].String(),
					Port:     entry.Port,
					Hostname: entry.HostName,
					Name:     entry.Instance,
					Type:     extractShellyType(entry.Instance),
				}
				devices = append(devices, device)

				c.logger.WithFields(logrus.Fields{
					"ip":       device.IP,
					"hostname": device.Hostname,
					"name":     device.Name,
					"type":     device.Type,
				}).Debug("Discovered Shelly device")
			}
		}
	}()

	discoveryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = resolver.Browse(discoveryCtx, shellyServiceType, shellyDomain, entries)
	close(entries)

	if err != nil {
		return nil, fmt.Errorf("mDNS discovery failed: %w", err)
	}

	c.logger.WithField("count", len(devices)).Info("Shelly device discovery completed")
	return devices, nil
}

// GetDeviceInfo retrieves device information from a Shelly device
func (c *ShellyClient) GetDeviceInfo(ctx context.Context, deviceIP string) (*ShellyDeviceInfo, error) {
	url := fmt.Sprintf("http://%s/shelly", deviceIP)

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info ShellyDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode device info: %w", err)
	}

	return &info, nil
}

// GetDeviceSettings retrieves device settings from a Shelly device
func (c *ShellyClient) GetDeviceSettings(ctx context.Context, deviceIP string) (*ShellySettings, error) {
	url := fmt.Sprintf("http://%s/settings", deviceIP)

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var settings ShellySettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode device settings: %w", err)
	}

	return &settings, nil
}

// GetDeviceStatus retrieves current status from a Shelly device
func (c *ShellyClient) GetDeviceStatus(ctx context.Context, deviceIP string) (*ShellyStatus, error) {
	url := fmt.Sprintf("http://%s/status", deviceIP)

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status ShellyStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode device status: %w", err)
	}

	return &status, nil
}

// SetRelay controls a relay output
func (c *ShellyClient) SetRelay(ctx context.Context, deviceIP string, relayIndex int, state bool, timer *int) error {
	params := url.Values{}
	params.Set("turn", map[bool]string{true: "on", false: "off"}[state])

	if timer != nil {
		params.Set("timer", strconv.Itoa(*timer))
	}

	url := fmt.Sprintf("http://%s/relay/%d?%s", deviceIP, relayIndex, params.Encode())

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SetLight controls a light (RGBW devices)
func (c *ShellyClient) SetLight(ctx context.Context, deviceIP string, lightIndex int, params map[string]interface{}) error {
	urlParams := url.Values{}

	if turn, ok := params["turn"].(bool); ok {
		urlParams.Set("turn", map[bool]string{true: "on", false: "off"}[turn])
	}

	if brightness, ok := params["brightness"].(int); ok {
		urlParams.Set("brightness", strconv.Itoa(brightness))
	}

	if red, ok := params["red"].(int); ok {
		urlParams.Set("red", strconv.Itoa(red))
	}

	if green, ok := params["green"].(int); ok {
		urlParams.Set("green", strconv.Itoa(green))
	}

	if blue, ok := params["blue"].(int); ok {
		urlParams.Set("blue", strconv.Itoa(blue))
	}

	if white, ok := params["white"].(int); ok {
		urlParams.Set("white", strconv.Itoa(white))
	}

	if temp, ok := params["temp"].(int); ok {
		urlParams.Set("temp", strconv.Itoa(temp))
	}

	if effect, ok := params["effect"].(int); ok {
		urlParams.Set("effect", strconv.Itoa(effect))
	}

	requestURL := fmt.Sprintf("http://%s/light/%d?%s", deviceIP, lightIndex, urlParams.Encode())

	resp, err := c.makeRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SetDimmer controls a dimmer
func (c *ShellyClient) SetDimmer(ctx context.Context, deviceIP string, dimmerIndex int, brightness int, turn *bool) error {
	params := url.Values{}
	params.Set("brightness", strconv.Itoa(brightness))

	if turn != nil {
		params.Set("turn", map[bool]string{true: "on", false: "off"}[*turn])
	}

	url := fmt.Sprintf("http://%s/light/%d?%s", deviceIP, dimmerIndex, params.Encode())

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Reboot reboots a Shelly device
func (c *ShellyClient) Reboot(ctx context.Context, deviceIP string) error {
	url := fmt.Sprintf("http://%s/reboot", deviceIP)

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// makeRequest makes an HTTP request to a Shelly device
func (c *ShellyClient) makeRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if password is set
	if c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	req.Header.Set("User-Agent", "PMA-Shelly-Integration/1.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return resp, nil
}

// IsDeviceReachable checks if a Shelly device is reachable
func (c *ShellyClient) IsDeviceReachable(ctx context.Context, deviceIP string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.GetDeviceInfo(ctx, deviceIP)
	return err == nil
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

// GetDeviceByHostname finds a device by hostname from discovered devices
func (c *ShellyClient) GetDeviceByHostname(devices []DiscoveredDevice, hostname string) *DiscoveredDevice {
	for _, device := range devices {
		if device.Hostname == hostname || device.Name == hostname {
			return &device
		}
	}
	return nil
}

// ValidateDevice validates that a device is actually a Shelly device
func (c *ShellyClient) ValidateDevice(ctx context.Context, deviceIP string) error {
	info, err := c.GetDeviceInfo(ctx, deviceIP)
	if err != nil {
		return fmt.Errorf("device validation failed: %w", err)
	}

	if !strings.Contains(strings.ToLower(info.Type), "shelly") {
		return fmt.Errorf("device is not a Shelly device: %s", info.Type)
	}

	return nil
}
