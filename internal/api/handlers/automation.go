package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/core/automation"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AutomationHandler handles automation-related HTTP requests
type AutomationHandler struct {
	engine *automation.AutomationEngine
	parser *automation.RuleParser
	logger *logrus.Logger
}

// NewAutomationHandler creates a new automation handler
func NewAutomationHandler(engine *automation.AutomationEngine, logger *logrus.Logger) *AutomationHandler {
	return &AutomationHandler{
		engine: engine,
		parser: automation.NewRuleParser(),
		logger: logger,
	}
}

// GetAutomations returns all automation rules
func (ah *AutomationHandler) GetAutomations(c *gin.Context) {
	ah.logger.Debug("Getting all automation rules")

	rules := ah.engine.GetAllRules()

	// Apply filters if specified
	if enabled := c.Query("enabled"); enabled != "" {
		if enabledBool, err := strconv.ParseBool(enabled); err == nil {
			filteredRules := make([]*automation.AutomationRule, 0)
			for _, rule := range rules {
				if rule.Enabled == enabledBool {
					filteredRules = append(filteredRules, rule)
				}
			}
			rules = filteredRules
		}
	}

	// Apply category filter
	if category := c.Query("category"); category != "" {
		filteredRules := make([]*automation.AutomationRule, 0)
		for _, rule := range rules {
			if rule.Category == category {
				filteredRules = append(filteredRules, rule)
			}
		}
		rules = filteredRules
	}

	// Apply tag filter
	if tag := c.Query("tag"); tag != "" {
		filteredRules := make([]*automation.AutomationRule, 0)
		for _, rule := range rules {
			if rule.HasTag(tag) {
				filteredRules = append(filteredRules, rule)
			}
		}
		rules = filteredRules
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rules": rules,
			"count": len(rules),
		},
	})
}

// CreateAutomation creates a new automation rule
func (ah *AutomationHandler) CreateAutomation(c *gin.Context) {
	ah.logger.Debug("Creating new automation rule")

	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		ah.logger.WithError(err).Error("Failed to parse request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Determine format (default to JSON)
	format := c.GetHeader("Content-Type")
	if strings.Contains(format, "yaml") || strings.Contains(format, "yml") {
		format = "yaml"
	} else {
		format = "json"
	}

	// Parse rule from request
	var rule *automation.AutomationRule
	var err error

	if format == "yaml" {
		// Convert back to YAML for parsing
		yamlData, err := json.Marshal(requestBody)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Failed to process YAML data",
			})
			return
		}
		rule, err = ah.parser.ParseFromJSON(yamlData) // Parse as JSON for now
	} else {
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Failed to process JSON data",
			})
			return
		}
		rule, err = ah.parser.ParseFromJSON(jsonData)
	}

	if err != nil {
		ah.logger.WithError(err).Error("Failed to parse automation rule")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to parse rule: %v", err),
		})
		return
	}

	// Add rule to engine
	if err := ah.engine.AddRule(rule); err != nil {
		ah.logger.WithError(err).Error("Failed to add automation rule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to add rule: %v", err),
		})
		return
	}

	ah.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Automation rule created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"rule": rule,
		},
	})
}

// GetAutomation returns a specific automation rule
func (ah *AutomationHandler) GetAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Getting automation rule")

	rule, err := ah.engine.GetRule(ruleID)
	if err != nil {
		ah.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to get automation rule")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Rule not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rule": rule,
		},
	})
}

// UpdateAutomation updates an existing automation rule
func (ah *AutomationHandler) UpdateAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Updating automation rule")

	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		ah.logger.WithError(err).Error("Failed to parse request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Ensure the rule ID matches the URL parameter
	requestBody["id"] = ruleID

	// Parse updated rule
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to process request data",
		})
		return
	}

	rule, err := ah.parser.ParseFromJSON(jsonData)
	if err != nil {
		ah.logger.WithError(err).Error("Failed to parse automation rule")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to parse rule: %v", err),
		})
		return
	}

	// Update rule in engine
	if err := ah.engine.UpdateRule(rule); err != nil {
		ah.logger.WithError(err).Error("Failed to update automation rule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to update rule: %v", err),
		})
		return
	}

	ah.logger.WithField("rule_id", ruleID).Info("Automation rule updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rule": rule,
		},
	})
}

