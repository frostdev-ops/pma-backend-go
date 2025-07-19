package display

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service manages display settings and hardware control
type Service struct {
	repo   repositories.DisplayRepository
	db     *sql.DB
	logger *logrus.Logger

	// Caching
	capabilitiesCache     *DisplayCapabilities
	capabilitiesCacheTime time.Time
	capabilitiesCacheTTL  time.Duration

	// Current settings cache
	currentSettings *DisplaySettings
	settingsMutex   sync.RWMutex

	// Hardware state
	hardwareDataCache     *HardwareDetectionData
	hardwareDataCacheTime time.Time
	hardwareDataCacheTTL  time.Duration

	initialized bool
	initMutex   sync.Mutex
}

// NewService creates a new display settings service
func NewService(repo repositories.DisplayRepository, db *sql.DB, logger *logrus.Logger) *Service {
	return &Service{
		repo:                 repo,
		db:                   db,
		logger:               logger,
		capabilitiesCacheTTL: 5 * time.Minute,
		hardwareDataCacheTTL: 2 * time.Minute,
	}
}

// Initialize sets up the display settings service
func (s *Service) Initialize(ctx context.Context) error {
	s.initMutex.Lock()
	defer s.initMutex.Unlock()

	if s.initialized {
		return nil
	}

	s.logger.Info("Initializing DisplaySettingsService...")

	// Create display settings table
	if err := s.createTable(ctx); err != nil {
		return fmt.Errorf("failed to create display settings table: %w", err)
	}

	// Load current settings
	if err := s.loadSettings(ctx); err != nil {
		s.logger.WithError(err).Warn("Failed to load display settings, using defaults")
		s.currentSettings = s.getDefaultSettings()
	}

	s.initialized = true
	s.logger.Info("DisplaySettingsService initialized successfully")

	return nil
}

// createTable creates the display_settings table
func (s *Service) createTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS display_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			brightness INTEGER NOT NULL DEFAULT 80,
			timeout INTEGER NOT NULL DEFAULT 300,
			orientation TEXT NOT NULL DEFAULT 'landscape',
			darkMode TEXT NOT NULL DEFAULT 'light',
			screensaver BOOLEAN NOT NULL DEFAULT 1,
			screensaverType TEXT NOT NULL DEFAULT 'logo',
			screensaverShowClock BOOLEAN NOT NULL DEFAULT 1,
			screensaverRotationSpeed INTEGER NOT NULL DEFAULT 5,
			screensaverPictureFrameImage TEXT NOT NULL DEFAULT '',
			screensaverUploadEnabled BOOLEAN NOT NULL DEFAULT 1,
			dimBeforeSleep BOOLEAN NOT NULL DEFAULT 1,
			dimLevel INTEGER NOT NULL DEFAULT 30,
			dimTimeout INTEGER NOT NULL DEFAULT 180,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// getDefaultSettings returns the default display settings
func (s *Service) getDefaultSettings() *DisplaySettings {
	screensaverType := ScreensaverLogo
	screensaverShowClock := true
	screensaverRotationSpeed := 5
	screensaverPictureFrameImage := ""
	screensaverUploadEnabled := true
	dimBeforeSleep := true
	dimLevel := 30
	dimTimeout := 180

	return &DisplaySettings{
		Brightness:                   80,
		Timeout:                      300, // 5 minutes
		Orientation:                  OrientationLandscape,
		DarkMode:                     DarkModeLight,
		Screensaver:                  true,
		ScreensaverType:              &screensaverType,
		ScreensaverShowClock:         &screensaverShowClock,
		ScreensaverRotationSpeed:     &screensaverRotationSpeed,
		ScreensaverPictureFrameImage: &screensaverPictureFrameImage,
		ScreensaverUploadEnabled:     &screensaverUploadEnabled,
		DimBeforeSleep:               &dimBeforeSleep,
		DimLevel:                     &dimLevel,
		DimTimeout:                   &dimTimeout,
	}
}

// loadSettings loads current settings from database using repository
func (s *Service) loadSettings(ctx context.Context) error {
	dbSettings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to load settings from repository: %w", err)
	}

	// Convert from models.DisplaySettings to display.DisplaySettings
	settings := s.convertFromModelSettings(dbSettings)

	s.settingsMutex.Lock()
	s.currentSettings = settings
	s.settingsMutex.Unlock()

	return nil
}

