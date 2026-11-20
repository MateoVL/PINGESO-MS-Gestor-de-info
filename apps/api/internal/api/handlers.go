package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/service"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

const maxBodyBytes = 5 << 20

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

type HealthChecker interface {
	Check(ctx context.Context) service.HealthResponse
}

type BatchService interface {
	Receive(ctx context.Context, in service.BatchInput) (service.BatchResult, error)
	Get(ctx context.Context, id string) (store.LoteDetalle, error)
	List(ctx context.Context, p store.Paginacion) (store.ListaLotes, error)
	ListRejections(ctx context.Context, batchID string) ([]store.Rechazo, error)
	ListRecords(ctx context.Context, f store.FiltroRegistros) (store.ListaRegistros, error)
	GetRecord(ctx context.Context, id string) (store.Registro, error)
}

type HealthHandler struct {
	service HealthChecker
}

func NewHealthHandler(s HealthChecker) *HealthHandler {
	return &HealthHandler{service: s}
}

type BatchHandler struct {
	service BatchService
}

func NewBatchHandler(s BatchService) *BatchHandler {
	return &BatchHandler{service: s}
}

type Dependencies struct {
	Health  *HealthHandler
	Batches *BatchHandler
}

type Options struct {
	AllowedOrigins []string
	Logger         *slog.Logger
	PublishDocs    bool
}

// Structs de contrato Huma (API-first OpenAPI 3.1)

type HealthOutput struct {
	Status int                    `header:"-"`
	Body   service.HealthResponse `json:"body"`
}

type LivenessStatus struct {
	Status string `json:"status" doc:"Estado del proceso HTTP" example:"ok"`
}

type LivenessOutput struct {
	Status int            `header:"-"`
	Body   LivenessStatus `json:"body"`
}

type CreateBatchInput struct {
	IdempotencyKey string              `header:"Idempotency-Key" doc:"Clave de idempotencia para evitar duplicados en reintentos" maxLength:"128" example:"lote-2026-001"`
	Body           store.SolicitudLote `json:"body"`
}

type CreateBatchOutput struct {
	Status   int               `header:"-"`
	Location string            `header:"Location" doc:"URI del lote creado"`
	Body     store.LoteDetalle `json:"body"`
}

type ListBatchesInput struct {
	Pagina       int `query:"pagina" default:"1" minimum:"1" doc:"Número de página (1-indexed)" example:"1"`
	TamanoPagina int `query:"tamano_pagina" default:"50" minimum:"1" maximum:"200" doc:"Cantidad de lotes por página" example:"50"`
}

type ListBatchesOutput struct {
	Body store.ListaLotes `json:"body"`
}

