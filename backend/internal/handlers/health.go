package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
)

type Handler struct {
	cfg       *config.Config
	db        *sql.DB
	startTime time.Time
}

func NewHandler(cfg *config.Config, db *sql.DB) *Handler {
	return &Handler{
		cfg:       cfg,
		db:        db,
		startTime: time.Now(),
	}
}

// HealthResponse estructura la respuesta del endpoint de salud
type HealthResponse struct {
	Status      string            `json:"status"`
	ServiceName string            `json:"service_name"`
	Environment string            `json:"environment"`
	Uptime      string            `json:"uptime"`
	Timestamp   string            `json:"timestamp"`
	Database    DatabaseStatus    `json:"database"`
	Role        string            `json:"role"`
	LineOfWork  LineOfWorkInfo    `json:"line_of_work"`
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
	dbStatus := DatabaseStatus{
		Engine: "PostgreSQL",
	}

	if h.db != nil {
		if err := h.db.Ping(); err == nil {
			dbStatus.Connected = true
			dbStatus.Message = "Base de datos conectada y operativa"
		} else {
			dbStatus.Connected = false
			dbStatus.Message = err.Error()
		}
	} else {
		dbStatus.Connected = false
		dbStatus.Message = "Sin conexión inicial a la base de datos"
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
		Role:        "MS Gestor de Información (Línea de Productos de Software)",
		LineOfWork: LineOfWorkInfo{
			DependsOn: []string{
				"MS Digitalización (Captura de documentos, OCR)",
			},
			Consumers: []string{
				"MS Reportes (Generación de reportes finales)",
			},
			Capabilities: []string{
				"Carga de información",
				"Procesamiento de datos",
				"Orden y estructura de datos",
			},
		},
	})
}

// InfoBase endpoint de base para comprobar que la ruta de API responde
func (h *Handler) InfoBase(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Base de MS Gestor de Info lista para implementar lógica de negocio.",
		"endpoints_planeados": []string{
			"POST /api/v1/info/ingest (Carga de información recibida de Digitalización)",
			"POST /api/v1/info/process (Procesamiento de datos cargados)",
			"GET  /api/v1/info/orders  (Consulta de información procesada y ordenada)",
		},
	})
}
