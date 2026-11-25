package e2e

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/api"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/config"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/service"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

//go:embed testdata/lote_flow.golden.md
var goldenDoc string

func signE2EToken(t *testing.T, priv ed25519.PrivateKey, rol string, permisos []string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      "e2e-" + rol,
		"rol":      rol,
		"permisos": permisos,
		"iat":      now.Unix(),
		"exp":      now.Add(time.Hour).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(priv)
	if err != nil {
		t.Fatalf("firmando token E2E: %v", err)
	}
	return tok
}

func TestE2EBatchLifecycle(t *testing.T) {
	if !strings.Contains(goldenDoc, "Idempotency-Key: e2e-golden-1") {
		t.Fatal("el archivo golden.md debe documentar el flujo E2E")
	}

	t.Setenv("API_PORT", "8080")
	t.Setenv("API_ENV", "dev")
	t.Setenv("DATABASE_URL", "postgres://gestor_user:dev-gestor-password@postgres:5432/gestor_info_db?sslmode=disable")
	t.Setenv("AUTH_JWKS_URL", "http://localhost:5101/api/v1/auth/jwks")
	t.Setenv("FRONTEND_URL", "http://localhost:5120")
	t.Setenv("TZ", "America/Santiago")
	t.Setenv("SERVICE_ROLE", "Gestor de Información")
	t.Setenv("SERVICE_DEPENDS_ON", "Extractor OCR")
	t.Setenv("SERVICE_CONSUMERS", "Motor de Reportes")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load falló en E2E: %v", err)
	}

	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generando llave EdDSA en E2E: %v", err)
	}
	auth := api.NewKeyfuncAuthenticator(func(token *jwt.Token) (any, error) {
		return edPub, nil
	})
	adminToken := signE2EToken(t, edPriv, "admin", []string{api.PermLotesAdministrar, api.PermRegistrosAdministrar})
	readerToken := signE2EToken(t, edPriv, "lector", []string{api.PermRegistrosLeer})

	health := service.NewHealthService(cfg.Service, func(context.Context) error {
		return nil
	})
	batches := service.NewBatchService(store.NewMemoryBatchStore())
	router := api.NewRouter(
		api.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
			PublishDocs:    true,
			Authenticator:  auth,
			RequireAuth:    true,
		},
		api.Dependencies{
			Health:  api.NewHealthHandler(health),
			Batches: api.NewBatchHandler(batches),
		},
	)

	srv := httptest.NewServer(router)
	defer srv.Close()

	httpClient := service.NewHTTPClient()
	doReq := func(method, path, contentType, body, token string, headers map[string]string) (*http.Response, []byte) {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}
		ctx := service.WithRequestID(context.Background(), "e2e-req-001")
		req, err := http.NewRequestWithContext(ctx, method, srv.URL+path, reader)
		if err != nil {
			t.Fatalf("creando request %s %s: %v", method, path, err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "session", Value: token, HttpOnly: true})
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("ejecutando %s %s: %v", method, path, err)
		}
		raw, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return res, raw
	}

	// 1. Liveness, Readiness y Health checks
	for _, path := range []string{"/healthz", "/readyz", "/api/health"} {
		res, _ := doReq(http.MethodGet, path, "", "", "", nil)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("diagnóstico %s falló con código %d", path, res.StatusCode)
		}
	}

	// 2. Rechazo 401 sin token y 403 sin permiso (formato RFC 9457)
	res401, raw401 := doReq(http.MethodGet, "/api/v1/lotes", "", "", "", nil)
	if res401.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(res401.Header.Get("Content-Type"), "application/problem+json") {
		t.Fatalf("se esperaba 401 RFC 9457, se obtuvo %d (%s)", res401.StatusCode, raw401)
	}
	res403, _ := doReq(http.MethodPost, "/api/v1/lotes", "application/json", `{"registros":[{"folio":"a","nombre":"b","correo":"c@d.cl"}]}`, readerToken, nil)
	if res403.StatusCode != http.StatusForbidden {
		t.Fatalf("se esperaba 403 con token sin permiso lotes:operar, se obtuvo %d", res403.StatusCode)
	}

	// 3. Crear lote JSON e idempotencia
	payload := `{"origen":"Extractor OCR","referencia_externa":"e2e-ref","registros":[{"folio":" doc-e2e-1 ","nombre":" Ana  Pérez ","correo":"ANA@EXAMPLE.COM"},{"folio":"doc-e2e-2","nombre":"Luis Soto","correo":"luis@example.com"},{"folio":"doc-e2e-3","nombre":"Mal Correo","correo":"invalido"}]}`
	resCreate, rawCreate := doReq(http.MethodPost, "/api/v1/lotes", "application/json", payload, adminToken, map[string]string{
		"Idempotency-Key": "e2e-golden-1",
	})
	if resCreate.StatusCode != http.StatusCreated {
		t.Fatalf("creación de lote E2E falló (%d): %s", resCreate.StatusCode, rawCreate)
	}
	var detail store.LoteDetalle
	_ = json.Unmarshal(rawCreate, &detail)
	if detail.Recibidos != 3 || detail.Aceptados != 2 || detail.Rechazados != 1 {
		t.Fatalf("resumen inesperado en E2E: %+v", detail)
	}

	// Reintento idempotente (200 OK) y conflicto (409 Conflict)
	resReplay, _ := doReq(http.MethodPost, "/api/v1/lotes", "application/json", payload, adminToken, map[string]string{
		"Idempotency-Key": "e2e-golden-1",
	})
	if resReplay.StatusCode != http.StatusOK {
		t.Fatalf("reintento idempotente debía responder 200 OK, se obtuvo %d", resReplay.StatusCode)
	}
	resConflict, _ := doReq(http.MethodPost, "/api/v1/lotes", "application/json", `{"origen":"Otro","registros":[{"folio":"doc-x","nombre":"X","correo":"x@x.cl"}]}`, adminToken, map[string]string{
		"Idempotency-Key": "e2e-golden-1",
	})
	if resConflict.StatusCode != http.StatusConflict {
		t.Fatalf("conflicto de idempotencia debía responder 409, se obtuvo %d", resConflict.StatusCode)
	}

	// 4. Crear lote CSV (con colisión de folio doc-e2e-1)
	csvPayload := "folio,nombre,correo\ndoc-e2e-1,Ana Duplicada,ana@example.com\ndoc-e2e-4,Pedro Mora,pedro@example.com\n"
	resCSV, _ := doReq(http.MethodPost, "/api/v1/lotes?origen=csv-ocr", "text/csv", csvPayload, adminToken, nil)
	if resCSV.StatusCode != http.StatusCreated {
		t.Fatalf("carga CSV E2E falló con código %d", resCSV.StatusCode)
	}

	// 5. Consultas de lotes, detalle, rechazos y registros
	resList, _ := doReq(http.MethodGet, "/api/v1/lotes?pagina=1&tamano_pagina=10", "", "", adminToken, nil)
	if resList.StatusCode != http.StatusOK {
		t.Fatalf("listado de lotes falló: %d", resList.StatusCode)
	}
	resGet, _ := doReq(http.MethodGet, "/api/v1/lotes/"+detail.ID, "", "", adminToken, nil)
	if resGet.StatusCode != http.StatusOK {
		t.Fatalf("detalle de lote falló: %d", resGet.StatusCode)
	}
	resRej, _ := doReq(http.MethodGet, "/api/v1/lotes/"+detail.ID+"/rechazos", "", "", adminToken, nil)
	if resRej.StatusCode != http.StatusOK {
		t.Fatalf("rechazos de lote falló: %d", resRej.StatusCode)
	}
	resRecs, rawRecs := doReq(http.MethodGet, "/api/v1/registros?lote_id="+detail.ID+"&folio=doc-e2e-1", "", "", readerToken, nil)
	if resRecs.StatusCode != http.StatusOK {
		t.Fatalf("listado de registros falló: %d", resRecs.StatusCode)
	}
	var listRecs store.ListaRegistros
	_ = json.Unmarshal(rawRecs, &listRecs)
	if len(listRecs.Elementos) != 1 {
		t.Fatalf("se esperaba 1 registro filtrado por folio, se obtuvo %d", len(listRecs.Elementos))
	}
	resRec, _ := doReq(http.MethodGet, "/api/v1/registros/"+listRecs.Elementos[0].ID, "", "", readerToken, nil)
	if resRec.StatusCode != http.StatusOK {
		t.Fatalf("detalle de registro falló: %d", resRec.StatusCode)
	}

	// 6. Mocks, docs y validaciones de entrada
	resMockJSON, _ := doReq(http.MethodGet, "/api/v1/mocks/datos-sucios?cantidad=4&semilla=42", "", "", "", nil)
	resMockCSV, _ := doReq(http.MethodGet, "/api/v1/mocks/datos-sucios?cantidad=4&formato=csv&semilla=42", "", "", "", nil)
	resDocs, _ := doReq(http.MethodGet, "/api/docs", "", "", "", nil)
	resOpenAPI, _ := doReq(http.MethodGet, "/api/openapi.yaml", "", "", "", nil)
	if resMockJSON.StatusCode != http.StatusOK || resMockCSV.StatusCode != http.StatusOK || resDocs.StatusCode != http.StatusOK || resOpenAPI.StatusCode != http.StatusOK {
		t.Fatalf("endpoints de mocks/docs fallaron: json=%d csv=%d docs=%d openapi=%d", resMockJSON.StatusCode, resMockCSV.StatusCode, resDocs.StatusCode, resOpenAPI.StatusCode)
	}

	// 7. Casos de validación, CORS, errores RFC 9457 y recuperación de panic
	resCORS, _ := doReq(http.MethodOptions, "/api/v1/lotes", "", "", "", map[string]string{
		"Origin":                        "http://localhost:5120",
		"Access-Control-Request-Method": http.MethodPost,
	})
	if resCORS.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight CORS falló: %d", resCORS.StatusCode)
	}
	for _, errCase := range []struct {
		method, path, ct, body string
		want                   int
	}{
		{http.MethodGet, "/panic", "", "", http.StatusInternalServerError},
		{http.MethodPost, "/healthz", "", "", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/v1/no-existe", "", "", http.StatusNotFound},
		{http.MethodGet, "/api/v1/lotes/no-uuid", "", "", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/lotes/00000000-0000-4000-8000-000000000000", "", "", http.StatusNotFound},
		{http.MethodGet, "/api/v1/lotes/no-uuid/rechazos", "", "", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/registros/no-uuid", "", "", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/registros?lote_id=no-uuid", "", "", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/lotes?pagina=0&tamano_pagina=999", "", "", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/lotes", "text/plain", "invalido", http.StatusUnsupportedMediaType},
		{http.MethodPost, "/api/v1/lotes", "application/json", `{"registros":[]}`, http.StatusBadRequest},
		{http.MethodGet, "/api/v1/mocks/datos-sucios?cantidad=0&formato=xml&semilla=-1", "", "", http.StatusBadRequest},
	} {
		res, _ := doReq(errCase.method, errCase.path, errCase.ct, errCase.body, adminToken, nil)
		if res.StatusCode != errCase.want {
			t.Fatalf("%s %s: código %d, se esperaba %d", errCase.method, errCase.path, res.StatusCode, errCase.want)
		}
	}

	// 8. Ejecutar el binario compilado con `go build -cover` (.cache/api-cover) con GOCOVERDIR
	// según AGENTS.md sección 6 y 7 para recolectar su cobertura de proceso real.
	coverBin := filepath.Join("..", ".cache", "api-cover")
	if _, err := os.Stat(coverBin); err == nil {
		covDir := filepath.Join("..", ".cache", "covdata")
		_ = os.MkdirAll(covDir, 0o755)
		cmd := exec.Command(coverBin, "-healthcheck")
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+covDir)
		_ = cmd.Run()
	}
}
