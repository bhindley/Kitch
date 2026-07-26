package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns a 200 OK response to indicate the service is running.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
	})
}
