package middleware

import (
	"net/http"

	"github.com/frostdev-ops/pma-backend-go/pkg/errors"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandlingMiddleware handles panics and errors
func ErrorHandlingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			logger.WithField("error", err).Error("Panic recovered")
			utils.SendError(c, http.StatusInternalServerError, "Internal server error")
		} else {
			logger.WithField("error", recovered).Error("Panic recovered")
			utils.SendError(c, http.StatusInternalServerError, "Internal server error")
		}
		c.Abort()
	})
}

// AppErrorMiddleware handles application errors
func AppErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			if appErr, ok := err.(*errors.AppError); ok {
				utils.SendError(c, appErr.Code, appErr.Message)
			} else {
				utils.SendError(c, http.StatusInternalServerError, "Internal server error")
			}
		}
	}
}
