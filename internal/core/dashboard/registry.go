package dashboard

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// WidgetRegistryImpl implements the WidgetRegistry interface
type WidgetRegistryImpl struct {
	widgets   map[string]WidgetDefinition
	renderers map[string]WidgetRenderer
	mutex     sync.RWMutex
	logger    *logrus.Logger
}

// NewWidgetRegistry creates a new widget registry
func NewWidgetRegistry(logger *logrus.Logger) WidgetRegistry {
	return &WidgetRegistryImpl{
		widgets:   make(map[string]WidgetDefinition),
		renderers: make(map[string]WidgetRenderer),
		logger:    logger,
	}
}

// RegisterWidget registers a widget type with its renderer
func (r *WidgetRegistryImpl) RegisterWidget(definition WidgetDefinition, renderer WidgetRenderer) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if definition.Type == "" {
		return fmt.Errorf("widget type cannot be empty")
	}

	if renderer == nil {
		return fmt.Errorf("widget renderer cannot be nil")
	}

	r.widgets[definition.Type] = definition
	r.renderers[definition.Type] = renderer

	r.logger.WithFields(logrus.Fields{
		"widget_type": definition.Type,
		"widget_name": definition.Name,
	}).Info("Widget registered")

	return nil
}

// UnregisterWidget removes a widget type from the registry
func (r *WidgetRegistryImpl) UnregisterWidget(widgetType string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.widgets[widgetType]; !exists {
		return fmt.Errorf("widget type %s not found", widgetType)
	}

	delete(r.widgets, widgetType)
	delete(r.renderers, widgetType)

	r.logger.WithField("widget_type", widgetType).Info("Widget unregistered")

	return nil
}

// GetWidget retrieves a widget definition and renderer by type
func (r *WidgetRegistryImpl) GetWidget(widgetType string) (WidgetDefinition, WidgetRenderer, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	definition, exists := r.widgets[widgetType]
	if !exists {
		return WidgetDefinition{}, nil, fmt.Errorf("widget type %s not found", widgetType)
	}

	renderer := r.renderers[widgetType]
	return definition, renderer, nil
}

// GetAvailableWidgets returns all available widget definitions
func (r *WidgetRegistryImpl) GetAvailableWidgets() []WidgetDefinition {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	definitions := make([]WidgetDefinition, 0, len(r.widgets))
	for _, definition := range r.widgets {
		definitions = append(definitions, definition)
	}

	return definitions
}

// ValidateWidget validates a widget configuration
func (r *WidgetRegistryImpl) ValidateWidget(widget *Widget) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	definition, exists := r.widgets[widget.Type]
	if !exists {
		return fmt.Errorf("widget type %s not found", widget.Type)
	}

	// Validate size constraints
	if widget.Size.Width < definition.MinSize.Width || widget.Size.Height < definition.MinSize.Height {
		return fmt.Errorf("widget size below minimum constraints")
	}

	if definition.MaxSize.Width > 0 && widget.Size.Width > definition.MaxSize.Width {
		return fmt.Errorf("widget width exceeds maximum")
	}

	if definition.MaxSize.Height > 0 && widget.Size.Height > definition.MaxSize.Height {
		return fmt.Errorf("widget height exceeds maximum")
	}

	// Validate required configuration options
	renderer := r.renderers[widget.Type]
	if renderer != nil {
		if err := renderer.ValidateConfig(widget.Config); err != nil {
			return fmt.Errorf("widget config validation failed: %w", err)
		}
	}

	return nil
}
