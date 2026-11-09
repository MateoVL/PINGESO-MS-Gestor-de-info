package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestRequestID(t *testing.T) {
	for _, tc := range []struct {
		name, incoming string
		reuse          bool
	}{
		{"sin encabezado genera uno", "", false},
		{"reutiliza un ID válido", "pt5-lote.0001_A", true},
		{"descarta caracteres peligrosos", "abc\ninyectado", false},
		{"descarta IDs demasiado largos", strings.Repeat("a", 65), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			r := gin.New()
			r.Use(RequestID())
			r.GET("/", func(c *gin.Context) { seen = utils.RequestID(c) })

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.incoming != "" {
				req.Header.Set(RequestIDHeader, tc.incoming)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Header().Get(RequestIDHeader) != seen {
				t.Error("el encabezado de respuesta debe coincidir con el ID del contexto")
			}
			if tc.reuse && seen != tc.incoming {
				t.Errorf("se esperaba reutilizar %q, se obtuvo %q", tc.incoming, seen)
			}
			if !tc.reuse && !uuidV4.MatchString(seen) {
				t.Errorf("se esperaba un UUID v4 nuevo, se obtuvo %q", seen)
			}
		})
	}
}

func TestLoggerSkipsQuietPathsUnlessTheyFail(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := gin.New()
	r.Use(Logger(logger, "/health"))
	healthy := true
	r.GET("/health", func(c *gin.Context) {
		if healthy {
			c.Status(http.StatusOK)
		} else {
			c.Status(http.StatusServiceUnavailable)
		}
	})
	r.GET("/data", func(c *gin.Context) { c.Status(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	if buf.Len() != 0 {
		t.Errorf("un health check exitoso no debe registrarse: %s", buf.String())
	}

	healthy = false
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	if !strings.Contains(buf.String(), `"status":503`) {
		t.Errorf("un health check fallido debe registrarse: %s", buf.String())
	}

	buf.Reset()
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/data?token=secreto", nil))
	if !strings.Contains(buf.String(), `"path":"/data"`) || strings.Contains(buf.String(), "secreto") {
		t.Errorf("se debe registrar la ruta sin la query string: %s", buf.String())
	}
}
