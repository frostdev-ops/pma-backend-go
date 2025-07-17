package ring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	ringAPIBaseURL = "https://api.ring.com"
	ringOAuthURL   = "https://oauth.ring.com/oauth/token"
	ringClientID   = "ring_official_android"
	ringUserAgent  = "PMA-Ring-Integration/1.0"
	ringAPIVersion = "11"
)

// RingClient handles communication with Ring API
type RingClient struct {
	httpClient   *http.Client
	oauth2Config *oauth2.Config
	token        *oauth2.Token
	logger       *logrus.Logger
	baseURL      string
}

// RingCredentials contains Ring authentication credentials
type RingCredentials struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// RingAuthResponse represents Ring OAuth response
type RingAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// RingDeviceData represents raw device data from Ring API
type RingDeviceData struct {
	ID                int                    `json:"id"`
	Description       string                 `json:"description"`
	DeviceType        string                 `json:"device_type"`
	Location          RingLocation           `json:"location"`
	BatteryLife       *int                   `json:"battery_life"`
	Settings          map[string]interface{} `json:"settings"`
	Features          map[string]interface{} `json:"features"`
	Kind              string                 `json:"kind"`
	Latitude          float64                `json:"latitude"`
	Longitude         float64                `json:"longitude"`
	Address           string                 `json:"address"`
	Timezone          string                 `json:"timezone"`
	SubscribedMotions bool                   `json:"subscribed_motions"`
	HasSubscription   bool                   `json:"has_subscription"`
	StreamingEnabled  bool                   `json:"streaming_enabled"`
	MotionDetection   bool                   `json:"motion_detection"`
	DoorbellSnooze    map[string]interface{} `json:"doorbell_snooze"`
	Motion            RingMotionSettings     `json:"motion"`
}

// RingLocation represents device location information
type RingLocation struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserID   int    `json:"user_id"`
	Address  string `json:"address"`
	Timezone string `json:"timezone"`
}

// RingMotionSettings represents motion detection settings
type RingMotionSettings struct {
	DetectionEnabled bool   `json:"detection_enabled"`
	SnoozeMinutes    int    `json:"snooze_minutes"`
	SnoozeUntil      *int64 `json:"snooze_until"`
}

// RingEvent represents a Ring event (doorbell, motion, etc.)
type RingEvent struct {
	ID              int        `json:"id"`
	IDStr           string     `json:"id_str"`
	State           string     `json:"state"`
	Protocol        string     `json:"protocol"`
	DoorbotID       int        `json:"doorbot_id"`
	DoorbotName     string     `json:"doorbot_description"`
	DeviceKind      string     `json:"device_kind"`
	MotionDetection bool       `json:"motion"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	AnsweredAt      *time.Time `json:"answered_at"`
	ExternalID      string     `json:"external_id"`
	RecordingStatus string     `json:"recording_status"`
	RecordingURL    string     `json:"recording_url"`
	SnapshotURL     string     `json:"snapshot_url"`
	FavoriteID      *int       `json:"favorite_id"`
	Kind            string     `json:"kind"`
	Longitude       float64    `json:"longitude"`
	Latitude        float64    `json:"latitude"`
	PublicKey       string     `json:"public_key"`
	ExpiresAt       *time.Time `json:"expires_at"`
	HasSnapshot     bool       `json:"has_snapshot"`
	HasRecording    bool       `json:"has_recording"`
	StreamingURL    string     `json:"streaming_url"`
}

// NewRingClient creates a new Ring API client
func NewRingClient(logger *logrus.Logger) *RingClient {
	oauth2Config := &oauth2.Config{
		ClientID: ringClientID,
		Endpoint: oauth2.Endpoint{
			TokenURL: ringOAuthURL,
		},
		Scopes: []string{},
	}

	return &RingClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		oauth2Config: oauth2Config,
		logger:       logger,
		baseURL:      ringAPIBaseURL,
	}
}

// Authenticate authenticates with Ring using email and password
func (c *RingClient) Authenticate(ctx context.Context, creds RingCredentials) error {
	if creds.RefreshToken != "" {
		// Use refresh token if available
		return c.refreshToken(ctx, creds.RefreshToken)
	}

	// Authenticate with email/password
	data := url.Values{
		"grant_type": {"password"},
		"username":   {creds.Email},
		"password":   {creds.Password},
		"client_id":  {ringClientID},
		"scope":      {"client"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ringOAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", ringUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var authResp RingAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.token = &oauth2.Token{
		AccessToken:  authResp.AccessToken,
		RefreshToken: authResp.RefreshToken,
		TokenType:    authResp.TokenType,
		Expiry:       time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
	}

	c.logger.Info("Successfully authenticated with Ring")
	return nil
}

// refreshToken refreshes the OAuth token
func (c *RingClient) refreshToken(ctx context.Context, refreshToken string) error {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {ringClientID},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ringOAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", ringUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var authResp RingAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode refresh response: %w", err)
	}

	c.token = &oauth2.Token{
		AccessToken:  authResp.AccessToken,
		RefreshToken: authResp.RefreshToken,
		TokenType:    authResp.TokenType,
		Expiry:       time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
	}

	c.logger.Info("Successfully refreshed Ring token")
	return nil
}

// makeAuthenticatedRequest makes an authenticated request to Ring API
func (c *RingClient) makeAuthenticatedRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if c.token == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	// Check if token needs refresh
	if c.token.Expiry.Before(time.Now().Add(5 * time.Minute)) {
		if err := c.refreshToken(ctx, c.token.RefreshToken); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	url := c.baseURL + path

	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	req.Header.Set("User-Agent", ringUserAgent)
	req.Header.Set("X-API-LANG", "en")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return resp, nil
}

// GetDevices retrieves all Ring devices
func (c *RingClient) GetDevices(ctx context.Context) ([]RingDeviceData, error) {
	resp, err := c.makeAuthenticatedRequest(ctx, "GET", "/clients_api/ring_devices", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Doorbots           []RingDeviceData `json:"doorbots"`
		AuthorizedDoorbots []RingDeviceData `json:"authorized_doorbots"`
		Chimes             []RingDeviceData `json:"chimes"`
		StickupCams        []RingDeviceData `json:"stickup_cams"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode devices response: %w", err)
	}

	// Combine all device types
	devices := make([]RingDeviceData, 0)
	devices = append(devices, result.Doorbots...)
	devices = append(devices, result.AuthorizedDoorbots...)
	devices = append(devices, result.Chimes...)
	devices = append(devices, result.StickupCams...)

	return devices, nil
}

