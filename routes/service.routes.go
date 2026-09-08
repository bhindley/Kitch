package routes

import (
	"github.com/bhindley/Kitch/controllers"
	"github.com/gin-gonic/gin"
)

// RegisterHealthRoutes attaches health-related routes to the provided router group.
func RegisterHealthRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", controllers.HealthCheck)
}
