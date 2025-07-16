package handlers

import (
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/sirupsen/logrus"
)

// Handlers holds all HTTP handlers and their dependencies
type Handlers struct {
	cfg    *config.Config
	repos  *database.Repositories
	logger *logrus.Logger
}

// NewHandlers creates a new handlers instance
func NewHandlers(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger) *Handlers {
	return &Handlers{
		cfg:    cfg,
		repos:  repos,
		logger: logger,
	}
}
