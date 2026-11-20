package config

import (
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("API_PORT", "8080")
	t.Setenv("API_ENV", "dev")
	t.Setenv("DATABASE_URL", "postgres://gestor_user:dev-gestor-password@postgres:5432/gestor_info_db?sslmode=disable")
	t.Setenv("AUTH_JWKS_URL", "http://localhost:5101/api/v1/auth/jwks")
	t.Setenv("FRONTEND_URL", "http://localhost:5120")
	t.Setenv("TZ", "America/Santiago")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("APP_NAME", "MS Gestor de Información")
	t.Setenv("APP_ENV", "development")
}

func TestLoadValid(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("la configuración válida falló: %v", err)
	}
	if cfg.Port != "8080" || cfg.APIEnv != "dev" || cfg.TZ != "America/Santiago" {
		t.Errorf("valores inesperados: %+v", cfg)
	}
	if cfg.DatabaseURL != "postgres://gestor_user:dev-gestor-password@postgres:5432/gestor_info_db?sslmode=disable" {
		t.Errorf("DATABASE_URL inesperada: %q", cfg.DatabaseURL)
	}
	if cfg.AuthJWKSURL != "http://localhost:5101/api/v1/auth/jwks" {
		t.Errorf("AUTH_JWKS_URL inesperada: %q", cfg.AuthJWKSURL)
	}
	if cfg.FrontendURL != "http://localhost:5120" {
		t.Errorf("FRONTEND_URL inesperada: %q", cfg.FrontendURL)
	}
}

func TestLoadServiceInfoFromEnv(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SERVICE_ROLE", "Grupo 9 · Ejemplo")
	t.Setenv("SERVICE_CONSUMERS", " Reportes , , Plantillas ")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service.Role != "Grupo 9 · Ejemplo" {
		t.Errorf("rol inesperado: %q", cfg.Service.Role)
	}
	if got := strings.Join(cfg.Service.Consumers, "|"); got != "Reportes|Plantillas" {
		t.Errorf("la lista debe ignorar espacios y elementos vacíos, se obtuvo %q", got)
	}
}

func TestLoadRejectsMissingOrInvalidMandatoryVariables(t *testing.T) {
	testCases := []struct {
		name    string
		mutate  func(t *testing.T)
		wantMsg string
	}{
		{
			name: "puerto no es 8080",
			mutate: func(t *testing.T) {
				t.Setenv("API_PORT", "5121")
			},
			wantMsg: "API_PORT",
		},
		{
			name: "puerto vacío",
			mutate: func(t *testing.T) {
				t.Setenv("API_PORT", "")
				t.Setenv("BACKEND_PORT", "")
			},
			wantMsg: "API_PORT",
		},
		{
			name: "API_ENV inválido",
			mutate: func(t *testing.T) {
				t.Setenv("API_ENV", "staging")
			},
			wantMsg: "API_ENV",
		},
		{
			name: "DATABASE_URL vacía",
			mutate: func(t *testing.T) {
				t.Setenv("DATABASE_URL", "")
				t.Setenv("POSTGRES_URI", "")
			},
			wantMsg: "DATABASE_URL",
		},
		{
			name: "AUTH_JWKS_URL vacía",
			mutate: func(t *testing.T) {
				t.Setenv("AUTH_JWKS_URL", "")
			},
			wantMsg: "AUTH_JWKS_URL",
		},
		{
			name: "FRONTEND_URL vacía",
			mutate: func(t *testing.T) {
				t.Setenv("FRONTEND_URL", "")
			},
			wantMsg: "FRONTEND_URL",
		},
		{
			name: "TZ no es America/Santiago",
			mutate: func(t *testing.T) {
				t.Setenv("TZ", "UTC")
			},
			wantMsg: "TZ",
		},
		{
			name: "LOG_LEVEL inválido",
			mutate: func(t *testing.T) {
				t.Setenv("LOG_LEVEL", "verbose")
			},
			wantMsg: "LOG_LEVEL",
		},
		{
			name: "usuario de postgres sin contraseña",
			mutate: func(t *testing.T) {
				t.Setenv("POSTGRES_USER", "solo_usuario")
				t.Setenv("POSTGRES_PASSWORD", "")
			},
			wantMsg: "POSTGRES_USER",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			tc.mutate(t)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("se esperaba un error que mencione %s, se obtuvo: %v", tc.wantMsg, err)
			}
		})
	}
}
