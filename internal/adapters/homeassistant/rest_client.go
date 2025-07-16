package homeassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RESTClient interface defines REST API operations
type RESTClient interface {
	// Core API calls
	GetConfig(ctx context.Context) (*HAConfig, error)
	GetStates(ctx context.Context) ([]EntityState, error)
	GetState(ctx context.Context, entityID string) (*EntityState, error)
	SetState(ctx context.Context, entityID string, state interface{}, attributes map[string]interface{}) error

	// Service calls
	CallService(ctx context.Context, domain, service string, data map[string]interface{}) error

	// Areas/Rooms
	GetAreas(ctx context.Context) ([]Area, error)
	GetArea(ctx context.Context, areaID string) (*Area, error)

	// Devices
	GetDevices(ctx context.Context) ([]Device, error)

	// Raw API call for extensibility
	DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error)
}

// restClient implements RESTClient
type restClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *logrus.Logger

	// Rate limiting and retry configuration
	requestTimeout time.Duration
	maxRetries     int
	retryDelay     time.Duration
	maxRetryDelay  time.Duration
}

// NewRESTClient creates a new REST client
func NewRESTClient(baseURL, token string, logger *logrus.Logger) RESTClient {
	return &restClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:         logger,
		requestTimeout: 30 * time.Second,
		maxRetries:     3,
		retryDelay:     time.Second,
		maxRetryDelay:  10 * time.Second,
	}
}

// GetConfig retrieves Home Assistant configuration
func (c *restClient) GetConfig(ctx context.Context) (*HAConfig, error) {
	c.logger.Debug("Getting Home Assistant configuration")

	data, err := c.DoRequest(ctx, "GET", "/api/config", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	var config HAConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, NewHAError(0, "Failed to parse config response", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.logger.WithFields(logrus.Fields{
		"version":  config.Version,
		"timezone": config.TimeZone,
	}).Debug("Retrieved Home Assistant configuration")

	return &config, nil
}

// GetStates retrieves all entity states
func (c *restClient) GetStates(ctx context.Context) ([]EntityState, error) {
	c.logger.Debug("Getting all entity states")

	data, err := c.DoRequest(ctx, "GET", "/api/states", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get states: %w", err)
	}

	var states []EntityState
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, NewHAError(0, "Failed to parse states response", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.logger.WithField("count", len(states)).Debug("Retrieved entity states")
	return states, nil
}

// GetState retrieves a specific entity state
func (c *restClient) GetState(ctx context.Context, entityID string) (*EntityState, error) {
	c.logger.WithField("entity_id", entityID).Debug("Getting entity state")

	path := fmt.Sprintf("/api/states/%s", entityID)
	data, err := c.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get state for entity %s: %w", entityID, err)
	}

	var state EntityState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, NewHAError(0, "Failed to parse state response", map[string]interface{}{
			"entity_id": entityID,
			"error":     err.Error(),
		})
	}

	c.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"state":     state.State,
	}).Debug("Retrieved entity state")

	return &state, nil
}

// SetState sets an entity state
func (c *restClient) SetState(ctx context.Context, entityID string, state interface{}, attributes map[string]interface{}) error {
	c.logger.WithField("entity_id", entityID).Debug("Setting entity state")

	body := map[string]interface{}{
		"state": state,
	}
	if attributes != nil {
		body["attributes"] = attributes
	}

	path := fmt.Sprintf("/api/states/%s", entityID)
	_, err := c.DoRequest(ctx, "POST", path, body)
	if err != nil {
		return fmt.Errorf("failed to set state for entity %s: %w", entityID, err)
	}

	c.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"state":     state,
	}).Debug("Set entity state")

	return nil
}

// CallService calls a Home Assistant service
func (c *restClient) CallService(ctx context.Context, domain, service string, data map[string]interface{}) error {
	c.logger.WithFields(logrus.Fields{
		"domain":  domain,
		"service": service,
	}).Debug("Calling Home Assistant service")

	path := fmt.Sprintf("/api/services/%s/%s", domain, service)

	body := make(map[string]interface{})
	if data != nil {
		for k, v := range data {
			body[k] = v
		}
	}

	_, err := c.DoRequest(ctx, "POST", path, body)
	if err != nil {
		return fmt.Errorf("failed to call service %s.%s: %w", domain, service, err)
	}

	c.logger.WithFields(logrus.Fields{
		"domain":  domain,
		"service": service,
	}).Debug("Called Home Assistant service")

	return nil
}