type GetByIDInput struct {
	ID string `path:"id" format:"uuid" doc:"UUID del recurso" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type GetBatchOutput struct {
	Body store.LoteDetalle `json:"body"`
}

type ListRejectionsOutput struct {
	Body store.ListaRechazos `json:"body"`
}

type ListRecordsInput struct {
	Pagina       int    `query:"pagina" default:"1" minimum:"1" doc:"Número de página (1-indexed)" example:"1"`
	TamanoPagina int    `query:"tamano_pagina" default:"50" minimum:"1" maximum:"200" doc:"Cantidad de registros por página" example:"50"`
	Folio        string `query:"folio" doc:"Filtro opcional por folio exacto (se normaliza a mayúsculas)" example:"DOC-0001"`
	LoteID       string `query:"lote_id" format:"uuid" doc:"Filtro opcional por UUID de lote" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type ListRecordsOutput struct {
	Body store.ListaRegistros `json:"body"`
}

type GetRecordOutput struct {
	Body store.Registro `json:"body"`
}

type DirtyMocksInput struct {
	Cantidad int    `query:"cantidad" default:"20" minimum:"1" maximum:"5000" doc:"Cantidad de registros simulados a generar" example:"20"`
	Formato  string `query:"formato" default:"json" enum:"json,csv" doc:"Formato de salida (json o csv)" example:"json"`
	Semilla  uint64 `query:"semilla" doc:"Semilla opcional para generación determinista" example:"7"`
}

type DirtyMocksOutput struct {
	Body store.SolicitudLote `json:"body"`
}

// NewHumaAPI registra las operaciones tipadas en Huma v2 para generar el contrato OpenAPI 3.1 y servir /api/docs en API_ENV=dev
func NewHumaAPI(mux *http.ServeMux, opts Options, deps Dependencies) huma.API {
	cfg := huma.DefaultConfig("MS Gestor de Información API", "1.0.0")
	cfg.Info.Description = "Contrato OpenAPI 3.1 del microservicio PT2 (Gestor de Información · PINGESO). Recibe lotes del Extractor de Datos OCR (PT5), limpia, valida y almacena registros en PostgreSQL para el Motor de Reportes (PT1)."
	if opts.PublishDocs {
		cfg.DocsPath = "/api/docs"
		cfg.OpenAPIPath = "/api/openapi"
	} else {
		cfg.DocsPath = ""
		cfg.OpenAPIPath = ""
	}

	api := humago.New(mux, cfg)

	huma.Register(api, huma.Operation{
		OperationID: "get-healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Liveness probe del contenedor API (vivo)",
		Tags:        []string{"Diagnóstico"},
	}, func(_ context.Context, _ *struct{}) (*LivenessOutput, error) {
		return &LivenessOutput{
			Status: http.StatusOK,
			Body:   LivenessStatus{Status: "ok"},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-readyz",
		Method:      http.MethodGet,
		Path:        "/readyz",
		Summary:     "Readiness probe del contenedor API con verificación de PostgreSQL",
		Tags:        []string{"Diagnóstico"},
	}, func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		if deps.Health == nil {
			return &HealthOutput{Status: http.StatusOK}, nil
		}
		res := deps.Health.service.Check(ctx)
		status := http.StatusOK
		if res.Status != service.HealthStatusHealthy {
			status = http.StatusServiceUnavailable
		}
		return &HealthOutput{Status: status, Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/api/health",
		Summary:     "Diagnóstico de salud del servicio y PostgreSQL",
		Tags:        []string{"Diagnóstico"},
	}, func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		if deps.Health == nil {
			return &HealthOutput{Status: http.StatusOK}, nil
		}
		res := deps.Health.service.Check(ctx)
		status := http.StatusOK
		if res.Status != service.HealthStatusHealthy {
			status = http.StatusServiceUnavailable
		}
		return &HealthOutput{Status: status, Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-lote",
		Method:        http.MethodPost,
		Path:          "/api/v1/lotes",
		Summary:       "Recibir, limpiar y validar un lote de registros (JSON o CSV)",
		Tags:          []string{"Lotes"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateBatchInput) (*CreateBatchOutput, error) {
		if deps.Batches == nil {
			return nil, huma.Error500InternalServerError("servicio no configurado")
		}
		res, err := deps.Batches.service.Receive(ctx, service.BatchInput{
			Origen:            in.Body.Origen,
			ReferenciaExterna: in.Body.ReferenciaExterna,
			ClaveIdempotencia: in.IdempotencyKey,
			Registros:         in.Body.Registros,
		})
		if err != nil {
			return nil, err
		}
		status := http.StatusCreated
		if res.Repetido {
			status = http.StatusOK
		}
		return &CreateBatchOutput{
			Status:   status,
			Location: "/api/v1/lotes/" + res.Detalle.ID,
			Body:     res.Detalle,
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-lotes",
		Method:      http.MethodGet,
		Path:        "/api/v1/lotes",
		Summary:     "Listar lotes procesados del más reciente al más antiguo",
		Tags:        []string{"Lotes"},
	}, func(ctx context.Context, in *ListBatchesInput) (*ListBatchesOutput, error) {
		if deps.Batches == nil {
			return &ListBatchesOutput{}, nil
		}
		list, err := deps.Batches.service.List(ctx, store.Paginacion{Numero: in.Pagina, Tamano: in.TamanoPagina})
		if err != nil {
			return nil, err
		}
		return &ListBatchesOutput{Body: list}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-lote",
		Method:      http.MethodGet,
		Path:        "/api/v1/lotes/{id}",
		Summary:     "Obtener el detalle de un lote y sus rechazos",
		Tags:        []string{"Lotes"},
	}, func(ctx context.Context, in *GetByIDInput) (*GetBatchOutput, error) {
		if deps.Batches == nil {
			return &GetBatchOutput{}, nil
		}
		detail, err := deps.Batches.service.Get(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &GetBatchOutput{Body: detail}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-lote-rechazos",
		Method:      http.MethodGet,
		Path:        "/api/v1/lotes/{id}/rechazos",
		Summary:     "Listar los registros rechazados de un lote",
		Tags:        []string{"Lotes"},
	}, func(ctx context.Context, in *GetByIDInput) (*ListRejectionsOutput, error) {
		if deps.Batches == nil {
			return &ListRejectionsOutput{}, nil
		}
		rej, err := deps.Batches.service.ListRejections(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &ListRejectionsOutput{Body: store.ListaRechazos{Elementos: rej}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-registros",
		Method:      http.MethodGet,
		Path:        "/api/v1/registros",
		Summary:     "Listar registros limpios y validados",
		Tags:        []string{"Registros"},
	}, func(ctx context.Context, in *ListRecordsInput) (*ListRecordsOutput, error) {
		if deps.Batches == nil {
			return &ListRecordsOutput{}, nil
		}
		list, err := deps.Batches.service.ListRecords(ctx, store.FiltroRegistros{
			Folio:      in.Folio,
			LoteID:     in.LoteID,
			Paginacion: store.Paginacion{Numero: in.Pagina, Tamano: in.TamanoPagina},
		})
		if err != nil {
			return nil, err
		}
		return &ListRecordsOutput{Body: list}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-registro",
		Method:      http.MethodGet,
		Path:        "/api/v1/registros/{id}",
		Summary:     "Obtener un registro validado por su UUID",
		Tags:        []string{"Registros"},
	}, func(ctx context.Context, in *GetByIDInput) (*GetRecordOutput, error) {
		if deps.Batches == nil {
			return &GetRecordOutput{}, nil
		}
		rec, err := deps.Batches.service.GetRecord(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &GetRecordOutput{Body: rec}, nil
	})

	if opts.PublishDocs {
		huma.Register(api, huma.Operation{
			OperationID: "get-mocks-datos-sucios",
			Method:      http.MethodGet,
			Path:        "/api/v1/mocks/datos-sucios",
			Summary:     "Generar lote simulado de datos sucios de OCR (solo entornos de desarrollo/prueba)",
			Tags:        []string{"Mocks"},
		}, func(_ context.Context, in *DirtyMocksInput) (*DirtyMocksOutput, error) {
			seed := in.Semilla
			if seed == 0 {
				seed = uint64(time.Now().UnixNano())
			}
			records := GenerateDirty(in.Cantidad, seed)
			return &DirtyMocksOutput{
				Body: store.SolicitudLote{
					Origen:            "Extractor OCR",
					ReferenciaExterna: "mock-" + strconv.FormatUint(seed, 10),
					Registros:         records,
				},
			}, nil
		})
	}

	return api
}

// NewRouter construye el handler HTTP completo con Huma para OpenAPI/docs y soporte nativo de JSON + CSV y formato de errores
func NewRouter(opts Options, deps Dependencies) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	humaMux := http.NewServeMux()
	_ = NewHumaAPI(humaMux, opts, deps)

	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Documentación interactiva Huma (/api/docs y /api/openapi.json | /api/openapi.yaml)
		if opts.PublishDocs && (path == "/api/docs" || strings.HasPrefix(path, "/api/openapi")) {
			humaMux.ServeHTTP(w, r)
			return
		}

		// Endpoint de prueba para panic en tests
		if path == "/panic" {
			if r.Method != http.MethodGet {
				RespondError(w, r, ErrMethodNotAllowed)
				return
			}
			panic("falla inesperada")
		}

		// Liveness check (vivo, sin PostgreSQL)
		if path == "/healthz" {
			if r.Method != http.MethodGet {
				RespondError(w, r, ErrMethodNotAllowed)
				return
			}
			writeJSON(w, http.StatusOK, LivenessStatus{Status: "ok"})
			return
		}

		// Readiness & Health checks (con PostgreSQL)
		if path == "/readyz" || path == "/health" || path == "/api/health" {
			if r.Method != http.MethodGet {
				RespondError(w, r, ErrMethodNotAllowed)
				return
			}
			deps.Health.handleHealth(w, r)
			return
		}

		// Lotes: /api/v1/lotes
		if path == "/api/v1/lotes" {
			switch r.Method {
			case http.MethodPost:
				deps.Batches.handleCreateBatch(w, r)
			case http.MethodGet:
				deps.Batches.handleListBatches(w, r)
			default:
				RespondError(w, r, ErrMethodNotAllowed)
			}
			return
		}

		// Lotes por ID y rechazos: /api/v1/lotes/{id} y /api/v1/lotes/{id}/rechazos
		if strings.HasPrefix(path, "/api/v1/lotes/") {
			rest := strings.TrimPrefix(path, "/api/v1/lotes/")
			parts := strings.Split(rest, "/")
			if len(parts) == 1 && parts[0] != "" {
				if r.Method != http.MethodGet {
					RespondError(w, r, ErrMethodNotAllowed)
					return
				}
				deps.Batches.handleGetBatch(w, r, parts[0])
				return
			}
			if len(parts) == 2 && parts[0] != "" && parts[1] == "rechazos" {
				if r.Method != http.MethodGet {
					RespondError(w, r, ErrMethodNotAllowed)
					return
				}
				deps.Batches.handleListRejections(w, r, parts[0])
				return
			}
			RespondError(w, r, ErrNotFound)
			return
		}

		// Registros: /api/v1/registros
		if path == "/api/v1/registros" {
			if r.Method != http.MethodGet {
				RespondError(w, r, ErrMethodNotAllowed)
				return
			}
			deps.Batches.handleListRecords(w, r)
			return
		}

		// Registro por ID: /api/v1/registros/{id}
		if strings.HasPrefix(path, "/api/v1/registros/") {
			id := strings.TrimPrefix(path, "/api/v1/registros/")
			if id == "" || strings.Contains(id, "/") {
				RespondError(w, r, ErrNotFound)
				return
			}
			if r.Method != http.MethodGet {
				RespondError(w, r, ErrMethodNotAllowed)
				return
			}
			deps.Batches.handleGetRecord(w, r, id)
			return
		}

		// Mocks: /api/v1/mocks/datos-sucios (solo cuando PublishDocs == true)
		if opts.PublishDocs && path == "/api/v1/mocks/datos-sucios" {
			if r.Method != http.MethodGet {
				RespondError(w, r, ErrMethodNotAllowed)
				return
			}
			handleDirtyData(w, r)
			return
		}

		RespondError(w, r, ErrNotFound)
	})

	var handler http.Handler = mainHandler
	handler = CORSMiddleware(opts.AllowedOrigins)(handler)
	handler = RecoveryMiddleware(logger)(handler)
	handler = LoggerMiddleware(logger, "/health", "/api/health", "/healthz", "/readyz")(handler)
	handler = RequestIDMiddleware(handler)
	return handler
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	result := h.service.Check(r.Context())
	status := http.StatusOK
	if result.Status != service.HealthStatusHealthy {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, result)
}

func (h *BatchHandler) handleCreateBatch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	in, err := readBatch(r)
	if err != nil {
		RespondError(w, r, err)
		return
	}
	if fieldErrs := validateBatch(in); len(fieldErrs) > 0 {
		RespondError(w, r, ErrInvalidData.WithDetails(fieldErrs))
		return
	}

	res, err := h.service.Receive(r.Context(), in)
	if err != nil {
		respondServiceError(w, r, err)
		return
	}
	status := http.StatusCreated
	if res.Repetido {
		status = http.StatusOK
	}
	w.Header().Set("Location", "/api/v1/lotes/"+res.Detalle.ID)
	writeJSON(w, status, res.Detalle)
}

func (h *BatchHandler) handleListBatches(w http.ResponseWriter, r *http.Request) {
	p, err := bindPagination(r)
	if err != nil {
		RespondError(w, r, err)
		return
	}
	list, err := h.service.List(r.Context(), p)
	if err != nil {
		respondServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *BatchHandler) handleGetBatch(w http.ResponseWriter, r *http.Request, id string) {
	if !isUUID(id) {
		RespondError(w, r, ErrInvalidData.WithDetails(map[string]string{"id": "debe ser un UUID"}))
		return
	}
	detail, err := h.service.Get(r.Context(), id)
	if err != nil {
		respondServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *BatchHandler) handleListRejections(w http.ResponseWriter, r *http.Request, id string) {
	if !isUUID(id) {
		RespondError(w, r, ErrInvalidData.WithDetails(map[string]string{"id": "debe ser un UUID"}))
		return
	}
	rejections, err := h.service.ListRejections(r.Context(), id)
	if err != nil {
		respondServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, store.ListaRechazos{Elementos: rejections})
}

func (h *BatchHandler) handleListRecords(w http.ResponseWriter, r *http.Request) {
	p, err := bindPagination(r)
	if err != nil {
		RespondError(w, r, err)
		return
	}
	q := r.URL.Query()
	batchID := q.Get("lote_id")
	if batchID != "" && !isUUID(batchID) {
		RespondError(w, r, ErrInvalidData.WithDetails(map[string]string{"lote_id": "debe ser un UUID"}))
		return
	}
	list, err := h.service.ListRecords(r.Context(), store.FiltroRegistros{
		Folio:      q.Get("folio"),
		LoteID:     batchID,
		Paginacion: p,
	})
	if err != nil {
		respondServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *BatchHandler) handleGetRecord(w http.ResponseWriter, r *http.Request, id string) {
	if !isUUID(id) {
		RespondError(w, r, ErrInvalidData.WithDetails(map[string]string{"id": "debe ser un UUID"}))
		return
	}
	record, err := h.service.GetRecord(r.Context(), id)
	if err != nil {
		respondServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func handleDirtyData(w http.ResponseWriter, r *http.Request) {
	errs := map[string]string{}
	count, ok := intQuery(r, "cantidad", 20, 1, store.MaxRegistrosPorLote)
	if !ok {
		errs["cantidad"] = "debe ser un entero entre 1 y 5000"
	}
	q := r.URL.Query()
	format := q.Get("formato")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		errs["formato"] = "debe ser json o csv"
	}
	seed := uint64(time.Now().UnixNano())
	if q.Has("semilla") {
		n, err := strconv.ParseUint(q.Get("semilla"), 10, 64)
		if err != nil {
			errs["semilla"] = "debe ser un entero mayor o igual a 0"
		}
		seed = n
	}
	if len(errs) > 0 {
		RespondError(w, r, ErrInvalidData.WithDetails(errs))
		return
	}

	records := GenerateDirty(count, seed)
	if format == "csv" {
		data, err := ToCSV(records)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	writeJSON(w, http.StatusOK, store.SolicitudLote{
		Origen:            "Extractor OCR",
		ReferenciaExterna: "mock-" + strconv.FormatUint(seed, 10),
		Registros:         records,
	})
}

func readBatch(r *http.Request) (service.BatchInput, error) {
	in := service.BatchInput{ClaveIdempotencia: r.Header.Get("Idempotency-Key")}

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch mediaType {
	case "application/json":
		var req store.SolicitudLote
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			return in, bodyError(err, "el cuerpo no es un JSON válido para crear un lote")
		}
		in.Origen, in.ReferenciaExterna, in.Registros = req.Origen, req.ReferenciaExterna, req.Registros
	case "text/csv":
		records, err := parseCSV(r.Body)
		if err != nil {
			return in, bodyError(err, err.Error())
		}
		q := r.URL.Query()
		in.Origen, in.ReferenciaExterna, in.Registros = q.Get("origen"), q.Get("referencia_externa"), records
	default:
		return in, ErrUnsupportedMediaType
	}
	return in, nil
}

func bodyError(err error, message string) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return ErrPayloadTooLarge
	}
	return ErrInvalidData.WithDetails(map[string]string{"body": message})
}

func parseCSV(body io.Reader) ([]store.RegistroEntrada, error) {
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

	var records []store.RegistroEntrada
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
		records = append(records, store.RegistroEntrada{Folio: cell("folio"), Nombre: cell("nombre"), Correo: cell("correo")})
	}
	return records, nil
}

func validateBatch(in service.BatchInput) map[string]string {
	errs := map[string]string{}
	switch n := len(in.Registros); {
	case n == 0:
		errs["registros"] = "debe tener al menos un registro"
	case n > store.MaxRegistrosPorLote:
		errs["registros"] = fmt.Sprintf("admite como máximo %d registros", store.MaxRegistrosPorLote)
	}
	if len(in.ClaveIdempotencia) > 128 {
		errs["Idempotency-Key"] = "admite como máximo 128 caracteres"
	}
	if len(in.Origen) > store.MaxLargoCampo {
		errs["origen"] = fmt.Sprintf("admite como máximo %d caracteres", store.MaxLargoCampo)
	}
	if len(in.ReferenciaExterna) > store.MaxLargoCampo {
		errs["referencia_externa"] = fmt.Sprintf("admite como máximo %d caracteres", store.MaxLargoCampo)
	}
	for i, r := range in.Registros {
		if len(r.Folio) > store.MaxLargoCampo || len(r.Nombre) > store.MaxLargoCampo || len(r.Correo) > store.MaxLargoCampo {
			errs[fmt.Sprintf("registros[%d]", i)] = fmt.Sprintf("cada campo admite como máximo %d caracteres", store.MaxLargoCampo)
		}
	}
	return errs
}

func bindPagination(r *http.Request) (store.Paginacion, error) {
	errs := map[string]string{}
	number, ok := intQuery(r, "pagina", 1, 1, 1<<20)
	if !ok {
		errs["pagina"] = "debe ser un entero mayor o igual a 1"
	}
	size, ok := intQuery(r, "tamano_pagina", store.TamanoPaginaPorDefecto, 1, store.TamanoPaginaMaximo)
	if !ok {
		errs["tamano_pagina"] = fmt.Sprintf("debe ser un entero entre 1 y %d", store.TamanoPaginaMaximo)
	}
	if len(errs) > 0 {
		return store.Paginacion{}, ErrInvalidData.WithDetails(errs)
	}
	return store.Paginacion{Numero: number, Tamano: size}, nil
}

func intQuery(r *http.Request, name string, def, lo, hi int) (int, bool) {
	q := r.URL.Query()
	if !q.Has(name) {
		return def, true
	}
	n, err := strconv.Atoi(q.Get(name))
	if err != nil || n < lo || n > hi {
		return 0, false
	}
	return n, true
}

func respondServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNoEncontrado):
		RespondError(w, r, ErrNotFound)
	case errors.Is(err, store.ErrClaveIdempotenciaConflicto):
		RespondError(w, r, ErrConflict.WithDetails(map[string]string{
			"Idempotency-Key": "ya se usó con un contenido distinto",
		}))
	default:
		RespondError(w, r, err)
	}
}

