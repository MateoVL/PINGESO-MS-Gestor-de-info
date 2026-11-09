package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

func TestHealthCheck(t *testing.T) {
	info := config.ServiceInfo{
		Name: "Pipeline", Environment: "test", Role: "Rol de prueba",
		DependsOn: []string{"Extractor OCR"}, Consumers: []string{"Reportes"}, Capabilities: []string{"Carga"},
	}

	for _, tc := range []struct {
		name      string
		pingErr   error
		status    string
		connected bool
	}{
		{"MongoDB disponible", nil, models.HealthStatusHealthy, true},
		{"MongoDB caído", errors.New("detalle interno de MongoDB"), models.HealthStatusDegraded, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewHealthService(info, func(ctx context.Context) error {
				if _, ok := ctx.Deadline(); !ok {
					t.Error("el ping debe tener un plazo máximo")
				}
				return tc.pingErr
			})
			svc.startedAt = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
			svc.now = func() time.Time { return svc.startedAt.Add(90 * time.Second) }

			got := svc.Check(context.Background())

			if got.Status != tc.status || got.Database.Connected != tc.connected {
				t.Errorf("diagnóstico inesperado: %+v", got)
			}
			if got.Uptime != "1m30s" || got.Timestamp != "2026-01-01T10:01:30Z" {
				t.Errorf("tiempos inesperados: uptime=%s timestamp=%s", got.Uptime, got.Timestamp)
			}
			if got.Role != info.Role || got.LineOfWork.DependsOn[0] != "Extractor OCR" {
				t.Errorf("la identidad del servicio debe venir de la configuración: %+v", got)
			}
			if strings.Contains(got.Database.Message, "detalle interno") {
				t.Error("el diagnóstico no debe exponer el error de MongoDB")
			}
		})
	}
}
