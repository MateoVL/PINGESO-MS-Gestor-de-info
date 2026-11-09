package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/services"
	"github.com/mateo/ms-gestor-de-info/backend/mocks"
)

// mocks
func seedDemoBatch(ctx context.Context, svc *services.BatchService) error {
	var req models.SolicitudLote
	if err := json.Unmarshal(mocks.DirtyBatchJSON, &req); err != nil {
		return fmt.Errorf("leyendo el lote de ejemplo: %w", err)
	}
	_, err := svc.Receive(ctx, services.BatchInput{
		Origen:            req.Origen,
		ReferenciaExterna: req.ReferenciaExterna,
		ClaveIdempotencia: "lote-demo",
		Registros:         req.Registros,
	})
	if err != nil {
		return fmt.Errorf("cargando el lote de ejemplo: %w", err)
	}
	return nil
}
