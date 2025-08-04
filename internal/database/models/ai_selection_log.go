package models

import (
	"time"

	"gorm.io/gorm"
)

// AISelectionLog represents a log entry for AI model selection and chat completion
type AISelectionLog struct {
	gorm.Model
	RequestID                                  string `gorm:"uniqueIndex"`
	Timestamp                                  time.Time
	UserID                                     string // GAP: Will be added later
	SessionID                                  string // GAP: Will be added later
	RequestMessageContent                      string `gorm:"type:text"`
	RequestMessageLength                       int
	RequestMessageTokenEstimate                int
	RequestHasToolPatterns                     bool
	RequestHasAutomationPatterns               bool
	RequestHasComplexPatterns                  bool
	RequestHasSimplePatterns                   bool
	RequestChatOptionsProviderOverride         string
	RequestChatOptionsModelOverride            string
	RequestChatOptionsMaxTokens                int
	RequestChatOptionsTemperature              float64
	RequestChatOptionsTopP                     float64
	RequestChatOptionsStream                   bool
	ConversationMessageCount                   int
	RuntimeModelQ2IsLoaded                     bool
	RuntimeModelQ4IsLoaded                     bool
	RuntimeModelQ8IsLoaded                     bool
	RuntimeModelFP16IsLoaded                   bool
	RuntimeProviderLlamaCppRequestCount        int64
	RuntimeProviderLlamaCppErrorCount          int64
	RuntimeProviderLlamaCppAvgResponseTimeMs   int64
	RuntimeProviderLlamaCppCircuitBreakerState string
	ConfigAIEnabled                            bool
	ConfigLlamaCppEnabled                      bool
	ConfigLlamaCppAutoStart                    bool
	ConfigModelQ2MaxTokens                     int
	ConfigModelQ2Temperature                   float64
	ConfigModelQ2ResponseTimeMs                int64
	ConfigModelQ2Accuracy                      float64
	ConfigModelQ2MemoryUsageMb                 int64
	ConfigModelQ4MaxTokens                     int
	ConfigModelQ4Temperature                   float64
	ConfigModelQ4ResponseTimeMs                int64
	ConfigModelQ4Accuracy                      float64
	ConfigModelQ4MemoryUsageMb                 int64
	ConfigModelQ8MaxTokens                     int
	ConfigModelQ8Temperature                   float64
	ConfigModelQ8ResponseTimeMs                int64
	ConfigModelQ8Accuracy                      float64
	ConfigModelQ8MemoryUsageMb                 int64
	ConfigModelFP16MaxTokens                   int
	ConfigModelFP16Temperature                 float64
	ConfigModelFP16ResponseTimeMs              int64
	ConfigModelFP16Accuracy                    float64
	ConfigModelFP16MemoryUsageMb               int64
	ChosenModel                                string
	ComplexityClass                            string
	Success                                    bool
	LatencyMs                                  int64
	ErrorType                                  string
	FallbackUsedInSelector                     bool
	FinishReason                               string
	TokensUsedTotal                            int
	OptimalityHeuristic                        bool    // Derived label
	CostUsd                                    float64 // GAP: Will be added later
	RetryAttemptCount                          int     // GAP: Will be added later
	FallbackProviderUsed                       string  // GAP: Will be added later
}
