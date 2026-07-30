package routes

import (
	"github.com/bhindley/Kitch/controllers"
	"github.com/gin-gonic/gin"
)

// Attaches health-related routes to the provided router group.
func RegisterServiceRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", controllers.HealthCheck)
}
