package repositories

import (
	"context"
	"slices"
	"sync"

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

// guarda los datos en memoria
type MemoryBatchRepository struct {
	mu         sync.RWMutex
	batches    []models.Lote
	byID       map[string]int
	byKey      map[string]string
	records    []models.Registro
	folios     map[string]bool
	rejections map[string][]models.Rechazo
}

func NewMemoryBatchRepository() *MemoryBatchRepository {
	return &MemoryBatchRepository{
		byID:       map[string]int{},
		byKey:      map[string]string{},
		folios:     map[string]bool{},
		rejections: map[string][]models.Rechazo{},
	}
}

var _ BatchRepository = (*MemoryBatchRepository)(nil)

func (m *MemoryBatchRepository) Create(_ context.Context, nb models.NuevoLote) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if key := nb.Lote.ClaveIdempotencia; key != "" {
		if _, exists := m.byKey[key]; exists {
			return models.ErrClaveIdempotenciaEnUso
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
		return &models.ErrFoliosDuplicados{Folios: duplicated}
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

func (m *MemoryBatchRepository) FindByID(_ context.Context, id string) (models.Lote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	i, ok := m.byID[id]
	if !ok {
		return models.Lote{}, models.ErrNoEncontrado
	}
	return m.batches[i], nil
}

func (m *MemoryBatchRepository) FindByIdempotencyKey(ctx context.Context, key string) (models.Lote, error) {
	m.mu.RLock()
	id, ok := m.byKey[key]
	m.mu.RUnlock()
	if !ok {
		return models.Lote{}, models.ErrNoEncontrado
	}
	return m.FindByID(ctx, id)
}

func (m *MemoryBatchRepository) List(_ context.Context, p models.Paginacion) ([]models.Lote, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	newest := slices.Clone(m.batches)
	slices.Reverse(newest)
	return page(newest, p), len(newest), nil
}

func (m *MemoryBatchRepository) ListRejections(_ context.Context, batchID string) ([]models.Rechazo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.rejections[batchID]), nil
}

func (m *MemoryBatchRepository) ListRecords(_ context.Context, f models.FiltroRegistros) ([]models.Registro, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var filtered []models.Registro
	for _, r := range m.records {
		if (f.Folio == "" || r.Folio == f.Folio) && (f.LoteID == "" || r.LoteID == f.LoteID) {
			filtered = append(filtered, r)
		}
	}
	return page(filtered, f.Paginacion), len(filtered), nil
}

func (m *MemoryBatchRepository) FindRecordByID(_ context.Context, id string) (models.Registro, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.records {
		if r.ID == id {
			return r, nil
		}
	}
	return models.Registro{}, models.ErrNoEncontrado
}

// devuelve una copia de la pagina pedida
func page[T any](items []T, p models.Paginacion) []T {
	start := min(p.Desplazamiento(), len(items))
	end := min(start+p.Tamano, len(items))
	result := make([]T, end-start)
	copy(result, items[start:end])
	return result
}