// GetAreas retrieves all areas
func (c *restClient) GetAreas(ctx context.Context) ([]Area, error) {
	c.logger.Debug("Getting all areas")

	data, err := c.DoRequest(ctx, "GET", "/api/config/area_registry", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get areas: %w", err)
	}

	var areas []Area
	if err := json.Unmarshal(data, &areas); err != nil {
		return nil, NewHAError(0, "Failed to parse areas response", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.logger.WithField("count", len(areas)).Debug("Retrieved areas")
	return areas, nil
}

// GetArea retrieves a specific area
func (c *restClient) GetArea(ctx context.Context, areaID string) (*Area, error) {
	c.logger.WithField("area_id", areaID).Debug("Getting area")

	areas, err := c.GetAreas(ctx)
	if err != nil {
		return nil, err
	}

	for _, area := range areas {
		if area.AreaID == areaID {
			c.logger.WithField("area_id", areaID).Debug("Found area")
			return &area, nil
		}
	}

	return nil, NewHAError(404, "Area not found", map[string]interface{}{
		"area_id": areaID,
	})
}

// GetDevices retrieves all devices
func (c *restClient) GetDevices(ctx context.Context) ([]Device, error) {
	c.logger.Debug("Getting all devices")

	data, err := c.DoRequest(ctx, "GET", "/api/config/device_registry", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	var devices []Device
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, NewHAError(0, "Failed to parse devices response", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.logger.WithField("count", len(devices)).Debug("Retrieved devices")
	return devices, nil
}

// DoRequest performs a raw HTTP request with retry logic and proper error handling
func (c *restClient) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, NewHAError(0, "Failed to marshal request body", map[string]interface{}{
				"error": err.Error(),
			})
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Implement retry logic with exponential backoff
	var lastErr error
	retryDelay := c.retryDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				// Continue with retry
			}

			// Exponential backoff
			retryDelay *= 2
			if retryDelay > c.maxRetryDelay {
				retryDelay = c.maxRetryDelay
			}

			// Reset body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBody)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			lastErr = NewHAError(0, "Failed to create request", map[string]interface{}{
				"error": err.Error(),
				"url":   url,
			})
			continue
		}

		// Set headers
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		c.logger.WithFields(logrus.Fields{
			"method":  method,
			"url":     url,
			"attempt": attempt + 1,
		}).Debug("Making HTTP request to Home Assistant")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = NewHAError(0, "HTTP request failed", map[string]interface{}{
				"error":   err.Error(),
				"url":     url,
				"attempt": attempt + 1,
			})

			c.logger.WithFields(logrus.Fields{
				"error":   err.Error(),
				"attempt": attempt + 1,
			}).Warn("HTTP request failed, will retry")
			continue
		}

		defer resp.Body.Close()
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = NewHAError(0, "Failed to read response body", map[string]interface{}{
				"error":       err.Error(),
				"status_code": resp.StatusCode,
			})
			continue
		}

		c.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"method":      method,
			"url":         url,
		}).Debug("Received HTTP response from Home Assistant")

		// Handle HTTP status codes
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseBody, nil
		}

		// Handle specific error status codes
		switch resp.StatusCode {
		case 401:
			return nil, ErrUnauthorized
		case 404:
			return nil, ErrEntityNotFound
		case 429:
			// Rate limited - wait longer before retry
			retryDelay = 5 * time.Second
			lastErr = NewHAError(resp.StatusCode, "Rate limited", map[string]interface{}{
				"response": string(responseBody),
			})
			continue
		default:
			// For 5xx errors, retry; for 4xx errors (except above), don't retry
			if resp.StatusCode >= 500 {
				lastErr = NewHAError(resp.StatusCode, "Server error", map[string]interface{}{
					"response": string(responseBody),
				})
				continue
			}

			// Client error - don't retry
			return nil, NewHAError(resp.StatusCode, "Client error", map[string]interface{}{
				"response": string(responseBody),
			})
		}
	}

	c.logger.WithError(lastErr).Error("All retry attempts failed")
	return nil, lastErr
}
