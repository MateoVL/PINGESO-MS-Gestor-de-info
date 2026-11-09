package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
)

func TestHealth(t *testing.T) {
	const internalMessage = "internal MongoDB error"
	for _, tc := range []struct {
		name       string
		pingErr    error
		statusCode int
		status     string
	}{
		{"MongoDB disponible", nil, http.StatusOK, "healthy"},
		{"MongoDB caído", errors.New(internalMessage), http.StatusServiceUnavailable, "degraded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(&config.Config{AppName: "Pipeline de Información", AppEnv: "test"}, func(ctx context.Context) error {
				if _, ok := ctx.Deadline(); !ok {
					t.Error("el ping debe tener un plazo máximo")
				}
				return tc.pingErr
			})
			router := gin.New()
			router.GET("/api/health", h.Health)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/health", nil))

			if res.Code != tc.statusCode {
				t.Fatalf("código HTTP %d, se esperaba %d", res.Code, tc.statusCode)
			}
			var body HealthResponse
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatalf("diagnóstico inválido: %v", err)
			}
			if body.Status != tc.status || body.Database.Engine != "MongoDB" || body.Database.Connected != (tc.pingErr == nil) {
				t.Errorf("diagnóstico inesperado: %+v", body)
			}
			if strings.Contains(res.Body.String(), internalMessage) {
				t.Error("el diagnóstico publicó un error interno de MongoDB")
			}
		})
	}
}
