package handlers

import (
	"context"
	"net/http"
	"reflect"
	"time"

	_ "hospital-middleware/internal/response"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response payload.
type HealthResponse struct {
	Status   string `json:"status" example:"healthy"`
	Database string `json:"database" example:"connected"`
	Error    string `json:"error,omitempty" example:""`
}

// DBPinger defines an interface for testing database connectivity.
// This decouples the handler from a concrete *sql.DB and enables easy mocking.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// HealthHandler manages system status and health check endpoints.
type HealthHandler struct {
	db DBPinger
}

// NewHealthHandler constructs a new HealthHandler instance.
func NewHealthHandler(db DBPinger) *HealthHandler {
	if db == nil || (reflect.ValueOf(db).Kind() == reflect.Ptr && reflect.ValueOf(db).IsNil()) {
		return &HealthHandler{db: nil}
	}
	return &HealthHandler{db: db}
}

// HealthCheck handles readiness probe requests (GET /health).
// @Summary      Liveness and Readiness probe
// @Description  Check server and database connectivity status
// @Tags         Health
// @Produce      json
// @Success      200 {object} response.Response[HealthResponse]
// @Failure      503 {object} response.Response[HealthResponse]
// @Router       /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "degraded",
			"database": "disconnected",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "degraded",
			"database": "disconnected",
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}
