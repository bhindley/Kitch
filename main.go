package main

import (
	"github.com/bhindley/Kitch/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	service_routes := router.Group("/service")
	routes.RegisterServiceRoutes(service_routes)

	router.Run() // :8080 by default
}
