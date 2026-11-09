package handlers

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/services"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

const maxBodyBytes = 5 << 20

type BatchService interface {
	Receive(ctx context.Context, in services.BatchInput) (services.BatchResult, error)
	Get(ctx context.Context, id string) (models.LoteDetalle, error)
	List(ctx context.Context, p models.Paginacion) (models.ListaLotes, error)
	ListRejections(ctx context.Context, batchID string) ([]models.Rechazo, error)
	ListRecords(ctx context.Context, f models.FiltroRegistros) (models.ListaRegistros, error)
	GetRecord(ctx context.Context, id string) (models.Registro, error)
}

type BatchHandler struct {
	service BatchService
}

func NewBatchHandler(service BatchService) *BatchHandler {
	return &BatchHandler{service: service}
}

func (h *BatchHandler) CreateBatch(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	in, err := readBatch(c)
	if err != nil {
		utils.RespondError(c, err)
		return
	}
	if fieldErrs := validateBatch(in); len(fieldErrs) > 0 {
		utils.RespondError(c, utils.ErrInvalidData.WithDetails(fieldErrs))
		return
	}

	res, err := h.service.Receive(c.Request.Context(), in)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	status := http.StatusCreated
	if res.Repetido {
		status = http.StatusOK
	}
	c.Header("Location", "/api/v1/lotes/"+res.Detalle.ID)
	c.JSON(status, res.Detalle)
}

func (h *BatchHandler) ListBatches(c *gin.Context) {
	p, err := bindPagination(c)
	if err != nil {
		utils.RespondError(c, err)
		return
	}
	list, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *BatchHandler) GetBatch(c *gin.Context) {
	id, ok := uuidParam(c)
	if !ok {
		return
	}
	detail, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *BatchHandler) ListRejections(c *gin.Context) {
	id, ok := uuidParam(c)
	if !ok {
		return
	}
	rejections, err := h.service.ListRejections(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.ListaRechazos{Elementos: rejections})
}

func (h *BatchHandler) ListRecords(c *gin.Context) {
	p, err := bindPagination(c)
	if err != nil {
		utils.RespondError(c, err)
		return
	}
	batchID := c.Query("lote_id")
	if batchID != "" && !isUUID(batchID) {
		utils.RespondError(c, utils.ErrInvalidData.WithDetails(map[string]string{"lote_id": "debe ser un UUID"}))
		return
	}
	list, err := h.service.ListRecords(c.Request.Context(), models.FiltroRegistros{
		Folio:      c.Query("folio"),
		LoteID:     batchID,
		Paginacion: p,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *BatchHandler) GetRecord(c *gin.Context) {
	id, ok := uuidParam(c)
	if !ok {
		return
	}
	record, err := h.service.GetRecord(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func readBatch(c *gin.Context) (services.BatchInput, error) {
	in := services.BatchInput{ClaveIdempotencia: c.GetHeader("Idempotency-Key")}

	mediaType, _, _ := mime.ParseMediaType(c.ContentType())
	switch mediaType {
	case "application/json":
		var req models.SolicitudLote
		if err := c.ShouldBindJSON(&req); err != nil {
			return in, bodyError(err, "el cuerpo no es un JSON válido para crear un lote")
		}
		in.Origen, in.ReferenciaExterna, in.Registros = req.Origen, req.ReferenciaExterna, req.Registros
	case "text/csv":
		records, err := parseCSV(c.Request.Body)
		if err != nil {
			return in, bodyError(err, err.Error())
		}
		in.Origen, in.ReferenciaExterna, in.Registros = c.Query("origen"), c.Query("referencia_externa"), records
	default:
		return in, utils.ErrUnsupportedMediaType
	}
	return in, nil
}

func bodyError(err error, message string) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return utils.ErrPayloadTooLarge
	}
	return utils.ErrInvalidData.WithDetails(map[string]string{"body": message})
}

func parseCSV(body io.Reader) ([]models.RegistroEntrada, error) {
	reader := csv.NewReader(body)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = false

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("el CSV debe tener un encabezado con folio, nombre y correo")
	}
	column := map[string]int{"folio": -1, "nombre": -1, "correo": -1}
	for i, name := range header {
		name = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\uFEFF")))
		if _, ok := column[name]; ok {
			column[name] = i
		}
	}
	for name, i := range column {
		if i < 0 {
			return nil, fmt.Errorf("al CSV le falta la columna %q", name)
		}
	}

	var records []models.RegistroEntrada
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				return nil, err
			}
			return nil, fmt.Errorf("CSV inválido: %v", err)
		}
		cell := func(name string) string {
			if i := column[name]; i < len(row) {
				return row[i]
			}
			return ""
		}
		records = append(records, models.RegistroEntrada{Folio: cell("folio"), Nombre: cell("nombre"), Correo: cell("correo")})
	}
	return records, nil
}

func validateBatch(in services.BatchInput) map[string]string {
	errs := map[string]string{}
	switch n := len(in.Registros); {
	case n == 0:
		errs["registros"] = "debe tener al menos un registro"
	case n > models.MaxRegistrosPorLote:
		errs["registros"] = fmt.Sprintf("admite como máximo %d registros", models.MaxRegistrosPorLote)
	}
	if len(in.ClaveIdempotencia) > 128 {
		errs["Idempotency-Key"] = "admite como máximo 128 caracteres"
	}
	if len(in.Origen) > models.MaxLargoCampo {
		errs["origen"] = fmt.Sprintf("admite como máximo %d caracteres", models.MaxLargoCampo)
	}
	if len(in.ReferenciaExterna) > models.MaxLargoCampo {
		errs["referencia_externa"] = fmt.Sprintf("admite como máximo %d caracteres", models.MaxLargoCampo)
	}
	for i, r := range in.Registros {
		if len(r.Folio) > models.MaxLargoCampo || len(r.Nombre) > models.MaxLargoCampo || len(r.Correo) > models.MaxLargoCampo {
			errs[fmt.Sprintf("registros[%d]", i)] = fmt.Sprintf("cada campo admite como máximo %d caracteres", models.MaxLargoCampo)
		}
	}
	return errs
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNoEncontrado):
		utils.RespondError(c, utils.ErrNotFound)
	case errors.Is(err, models.ErrClaveIdempotenciaConflicto):
		utils.RespondError(c, utils.ErrConflict.WithDetails(map[string]string{
			"Idempotency-Key": "ya se usó con un contenido distinto",
		}))
	default:
		utils.RespondError(c, err)
	}
}
