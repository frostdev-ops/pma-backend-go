package devices

import (
	"errors"
	"fmt"
)

// Common device errors
var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceOffline       = errors.New("device is offline")
	ErrCommandNotSupported = errors.New("command not supported")
	ErrInvalidState        = errors.New("invalid device state")
	ErrAdapterNotFound     = errors.New("adapter not found")
	ErrConnectionFailed    = errors.New("connection failed")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrDiscoveryFailed     = errors.New("device discovery failed")
	ErrInvalidDeviceType   = errors.New("invalid device type")
	ErrInvalidCapability   = errors.New("invalid capability")
	ErrTimeout             = errors.New("operation timeout")
	ErrRateLimited         = errors.New("rate limited")
)

// DeviceError represents a device-specific error
type DeviceError struct {
	DeviceID string
	Type     DeviceType
	Op       string
	Err      error
}

func (e *DeviceError) Error() string {
	return fmt.Sprintf("device error: device=%s type=%s op=%s: %v", 
		e.DeviceID, e.Type, e.Op, e.Err)
}

func (e *DeviceError) Unwrap() error {
	return e.Err
}

// NewDeviceError creates a new device error
func NewDeviceError(deviceID string, deviceType DeviceType, op string, err error) error {
	return &DeviceError{
		DeviceID: deviceID,
		Type:     deviceType,
		Op:       op,
		Err:      err,
	}
}

// AdapterError represents an adapter-specific error
type AdapterError struct {
	Adapter string
	Op      string
	Err     error
}

func (e *AdapterError) Error() string {
	return fmt.Sprintf("adapter error: adapter=%s op=%s: %v", e.Adapter, e.Op, e.Err)
}

func (e *AdapterError) Unwrap() error {
	return e.Err
}

// NewAdapterError creates a new adapter error
func NewAdapterError(adapter, op string, err error) error {
	return &AdapterError{
		Adapter: adapter,
		Op:      op,
		Err:     err,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field=%s value=%v: %s", 
		e.Field, e.Value, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for specific retryable errors
	switch {
	case errors.Is(err, ErrConnectionFailed),
		 errors.Is(err, ErrTimeout),
		 errors.Is(err, ErrDeviceOffline),
		 errors.Is(err, ErrRateLimited):
		return true
	}
	
	return false
}