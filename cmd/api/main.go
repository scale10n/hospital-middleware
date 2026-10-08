package main

import (
	"database/sql"
	"log"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/handlers"
	"hospital-middleware/migrations"

	"github.com/gin-gonic/gin"
)

func setupRouter(db *sql.DB) *gin.Engine {
	router := gin.Default()

	// Disable trusted proxies to mitigate IP spoofing and suppress Gin startup warning
	_ = router.SetTrustedProxies(nil)

	// Initialize handlers (Delivery layer)
	healthHandler := handlers.NewHealthHandler(db)

	// Register application routes and middlewares
	handlers.RegisterRoutes(router, healthHandler)

	return router
}

func main() {
	// Load application configuration from .env and environment variables
	cfg := config.Load()

	// Set Gin mode (debug / release)
	gin.SetMode(cfg.Server.Mode)

	// Connect to PostgreSQL database
	db, err := database.New(cfg.DB)
	if err != nil {
		log.Printf("[WARNING] Database connection failed on startup: %v", err)
	} else {
		defer db.Close()
		log.Println("[INFO] Successfully connected to PostgreSQL database")

		// Run database auto-migrations if enabled
		if cfg.DB.AutoMigrate {
			if err := database.RunMigrations(db, migrations.FS); err != nil {
				log.Printf("[ERROR] Database migration failed: %v", err)
			}
		}
	}

	router := setupRouter(db)

	log.Printf("Starting server on :%s (mode: %s)...", cfg.Server.Port, cfg.Server.Mode)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
