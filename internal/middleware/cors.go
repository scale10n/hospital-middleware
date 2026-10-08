package middleware

import (
	"net/http"
	"strconv"

	"hospital-middleware/internal/config"

	"github.com/gin-gonic/gin"
)

// CORS handles Cross-Origin Resource Sharing (CORS) headers for incoming HTTP requests.
// It accepts optional config.CORSConfig; if none is provided, it loads from environment variables.
func CORS(cfgs ...config.CORSConfig) gin.HandlerFunc {
	var cfg config.CORSConfig
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	} else {
		appCfg := config.Load()
		cfg = appCfg.CORS
	}

	maxAgeStr := "86400"
	if cfg.MaxAgeSeconds > 0 {
		maxAgeStr = strconv.Itoa(cfg.MaxAgeSeconds)
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin == "" {
			// Non-browser or same-origin request without Origin header
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if isOriginAllowed(origin, cfg.AllowedOrigins) {
			// Allowed origin from Whitelist or Wildcard
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", maxAgeStr)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if the given origin is permitted based on allowed origins list.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
