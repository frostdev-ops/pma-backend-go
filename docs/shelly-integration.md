# Shelly Device Integration

## Overview

The PMAutomation system includes comprehensive support for Shelly devices through a dedicated adapter that provides automatic discovery, device management, and control capabilities.

## Features

- **Automatic Device Discovery**: Discovers Shelly devices on the network using mDNS and network scanning
- **Multi-Generation Support**: Supports Gen1, Gen2, Gen3, and Gen4 Shelly devices
- **Device Types Supported**:
  - Switches (relays)
  - Lights (including dimmers and RGBW)
  - Sensors (temperature, humidity)
  - Covers (rollers, shutters)
- **Real-time Status Updates**: Monitors device status and synchronizes with the unified entity system
- **WiFi Configuration**: Supports automatic WiFi setup for new devices

## API Endpoints

### Discovery
- `POST /api/shelly/discover` - Start device discovery
- `GET /api/shelly/devices` - List all Shelly devices
- `GET /api/shelly/devices/:id` - Get specific device details

### Control
- `POST /api/shelly/devices/:id/control` - Control a device (on/off, brightness, color, etc.)

### Configuration
- `PUT /api/shelly/config` - Update Shelly adapter configuration
- `GET /api/shelly/status` - Get adapter status and health

## Configuration

The Shelly adapter can be configured in the application config file:

```yaml
devices:
  shelly:
    enabled: true
    discovery_interval: "5m"
    discovery_timeout: "30s"
    network_scan_enabled: true
    network_scan_ranges:
      - "192.168.1.0/24"
    auto_wifi_setup: false
    poll_interval: "30s"
    health_check_interval: "1m"
```

## Usage

### Frontend Service

The frontend includes a ShellyService for interacting with Shelly devices:

```typescript
import shellyService from ''services/shellyService'' (see below for file content);

// Discover devices
const result = await shellyService.discoverDevices();

// List devices
const devices = await shellyService.listDevices();

// Control a device
await shellyService.controlDevice(deviceId, {
  action: 'turn_on',
  parameters: { brightness: 80 }
});
```

### Entity Integration

Shelly devices are automatically integrated with the unified entity system and appear as standard entities (switches, lights, sensors, covers) in the UI. They can be controlled through the standard entity controls without any special handling.

## Troubleshooting

1. **Devices not discovered**: Ensure devices are on the same network and mDNS is not blocked
2. **Memory issues**: The adapter includes proper cleanup mechanisms for goroutines and channels
3. **Authentication**: Some Shelly devices may require authentication - configure username/password in the adapter settings

## Auto-Configuration Module

The system includes an advanced auto-configuration module for Shelly devices that can:
- Automatically detect unconfigured devices
- Configure WiFi settings
- Register devices with the system
- Support AI-assisted configuration workflows