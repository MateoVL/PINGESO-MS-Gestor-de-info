package services

import (
	"context"
	"time"

	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

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

// consulta la base de datos y devuelve el diagnostico
func (s *HealthService) Check(ctx context.Context) models.HealthResponse {
	ctx, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()

	db := models.DatabaseStatus{
		Engine:  "MongoDB",
		Message: "MongoDB no está disponible",
	}
	if err := s.pingDB(ctx); err == nil {
		db.Connected = true
		db.Message = "Base de datos conectada y operativa"
	}

	status := models.HealthStatusHealthy
	if !db.Connected {
		status = models.HealthStatusDegraded
	}

	now := s.now()
	return models.HealthResponse{
		Status:      status,
		ServiceName: s.info.Name,
		Environment: s.info.Environment,
		Uptime:      now.Sub(s.startedAt).Round(time.Second).String(),
		Timestamp:   now.UTC().Format(time.RFC3339),
		Database:    db,
		Role:        s.info.Role,
		LineOfWork: models.LineOfWorkInfo{
			DependsOn:    nonNil(s.info.DependsOn),
			Consumers:    nonNil(s.info.Consumers),
			Capabilities: nonNil(s.info.Capabilities),
		},
	}
}

// evita que una lista vacia se publique como null en el JSON
func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
