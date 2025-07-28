package i18n

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// LocaleManagerImpl implements the LocaleManager interface
type LocaleManagerImpl struct {
	supportedLocales map[string]Locale
	userLocales      map[string]string            // userID -> localeCode
	translations     map[string]map[string]string // locale -> key -> translation
	fallbackChains   map[string][]string
	logger           *logrus.Logger
	mu               sync.RWMutex
}

// NewLocaleManager creates a new LocaleManager implementation
func NewLocaleManager(logger *logrus.Logger) *LocaleManagerImpl {
	manager := &LocaleManagerImpl{
		supportedLocales: make(map[string]Locale),
		userLocales:      make(map[string]string),
		translations:     make(map[string]map[string]string),
		fallbackChains:   make(map[string][]string),
		logger:           logger,
	}

	// Initialize with default locales
	for _, locale := range DefaultLocales() {
		manager.supportedLocales[locale.Code] = locale
	}

	// Setup fallback chains
	manager.fallbackChains["en-US"] = []string{"en"}
	manager.fallbackChains["es-ES"] = []string{"es", "en-US", "en"}
	manager.fallbackChains["fr-FR"] = []string{"fr", "en-US", "en"}
	manager.fallbackChains["de-DE"] = []string{"de", "en-US", "en"}

	// Initialize basic translations
	manager.initializeBasicTranslations()

	return manager
}

// GetSupportedLocales returns all supported locales
func (lm *LocaleManagerImpl) GetSupportedLocales() []Locale {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	locales := make([]Locale, 0, len(lm.supportedLocales))
	for _, locale := range lm.supportedLocales {
		locales = append(locales, locale)
	}
	return locales
}

// GetUserLocale returns the user's preferred locale
func (lm *LocaleManagerImpl) GetUserLocale(userID string) (*Locale, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	localeCode, exists := lm.userLocales[userID]
	if !exists {
		// Return default locale (en-US)
		if defaultLocale, ok := lm.supportedLocales["en-US"]; ok {
			return &defaultLocale, nil
		}
		return nil, fmt.Errorf("no locale found for user %s and default locale not available", userID)
	}

	locale, exists := lm.supportedLocales[localeCode]
	if !exists {
		return nil, fmt.Errorf("locale %s not supported", localeCode)
	}

	return &locale, nil
}

// SetUserLocale sets the user's preferred locale
func (lm *LocaleManagerImpl) SetUserLocale(userID string, localeCode string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Verify locale is supported
	if _, exists := lm.supportedLocales[localeCode]; !exists {
		return fmt.Errorf("locale %s not supported", localeCode)
	}

	lm.userLocales[userID] = localeCode
	lm.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"locale":  localeCode,
	}).Info("User locale updated")

	return nil
}

// Translate translates a key for the given locale
func (lm *LocaleManagerImpl) Translate(key string, locale string, args ...interface{}) string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Try to get translation for the requested locale
	if translations, exists := lm.translations[locale]; exists {
		if translation, exists := translations[key]; exists {
			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// Fall back through the fallback chain
	if fallbacks, exists := lm.fallbackChains[locale]; exists {
		for _, fallback := range fallbacks {
			if translations, exists := lm.translations[fallback]; exists {
				if translation, exists := translations[key]; exists {
					if len(args) > 0 {
						return fmt.Sprintf(translation, args...)
					}
					return translation
				}
			}
		}
	}

	// If no translation found, return the key itself
	lm.logger.WithFields(logrus.Fields{
		"key":    key,
		"locale": locale,
	}).Debug("Translation not found, returning key")

	return key
}

// TranslateWithPlurals translates a key with plural forms
func (lm *LocaleManagerImpl) TranslateWithPlurals(key string, locale string, count int, args ...interface{}) string {
	// Simple plural implementation - can be enhanced with CLDR rules
	pluralKey := key
	if count == 0 {
		pluralKey = key + ".zero"
	} else if count == 1 {
		pluralKey = key + ".one"
	} else {
		pluralKey = key + ".other"
	}

	// Try plural key first, fall back to base key
	translation := lm.Translate(pluralKey, locale, args...)
	if translation == pluralKey {
		// Plural key not found, try base key
		translation = lm.Translate(key, locale, args...)
	}

	// Replace count placeholder if present
	allArgs := append([]interface{}{count}, args...)
	return fmt.Sprintf(translation, allArgs...)
}

// GetTranslations returns all translations for a locale
func (lm *LocaleManagerImpl) GetTranslations(locale string) (map[string]string, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	translations, exists := lm.translations[locale]
	if !exists {
		return nil, fmt.Errorf("no translations found for locale %s", locale)
	}

	// Return a copy to prevent external modification
	result := make(map[string]string)
	for k, v := range translations {
		result[k] = v
	}

	return result, nil
}

// LoadTranslationFile loads translations from a JSON file
func (lm *LocaleManagerImpl) LoadTranslationFile(locale string, data []byte) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	var translationFile TranslationFile
	if err := json.Unmarshal(data, &translationFile); err != nil {
		return fmt.Errorf("failed to parse translation file: %w", err)
	}

	// Initialize translations map for locale if not exists
	if lm.translations[locale] == nil {
		lm.translations[locale] = make(map[string]string)
	}

	// Load translations
	for key, translation := range translationFile.Translations {
		lm.translations[locale][key] = translation.Value
	}

	lm.logger.WithFields(logrus.Fields{
		"locale":             locale,
		"translations_count": len(translationFile.Translations),
	}).Info("Translation file loaded")

	return nil
}