// GetSettings returns the current display settings
func (s *Service) GetSettings(ctx context.Context) (*DisplaySettings, error) {
	if err := s.Initialize(ctx); err != nil {
		return nil, err
	}

	s.settingsMutex.RLock()
	defer s.settingsMutex.RUnlock()

	return s.currentSettings, nil
}

// UpdateSettings updates display settings
func (s *Service) UpdateSettings(ctx context.Context, request *DisplaySettingsRequest) (*DisplaySettings, error) {
	if err := s.Initialize(ctx); err != nil {
		return nil, err
	}

	s.settingsMutex.Lock()
	defer s.settingsMutex.Unlock()

	// Start with current settings
	settings := *s.currentSettings

	// Apply updates
	if request.Brightness != nil {
		settings.Brightness = *request.Brightness
	}
	if request.Timeout != nil {
		settings.Timeout = *request.Timeout
	}
	if request.Orientation != nil {
		settings.Orientation = *request.Orientation
	}
	if request.DarkMode != nil {
		settings.DarkMode = *request.DarkMode
	}
	if request.Screensaver != nil {
		settings.Screensaver = *request.Screensaver
	}
	if request.ScreensaverType != nil {
		settings.ScreensaverType = request.ScreensaverType
	}
	if request.ScreensaverShowClock != nil {
		settings.ScreensaverShowClock = request.ScreensaverShowClock
	}
	if request.ScreensaverRotationSpeed != nil {
		settings.ScreensaverRotationSpeed = request.ScreensaverRotationSpeed
	}
	if request.ScreensaverPictureFrameImage != nil {
		settings.ScreensaverPictureFrameImage = request.ScreensaverPictureFrameImage
	}
	if request.ScreensaverUploadEnabled != nil {
		settings.ScreensaverUploadEnabled = request.ScreensaverUploadEnabled
	}
	if request.DimBeforeSleep != nil {
		settings.DimBeforeSleep = request.DimBeforeSleep
	}
	if request.DimLevel != nil {
		settings.DimLevel = request.DimLevel
	}
	if request.DimTimeout != nil {
		settings.DimTimeout = request.DimTimeout
	}

	// Save to database
	if err := s.saveSettings(ctx, &settings); err != nil {
		return nil, fmt.Errorf("failed to save settings: %w", err)
	}

	// Apply hardware settings in background
	go func() {
		if err := s.applyHardwareSettings(&settings); err != nil {
			s.logger.WithError(err).Error("Failed to apply hardware settings")
		}
	}()

	// Update current settings
	s.currentSettings = &settings

	return &settings, nil
}

// saveSettings saves settings to database using repository
func (s *Service) saveSettings(ctx context.Context, settings *DisplaySettings) error {
	// Convert to models.DisplaySettings
	dbSettings := s.convertToModelSettings(settings)
	if dbSettings == nil {
		return fmt.Errorf("failed to convert settings for database storage")
	}

	// Save using repository
	err := s.repo.UpdateSettings(ctx, dbSettings)
	if err != nil {
		return fmt.Errorf("failed to save settings via repository: %w", err)
	}

	return nil
}

