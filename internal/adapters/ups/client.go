package ups

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultNUTPort     = 3493
	nutTimeout         = 10 * time.Second
	nutProtocolVersion = "2"
)

// NUTClient handles communication with NUT (Network UPS Tools) servers
type NUTClient struct {
	logger *logrus.Logger
	conn   net.Conn
	host   string
	port   int
}

// UPSData represents UPS information from NUT
type UPSData struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Status          string            `json:"status"`
	BatteryCharge   *float64          `json:"battery_charge"`
	BatteryRuntime  *float64          `json:"battery_runtime"`
	BatteryVoltage  *float64          `json:"battery_voltage"`
	InputVoltage    *float64          `json:"input_voltage"`
	OutputVoltage   *float64          `json:"output_voltage"`
	LoadPercent     *float64          `json:"load_percent"`
	Power           *float64          `json:"power"`
	Frequency       *float64          `json:"frequency"`
	Temperature     *float64          `json:"temperature"`
	Model           string            `json:"model"`
	Manufacturer    string            `json:"manufacturer"`
	Serial          string            `json:"serial"`
	FirmwareVersion string            `json:"firmware_version"`
	Variables       map[string]string `json:"variables"`
	LastUpdated     time.Time         `json:"last_updated"`
}

// UPSVariable represents a NUT variable
type UPSVariable struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Writable    bool   `json:"writable"`
}

// NewNUTClient creates a new NUT client
func NewNUTClient(host string, port int, logger *logrus.Logger) *NUTClient {
	if port == 0 {
		port = defaultNUTPort
	}

	return &NUTClient{
		logger: logger,
		host:   host,
		port:   port,
	}
}

// Connect establishes connection to NUT server
func (c *NUTClient) Connect(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", c.host, c.port)
	c.logger.WithField("address", address).Info("Connecting to NUT server...")

	dialer := &net.Dialer{
		Timeout: nutTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to NUT server: %w", err)
	}

	c.conn = conn
	c.logger.Info("Connected to NUT server")

	// Set protocol version
	if err := c.sendCommand("VER"); err != nil {
		c.Close()
		return fmt.Errorf("failed to set protocol version: %w", err)
	}

	return nil
}

// Close closes the connection to NUT server
func (c *NUTClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.logger.Info("Disconnected from NUT server")
		return err
	}
	return nil
}

// ListUPS lists all UPS devices available on the NUT server
func (c *NUTClient) ListUPS(ctx context.Context) ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to NUT server")
	}

	response, err := c.sendCommandWithResponse("LIST UPS")
	if err != nil {
		return nil, err
	}

	var upsList []string
	for _, line := range response {
		if strings.HasPrefix(line, "UPS ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				upsName := strings.Trim(parts[1], "\"")
				upsList = append(upsList, upsName)
			}
		}
	}

	return upsList, nil
}

// GetUPSData retrieves comprehensive UPS data
func (c *NUTClient) GetUPSData(ctx context.Context, upsName string) (*UPSData, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to NUT server")
	}

	variables, err := c.GetUPSVariables(ctx, upsName)
	if err != nil {
		return nil, err
	}

	ups := &UPSData{
		Name:        upsName,
		Variables:   make(map[string]string),
		LastUpdated: time.Now(),
	}

	// Parse common variables
	for name, variable := range variables {
		ups.Variables[name] = variable.Value

		switch name {
		case "ups.status":
			ups.Status = variable.Value
		case "ups.model":
			ups.Model = variable.Value
		case "ups.mfr":
			ups.Manufacturer = variable.Value
		case "ups.serial":
			ups.Serial = variable.Value
		case "ups.firmware":
			ups.FirmwareVersion = variable.Value
		case "ups.description":
			ups.Description = variable.Value
		case "battery.charge":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.BatteryCharge = &val
			}
		case "battery.runtime":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.BatteryRuntime = &val
			}
		case "battery.voltage":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.BatteryVoltage = &val
			}
		case "input.voltage":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.InputVoltage = &val
			}
		case "output.voltage":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.OutputVoltage = &val
			}
		case "ups.load":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.LoadPercent = &val
			}
		case "ups.power":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.Power = &val
			}
		case "input.frequency":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.Frequency = &val
			}
		case "ups.temperature":
			if val, err := strconv.ParseFloat(variable.Value, 64); err == nil {
				ups.Temperature = &val
			}
		}
	}

	return ups, nil
}

