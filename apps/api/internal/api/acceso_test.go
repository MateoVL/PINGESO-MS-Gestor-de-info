package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/service"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

type testKeyPair struct {
	edPub   ed25519.PublicKey
	edPriv  ed25519.PrivateKey
	rsaPriv *rsa.PrivateKey
}

func newTestKeyPair(t *testing.T) testKeyPair {
	t.Helper()
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generando llave EdDSA: %v", err)
	}
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generando llave RS256: %v", err)
	}
	return testKeyPair{edPub: edPub, edPriv: edPriv, rsaPriv: rsaPriv}
}

func (k testKeyPair) authenticator() *Authenticator {
	return NewKeyfuncAuthenticator(func(token *jwt.Token) (any, error) {
		switch token.Method.Alg() {
		case jwt.SigningMethodEdDSA.Alg():
			return k.edPub, nil
		case jwt.SigningMethodRS256.Alg():
			return &k.rsaPriv.PublicKey, nil
		default:
			return nil, jwt.ErrTokenSignatureInvalid
		}
	})
}

func (k testKeyPair) signEdDSA(t *testing.T, rol string, permisos []string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := authClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-" + rol,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Rol:      rol,
		Permisos: permisos,
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(k.edPriv)
	if err != nil {
		t.Fatalf("firmando JWT EdDSA: %v", err)
	}
	return tok
}

func (k testKeyPair) signRS256(t *testing.T, rol string, permisos []string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := authClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-" + rol,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Rol:      rol,
		Permisos: permisos,
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(k.rsaPriv)
	if err != nil {
		t.Fatalf("firmando JWT RS256: %v", err)
	}
	return tok
}

