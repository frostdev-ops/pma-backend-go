package preferences

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// ThemeManager defines the interface for managing themes
type ThemeManager interface {
	GetAvailableThemes() []ThemeDefinition
	GetTheme(themeID string) (*ThemeDefinition, error)
	CreateCustomTheme(userID string, theme ThemeDefinition) error
	ApplyTheme(userID string, themeID string) error
	DeleteCustomTheme(userID string, themeID string) error
	GetUserCustomThemes(userID string) ([]ThemeDefinition, error)
}

// ThemeDefinition represents a complete theme definition
type ThemeDefinition struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	Version      string                 `json:"version"`
	IsCustom     bool                   `json:"is_custom"`
	UserID       string                 `json:"user_id,omitempty" db:"user_id"`
	ColorScheme  string                 `json:"color_scheme"`
	Colors       ThemeColors            `json:"colors"`
	Typography   ThemeTypography        `json:"typography"`
	Spacing      ThemeSpacing           `json:"spacing"`
	BorderRadius ThemeBorderRadius      `json:"border_radius"`
	Shadows      ThemeShadows           `json:"shadows"`
	Components   ThemeComponents        `json:"components"`
	CustomCSS    string                 `json:"custom_css"`
	PreviewImage string                 `json:"preview_image"`
	Tags         []string               `json:"tags"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ThemeColors defines color palette for a theme
type ThemeColors struct {
	Primary      string `json:"primary"`
	Secondary    string `json:"secondary"`
	Accent       string `json:"accent"`
	Background   string `json:"background"`
	Surface      string `json:"surface"`
	Error        string `json:"error"`
	Warning      string `json:"warning"`
	Info         string `json:"info"`
	Success      string `json:"success"`
	OnPrimary    string `json:"on_primary"`
	OnSecondary  string `json:"on_secondary"`
	OnBackground string `json:"on_background"`
	OnSurface    string `json:"on_surface"`
	OnError      string `json:"on_error"`
	Divider      string `json:"divider"`
	Outline      string `json:"outline"`
	Shadow       string `json:"shadow"`
}

// ThemeTypography defines typography settings
type ThemeTypography struct {
	FontFamily    string             `json:"font_family"`
	HeadingFamily string             `json:"heading_family"`
	MonoFamily    string             `json:"mono_family"`
	FontSizeBase  string             `json:"font_size_base"`
	FontWeights   map[string]int     `json:"font_weights"`
	LineHeights   map[string]float64 `json:"line_heights"`
	LetterSpacing map[string]string  `json:"letter_spacing"`
}

// ThemeSpacing defines spacing scale
type ThemeSpacing struct {
	Unit   string            `json:"unit"`
	Scale  map[string]string `json:"scale"`
	Custom map[string]string `json:"custom"`
}

// ThemeBorderRadius defines border radius values
type ThemeBorderRadius struct {
	None   string `json:"none"`
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Large  string `json:"large"`
	XLarge string `json:"xlarge"`
	Full   string `json:"full"`
}

// ThemeShadows defines shadow styles
type ThemeShadows struct {
	None   string `json:"none"`
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Large  string `json:"large"`
	XLarge string `json:"xlarge"`
	Inner  string `json:"inner"`
}

// ThemeComponents defines component-specific styling
type ThemeComponents struct {
	Button  ComponentTheme            `json:"button"`
	Card    ComponentTheme            `json:"card"`
	Input   ComponentTheme            `json:"input"`
	Modal   ComponentTheme            `json:"modal"`
	Sidebar ComponentTheme            `json:"sidebar"`
	Header  ComponentTheme            `json:"header"`
	Widget  ComponentTheme            `json:"widget"`
	Custom  map[string]ComponentTheme `json:"custom"`
}

// ComponentTheme defines styling for a specific component
type ComponentTheme struct {
	Background   string            `json:"background"`
	Border       string            `json:"border"`
	BorderRadius string            `json:"border_radius"`
	Shadow       string            `json:"shadow"`
	Padding      string            `json:"padding"`
	Margin       string            `json:"margin"`
	Typography   map[string]string `json:"typography"`
	States       map[string]string `json:"states"` // hover, active, disabled, etc.
	Custom       map[string]string `json:"custom"`
}

// ThemeManagerImpl implements the ThemeManager interface
type ThemeManagerImpl struct {
	db            *sql.DB
	logger        *logrus.Logger
	prefsManager  PreferencesManager
	builtinThemes []ThemeDefinition
}

// NewThemeManager creates a new theme manager
func NewThemeManager(db *sql.DB, logger *logrus.Logger, prefsManager PreferencesManager) *ThemeManagerImpl {
	tm := &ThemeManagerImpl{
		db:           db,
		logger:       logger,
		prefsManager: prefsManager,
	}

	tm.builtinThemes = tm.getBuiltinThemes()
	return tm
}

// GetAvailableThemes returns all available themes (builtin + custom)
func (tm *ThemeManagerImpl) GetAvailableThemes() []ThemeDefinition {
	themes := make([]ThemeDefinition, len(tm.builtinThemes))
	copy(themes, tm.builtinThemes)

	// Add custom themes from database
	customThemes, err := tm.getAllCustomThemes()
	if err != nil {
		tm.logger.WithError(err).Error("Failed to get custom themes")
	} else {
		themes = append(themes, customThemes...)
	}

	return themes
}

// GetTheme retrieves a specific theme by ID
func (tm *ThemeManagerImpl) GetTheme(themeID string) (*ThemeDefinition, error) {
	// Check builtin themes first
	for _, theme := range tm.builtinThemes {
		if theme.ID == themeID {
			return &theme, nil
		}
	}

	// Check custom themes
	return tm.getCustomTheme(themeID)
}

// CreateCustomTheme creates a new custom theme for a user
func (tm *ThemeManagerImpl) CreateCustomTheme(userID string, theme ThemeDefinition) error {
	theme.ID = fmt.Sprintf("custom_%s_%d", userID, time.Now().Unix())
	theme.IsCustom = true
	theme.UserID = userID
	theme.CreatedAt = time.Now()
	theme.UpdatedAt = time.Now()

	// Validate theme
	if err := tm.validateTheme(&theme); err != nil {
		return fmt.Errorf("theme validation failed: %w", err)
	}

	// Check user's custom theme limit
	userThemes, err := tm.GetUserCustomThemes(userID)
	if err != nil {
		return err
	}

	if len(userThemes) >= 10 { // Max 10 custom themes per user
		return fmt.Errorf("maximum custom themes limit reached")
	}

	themeJSON, err := json.Marshal(theme)
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}

	query := `
		INSERT INTO custom_themes (id, user_id, name, definition, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = tm.db.Exec(query, theme.ID, userID, theme.Name, string(themeJSON), theme.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save custom theme: %w", err)
	}

	tm.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"theme_id": theme.ID,
		"action":   "create_custom_theme",
	}).Info("Custom theme created")

	return nil
}