// GetCapabilities returns display hardware capabilities
func (s *Service) GetCapabilities(ctx context.Context) (*DisplayCapabilities, error) {
	if err := s.Initialize(ctx); err != nil {
		return nil, err
	}

	// Check cache first
	if s.capabilitiesCache != nil && time.Since(s.capabilitiesCacheTime) < s.capabilitiesCacheTTL {
		return s.capabilitiesCache, nil
	}

	s.logger.Info("Checking hardware display capabilities...")

	// Get hardware detection data
	hardwareData, err := s.getHardwareDetectionData(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Hardware detection failed, using fallback capabilities")
		return s.getFallbackCapabilities(), nil
	}

	// Convert to capabilities
	capabilities := &DisplayCapabilities{
		Brightness: BrightnessCapability{
			Supported: false,
			Min:       10,
			Max:       100,
			Current:   80,
		},
		Orientation: OrientationCapability{
			Supported: false,
			Available: []DisplayOrientation{OrientationAuto, OrientationLandscape, OrientationPortrait, OrientationLandscapeFlipped, OrientationPortraitFlipped},
			Current:   OrientationLandscape,
			Hardware:  false,
			Software:  true,
			Detection: false,
		},
		Screensaver: ScreensaverCapability{
			Supported: false,
			Enabled:   false,
		},
	}

	// Update brightness capabilities
	if hardwareData != nil && hardwareData.Brightness != nil {
		capabilities.Brightness.Supported = hardwareData.Brightness.Supported
		capabilities.Brightness.Max = hardwareData.Brightness.MaxBrightness
		capabilities.Brightness.Current = hardwareData.Brightness.CurrentBrightness
	}

	// Update display capabilities
	if hardwareData != nil && hardwareData.Display != nil {
		capabilities.Orientation.Supported = hardwareData.Display.Supported
		capabilities.Screensaver.Supported = hardwareData.Display.Supported
	}

	// Cache the result
	s.capabilitiesCache = capabilities
	s.capabilitiesCacheTime = time.Now()

	s.logger.WithField("capabilities", capabilities).Info("Hardware capabilities detected")

	return capabilities, nil
}

// getFallbackCapabilities returns basic fallback capabilities
func (s *Service) getFallbackCapabilities() *DisplayCapabilities {
	return &DisplayCapabilities{
		Brightness: BrightnessCapability{
			Supported: false,
			Min:       10,
			Max:       100,
			Current:   80,
		},
		Orientation: OrientationCapability{
			Supported: true,
			Available: []DisplayOrientation{OrientationLandscape, OrientationPortrait},
			Current:   OrientationLandscape,
			Hardware:  false,
			Software:  true,
			Detection: false,
		},
		Screensaver: ScreensaverCapability{
			Supported: false,
			Enabled:   false,
		},
	}
}

// getHardwareDetectionData gets current hardware capabilities
func (s *Service) getHardwareDetectionData(ctx context.Context) (*HardwareDetectionData, error) {
	// Check cache first
	if s.hardwareDataCache != nil && time.Since(s.hardwareDataCacheTime) < s.hardwareDataCacheTTL {
		return s.hardwareDataCache, nil
	}

	data := &HardwareDetectionData{}

	// Check brightness capability
	brightnessData, err := s.getBrightnessCapability(ctx)
	if err == nil && brightnessData.Supported {
		data.Brightness = &struct {
			Supported         bool `json:"supported"`
			MaxBrightness     int  `json:"max_brightness"`
			CurrentBrightness int  `json:"current_brightness"`
		}{
			Supported:         true,
			MaxBrightness:     brightnessData.Max,
			CurrentBrightness: brightnessData.Current,
		}
	}

	// Check display availability
	displayAvailable := s.isDisplayAvailable()
	if displayAvailable {
		data.Display = &struct {
			Supported bool `json:"supported"`
		}{
			Supported: true,
		}
	}

	// Cache the result
	s.hardwareDataCache = data
	s.hardwareDataCacheTime = time.Now()

	return data, nil
}

// isDisplayAvailable checks if display is available
func (s *Service) isDisplayAvailable() bool {
	display := os.Getenv("DISPLAY")
	return display != "" && strings.TrimSpace(display) != ""
}

// execWithTimeout executes a command with timeout
func (s *Service) execWithTimeout(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.Output()

	return string(output), err
}

