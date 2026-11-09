package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
	"github.com/mateo/ms-gestor-de-info/backend/internal/handlers"
	"github.com/mateo/ms-gestor-de-info/backend/internal/middleware"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/repositories"
	"github.com/mateo/ms-gestor-de-info/backend/internal/services"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func newTestRouter(pingErr error) *gin.Engine {
	health := services.NewHealthService(config.ServiceInfo{Name: "Pipeline", Environment: "test"},
		func(context.Context) error { return pingErr })

	r := NewRouter(
		Options{AllowedOrigins: []string{"http://localhost:3000"}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), PublishDocs: true},
		Dependencies{
			Health:  handlers.NewHealthHandler(health),
			Batches: handlers.NewBatchHandler(services.NewBatchService(repositories.NewMemoryBatchRepository())),
		},
	)
	r.GET("/panic", func(*gin.Context) { panic("falla inesperada") })
	return r
}

func do(r http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHealthEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pingErr error
		code    int
		status  string
	}{
		{"MongoDB disponible", nil, http.StatusOK, models.HealthStatusHealthy},
		{"MongoDB caído", errors.New("internal MongoDB error"), http.StatusServiceUnavailable, models.HealthStatusDegraded},
	} {
		for _, path := range []string{"/health", "/api/health"} {
			t.Run(tc.name+" "+path, func(t *testing.T) {
				rec := do(newTestRouter(tc.pingErr), http.MethodGet, path, nil)

				if rec.Code != tc.code {
					t.Fatalf("código HTTP %d, se esperaba %d", rec.Code, tc.code)
				}
				var body models.HealthResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("diagnóstico inválido: %v", err)
				}
				if body.Status != tc.status || body.Database.Engine != "MongoDB" {
					t.Errorf("diagnóstico inesperado: %+v", body)
				}
				if strings.Contains(rec.Body.String(), "internal MongoDB error") {
					t.Error("el diagnóstico publicó un error interno de MongoDB")
				}
			})
		}
	}
}

func TestErrorFormat(t *testing.T) {
	for _, tc := range []struct {
		name, method, path string
		code               int
		errCode            string
	}{
		{"ruta inexistente", http.MethodGet, "/api/no-existe", http.StatusNotFound, "not_found"},
		{"método no permitido", http.MethodPost, "/api/health", http.StatusMethodNotAllowed, "method_not_allowed"},
		{"panic recuperado", http.MethodGet, "/panic", http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(newTestRouter(nil), tc.method, tc.path, map[string]string{middleware.RequestIDHeader: "req-abc"})

			if rec.Code != tc.code {
				t.Fatalf("código HTTP %d, se esperaba %d", rec.Code, tc.code)
			}
			var body models.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("el error debe usar el formato estándar: %v (%s)", err, rec.Body.String())
			}
			if body.Error.Code != tc.errCode || body.Error.RequestID != "req-abc" {
				t.Errorf("cuerpo inesperado: %+v", body.Error)
			}
			if strings.Contains(rec.Body.String(), "falla inesperada") {
				t.Error("el mensaje del panic no debe llegar al cliente")
			}
		})
	}
}

func TestCORS(t *testing.T) {
	r := newTestRouter(nil)

	rec := do(r, http.MethodOptions, "/api/health", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": http.MethodGet,
	})
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("el origen configurado debe estar permitido, encabezados: %v", rec.Header())
	}

	rec = do(r, http.MethodGet, "/api/health", map[string]string{"Origin": "http://evil.example"})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("un origen no configurado no debe recibir permiso CORS")
	}
}
