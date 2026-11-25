package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/repository"
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const postgresImageDigest = "postgres:17.11@sha256:ff87138949ad9a46f32ff23bd6852b52ce024a13dffe9b4087be98292cdd3b4f"

type fakeQuerier struct {
	lotes      map[string]repository.Lote
	byKey      map[string]string
	registros  []repository.Registro
	rechazos   map[string][]repository.Rechazo
	folios     map[string]bool
	forcePgDup bool
	auditoria  []repository.CreateEventoAuditoriaParams
}

func (f *fakeQuerier) CreateEventoAuditoria(_ context.Context, arg repository.CreateEventoAuditoriaParams) error {
	f.auditoria = append(f.auditoria, arg)
	return nil
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		lotes:    map[string]repository.Lote{},
		byKey:    map[string]string{},
		rechazos: map[string][]repository.Rechazo{},
		folios:   map[string]bool{},
	}
}

func (f *fakeQuerier) CountLotes(_ context.Context) (int64, error) {
	return int64(len(f.lotes)), nil
}

func (f *fakeQuerier) CountRegistros(_ context.Context, arg repository.CountRegistrosParams) (int64, error) {
	var n int64
	for _, r := range f.registros {
		if (!arg.Folio.Valid || r.Folio == arg.Folio.String) && (!arg.LoteID.Valid || r.LoteID == arg.LoteID) {
			n++
		}
	}
	return n, nil
}

func (f *fakeQuerier) CreateLote(_ context.Context, arg repository.CreateLoteParams) (repository.Lote, error) {
	if arg.ClaveIdempotencia.Valid {
		if _, exists := f.byKey[arg.ClaveIdempotencia.String]; exists {
			return repository.Lote{}, &pgconn.PgError{Code: pgUniqueViolation}
		}
	}
	l := repository.Lote{
		ID:                arg.ID,
		Origen:            arg.Origen,
		ReferenciaExterna: arg.ReferenciaExterna,
		Recibidos:         arg.Recibidos,
		Aceptados:         arg.Aceptados,
		Rechazados:        arg.Rechazados,
		ClaveIdempotencia: arg.ClaveIdempotencia,
		HuellaContenido:   arg.HuellaContenido,
		CreadoEn:          arg.CreadoEn,
	}
	id := formatPGUUID(arg.ID)
	f.lotes[id] = l
	if arg.ClaveIdempotencia.Valid {
		f.byKey[arg.ClaveIdempotencia.String] = id
	}
	return l, nil
}

func (f *fakeQuerier) CreateRechazo(_ context.Context, arg repository.CreateRechazoParams) (repository.Rechazo, error) {
	id := formatPGUUID(arg.LoteID)
	rej := repository.Rechazo{
		ID:       int64(len(f.rechazos[id]) + 1),
		LoteID:   arg.LoteID,
		Posicion: arg.Posicion,
		Folio:    arg.Folio,
		Motivos:  arg.Motivos,
	}
	f.rechazos[id] = append(f.rechazos[id], rej)
	return rej, nil
}

func (f *fakeQuerier) CreateRegistro(_ context.Context, arg repository.CreateRegistroParams) (repository.Registro, error) {
	if f.forcePgDup || f.folios[arg.Folio] {
		return repository.Registro{}, &pgconn.PgError{Code: pgUniqueViolation}
	}
	reg := repository.Registro{
		ID:       arg.ID,
		LoteID:   arg.LoteID,
		Posicion: arg.Posicion,
		Folio:    arg.Folio,
		Nombre:   arg.Nombre,
		Correo:   arg.Correo,
		CreadoEn: arg.CreadoEn,
	}
	f.folios[arg.Folio] = true
	f.registros = append(f.registros, reg)
	return reg, nil
}

func (f *fakeQuerier) FindExistingFolios(_ context.Context, folios []string) ([]string, error) {
	if f.forcePgDup {
		return nil, nil
	}
	var out []string
	for _, folio := range folios {
		if f.folios[folio] {
			out = append(out, folio)
		}
	}
	return out, nil
}

func (f *fakeQuerier) GetLoteByID(_ context.Context, id pgtype.UUID) (repository.Lote, error) {
	l, ok := f.lotes[formatPGUUID(id)]
	if !ok {
		return repository.Lote{}, pgx.ErrNoRows
	}
	return l, nil
}