// GetUPSVariables retrieves all variables for a UPS
func (c *NUTClient) GetUPSVariables(ctx context.Context, upsName string) (map[string]UPSVariable, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to NUT server")
	}

	command := fmt.Sprintf("LIST VAR %s", upsName)
	response, err := c.sendCommandWithResponse(command)
	if err != nil {
		return nil, err
	}

	variables := make(map[string]UPSVariable)
	for _, line := range response {
		if strings.HasPrefix(line, "VAR ") {
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 4 {
				varName := parts[2]
				varValue := strings.Trim(parts[3], "\"")

				variables[varName] = UPSVariable{
					Name:  varName,
					Value: varValue,
					Type:  "string", // NUT doesn't provide type info in LIST VAR
				}
			}
		}
	}

	return variables, nil
}

// GetUPSVariable retrieves a specific variable value
func (c *NUTClient) GetUPSVariable(ctx context.Context, upsName, varName string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected to NUT server")
	}

	command := fmt.Sprintf("GET VAR %s %s", upsName, varName)
	response, err := c.sendCommandWithResponse(command)
	if err != nil {
		return "", err
	}

	for _, line := range response {
		if strings.HasPrefix(line, "VAR ") {
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 4 {
				return strings.Trim(parts[3], "\""), nil
			}
		}
	}

	return "", fmt.Errorf("variable %s not found for UPS %s", varName, upsName)
}

// SetUPSVariable sets a variable value (for writable variables)
func (c *NUTClient) SetUPSVariable(ctx context.Context, upsName, varName, value string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to NUT server")
	}

	command := fmt.Sprintf("SET VAR %s %s \"%s\"", upsName, varName, value)
	return c.sendCommand(command)
}

// SendInstantCommand sends an instant command to the UPS
func (c *NUTClient) SendInstantCommand(ctx context.Context, upsName, command string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to NUT server")
	}

	cmd := fmt.Sprintf("INSTCMD %s %s", upsName, command)
	return c.sendCommand(cmd)
}

// ListInstantCommands lists available instant commands for a UPS
func (c *NUTClient) ListInstantCommands(ctx context.Context, upsName string) ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to NUT server")
	}

	command := fmt.Sprintf("LIST CMD %s", upsName)
	response, err := c.sendCommandWithResponse(command)
	if err != nil {
		return nil, err
	}

	var commands []string
	for _, line := range response {
		if strings.HasPrefix(line, "CMD ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				commands = append(commands, parts[2])
			}
		}
	}

	return commands, nil
}

// IsConnected checks if the client is connected to NUT server
func (c *NUTClient) IsConnected() bool {
	return c.conn != nil
}

// sendCommand sends a command to NUT server
func (c *NUTClient) sendCommand(command string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(nutTimeout))
	_, err := fmt.Fprintf(c.conn, "%s\n", command)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Read response to check for errors
	c.conn.SetReadDeadline(time.Now().Add(nutTimeout))
	scanner := bufio.NewScanner(c.conn)

	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ERR ") {
			return fmt.Errorf("NUT error: %s", line)
		}
	}

	return scanner.Err()
}

// sendCommandWithResponse sends a command and returns the response lines
func (c *NUTClient) sendCommandWithResponse(command string) ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(nutTimeout))
	_, err := fmt.Fprintf(c.conn, "%s\n", command)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	c.conn.SetReadDeadline(time.Now().Add(nutTimeout))
	scanner := bufio.NewScanner(c.conn)

	var response []string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "ERR ") {
			return nil, fmt.Errorf("NUT error: %s", line)
		}

		if line == "BEGIN LIST" || strings.HasPrefix(line, "BEGIN ") {
			continue // Skip BEGIN markers
		}

		if line == "END LIST" || strings.HasPrefix(line, "END ") {
			break // End of response
		}

		response = append(response, line)
	}

	return response, scanner.Err()
}

// Ping tests the connection to NUT server
func (c *NUTClient) Ping(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.sendCommand("VER")
}

// GetNUTVersion gets the NUT server version
func (c *NUTClient) GetNUTVersion(ctx context.Context) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected")
	}

	response, err := c.sendCommandWithResponse("VER")
	if err != nil {
		return "", err
	}

	for _, line := range response {
		if strings.HasPrefix(line, "Network UPS Tools") {
			return line, nil
		}
	}

	return "Unknown", nil
}
