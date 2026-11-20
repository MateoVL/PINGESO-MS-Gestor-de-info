package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
)

// Límites de un lote y de la paginación
const (
	MaxRegistrosPorLote    = 5000
	MaxLargoCampo          = 254
	TamanoPaginaPorDefecto = 50
	TamanoPaginaMaximo     = 200
)

// RegistroEntrada representa un registro tal como llega, sin limpiar
type RegistroEntrada struct {
	Folio  string `json:"folio" doc:"Folio identificador del documento" example:"DOC-0001"`
	Nombre string `json:"nombre" doc:"Nombre completo del titular" example:"Camila Soto"`
	Correo string `json:"correo" doc:"Correo electrónico de contacto" example:"camila@example.com"`
}

// SolicitudLote representa el payload JSON de carga de lotes
type SolicitudLote struct {
	Origen            string            `json:"origen" doc:"Sistema emisor del lote" example:"Extractor OCR"`
	ReferenciaExterna string            `json:"referencia_externa" doc:"Identificador de correlación en el emisor" example:"lote-2026-001"`
	Registros         []RegistroEntrada `json:"registros" doc:"Lista de registros a procesar"`
}

// Lote resume un lote procesado
type Lote struct {
	ID                string    `json:"id" doc:"UUID del lote" example:"550e8400-e29b-41d4-a716-446655440000"`
	Origen            string    `json:"origen" doc:"Sistema emisor" example:"Extractor OCR"`
	ReferenciaExterna string    `json:"referencia_externa" doc:"Referencia externa" example:"lote-2026-001"`
	Recibidos         int       `json:"recibidos" doc:"Total de registros recibidos" example:"20"`
	Aceptados         int       `json:"aceptados" doc:"Total de registros aceptados" example:"12"`
	Rechazados        int       `json:"rechazados" doc:"Total de registros rechazados" example:"8"`
	CreadoEn          time.Time `json:"creado_en" doc:"Fecha de creación en UTC"`

	ClaveIdempotencia string `json:"-"`
	HuellaContenido   string `json:"-"`
}

// Registro limpio y validado
type Registro struct {
	ID       string    `json:"id" doc:"UUID del registro" example:"660e8400-e29b-41d4-a716-446655440000"`
	LoteID   string    `json:"lote_id" doc:"UUID del lote al que pertenece" example:"550e8400-e29b-41d4-a716-446655440000"`
	Posicion int       `json:"posicion" doc:"Posición 1-indexed dentro del lote original" example:"1"`
	Folio    string    `json:"folio" doc:"Folio normalizado en mayúsculas" example:"DOC-0001"`
	Nombre   string    `json:"nombre" doc:"Nombre normalizado" example:"Camila Soto"`
	Correo   string    `json:"correo" doc:"Correo normalizado en minúsculas" example:"camila@example.com"`
	CreadoEn time.Time `json:"creado_en" doc:"Fecha de registro en UTC"`
}

// Rechazo explica por qué no se aceptó un registro del lote
type Rechazo struct {
	LoteID   string   `json:"-"`
	Posicion int      `json:"posicion" doc:"Posición 1-indexed dentro del lote" example:"2"`
	Folio    string   `json:"folio" doc:"Folio normalizado o vacío" example:"DOC-0001"`
	Motivos  []string `json:"motivos" doc:"Códigos de motivo de rechazo" example:"[\"folio_duplicado\"]"`
}

// LoteDetalle incluye el resumen del lote con sus rechazos
type LoteDetalle struct {
	Lote
	Rechazos []Rechazo `json:"rechazos" doc:"Detalle de los registros rechazados"`
}

// InfoPagina describe la página devuelta en un listado
type InfoPagina struct {
	Total        int `json:"total" doc:"Total de elementos encontrados" example:"42"`
	Pagina       int `json:"pagina" doc:"Número de página actual (1-indexed)" example:"1"`
	TamanoPagina int `json:"tamano_pagina" doc:"Cantidad máxima por página" example:"50"`
}

type ListaLotes struct {
	InfoPagina
	Elementos []Lote `json:"elementos" doc:"Lotes de la página"`
}

type ListaRegistros struct {
	InfoPagina
	Elementos []Registro `json:"elementos" doc:"Registros aceptados de la página"`
}

type ListaRechazos struct {
	Elementos []Rechazo `json:"elementos" doc:"Rechazos del lote"`
}

type Paginacion struct {
	Numero int
	Tamano int
}

func (p Paginacion) Desplazamiento() int {
	return (p.Numero - 1) * p.Tamano
}

type FiltroRegistros struct {
	Folio      string
	LoteID     string
	Paginacion Paginacion
}

type NuevoLote struct {
	Lote      Lote
	Registros []Registro
	Rechazos  []Rechazo
}

// Errores de dominio y almacenamiento
var (
	ErrNoEncontrado               = errors.New("recurso no encontrado")
	ErrClaveIdempotenciaEnUso     = errors.New("la Idempotency-Key ya está en uso")
	ErrClaveIdempotenciaConflicto = errors.New("la Idempotency-Key ya se usó con otro contenido")
	ErrDemasiadosConflictos       = errors.New("demasiados conflictos de folio concurrentes")
)

