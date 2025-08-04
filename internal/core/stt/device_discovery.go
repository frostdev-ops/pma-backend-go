package stt

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Additional methods for AudioManager to handle device discovery

// discoverUSBMicrophones discovers USB audio input devices
func (am *AudioManager) discoverUSBMicrophones() {
	devices, err := am.listUSBAudioDevices()
	if err != nil {
		am.logger.WithError(err).Error("Failed to discover USB audio devices")
		return
	}

	for _, device := range devices {
		// Check if device is already known
		am.mu.RLock()
		_, exists := am.devices[device.ID]
		am.mu.RUnlock()

		if !exists {
			am.addDevice(device)
		} else {
			// Update last seen time
			am.mu.Lock()
			if existing, ok := am.devices[device.ID]; ok {
				existing.LastSeen = time.Now()
				existing.Quality = device.Quality
			}
			am.mu.Unlock()
		}
	}
}

// discoverBluetoothDevices discovers Bluetooth audio devices
func (am *AudioManager) discoverBluetoothDevices() {
	devices, err := am.listBluetoothAudioDevices()
	if err != nil {
		am.logger.WithError(err).Error("Failed to discover Bluetooth audio devices")
		return
	}

	for _, device := range devices {
		// Check if device is already known
		am.mu.RLock()
		_, exists := am.devices[device.ID]
		am.mu.RUnlock()

		if !exists {
			am.addDevice(device)
		} else {
			// Update device status
			am.mu.Lock()
			if existing, ok := am.devices[device.ID]; ok {
				existing.LastSeen = time.Now()
				existing.IsActive = device.IsActive
				existing.Quality = device.Quality
			}
			am.mu.Unlock()
		}
	}
}

// discoverInternalMicrophones discovers built-in microphones
func (am *AudioManager) discoverInternalMicrophones() {
	devices, err := am.listInternalAudioDevices()
	if err != nil {
		am.logger.WithError(err).Error("Failed to discover internal audio devices")
		return
	}

	for _, device := range devices {
		// Check if device is already known
		am.mu.RLock()
		_, exists := am.devices[device.ID]
		am.mu.RUnlock()

		if !exists {
			am.addDevice(device)
		}
	}
}

// listUSBAudioDevices lists USB audio input devices using system commands
func (am *AudioManager) listUSBAudioDevices() ([]*AudioDevice, error) {
	devices := make([]*AudioDevice, 0)

	// Try using arecord first (ALSA)
	if alsaDevices, err := am.listALSADevices(); err == nil {
		devices = append(devices, alsaDevices...)
	}

	// Try using PulseAudio
	if pulseDevices, err := am.listPulseAudioDevices(); err == nil {
		devices = append(devices, pulseDevices...)
	}

	// Filter for USB devices
	usbDevices := make([]*AudioDevice, 0)
	for _, device := range devices {
		if strings.Contains(strings.ToLower(device.Name), "usb") ||
			strings.Contains(strings.ToLower(device.ID), "usb") {
			device.Source = SourceUSB
			device.Priority = 4 // High priority for USB mics
			usbDevices = append(usbDevices, device)
		}
	}

	return usbDevices, nil
}

