package mocks

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
)

func TestGenerateDirtyIsDeterministic(t *testing.T) {
	a, b := GenerateDirty(50, 9), GenerateDirty(50, 9)
	if !reflect.DeepEqual(a, b) {
		t.Error("con la misma semilla se deben generar los mismos datos")
	}
	if len(a) != 50 {
		t.Errorf("se esperaban 50 registros, hay %d", len(a))
	}
}

func TestStaticFilesAreValid(t *testing.T) {
	var req models.SolicitudLote
	if err := json.Unmarshal(DirtyBatchJSON, &req); err != nil || len(req.Registros) == 0 {
		t.Fatalf("lote-sucio.json debe ser un lote válido: %v", err)
	}
	csv, err := ToCSV(req.Registros)
	if err != nil {
		t.Fatal(err)
	}
	if string(csv) != string(DirtyBatchCSV) {
		t.Error("lote-sucio.csv debe tener los mismos registros que lote-sucio.json; regenera ambos con la misma semilla")
	}
}
