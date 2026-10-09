package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/handlers"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/service"
	"hospital-middleware/migrations"

	"github.com/gin-gonic/gin"
)

func setupRouter(db *sql.DB, cfgs ...*config.Config) *gin.Engine {
	router := gin.Default()

	// Disable trusted proxies to mitigate IP spoofing and suppress Gin startup warning
	_ = router.SetTrustedProxies(nil)

	var cfg *config.Config
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	} else {
		cfg = config.Load()
	}

	// Initialize handlers, services, and repositories
	healthHandler := handlers.NewHealthHandler(db)

	jwtService := auth.NewJWTService(cfg.JWT.Secret, time.Duration(cfg.JWT.ExpirySeconds)*time.Second)

	var staffService service.StaffService
	if db != nil {
		hospitalRepo := repository.NewHospitalRepository(db)
		staffRepo := repository.NewStaffRepository(db)
		sessionRepo := repository.NewStaffSessionRepository(db)
		staffService = service.NewStaffService(hospitalRepo, staffRepo, sessionRepo, jwtService)
	} else {
		staffService = service.NewStaffService(nil, nil, jwtService)
	}
	staffHandler := handlers.NewStaffHandlerWithService(staffService)

	// Register application routes and middlewares
	handlers.RegisterRoutes(router, healthHandler, staffHandler)

	return router
}

// @title                      Hospital Middleware API
// @version                    1.0
// @description                Middleware service handling staff authentication and hospital integrations.
//
// @host                       localhost:8080
// @BasePath                   /
//
// @securityDefinitions.apikey CookieAuth
// @in                         cookie
// @name                       session_token
// @description                Session token cookie for authenticated endpoints.
//
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Type "Bearer" followed by a space and the JWT token.
func main() {
	// Load application configuration from .env and environment variables
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[FATAL] Invalid configuration: %v", err)
	}

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

	// Pass cfg into setupRouter to reuse loaded configuration
	router := setupRouter(db, cfg)

	// Configure HTTP server with timeouts to prevent resource leaks and slowloris attacks
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run HTTP server in background goroutine for non-blocking lifecycle management
	go func() {
		log.Printf("[INFO] Starting server on :%s (mode: %s)...", cfg.Server.Port, cfg.Server.Mode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] Failed to start server: %v", err)
		}
	}()

	// Listen for OS shutdown signals (SIGINT, SIGTERM) for graceful termination
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[INFO] Received signal %v, initiating graceful shutdown...", sig)

	// Allow up to 5 seconds for in-flight requests to complete before terminating
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server exited cleanly")
}
