package store

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/repository"
)

const pgUniqueViolation = "23505"

// PostgresBatchStore implementa BatchStore sobre PostgreSQL usando sqlc (internal/repository) y pgx/v5.
type PostgresBatchStore struct {
	pool    *pgxpool.Pool
	queries *repository.Queries
}

func NewPostgresBatchStore(pool *pgxpool.Pool) *PostgresBatchStore {
	return &PostgresBatchStore{
		pool:    pool,
		queries: repository.New(pool),
	}
}

var _ BatchStore = (*PostgresBatchStore)(nil)

func (s *PostgresBatchStore) Create(ctx context.Context, nb NuevoLote) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciando transacción: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	if len(nb.Registros) > 0 {
		folios := make([]string, len(nb.Registros))
		for i, r := range nb.Registros {
			folios[i] = r.Folio
		}
		rows, err := tx.Query(ctx, `SELECT folio FROM registros WHERE folio = ANY($1)`, folios)
		if err != nil {
			return fmt.Errorf("verificando folios existentes: %w", err)
		}
		var dup []string
		for rows.Next() {
			var f string
			if err := rows.Scan(&f); err != nil {
				rows.Close()
				return err
			}
			dup = append(dup, f)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(dup) > 0 {
			slices.Sort(dup)
			return &ErrFoliosDuplicados{Folios: dup}
		}
	}

	loteUUID, err := parsePGUUID(nb.Lote.ID)
	if err != nil {
		return err
	}
	var idempKey pgtype.Text
	if nb.Lote.ClaveIdempotencia != "" {
		idempKey = pgtype.Text{String: nb.Lote.ClaveIdempotencia, Valid: true}
	}

	_, err = qtx.CreateLote(ctx, repository.CreateLoteParams{
		ID:                loteUUID,
		Origen:            nb.Lote.Origen,
		ReferenciaExterna: nb.Lote.ReferenciaExterna,
		Recibidos:         int32(nb.Lote.Recibidos),
		Aceptados:         int32(nb.Lote.Aceptados),
		Rechazados:        int32(nb.Lote.Rechazados),
		ClaveIdempotencia: idempKey,
		HuellaContenido:   nb.Lote.HuellaContenido,
		CreadoEn:          nb.Lote.CreadoEn,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return ErrClaveIdempotenciaEnUso
		}
		return fmt.Errorf("insertando lote: %w", err)
	}

	for _, r := range nb.Registros {
		regUUID, err := parsePGUUID(r.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO registros (id, lote_id, posicion, folio, nombre, correo, creado_en) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			regUUID, loteUUID, int32(r.Posicion), r.Folio, r.Nombre, r.Correo, r.CreadoEn,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return &ErrFoliosDuplicados{Folios: []string{r.Folio}}
			}
			return fmt.Errorf("insertando registro: %w", err)
		}
	}

	for _, rej := range nb.Rechazos {
		_, err := tx.Exec(ctx,
			`INSERT INTO rechazos (lote_id, posicion, folio, motivos) VALUES ($1, $2, $3, $4)`,
			loteUUID, int32(rej.Posicion), rej.Folio, rej.Motivos,
		)
		if err != nil {
			return fmt.Errorf("insertando rechazo: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresBatchStore) FindByID(ctx context.Context, id string) (Lote, error) {
	uid, err := parsePGUUID(id)
	if err != nil {
		return Lote{}, ErrNoEncontrado
	}
	row, err := s.queries.GetLoteByID(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lote{}, ErrNoEncontrado
	}
	if err != nil {
		return Lote{}, err
	}
	return mapRepoLote(row), nil
}

func (s *PostgresBatchStore) FindByIdempotencyKey(ctx context.Context, key string) (Lote, error) {
	row, err := s.queries.GetLoteByIdempotencyKey(ctx, pgtype.Text{String: key, Valid: key != ""})
	if errors.Is(err, pgx.ErrNoRows) {
		return Lote{}, ErrNoEncontrado
	}
	if err != nil {
		return Lote{}, err
	}
	return mapRepoLote(row), nil
}

func (s *PostgresBatchStore) List(ctx context.Context, p Paginacion) ([]Lote, int, error) {
	total, err := s.queries.CountLotes(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.queries.ListLotes(ctx, repository.ListLotesParams{
		Limit:  int32(p.Tamano),
		Offset: int32(p.Desplazamiento()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]Lote, len(rows))
	for i, r := range rows {
		out[i] = mapRepoLote(r)
	}
	return out, int(total), nil
}

func (s *PostgresBatchStore) ListRejections(ctx context.Context, batchID string) ([]Rechazo, error) {
	uid, err := parsePGUUID(batchID)
	if err != nil {
		return nil, ErrNoEncontrado
	}
	rows, err := s.pool.Query(ctx,
		`SELECT lote_id, posicion, folio, motivos FROM rechazos WHERE lote_id = $1 ORDER BY posicion ASC`,
		uid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Rechazo
	for rows.Next() {
		var loteUID pgtype.UUID
		var pos int32
		var folio string
		var motivos []string
		if err := rows.Scan(&loteUID, &pos, &folio, &motivos); err != nil {
			return nil, err
		}
		out = append(out, Rechazo{
			LoteID:   formatPGUUID(loteUID),
			Posicion: int(pos),
			Folio:    folio,
			Motivos:  motivos,
		})
	}
	return out, rows.Err()
}

func (s *PostgresBatchStore) ListRecords(ctx context.Context, f FiltroRegistros) ([]Registro, int, error) {
	var folioArg any
	if f.Folio != "" {
		folioArg = f.Folio
	}
	var loteArg any
	if f.LoteID != "" {
		uid, err := parsePGUUID(f.LoteID)
		if err != nil {
			return []Registro{}, 0, nil
		}
		loteArg = uid
	}

	var total int64
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM registros WHERE ($1::varchar IS NULL OR folio = $1) AND ($2::uuid IS NULL OR lote_id = $2)`,
		folioArg, loteArg,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, lote_id, posicion, folio, nombre, correo, creado_en FROM registros
		 WHERE ($1::varchar IS NULL OR folio = $1) AND ($2::uuid IS NULL OR lote_id = $2)
		 ORDER BY creado_en ASC, posicion ASC LIMIT $3 OFFSET $4`,
		folioArg, loteArg, int32(f.Paginacion.Tamano), int32(f.Paginacion.Desplazamiento()),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []Registro{}
	for rows.Next() {
		var idUID, loteUID pgtype.UUID
		var pos int32
		var rec Registro
		if err := rows.Scan(&idUID, &loteUID, &pos, &rec.Folio, &rec.Nombre, &rec.Correo, &rec.CreadoEn); err != nil {
			return nil, 0, err
		}
		rec.ID = formatPGUUID(idUID)
		rec.LoteID = formatPGUUID(loteUID)
		rec.Posicion = int(pos)
		out = append(out, rec)
	}
	return out, int(total), rows.Err()
}

func (s *PostgresBatchStore) FindRecordByID(ctx context.Context, id string) (Registro, error) {
	uid, err := parsePGUUID(id)
	if err != nil {
		return Registro{}, ErrNoEncontrado
	}
	var idUID, loteUID pgtype.UUID
	var pos int32
	var rec Registro
	err = s.pool.QueryRow(ctx,
		`SELECT id, lote_id, posicion, folio, nombre, correo, creado_en FROM registros WHERE id = $1 LIMIT 1`,
		uid,
	).Scan(&idUID, &loteUID, &pos, &rec.Folio, &rec.Nombre, &rec.Correo, &rec.CreadoEn)
	if errors.Is(err, pgx.ErrNoRows) {
		return Registro{}, ErrNoEncontrado
	}
	if err != nil {
		return Registro{}, err
	}
	rec.ID = formatPGUUID(idUID)
	rec.LoteID = formatPGUUID(loteUID)
	rec.Posicion = int(pos)
	return rec, nil
}

func parsePGUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("UUID inválido %q: %w", s, err)
	}
	return u, nil
}

func formatPGUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func mapRepoLote(r repository.Lote) Lote {
	return Lote{
		ID:                formatPGUUID(r.ID),
		Origen:            r.Origen,
		ReferenciaExterna: r.ReferenciaExterna,
		Recibidos:         int(r.Recibidos),
		Aceptados:         int(r.Aceptados),
		Rechazados:        int(r.Rechazados),
		CreadoEn:          r.CreadoEn,
		ClaveIdempotencia: r.ClaveIdempotencia.String,
		HuellaContenido:   r.HuellaContenido,
	}
}
