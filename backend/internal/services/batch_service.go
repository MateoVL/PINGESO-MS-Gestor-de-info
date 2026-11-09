package services

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

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/repositories"
)

// motivos de rechazo de un registro
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
	Registros         []models.RegistroEntrada
}

type BatchResult struct {
	Detalle  models.LoteDetalle
	Repetido bool
}

type BatchService struct {
	repo  repositories.BatchRepository
	now   func() time.Time
	newID func() string
}

func NewBatchService(repo repositories.BatchRepository) *BatchService {
	return &BatchService{repo: repo, now: time.Now, newID: NewUUID}
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
	batch := models.Lote{
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
		err := s.repo.Create(ctx, nb)

		var duplicated *models.ErrFoliosDuplicados
		switch {
		case err == nil:
			return BatchResult{Detalle: models.LoteDetalle{Lote: nb.Lote, Rechazos: nb.Rechazos}}, nil
		case errors.As(err, &duplicated):
			valid, rejected = moveToRejected(valid, rejected, duplicated.Folios)
		case errors.Is(err, models.ErrClaveIdempotenciaEnUso):
			res, _, err := s.existing(ctx, in.ClaveIdempotencia, hash)
			return res, err
		default:
			return BatchResult{}, fmt.Errorf("guardando el lote: %w", err)
		}
	}
	return BatchResult{}, models.ErrDemasiadosConflictos
}

func (s *BatchService) existing(ctx context.Context, key, hash string) (BatchResult, bool, error) {
	batch, err := s.repo.FindByIdempotencyKey(ctx, key)
	if errors.Is(err, models.ErrNoEncontrado) {
		return BatchResult{}, false, nil
	}
	if err != nil {
		return BatchResult{}, false, fmt.Errorf("buscando la Idempotency-Key: %w", err)
	}
	if batch.HuellaContenido != hash {
		return BatchResult{}, true, models.ErrClaveIdempotenciaConflicto
	}
	rejections, err := s.rejections(ctx, batch.ID)
	if err != nil {
		return BatchResult{}, true, err
	}
	return BatchResult{Detalle: models.LoteDetalle{Lote: batch, Rechazos: rejections}, Repetido: true}, true, nil
}

func (s *BatchService) build(base models.Lote, valid []models.Registro, rejected []models.Rechazo) models.NuevoLote {
	batch := base
	batch.Aceptados = len(valid)
	batch.Rechazados = len(rejected)

	records := make([]models.Registro, len(valid))
	for i, r := range valid {
		r.ID = s.newID()
		r.LoteID = batch.ID
		r.CreadoEn = batch.CreadoEn
		records[i] = r
	}
	rejections := make([]models.Rechazo, len(rejected))
	for i, r := range rejected {
		r.LoteID = batch.ID
		rejections[i] = r
	}
	return models.NuevoLote{Lote: batch, Registros: records, Rechazos: rejections}
}

// devuelve un lote con sus rechazos
func (s *BatchService) Get(ctx context.Context, id string) (models.LoteDetalle, error) {
	batch, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return models.LoteDetalle{}, err
	}
	rejections, err := s.rejections(ctx, id)
	if err != nil {
		return models.LoteDetalle{}, err
	}
	return models.LoteDetalle{Lote: batch, Rechazos: rejections}, nil
}

// devuelve los lotes del mas reciente al mas antiguo
func (s *BatchService) List(ctx context.Context, p models.Paginacion) (models.ListaLotes, error) {
	p = normalizePagination(p)
	batches, total, err := s.repo.List(ctx, p)
	if err != nil {
		return models.ListaLotes{}, err
	}
	return models.ListaLotes{InfoPagina: models.InfoPagina{Total: total, Pagina: p.Numero, TamanoPagina: p.Tamano}, Elementos: batches}, nil
}

// devuelve los rechazos de un lote existente
func (s *BatchService) ListRejections(ctx context.Context, batchID string) ([]models.Rechazo, error) {
	if _, err := s.repo.FindByID(ctx, batchID); err != nil {
		return nil, err
	}
	return s.rejections(ctx, batchID)
}

// devuelve registros aceptados
func (s *BatchService) ListRecords(ctx context.Context, f models.FiltroRegistros) (models.ListaRegistros, error) {
	f.Folio = normalizeFolio(f.Folio)
	f.Paginacion = normalizePagination(f.Paginacion)
	records, total, err := s.repo.ListRecords(ctx, f)
	if err != nil {
		return models.ListaRegistros{}, err
	}
	return models.ListaRegistros{InfoPagina: models.InfoPagina{Total: total, Pagina: f.Paginacion.Numero, TamanoPagina: f.Paginacion.Tamano}, Elementos: records}, nil
}

// devuelve UN registro aceptado
func (s *BatchService) GetRecord(ctx context.Context, id string) (models.Registro, error) {
	return s.repo.FindRecordByID(ctx, id)
}

func (s *BatchService) rejections(ctx context.Context, batchID string) ([]models.Rechazo, error) {
	rejections, err := s.repo.ListRejections(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("leyendo los rechazos del lote: %w", err)
	}
	if rejections == nil {
		rejections = []models.Rechazo{}
	}
	return rejections, nil
}

// limpia cada registro y lo separa en validos y rechazados
func classify(input []models.RegistroEntrada) ([]models.Registro, []models.Rechazo) {
	valid := []models.Registro{}
	rejected := []models.Rechazo{}
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
			rejected = append(rejected, models.Rechazo{Posicion: position, Folio: folio, Motivos: reasons})
			continue
		}
		seen[folio] = true
		valid = append(valid, models.Registro{Posicion: position, Folio: folio, Nombre: nombre, Correo: correo})
	}
	return valid, rejected
}

// registros cuyos folios ya existian en otro lote --> rechazafos
func moveToRejected(valid []models.Registro, rejected []models.Rechazo, folios []string) ([]models.Registro, []models.Rechazo) {
	remaining := make([]models.Registro, 0, len(valid))
	for _, r := range valid {
		if slices.Contains(folios, r.Folio) {
			rejected = append(rejected, models.Rechazo{Posicion: r.Posicion, Folio: r.Folio, Motivos: []string{ReasonFolioAlreadyRegistered}})
			continue
		}
		remaining = append(remaining, r)
	}
	slices.SortFunc(rejected, func(a, b models.Rechazo) int { return a.Posicion - b.Posicion })
	return remaining, rejected
}

func normalizeFolio(folio string) string {
	return strings.ToUpper(strings.TrimSpace(folio))
}

func normalizePagination(p models.Paginacion) models.Paginacion {
	if p.Numero < 1 {
		p.Numero = 1
	}
	if p.Tamano < 1 {
		p.Tamano = models.TamanoPaginaPorDefecto
	}
	p.Tamano = min(p.Tamano, models.TamanoPaginaMaximo)
	return p
}

func contentHash(in BatchInput) (string, error) {
	content := struct {
		Origen, Referencia string
		Registros          []models.RegistroEntrada
	}{in.Origen, in.ReferenciaExterna, in.Registros}
	data, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("calculando la huella del lote: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// genera un UUID
func NewUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
