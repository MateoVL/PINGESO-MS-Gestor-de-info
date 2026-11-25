package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/repository"
)

const pgUniqueViolation = "23505"

// txBeginner abstrae el inicio de transacciones para permitir pruebas unitarias y uso con pgxpool.Pool.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PostgresBatchStore implementa BatchStore sobre PostgreSQL usando sqlc (internal/repository) y pgx/v5.
type PostgresBatchStore struct {
	txPool  txBeginner
	queries repository.Querier
	withTx  func(tx pgx.Tx) repository.Querier
}

func NewPostgresBatchStore(pool *pgxpool.Pool) *PostgresBatchStore {
	baseQueries := repository.New(pool)
	return &PostgresBatchStore{
		txPool:  pool,
		queries: baseQueries,
		withTx: func(tx pgx.Tx) repository.Querier {
			return baseQueries.WithTx(tx)
		},
	}
}

func newPostgresBatchStoreWithQuerier(txPool txBeginner, q repository.Querier, withTx func(tx pgx.Tx) repository.Querier) *PostgresBatchStore {
	if withTx == nil {
		withTx = func(_ pgx.Tx) repository.Querier { return q }
	}
	return &PostgresBatchStore{
		txPool:  txPool,
		queries: q,
		withTx:  withTx,
	}
}

var _ BatchStore = (*PostgresBatchStore)(nil)

func (s *PostgresBatchStore) Create(ctx context.Context, nb NuevoLote) error {
	var qtx repository.Querier = s.queries
	var tx pgx.Tx
	if s.txPool != nil {
		var err error
		tx, err = s.txPool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("iniciando transacción: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		qtx = s.withTx(tx)
	}

	if len(nb.Registros) > 0 {
		folios := make([]string, len(nb.Registros))
		for i, r := range nb.Registros {
			folios[i] = r.Folio
		}
		dup, err := qtx.FindExistingFolios(ctx, folios)
		if err != nil {
			return fmt.Errorf("verificando folios existentes: %w", err)
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
		CreadoEn:          toPGTimestamptz(nb.Lote.CreadoEn),
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
		_, err = qtx.CreateRegistro(ctx, repository.CreateRegistroParams{
			ID:       regUUID,
			LoteID:   loteUUID,
			Posicion: int32(r.Posicion),
			Folio:    r.Folio,
			Nombre:   r.Nombre,
			Correo:   r.Correo,
			CreadoEn: toPGTimestamptz(r.CreadoEn),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return &ErrFoliosDuplicados{Folios: []string{r.Folio}}
			}
			return fmt.Errorf("insertando registro: %w", err)
		}
	}

	for _, rej := range nb.Rechazos {
		_, err := qtx.CreateRechazo(ctx, repository.CreateRechazoParams{
			LoteID:   loteUUID,
			Posicion: int32(rej.Posicion),
			Folio:    rej.Folio,
			Motivos:  rej.Motivos,
		})
		if err != nil {
			return fmt.Errorf("insertando rechazo: %w", err)
		}
	}

	if nb.Auditoria.Accion != "" {
		if err := insertarAuditoria(ctx, qtx, nb.Auditoria); err != nil {
			return err
		}
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
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
	rows, err := s.queries.ListRechazosByLoteID(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]Rechazo, len(rows))
	for i, r := range rows {
		out[i] = Rechazo{
			LoteID:   formatPGUUID(r.LoteID),
			Posicion: int(r.Posicion),
			Folio:    r.Folio,
			Motivos:  r.Motivos,
		}
	}
	return out, nil
}

func (s *PostgresBatchStore) ListRecords(ctx context.Context, f FiltroRegistros) ([]Registro, int, error) {
	var folioArg pgtype.Text
	if f.Folio != "" {
		folioArg = pgtype.Text{String: f.Folio, Valid: true}
	}
	var loteArg pgtype.UUID
	if f.LoteID != "" {
		uid, err := parsePGUUID(f.LoteID)
		if err != nil {
			return []Registro{}, 0, nil
		}
		loteArg = uid
	}

	total, err := s.queries.CountRegistros(ctx, repository.CountRegistrosParams{
		Folio:  folioArg,
		LoteID: loteArg,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.queries.ListRegistros(ctx, repository.ListRegistrosParams{
		Limit:  int32(f.Paginacion.Tamano),
		Offset: int32(f.Paginacion.Desplazamiento()),
		Folio:  folioArg,
		LoteID: loteArg,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]Registro, len(rows))
	for i, r := range rows {
		out[i] = mapRepoRegistro(r)
	}
	return out, int(total), nil
}

func (s *PostgresBatchStore) FindRecordByID(ctx context.Context, id string) (Registro, error) {
	uid, err := parsePGUUID(id)
	if err != nil {
		return Registro{}, ErrNoEncontrado
	}
	row, err := s.queries.GetRegistroByID(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return Registro{}, ErrNoEncontrado
	}
	if err != nil {
		return Registro{}, err
	}
	return mapRepoRegistro(row), nil
}

// guarda quien, que y cuando dentro de la transaccion en curso
func insertarAuditoria(ctx context.Context, q repository.Querier, ev EventoAuditoria) error {
	entidadID, err := parsePGUUID(ev.EntidadID)
	if err != nil {
		return err
	}
	detalle := ev.Detalle
	if detalle == nil {
		detalle = map[string]any{}
	}
	detalleJSON, err := json.Marshal(detalle)
	if err != nil {
		return fmt.Errorf("serializando el detalle de auditoría: %w", err)
	}
	err = q.CreateEventoAuditoria(ctx, repository.CreateEventoAuditoriaParams{
		OcurridoEn: toPGTimestamptz(ev.OcurridoEn),
		Actor:      ev.Actor,
		Accion:     ev.Accion,
		Entidad:    ev.Entidad,
		EntidadID:  entidadID,
		RequestID:  ev.RequestID,
		Detalle:    detalleJSON,
	})
	if err != nil {
		return fmt.Errorf("insertando auditoría: %w", err)
	}
	return nil
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

func toPGTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: !t.IsZero()}
}

func mapRepoLote(r repository.Lote) Lote {
	return Lote{
		ID:                formatPGUUID(r.ID),
		Origen:            r.Origen,
		ReferenciaExterna: r.ReferenciaExterna,
		Recibidos:         int(r.Recibidos),
		Aceptados:         int(r.Aceptados),
		Rechazados:        int(r.Rechazados),
		CreadoEn:          r.CreadoEn.Time,
		ClaveIdempotencia: r.ClaveIdempotencia.String,
		HuellaContenido:   r.HuellaContenido,
	}
}

func mapRepoRegistro(r repository.Registro) Registro {
	return Registro{
		ID:       formatPGUUID(r.ID),
		LoteID:   formatPGUUID(r.LoteID),
		Posicion: int(r.Posicion),
		Folio:    r.Folio,
		Nombre:   r.Nombre,
		Correo:   r.Correo,
		CreadoEn: r.CreadoEn.Time,
	}
}
