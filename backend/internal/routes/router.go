package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/handlers"
	"github.com/mateo/ms-gestor-de-info/backend/internal/middleware"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

// agrupa los handlers que el router necesita
type Dependencies struct {
	Health  *handlers.HealthHandler
	Batches *handlers.BatchHandler
}

// configuracion de los middlewares globales y de lo que se publica
type Options struct {
	AllowedOrigins []string
	Logger         *slog.Logger
	PublishDocs    bool
}

func NewRouter(opts Options, deps Dependencies) *gin.Engine {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true

	r.Use(
		middleware.RequestID(),
		middleware.Logger(logger, "/health", "/api/health"),
		middleware.Recovery(logger),
		middleware.CORS(opts.AllowedOrigins),
	)

	r.NoRoute(func(c *gin.Context) { utils.RespondError(c, utils.ErrNotFound) })
	r.NoMethod(func(c *gin.Context) { utils.RespondError(c, utils.ErrMethodNotAllowed) })

	registerHealthRoutes(r, deps.Health)

	api := r.Group("/api")
	registerHealthRoutes(api, deps.Health)

	v1 := api.Group("/v1")
	registerBatchRoutes(v1, deps.Batches)
	if opts.PublishDocs {
		registerMockRoutes(v1)
	}

	return r
}