// getBrightnessCapability checks brightness control capability
func (s *Service) getBrightnessCapability(ctx context.Context) (*BrightnessCapability, error) {
	capability := &BrightnessCapability{
		Supported: false,
		Min:       10,
		Max:       100,
		Current:   80,
	}

	if !s.isDisplayAvailable() {
		return capability, fmt.Errorf("no display available")
	}

	// Check for Raspberry Pi touchscreen backlight first
	brightnessOutput, err1 := s.execWithTimeout("cat /sys/class/backlight/rpi_backlight/brightness", 1*time.Second)
	maxBrightnessOutput, err2 := s.execWithTimeout("cat /sys/class/backlight/rpi_backlight/max_brightness", 1*time.Second)

	if err1 == nil && err2 == nil {
		if maxBrightness, err := strconv.Atoi(strings.TrimSpace(maxBrightnessOutput)); err == nil && maxBrightness > 0 {
			if currentBrightness, err := strconv.Atoi(strings.TrimSpace(brightnessOutput)); err == nil {
				capability.Supported = true
				capability.Current = int(math.Round(float64(currentBrightness) / float64(maxBrightness) * 100))
				s.logger.WithFields(logrus.Fields{
					"current": capability.Current,
					"max":     maxBrightness,
				}).Info("Raspberry Pi touchscreen detected")
				return capability, nil
			}
		}
	}

	// Check for other backlight devices
	backlightOutput, err := s.execWithTimeout("ls /sys/class/backlight/", 1*time.Second)
	if err == nil && strings.TrimSpace(backlightOutput) != "" {
		backlights := strings.Fields(strings.TrimSpace(backlightOutput))
		for _, backlight := range backlights {
			brightnessPath := fmt.Sprintf("/sys/class/backlight/%s/brightness", backlight)
			maxBrightnessPath := fmt.Sprintf("/sys/class/backlight/%s/max_brightness", backlight)

			brightnessOutput, err1 := s.execWithTimeout(fmt.Sprintf("cat %s", brightnessPath), 1*time.Second)
			maxBrightnessOutput, err2 := s.execWithTimeout(fmt.Sprintf("cat %s", maxBrightnessPath), 1*time.Second)

			if err1 == nil && err2 == nil {
				if maxBrightness, err := strconv.Atoi(strings.TrimSpace(maxBrightnessOutput)); err == nil && maxBrightness > 0 {
					if currentBrightness, err := strconv.Atoi(strings.TrimSpace(brightnessOutput)); err == nil {
						capability.Supported = true
						capability.Current = int(math.Round(float64(currentBrightness) / float64(maxBrightness) * 100))
						s.logger.WithFields(logrus.Fields{
							"device":  backlight,
							"current": capability.Current,
						}).Info("Backlight device detected")
						return capability, nil
					}
				}
			}
		}
	}

	// Check xrandr availability as fallback
	if _, err := s.execWithTimeout("which xrandr", 1*time.Second); err == nil {
		capability.Supported = true
		s.logger.Info("xrandr brightness control available")
		return capability, nil
	}

	return capability, fmt.Errorf("no brightness control available")
}

