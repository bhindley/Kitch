package main

import (
	"log"

	"github.com/bhindley/Kitch/clients"
	"github.com/bhindley/Kitch/config"
	"github.com/bhindley/Kitch/controllers"
	"github.com/bhindley/Kitch/repositories"
	"github.com/bhindley/Kitch/routes"
	"github.com/bhindley/Kitch/services"
	"github.com/gin-gonic/gin"
)

func main() {
	// Database
	pool, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := config.RunMigrations(pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Dependencies
	itemRepo := repositories.NewPostgresItemRepository(pool)
	offClient := clients.NewOpenFoodFactsClient()
	itemService := services.NewItemService(itemRepo, offClient)
	itemController := controllers.NewItemController(itemService)

	// Router
	router := gin.Default()
	api := router.Group("/api")

	routes.RegisterHealthRoutes(api)
	routes.RegisterItemRoutes(api, itemController)

	router.Run() // listens on 0.0.0.0:8080 by default
}