// ApplyTheme applies a theme to a user's preferences
func (tm *ThemeManagerImpl) ApplyTheme(userID string, themeID string) error {
	theme, err := tm.GetTheme(themeID)
	if err != nil {
		return fmt.Errorf("theme not found: %w", err)
	}

	// Get current preferences
	prefs, err := tm.prefsManager.GetUserPreferences(userID)
	if err != nil {
		return err
	}

	// Apply theme to preferences
	prefs.Theme.ColorScheme = theme.ColorScheme
	prefs.Theme.PrimaryColor = theme.Colors.Primary
	prefs.Theme.AccentColor = theme.Colors.Accent
	prefs.Theme.FontFamily = theme.Typography.FontFamily
	prefs.Theme.CustomCSS = theme.CustomCSS

	// Save updated preferences
	if err := tm.prefsManager.UpdateUserPreferences(userID, prefs); err != nil {
		return err
	}

	tm.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"theme_id": themeID,
		"action":   "apply_theme",
	}).Info("Theme applied to user")

	return nil
}

// DeleteCustomTheme deletes a user's custom theme
func (tm *ThemeManagerImpl) DeleteCustomTheme(userID string, themeID string) error {
	query := `DELETE FROM custom_themes WHERE id = ? AND user_id = ?`

	result, err := tm.db.Exec(query, themeID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete custom theme: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("theme not found or not owned by user")
	}

	tm.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"theme_id": themeID,
		"action":   "delete_custom_theme",
	}).Info("Custom theme deleted")

	return nil
}

