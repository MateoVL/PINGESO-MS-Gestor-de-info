package service

import (
	"context"
	"testing"
	"time"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

// fecha de creacion se guarda siempre en UTC
func TestCreadoEnNoDependeDeLaZonaHoraria(t *testing.T) {
	instante := time.Date(2026, 11, 30, 2, 30, 0, 0, time.UTC)
	original := time.Local
	t.Cleanup(func() { time.Local = original })

	for _, zona := range []string{"UTC", "Pacific/Auckland", "America/Santiago"} {
		t.Run(zona, func(t *testing.T) {
			loc, err := time.LoadLocation(zona)
			if err != nil {
				t.Fatalf("zona %s: %v", zona, err)
			}
			time.Local = loc

			st := store.NewMemoryBatchStore()
			svc := NewBatchService(st)
			svc.now = func() time.Time { return instante.In(loc) }

			res, err := svc.Receive(context.Background(), BatchInput{
				Registros: []store.RegistroEntrada{{Folio: "tz-1", Nombre: "Ana", Correo: "ana@x.cl"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			creado := res.Detalle.CreadoEn
			if creado.Location() != time.UTC || !creado.Equal(instante) {
				t.Errorf("creado_en debe ser %s en UTC, quedó %s", instante, creado)
			}
			registros, err := svc.ListRecords(context.Background(), store.FiltroRegistros{})
			if err != nil {
				t.Fatal(err)
			}
			if len(registros.Elementos) != 1 || !registros.Elementos[0].CreadoEn.Equal(instante) {
				t.Errorf("el registro debe heredar la misma fecha: %+v", registros.Elementos)
			}
		})
	}
}
