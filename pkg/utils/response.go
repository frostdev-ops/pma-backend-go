package utils

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
    Success   bool        `json:"success"`
    Data      interface{} `json:"data,omitempty"`
    Error     string      `json:"error,omitempty"`
    Timestamp string      `json:"timestamp"`
    Meta      interface{} `json:"meta,omitempty"`
}

// SendSuccess sends a successful response
func SendSuccess(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Success:   true,
        Data:      data,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })
}

// SendError sends an error response
func SendError(c *gin.Context, statusCode int, message string) {
    c.JSON(statusCode, Response{
        Success:   false,
        Error:     message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })
}

// SendSuccessWithMeta sends a successful response with metadata
func SendSuccessWithMeta(c *gin.Context, data interface{}, meta interface{}) {
    c.JSON(http.StatusOK, Response{
        Success:   true,
        Data:      data,
        Meta:      meta,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })
} 