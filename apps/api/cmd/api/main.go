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

	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/api"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/config"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/service"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/store"
)

const (
	shutdownTimeout = 10 * time.Second
	nombreServicio  = "gestor-info"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		if err := checkLocalHealth("http://127.0.0.1:8080/healthz"); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := run(); err != nil {
		slog.Error("el servicio terminó con error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func checkLocalHealth(url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck respondió HTTP %d", resp.StatusCode)
	}
	return nil
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
		return fmt.Errorf("no se pudo crear el cliente de PostgreSQL, revisa DATABASE_URL y las credenciales: %w", err)
	}
	defer db.Close()

	healthService := service.NewHealthService(cfg.Service, func(ctx context.Context) error {
		return db.Ping(ctx)
	})

	batchService := service.NewBatchService(store.NewPostgresBatchStore(db))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	authenticator, authErr := api.NewJWKSAuthenticator(ctx, cfg.AuthJWKSURL)
	if authErr != nil {
		logger.Warn("JWKS no disponible al iniciar; las peticiones protegidas requerirán JWKS activo cuando RequireAuth esté habilitado",
			slog.String("jwks_url", cfg.AuthJWKSURL),
			slog.String("error", authErr.Error()),
		)
	}

	router := api.NewRouter(
		api.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			Logger:         logger,
			PublishDocs:    cfg.APIEnv == "dev",
			Authenticator:  authenticator,
			RequireAuth:    cfg.APIEnv == "prod",
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

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("servidor iniciado",
			slog.String("nombre", cfg.AppName),
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

func newLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	_ = level.UnmarshalText([]byte(cfg.LogLevel))

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).With(
		slog.String("servicio", nombreServicio),
		slog.String("version", version),
	)
}