func (f *fakeQuerier) GetLoteByIdempotencyKey(_ context.Context, key pgtype.Text) (repository.Lote, error) {
	id, ok := f.byKey[key.String]
	if !ok {
		return repository.Lote{}, pgx.ErrNoRows
	}
	return f.lotes[id], nil
}

func (f *fakeQuerier) GetRegistroByID(_ context.Context, id pgtype.UUID) (repository.Registro, error) {
	for _, r := range f.registros {
		if r.ID == id {
			return r, nil
		}
	}
	return repository.Registro{}, pgx.ErrNoRows
}

func (f *fakeQuerier) ListLotes(_ context.Context, _ repository.ListLotesParams) ([]repository.Lote, error) {
	out := make([]repository.Lote, 0, len(f.lotes))
	for _, l := range f.lotes {
		out = append(out, l)
	}
	return out, nil
}

func (f *fakeQuerier) ListRechazosByLoteID(_ context.Context, loteID pgtype.UUID) ([]repository.Rechazo, error) {
	return slices.Clone(f.rechazos[formatPGUUID(loteID)]), nil
}

func (f *fakeQuerier) ListRegistros(_ context.Context, arg repository.ListRegistrosParams) ([]repository.Registro, error) {
	var out []repository.Registro
	for _, r := range f.registros {
		if (!arg.Folio.Valid || r.Folio == arg.Folio.String) && (!arg.LoteID.Valid || r.LoteID == arg.LoteID) {
			out = append(out, r)
		}
	}
	return out, nil
}

func TestUUIDHelpersAndMemoryStore(t *testing.T) {
	const sampleID = "550e8400-e29b-41d4-a716-446655440000"
	uid, err := parsePGUUID(sampleID)
	if err != nil {
		t.Fatalf("parsePGUUID falló: %v", err)
	}
	if got := formatPGUUID(uid); got != sampleID {
		t.Fatalf("formatPGUUID = %q, se esperaba %q", got, sampleID)
	}
	if got := formatPGUUID(pgtype.UUID{}); got != "" {
		t.Fatalf("formatPGUUID vacío = %q, se esperaba cadena vacía", got)
	}
	if _, err := parsePGUUID("no-es-uuid"); err == nil {
		t.Fatal("parsePGUUID debía fallar con un UUID inválido")
	}

	dupErr := &ErrFoliosDuplicados{Folios: []string{"A", "B"}}
	if dupErr.Error() == "" {
		t.Fatal("ErrFoliosDuplicados.Error() no debe ser vacío")
	}

	mem := NewMemoryBatchStore()
	ctx := context.Background()
	now := time.Now().UTC()
	nb := NuevoLote{
		Lote: Lote{
			ID:                sampleID,
			Origen:            "OCR",
			ReferenciaExterna: "ref-1",
			Recibidos:         2,
			Aceptados:         1,
			Rechazados:        1,
			CreadoEn:          now,
			ClaveIdempotencia: "key-1",
			HuellaContenido:   "hash-1",
		},
		Registros: []Registro{
			{ID: "660e8400-e29b-41d4-a716-446655440000", LoteID: sampleID, Posicion: 1, Folio: "F-1", Nombre: "Ana", Correo: "ana@example.com", CreadoEn: now},
		},
		Rechazos: []Rechazo{
			{LoteID: sampleID, Posicion: 2, Folio: "F-2", Motivos: []string{"correo_invalido"}},
		},
	}
	if err := mem.Create(ctx, nb); err != nil {
		t.Fatalf("MemoryBatchStore.Create falló: %v", err)
	}
	if err := mem.Create(ctx, nb); !errors.Is(err, ErrClaveIdempotenciaEnUso) {
		t.Fatalf("se esperaba ErrClaveIdempotenciaEnUso, se obtuvo %v", err)
	}
	nbDupFolio := nb
	nbDupFolio.Lote.ClaveIdempotencia = "key-2"
	var dupFolios *ErrFoliosDuplicados
	if err := mem.Create(ctx, nbDupFolio); !errors.As(err, &dupFolios) {
		t.Fatalf("se esperaba ErrFoliosDuplicados, se obtuvo %v", err)
	}
	if _, err := mem.FindByID(ctx, sampleID); err != nil {
		t.Fatalf("FindByID falló: %v", err)
	}
	if _, err := mem.FindByID(ctx, "no-existe"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("se esperaba ErrNoEncontrado, se obtuvo %v", err)
	}
	if _, err := mem.FindByIdempotencyKey(ctx, "key-1"); err != nil {
		t.Fatalf("FindByIdempotencyKey falló: %v", err)
	}
	if _, err := mem.FindByIdempotencyKey(ctx, "no-existe"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("se esperaba ErrNoEncontrado en FindByIdempotencyKey, se obtuvo %v", err)
	}
	if list, total, err := mem.List(ctx, Paginacion{Numero: 1, Tamano: 10}); err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List inesperado: %v, total=%d", err, total)
	}
	if rejs, err := mem.ListRejections(ctx, sampleID); err != nil || len(rejs) != 1 {
		t.Fatalf("ListRejections inesperado: %v, len=%d", err, len(rejs))
	}
	if recs, total, err := mem.ListRecords(ctx, FiltroRegistros{Folio: "F-1", LoteID: sampleID, Paginacion: Paginacion{Numero: 1, Tamano: 10}}); err != nil || total != 1 || len(recs) != 1 {
		t.Fatalf("ListRecords inesperado: %v, total=%d", err, total)
	}
	if _, err := mem.FindRecordByID(ctx, "660e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatalf("FindRecordByID falló: %v", err)
	}
	if _, err := mem.FindRecordByID(ctx, "no-existe"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("se esperaba ErrNoEncontrado en FindRecordByID, se obtuvo %v", err)
	}
}

