package routes

import (
	"github.com/bhindley/Kitch/controllers"
	"github.com/gin-gonic/gin"
)

// RegisterItemRoutes attaches item-related routes to the provided router group.
func RegisterItemRoutes(rg *gin.RouterGroup, controller *controllers.ItemController) {
	rg.GET("/barcode/:barcode", controller.LookupBarcode)
	rg.POST("/items", controller.CreateItem)
	rg.PUT("/items/:id", controller.UpdateItem)
	rg.DELETE("/items/:id", controller.DeleteItem)
}
