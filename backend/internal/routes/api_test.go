package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mateo/ms-gestor-de-info/backend/mocks"
)

type cliente struct {
	t       *testing.T
	router  http.Handler
	validar func(req *http.Request, resp *http.Response, validarPeticion bool)
}

type respuesta struct {
	estado int
	cuerpo []byte
	json   map[string]any
}

func (c *cliente) pedir(metodo, ruta, contentType string, cuerpo []byte, encabezados map[string]string) respuesta {
	c.t.Helper()
	return c.ejecutar(metodo, ruta, contentType, cuerpo, encabezados, true)
}

// envia una peticion que incumple el contrato a proposito y valida solo que la respuesta sea el error documentado
func (c *cliente) pedirInvalida(metodo, ruta, contentType string, cuerpo []byte) respuesta {
	c.t.Helper()
	return c.ejecutar(metodo, ruta, contentType, cuerpo, nil, false)
}

func (c *cliente) ejecutar(metodo, ruta, contentType string, cuerpo []byte, encabezados map[string]string, validarPeticion bool) respuesta {
	c.t.Helper()
	nueva := func() *http.Request {
		req := httptest.NewRequest(metodo, "http://localhost:8080"+ruta, bytes.NewReader(cuerpo))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for k, v := range encabezados {
			req.Header.Set(k, v)
		}
		return req
	}

	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, nueva())
	resp := rec.Result()
	datos, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(datos))

	if c.validar != nil {
		c.validar(nueva(), resp, validarPeticion)
	}

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
	recorrerEndpoints(&cliente{t: t, router: newTestRouter(nil)})
}

// recorre todos los endpoints con casos validos y errores esperados
func recorrerEndpoints(c *cliente) {
	t := c.t
	t.Helper()
	const jsonCT = "application/json"

	creado := c.pedir("POST", "/api/v1/lotes", jsonCT, mocks.DirtyBatchJSON, map[string]string{"Idempotency-Key": "lote-1"})
	c.esperar(creado, http.StatusCreated, "crear lote JSON")
	idLote, _ := creado.json["id"].(string)
	if creado.json["aceptados"].(float64)+creado.json["rechazados"].(float64) != creado.json["recibidos"].(float64) {
		t.Errorf("aceptados + rechazados debe ser igual a recibidos: %s", creado.cuerpo)
	}

	c.esperar(c.pedir("POST", "/api/v1/lotes", jsonCT, mocks.DirtyBatchJSON, map[string]string{"Idempotency-Key": "lote-1"}), http.StatusOK, "reintento idempotente")
	c.esperar(c.pedir("POST", "/api/v1/lotes", jsonCT, []byte(`{"registros":[{"folio":"x"}]}`), map[string]string{"Idempotency-Key": "lote-1"}), http.StatusConflict, "misma clave con otro contenido")
	csv := "correo,folio,confianza,nombre\nana@example.com, csv-1 ,0.9,Ana Rojas\nmal-correo,csv-2,0.4,Luis\n"
	creadoCSV := c.pedir("POST", "/api/v1/lotes?origen=extractor-ocr", "text/csv", []byte(csv), nil)
	c.esperar(creadoCSV, http.StatusCreated, "crear lote CSV")
	if creadoCSV.json["aceptados"] != float64(1) || creadoCSV.json["origen"] != "extractor-ocr" {
		t.Errorf("lote CSV inesperado: %s", creadoCSV.cuerpo)
	}
	c.esperar(c.pedir("POST", "/api/v1/lotes", "text/csv", mocks.DirtyBatchCSV, nil), http.StatusCreated, "CSV de mocks")
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

	// mocks
	c.esperar(c.pedir("GET", "/api/v1/mocks/datos-sucios?cantidad=5&semilla=7", "", nil, nil), http.StatusOK, "mocks JSON")
	c.esperar(c.pedir("GET", "/api/v1/mocks/datos-sucios?formato=csv&cantidad=3", "", nil, nil), http.StatusOK, "mocks CSV")

	// errores documentados
	c.esperar(c.pedirInvalida("POST", "/api/v1/lotes", jsonCT, []byte(`{"registros":[]}`)), http.StatusBadRequest, "lote vacío")
	c.esperar(c.pedirInvalida("POST", "/api/v1/lotes", jsonCT, []byte(`{"registros":`)), http.StatusBadRequest, "JSON roto")
	c.esperar(c.pedir("POST", "/api/v1/lotes", "text/csv", []byte("folio,nombre\na,b\n"), nil), http.StatusBadRequest, "CSV sin columna correo")
	c.esperar(c.pedir("GET", "/api/v1/lotes/00000000-0000-4000-8000-000000000000", "", nil, nil), http.StatusNotFound, "lote inexistente")
	c.esperar(c.pedir("GET", "/api/v1/registros/00000000-0000-4000-8000-000000000000", "", nil, nil), http.StatusNotFound, "registro inexistente")
	c.esperar(c.pedirInvalida("GET", "/api/v1/lotes?tamano_pagina=500", "", nil), http.StatusBadRequest, "tamaño de página fuera de rango")
	c.esperar(c.pedirInvalida("GET", "/api/v1/mocks/datos-sucios?formato=xml", "", nil), http.StatusBadRequest, "formato de mock inválido")
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

func TestMocksSoloFueraDeProduccion(t *testing.T) {
	ruta := "/api/v1/mocks/datos-sucios"
	if rec := do(newTestRouter(nil), http.MethodGet, ruta, nil); rec.Code != http.StatusOK {
		t.Errorf("%s debería publicarse fuera de producción, código %d", ruta, rec.Code)
	}
	if rec := do(NewRouter(Options{AllowedOrigins: []string{"http://localhost:3000"}}, Dependencies{}), http.MethodGet, ruta, nil); rec.Code != http.StatusNotFound {
		t.Errorf("%s no debería publicarse en producción, código %d", ruta, rec.Code)
	}
}