func newProtectedTestRouter(t *testing.T, auth *Authenticator) (http.Handler, string, string) {
	t.Helper()
	health := service.NewHealthService(
		config.ServiceInfo{Name: "MS Gestor de Información", Environment: "test"},
		func(context.Context) error { return nil },
	)
	batchSvc := service.NewBatchService(store.NewMemoryBatchStore())
	res, err := batchSvc.Receive(context.Background(), service.BatchInput{
		Origen:            "Extractor OCR",
		ReferenciaExterna: "seed-acceso",
		Registros: []store.RegistroEntrada{
			{Folio: "DOC-ACC-1", Nombre: "Ana Pérez", Correo: "ana@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("sembrando lote inicial: %v", err)
	}
	recs, err := batchSvc.ListRecords(context.Background(), store.FiltroRegistros{
		LoteID:     res.Detalle.ID,
		Paginacion: store.Paginacion{Numero: 1, Tamano: 10},
	})
	if err != nil || len(recs.Elementos) == 0 {
		t.Fatalf("obteniendo registro inicial: %v", err)
	}

	router := NewRouter(
		Options{
			AllowedOrigins: []string{"http://localhost:5120"},
			Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
			PublishDocs:    false,
			Authenticator:  auth,
			RequireAuth:    true,
		},
		Dependencies{
			Health:  NewHealthHandler(health),
			Batches: NewBatchHandler(batchSvc),
		},
	)
	return router, res.Detalle.ID, recs.Elementos[0].ID
}

// TestMatrizDeAccesoPorRol verifica la matriz de acceso por rol sobre todos los endpoints
// según AGENTS.md sección 6 (Controles de aplicación).
func TestMatrizDeAccesoPorRol(t *testing.T) {
	keys := newTestKeyPair(t)
	router, loteID, registroID := newProtectedTestRouter(t, keys.authenticator())

	roles := map[string]string{
		"anonimo":          "",
		"sin_permisos":     keys.signEdDSA(t, "invitado", nil, time.Hour),
		"lector_lotes":     keys.signEdDSA(t, "lector_lotes", []string{PermLotesLeer}, time.Hour),
		"lector_registros": keys.signRS256(t, "motor_reportes", []string{PermRegistrosLeer}, time.Hour),
		"operador_ocr":     keys.signEdDSA(t, "extractor_ocr", []string{PermLotesOperar}, time.Hour),
		"administrador":    keys.signRS256(t, "admin", []string{PermLotesAdministrar, PermRegistrosAdministrar}, time.Hour),
	}

	type endpointCase struct {
		nombre   string
		metodo   string
		ruta     string
		cuerpo   string
		esperado map[string]int
	}

	validBatchBody := `{"origen":"Extractor OCR","referencia_externa":"acc-test","registros":[{"folio":"DOC-NEW-1","nombre":"Luis Soto","correo":"luis@example.com"}]}`

	casos := []endpointCase{
		{
			nombre: "GET /healthz (público)",
			metodo: http.MethodGet,
			ruta:   "/healthz",
			esperado: map[string]int{
				"anonimo":          http.StatusOK,
				"sin_permisos":     http.StatusOK,
				"lector_lotes":     http.StatusOK,
				"lector_registros": http.StatusOK,
				"operador_ocr":     http.StatusOK,
				"administrador":    http.StatusOK,
			},
		},
		{
			nombre: "GET /readyz (público)",
			metodo: http.MethodGet,
			ruta:   "/readyz",
			esperado: map[string]int{
				"anonimo":          http.StatusOK,
				"sin_permisos":     http.StatusOK,
				"lector_lotes":     http.StatusOK,
				"lector_registros": http.StatusOK,
				"operador_ocr":     http.StatusOK,
				"administrador":    http.StatusOK,
			},
		},
		{
			nombre: "POST /api/v1/lotes (lotes:operar)",
			metodo: http.MethodPost,
			ruta:   "/api/v1/lotes",
			cuerpo: validBatchBody,
			esperado: map[string]int{
				"anonimo":          http.StatusUnauthorized,
				"sin_permisos":     http.StatusForbidden,
				"lector_lotes":     http.StatusForbidden,
				"lector_registros": http.StatusForbidden,
				"operador_ocr":     http.StatusCreated,
				"administrador":    http.StatusCreated,
			},
		},
		{
			nombre: "GET /api/v1/lotes (lotes:leer)",
			metodo: http.MethodGet,
			ruta:   "/api/v1/lotes",
			esperado: map[string]int{
				"anonimo":          http.StatusUnauthorized,
				"sin_permisos":     http.StatusForbidden,
				"lector_lotes":     http.StatusOK,
				"lector_registros": http.StatusForbidden,
				"operador_ocr":     http.StatusForbidden,
				"administrador":    http.StatusOK,
			},
		},
		{
			nombre: "GET /api/v1/lotes/{id} (lotes:leer)",
			metodo: http.MethodGet,
			ruta:   "/api/v1/lotes/" + loteID,
			esperado: map[string]int{
				"anonimo":          http.StatusUnauthorized,
				"sin_permisos":     http.StatusForbidden,
				"lector_lotes":     http.StatusOK,
				"lector_registros": http.StatusForbidden,
				"operador_ocr":     http.StatusForbidden,
				"administrador":    http.StatusOK,
			},
		},
		{
			nombre: "GET /api/v1/lotes/{id}/rechazos (lotes:leer)",
			metodo: http.MethodGet,
			ruta:   "/api/v1/lotes/" + loteID + "/rechazos",
			esperado: map[string]int{
				"anonimo":          http.StatusUnauthorized,
				"sin_permisos":     http.StatusForbidden,
				"lector_lotes":     http.StatusOK,
				"lector_registros": http.StatusForbidden,
				"operador_ocr":     http.StatusForbidden,
				"administrador":    http.StatusOK,
			},
		},
		{
			nombre: "GET /api/v1/registros (registros:leer)",
			metodo: http.MethodGet,
			ruta:   "/api/v1/registros",
			esperado: map[string]int{
				"anonimo":          http.StatusUnauthorized,
				"sin_permisos":     http.StatusForbidden,
				"lector_lotes":     http.StatusForbidden,
				"lector_registros": http.StatusOK,
				"operador_ocr":     http.StatusForbidden,
				"administrador":    http.StatusOK,
			},
		},
		{
			nombre: "GET /api/v1/registros/{id} (registros:leer)",
			metodo: http.MethodGet,
			ruta:   "/api/v1/registros/" + registroID,
			esperado: map[string]int{
				"anonimo":          http.StatusUnauthorized,
				"sin_permisos":     http.StatusForbidden,
				"lector_lotes":     http.StatusForbidden,
				"lector_registros": http.StatusOK,
				"operador_ocr":     http.StatusForbidden,
				"administrador":    http.StatusOK,
			},
		},
	}

	for _, caso := range casos {
		for rol, token := range roles {
			t.Run(caso.nombre+"/"+rol, func(t *testing.T) {
				var body io.Reader
				if caso.cuerpo != "" {
					body = bytes.NewBufferString(caso.cuerpo)
				}
				req := httptest.NewRequest(caso.metodo, caso.ruta, body)
				if caso.cuerpo != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				if token != "" {
					req.AddCookie(&http.Cookie{
						Name:     "session",
						Value:    token,
						HttpOnly: true,
						Secure:   true,
						SameSite: http.SameSiteLaxMode,
					})
				}

				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				want := caso.esperado[rol]
				if rec.Code != want {
					t.Fatalf("rol=%s %s %s: código %d, se esperaba %d. Cuerpo: %s",
						rol, caso.metodo, caso.ruta, rec.Code, want, rec.Body.String())
				}

				if want == http.StatusUnauthorized || want == http.StatusForbidden {
					if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
						t.Errorf("error %d debe responder application/problem+json, se obtuvo %q", want, ct)
					}
					var problem ProblemDetails
					if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
						t.Fatalf("cuerpo RFC 9457 inválido: %v", err)
					}
					if problem.Status != want || problem.Type == "" || problem.Title == "" || problem.Detail == "" {
						t.Errorf("campos RFC 9457 incompletos: %+v", problem)
					}
				}
			})
		}
	}
}

func TestAutenticacionJWTControlesDeSeguridad(t *testing.T) {
	keys := newTestKeyPair(t)
	router, _, _ := newProtectedTestRouter(t, keys.authenticator())

	t.Run("acepta Bearer token válido para llamadas entre servicios", func(t *testing.T) {
		tok := keys.signEdDSA(t, "motor_reportes", []string{PermRegistrosLeer}, time.Hour)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/registros", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("se esperaba 200 OK con Bearer válido, se obtuvo %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rechaza JWT expirado", func(t *testing.T) {
		tok := keys.signEdDSA(t, "admin", []string{PermLotesAdministrar}, -time.Minute)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lotes", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: tok, HttpOnly: true})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("JWT expirado debe devolver 401, se obtuvo %d", rec.Code)
		}
	})

	t.Run("rechaza algoritmo simétrico HS256", func(t *testing.T) {
		hsToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "atacante",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			Permisos: []string{PermLotesAdministrar},
		}).SignedString([]byte("clave-simetrica-no-permitida"))
		if err != nil {
			t.Fatalf("firmando HS256: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lotes", nil)
		req.Header.Set("Authorization", "Bearer "+hsToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("HS256 debe rechazarse con 401, se obtuvo %d", rec.Code)
		}
	})

	t.Run("rechaza firma con otra llave privada", func(t *testing.T) {
		otherKeys := newTestKeyPair(t)
		tok := otherKeys.signEdDSA(t, "admin", []string{PermLotesAdministrar}, time.Hour)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lotes", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: tok, HttpOnly: true})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("JWT firmado con otra llave debe devolver 401, se obtuvo %d", rec.Code)
		}
	})
}