// DeleteAutomation deletes an automation rule
func (ah *AutomationHandler) DeleteAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Deleting automation rule")

	if err := ah.engine.RemoveRule(ruleID); err != nil {
		ah.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to delete automation rule")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Rule not found",
		})
		return
	}

	ah.logger.WithField("rule_id", ruleID).Info("Automation rule deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rule deleted successfully",
	})
}

// EnableAutomation enables an automation rule
func (ah *AutomationHandler) EnableAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Enabling automation rule")

	if err := ah.engine.EnableRule(ruleID); err != nil {
		ah.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to enable automation rule")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ah.logger.WithField("rule_id", ruleID).Info("Automation rule enabled successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rule enabled successfully",
	})
}

// DisableAutomation disables an automation rule
func (ah *AutomationHandler) DisableAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Disabling automation rule")

	if err := ah.engine.DisableRule(ruleID); err != nil {
		ah.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to disable automation rule")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ah.logger.WithField("rule_id", ruleID).Info("Automation rule disabled successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rule disabled successfully",
	})
}

// TestAutomation tests an automation rule
func (ah *AutomationHandler) TestAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Testing automation rule")

	var testData map[string]interface{}
	if err := c.ShouldBindJSON(&testData); err != nil {
		// Use empty test data if none provided
		testData = make(map[string]interface{})
	}

	execCtx, err := ah.engine.TestRule(ruleID, testData)
	if err != nil {
		ah.logger.WithError(err).WithField("rule_id", ruleID).Error("Failed to test automation rule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to test rule: %v", err),
		})
		return
	}

	// Get execution summary
	summary := execCtx.GetSummary()
	trace := execCtx.GetTrace()

	ah.logger.WithField("rule_id", ruleID).Info("Automation rule tested successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"execution": summary,
			"trace":     trace,
		},
	})
}

// GetAutomationHistory returns execution history for a rule
func (ah *AutomationHandler) GetAutomationHistory(c *gin.Context) {
	ruleID := c.Param("id")
	ah.logger.WithField("rule_id", ruleID).Debug("Getting automation rule history")

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// For now, return mock data as the actual history would require database storage
	// In a real implementation, this would query the database for execution history
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"executions": []gin.H{},
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": 0,
			},
		},
	})
}

// ValidateAutomation validates automation rule syntax
func (ah *AutomationHandler) ValidateAutomation(c *gin.Context) {
	ah.logger.Debug("Validating automation rule")

	// Get content type and body
	contentType := c.GetHeader("Content-Type")
	format := "json"
	if strings.Contains(contentType, "yaml") || strings.Contains(contentType, "yml") {
		format = "yaml"
	}

	// Read raw body
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to read request body",
		})
		return
	}

	// Validate syntax
	validation := ah.parser.ValidateRuleSyntax(body, format)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"validation": validation,
		},
	})
}

// GetAutomationStatistics returns automation engine statistics
func (ah *AutomationHandler) GetAutomationStatistics(c *gin.Context) {
	ah.logger.Debug("Getting automation statistics")

	stats := ah.engine.GetStatistics()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"statistics": stats,
		},
	})
}

// ExportAutomation exports a rule to YAML or JSON
func (ah *AutomationHandler) ExportAutomation(c *gin.Context) {
	ruleID := c.Param("id")
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	ah.logger.WithFields(logrus.Fields{
		"rule_id": ruleID,
		"format":  format,
	}).Debug("Exporting automation rule")

	rule, err := ah.engine.GetRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Rule not found",
		})
		return
	}

	var data []byte
	var contentType string

	switch strings.ToLower(format) {
	case "yaml", "yml":
		data, err = ah.parser.SerializeToYAML(rule)
		contentType = "application/x-yaml"
	default:
		data, err = ah.parser.SerializeToJSON(rule)
		contentType = "application/json"
	}

	if err != nil {
		ah.logger.WithError(err).Error("Failed to serialize rule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to export rule",
		})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.%s\"", rule.Name, format))
	c.Data(http.StatusOK, contentType, data)
}

