package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/api"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/config"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/service"
	"github.com/mateo/ms-gestor-de-info/apps/api/internal/store"
)

const shutdownTimeout = 10 * time.Second

var version = "dev"

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

	db, err := store.ConnectDB(cfg)
	if err != nil {
		return fmt.Errorf("no se pudo crear el cliente de PostgreSQL, revisa POSTGRES_URI y las credenciales: %w", err)
	}
	defer db.Close()

	healthService := service.NewHealthService(cfg.Service, func(ctx context.Context) error {
		return db.Ping(ctx)
	})

	batchService := service.NewBatchService(store.NewMemoryBatchStore())
	if cfg.AppEnv == "development" {
		if err := seedDemoBatch(context.Background(), batchService); err != nil {
			return err
		}
	}

	router := api.NewRouter(
		api.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			Logger:         logger,
			PublishDocs:    cfg.APIEnv == "dev" || cfg.AppEnv != "production",
		},
		api.Dependencies{
			Health:  api.NewHealthHandler(healthService),
			Batches: api.NewBatchHandler(batchService),
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
			slog.String("version", version),
			slog.String("environment", cfg.AppEnv),
			slog.String("api_env", cfg.APIEnv),
			slog.String("addr", srv.Addr),
			slog.String("database", cfg.PostgresDatabase),
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

func seedDemoBatch(ctx context.Context, svc *service.BatchService) error {
	var req store.SolicitudLote
	if err := json.Unmarshal(api.DirtyBatchJSON, &req); err != nil {
		return fmt.Errorf("leyendo el lote de ejemplo: %w", err)
	}
	_, err := svc.Receive(ctx, service.BatchInput{
		Origen:            req.Origen,
		ReferenciaExterna: req.ReferenciaExterna,
		ClaveIdempotencia: "lote-demo",
		Registros:         req.Registros,
	})
	if err != nil {
		return fmt.Errorf("cargando el lote de ejemplo: %w", err)
	}
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

