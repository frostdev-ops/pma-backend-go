# Shelly Device Discovery System

## Overview

The PMA Backend includes a comprehensive Shelly device discovery system that automatically detects and configures Shelly IoT devices on your network. The system supports both mDNS-based discovery and network scanning with automatic subnet detection to ensure maximum device coverage.

## Discovery Methods

### 1. mDNS Discovery (Bonjour/Zeroconf)
- **Service Type**: `_shelly._tcp`
- **Automatic**: Yes
- **Supported Generations**: All (Gen1, Gen2, Gen3, Gen4)
- **Advantages**: Fast, efficient, real-time discovery
- **Limitations**: Requires devices to broadcast mDNS

### 2. Network Range Scanning
- **Method**: HTTP endpoint probing
- **Automatic**: Yes (with subnet auto-detection)
- **Supported Generations**: All
- **Advantages**: Discovers devices that don't broadcast mDNS
- **Limitations**: Slower than mDNS

### 3. Automatic Subnet Detection (NEW)
- **Method**: Network interface enumeration
- **Automatic**: Yes
- **Configuration**: Highly configurable with filters
- **Advantages**: Automatically adapts to network changes
- **Security**: Only scans private network ranges

## Configuration

### Basic Configuration

```yaml
devices:
  shelly:
    enabled: true
    discovery_interval: "5m"
    discovery_timeout: "30s"
    network_scan_enabled: true
    network_scan_ranges: ["192.168.1.0/24"]  # Manual/fallback ranges
    
    # Automatic subnet detection (NEW)
    auto_detect_subnets: true
    exclude_loopback: true
    exclude_docker_interfaces: true
    min_subnet_size: 16  # Don't scan subnets smaller than /16
```

### Advanced Auto-Detection Configuration

```yaml
devices:
  shelly:
    # ... basic config ...
    
    # Advanced auto-detection options
    auto_detect_subnets: true
    auto_detect_interface_filter: ["eth0", "wlan0"]  # Only scan specific interfaces
    exclude_loopback: true
    exclude_docker_interfaces: true
    min_subnet_size: 24  # Only scan /24 networks or smaller
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto_detect_subnets` | boolean | `true` | Enable automatic network interface detection |
| `auto_detect_interface_filter` | array | `[]` | Only scan specified interfaces (empty = all interfaces) |
| `exclude_loopback` | boolean | `true` | Exclude loopback interfaces (lo, 127.0.0.1) |
| `exclude_docker_interfaces` | boolean | `true` | Exclude Docker and virtual interfaces |
| `min_subnet_size` | integer | `16` | Minimum subnet size to scan (larger numbers = smaller subnets) |
| `network_scan_ranges` | array | Manual ranges | Additional/fallback subnet ranges |

### Security Features

- **Private Networks Only**: Auto-detection only scans private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
- **Interface Filtering**: Automatically excludes virtual and Docker interfaces
- **Subnet Size Limits**: Prevents scanning overly large networks
- **Manual Override**: Manual ranges can be combined with auto-detection

## How Auto-Detection Works

### 1. Interface Discovery
```
INFO Auto-detecting local network subnets...
INFO Found network interface: eth0 (192.168.1.0/24)
INFO Found network interface: wlan0 (10.0.1.0/24)
INFO Excluding docker0 interface (docker interface)
```

### 2. Subnet Validation
- Checks if interface is up and active
- Validates private IP ranges only
- Applies size filters (min_subnet_size)
- Excludes virtual/Docker interfaces

### 3. Range Combination
```
INFO Auto-detected subnets: [192.168.1.0/24, 10.0.1.0/24]
INFO Combined scan ranges: [192.168.1.0/24, 10.0.1.0/24, 192.168.100.0/24]
```

### 4. Discovery Execution
- Starts mDNS discovery (continuous)
- Starts network scanning with combined ranges
- Monitors device health and availability

## Interface Filtering

### Automatic Exclusions (when `exclude_docker_interfaces: true`)
- `docker*` - Docker bridge interfaces
- `br-*` - Bridge interfaces
- `veth*` - Virtual Ethernet interfaces
- `lo` - Loopback interface
- `tun*` - Tunnel interfaces
- `tap*` - TAP interfaces

### Manual Filtering
```yaml
# Only scan specific interfaces
auto_detect_interface_filter: ["eth0", "wlan0"]

# Or scan all physical interfaces (empty array)
auto_detect_interface_filter: []
```

## Example Configurations

### Home Network (Default)
```yaml
devices:
  shelly:
    enabled: true
    auto_detect_subnets: true        # Auto-detect home network
    exclude_loopback: true
    exclude_docker_interfaces: true
    min_subnet_size: 16             # /16 to /32 networks
    network_scan_ranges: []         # No manual ranges needed
```

