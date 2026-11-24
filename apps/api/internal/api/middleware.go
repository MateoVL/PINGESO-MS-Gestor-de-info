package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/service"
)

const (
	RequestIDHeader      = "X-Request-ID"
	maxRequestIDLength   = 64
	problemJSONMediaType = "application/problem+json; charset=utf-8"
	problemTypeBaseURI   = "https://api.pingeso.cl/problems/"
)

// ProblemDetails implementa RFC 9457 (application/problem+json) sin exponer trazas, SQL ni rutas internas.
type ProblemDetails struct {
	Type      string `json:"type" doc:"URI que identifica el tipo de problema" example:"https://api.pingeso.cl/problems/invalid_data"`
	Title     string `json:"title" doc:"Resumen corto del tipo de problema" example:"Bad Request"`
	Status    int    `json:"status" doc:"Código de estado HTTP" example:"400"`
	Detail    string `json:"detail" doc:"Explicación legible para el usuario" example:"La solicitud contiene datos inválidos"`
	Instance  string `json:"instance,omitempty" doc:"Ruta de la petición que originó el problema" example:"/api/v1/lotes"`
	Code      string `json:"code" doc:"Código identificador del error" example:"invalid_data"`
	RequestID string `json:"request_id,omitempty" doc:"Identificador de correlación de la petición" example:"550e8400-e29b-41d4-a716-446655440000"`
	Details   any    `json:"details,omitempty" doc:"Detalles adicionales por campo"`
}

type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func (e *AppError) GetStatus() int {
	return e.Status
}

func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	return ok && t.Code == e.Code
}

func (e *AppError) WithDetails(details any) *AppError {
	clone := *e
	clone.Details = details
	return &clone
}

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

func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = ErrInternal
	}
	w.Header().Set("Content-Type", problemJSONMediaType)
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Type:      problemTypeBaseURI + appErr.Code,
		Title:     http.StatusText(appErr.Status),
		Status:    appErr.Status,
		Detail:    appErr.Message,
		Instance:  r.URL.Path,
		Code:      appErr.Code,
		RequestID: service.RequestIDFromContext(r.Context()),
		Details:   appErr.Details,
	})
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !isValidRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := service.WithRequestID(r.Context(), id)
		if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
			ctx = service.WithAuthToken(ctx, auth)
		} else if tok := extractToken(r); tok != "" {
			ctx = service.WithAuthToken(ctx, "Bearer "+tok)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func LoggerMiddleware(logger *slog.Logger, quietPaths ...string) func(http.Handler) http.Handler {
	quiet := make(map[string]struct{}, len(quietPaths))
	for _, p := range quietPaths {
		quiet[p] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			path := r.URL.Path
			if _, ok := quiet[path]; ok && rec.status < http.StatusBadRequest {
				return
			}

			level := slog.LevelInfo
			switch {
			case rec.status >= http.StatusInternalServerError:
				level = slog.LevelError
			case rec.status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}

			clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				clientIP = r.RemoteAddr
			}

			logger.LogAttrs(r.Context(), level, "petición HTTP",
				slog.String("request_id", service.RequestIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", path),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("client_ip", clientIP),
			)
		})
	}
}

func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recuperado",
						slog.String("request_id", service.RequestIDFromContext(r.Context())),
						slog.String("panic", fmt.Sprint(recovered)),
						slog.String("stack", string(debug.Stack())),
					)
					RespondError(w, r, ErrInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowedMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowedMap[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if allowedMap[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Add("Vary", "Origin")
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, Idempotency-Key, "+RequestIDHeader)
				w.Header().Set("Access-Control-Expose-Headers", RequestIDHeader)
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, r := range id {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !isAlnum && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isAllowedMethod(method string, allowed ...string) bool {
	for _, m := range allowed {
		if strings.EqualFold(method, m) {
			return true
		}
	}
	return false
}