func TestAuthHelpersAndJWKS(t *testing.T) {
	p := Principal{
		Subject:  "usr-1",
		Rol:      "operador",
		Permisos: []string{PermLotesAdministrar},
	}
	if !p.HasPermission("") {
		t.Error("permiso vacío debe permitirse")
	}
	if !p.HasPermission(PermLotesOperar) {
		t.Error("lotes:administrar debe otorgar lotes:operar")
	}
	if p.HasPermission(PermRegistrosLeer) {
		t.Error("no debe otorgar permiso de otro recurso")
	}

	ctx := WithPrincipal(context.Background(), p)
	got, ok := PrincipalFromContext(ctx)
	if !ok || got.Subject != "usr-1" {
		t.Fatalf("PrincipalFromContext inesperado: %+v, ok=%v", got, ok)
	}
	if _, ok := PrincipalFromContext(context.Background()); ok {
		t.Error("contexto vacío no debe tener Principal")
	}

	if _, err := NewJWKSAuthenticator(context.Background(), "   "); err == nil {
		t.Error("NewJWKSAuthenticator con URL vacía debe fallar")
	}

	var nilAuth *Authenticator
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lotes", nil)
	if _, err := nilAuth.Authenticate(req); err == nil {
		t.Error("Authenticator nil debe rechazar")
	}
	emptyAuth := &Authenticator{}
	if _, err := emptyAuth.Authenticate(req); err == nil {
		t.Error("Authenticator sin keyfunc debe rechazar")
	}

	// Servidor JWKS simulado
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[{"kty":"OKP","crv":"Ed25519","kid":"test-key","x":"11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"}]}`))
	}))
	defer jwksSrv.Close()

	jwksAuth, err := NewJWKSAuthenticator(context.Background(), jwksSrv.URL)
	if err != nil || jwksAuth == nil {
		t.Fatalf("NewJWKSAuthenticator con servidor válido falló: %v", err)
	}

	// Reconexión perezosa cuando keyfunc es nil inicialmente
	lazyAuth := &Authenticator{jwksURL: jwksSrv.URL}
	reqWithCookie := httptest.NewRequest(http.MethodGet, "/api/v1/lotes", nil)
	reqWithCookie.AddCookie(&http.Cookie{Name: "access_token", Value: "token-invalido"})
	if _, err := lazyAuth.Authenticate(reqWithCookie); err == nil {
		t.Error("token inválido debe fallar tras cargar JWKS")
	}

	// Claims alternativos: role, permissions, scope
	keys := newTestKeyPair(t)
	now := time.Now()
	altToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, authClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "svc-alt",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		Role:        "reportes",
		Permissions: []string{PermRegistrosLeer, PermRegistrosLeer, " "},
		Scope:       "lotes:leer registros:leer",
	}).SignedString(keys.edPriv)
	if err != nil {
		t.Fatalf("firmando token alternativo: %v", err)
	}
	reqAlt := httptest.NewRequest(http.MethodGet, "/api/v1/registros", nil)
	reqAlt.AddCookie(&http.Cookie{Name: "access_token", Value: altToken})
	principal, err := keys.authenticator().Authenticate(reqAlt)
	if err != nil {
		t.Fatalf("Authenticate con claims alternativos falló: %v", err)
	}
	if principal.Rol != "reportes" || !principal.HasPermission(PermLotesLeer) || !principal.HasPermission(PermRegistrosLeer) {
		t.Errorf("principal con claims alternativos inesperado: %+v", principal)
	}
}