// GetEvents retrieves recent Ring events
func (c *RingClient) GetEvents(ctx context.Context, limit int) ([]RingEvent, error) {
	path := fmt.Sprintf("/clients_api/doorbots/history?limit=%d", limit)

	resp, err := c.makeAuthenticatedRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []RingEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode events response: %w", err)
	}

	return events, nil
}

// GetDevice retrieves a specific Ring device
func (c *RingClient) GetDevice(ctx context.Context, deviceID string) (*RingDeviceData, error) {
	devices, err := c.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if fmt.Sprintf("%d", device.ID) == deviceID {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", deviceID)
}

// SetLights controls device lights (if supported)
func (c *RingClient) SetLights(ctx context.Context, deviceID string, enabled bool) error {
	path := fmt.Sprintf("/clients_api/doorbots/%s/floodlight_light_%s", deviceID, map[bool]string{true: "on", false: "off"}[enabled])

	_, err := c.makeAuthenticatedRequest(ctx, "PUT", path, nil)
	return err
}

// SetSiren controls device siren (if supported)
func (c *RingClient) SetSiren(ctx context.Context, deviceID string, enabled bool) error {
	action := map[bool]string{true: "on", false: "off"}[enabled]
	path := fmt.Sprintf("/clients_api/doorbots/%s/siren_%s", deviceID, action)

	_, err := c.makeAuthenticatedRequest(ctx, "PUT", path, nil)
	return err
}

// GetSnapshot retrieves a snapshot URL for a device
func (c *RingClient) GetSnapshot(ctx context.Context, deviceID string) (string, error) {
	path := fmt.Sprintf("/clients_api/snapshots/image/%s", deviceID)

	resp, err := c.makeAuthenticatedRequest(ctx, "POST", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode snapshot response: %w", err)
	}

	return result.URL, nil
}

// GetLiveStreamURL retrieves a live stream URL for a device
func (c *RingClient) GetLiveStreamURL(ctx context.Context, deviceID string) (string, error) {
	path := fmt.Sprintf("/clients_api/dings/active/%s", deviceID)

	resp, err := c.makeAuthenticatedRequest(ctx, "GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		StreamingURL string `json:"streaming_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode stream response: %w", err)
	}

	return result.StreamingURL, nil
}

// IsAuthenticated checks if the client is authenticated
func (c *RingClient) IsAuthenticated() bool {
	return c.token != nil && c.token.Valid()
}

// GetRefreshToken returns the current refresh token
func (c *RingClient) GetRefreshToken() string {
	if c.token == nil {
		return ""
	}
	return c.token.RefreshToken
}
