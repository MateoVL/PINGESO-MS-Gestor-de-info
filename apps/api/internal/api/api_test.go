package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/service"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

func newTestRouter(pingErr error) http.Handler {
	health := service.NewHealthService(config.ServiceInfo{Name: "Pipeline", Environment: "test"},
		func(context.Context) error { return pingErr })

	return NewRouter(
		Options{AllowedOrigins: []string{"http://localhost:3000"}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), PublishDocs: true},
		Dependencies{
			Health:  NewHealthHandler(health),
			Batches: NewBatchHandler(service.NewBatchService(store.NewMemoryBatchStore())),
		},
	)
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
		{"PostgreSQL disponible", nil, http.StatusOK, service.HealthStatusHealthy},
		{"PostgreSQL caído", errors.New("internal PostgreSQL error"), http.StatusServiceUnavailable, service.HealthStatusDegraded},
	} {
		for _, path := range []string{"/readyz", "/health", "/api/health"} {
			t.Run(tc.name+" "+path, func(t *testing.T) {
				rec := do(newTestRouter(tc.pingErr), http.MethodGet, path, nil)

				if rec.Code != tc.code {
					t.Fatalf("código HTTP %d, se esperaba %d", rec.Code, tc.code)
				}
				var body service.HealthResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("diagnóstico inválido: %v", err)
				}
				if body.Status != tc.status || body.Database.Engine != "PostgreSQL" {
					t.Errorf("diagnóstico inesperado: %+v", body)
				}
				if strings.Contains(rec.Body.String(), "internal PostgreSQL error") {
					t.Error("el diagnóstico publicó un error interno de PostgreSQL")
				}
			})
		}

		t.Run(tc.name+" /healthz (vivo)", func(t *testing.T) {
			rec := do(newTestRouter(tc.pingErr), http.MethodGet, "/healthz", nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("/healthz debe responder 200 OK incluso si PostgreSQL falla, se obtuvo %d", rec.Code)
			}
			var body LivenessStatus
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Status != "ok" {
				t.Fatalf("respuesta inesperada en /healthz: %s (%v)", rec.Body.String(), err)
			}
		})
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
			rec := do(newTestRouter(nil), tc.method, tc.path, map[string]string{RequestIDHeader: "req-abc"})

			if rec.Code != tc.code {
				t.Fatalf("código HTTP %d, se esperaba %d", rec.Code, tc.code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
				t.Errorf("Content-Type esperado application/problem+json, se obtuvo %q", ct)
			}
			var body ProblemDetails
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("el error debe usar RFC 9457 (application/problem+json): %v (%s)", err, rec.Body.String())
			}
			if body.Code != tc.errCode || body.Status != tc.code || body.RequestID != "req-abc" || body.Title == "" || body.Type == "" {
				t.Errorf("cuerpo RFC 9457 inesperado: %+v", body)
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

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestRequestIDMiddleware(t *testing.T) {
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
			h := RequestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seen = service.RequestIDFromContext(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.incoming != "" {
				req.Header.Set(RequestIDHeader, tc.incoming)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

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

type cliente struct {
	t      *testing.T
	router http.Handler
}

type respuesta struct {
	estado int
	cuerpo []byte
	json   map[string]any
}

func (c *cliente) pedir(metodo, ruta, contentType string, cuerpo []byte, encabezados map[string]string) respuesta {
	c.t.Helper()
	req := httptest.NewRequest(metodo, "http://localhost:8080"+ruta, bytes.NewReader(cuerpo))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range encabezados {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	resp := rec.Result()
	datos, _ := io.ReadAll(resp.Body)
	r := respuesta{estado: resp.StatusCode, cuerpo: datos}
	_ = json.Unmarshal(datos, &r.json)
	return r
}

func (c *cliente) esperar(r respuesta, estado int, contexto string) {
	c.t.Helper()
	if r.estado != estado {
		c.t.Errorf("%s: código %d, se esperaba %d. Cuerpo: %s", contexto, r.estado, estado, r.cuerpo)
	}
}

func TestEndpoints(t *testing.T) {
	c := &cliente{t: t, router: newTestRouter(nil)}
	const jsonCT = "application/json"

	creado := c.pedir("POST", "/api/v1/lotes", jsonCT, DirtyBatchJSON, map[string]string{"Idempotency-Key": "lote-1"})
	c.esperar(creado, http.StatusCreated, "crear lote JSON")
	idLote, _ := creado.json["id"].(string)
	if creado.json["aceptados"].(float64)+creado.json["rechazados"].(float64) != creado.json["recibidos"].(float64) {
		t.Errorf("aceptados + rechazados debe ser igual a recibidos: %s", creado.cuerpo)
	}

	c.esperar(c.pedir("POST", "/api/v1/lotes", jsonCT, DirtyBatchJSON, map[string]string{"Idempotency-Key": "lote-1"}), http.StatusOK, "reintento idempotente")
	c.esperar(c.pedir("POST", "/api/v1/lotes", jsonCT, []byte(`{"registros":[{"folio":"x"}]}`), map[string]string{"Idempotency-Key": "lote-1"}), http.StatusConflict, "misma clave con otro contenido")
	csv := "correo,folio,confianza,nombre\nana@example.com, csv-1 ,0.9,Ana Rojas\nmal-correo,csv-2,0.4,Luis\n"
	creadoCSV := c.pedir("POST", "/api/v1/lotes?origen=extractor-ocr", "text/csv", []byte(csv), nil)
	c.esperar(creadoCSV, http.StatusCreated, "crear lote CSV")
	if creadoCSV.json["aceptados"] != float64(1) || creadoCSV.json["origen"] != "extractor-ocr" {
		t.Errorf("lote CSV inesperado: %s", creadoCSV.cuerpo)
	}
	c.esperar(c.pedir("POST", "/api/v1/lotes", "text/csv", DirtyBatchCSV, nil), http.StatusCreated, "CSV de mocks")
	c.esperar(c.pedir("GET", "/api/v1/lotes?pagina=1&tamano_pagina=2", "", nil, nil), http.StatusOK, "listar lotes")
	c.esperar(c.pedir("GET", "/api/v1/lotes/"+idLote, "", nil, nil), http.StatusOK, "obtener lote")
	c.esperar(c.pedir("GET", "/api/v1/lotes/"+idLote+"/rechazos", "", nil, nil), http.StatusOK, "rechazos del lote")
	registros := c.pedir("GET", "/api/v1/registros?lote_id="+idLote, "", nil, nil)
	c.esperar(registros, http.StatusOK, "listar registros")
	porFolio := c.pedir("GET", "/api/v1/registros?folio=%20csv-1%20", "", nil, nil)
	if porFolio.json["total"] != float64(1) {
		t.Errorf("el filtro por folio debe normalizarse: %s", porFolio.cuerpo)
	}
	elementos, _ := registros.json["elementos"].([]any)
	if len(elementos) == 0 {
		t.Fatalf("el lote de ejemplo debería tener registros aceptados: %s", registros.cuerpo)
	}
	idRegistro := elementos[0].(map[string]any)["id"].(string)
	c.esperar(c.pedir("GET", "/api/v1/registros/"+idRegistro, "", nil, nil), http.StatusOK, "obtener registro")

	// mocks y docs
	c.esperar(c.pedir("GET", "/api/v1/mocks/datos-sucios?cantidad=5&semilla=7", "", nil, nil), http.StatusOK, "mocks JSON")
	c.esperar(c.pedir("GET", "/api/v1/mocks/datos-sucios?formato=csv&cantidad=3", "", nil, nil), http.StatusOK, "mocks CSV")
	c.esperar(c.pedir("GET", "/api/docs", "", nil, nil), http.StatusOK, "documentación interactiva Huma en /api/docs")
	c.esperar(c.pedir("GET", "/api/openapi.yaml", "", nil, nil), http.StatusOK, "contrato OpenAPI 3.1 en /api/openapi.yaml")

	// errores documentados
	c.esperar(c.pedir("POST", "/api/v1/lotes", jsonCT, []byte(`{"registros":[]}`), nil), http.StatusBadRequest, "lote vacío")
	c.esperar(c.pedir("POST", "/api/v1/lotes", jsonCT, []byte(`{"registros":`), nil), http.StatusBadRequest, "JSON roto")
	c.esperar(c.pedir("POST", "/api/v1/lotes", "text/csv", []byte("folio,nombre\na,b\n"), nil), http.StatusBadRequest, "CSV sin columna correo")
	c.esperar(c.pedir("GET", "/api/v1/lotes/00000000-0000-4000-8000-000000000000", "", nil, nil), http.StatusNotFound, "lote inexistente")
	c.esperar(c.pedir("GET", "/api/v1/registros/00000000-0000-4000-8000-000000000000", "", nil, nil), http.StatusNotFound, "registro inexistente")
	c.esperar(c.pedir("GET", "/api/v1/lotes?tamano_pagina=500", "", nil, nil), http.StatusBadRequest, "tamaño de página fuera de rango")
	c.esperar(c.pedir("GET", "/api/v1/mocks/datos-sucios?formato=xml", "", nil, nil), http.StatusBadRequest, "formato de mock inválido")
	c.esperar(c.pedir("GET", "/api/health", "", nil, nil), http.StatusOK, "salud")
}

func TestErroresFueraDelContrato(t *testing.T) {
	r := newTestRouter(nil)
	for _, tc := range []struct {
		nombre, metodo, ruta, contentType, cuerpo string
		estado                                    int
		codigo                                    string
	}{
		{"Content-Type no soportado", "POST", "/api/v1/lotes", "text/plain", "hola", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"id que no es UUID", "GET", "/api/v1/lotes/123", "", "", http.StatusBadRequest, "invalid_data"},
		{"cuerpo de más de 5 MB", "POST", "/api/v1/lotes", "text/csv", "folio,nombre,correo\n" + strings.Repeat("x,y,z@z.cl\n", 500_000), http.StatusRequestEntityTooLarge, "payload_too_large"},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			req := httptest.NewRequest(tc.metodo, tc.ruta, strings.NewReader(tc.cuerpo))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.estado || !strings.Contains(rec.Body.String(), `"`+tc.codigo+`"`) {
				t.Errorf("se esperaba %d %s, se obtuvo %d: %s", tc.estado, tc.codigo, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMocksYDocsSoloEnDev(t *testing.T) {
	rutaMocks := "/api/v1/mocks/datos-sucios"
	if rec := do(newTestRouter(nil), http.MethodGet, rutaMocks, nil); rec.Code != http.StatusOK {
		t.Errorf("%s debería publicarse en dev, código %d", rutaMocks, rec.Code)
	}
	prodRouter := NewRouter(Options{AllowedOrigins: []string{"http://localhost:3000"}, PublishDocs: false}, Dependencies{})
	if rec := do(prodRouter, http.MethodGet, rutaMocks, nil); rec.Code != http.StatusNotFound {
		t.Errorf("%s no debería publicarse en producción, código %d", rutaMocks, rec.Code)
	}
	if rec := do(prodRouter, http.MethodGet, "/api/docs", nil); rec.Code != http.StatusNotFound {
		t.Errorf("/api/docs no debería publicarse fuera de dev, código %d", rec.Code)
	}
}

func TestGenerateDirtyIsDeterministic(t *testing.T) {
	a, b := GenerateDirty(50, 9), GenerateDirty(50, 9)
	if !reflect.DeepEqual(a, b) {
		t.Error("con la misma semilla se deben generar los mismos datos")
	}
	if len(a) != 50 {
		t.Errorf("se esperaban 50 registros, hay %d", len(a))
	}
}

func TestStaticMockFilesAreValid(t *testing.T) {
	var req store.SolicitudLote
	if err := json.Unmarshal(DirtyBatchJSON, &req); err != nil || len(req.Registros) == 0 {
		t.Fatalf("lote-sucio.json debe ser un lote válido: %v", err)
	}
	csvRecords, err := parseCSV(bytes.NewReader(DirtyBatchCSV))
	if err != nil {
		t.Fatalf("lote-sucio.csv debe ser un CSV válido: %v", err)
	}
	if !reflect.DeepEqual(req.Registros, csvRecords) {
		t.Error("lote-sucio.csv debe tener los mismos registros que lote-sucio.json")
	}
}
