package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
	"github.com/mateo/ms-gestor-de-info/backend/internal/database"
	"github.com/mateo/ms-gestor-de-info/backend/internal/handlers"
	"github.com/mateo/ms-gestor-de-info/backend/internal/repositories"
	"github.com/mateo/ms-gestor-de-info/backend/internal/routes"
	"github.com/mateo/ms-gestor-de-info/backend/internal/services"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("el servicio terminó con error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)
	gin.SetMode(cfg.GinMode)

	// El driver conecta en segundo plano y el diagnóstico verifica la disponibilidad.
	db, err := database.ConnectDB(cfg)
	if err != nil {
		return fmt.Errorf("no se pudo crear el cliente de MongoDB, revisa MONGO_URI y las credenciales: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.Disconnect(ctx); err != nil {
			logger.Error("error al cerrar el cliente de MongoDB", slog.String("error", err.Error()))
		}
	}()

	healthService := services.NewHealthService(cfg.Service, func(ctx context.Context) error {
		return db.Ping(ctx, nil)
	})

	batchService := services.NewBatchService(repositories.NewMemoryBatchRepository())
	if cfg.AppEnv == "development" {
		if err := seedDemoBatch(context.Background(), batchService); err != nil {
			return err
		}
	}

	router := routes.NewRouter(
		routes.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			Logger:         logger,
			PublishDocs:    cfg.AppEnv != "production",
		},
		routes.Dependencies{
			Health:  handlers.NewHealthHandler(healthService),
			Batches: handlers.NewBatchHandler(batchService),
		},
	)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("servidor iniciado",
			slog.String("service", cfg.AppName),
			slog.String("environment", cfg.AppEnv),
			slog.String("addr", srv.Addr),
			slog.String("database", cfg.MongoDatabase),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("el servidor HTTP falló: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("señal de término recibida, apagando el servicio")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("no se pudo apagar el servidor ordenadamente: %w", err)
	}
	logger.Info("servicio detenido correctamente")
	return nil
}

func newLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	_ = level.UnmarshalText([]byte(cfg.LogLevel))

	opts := &slog.HandlerOptions{Level: level}
	if cfg.AppEnv == "development" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
