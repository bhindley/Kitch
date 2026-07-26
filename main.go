package main

import (
	"github.com/bhindley/Kitch/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	api := router.Group("/api")
	routes.RegisterHealthRoutes(api)

	router.Run() // listens on 0.0.0.0:8080 by default
}
