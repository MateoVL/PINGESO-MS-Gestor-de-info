package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/handlers"
)

func registerHealthRoutes(rg gin.IRoutes, h *handlers.HealthHandler) {
	rg.GET("/health", h.Health)
}

func registerBatchRoutes(rg *gin.RouterGroup, h *handlers.BatchHandler) {
	lotes := rg.Group("/lotes")
	lotes.POST("", h.CreateBatch)
	lotes.GET("", h.ListBatches)
	lotes.GET("/:id", h.GetBatch)
	lotes.GET("/:id/rechazos", h.ListRejections)

	registros := rg.Group("/registros")
	registros.GET("", h.ListRecords)
	registros.GET("/:id", h.GetRecord)
}

func registerMockRoutes(rg gin.IRoutes) {
	rg.GET("/mocks/datos-sucios", handlers.DirtyData)
}
