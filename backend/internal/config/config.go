package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config almacena toda la configuración leída desde variables de entorno
type Config struct {
	AppName        string
	AppEnv         string
	Port           string
	GinMode        string
	LogLevel       string
	AllowedOrigins []string

	MongoURI      string
	MongoDatabase string
	MongoUser     string
	MongoPassword string

	MSDigitalizacionURL string
	MSReportesURL       string

	Service ServiceInfo
}

// cada equipo que use la plantilla la ajusta por entorno
type ServiceInfo struct {
	Name         string
	Environment  string
	Role         string
	DependsOn    []string
	Consumers    []string
	Capabilities []string
}

// lee .env y las variables del sistema, y valida el resultado
func Load() (*Config, error) {
	_ = godotenv.Load(".env")
	if _, err := os.Stat("go.mod"); err == nil {
		_ = godotenv.Load("../.env")
	}

	cfg := &Config{
		AppName:        getEnv("APP_NAME", "MS Gestor de Información"),
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("BACKEND_PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		LogLevel:       strings.ToLower(getEnv("LOG_LEVEL", "info")),
		AllowedOrigins: getList("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),

		MongoURI:      getEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
		MongoDatabase: getEnv("MONGO_DATABASE", "gestor_info_db"),
		MongoUser:     os.Getenv("MONGO_USER"),
		MongoPassword: os.Getenv("MONGO_PASSWORD"),

		MSDigitalizacionURL: getEnv("MS_DIGITALIZACION_URL", "http://localhost:8081"),
		MSReportesURL:       getEnv("MS_REPORTES_URL", "http://localhost:8082"),
	}

	// la identidad del servicio se define solo en .env, sin valores se muestran campos vacios
	cfg.Service = ServiceInfo{
		Name:         cfg.AppName,
		Environment:  cfg.AppEnv,
		Role:         getEnv("SERVICE_ROLE", ""),
		DependsOn:    getList("SERVICE_DEPENDS_ON", ""),
		Consumers:    getList("SERVICE_CONSUMERS", ""),
		Capabilities: getList("SERVICE_CAPABILITIES", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuración inválida: %w", err)
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	if !oneOf(c.AppEnv, "development", "test", "production") {
		errs = append(errs, fmt.Errorf("APP_ENV debe ser development, test o production (valor: %q)", c.AppEnv))
	}
	if !oneOf(c.GinMode, "debug", "release", "test") {
		errs = append(errs, fmt.Errorf("GIN_MODE debe ser debug, release o test (valor: %q)", c.GinMode))
	}
	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		errs = append(errs, fmt.Errorf("LOG_LEVEL debe ser debug, info, warn o error (valor: %q)", c.LogLevel))
	}
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		errs = append(errs, fmt.Errorf("BACKEND_PORT debe ser un puerto entre 1 y 65535 (valor: %q)", c.Port))
	}
	if len(c.AllowedOrigins) == 0 {
		errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS no puede quedar vacío"))
	}
	if (c.MongoUser == "") != (c.MongoPassword == "") {
		errs = append(errs, errors.New("MONGO_USER y MONGO_PASSWORD deben configurarse juntos"))
	}

	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

// getList lee una lista separada por comas, nunca devuelve nil, para que el JSON muestre []
func getList(key, fallback string) []string {
	parts := strings.Split(getEnv(key, fallback), ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func oneOf(value string, options ...string) bool {
	for _, o := range options {
		if value == o {
			return true
		}
	}
	return false
}
