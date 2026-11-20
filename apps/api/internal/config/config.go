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
	APIEnv         string
	Port           string
	LogLevel       string
	AllowedOrigins []string

	PostgresURI      string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string

	MSDigitalizacionURL string
	MSReportesURL       string

	Service ServiceInfo
}

type ServiceInfo struct {
	Name         string
	Environment  string
	Role         string
	DependsOn    []string
	Consumers    []string
	Capabilities []string
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")
	if _, err := os.Stat("go.mod"); err == nil {
		_ = godotenv.Load("../../.env")
	}

	appEnv := getEnv("APP_ENV", "development")
	apiEnv := getEnv("API_ENV", "")
	if apiEnv == "" {
		if appEnv == "development" {
			apiEnv = "dev"
		} else {
			apiEnv = appEnv
		}
	}

	cfg := &Config{
		AppName:        getEnv("APP_NAME", "MS Gestor de Información"),
		AppEnv:         appEnv,
		APIEnv:         apiEnv,
		Port:           getEnv("BACKEND_PORT", "8080"),
		LogLevel:       strings.ToLower(getEnv("LOG_LEVEL", "info")),
		AllowedOrigins: getList("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),

		PostgresURI:      getEnv("POSTGRES_URI", "postgres://127.0.0.1:5432/gestor_info_db?sslmode=disable"),
		PostgresDatabase: getEnv("POSTGRES_DB", "gestor_info_db"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),

		MSDigitalizacionURL: getEnv("MS_DIGITALIZACION_URL", "http://localhost:8081"),
		MSReportesURL:       getEnv("MS_REPORTES_URL", "http://localhost:8082"),
	}

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
	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		errs = append(errs, fmt.Errorf("LOG_LEVEL debe ser debug, info, warn o error (valor: %q)", c.LogLevel))
	}
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		errs = append(errs, fmt.Errorf("BACKEND_PORT debe ser un puerto entre 1 y 65535 (valor: %q)", c.Port))
	}
	if len(c.AllowedOrigins) == 0 {
		errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS no puede quedar vacío"))
	}
	if (c.PostgresUser == "") != (c.PostgresPassword == "") {
		errs = append(errs, errors.New("POSTGRES_USER y POSTGRES_PASSWORD deben configurarse juntos"))
	}

	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

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

