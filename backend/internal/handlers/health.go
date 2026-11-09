package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
)

type Handler struct {
	cfg          *config.Config
	pingDatabase func(context.Context) error
	startTime    time.Time
}

func NewHandler(cfg *config.Config, pingDatabase func(context.Context) error) *Handler {
	return &Handler{
		cfg:          cfg,
		pingDatabase: pingDatabase,
		startTime:    time.Now(),
	}
}

// HealthResponse estructura la respuesta del endpoint de salud
type HealthResponse struct {
	Status      string         `json:"status"`
	ServiceName string         `json:"service_name"`
	Environment string         `json:"environment"`
	Uptime      string         `json:"uptime"`
	Timestamp   string         `json:"timestamp"`
	Database    DatabaseStatus `json:"database"`
	Role        string         `json:"role"`
	LineOfWork  LineOfWorkInfo `json:"line_of_work"`
}

type DatabaseStatus struct {
	Connected bool   `json:"connected"`
	Engine    string `json:"engine"`
	Message   string `json:"message"`
}

type LineOfWorkInfo struct {
	DependsOn    []string `json:"depends_on"`
	Consumers    []string `json:"consumers"`
	Capabilities []string `json:"capabilities"`
}

// Health verifica el estado del microservicio y la conexión a la base de datos
func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := DatabaseStatus{
		Engine:  "MongoDB",
		Message: "MongoDB no está disponible",
	}

	if err := h.pingDatabase(ctx); err == nil {
		dbStatus.Connected = true
		dbStatus.Message = "Base de datos conectada y operativa"
	}

	overallStatus := "healthy"
	httpStatus := http.StatusOK
	if !dbStatus.Connected {
		overallStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, HealthResponse{
		Status:      overallStatus,
		ServiceName: h.cfg.AppName,
		Environment: h.cfg.AppEnv,
		Uptime:      time.Since(h.startTime).Round(time.Second).String(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Database:    dbStatus,
		Role:        "Grupo 2 · PT2 · Pipeline de Información",
		LineOfWork: LineOfWorkInfo{
			DependsOn: []string{
				"PT5 · Extractor de Datos OCR",
			},
			Consumers: []string{
				"PT1 · Motor de Reportes",
			},
			Capabilities: []string{
				"Carga de información",
				"Limpieza y validación de datos",
				"Inserción masiva de datos",
			},
		},
	})
}

// InfoBase endpoint de base para comprobar que la ruta de API responde
func (h *Handler) InfoBase(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Grupo 2 · Pipeline de Información · Hito 0",
		"endpoints_planeados": []string{
			"POST /api/v1/info/ingest (Carga de información recibida de Digitalización)",
			"POST /api/v1/info/process (Procesamiento de datos cargados)",
			"GET  /api/v1/info/orders  (Consulta de información procesada y ordenada)",
		},
	})
}
