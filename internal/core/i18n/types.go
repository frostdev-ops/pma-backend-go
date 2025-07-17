package i18n

import (
	"time"
)

// LocaleManager defines the interface for managing locales and translations
type LocaleManager interface {
	GetSupportedLocales() []Locale
	GetUserLocale(userID string) (*Locale, error)
	SetUserLocale(userID string, localeCode string) error
	Translate(key string, locale string, args ...interface{}) string
	TranslateWithPlurals(key string, locale string, count int, args ...interface{}) string
	GetTranslations(locale string) (map[string]string, error)
	LoadTranslationFile(locale string, data []byte) error
	ReloadTranslations() error
	GetFallbackChain(locale string) []string
}

// Locale represents a supported language/region
type Locale struct {
	Code         string       `json:"code"`        // en-US, es-ES, etc.
	Language     string       `json:"language"`    // en, es, etc.
	Region       string       `json:"region"`      // US, ES, etc.
	Name         string       `json:"name"`        // English (United States)
	NativeName   string       `json:"native_name"` // English (United States)
	Direction    string       `json:"direction"`   // ltr, rtl
	DateFormat   string       `json:"date_format"` // MM/DD/YYYY, DD/MM/YYYY, etc.
	TimeFormat   string       `json:"time_format"` // 12h, 24h
	NumberFormat NumberFormat `json:"number_format"`
	Currency     Currency     `json:"currency"`
	Enabled      bool         `json:"enabled"`
	Progress     float64      `json:"progress"` // Translation completion percentage
	UpdatedAt    time.Time    `json:"updated_at"`
}

// NumberFormat defines number formatting for a locale
type NumberFormat struct {
	Decimal   string `json:"decimal"`   // "."
	Thousands string `json:"thousands"` // ","
	Grouping  []int  `json:"grouping"`  // [3] for 1,000 or [3,2] for 1,00,000
	Pattern   string `json:"pattern"`   // "#,##0.##"
	Precision int    `json:"precision"` // Default decimal places
}

// Currency defines currency formatting for a locale
type Currency struct {
	Code     string `json:"code"`     // USD, EUR, etc.
	Symbol   string `json:"symbol"`   // $, €, etc.
	Position string `json:"position"` // before, after
	Pattern  string `json:"pattern"`  // ¤#,##0.00
}

// Translation represents a translation entry
type Translation struct {
	Key     string            `json:"key"`
	Value   string            `json:"value"`
	Plurals map[string]string `json:"plurals,omitempty"` // zero, one, two, few, many, other
	Context string            `json:"context,omitempty"`
	Comment string            `json:"comment,omitempty"`
	Tags    []string          `json:"tags,omitempty"`
}

