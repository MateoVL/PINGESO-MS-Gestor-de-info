package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	cfg := cors.DefaultConfig()
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowOrigins = allowedOrigins
	}
	cfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	cfg.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", RequestIDHeader}
	cfg.ExposeHeaders = []string{RequestIDHeader}
	cfg.AllowCredentials = !cfg.AllowAllOrigins
	return cors.New(cfg)
}