// GetUserCustomThemes returns all custom themes for a user
func (tm *ThemeManagerImpl) GetUserCustomThemes(userID string) ([]ThemeDefinition, error) {
	query := `SELECT definition FROM custom_themes WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := tm.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query custom themes: %w", err)
	}
	defer rows.Close()

	var themes []ThemeDefinition
	for rows.Next() {
		var themeJSON string
		if err := rows.Scan(&themeJSON); err != nil {
			tm.logger.WithError(err).Error("Failed to scan theme row")
			continue
		}

		var theme ThemeDefinition
		if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
			tm.logger.WithError(err).Error("Failed to unmarshal theme")
			continue
		}

		themes = append(themes, theme)
	}

	return themes, nil
}

// getBuiltinThemes returns the predefined system themes
func (tm *ThemeManagerImpl) getBuiltinThemes() []ThemeDefinition {
	return []ThemeDefinition{
		{
			ID:          "light",
			Name:        "Light Theme",
			Description: "Clean and bright theme for daytime use",
			Author:      "PMA System",
			Version:     "1.0",
			IsCustom:    false,
			ColorScheme: "light",
			Colors: ThemeColors{
				Primary:      "#1976D2",
				Secondary:    "#424242",
				Accent:       "#FF5722",
				Background:   "#FFFFFF",
				Surface:      "#F5F5F5",
				Error:        "#F44336",
				Warning:      "#FF9800",
				Info:         "#2196F3",
				Success:      "#4CAF50",
				OnPrimary:    "#FFFFFF",
				OnSecondary:  "#FFFFFF",
				OnBackground: "#000000",
				OnSurface:    "#000000",
				OnError:      "#FFFFFF",
				Divider:      "#E0E0E0",
				Outline:      "#757575",
				Shadow:       "#00000020",
			},
			Typography: ThemeTypography{
				FontFamily:    "system-ui, -apple-system, sans-serif",
				HeadingFamily: "system-ui, -apple-system, sans-serif",
				MonoFamily:    "Monaco, 'Cascadia Code', monospace",
				FontSizeBase:  "14px",
				FontWeights: map[string]int{
					"light":  300,
					"normal": 400,
					"medium": 500,
					"bold":   700,
				},
				LineHeights: map[string]float64{
					"tight":  1.25,
					"normal": 1.5,
					"loose":  1.75,
				},
			},
		},
		{
			ID:          "dark",
			Name:        "Dark Theme",
			Description: "Modern dark theme for low-light environments",
			Author:      "PMA System",
			Version:     "1.0",
			IsCustom:    false,
			ColorScheme: "dark",
			Colors: ThemeColors{
				Primary:      "#90CAF9",
				Secondary:    "#B0BEC5",
				Accent:       "#FF7043",
				Background:   "#121212",
				Surface:      "#1E1E1E",
				Error:        "#CF6679",
				Warning:      "#FFB74D",
				Info:         "#81C7E4",
				Success:      "#81C784",
				OnPrimary:    "#000000",
				OnSecondary:  "#000000",
				OnBackground: "#FFFFFF",
				OnSurface:    "#FFFFFF",
				OnError:      "#000000",
				Divider:      "#373737",
				Outline:      "#8A8A8A",
				Shadow:       "#00000040",
			},
			Typography: ThemeTypography{
				FontFamily:    "system-ui, -apple-system, sans-serif",
				HeadingFamily: "system-ui, -apple-system, sans-serif",
				MonoFamily:    "Monaco, 'Cascadia Code', monospace",
				FontSizeBase:  "14px",
				FontWeights: map[string]int{
					"light":  300,
					"normal": 400,
					"medium": 500,
					"bold":   700,
				},
				LineHeights: map[string]float64{
					"tight":  1.25,
					"normal": 1.5,
					"loose":  1.75,
				},
			},
		},
		{
			ID:          "high-contrast",
			Name:        "High Contrast",
			Description: "High contrast theme for accessibility",
			Author:      "PMA System",
			Version:     "1.0",
			IsCustom:    false,
			ColorScheme: "dark",
			Colors: ThemeColors{
				Primary:      "#FFFF00",
				Secondary:    "#FFFFFF",
				Accent:       "#00FFFF",
				Background:   "#000000",
				Surface:      "#1A1A1A",
				Error:        "#FF0000",
				Warning:      "#FFFF00",
				Info:         "#00FFFF",
				Success:      "#00FF00",
				OnPrimary:    "#000000",
				OnSecondary:  "#000000",
				OnBackground: "#FFFFFF",
				OnSurface:    "#FFFFFF",
				OnError:      "#000000",
				Divider:      "#FFFFFF",
				Outline:      "#FFFFFF",
				Shadow:       "#FFFFFF20",
			},
			Typography: ThemeTypography{
				FontFamily:    "system-ui, -apple-system, sans-serif",
				HeadingFamily: "system-ui, -apple-system, sans-serif",
				MonoFamily:    "Monaco, 'Cascadia Code', monospace",
				FontSizeBase:  "16px",
				FontWeights: map[string]int{
					"light":  400,
					"normal": 600,
					"medium": 700,
					"bold":   800,
				},
				LineHeights: map[string]float64{
					"tight":  1.4,
					"normal": 1.6,
					"loose":  1.8,
				},
			},
		},
	}
}

// getAllCustomThemes returns all custom themes from database
func (tm *ThemeManagerImpl) getAllCustomThemes() ([]ThemeDefinition, error) {
	query := `SELECT definition FROM custom_themes ORDER BY created_at DESC`

	rows, err := tm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query custom themes: %w", err)
	}
	defer rows.Close()

	var themes []ThemeDefinition
	for rows.Next() {
		var themeJSON string
		if err := rows.Scan(&themeJSON); err != nil {
			tm.logger.WithError(err).Error("Failed to scan theme row")
			continue
		}

		var theme ThemeDefinition
		if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
			tm.logger.WithError(err).Error("Failed to unmarshal theme")
			continue
		}

		themes = append(themes, theme)
	}

	return themes, nil
}

// getCustomTheme retrieves a specific custom theme
func (tm *ThemeManagerImpl) getCustomTheme(themeID string) (*ThemeDefinition, error) {
	query := `SELECT definition FROM custom_themes WHERE id = ?`

	var themeJSON string
	err := tm.db.QueryRow(query, themeID).Scan(&themeJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("theme not found")
		}
		return nil, fmt.Errorf("failed to get custom theme: %w", err)
	}

	var theme ThemeDefinition
	if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
		return nil, fmt.Errorf("failed to unmarshal theme: %w", err)
	}

	return &theme, nil
}

// validateTheme validates a theme definition
func (tm *ThemeManagerImpl) validateTheme(theme *ThemeDefinition) error {
	if theme.Name == "" {
		return fmt.Errorf("theme name is required")
	}

	if theme.ColorScheme != "light" && theme.ColorScheme != "dark" {
		return fmt.Errorf("color scheme must be 'light' or 'dark'")
	}

	// Validate required colors
	requiredColors := []string{
		theme.Colors.Primary,
		theme.Colors.Background,
		theme.Colors.OnBackground,
	}

	for _, color := range requiredColors {
		if color == "" {
			return fmt.Errorf("required colors cannot be empty")
		}
	}

	return nil
}
