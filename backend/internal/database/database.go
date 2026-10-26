package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
)

// ConnectDB inicializa el pool de conexiones con reintentos para soportar el arranque de Docker
func ConnectDB(cfg *config.Config) (*sql.DB, error) {
	dsn := cfg.DBDSN()
	log.Printf("[DB] Conectando a PostgreSQL en %s:%s/%s...", cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al abrir conexión con postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Reintentos automáticos durante arranque
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		err = db.Ping()
		if err == nil {
			log.Println("[DB] ¡Conexión con PostgreSQL establecida exitosamente!")
			return db, nil
		}
		log.Printf("[DB] Intento %d/%d fallido (%v). Reintentando en 2s...", i, maxRetries, err)
		time.Sleep(2 * time.Second)
	}

	return db, fmt.Errorf("no se pudo conectar a la base de datos tras %d intentos: %w", maxRetries, err)
}
