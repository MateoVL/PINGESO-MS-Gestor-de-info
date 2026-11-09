package config

import (
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
	MongoURI            string
	MongoDatabase       string
	MongoUser           string
	MongoPassword       string
	MSDigitalizacionURL string
	MSReportesURL       string
}

// LoadConfig carga variables de entorno desde .env (si existe) y del sistema operativo
func LoadConfig() *Config {
	_ = godotenv.Load(".env")
	if _, err := os.Stat("go.mod"); err == nil {
		_ = godotenv.Load("../.env")
	}

	cfg := &Config{
		AppName:             getEnv("APP_NAME", "MS Gestor de Información"),
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnv("BACKEND_PORT", "8080"),
		GinMode:             getEnv("GIN_MODE", "debug"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
		MongoDatabase:       getEnv("MONGO_DATABASE", "gestor_info_db"),
		MongoUser:           os.Getenv("MONGO_USER"),
		MongoPassword:       os.Getenv("MONGO_PASSWORD"),
		MSDigitalizacionURL: getEnv("MS_DIGITALIZACION_URL", "http://localhost:8081"),
		MSReportesURL:       getEnv("MS_REPORTES_URL", "http://localhost:8082"),
	}

	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	cfg.AllowedOrigins = parseCommaSeparated(origins)

	log.Printf("[Config] Entorno: %s | Puerto: %s | MongoDB: %s", cfg.AppEnv, cfg.Port, cfg.MongoDatabase)

	return cfg
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
