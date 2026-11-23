package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

const (
	ReasonMissingFolio           = "falta_folio"
	ReasonMissingName            = "falta_nombre"
	ReasonInvalidEmail           = "correo_invalido"
	ReasonDuplicateFolio         = "folio_duplicado"
	ReasonFolioAlreadyRegistered = "folio_ya_registrado"
)

const maxAttempts = 5

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type BatchInput struct {
	Origen            string
	ReferenciaExterna string
	ClaveIdempotencia string
	Registros         []store.RegistroEntrada
}

type BatchResult struct {
	Detalle  store.LoteDetalle
	Repetido bool
}

type BatchService struct {
	store store.BatchStore
	now   func() time.Time
	newID func() string
}

func NewBatchService(st store.BatchStore) *BatchService {
	return &BatchService{store: st, now: time.Now, newID: NewUUID}
}

func (s *BatchService) Receive(ctx context.Context, in BatchInput) (BatchResult, error) {
	hash, err := contentHash(in)
	if err != nil {
		return BatchResult{}, err
	}

	if in.ClaveIdempotencia != "" {
		if res, found, err := s.existing(ctx, in.ClaveIdempotencia, hash); found || err != nil {
			return res, err
		}
	}

	valid, rejected := classify(in.Registros)
	batch := store.Lote{
		ID:                s.newID(),
		Origen:            strings.TrimSpace(in.Origen),
		ReferenciaExterna: strings.TrimSpace(in.ReferenciaExterna),
		Recibidos:         len(in.Registros),
		CreadoEn:          s.now().UTC(),
		ClaveIdempotencia: in.ClaveIdempotencia,
		HuellaContenido:   hash,
	}

	for range maxAttempts {
		nb := s.build(batch, valid, rejected)
		err := s.store.Create(ctx, nb)

		var duplicated *store.ErrFoliosDuplicados
		switch {
		case err == nil:
			return BatchResult{Detalle: store.LoteDetalle{Lote: nb.Lote, Rechazos: nb.Rechazos}}, nil
		case errors.As(err, &duplicated):
			valid, rejected = moveToRejected(valid, rejected, duplicated.Folios)
		case errors.Is(err, store.ErrClaveIdempotenciaEnUso):
			res, _, err := s.existing(ctx, in.ClaveIdempotencia, hash)
			return res, err
		default:
			return BatchResult{}, fmt.Errorf("guardando el lote: %w", err)
		}
	}
	return BatchResult{}, store.ErrDemasiadosConflictos
}

func (s *BatchService) existing(ctx context.Context, key, hash string) (BatchResult, bool, error) {
	batch, err := s.store.FindByIdempotencyKey(ctx, key)
	if errors.Is(err, store.ErrNoEncontrado) {
		return BatchResult{}, false, nil
	}
	if err != nil {
		return BatchResult{}, false, fmt.Errorf("buscando la Idempotency-Key: %w", err)
	}
	if batch.HuellaContenido != hash {
		return BatchResult{}, true, store.ErrClaveIdempotenciaConflicto
	}
	rejections, err := s.rejections(ctx, batch.ID)
	if err != nil {
		return BatchResult{}, true, err
	}
	return BatchResult{Detalle: store.LoteDetalle{Lote: batch, Rechazos: rejections}, Repetido: true}, true, nil
}

func (s *BatchService) build(base store.Lote, valid []store.Registro, rejected []store.Rechazo) store.NuevoLote {
	batch := base
	batch.Aceptados = len(valid)
	batch.Rechazados = len(rejected)

	records := make([]store.Registro, len(valid))
	for i, r := range valid {
		r.ID = s.newID()
		r.LoteID = batch.ID
		r.CreadoEn = batch.CreadoEn
		records[i] = r
	}
	rejections := make([]store.Rechazo, len(rejected))
	for i, r := range rejected {
		r.LoteID = batch.ID
		rejections[i] = r
	}
	return store.NuevoLote{Lote: batch, Registros: records, Rechazos: rejections}
}

func (s *BatchService) Get(ctx context.Context, id string) (store.LoteDetalle, error) {
	batch, err := s.store.FindByID(ctx, id)
	if err != nil {
		return store.LoteDetalle{}, err
	}
	rejections, err := s.rejections(ctx, id)
	if err != nil {
		return store.LoteDetalle{}, err
	}
	return store.LoteDetalle{Lote: batch, Rechazos: rejections}, nil
}

