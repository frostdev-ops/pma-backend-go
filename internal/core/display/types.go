package display

import "time"

// DisplayOrientation represents available display orientations
type DisplayOrientation string

const (
	OrientationAuto             DisplayOrientation = "auto"
	OrientationLandscape        DisplayOrientation = "landscape"
	OrientationPortrait         DisplayOrientation = "portrait"
	OrientationLandscapeFlipped DisplayOrientation = "landscape-flipped"
	OrientationPortraitFlipped  DisplayOrientation = "portrait-flipped"
)

// DarkMode represents dark mode settings
type DarkMode string

const (
	DarkModeAuto  DarkMode = "auto"
	DarkModeLight DarkMode = "light"
	DarkModeDark  DarkMode = "dark"
)

// ScreensaverType represents different screensaver types
type ScreensaverType string

const (
	ScreensaverLogo         ScreensaverType = "logo"
	ScreensaverClock        ScreensaverType = "clock"
	ScreensaverSlideshow    ScreensaverType = "slideshow"
	ScreensaverPictureframe ScreensaverType = "pictureframe"
)

// DisplaySettings represents the complete display configuration
type DisplaySettings struct {
	ID                           *int               `json:"id,omitempty" db:"id"`
	Brightness                   int                `json:"brightness" db:"brightness"`
	Timeout                      int                `json:"timeout" db:"timeout"` // seconds, 0 = never
	Orientation                  DisplayOrientation `json:"orientation" db:"orientation"`
	DarkMode                     DarkMode           `json:"darkMode" db:"darkMode"`
	Screensaver                  bool               `json:"screensaver" db:"screensaver"`
	ScreensaverType              *ScreensaverType   `json:"screensaverType,omitempty" db:"screensaverType"`
	ScreensaverShowClock         *bool              `json:"screensaverShowClock,omitempty" db:"screensaverShowClock"`
	ScreensaverRotationSpeed     *int               `json:"screensaverRotationSpeed,omitempty" db:"screensaverRotationSpeed"`
	ScreensaverPictureFrameImage *string            `json:"screensaverPictureFrameImage,omitempty" db:"screensaverPictureFrameImage"`
	ScreensaverUploadEnabled     *bool              `json:"screensaverUploadEnabled,omitempty" db:"screensaverUploadEnabled"`
	DimBeforeSleep               *bool              `json:"dimBeforeSleep,omitempty" db:"dimBeforeSleep"`
	DimLevel                     *int               `json:"dimLevel,omitempty" db:"dimLevel"`
	DimTimeout                   *int               `json:"dimTimeout,omitempty" db:"dimTimeout"`
	CreatedAt                    *time.Time         `json:"createdAt,omitempty" db:"created_at"`
	UpdatedAt                    *time.Time         `json:"updatedAt,omitempty" db:"updated_at"`
}

// DisplaySettingsRequest represents a partial update request
type DisplaySettingsRequest struct {
	Brightness                   *int                `json:"brightness,omitempty"`
	Timeout                      *int                `json:"timeout,omitempty"`
	Orientation                  *DisplayOrientation `json:"orientation,omitempty"`
	DarkMode                     *DarkMode           `json:"darkMode,omitempty"`
	Screensaver                  *bool               `json:"screensaver,omitempty"`
	ScreensaverType              *ScreensaverType    `json:"screensaverType,omitempty"`
	ScreensaverShowClock         *bool               `json:"screensaverShowClock,omitempty"`
	ScreensaverRotationSpeed     *int                `json:"screensaverRotationSpeed,omitempty"`
	ScreensaverPictureFrameImage *string             `json:"screensaverPictureFrameImage,omitempty"`
	ScreensaverUploadEnabled     *bool               `json:"screensaverUploadEnabled,omitempty"`
	DimBeforeSleep               *bool               `json:"dimBeforeSleep,omitempty"`
	DimLevel                     *int                `json:"dimLevel,omitempty"`
	DimTimeout                   *int                `json:"dimTimeout,omitempty"`
}

// BrightnessCapability represents brightness control capabilities
type BrightnessCapability struct {
	Supported bool `json:"supported"`
	Min       int  `json:"min"`
	Max       int  `json:"max"`
	Current   int  `json:"current"`
}

// OrientationCapability represents orientation control capabilities
type OrientationCapability struct {
	Supported bool                 `json:"supported"`
	Available []DisplayOrientation `json:"available"`
	Current   DisplayOrientation   `json:"current"`
	Hardware  bool                 `json:"hardware"`  // true if hardware orientation is available
	Software  bool                 `json:"software"`  // true if software orientation is available
	Detection bool                 `json:"detection"` // true if auto orientation detection is available
}

// ScreensaverCapability represents screensaver capabilities
type ScreensaverCapability struct {
	Supported bool `json:"supported"`
	Enabled   bool `json:"enabled"`
}

// DisplayCapabilities represents the hardware capabilities
type DisplayCapabilities struct {
	Brightness  BrightnessCapability  `json:"brightness"`
	Orientation OrientationCapability `json:"orientation"`
	Screensaver ScreensaverCapability `json:"screensaver"`
}

// HardwareDetectionData represents hardware detection result
type HardwareDetectionData struct {
	Brightness *struct {
		Supported         bool `json:"supported"`
		MaxBrightness     int  `json:"max_brightness"`
		CurrentBrightness int  `json:"current_brightness"`
	} `json:"brightness,omitempty"`
	Display *struct {
		Supported bool `json:"supported"`
	} `json:"display,omitempty"`
}

// WakeScreenRequest represents a wake screen request
type WakeScreenRequest struct {
	Duration *int `json:"duration,omitempty"` // Duration in seconds to keep screen awake
}

// HardwareInfo represents detailed hardware information
type HardwareInfo struct {
	DisplayConnected      bool                 `json:"displayConnected"`
	BacklightDevices      []string             `json:"backlightDevices"`
	XrandrAvailable       bool                 `json:"xrandrAvailable"`
	XsetAvailable         bool                 `json:"xsetAvailable"`
	DisplayVariable       string               `json:"displayVariable"`
	AvailableOrientations []DisplayOrientation `json:"availableOrientations"`
}
