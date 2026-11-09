package models

// estados posibles del diagnostico
const (
	HealthStatusHealthy  = "healthy"
	HealthStatusDegraded = "degraded"
)

// diagnostico publico del servicio
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

// resume la disponibilidad de la base de datos sin exponer errores internos
type DatabaseStatus struct {
	Connected bool   `json:"connected"`
	Engine    string `json:"engine"`
	Message   string `json:"message"`
}

// indica de quien depende el servicio, quien lo consume y que ofrece
type LineOfWorkInfo struct {
	DependsOn    []string `json:"depends_on"`
	Consumers    []string `json:"consumers"`
	Capabilities []string `json:"capabilities"`
}
