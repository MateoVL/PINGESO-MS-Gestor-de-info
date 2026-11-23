package config

import (
	"errors"
	"fmt"
	"os"
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

	DatabaseURL string
	AuthJWKSURL string
	FrontendURL string
	TZ          string

	PostgresURI      string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string

	MSDigitalizacionURL string
	MSReportesURL       string
	DigitalizacionURL   string
	ReportesURL         string
	IdentidadURL        string

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
		} else if appEnv == "production" {
			apiEnv = "prod"
		} else {
			apiEnv = appEnv
		}
	}

	port := getEnv("API_PORT", "")
	if port == "" {
		port = getEnv("BACKEND_PORT", "")
	}

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		databaseURL = getEnv("POSTGRES_URI", "")
	}

	frontendURL := getEnv("FRONTEND_URL", "")
	allowedOrigins := getList("CORS_ALLOWED_ORIGINS", "")
	if frontendURL != "" && !contains(allowedOrigins, frontendURL) {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	digitalizacionURL := getEnv("DIGITALIZACION_URL", getEnv("MS_DIGITALIZACION_URL", "http://localhost:5151"))
	reportesURL := getEnv("REPORTES_URL", getEnv("MS_REPORTES_URL", "http://localhost:5111"))
	identidadURL := getEnv("IDENTIDAD_URL", "http://localhost:5101")

	cfg := &Config{
		AppName:        getEnv("APP_NAME", "MS Gestor de Información"),
		AppEnv:         appEnv,
		APIEnv:         apiEnv,
		Port:           port,
		LogLevel:       strings.ToLower(getEnv("LOG_LEVEL", "info")),
		AllowedOrigins: allowedOrigins,

		DatabaseURL: databaseURL,
		AuthJWKSURL: getEnv("AUTH_JWKS_URL", ""),
		FrontendURL: frontendURL,
		TZ:          getEnv("TZ", ""),

		PostgresURI:      databaseURL,
		PostgresDatabase: getEnv("POSTGRES_DB", "gestor_info_db"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),

		DigitalizacionURL:   digitalizacionURL,
		ReportesURL:         reportesURL,
		IdentidadURL:        identidadURL,
		MSDigitalizacionURL: digitalizacionURL,
		MSReportesURL:       reportesURL,
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

	// Variables obligatorias según AGENTS.md sección 5
	if c.Port != "8080" {
		errs = append(errs, fmt.Errorf("API_PORT es obligatoria y debe ser siempre 8080 (valor: %q)", c.Port))
	}
	if !oneOf(c.APIEnv, "dev", "test", "prod") {
		errs = append(errs, fmt.Errorf("API_ENV es obligatoria y debe ser dev, test o prod (valor: %q)", c.APIEnv))
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		errs = append(errs, errors.New("DATABASE_URL es obligatoria"))
	}
	if strings.TrimSpace(c.AuthJWKSURL) == "" {
		errs = append(errs, errors.New("AUTH_JWKS_URL es obligatoria"))
	}
	if strings.TrimSpace(c.FrontendURL) == "" {
		errs = append(errs, errors.New("FRONTEND_URL es obligatoria"))
	}
	if c.TZ != "America/Santiago" {
		errs = append(errs, fmt.Errorf("TZ es obligatoria y debe ser America/Santiago (valor: %q)", c.TZ))
	}

	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		errs = append(errs, fmt.Errorf("LOG_LEVEL debe ser debug, info, warn o error (valor: %q)", c.LogLevel))
	}
	if len(c.AllowedOrigins) == 0 {
		errs = append(errs, errors.New("AllowedOrigins no puede quedar vacío (define FRONTEND_URL o CORS_ALLOWED_ORIGINS)"))
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
	val := getEnv(key, fallback)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
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

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
