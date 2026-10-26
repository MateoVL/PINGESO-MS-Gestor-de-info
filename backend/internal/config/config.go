package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config almacena toda la configuración leída desde variables de entorno
type Config struct {
	AppName             string
	AppEnv              string
	Port                string
	GinMode             string
	AllowedOrigins      []string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	MSDigitalizacionURL string
	MSReportesURL       string
}

// LoadConfig carga variables de entorno desde .env (si existe) y del sistema operativo
func LoadConfig() *Config {
	// Intentar cargar .env en la raíz o en el directorio actual
	_ = godotenv.Load(".env", "../../.env")

	cfg := &Config{
		AppName:             getEnv("APP_NAME", "MS Gestor de Información"),
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnv("BACKEND_PORT", "8080"),
		GinMode:             getEnv("GIN_MODE", "debug"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "gestor_user"),
		DBPassword:          getEnv("DB_PASSWORD", "gestor_password"),
		DBName:              getEnv("DB_NAME", "gestor_info_db"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		MSDigitalizacionURL: getEnv("MS_DIGITALIZACION_URL", "http://localhost:8081"),
		MSReportesURL:       getEnv("MS_REPORTES_URL", "http://localhost:8082"),
	}

	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	cfg.AllowedOrigins = parseCommaSeparated(origins)

	log.Printf("[Config] Entorno: %s | Puerto: %s | DB: %s@%s:%s/%s",
		cfg.AppEnv, cfg.Port, cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	return cfg
}

// DBDSN genera la cadena de conexión para PostgreSQL
func (c *Config) DBDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func parseCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