func (s *BatchService) List(ctx context.Context, p store.Paginacion) (store.ListaLotes, error) {
	p = normalizePagination(p)
	batches, total, err := s.store.List(ctx, p)
	if err != nil {
		return store.ListaLotes{}, err
	}
	return store.ListaLotes{InfoPagina: store.InfoPagina{Total: total, Pagina: p.Numero, TamanoPagina: p.Tamano}, Elementos: batches}, nil
}

func (s *BatchService) ListRejections(ctx context.Context, batchID string) ([]store.Rechazo, error) {
	if _, err := s.store.FindByID(ctx, batchID); err != nil {
		return nil, err
	}
	return s.rejections(ctx, batchID)
}

func (s *BatchService) ListRecords(ctx context.Context, f store.FiltroRegistros) (store.ListaRegistros, error) {
	f.Folio = normalizeFolio(f.Folio)
	f.Paginacion = normalizePagination(f.Paginacion)
	records, total, err := s.store.ListRecords(ctx, f)
	if err != nil {
		return store.ListaRegistros{}, err
	}
	return store.ListaRegistros{InfoPagina: store.InfoPagina{Total: total, Pagina: f.Paginacion.Numero, TamanoPagina: f.Paginacion.Tamano}, Elementos: records}, nil
}

func (s *BatchService) GetRecord(ctx context.Context, id string) (store.Registro, error) {
	return s.store.FindRecordByID(ctx, id)
}

func (s *BatchService) rejections(ctx context.Context, batchID string) ([]store.Rechazo, error) {
	rejections, err := s.store.ListRejections(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("leyendo los rechazos del lote: %w", err)
	}
	if rejections == nil {
		rejections = []store.Rechazo{}
	}
	return rejections, nil
}

func classify(input []store.RegistroEntrada) ([]store.Registro, []store.Rechazo) {
	valid := []store.Registro{}
	rejected := []store.Rechazo{}
	seen := make(map[string]bool, len(input))

	for i, in := range input {
		position := i + 1
		folio := normalizeFolio(in.Folio)
		nombre := strings.Join(strings.Fields(in.Nombre), " ")
		correo := strings.ToLower(strings.TrimSpace(in.Correo))

		var reasons []string
		if folio == "" {
			reasons = append(reasons, ReasonMissingFolio)
		}
		if nombre == "" {
			reasons = append(reasons, ReasonMissingName)
		}
		if !emailPattern.MatchString(correo) {
			reasons = append(reasons, ReasonInvalidEmail)
		}
		if folio != "" && seen[folio] {
			reasons = append(reasons, ReasonDuplicateFolio)
		}

		if len(reasons) > 0 {
			rejected = append(rejected, store.Rechazo{Posicion: position, Folio: folio, Motivos: reasons})
			continue
		}
		seen[folio] = true
		valid = append(valid, store.Registro{Posicion: position, Folio: folio, Nombre: nombre, Correo: correo})
	}
	return valid, rejected
}

func moveToRejected(valid []store.Registro, rejected []store.Rechazo, folios []string) ([]store.Registro, []store.Rechazo) {
	remaining := make([]store.Registro, 0, len(valid))
	for _, r := range valid {
		if slices.Contains(folios, r.Folio) {
			rejected = append(rejected, store.Rechazo{Posicion: r.Posicion, Folio: r.Folio, Motivos: []string{ReasonFolioAlreadyRegistered}})
			continue
		}
		remaining = append(remaining, r)
	}
	slices.SortFunc(rejected, func(a, b store.Rechazo) int { return a.Posicion - b.Posicion })
	return remaining, rejected
}

func normalizeFolio(folio string) string {
	return strings.ToUpper(strings.TrimSpace(folio))
}

func normalizePagination(p store.Paginacion) store.Paginacion {
	if p.Numero < 1 {
		p.Numero = 1
	}
	if p.Tamano < 1 {
		p.Tamano = store.TamanoPaginaPorDefecto
	}
	p.Tamano = min(p.Tamano, store.TamanoPaginaMaximo)
	return p
}

func contentHash(in BatchInput) (string, error) {
	content := struct {
		Origen, Referencia string
		Registros          []store.RegistroEntrada
	}{in.Origen, in.ReferenciaExterna, in.Registros}
	data, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("calculando la huella del lote: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func NewUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