// TranslationFile represents a translation file structure
type TranslationFile struct {
	Locale       string                 `json:"locale"`
	Version      string                 `json:"version"`
	Namespace    string                 `json:"namespace,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Translations map[string]Translation `json:"translations"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PluralRule defines plural rules for a language
type PluralRule struct {
	Language string   `json:"language"`
	Rules    []string `json:"rules"` // CLDR plural rule expressions
}

// TranslationContext provides context for translation operations
type TranslationContext struct {
	UserID    string                 `json:"user_id"`
	Locale    string                 `json:"locale"`
	Fallbacks []string               `json:"fallbacks"`
	Timezone  string                 `json:"timezone"`
	Variables map[string]interface{} `json:"variables"`
	Namespace string                 `json:"namespace,omitempty"`
}

// TranslationStats represents translation statistics
type TranslationStats struct {
	Locale         string    `json:"locale"`
	TotalKeys      int       `json:"total_keys"`
	TranslatedKeys int       `json:"translated_keys"`
	MissingKeys    int       `json:"missing_keys"`
	Progress       float64   `json:"progress"`
	LastUpdated    time.Time `json:"last_updated"`
}

// Translator defines the interface for translation providers
type Translator interface {
	Translate(key string, locale string, args ...interface{}) (string, error)
	TranslateWithContext(key string, context TranslationContext) (string, error)
	HasTranslation(key string, locale string) bool
	GetTranslations(locale string, namespace string) (map[string]string, error)
	ReloadTranslations(locale string) error
}

// DefaultLocales returns the list of supported locales with their configurations
func DefaultLocales() []Locale {
	return []Locale{
		{
			Code:       "en-US",
			Language:   "en",
			Region:     "US",
			Name:       "English (United States)",
			NativeName: "English (United States)",
			Direction:  "ltr",
			DateFormat: "MM/DD/YYYY",
			TimeFormat: "12h",
			NumberFormat: NumberFormat{
				Decimal:   ".",
				Thousands: ",",
				Grouping:  []int{3},
				Pattern:   "#,##0.##",
				Precision: 2,
			},
			Currency: Currency{
				Code:     "USD",
				Symbol:   "$",
				Position: "before",
				Pattern:  "¤#,##0.00",
			},
			Enabled:   true,
			Progress:  100.0,
			UpdatedAt: time.Now(),
		},
		{
			Code:       "es-ES",
			Language:   "es",
			Region:     "ES",
			Name:       "Spanish (Spain)",
			NativeName: "Español (España)",
			Direction:  "ltr",
			DateFormat: "DD/MM/YYYY",
			TimeFormat: "24h",
			NumberFormat: NumberFormat{
				Decimal:   ",",
				Thousands: ".",
				Grouping:  []int{3},
				Pattern:   "#.##0,##",
				Precision: 2,
			},
			Currency: Currency{
				Code:     "EUR",
				Symbol:   "€",
				Position: "after",
				Pattern:  "#,##0.00 ¤",
			},
			Enabled:   true,
			Progress:  95.0,
			UpdatedAt: time.Now(),
		},
		{
			Code:       "fr-FR",
			Language:   "fr",
			Region:     "FR",
			Name:       "French (France)",
			NativeName: "Français (France)",
			Direction:  "ltr",
			DateFormat: "DD/MM/YYYY",
			TimeFormat: "24h",
			NumberFormat: NumberFormat{
				Decimal:   ",",
				Thousands: " ",
				Grouping:  []int{3},
				Pattern:   "# ##0,##",
				Precision: 2,
			},
			Currency: Currency{
				Code:     "EUR",
				Symbol:   "€",
				Position: "after",
				Pattern:  "#,##0.00 ¤",
			},
			Enabled:   true,
			Progress:  90.0,
			UpdatedAt: time.Now(),
		},
		{
			Code:       "de-DE",
			Language:   "de",
			Region:     "DE",
			Name:       "German (Germany)",
			NativeName: "Deutsch (Deutschland)",
			Direction:  "ltr",
			DateFormat: "DD.MM.YYYY",
			TimeFormat: "24h",
			NumberFormat: NumberFormat{
				Decimal:   ",",
				Thousands: ".",
				Grouping:  []int{3},
				Pattern:   "#.##0,##",
				Precision: 2,
			},
			Currency: Currency{
				Code:     "EUR",
				Symbol:   "€",
				Position: "after",
				Pattern:  "#,##0.00 ¤",
			},
			Enabled:   true,
			Progress:  85.0,
			UpdatedAt: time.Now(),
		},
		{
			Code:       "zh-CN",
			Language:   "zh",
			Region:     "CN",
			Name:       "Chinese (Simplified)",
			NativeName: "中文（简体）",
			Direction:  "ltr",
			DateFormat: "YYYY/MM/DD",
			TimeFormat: "24h",
			NumberFormat: NumberFormat{
				Decimal:   ".",
				Thousands: ",",
				Grouping:  []int{3},
				Pattern:   "#,##0.##",
				Precision: 2,
			},
			Currency: Currency{
				Code:     "CNY",
				Symbol:   "¥",
				Position: "before",
				Pattern:  "¤#,##0.00",
			},
			Enabled:   true,
			Progress:  80.0,
			UpdatedAt: time.Now(),
		},
		{
			Code:       "ja-JP",
			Language:   "ja",
			Region:     "JP",
			Name:       "Japanese (Japan)",
			NativeName: "日本語（日本）",
			Direction:  "ltr",
			DateFormat: "YYYY/MM/DD",
			TimeFormat: "24h",
			NumberFormat: NumberFormat{
				Decimal:   ".",
				Thousands: ",",
				Grouping:  []int{3},
				Pattern:   "#,##0.##",
				Precision: 2,
			},
			Currency: Currency{
				Code:     "JPY",
				Symbol:   "¥",
				Position: "before",
				Pattern:  "¤#,##0",
			},
			Enabled:   true,
			Progress:  75.0,
			UpdatedAt: time.Now(),
		},
	}
}

// GetPluralRules returns CLDR plural rules for supported languages
func GetPluralRules() map[string]PluralRule {
	return map[string]PluralRule{
		"en": {
			Language: "en",
			Rules: []string{
				"one: n = 1",
				"other: true",
			},
		},
		"es": {
			Language: "es",
			Rules: []string{
				"one: n = 1",
				"other: true",
			},
		},
		"fr": {
			Language: "fr",
			Rules: []string{
				"one: n = 0..1",
				"other: true",
			},
		},
		"de": {
			Language: "de",
			Rules: []string{
				"one: n = 1",
				"other: true",
			},
		},
		"zh": {
			Language: "zh",
			Rules: []string{
				"other: true",
			},
		},
		"ja": {
			Language: "ja",
			Rules: []string{
				"other: true",
			},
		},
	}
}
