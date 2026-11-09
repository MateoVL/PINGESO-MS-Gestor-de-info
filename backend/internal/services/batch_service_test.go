package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/repositories"
)

func newService() *BatchService {
	return NewBatchService(repositories.NewMemoryBatchRepository())
}

func TestReceiveCleansAndValidates(t *testing.T) {
	svc := newService()
	res, err := svc.Receive(context.Background(), BatchInput{Registros: []models.RegistroEntrada{
		{Folio: " doc-001 ", Nombre: "  Camila   Soto ", Correo: " CAMILA@EXAMPLE.COM "},
		{Folio: "doc-002", Nombre: " ", Correo: "prueba@example.com"},
		{Folio: "", Nombre: "Sin Folio", Correo: "no-es-correo"},
		{Folio: " DOC-001", Nombre: "Camila Soto", Correo: "camila@example.com"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	d := res.Detalle
	if d.Recibidos != 4 || d.Aceptados != 1 || d.Rechazados != 3 {
		t.Fatalf("conteos inesperados: %+v", d.Lote)
	}
	got := map[int]string{}
	for _, r := range d.Rechazos {
		got[r.Posicion] = strings.Join(r.Motivos, " | ")
	}
	want := map[int]string{
		2: ReasonMissingName,
		3: ReasonMissingFolio + " | " + ReasonInvalidEmail,
		4: ReasonDuplicateFolio,
	}
	for pos, motivos := range want {
		if got[pos] != motivos {
			t.Errorf("rechazo en posición %d: %q, se esperaba %q", pos, got[pos], motivos)
		}
	}

	list, _ := svc.ListRecords(context.Background(), models.FiltroRegistros{LoteID: d.ID})
	if len(list.Elementos) != 1 {
		t.Fatalf("se esperaba 1 registro guardado, hay %d", len(list.Elementos))
	}
	r := list.Elementos[0]
	if r.Folio != "DOC-001" || r.Nombre != "Camila Soto" || r.Correo != "camila@example.com" || r.Posicion != 1 {
		t.Errorf("registro sin normalizar: %+v", r)
	}
}

func TestReceiveIdempotency(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	in := BatchInput{ClaveIdempotencia: "clave-1", Registros: []models.RegistroEntrada{{Folio: "a", Nombre: "Ana", Correo: "ana@x.cl"}}}

	first, err := svc.Receive(ctx, in)
	if err != nil || first.Repetido {
		t.Fatalf("primera carga: %+v, %v", first, err)
	}
	again, err := svc.Receive(ctx, in)
	if err != nil || !again.Repetido || again.Detalle.ID != first.Detalle.ID {
		t.Fatalf("el reintento debe devolver el mismo lote: %+v, %v", again, err)
	}

	in.Registros = []models.RegistroEntrada{{Folio: "b", Nombre: "Otro", Correo: "otro@x.cl"}}
	if _, err := svc.Receive(ctx, in); !errors.Is(err, models.ErrClaveIdempotenciaConflicto) {
		t.Fatalf("misma clave con otro contenido debe dar conflicto, se obtuvo %v", err)
	}

	list, _ := svc.List(ctx, models.Paginacion{})
	if list.Total != 1 {
		t.Errorf("solo debe existir un lote, hay %d", list.Total)
	}
}

// varios lotes simultaneos compiten por el mismo folio, por lo que solo uno debe aceptarlo
// y los demas deben rechazarlo (con su motivo obvio), sin perder sus otros registros (por transparencia)
func TestReceiveConcurrentBatchesShareFolio(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	const lotes = 20

	var wg sync.WaitGroup
	results := make([]BatchResult, lotes)
	errs := make([]error, lotes)
	for i := range lotes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = svc.Receive(ctx, BatchInput{Registros: []models.RegistroEntrada{
				{Folio: "DOC-900", Nombre: "Compartido", Correo: "c@x.cl"},
				{Folio: fmt.Sprintf("propio-%d", i), Nombre: "Propio", Correo: "p@x.cl"},
			}})
		}()
	}
	wg.Wait()

	winners := 0
	for i, res := range results {
		if errs[i] != nil {
			t.Fatalf("lote %d falló: %v", i, errs[i])
		}
		switch res.Detalle.Aceptados {
		case 2:
			winners++
		case 1:
			if len(res.Detalle.Rechazos) != 1 || !slices.Equal(res.Detalle.Rechazos[0].Motivos, []string{ReasonFolioAlreadyRegistered}) {
				t.Errorf("rechazo inesperado: %+v", res.Detalle.Rechazos)
			}
		default:
			t.Errorf("lote con %d aceptados", res.Detalle.Aceptados)
		}
	}
	if winners != 1 {
		t.Errorf("exactamente un lote debe aceptar DOC-900, lo aceptaron %d", winners)
	}
	shared, _ := svc.ListRecords(ctx, models.FiltroRegistros{Folio: "doc-900"})
	if shared.Total != 1 {
		t.Errorf("DOC-900 debe guardarse una sola vez, hay %d", shared.Total)
	}
	all, _ := svc.ListRecords(ctx, models.FiltroRegistros{Paginacion: models.Paginacion{Tamano: 200}})
	if all.Total != lotes+1 {
		t.Errorf("se esperaban %d registros (1 compartido + %d propios), hay %d", lotes+1, lotes, all.Total)
	}
}

func TestQueriesAndPagination(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	var ids []string
	for i := range 3 {
		res, err := svc.Receive(ctx, BatchInput{Registros: []models.RegistroEntrada{{Folio: fmt.Sprintf("f-%d", i), Nombre: "N", Correo: "n@x.cl"}}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, res.Detalle.ID)
	}

	page, _ := svc.List(ctx, models.Paginacion{Numero: 1, Tamano: 2})
	if page.Total != 3 || len(page.Elementos) != 2 || page.Elementos[0].ID != ids[2] {
		t.Errorf("la primera página debe traer los 2 lotes más recientes: %+v", page)
	}
	last, _ := svc.List(ctx, models.Paginacion{Numero: 3, Tamano: 2})
	if len(last.Elementos) != 0 || last.Elementos == nil {
		t.Errorf("una página vacía debe ser [] y no null: %+v", last)
	}
	huge, _ := svc.List(ctx, models.Paginacion{Tamano: 10_000})
	if huge.TamanoPagina != models.TamanoPaginaMaximo {
		t.Errorf("el tamaño de página debe limitarse a %d, quedó %d", models.TamanoPaginaMaximo, huge.TamanoPagina)
	}

	if _, err := svc.Get(ctx, "no-existe"); !errors.Is(err, models.ErrNoEncontrado) {
		t.Errorf("lote inexistente: %v", err)
	}
	if _, err := svc.ListRejections(ctx, "no-existe"); !errors.Is(err, models.ErrNoEncontrado) {
		t.Errorf("rechazos de un lote inexistente: %v", err)
	}
	if _, err := svc.GetRecord(ctx, "no-existe"); !errors.Is(err, models.ErrNoEncontrado) {
		t.Errorf("registro inexistente: %v", err)
	}
	detail, _ := svc.Get(ctx, ids[0])
	if detail.Rechazos == nil {
		t.Error("un lote sin rechazos debe devolver [] y no null")
	}
}