type ErrFoliosDuplicados struct {
	Folios []string
}

func (e *ErrFoliosDuplicados) Error() string {
	return fmt.Sprintf("folios ya registrados: %s", strings.Join(e.Folios, ", "))
}

// BatchStore define la capa de acceso a datos de lotes y registros (Clean Architecture: Handler -> Service -> Store)
type BatchStore interface {
	Create(ctx context.Context, batch NuevoLote) error
	FindByID(ctx context.Context, id string) (Lote, error)
	FindByIdempotencyKey(ctx context.Context, key string) (Lote, error)
	List(ctx context.Context, p Paginacion) ([]Lote, int, error)
	ListRejections(ctx context.Context, batchID string) ([]Rechazo, error)
	ListRecords(ctx context.Context, f FiltroRegistros) ([]Registro, int, error)
	FindRecordByID(ctx context.Context, id string) (Registro, error)
}

// ConnectDB crea un pool de conexiones hacia PostgreSQL
func ConnectDB(cfg *config.Config) (*pgxpool.Pool, error) {
	connURI := cfg.DatabaseURL
	if connURI == "" {
		connURI = cfg.PostgresURI
	}
	u, err := url.Parse(connURI)
	if err != nil {
		return nil, err
	}
	if cfg.PostgresUser != "" && u.User == nil {
		u.User = url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword)
	}
	if (u.Path == "" || u.Path == "/") && cfg.PostgresDatabase != "" {
		u.Path = "/" + strings.TrimPrefix(cfg.PostgresDatabase, "/")
	}

	poolCfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(context.Background(), poolCfg)
}

// MemoryBatchStore implementa BatchStore en memoria para modo API-first / pruebas rápidas
type MemoryBatchStore struct {
	mu         sync.RWMutex
	batches    []Lote
	byID       map[string]int
	byKey      map[string]string
	records    []Registro
	folios     map[string]bool
	rejections map[string][]Rechazo
}

func NewMemoryBatchStore() *MemoryBatchStore {
	return &MemoryBatchStore{
		byID:       map[string]int{},
		byKey:      map[string]string{},
		folios:     map[string]bool{},
		rejections: map[string][]Rechazo{},
	}
}

var _ BatchStore = (*MemoryBatchStore)(nil)

func (m *MemoryBatchStore) Create(_ context.Context, nb NuevoLote) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if key := nb.Lote.ClaveIdempotencia; key != "" {
		if _, exists := m.byKey[key]; exists {
			return ErrClaveIdempotenciaEnUso
		}
	}
	var duplicated []string
	for _, r := range nb.Registros {
		if m.folios[r.Folio] {
			duplicated = append(duplicated, r.Folio)
		}
	}
	if len(duplicated) > 0 {
		slices.Sort(duplicated)
		return &ErrFoliosDuplicados{Folios: duplicated}
	}

	m.byID[nb.Lote.ID] = len(m.batches)
	m.batches = append(m.batches, nb.Lote)
	if nb.Lote.ClaveIdempotencia != "" {
		m.byKey[nb.Lote.ClaveIdempotencia] = nb.Lote.ID
	}
	for _, r := range nb.Registros {
		m.folios[r.Folio] = true
	}
	m.records = append(m.records, nb.Registros...)
	m.rejections[nb.Lote.ID] = slices.Clone(nb.Rechazos)
	return nil
}

func (m *MemoryBatchStore) FindByID(_ context.Context, id string) (Lote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	i, ok := m.byID[id]
	if !ok {
		return Lote{}, ErrNoEncontrado
	}
	return m.batches[i], nil
}

func (m *MemoryBatchStore) FindByIdempotencyKey(ctx context.Context, key string) (Lote, error) {
	m.mu.RLock()
	id, ok := m.byKey[key]
	m.mu.RUnlock()
	if !ok {
		return Lote{}, ErrNoEncontrado
	}
	return m.FindByID(ctx, id)
}

func (m *MemoryBatchStore) List(_ context.Context, p Paginacion) ([]Lote, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	newest := slices.Clone(m.batches)
	slices.Reverse(newest)
	return page(newest, p), len(newest), nil
}

func (m *MemoryBatchStore) ListRejections(_ context.Context, batchID string) ([]Rechazo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.rejections[batchID]), nil
}

func (m *MemoryBatchStore) ListRecords(_ context.Context, f FiltroRegistros) ([]Registro, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var filtered []Registro
	for _, r := range m.records {
		if (f.Folio == "" || r.Folio == f.Folio) && (f.LoteID == "" || r.LoteID == f.LoteID) {
			filtered = append(filtered, r)
		}
	}
	return page(filtered, f.Paginacion), len(filtered), nil
}

func (m *MemoryBatchStore) FindRecordByID(_ context.Context, id string) (Registro, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.records {
		if r.ID == id {
			return r, nil
		}
	}
	return Registro{}, ErrNoEncontrado
}

func page[T any](items []T, p Paginacion) []T {
	start := min(p.Desplazamiento(), len(items))
	end := min(start+p.Tamano, len(items))
	result := make([]T, end-start)
	copy(result, items[start:end])
	return result
}

