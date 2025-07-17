package devices

import "errors"

var (
	// Device validation errors
	ErrInvalidDeviceID    = errors.New("invalid device ID")
	ErrInvalidDeviceName  = errors.New("invalid device name")
	ErrInvalidDeviceType  = errors.New("invalid device type")
	ErrInvalidAdapterType = errors.New("invalid adapter type")

	// Device operation errors
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceOffline       = errors.New("device is offline")
	ErrCommandNotSupported = errors.New("command not supported")
	ErrInvalidCommand      = errors.New("invalid command")
	ErrInvalidParams       = errors.New("invalid command parameters")

	// Adapter errors
	ErrAdapterNotConnected  = errors.New("adapter not connected")
	ErrAdapterNotFound      = errors.New("adapter not found")
	ErrDiscoveryFailed      = errors.New("device discovery failed")
	ErrConnectionFailed     = errors.New("connection failed")
	ErrAuthenticationFailed = errors.New("authentication failed")

	// Manager errors
	ErrManagerNotInitialized = errors.New("device manager not initialized")
	ErrAdapterAlreadyExists  = errors.New("adapter already exists")
	ErrInvalidConfiguration  = errors.New("invalid configuration")
)
