package handlers

import (
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Health returns the health status of the service
func (h *Handlers) Health(c *gin.Context) {
	utils.SendSuccess(c, gin.H{
		"status":  "healthy",
		"service": "pma-backend-go",
		"version": "1.0.0",
	})
}