// ReloadTranslations reloads all translations
func (lm *LocaleManagerImpl) ReloadTranslations() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// For now, just reinitialize basic translations
	// In a real implementation, this would reload from files/database
	lm.translations = make(map[string]map[string]string)
	lm.initializeBasicTranslations()

	lm.logger.Info("Translations reloaded")
	return nil
}

// GetFallbackChain returns the fallback chain for a locale
func (lm *LocaleManagerImpl) GetFallbackChain(locale string) []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if fallbacks, exists := lm.fallbackChains[locale]; exists {
		// Return a copy
		result := make([]string, len(fallbacks))
		copy(result, fallbacks)
		return result
	}

	// Default fallback chain
	return []string{"en-US", "en"}
}

// initializeBasicTranslations sets up basic translations for testing
func (lm *LocaleManagerImpl) initializeBasicTranslations() {
	// English translations
	lm.translations["en-US"] = map[string]string{
		"common.loading":        "Loading...",
		"common.error":          "Error",
		"common.success":        "Success",
		"common.cancel":         "Cancel",
		"common.save":           "Save",
		"common.delete":         "Delete",
		"common.edit":           "Edit",
		"common.add":            "Add",
		"common.remove":         "Remove",
		"common.confirm":        "Confirm",
		"common.close":          "Close",
		"common.search":         "Search",
		"common.filter":         "Filter",
		"common.sort":           "Sort",
		"preferences.title":     "Preferences",
		"preferences.theme":     "Theme",
		"preferences.language":  "Language",
		"preferences.timezone":  "Timezone",
		"preferences.dashboard": "Dashboard",
		"system.status":         "System Status",
		"system.health":         "System Health",
		"system.uptime":         "Uptime",
		"automation.title":      "Automation",
		"automation.rules":      "Rules",
		"automation.triggers":   "Triggers",
		"entities.count.zero":   "No entities",
		"entities.count.one":    "%d entity",
		"entities.count.other":  "%d entities",
	}

	// Spanish translations
	lm.translations["es-ES"] = map[string]string{
		"common.loading":        "Cargando...",
		"common.error":          "Error",
		"common.success":        "Éxito",
		"common.cancel":         "Cancelar",
		"common.save":           "Guardar",
		"common.delete":         "Eliminar",
		"common.edit":           "Editar",
		"common.add":            "Añadir",
		"common.remove":         "Quitar",
		"common.confirm":        "Confirmar",
		"common.close":          "Cerrar",
		"common.search":         "Buscar",
		"common.filter":         "Filtrar",
		"common.sort":           "Ordenar",
		"preferences.title":     "Preferencias",
		"preferences.theme":     "Tema",
		"preferences.language":  "Idioma",
		"preferences.timezone":  "Zona horaria",
		"preferences.dashboard": "Panel de control",
		"system.status":         "Estado del sistema",
		"system.health":         "Salud del sistema",
		"system.uptime":         "Tiempo activo",
		"automation.title":      "Automatización",
		"automation.rules":      "Reglas",
		"automation.triggers":   "Disparadores",
		"entities.count.zero":   "Sin entidades",
		"entities.count.one":    "%d entidad",
		"entities.count.other":  "%d entidades",
	}

	// French translations
	lm.translations["fr-FR"] = map[string]string{
		"common.loading":        "Chargement...",
		"common.error":          "Erreur",
		"common.success":        "Succès",
		"common.cancel":         "Annuler",
		"common.save":           "Enregistrer",
		"common.delete":         "Supprimer",
		"common.edit":           "Modifier",
		"common.add":            "Ajouter",
		"common.remove":         "Retirer",
		"common.confirm":        "Confirmer",
		"common.close":          "Fermer",
		"common.search":         "Rechercher",
		"common.filter":         "Filtrer",
		"common.sort":           "Trier",
		"preferences.title":     "Préférences",
		"preferences.theme":     "Thème",
		"preferences.language":  "Langue",
		"preferences.timezone":  "Fuseau horaire",
		"preferences.dashboard": "Tableau de bord",
		"system.status":         "État du système",
		"system.health":         "Santé du système",
		"system.uptime":         "Temps de fonctionnement",
		"automation.title":      "Automatisation",
		"automation.rules":      "Règles",
		"automation.triggers":   "Déclencheurs",
		"entities.count.zero":   "Aucune entité",
		"entities.count.one":    "%d entité",
		"entities.count.other":  "%d entités",
	}

	// German translations
	lm.translations["de-DE"] = map[string]string{
		"common.loading":        "Laden...",
		"common.error":          "Fehler",
		"common.success":        "Erfolg",
		"common.cancel":         "Abbrechen",
		"common.save":           "Speichern",
		"common.delete":         "Löschen",
		"common.edit":           "Bearbeiten",
		"common.add":            "Hinzufügen",
		"common.remove":         "Entfernen",
		"common.confirm":        "Bestätigen",
		"common.close":          "Schließen",
		"common.search":         "Suchen",
		"common.filter":         "Filtern",
		"common.sort":           "Sortieren",
		"preferences.title":     "Einstellungen",
		"preferences.theme":     "Design",
		"preferences.language":  "Sprache",
		"preferences.timezone":  "Zeitzone",
		"preferences.dashboard": "Dashboard",
		"system.status":         "Systemstatus",
		"system.health":         "Systemzustand",
		"system.uptime":         "Betriebszeit",
		"automation.title":      "Automatisierung",
		"automation.rules":      "Regeln",
		"automation.triggers":   "Auslöser",
		"entities.count.zero":   "Keine Entitäten",
		"entities.count.one":    "%d Entität",
		"entities.count.other":  "%d Entitäten",
	}
}
