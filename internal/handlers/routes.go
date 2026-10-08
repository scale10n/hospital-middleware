package handlers

import (
	"hospital-middleware/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes configures global middleware, routes, and API groups.
func RegisterRoutes(router *gin.Engine, healthHandler *HealthHandler, staffHandlers ...*StaffHandler) {
	// Register global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	// Readiness and Health probes (root level for Docker/K8s)
	router.GET("/health", healthHandler.HealthCheck)

	var staffHandler *StaffHandler
	if len(staffHandlers) > 0 && staffHandlers[0] != nil {
		staffHandler = staffHandlers[0]
	} else {
		staffHandler = NewStaffHandler()
	}

	// Staff routes (/staff/create)
	staffGroup := router.Group("/staff")
	{
		staffHandler.RegisterRoutes(staffGroup)
	}
}
