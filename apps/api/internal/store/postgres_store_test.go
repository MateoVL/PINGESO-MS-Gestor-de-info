package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestUUIDHelpersAndMemoryStore(t *testing.T) {
	const sampleID = "550e8400-e29b-41d4-a716-446655440000"
	uid, err := parsePGUUID(sampleID)
	if err != nil {
		t.Fatalf("parsePGUUID falló: %v", err)
	}
	if got := formatPGUUID(uid); got != sampleID {
		t.Fatalf("formatPGUUID = %q, se esperaba %q", got, sampleID)
	}
	if _, err := parsePGUUID("no-es-uuid"); err == nil {
		t.Fatal("parsePGUUID debía fallar con un UUID inválido")
	}
}

func TestPostgresBatchStoreWithTestcontainers(t *testing.T) {
	if os.Getenv("RUN_TESTCONTAINERS") != "1" {
		t.Skip("RUN_TESTCONTAINERS!=1; se omite prueba con contenedor Docker real en entorno sin daemon")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:17.11",
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
}
