package analytics

import (
	"database/sql"

	"github.com/sirupsen/logrus"
)

// For now, we'll use the NewSimpleAnalyticsManager as the main implementation
// This avoids import cycle issues and provides a working implementation

// NewAnalyticsManager creates a new analytics manager (using simple implementation)
func NewAnalyticsManager(db *sql.DB, config *AnalyticsConfig, logger *logrus.Logger) (AnalyticsManager, error) {
	// For now, delegate to the simple manager to avoid complexity
	return NewSimpleAnalyticsManager(db, logger), nil
}
