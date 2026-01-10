package main

import (
	"log"

	"channelmanager/cache"
	"channelmanager/config"
	"channelmanager/database"
	"channelmanager/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	log.Println("Configuration loaded")

	// Initialize database
	db, err := database.InitializeDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized")

	// Initialize Redis
	redis, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redis.Close()
	log.Println("Redis initialized")

	// Initialize Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Initialize handlers
	handler := handlers.NewHandler(db, redis)

	// Setup routes
	setupRoutes(router, handler)

	// Initialize and start event listener for cache invalidation
	eventListener := handlers.NewEventListener(db, redis)
	eventListener.Start()
	defer eventListener.Stop()

	log.Println("Event listener started")

	// Start server
	log.Printf("Starting server on %s:%s", cfg.Server.Host, cfg.Server.Port)
	if err := router.Run(cfg.Server.Host + ":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupRoutes sets up all API routes
func setupRoutes(router *gin.Engine, handler *handlers.Handler) {
	// Health check
	router.GET("/health", handler.HealthCheck)

	// Property search and retrieval
	api := router.Group("/api/v1")
	{
		// Search properties
		api.POST("/properties/search", handler.SearchProperties)

		// Get single property
		api.GET("/properties/:id", handler.GetProperty)

		// Get property availability
		api.GET("/properties/:id/availability", handler.GetPropertyAvailability)

		// Get amenities
		api.GET("/amenities", handler.GetAmenities)

		// Get conditions
		api.GET("/conditions", handler.GetConditions)
	}

	log.Println("Routes configured")
}