// listALSADevices lists ALSA audio devices
func (am *AudioManager) listALSADevices() ([]*AudioDevice, error) {
	cmd := exec.Command("arecord", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list ALSA devices: %w", err)
	}

	devices := make([]*AudioDevice, 0)
	lines := strings.Split(string(output), "\n")

	// Parse arecord output
	// Format: card 1: USB [USB Audio], device 0: USB Audio [USB Audio]
	cardRegex := regexp.MustCompile(`card (\d+): (.+) \[(.+)\], device (\d+): (.+) \[(.+)\]`)

	for _, line := range lines {
		matches := cardRegex.FindStringSubmatch(line)
		if len(matches) >= 7 {
			cardNum := matches[1]
			deviceNum := matches[4]
			deviceName := matches[3]

			device := &AudioDevice{
				ID:       fmt.Sprintf("alsa_hw_%s_%s", cardNum, deviceNum),
				Name:     deviceName,
				Source:   SourceUSB, // Will be corrected later if not USB
				IsActive: true,
				Priority: 3,
				Quality:  0.8,
				Format: AudioFormat{
					SampleRate: am.config.DefaultSampleRate,
					Channels:   am.config.DefaultChannels,
					BitDepth:   16,
					Format:     "pcm",
				},
				LastSeen:     time.Now(),
				Capabilities: []string{"full_duplex"},
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// listPulseAudioDevices lists PulseAudio devices
func (am *AudioManager) listPulseAudioDevices() ([]*AudioDevice, error) {
	cmd := exec.Command("pactl", "list", "short", "sources")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list PulseAudio devices: %w", err)
	}

	devices := make([]*AudioDevice, 0)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			deviceName := parts[1]

			// Skip monitor sources (output devices)
			if strings.Contains(deviceName, ".monitor") {
				continue
			}

			device := &AudioDevice{
				ID:       fmt.Sprintf("pulse_%s", deviceName),
				Name:     deviceName,
				Source:   SourceUSB, // Will be corrected later
				IsActive: true,
				Priority: 3,
				Quality:  0.8,
				Format: AudioFormat{
					SampleRate: am.config.DefaultSampleRate,
					Channels:   am.config.DefaultChannels,
					BitDepth:   16,
					Format:     "pcm",
				},
				LastSeen:     time.Now(),
				Capabilities: []string{"full_duplex"},
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// listBluetoothAudioDevices lists connected Bluetooth audio devices
func (am *AudioManager) listBluetoothAudioDevices() ([]*AudioDevice, error) {
	devices := make([]*AudioDevice, 0)

	// Try using bluetoothctl to list connected devices
	cmd := exec.Command("bluetoothctl", "devices", "Connected")
	output, err := cmd.Output()
	if err != nil {
		// Bluetooth might not be available
		return devices, nil
	}

	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse bluetooth device line
		// Format: Device AA:BB:CC:DD:EE:FF Device Name
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[0] == "Device" {
			macAddress := parts[1]
			deviceName := strings.Join(parts[2:], " ")

			// Check if device supports audio
			if am.isBluetoothAudioDevice(macAddress) {
				device := &AudioDevice{
					ID:       fmt.Sprintf("bluetooth_%s", strings.ReplaceAll(macAddress, ":", "_")),
					Name:     fmt.Sprintf("Bluetooth: %s", deviceName),
					Source:   SourceBluetooth,
					IsActive: true,
					Priority: 2,   // Lower priority than USB
					Quality:  0.6, // Bluetooth typically has lower quality
					Format: AudioFormat{
						SampleRate: 16000, // Bluetooth often uses lower sample rates
						Channels:   1,
						BitDepth:   16,
						Format:     "pcm",
					},
					LastSeen:     time.Now(),
					Capabilities: []string{"wake_word"},
				}

				devices = append(devices, device)
			}
		}
	}

	return devices, nil
}

// isBluetoothAudioDevice checks if a Bluetooth device supports audio
func (am *AudioManager) isBluetoothAudioDevice(macAddress string) bool {
	// Check device info for audio capabilities
	cmd := exec.Command("bluetoothctl", "info", macAddress)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	info := string(output)

	// Look for audio service UUIDs
	audioUUIDs := []string{
		"0000110b", // Audio Sink
		"0000110c", // Remote Control Target
		"0000110e", // A2DP
		"0000111e", // Handsfree
		"00001108", // Headset
	}

	for _, uuid := range audioUUIDs {
		if strings.Contains(strings.ToLower(info), uuid) {
			return true
		}
	}

	// Also check for common audio device names
	audioKeywords := []string{
		"headset", "headphone", "speaker", "microphone",
		"audio", "a2dp", "handsfree", "earphone", "earbud",
	}

	infoLower := strings.ToLower(info)
	for _, keyword := range audioKeywords {
		if strings.Contains(infoLower, keyword) {
			return true
		}
	}

	return false
}

// listInternalAudioDevices discovers built-in microphones
func (am *AudioManager) listInternalAudioDevices() ([]*AudioDevice, error) {
	devices := make([]*AudioDevice, 0)

	// Check for built-in microphone on Raspberry Pi
	if am.isRaspberryPi() {
		// Check if the built-in audio is enabled
		if am.hasBuiltInAudio() {
			device := &AudioDevice{
				ID:       "internal_microphone",
				Name:     "Built-in Microphone",
				Source:   SourceInternal,
				IsActive: true,
				Priority: 1,   // Lowest priority
				Quality:  0.4, // Built-in mics are usually lower quality
				Format: AudioFormat{
					SampleRate: am.config.DefaultSampleRate,
					Channels:   1,
					BitDepth:   16,
					Format:     "pcm",
				},
				LastSeen:     time.Now(),
				Capabilities: []string{"wake_word"},
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// Helper functions
func (am *AudioManager) isRaspberryPi() bool {
	// Check if running on Raspberry Pi
	cmd := exec.Command("cat", "/proc/device-tree/model")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	model := strings.ToLower(string(output))
	return strings.Contains(model, "raspberry pi")
}

func (am *AudioManager) hasBuiltInAudio() bool {
	// Check if ALSA recognizes built-in audio
	cmd := exec.Command("aplay", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	audioInfo := strings.ToLower(string(output))
	return strings.Contains(audioInfo, "bcm2835") ||
		strings.Contains(audioInfo, "headphones") ||
		strings.Contains(audioInfo, "built-in")
}

// monitorUSBDevices continuously monitors for USB device changes
func (am *AudioManager) monitorUSBDevices(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check for new or disconnected USB devices
			am.checkUSBDeviceChanges()
		}
	}
}

// monitorBluetoothDevices continuously monitors for Bluetooth device changes
func (am *AudioManager) monitorBluetoothDevices(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check for new or disconnected Bluetooth devices
			am.checkBluetoothDeviceChanges()
		}
	}
}

func (am *AudioManager) checkUSBDeviceChanges() {
	currentDevices, err := am.listUSBAudioDevices()
	if err != nil {
		am.logger.WithError(err).Error("Failed to check USB device changes")
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Mark all USB devices as potentially disconnected
	for _, device := range am.devices {
		if device.Source == SourceUSB {
			device.IsActive = false
		}
	}

	// Update with current devices
	for _, currentDevice := range currentDevices {
		if existing, exists := am.devices[currentDevice.ID]; exists {
			existing.IsActive = true
			existing.LastSeen = time.Now()
		} else {
			am.devices[currentDevice.ID] = currentDevice
			// Send device connected event
			select {
			case am.deviceEvents <- DeviceEvent{
				Type:      "connected",
				Device:    currentDevice,
				Timestamp: time.Now(),
			}:
			default:
			}
		}
	}

	// Remove devices that are no longer active
	disconnectedDevices := make([]string, 0)
	for id, device := range am.devices {
		if device.Source == SourceUSB && !device.IsActive {
			disconnectedDevices = append(disconnectedDevices, id)
		}
	}

	for _, id := range disconnectedDevices {
		device := am.devices[id]
		delete(am.devices, id)

		// Send device disconnected event
		select {
		case am.deviceEvents <- DeviceEvent{
			Type:      "disconnected",
			Device:    device,
			Timestamp: time.Now(),
		}:
		default:
		}
	}
}

func (am *AudioManager) checkBluetoothDeviceChanges() {
	currentDevices, err := am.listBluetoothAudioDevices()
	if err != nil {
		am.logger.WithError(err).Error("Failed to check Bluetooth device changes")
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Mark all Bluetooth devices as potentially disconnected
	for _, device := range am.devices {
		if device.Source == SourceBluetooth {
			device.IsActive = false
		}
	}

	// Update with current devices
	for _, currentDevice := range currentDevices {
		if existing, exists := am.devices[currentDevice.ID]; exists {
			existing.IsActive = true
			existing.LastSeen = time.Now()
		} else {
			am.devices[currentDevice.ID] = currentDevice
			// Send device connected event
			select {
			case am.deviceEvents <- DeviceEvent{
				Type:      "connected",
				Device:    currentDevice,
				Timestamp: time.Now(),
			}:
			default:
			}
		}
	}

	// Remove devices that are no longer active
	disconnectedDevices := make([]string, 0)
	for id, device := range am.devices {
		if device.Source == SourceBluetooth && !device.IsActive {
			disconnectedDevices = append(disconnectedDevices, id)
		}
	}

	for _, id := range disconnectedDevices {
		device := am.devices[id]
		delete(am.devices, id)

		// Send device disconnected event
		select {
		case am.deviceEvents <- DeviceEvent{
			Type:      "disconnected",
			Device:    device,
			Timestamp: time.Now(),
		}:
		default:
		}
	}
}

// processEvents processes device events and forwards them
func (am *AudioManager) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-am.deviceEvents:
			am.logger.WithFields(logrus.Fields{
				"event_type":  event.Type,
				"device_id":   event.Device.ID,
				"device_name": event.Device.Name,
				"source":      event.Device.Source,
			}).Info("Audio device event")

			// Additional processing for specific event types
			if event.Type == "connected" && am.config.AutoSourceSwitch {
				// Auto-switch to higher priority devices
				am.checkAutoSourceSwitch(event.Device)
			}
		}
	}
}

// processAudio processes incoming audio chunks
func (am *AudioManager) processAudio(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-am.audioOutput:
			// This is where audio chunks would be forwarded to the wake word engine
			// and STT processor. The actual processing is handled by the enhanced STT service.
			am.logger.WithFields(logrus.Fields{
				"source":    chunk.Source,
				"device_id": chunk.DeviceID,
				"size":      len(chunk.Data),
				"quality":   chunk.Quality,
			}).Debug("Audio chunk processed")
		}
	}
}

func (am *AudioManager) checkAutoSourceSwitch(newDevice *AudioDevice) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Find current primary device
	var currentPrimary *AudioDevice
	highestPriority := 0

	for _, device := range am.devices {
		if device.IsActive && device.Priority > highestPriority {
			highestPriority = device.Priority
			currentPrimary = device
		}
	}

	// Switch to new device if it has higher priority
	if newDevice.Priority > highestPriority {
		am.logger.WithFields(logrus.Fields{
			"new_device":   newDevice.Name,
			"new_priority": newDevice.Priority,
			"old_device": func() string {
				if currentPrimary != nil {
					return currentPrimary.Name
				}
				return "none"
			}(),
			"old_priority": highestPriority,
		}).Info("Auto-switching primary audio device")

		// Set new device as primary
		newDevice.Priority = 10 // Highest priority

		// Send device change event
		select {
		case am.deviceEvents <- DeviceEvent{
			Type:      "primary_changed",
			Device:    newDevice,
			Timestamp: time.Now(),
		}:
		default:
		}
	}
}
