package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

// registra cada peticion con slog, metodo, ruta, estado, duracion e ID
func Logger(logger *slog.Logger, quietPaths ...string) gin.HandlerFunc {
	quiet := make(map[string]struct{}, len(quietPaths))
	for _, p := range quietPaths {
		quiet[p] = struct{}{}
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		path := c.Request.URL.Path
		if _, ok := quiet[path]; ok && status < http.StatusBadRequest {
			return
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		// query string no se registra porque podria contener datos sensibles
		attrs := []slog.Attr{
			slog.String("request_id", utils.RequestID(c)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", strings.Join(c.Errors.Errors(), "; ")))
		}

		logger.LogAttrs(c.Request.Context(), level, "petición HTTP", attrs...)
	}
}
