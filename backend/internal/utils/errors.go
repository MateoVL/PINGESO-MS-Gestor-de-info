package utils

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

// error que se puede mostrar al cliente
type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

// compara por codigo
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	return ok && t.Code == e.Code
}

// devuelve una copia del error con informacion extra para el cliente
func (e *AppError) WithDetails(details any) *AppError {
	clone := *e
	clone.Details = details
	return &clone
}

// errores conocidos de la API
var (
	ErrInvalidData          = &AppError{Status: http.StatusBadRequest, Code: "invalid_data", Message: "La solicitud contiene datos inválidos"}
	ErrUnauthorized         = &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "Se requiere autenticación"}
	ErrForbidden            = &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: "No tienes permisos para realizar esta acción"}
	ErrNotFound             = &AppError{Status: http.StatusNotFound, Code: "not_found", Message: "El recurso solicitado no existe"}
	ErrMethodNotAllowed     = &AppError{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "El método HTTP no está permitido en esta ruta"}
	ErrConflict             = &AppError{Status: http.StatusConflict, Code: "conflict", Message: "El recurso entra en conflicto con uno existente"}
	ErrPayloadTooLarge      = &AppError{Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: "El cuerpo de la solicitud supera el tamaño permitido"}
	ErrUnsupportedMediaType = &AppError{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: "El Content-Type debe ser application/json o text/csv"}
	ErrInternal             = &AppError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Ocurrió un error interno"}
	ErrUnavailable          = &AppError{Status: http.StatusServiceUnavailable, Code: "service_unavailable", Message: "El servicio no está disponible temporalmente"}
)

func RespondError(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = ErrInternal
	}
	if appErr.Status >= http.StatusInternalServerError {
		_ = c.Error(err)
	}

	c.AbortWithStatusJSON(appErr.Status, models.ErrorResponse{
		Error: models.ErrorBody{
			Code:      appErr.Code,
			Message:   appErr.Message,
			RequestID: RequestID(c),
			Details:   appErr.Details,
		},
	})
}
