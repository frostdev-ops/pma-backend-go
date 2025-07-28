package ups

import (
	"encoding/binary"
	"fmt"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

// MAX17040 represents the MAX17040 fuel gauge IC used in Geekworm X1202 UPS
type MAX17040 struct {
	dev  i2c.Dev
	addr uint16
}

// Register addresses for MAX17040 as specified in the documentation
const (
	MAX17040_I2C_ADDRESS = 0x36 // 7-bit I2C slave address

	// Register addresses (16-bit registers)
	VCELL_REG   = 0x02 // Cell voltage register
	SOC_REG     = 0x04 // State of charge register
	MODE_REG    = 0x06 // Mode register for commands
	VERSION_REG = 0x08 // IC version register
	RCOMP_REG   = 0x0C // Compensation register for tuning
	COMMAND_REG = 0xFE // Command register for reset

	// Default values
	DEFAULT_RCOMP = 0x9700 // Factory default RCOMP value

	// Commands
	QUICK_START_CMD = 0x4000 // Quick start command for MODE register
	POR_CMD         = 0x0054 // Power-on reset command for COMMAND register
)

// MAX17040Data represents the data read from the MAX17040 fuel gauge
type MAX17040Data struct {
	Voltage       float64   // Battery voltage in volts
	StateOfCharge float64   // State of charge in percentage
	IsCharging    bool      // Whether the battery is charging
	IsConnected   bool      // Whether AC power is connected
	LastUpdated   time.Time // Last time data was updated
	ICVersion     uint16    // IC version for diagnostics
}

// NewMAX17040 creates a new MAX17040 client with I2C detection
func NewMAX17040() (*MAX17040, error) {
	// Initialize the host drivers
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize host drivers: %w", err)
	}

	// Open the I2C bus (usually bus 1 on Raspberry Pi)
	bus, err := i2creg.Open("1")
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus: %w", err)
	}

	// Create device handle
	dev := &i2c.Dev{Addr: MAX17040_I2C_ADDRESS, Bus: bus}

	max17040 := &MAX17040{
		dev:  *dev,
		addr: MAX17040_I2C_ADDRESS,
	}

	// Test connectivity by reading the VERSION register
	if err := max17040.testConnection(); err != nil {
		return nil, fmt.Errorf("MAX17040 not detected on I2C bus: %w", err)
	}

	return max17040, nil
}

// testConnection verifies that the MAX17040 is present and responding
func (m *MAX17040) testConnection() error {
	// Try to read the VERSION register to verify the device is present
	_, err := m.readRegister16(VERSION_REG)
	if err != nil {
		return fmt.Errorf("failed to read VERSION register: %w", err)
	}
	return nil
}

// readRegister16 reads a 16-bit register with proper endianness handling
// The MAX17040 is a big-endian device, but most processors are little-endian
func (m *MAX17040) readRegister16(regAddr uint8) (uint16, error) {
	// Prepare write buffer with register address
	writeBuffer := []byte{regAddr}
	// Prepare read buffer for 16-bit response
	readBuffer := make([]byte, 2)

	// Perform I2C transaction (write register address, then read 16-bit value)
	if err := m.dev.Tx(writeBuffer, readBuffer); err != nil {
		return 0, fmt.Errorf("I2C transaction failed: %w", err)
	}

	// Convert from big-endian (device format) to host byte order
	// The device sends MSB first, then LSB
	value := binary.BigEndian.Uint16(readBuffer)

	return value, nil
}

// writeRegister16 writes a 16-bit value to a register with proper endianness
func (m *MAX17040) writeRegister16(regAddr uint8, value uint16) error {
	// Prepare buffer: register address + 16-bit value in big-endian format
	buffer := make([]byte, 3)
	buffer[0] = regAddr
	binary.BigEndian.PutUint16(buffer[1:], value)

	// Write to device
	if _, err := m.dev.Write(buffer); err != nil {
		return fmt.Errorf("I2C write failed: %w", err)
	}

	return nil
}

// GetCellVoltage reads and calculates the battery cell voltage
// VCELL format: 12 bits, left-aligned. LSB = 1.25 mV
func (m *MAX17040) GetCellVoltage() (float64, error) {
	rawValue, err := m.readRegister16(VCELL_REG)
	if err != nil {
		return 0, err
	}

	// The 12-bit value is left-aligned in the 16-bit register
	// So we need to shift right by 4 bits to get the actual value
	// Each LSB represents 1.25mV
	voltage := float64(rawValue>>4) * 0.00125 // Convert to volts

	return voltage, nil
}

// GetStateOfCharge reads and calculates the battery state of charge
// SOC format: High byte is integer %, Low byte is fractional part (1/256%)
func (m *MAX17040) GetStateOfCharge() (float64, error) {
	rawValue, err := m.readRegister16(SOC_REG)
	if err != nil {
		return 0, err
	}

	// Extract integer and fractional parts
	integerPart := float64(rawValue >> 8)        // High byte
	fractionalPart := float64(rawValue & 0x00FF) // Low byte

	// Calculate final SOC: integer part + (fractional part / 256)
	soc := integerPart + (fractionalPart / 256.0)

	return soc, nil
}

