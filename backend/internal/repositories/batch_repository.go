package repositories

import (
	"context"

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

// guarda lotes, registros aceptados y rechazos
type BatchRepository interface {
	Create(ctx context.Context, batch models.NuevoLote) error
	FindByID(ctx context.Context, id string) (models.Lote, error)
	FindByIdempotencyKey(ctx context.Context, key string) (models.Lote, error)
	List(ctx context.Context, p models.Paginacion) ([]models.Lote, int, error)
	ListRejections(ctx context.Context, batchID string) ([]models.Rechazo, error)
	ListRecords(ctx context.Context, f models.FiltroRegistros) ([]models.Registro, int, error)
	FindRecordByID(ctx context.Context, id string) (models.Registro, error)
}
