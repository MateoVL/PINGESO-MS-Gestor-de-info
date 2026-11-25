package service

import (
	"context"
	"testing"

	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/store"
)

func TestAuditoriaRegistraQuienQueYCuando(t *testing.T) {
	st := store.NewMemoryBatchStore()
	svc := NewBatchService(st)
	ctx := WithActor(WithRequestID(context.Background(), "req-1"), "usuario-42")
	entrada := BatchInput{
		Origen:            "extractor-ocr",
		ClaveIdempotencia: "clave-auditoria",
		Registros: []store.RegistroEntrada{
			{Folio: "a-1", Nombre: "Ana", Correo: "ana@x.cl"},
			{Folio: "", Nombre: "Sin folio", Correo: "s@x.cl"},
		},
	}

	res, err := svc.Receive(ctx, entrada)
	if err != nil {
		t.Fatal(err)
	}
	eventos := st.EventosAuditoria()
	if len(eventos) != 1 {
		t.Fatalf("se esperaba 1 evento de auditoría, hay %d", len(eventos))
	}
	ev := eventos[0]
	if ev.Actor != "usuario-42" || ev.Accion != store.AccionCrearLote || ev.Entidad != store.EntidadLote ||
		ev.EntidadID != res.Detalle.ID || ev.RequestID != "req-1" || !ev.OcurridoEn.Equal(res.Detalle.CreadoEn) {
		t.Errorf("evento inesperado: %+v", ev)
	}
	if ev.Detalle["aceptados"] != 1 || ev.Detalle["rechazados"] != 1 {
		t.Errorf("el detalle debe resumir el lote: %+v", ev.Detalle)
	}

	if _, err := svc.Receive(ctx, entrada); err != nil {
		t.Fatal(err)
	}
	if n := len(st.EventosAuditoria()); n != 1 {
		t.Errorf("el reintento no debe generar otro evento, hay %d", n)
	}
}

func TestAuditoriaSinJWTUsaActorAnonimo(t *testing.T) {
	st := store.NewMemoryBatchStore()
	svc := NewBatchService(st)
	_, err := svc.Receive(context.Background(), BatchInput{Registros: []store.RegistroEntrada{{Folio: "b-1", Nombre: "B", Correo: "b@x.cl"}}})
	if err != nil {
		t.Fatal(err)
	}
	if ev := st.EventosAuditoria(); len(ev) != 1 || ev[0].Actor != actorAnonimo {
		t.Errorf("sin JWT el actor debe ser %q: %+v", actorAnonimo, ev)
	}
}
