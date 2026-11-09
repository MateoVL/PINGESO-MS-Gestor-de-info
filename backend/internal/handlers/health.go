package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

type HealthChecker interface {
	Check(ctx context.Context) models.HealthResponse
}

type HealthHandler struct {
	service HealthChecker
}

func NewHealthHandler(service HealthChecker) *HealthHandler {
	return &HealthHandler{service: service}
}

// 200 sano, 503 si falla
func (h *HealthHandler) Health(c *gin.Context) {
	result := h.service.Check(c.Request.Context())

	status := http.StatusOK
	if result.Status != models.HealthStatusHealthy {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, result)
}
