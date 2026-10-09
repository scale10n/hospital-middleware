package handlers

import (
	_ "hospital-middleware/docs" // Blank import for Swagger doc registration
	"hospital-middleware/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes configures global middleware, routes, and API groups.
func RegisterRoutes(router *gin.Engine, healthHandler *HealthHandler, staffHandlers ...*StaffHandler) {
	// Register global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health
	router.GET("/health", healthHandler.HealthCheck)

	var staffHandler *StaffHandler
	if len(staffHandlers) > 0 && staffHandlers[0] != nil {
		staffHandler = staffHandlers[0]
	} else {
		staffHandler = NewStaffHandler()
	}

	// Staff
	router.POST("/staff/create", staffHandler.CreateStaff)
	router.POST("/staff/login", staffHandler.LoginStaff)

	// TODO: implement patient handler
}

