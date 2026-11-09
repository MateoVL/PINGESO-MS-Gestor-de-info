package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func respond(err error) (*httptest.ResponseRecorder, *gin.Context) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(RequestIDKey, "req-123")
	RespondError(c, err)
	return rec, c
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) models.ErrorBody {
	t.Helper()
	var body models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta de error inválida: %v", err)
	}
	return body.Error
}

func TestRespondErrorKnownError(t *testing.T) {
	rec, c := respond(fmt.Errorf("buscando registro 42: %w", ErrNotFound))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("código HTTP %d, se esperaba 404", rec.Code)
	}
	body := decode(t, rec)
	if body.Code != "not_found" || body.RequestID != "req-123" {
		t.Errorf("cuerpo inesperado: %+v", body)
	}
	if strings.Contains(rec.Body.String(), "registro 42") {
		t.Error("el contexto interno del error no debe llegar al cliente")
	}
	if len(c.Errors) != 0 {
		t.Error("los errores 4xx no deben registrarse como fallas internas")
	}
}

func TestRespondErrorUnknownErrorIsHidden(t *testing.T) {
	rec, c := respond(errors.New("conexión rechazada por mongo:27017"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("código HTTP %d, se esperaba 500", rec.Code)
	}
	if body := decode(t, rec); body.Code != "internal_error" {
		t.Errorf("código inesperado: %q", body.Code)
	}
	if strings.Contains(rec.Body.String(), "mongo") {
		t.Error("el error interno se filtró al cliente")
	}
	if len(c.Errors) != 1 {
		t.Error("la causa real debe quedar en c.Errors para el logger")
	}
}

func TestWithDetailsKeepsIdentity(t *testing.T) {
	details := map[string]string{"correo": "formato inválido"}
	err := ErrInvalidData.WithDetails(details)

	if !errors.Is(err, ErrInvalidData) {
		t.Error("errors.Is debe reconocer la copia con detalles")
	}
	if ErrInvalidData.Details != nil {
		t.Error("WithDetails no debe modificar el error original")
	}

	rec, _ := respond(err)
	if !strings.Contains(rec.Body.String(), "formato inválido") {
		t.Errorf("los detalles deben llegar al cliente: %s", rec.Body.String())
	}
}
