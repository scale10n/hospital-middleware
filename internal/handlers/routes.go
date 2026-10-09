package handlers

import (
	"time"

	_ "hospital-middleware/docs" // Blank import for Swagger doc registration
	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes configures global middleware, routes, and API groups.
// It accepts optional handler instances (*StaffHandler, *PatientHandler, auth.TokenService).
func RegisterRoutes(router *gin.Engine, healthHandler *HealthHandler, optionalArgs ...any) {
	// Register global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health
	if healthHandler != nil {
		router.GET("/health", healthHandler.HealthCheck)
	}

	var staffHandler *StaffHandler
	var patientHandler *PatientHandler
	var tokenService auth.TokenService

	for _, arg := range optionalArgs {
		switch v := arg.(type) {
		case *StaffHandler:
			staffHandler = v
		case *PatientHandler:
			patientHandler = v
		case auth.TokenService:
			tokenService = v
		}
	}

	if staffHandler == nil {
		staffHandler = NewStaffHandler()
	}
	if patientHandler == nil {
		patientHandler = NewPatientHandler()
	}
	if tokenService == nil {
		tokenService = auth.NewJWTService("hospital-middleware-default-secret-key-32bytes", 24*time.Hour)
	}

	// Staff routes
	router.POST("/staff/create", staffHandler.CreateStaff)
	router.POST("/staff/login", staffHandler.LoginStaff)

	// Patient routes protected by CookieAuth
	patientGroup := router.Group("/patient")
	patientGroup.Use(middleware.CookieAuth(tokenService))
	{
		patientGroup.POST("/search", patientHandler.SearchPatients)
	}
}
