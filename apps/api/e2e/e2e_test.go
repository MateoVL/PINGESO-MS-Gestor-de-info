package e2e

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/api"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/service"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

//go:embed testdata/lote_flow.golden.md
var goldenDoc string

func TestE2EBatchLifecycle(t *testing.T) {
	if !strings.Contains(goldenDoc, "Idempotency-Key: e2e-golden-1") {
		t.Fatal("el archivo golden.md debe documentar el flujo E2E")
	}

	health := service.NewHealthService(config.ServiceInfo{Name: "MS Gestor de Información", Environment: "test"}, func(context.Context) error {
		return nil
	})
	batches := service.NewBatchService(store.NewMemoryBatchStore())
	router := api.NewRouter(
		api.Options{AllowedOrigins: []string{"http://localhost:3000"}, PublishDocs: true},
		api.Dependencies{
			Health:  api.NewHealthHandler(health),
			Batches: api.NewBatchHandler(batches),
		},
	)

	srv := httptest.NewServer(router)
	defer srv.Close()

	// 1. Health check
	resHealth, err := http.Get(srv.URL + "/api/health")
	if err != nil || resHealth.StatusCode != http.StatusOK {
		t.Fatalf("health check falló: %v", err)
	}
	resHealth.Body.Close()

	// 2. Crear lote
	payload := `{"origen":"Extractor OCR","referencia_externa":"e2e-ref","registros":[{"folio":" doc-e2e-1 ","nombre":" Ana  Pérez ","correo":"ANA@EXAMPLE.COM"},{"folio":"doc-e2e-2","nombre":"Luis Soto","correo":"luis@example.com"},{"folio":"doc-e2e-3","nombre":"Mal Correo","correo":"invalido"}]}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/lotes", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "e2e-golden-1")

	resCreate, err := http.DefaultClient.Do(req)
	if err != nil || resCreate.StatusCode != http.StatusCreated {
		t.Fatalf("creación de lote E2E falló: %v", err)
	}
	var detail store.LoteDetalle
	_ = json.NewDecoder(resCreate.Body).Decode(&detail)
	resCreate.Body.Close()

	if detail.Recibidos != 3 || detail.Aceptados != 2 || detail.Rechazados != 1 {
		t.Fatalf("resumen inesperado en E2E: %+v", detail)
	}
}