func TestPostgresBatchStoreWithQuerier(t *testing.T) {
	ctx := context.Background()
	fq := newFakeQuerier()
	st := newPostgresBatchStoreWithQuerier(nil, fq, nil)

	const loteID = "550e8400-e29b-41d4-a716-446655440000"
	const regID = "660e8400-e29b-41d4-a716-446655440000"
	now := time.Now().UTC().Truncate(time.Second)

	nb := NuevoLote{
		Lote: Lote{
			ID:                loteID,
			Origen:            "Extractor OCR",
			ReferenciaExterna: "ref-pg-1",
			Recibidos:         2,
			Aceptados:         1,
			Rechazados:        1,
			CreadoEn:          now,
			ClaveIdempotencia: "idem-pg-1",
			HuellaContenido:   "hash-pg-1",
		},
		Registros: []Registro{
			{ID: regID, LoteID: loteID, Posicion: 1, Folio: "DOC-PG-1", Nombre: "Ana", Correo: "ana@example.com", CreadoEn: now},
		},
		Rechazos: []Rechazo{
			{LoteID: loteID, Posicion: 2, Folio: "DOC-PG-2", Motivos: []string{"correo_invalido"}},
		},
		Auditoria: EventoAuditoria{
			OcurridoEn: now, Actor: "usuario-1", Accion: AccionCrearLote, Entidad: EntidadLote,
			EntidadID: loteID, RequestID: "req-pg-1", Detalle: map[string]any{"aceptados": 1},
		},
	}

	if err := st.Create(ctx, nb); err != nil {
		t.Fatalf("PostgresBatchStore.Create falló: %v", err)
	}
	if len(fq.auditoria) != 1 || fq.auditoria[0].Actor != "usuario-1" || fq.auditoria[0].Accion != AccionCrearLote ||
		formatPGUUID(fq.auditoria[0].EntidadID) != loteID || string(fq.auditoria[0].Detalle) != `{"aceptados":1}` {
		t.Fatalf("el lote debe dejar su evento de auditoría: %+v", fq.auditoria)
	}

	// Clave de idempotencia repetida (con folio nuevo para llegar a CreateLote)
	nbSameKey := nb
	nbSameKey.Registros = []Registro{
		{ID: "990e8400-e29b-41d4-a716-446655440000", LoteID: loteID, Posicion: 1, Folio: "DOC-PG-NEW", Nombre: "Ana", Correo: "ana@example.com", CreadoEn: now},
	}
	if err := st.Create(ctx, nbSameKey); !errors.Is(err, ErrClaveIdempotenciaEnUso) {
		t.Fatalf("se esperaba ErrClaveIdempotenciaEnUso, se obtuvo %v", err)
	}

	// Folio duplicado detectado por FindExistingFolios
	nbDup := nb
	nbDup.Lote.ClaveIdempotencia = "idem-pg-2"
	var dupErr *ErrFoliosDuplicados
	if err := st.Create(ctx, nbDup); !errors.As(err, &dupErr) {
		t.Fatalf("se esperaba ErrFoliosDuplicados, se obtuvo %v", err)
	}

	// Folio duplicado detectado por unique constraint en CreateRegistro
	fq.forcePgDup = true
	nbRace := nb
	nbRace.Lote.ID = "770e8400-e29b-41d4-a716-446655440000"
	nbRace.Lote.ClaveIdempotencia = "idem-pg-3"
	nbRace.Registros = []Registro{
		{ID: "880e8400-e29b-41d4-a716-446655440000", LoteID: nbRace.Lote.ID, Posicion: 1, Folio: "DOC-PG-RACE", Nombre: "Luis", Correo: "luis@example.com", CreadoEn: now},
	}
	if err := st.Create(ctx, nbRace); !errors.As(err, &dupErr) {
		t.Fatalf("se esperaba ErrFoliosDuplicados en constraint violation, se obtuvo %v", err)
	}
	fq.forcePgDup = false

	// Consultas FindByID, FindByIdempotencyKey, List, ListRejections, ListRecords, FindRecordByID
	gotLote, err := st.FindByID(ctx, loteID)
	if err != nil || gotLote.ID != loteID {
		t.Fatalf("FindByID inesperado: %+v (%v)", gotLote, err)
	}
	if _, err := st.FindByID(ctx, "no-uuid"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("FindByID con UUID inválido debe devolver ErrNoEncontrado: %v", err)
	}
	if _, err := st.FindByID(ctx, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("FindByID inexistente debe devolver ErrNoEncontrado: %v", err)
	}

	gotByKey, err := st.FindByIdempotencyKey(ctx, "idem-pg-1")
	if err != nil || gotByKey.ID != loteID {
		t.Fatalf("FindByIdempotencyKey inesperado: %+v (%v)", gotByKey, err)
	}
	if _, err := st.FindByIdempotencyKey(ctx, "no-existe"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("FindByIdempotencyKey inexistente debe devolver ErrNoEncontrado: %v", err)
	}

	lotes, total, err := st.List(ctx, Paginacion{Numero: 1, Tamano: 10})
	if err != nil || total == 0 || len(lotes) == 0 {
		t.Fatalf("List inesperado: len=%d total=%d err=%v", len(lotes), total, err)
	}

	rejs, err := st.ListRejections(ctx, loteID)
	if err != nil || len(rejs) != 1 || rejs[0].Folio != "DOC-PG-2" {
		t.Fatalf("ListRejections inesperado: %+v (%v)", rejs, err)
	}
	if _, err := st.ListRejections(ctx, "no-uuid"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("ListRejections con UUID inválido debe devolver ErrNoEncontrado: %v", err)
	}

	recs, recTotal, err := st.ListRecords(ctx, FiltroRegistros{
		Folio:      "DOC-PG-1",
		LoteID:     loteID,
		Paginacion: Paginacion{Numero: 1, Tamano: 10},
	})
	if err != nil || recTotal != 1 || len(recs) != 1 || recs[0].ID != regID {
		t.Fatalf("ListRecords inesperado: %+v total=%d err=%v", recs, recTotal, err)
	}
	if emptyRecs, emptyTotal, err := st.ListRecords(ctx, FiltroRegistros{LoteID: "no-uuid", Paginacion: Paginacion{Numero: 1, Tamano: 10}}); err != nil || emptyTotal != 0 || len(emptyRecs) != 0 {
		t.Fatalf("ListRecords con LoteID inválido debía devolver lista vacía: %v", err)
	}

	gotRec, err := st.FindRecordByID(ctx, regID)
	if err != nil || gotRec.Folio != "DOC-PG-1" {
		t.Fatalf("FindRecordByID inesperado: %+v (%v)", gotRec, err)
	}
	if _, err := st.FindRecordByID(ctx, "no-uuid"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("FindRecordByID con UUID inválido debe devolver ErrNoEncontrado: %v", err)
	}
	if _, err := st.FindRecordByID(ctx, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("FindRecordByID inexistente debe devolver ErrNoEncontrado: %v", err)
	}
}

