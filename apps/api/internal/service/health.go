package service

import (
	"context"
	"time"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
)

const (
	HealthStatusHealthy  = "healthy"
	HealthStatusDegraded = "degraded"
)

type HealthResponse struct {
	Status      string         `json:"status" doc:"Estado global del servicio (healthy o degraded)" example:"healthy"`
	ServiceName string         `json:"service_name" doc:"Nombre del microservicio" example:"MS Gestor de Información"`
	Environment string         `json:"environment" doc:"Entorno de ejecución" example:"development"`
	Uptime      string         `json:"uptime" doc:"Tiempo activo desde el arranque" example:"1m30s"`
	Timestamp   string         `json:"timestamp" doc:"Marca temporal ISO-8601 en UTC" example:"2026-11-20T12:00:00Z"`
	Database    DatabaseStatus `json:"database" doc:"Diagnóstico de conectividad con PostgreSQL"`
	Role        string         `json:"role" doc:"Rol dentro de la Línea de Productos de Software" example:"Gestor de Información"`
	LineOfWork  LineOfWorkInfo `json:"line_of_work" doc:"Relaciones upstream, downstream y capacidades"`
}

type DatabaseStatus struct {
	Connected bool   `json:"connected" doc:"Indica si PostgreSQL respondió al ping" example:"true"`
	Engine    string `json:"engine" doc:"Motor de base de datos" example:"PostgreSQL"`
	Message   string `json:"message" doc:"Mensaje descriptivo del estado" example:"Base de datos conectada y operativa"`
}

type LineOfWorkInfo struct {
	DependsOn    []string `json:"depends_on" nullable:"false" doc:"Servicios de los que depende (upstream)"`
	Consumers    []string `json:"consumers" nullable:"false" doc:"Servicios que consumen sus datos (downstream)"`
	Capabilities []string `json:"capabilities" nullable:"false" doc:"Capacidades expuestas por este microservicio"`
}

type PingFunc func(ctx context.Context) error

const databasePingTimeout = 2 * time.Second

type HealthService struct {
	info      config.ServiceInfo
	pingDB    PingFunc
	startedAt time.Time
	now       func() time.Time
}

func NewHealthService(info config.ServiceInfo, pingDB PingFunc) *HealthService {
	return &HealthService{
		info:      info,
		pingDB:    pingDB,
		startedAt: time.Now(),
		now:       time.Now,
	}
}

func (s *HealthService) Check(ctx context.Context) HealthResponse {
	ctx, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()

	db := DatabaseStatus{
		Engine:  "PostgreSQL",
		Message: "PostgreSQL no está disponible",
	}
	if err := s.pingDB(ctx); err == nil {
		db.Connected = true
		db.Message = "Base de datos conectada y operativa"
	}

	status := HealthStatusHealthy
	if !db.Connected {
		status = HealthStatusDegraded
	}

	now := s.now()
	return HealthResponse{
		Status:      status,
		ServiceName: s.info.Name,
		Environment: s.info.Environment,
		Uptime:      now.Sub(s.startedAt).Round(time.Second).String(),
		Timestamp:   now.UTC().Format(time.RFC3339),
		Database:    db,
		Role:        s.info.Role,
		LineOfWork: LineOfWorkInfo{
			DependsOn:    nonNil(s.info.DependsOn),
			Consumers:    nonNil(s.info.Consumers),
			Capabilities: nonNil(s.info.Capabilities),
		},
	}
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
