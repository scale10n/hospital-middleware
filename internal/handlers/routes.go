package handlers

import (
	"hospital-middleware/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes configures global middleware, routes, and API groups.
func RegisterRoutes(router *gin.Engine, healthHandler *HealthHandler) {
	// Register global middleware
	router.Use(middleware.CORS())

	// Readiness and Health probes (root level for Docker/K8s)
	router.GET("/health", healthHandler.HealthCheck)

	// API v1 route group for business domain resources
	v1 := router.Group("/api/v1")
	{
		// Future domain handlers will be registered here, e.g.:
		// patientHandler.RegisterRoutes(v1.Group("/patients"))
		v1.GET("/health", healthHandler.HealthCheck)
	}
}