// ImportAutomation imports a rule from YAML or JSON
func (ah *AutomationHandler) ImportAutomation(c *gin.Context) {
	ah.logger.Debug("Importing automation rule")

	// Read raw body
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to read request body",
		})
		return
	}

	// Determine format from content type or query parameter
	contentType := c.GetHeader("Content-Type")
	format := c.Query("format")

	if format == "" {
		if strings.Contains(contentType, "yaml") || strings.Contains(contentType, "yml") {
			format = "yaml"
		} else {
			format = "json"
		}
	}

	// Parse rule
	var rule *automation.AutomationRule
	switch strings.ToLower(format) {
	case "yaml", "yml":
		rule, err = ah.parser.ParseFromYAML(body)
	default:
		rule, err = ah.parser.ParseFromJSON(body)
	}

	if err != nil {
		ah.logger.WithError(err).Error("Failed to parse imported rule")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to parse rule: %v", err),
		})
		return
	}

	// Add rule to engine
	if err := ah.engine.AddRule(rule); err != nil {
		ah.logger.WithError(err).Error("Failed to add imported rule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to import rule: %v", err),
		})
		return
	}

	ah.logger.WithField("rule_id", rule.ID).Info("Automation rule imported successfully")

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"rule": rule,
		},
	})
}

// GetAutomationTemplates returns common automation templates
func (ah *AutomationHandler) GetAutomationTemplates(c *gin.Context) {
	ah.logger.Debug("Getting automation templates")

	// Common automation templates
	templates := []gin.H{
		{
			"id":          "motion_lights",
			"name":        "Motion Activated Lights",
			"description": "Turn on lights when motion is detected",
			"category":    "lighting",
			"template": gin.H{
				"name": "Motion Activated Lights",
				"triggers": []gin.H{
					{
						"platform":  "state",
						"entity_id": "binary_sensor.motion_sensor",
						"to":        "on",
					},
				},
				"conditions": []gin.H{
					{
						"condition": "time",
						"after":     "sunset",
						"before":    "sunrise",
					},
				},
				"actions": []gin.H{
					{
						"service":   "light.turn_on",
						"entity_id": "light.living_room",
						"data": gin.H{
							"brightness": 255,
						},
					},
				},
			},
		},
		{
			"id":          "sunrise_lights_off",
			"name":        "Turn Off Lights at Sunrise",
			"description": "Automatically turn off lights when the sun rises",
			"category":    "lighting",
			"template": gin.H{
				"name": "Turn Off Lights at Sunrise",
				"triggers": []gin.H{
					{
						"platform": "sun",
						"event":    "sunrise",
					},
				},
				"actions": []gin.H{
					{
						"service": "light.turn_off",
						"target": gin.H{
							"area_id": "living_room",
						},
					},
				},
			},
		},
		{
			"id":          "temperature_alert",
			"name":        "Temperature Alert",
			"description": "Send notification when temperature exceeds threshold",
			"category":    "monitoring",
			"template": gin.H{
				"name": "Temperature Alert",
				"triggers": []gin.H{
					{
						"platform":  "numeric_state",
						"entity_id": "sensor.temperature",
						"above":     25.0,
					},
				},
				"actions": []gin.H{
					{
						"service": "notify.mobile_app",
						"data": gin.H{
							"title":   "Temperature Alert",
							"message": "Temperature is above 25°C: {{ states('sensor.temperature') }}°C",
						},
					},
				},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"templates": templates,
		},
	})
}

// RegisterRoutes registers automation routes
func (ah *AutomationHandler) RegisterRoutes(router *gin.RouterGroup) {
	automations := router.Group("/automations")
	{
		automations.GET("", ah.GetAutomations)
		automations.POST("", ah.CreateAutomation)
		automations.GET("/:id", ah.GetAutomation)
		automations.PUT("/:id", ah.UpdateAutomation)
		automations.DELETE("/:id", ah.DeleteAutomation)
		automations.POST("/:id/enable", ah.EnableAutomation)
		automations.POST("/:id/disable", ah.DisableAutomation)
		automations.POST("/:id/test", ah.TestAutomation)
		automations.GET("/:id/history", ah.GetAutomationHistory)
		automations.GET("/:id/export", ah.ExportAutomation)
		automations.POST("/validate", ah.ValidateAutomation)
		automations.POST("/import", ah.ImportAutomation)
		automations.GET("/templates", ah.GetAutomationTemplates)
		automations.GET("/statistics", ah.GetAutomationStatistics)
	}
}
