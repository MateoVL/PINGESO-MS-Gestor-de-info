package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

// convierte un panic en un 500 con el formato de error estandar
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		logger.ErrorContext(c.Request.Context(), "panic recuperado",
			slog.String("request_id", utils.RequestID(c)),
			slog.String("panic", fmt.Sprint(recovered)),
			slog.String("stack", string(debug.Stack())),
		)
		utils.RespondError(c, utils.ErrInternal)
	})
}
