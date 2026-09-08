package routes

import (
	"github.com/bhindley/Kitch/controllers"
	"github.com/gin-gonic/gin"
)

// RegisterItemRoutes attaches item-related routes to the provided router group.
func RegisterItemRoutes(rg *gin.RouterGroup, controller *controllers.ItemController) {
	rg.POST("/items", controller.CreateItem)
}
