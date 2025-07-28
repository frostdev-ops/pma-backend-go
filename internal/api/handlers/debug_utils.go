package handlers

import "github.com/frostdev-ops/pma-backend-go/pkg/debug"

// DebugHandlerUtils is a deprecated alias for debug.ServiceLogger.
// Use debug.NewServiceLogger("handlers", logger) instead.
type DebugHandlerUtils = debug.ServiceLogger

// NewDebugHandlerUtils is a deprecated constructor for DebugHandlerUtils.
// Use debug.NewServiceLogger("handlers", logger) instead.
func NewDebugHandlerUtils(logger *debug.DebugLogger) *DebugHandlerUtils {
	return debug.NewServiceLogger("handlers", logger)
}