// applyHardwareSettings applies settings to hardware
func (s *Service) applyHardwareSettings(settings *DisplaySettings) error {
	s.logger.Info("Applying hardware settings...")

	// Apply settings concurrently
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		if err := s.setBrightness(settings.Brightness); err != nil {
			s.logger.WithError(err).Warn("Failed to set brightness")
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.setScreenTimeout(settings.Timeout); err != nil {
			s.logger.WithError(err).Warn("Failed to set screen timeout")
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.setOrientation(string(settings.Orientation)); err != nil {
			s.logger.WithError(err).Warn("Failed to set orientation")
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.setScreensaver(settings.Screensaver); err != nil {
			s.logger.WithError(err).Warn("Failed to set screensaver")
		}
	}()

	wg.Wait()
	s.logger.Info("Hardware settings applied")

	return nil
}

// setBrightness sets display brightness
func (s *Service) setBrightness(brightness int) error {
	s.logger.WithField("brightness", brightness).Info("Setting brightness")

	// Ensure brightness is within valid range
	brightness = int(math.Max(10, math.Min(100, float64(brightness))))

	if !s.isDisplayAvailable() {
		s.logger.Info("No DISPLAY environment variable set - running in headless mode")
		return nil
	}

	// Try Raspberry Pi backlight first
	backlightOutput, err := s.execWithTimeout("ls /sys/class/backlight/", 1*time.Second)
	if err == nil && strings.TrimSpace(backlightOutput) != "" {
		backlights := strings.Fields(strings.TrimSpace(backlightOutput))
		for _, backlight := range backlights {
			maxBrightnessPath := fmt.Sprintf("/sys/class/backlight/%s/max_brightness", backlight)
			brightnessPath := fmt.Sprintf("/sys/class/backlight/%s/brightness", backlight)

			maxBrightnessOutput, err := s.execWithTimeout(fmt.Sprintf("cat %s", maxBrightnessPath), 1*time.Second)
			if err == nil {
				if maxBrightness, err := strconv.Atoi(strings.TrimSpace(maxBrightnessOutput)); err == nil && maxBrightness > 0 {
					targetBrightness := int(math.Round(float64(brightness) / 100.0 * float64(maxBrightness)))

					// Try to write brightness
					cmd := fmt.Sprintf("echo %d | sudo tee %s", targetBrightness, brightnessPath)
					if _, err := s.execWithTimeout(cmd, 2*time.Second); err == nil {
						s.logger.WithFields(logrus.Fields{
							"device":     backlight,
							"brightness": brightness,
							"target":     targetBrightness,
						}).Info("Set brightness via backlight")
						return nil
					}
				}
			}
		}
	}

	// Try xrandr as fallback
	if _, err := s.execWithTimeout("which xrandr", 1*time.Second); err == nil {
		displayOutput, err := s.execWithTimeout("xrandr --query | grep \" connected\" | head -1", 2*time.Second)
		if err == nil && displayOutput != "" {
			re := regexp.MustCompile(`^(\S+)\s+connected`)
			matches := re.FindStringSubmatch(displayOutput)
			if len(matches) > 1 {
				displayName := matches[1]
				brightnessDecimal := float64(brightness) / 100.0

				display := os.Getenv("DISPLAY")
				if display == "" {
					display = ":0"
				}

				cmd := fmt.Sprintf("DISPLAY=%s xrandr --output %s --brightness %.2f", display, displayName, brightnessDecimal)
				if _, err := s.execWithTimeout(cmd, 3*time.Second); err == nil {
					s.logger.WithFields(logrus.Fields{
						"display":    displayName,
						"brightness": brightness,
					}).Info("Set brightness via xrandr")
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no brightness control method available")
}

// setScreenTimeout sets screen timeout
func (s *Service) setScreenTimeout(timeoutSeconds int) error {
	s.logger.WithField("timeout", timeoutSeconds).Info("Setting screen timeout")

	if !s.isDisplayAvailable() {
		s.logger.Info("No DISPLAY environment variable set - cannot set screen timeout in headless mode")
		return nil
	}

	display := os.Getenv("DISPLAY")
	if display == "" {
		display = ":0"
	}

	if timeoutSeconds <= 0 {
		// Disable screen timeout
		_, err1 := s.execWithTimeout(fmt.Sprintf("DISPLAY=%s xset s off", display), 2*time.Second)
		_, err2 := s.execWithTimeout(fmt.Sprintf("DISPLAY=%s xset -dpms", display), 2*time.Second)

		if err1 != nil || err2 != nil {
			return fmt.Errorf("failed to disable screen timeout")
		}
		s.logger.Info("Screen timeout disabled")
	} else {
		// Set screen timeout
		_, err1 := s.execWithTimeout(fmt.Sprintf("DISPLAY=%s xset s %d", display, timeoutSeconds), 2*time.Second)
		_, err2 := s.execWithTimeout(fmt.Sprintf("DISPLAY=%s xset dpms %d %d %d", display, timeoutSeconds, timeoutSeconds, timeoutSeconds), 2*time.Second)

		if err1 != nil || err2 != nil {
			return fmt.Errorf("failed to set screen timeout")
		}
		s.logger.WithField("timeout", timeoutSeconds).Info("Screen timeout set")
	}

	return nil
}

// setOrientation sets display orientation
func (s *Service) setOrientation(orientation string) error {
	s.logger.WithField("orientation", orientation).Info("Setting orientation")

	if !s.isDisplayAvailable() {
		s.logger.Info("No DISPLAY environment variable set - cannot set orientation in headless mode")
		return nil
	}

	// Skip hardware orientation for auto mode
	if orientation == "auto" {
		s.logger.Info("Auto orientation mode - skipping hardware orientation setting")
		return nil
	}

	var xrandrOrientation string
	switch orientation {
	case "portrait":
		xrandrOrientation = "left"
	case "portrait-flipped":
		xrandrOrientation = "right"
	case "landscape-flipped":
		xrandrOrientation = "inverted"
	case "landscape":
		fallthrough
	default:
		xrandrOrientation = "normal"
	}

	// Get display name
	displayOutput, err := s.execWithTimeout("xrandr --query | grep \" connected\" | head -1", 2*time.Second)
	if err != nil || displayOutput == "" {
		return fmt.Errorf("failed to get display information")
	}

	re := regexp.MustCompile(`^(\S+)\s+connected`)
	matches := re.FindStringSubmatch(displayOutput)
	if len(matches) <= 1 {
		return fmt.Errorf("failed to parse display name")
	}

	displayName := matches[1]
	display := os.Getenv("DISPLAY")
	if display == "" {
		display = ":0"
	}

	// Rotate the display
	cmd := fmt.Sprintf("DISPLAY=%s xrandr --output %s --rotate %s", display, displayName, xrandrOrientation)
	if _, err := s.execWithTimeout(cmd, 3*time.Second); err != nil {
		return fmt.Errorf("failed to rotate display: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"display":     displayName,
		"orientation": orientation,
		"xrandr":      xrandrOrientation,
	}).Info("Set hardware orientation")

	return nil
}

// setScreensaver enables/disables screensaver
func (s *Service) setScreensaver(enabled bool) error {
	s.logger.WithField("enabled", enabled).Info("Setting screensaver")

	if !s.isDisplayAvailable() {
		s.logger.Info("No DISPLAY environment variable set - cannot control screensaver in headless mode")
		return nil
	}

	display := os.Getenv("DISPLAY")
	if display == "" {
		display = ":0"
	}

	var cmd string
	if enabled {
		cmd = fmt.Sprintf("DISPLAY=%s xset s on", display)
	} else {
		cmd = fmt.Sprintf("DISPLAY=%s xset s off", display)
	}

	if _, err := s.execWithTimeout(cmd, 2*time.Second); err != nil {
		return fmt.Errorf("failed to set screensaver: %w", err)
	}

	if enabled {
		s.logger.Info("Screensaver enabled")
	} else {
		s.logger.Info("Screensaver disabled")
	}

	return nil
}

// WakeScreen wakes up the display
func (s *Service) WakeScreen(ctx context.Context, request *WakeScreenRequest) error {
	s.logger.Info("Waking screen")

	if !s.isDisplayAvailable() {
		s.logger.Info("No DISPLAY environment variable set - cannot wake screen in headless mode")
		return nil
	}

	display := os.Getenv("DISPLAY")
	if display == "" {
		display = ":0"
	}

	// Wake screen by moving mouse slightly
	cmd := fmt.Sprintf("DISPLAY=%s xdotool mousemove_relative 1 0 && DISPLAY=%s xdotool mousemove_relative -- -1 0", display, display)
	if _, err := s.execWithTimeout(cmd, 2*time.Second); err != nil {
		// Fallback: try xset screen activation
		cmd = fmt.Sprintf("DISPLAY=%s xset s activate", display)
		if _, err := s.execWithTimeout(cmd, 2*time.Second); err != nil {
			return fmt.Errorf("failed to wake screen: %w", err)
		}
	}

	s.logger.Info("Screen woken")
	return nil
}

// GetHardwareInfo returns detailed hardware information
func (s *Service) GetHardwareInfo(ctx context.Context) (*HardwareInfo, error) {
	info := &HardwareInfo{
		DisplayConnected:      s.isDisplayAvailable(),
		BacklightDevices:      []string{},
		XrandrAvailable:       false,
		XsetAvailable:         false,
		DisplayVariable:       os.Getenv("DISPLAY"),
		AvailableOrientations: []DisplayOrientation{OrientationLandscape, OrientationPortrait},
	}

	// Check backlight devices
	if backlightOutput, err := s.execWithTimeout("ls /sys/class/backlight/", 1*time.Second); err == nil {
		devices := strings.Fields(strings.TrimSpace(backlightOutput))
		info.BacklightDevices = devices
	}

	// Check xrandr availability
	if _, err := s.execWithTimeout("which xrandr", 1*time.Second); err == nil {
		info.XrandrAvailable = true
		info.AvailableOrientations = []DisplayOrientation{
			OrientationAuto, OrientationLandscape, OrientationPortrait,
			OrientationLandscapeFlipped, OrientationPortraitFlipped,
		}
	}

	// Check xset availability
	if _, err := s.execWithTimeout("which xset", 1*time.Second); err == nil {
		info.XsetAvailable = true
	}

	return info, nil
}

// convertFromModelSettings converts models.DisplaySettings to display.DisplaySettings
func (s *Service) convertFromModelSettings(dbSettings *models.DisplaySettings) *DisplaySettings {
	if dbSettings == nil {
		return s.getDefaultSettings()
	}

	settings := &DisplaySettings{
		ID:          &dbSettings.ID,
		Brightness:  dbSettings.Brightness,
		Timeout:     dbSettings.Timeout,
		Orientation: DisplayOrientation(dbSettings.Orientation),
		DarkMode:    DarkMode(dbSettings.DarkMode),
		Screensaver: dbSettings.Screensaver,
		CreatedAt:   &dbSettings.CreatedAt,
		UpdatedAt:   &dbSettings.UpdatedAt,
	}

	// Handle optional fields with pointers
	if dbSettings.ScreensaverType != "" {
		screensaverType := ScreensaverType(dbSettings.ScreensaverType)
		settings.ScreensaverType = &screensaverType
	}

	settings.ScreensaverShowClock = &dbSettings.ScreensaverShowClock
	settings.ScreensaverRotationSpeed = &dbSettings.ScreensaverRotationSpeed

	if dbSettings.ScreensaverPictureFrameImage != "" {
		settings.ScreensaverPictureFrameImage = &dbSettings.ScreensaverPictureFrameImage
	}

	settings.ScreensaverUploadEnabled = &dbSettings.ScreensaverUploadEnabled
	settings.DimBeforeSleep = &dbSettings.DimBeforeSleep
	settings.DimLevel = &dbSettings.DimLevel
	settings.DimTimeout = &dbSettings.DimTimeout

	return settings
}

// convertToModelSettings converts display.DisplaySettings to models.DisplaySettings
func (s *Service) convertToModelSettings(settings *DisplaySettings) *models.DisplaySettings {
	if settings == nil {
		return nil
	}

	dbSettings := &models.DisplaySettings{
		Brightness:  settings.Brightness,
		Timeout:     settings.Timeout,
		Orientation: string(settings.Orientation),
		DarkMode:    string(settings.DarkMode),
		Screensaver: settings.Screensaver,
	}

	// Set ID if available
	if settings.ID != nil {
		dbSettings.ID = *settings.ID
	} else {
		dbSettings.ID = 1 // Default ID for singleton settings
	}

	// Handle optional fields
	if settings.ScreensaverType != nil {
		dbSettings.ScreensaverType = string(*settings.ScreensaverType)
	} else {
		dbSettings.ScreensaverType = "clock" // default
	}

	if settings.ScreensaverShowClock != nil {
		dbSettings.ScreensaverShowClock = *settings.ScreensaverShowClock
	} else {
		dbSettings.ScreensaverShowClock = true // default
	}

	if settings.ScreensaverRotationSpeed != nil {
		dbSettings.ScreensaverRotationSpeed = *settings.ScreensaverRotationSpeed
	} else {
		dbSettings.ScreensaverRotationSpeed = 5 // default
	}

	if settings.ScreensaverPictureFrameImage != nil {
		dbSettings.ScreensaverPictureFrameImage = *settings.ScreensaverPictureFrameImage
	}

	if settings.ScreensaverUploadEnabled != nil {
		dbSettings.ScreensaverUploadEnabled = *settings.ScreensaverUploadEnabled
	} else {
		dbSettings.ScreensaverUploadEnabled = true // default
	}

	if settings.DimBeforeSleep != nil {
		dbSettings.DimBeforeSleep = *settings.DimBeforeSleep
	} else {
		dbSettings.DimBeforeSleep = true // default
	}

	if settings.DimLevel != nil {
		dbSettings.DimLevel = *settings.DimLevel
	} else {
		dbSettings.DimLevel = 30 // default
	}

	if settings.DimTimeout != nil {
		dbSettings.DimTimeout = *settings.DimTimeout
	} else {
		dbSettings.DimTimeout = 60 // default
	}

	// Set timestamps
	if settings.CreatedAt != nil {
		dbSettings.CreatedAt = *settings.CreatedAt
	} else {
		dbSettings.CreatedAt = time.Now()
	}

	if settings.UpdatedAt != nil {
		dbSettings.UpdatedAt = *settings.UpdatedAt
	} else {
		dbSettings.UpdatedAt = time.Now()
	}

	return dbSettings
}
