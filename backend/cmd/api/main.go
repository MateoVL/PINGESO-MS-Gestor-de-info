package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
	"github.com/mateo/ms-gestor-de-info/backend/internal/database"
	"github.com/mateo/ms-gestor-de-info/backend/internal/handlers"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Conectar a la base de datos
	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Printf("[WARN] Base de datos no disponible inicialmente: %v", err)
	} else {
		defer db.Close()
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Configuración de CORS
	corsConfig := cors.DefaultConfig()
	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = cfg.AllowedOrigins
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	// Registrar Handlers
	h := handlers.NewHandler(cfg, db)

	// Rutas
	router.GET("/health", h.Health)
	api := router.Group("/api")
	{
		api.GET("/health", h.Health)
		api.GET("/info", h.InfoBase)
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Servidor en goroutine para apagado elegante (graceful shutdown)
	go func() {
		log.Printf("[Server] Iniciando %s en http://0.0.0.0%s", cfg.AppName, addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Server] Error fatal al iniciar: %v", err)
		}
	}()

	// Manejo de señales del SO (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Server] Señal de terminación recibida. Apagando servicio...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[Server] Forzado de apagado: %v", err)
	}
	log.Println("[Server] Servicio detenido correctamente.")
}