// GetVersion returns the IC version for diagnostics
func (m *MAX17040) GetVersion() (uint16, error) {
	return m.readRegister16(VERSION_REG)
}

// GetRCOMP returns the current compensation value
func (m *MAX17040) GetRCOMP() (uint16, error) {
	return m.readRegister16(RCOMP_REG)
}

// SetRCOMP sets the compensation value for battery tuning
func (m *MAX17040) SetRCOMP(rcomp uint16) error {
	return m.writeRegister16(RCOMP_REG, rcomp)
}

// QuickStart issues a Quick-Start command to reinitialize the fuel gauge
func (m *MAX17040) QuickStart() error {
	return m.writeRegister16(MODE_REG, QUICK_START_CMD)
}

// Reset issues a Power-On Reset command
// Note: The IC does not send ACK after this command, so we ignore the error
func (m *MAX17040) Reset() error {
	// The device will reset and not send ACK, so we expect this to "fail"
	m.writeRegister16(COMMAND_REG, POR_CMD)
	return nil
}

// ReadUPSData reads comprehensive UPS data
func (m *MAX17040) ReadUPSData() (*MAX17040Data, error) {
	voltage, err := m.GetCellVoltage()
	if err != nil {
		return nil, fmt.Errorf("failed to read voltage: %w", err)
	}

	soc, err := m.GetStateOfCharge()
	if err != nil {
		return nil, fmt.Errorf("failed to read state of charge: %w", err)
	}

	version, err := m.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	// Determine charging status based on voltage
	// Typical Li-Ion charging voltage is around 4.0V+
	// This is a simplified heuristic - more sophisticated detection could be added
	isCharging := voltage > 4.0

	// Determine if AC power is connected
	// When on AC power, voltage is typically higher due to charging circuit
	isConnected := voltage > 3.9

	return &MAX17040Data{
		Voltage:       voltage,
		StateOfCharge: soc,
		IsCharging:    isCharging,
		IsConnected:   isConnected,
		LastUpdated:   time.Now(),
		ICVersion:     version,
	}, nil
}

// DetectUPS attempts to detect if a UPS with MAX17040 is present
func DetectUPS() bool {
	client, err := NewMAX17040()
	if err != nil {
		return false
	}

	// Try to read basic data to verify it's working
	_, err = client.GetVersion()
	return err == nil
}

// InitializeWithOptimalRCOMP initializes the MAX17040 with tuning
// This function can be extended to load optimal RCOMP values from config
func (m *MAX17040) InitializeWithOptimalRCOMP(rcomp uint16) error {
	// Set the RCOMP value if different from default
	if rcomp != DEFAULT_RCOMP {
		if err := m.SetRCOMP(rcomp); err != nil {
			return fmt.Errorf("failed to set RCOMP: %w", err)
		}
	}

	// Optional: Issue Quick-Start to reinitialize with new settings
	if err := m.QuickStart(); err != nil {
		return fmt.Errorf("failed to quick start: %w", err)
	}

	// Allow time for the IC to complete initialization
	time.Sleep(500 * time.Millisecond)

	return nil
}

// ConvertToUPSData converts MAX17040Data to the existing UPSData format
func (data *MAX17040Data) ConvertToUPSData(upsName string) *UPSData {
	var batteryRuntime *float64
	if data.StateOfCharge > 0 {
		// Estimate runtime based on SOC (rough calculation)
		// Assume ~4 hours at 100% charge under normal load
		estimatedRuntime := (data.StateOfCharge / 100.0) * 14400 // seconds
		batteryRuntime = &estimatedRuntime
	}

	// Determine status based on power state
	status := "OL" // Online
	if !data.IsConnected {
		status = "OB" // On Battery
	}
	if data.StateOfCharge < 20 {
		status += " LB" // Low Battery
	}

	return &UPSData{
		Name:            upsName,
		Description:     "Geekworm X1202 UPS with MAX17040 Fuel Gauge",
		Status:          status,
		BatteryCharge:   &data.StateOfCharge,
		BatteryRuntime:  batteryRuntime,
		BatteryVoltage:  &data.Voltage,
		InputVoltage:    nil, // Not available from MAX17040
		OutputVoltage:   nil, // Not available from MAX17040
		LoadPercent:     nil, // Not available from MAX17040
		Power:           nil, // Not available from MAX17040
		Frequency:       nil, // Not available from MAX17040
		Temperature:     nil, // Not available from MAX17040
		Model:           "X1202",
		Manufacturer:    "Geekworm",
		Serial:          fmt.Sprintf("MAX17040-v%04X", data.ICVersion),
		FirmwareVersion: fmt.Sprintf("v%04X", data.ICVersion),
		Variables: map[string]string{
			"battery.charge":  fmt.Sprintf("%.2f", data.StateOfCharge),
			"battery.voltage": fmt.Sprintf("%.2f", data.Voltage),
			"ups.status":      status,
			"ups.model":       "X1202",
			"ups.mfr":         "Geekworm",
			"device.type":     "ups",
			"driver.name":     "max17040",
		},
		LastUpdated: data.LastUpdated,
	}
}