### Multi-VLAN Environment
```yaml
devices:
  shelly:
    enabled: true
    auto_detect_subnets: true
    auto_detect_interface_filter: ["eth0.100", "eth0.200"]  # Specific VLANs
    exclude_docker_interfaces: true
    min_subnet_size: 24             # /24 to /32 networks only
    network_scan_ranges: ["192.168.10.0/24"]  # Additional static range
```

### Docker Host
```yaml
devices:
  shelly:
    enabled: true
    auto_detect_subnets: true
    exclude_docker_interfaces: true  # Exclude Docker networks
    min_subnet_size: 20              # /20 to /32 networks
    network_scan_ranges: ["192.168.1.0/24"]  # Fallback range
```

### Development/Testing
```yaml
devices:
  shelly:
    enabled: true
    auto_detect_subnets: false       # Disable auto-detection
    network_scan_ranges: ["192.168.1.0/24", "10.0.0.0/24"]  # Manual only
```

## Troubleshooting Auto-Detection

### Check Auto-Detection Status
Look for these log messages on startup:

```
INFO Starting enhanced Shelly device discovery
INFO Auto-detecting local network subnets...
INFO Found network interface: eth0 (192.168.1.0/24)
INFO Auto-detected subnets: [192.168.1.0/24]
INFO Combined scan ranges: [192.168.1.0/24]
```

### Common Issues

#### No Subnets Detected
```
WARN No suitable subnets auto-detected
```
**Solutions:**
- Check if interfaces are up: `ip addr show`
- Verify private IP addresses are assigned
- Check `min_subnet_size` setting (lower number = larger networks)
- Add manual ranges as fallback

#### Auto-Detection Failed
```
WARN Auto-detection failed, falling back to manual ranges
```
**Solutions:**
- Check system permissions for network interface access
- Verify network interface enumeration works: `ip addr`
- Enable debug logging to see detailed error

#### Too Many/Wrong Interfaces
```
INFO Excluding docker0 interface (docker interface)
```
**Solutions:**
- Use `auto_detect_interface_filter` to specify exact interfaces
- Adjust `exclude_docker_interfaces` setting
- Increase `min_subnet_size` to exclude large networks

### Debug Commands

```bash
# Check network interfaces
ip addr show

# Test manual network scanning
curl -s http://192.168.1.100/shelly
curl -s http://192.168.1.100/rpc/Shelly.GetDeviceInfo

# Check PMA logs for auto-detection
journalctl -u pma-backend -f | grep "Auto-detect"
```

## Performance Considerations

### Network Scanning Impact
- **Concurrent Scanning**: Limited to 50 concurrent connections
- **Rate Limiting**: Built-in delays to prevent network flooding
- **Scan Frequency**: Every 5 minutes (configurable)
- **Timeout**: 10 seconds per device probe

### Auto-Detection Overhead
- **Startup Cost**: ~100ms additional startup time
- **Memory Usage**: Minimal (interface enumeration only)
- **CPU Usage**: Negligible
- **Network Impact**: None (only local interface queries)

### Optimization Tips
1. **Use Interface Filters**: Specify exact interfaces to scan
2. **Increase min_subnet_size**: Scan smaller networks only
3. **Disable for Large Networks**: Use manual ranges for /8 or /12 networks
4. **Monitor Logs**: Watch for performance warnings

## Migration from Manual Configuration

### Step 1: Enable Auto-Detection
```yaml
# Add to existing configuration
auto_detect_subnets: true
exclude_loopback: true
exclude_docker_interfaces: true
```

### Step 2: Verify Detection
Check logs for detected subnets and compare with your manual ranges.

### Step 3: Remove Redundant Manual Ranges
Keep only ranges that auto-detection doesn't cover:
```yaml
# Before
network_scan_ranges: ["192.168.1.0/24", "192.168.100.0/24", "10.0.0.0/24"]

# After (if auto-detection finds 192.168.1.0/24 and 10.0.0.0/24)
network_scan_ranges: ["192.168.100.0/24"]  # Only non-detected ranges
```

### Step 4: Test Discovery
Verify all your Shelly devices are still discovered after migration.

## Security Best Practices

1. **Network Segmentation**: Use VLANs to isolate IoT devices
2. **Firewall Rules**: Restrict Shelly device internet access
3. **Interface Filtering**: Only scan trusted network segments
4. **Subnet Size Limits**: Prevent scanning large networks
5. **Monitor Discovery**: Watch logs for unexpected subnet detection

## API Endpoints

### Discovery Status
```http
GET /api/v1/adapters/shelly/discovery/status
```

### Detected Subnets
```http
GET /api/v1/adapters/shelly/discovery/subnets
```

### Trigger Re-detection
```http
POST /api/v1/adapters/shelly/discovery/refresh
```

## Support

For auto-detection issues:
1. Check system logs for interface enumeration errors
2. Verify network interface configuration
3. Test manual discovery with known device IPs
4. Report issues with network topology details

The auto-detection feature is designed to work seamlessly in most environments while providing extensive configuration options for complex network setups. 