package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"APP_ENV", "GIN_MODE", "LOG_LEVEL", "BACKEND_PORT", "MONGO_USER", "MONGO_PASSWORD", "SERVICE_ROLE", "SERVICE_DEPENDS_ON", "SERVICE_CONSUMERS", "SERVICE_CAPABILITIES"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("la configuración por defecto debería ser válida: %v", err)
	}
	if cfg.Port != "8080" || cfg.AppEnv != "development" || cfg.LogLevel != "info" {
		t.Errorf("valores por defecto inesperados: %+v", cfg)
	}
	if cfg.Service.Name != cfg.AppName || cfg.Service.Environment != cfg.AppEnv {
		t.Errorf("la identidad del servicio debe reflejar APP_NAME y APP_ENV: %+v", cfg.Service)
	}
	// sin .env la identidad queda vacia, pero las listas deben ser [] y no null en el JSON !!
	s := cfg.Service
	if s.Role != "" || s.DependsOn == nil || s.Consumers == nil || s.Capabilities == nil ||
		len(s.DependsOn)+len(s.Consumers)+len(s.Capabilities) != 0 {
		t.Errorf("la identidad por defecto debe estar vacía: %+v", s)
	}
}

func TestLoadServiceInfoFromEnv(t *testing.T) {
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

func TestLoadRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name, key, value, wantMsg string
	}{
		{"puerto no numérico", "BACKEND_PORT", "abc", "BACKEND_PORT"},
		{"puerto fuera de rango", "BACKEND_PORT", "70000", "BACKEND_PORT"},
		{"entorno desconocido", "APP_ENV", "staging", "APP_ENV"},
		{"modo de Gin desconocido", "GIN_MODE", "verbose", "GIN_MODE"},
		{"nivel de log desconocido", "LOG_LEVEL", "trace", "LOG_LEVEL"},
		{"usuario sin contraseña", "MONGO_USER", "solo_usuario", "MONGO_PASSWORD"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MONGO_PASSWORD", "")
			t.Setenv(tc.key, tc.value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("se esperaba un error que mencione %s, se obtuvo: %v", tc.wantMsg, err)
			}
		})
	}
}