func TestConnectDBConfigParsing(t *testing.T) {
	pool, err := ConnectDB(&config.Config{
		PostgresURI:      "postgres://localhost:5432",
		PostgresUser:     "gestor",
		PostgresPassword: "secret",
		PostgresDatabase: "gestor_db",
	})
	if err != nil {
		t.Fatalf("ConnectDB con URI válida falló al crear config: %v", err)
	}
	pool.Close()

	if _, err := ConnectDB(&config.Config{DatabaseURL: "://bad-url"}); err == nil {
		t.Fatal("ConnectDB debía fallar con URL malformada")
	}
}

func TestPostgresBatchStoreWithTestcontainers(t *testing.T) {
	if os.Getenv("SKIP_TESTCONTAINERS") == "1" {
		t.Skip("SKIP_TESTCONTAINERS=1; omitiendo testcontainers")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Docker daemon no accesible para testcontainers en este entorno: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgContainer, err := tcpostgres.Run(ctx,
		postgresImageDigest,
		tcpostgres.WithDatabase("gestor_test"),
		tcpostgres.WithUsername("gestor"),
		tcpostgres.WithPassword("gestor_pass"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("Docker no disponible para testcontainers: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(context.Background()) }()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("obteniendo connection string: %v", err)
	}

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("abriendo sql.DB para goose: %v", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("configurando dialecto goose: %v", err)
	}
	migrationsDir := filepath.Join("..", "..", "db", "migrations")
	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		t.Fatalf("aplicando migraciones goose: %v", err)
	}

	pool, err := ConnectDB(&config.Config{DatabaseURL: connStr})
	if err != nil {
		t.Fatalf("conectando pgxpool: %v", err)
	}
	defer pool.Close()

	st := NewPostgresBatchStore(pool)
	loteID := "550e8400-e29b-41d4-a716-446655440000"
	regID := "660e8400-e29b-41d4-a716-446655440000"
	now := time.Now().UTC().Truncate(time.Microsecond)

	err = st.Create(ctx, NuevoLote{
		Lote: Lote{
			ID:                loteID,
			Origen:            "Extractor OCR",
			ReferenciaExterna: "tc-1",
			Recibidos:         2,
			Aceptados:         1,
			Rechazados:        1,
			ClaveIdempotencia: "idem-tc-1",
			HuellaContenido:   "hash-1",
			CreadoEn:          now,
		},
		Registros: []Registro{
			{ID: regID, LoteID: loteID, Posicion: 1, Folio: "DOC-TC-1", Nombre: "Ana", Correo: "ana@example.com", CreadoEn: now},
		},
		Rechazos: []Rechazo{
			{LoteID: loteID, Posicion: 2, Folio: "DOC-TC-2", Motivos: []string{"correo_invalido"}},
		},
	})
	if err != nil {
		t.Fatalf("Create en PostgresBatchStore falló: %v", err)
	}

	got, err := st.FindByID(ctx, loteID)
	if err != nil || got.ID != loteID {
		t.Fatalf("FindByID inesperado: %+v (%v)", got, err)
	}
	if _, err := st.FindByIdempotencyKey(ctx, "idem-tc-1"); err != nil {
		t.Fatalf("FindByIdempotencyKey falló: %v", err)
	}
	if list, total, err := st.List(ctx, Paginacion{Numero: 1, Tamano: 10}); err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List falló: %v", err)
	}
	if rejs, err := st.ListRejections(ctx, loteID); err != nil || len(rejs) != 1 {
		t.Fatalf("ListRejections falló: %v", err)
	}
	if recs, total, err := st.ListRecords(ctx, FiltroRegistros{Folio: "DOC-TC-1", LoteID: loteID, Paginacion: Paginacion{Numero: 1, Tamano: 10}}); err != nil || total != 1 || len(recs) != 1 {
		t.Fatalf("ListRecords falló: %v", err)
	}
	if rec, err := st.FindRecordByID(ctx, regID); err != nil || rec.ID != regID {
		t.Fatalf("FindRecordByID falló: %v", err)
	}
}
