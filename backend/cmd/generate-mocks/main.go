package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/mocks"
)

func main() {
	count := flag.Int("cantidad", 20, "número de registros")
	format := flag.String("formato", "json", "json o csv")
	seed := flag.Uint64("semilla", uint64(time.Now().UnixNano()), "semilla para repetir los mismos datos")
	flag.Parse()

	if *count < 1 || *count > models.MaxRegistrosPorLote {
		fail(fmt.Errorf("-cantidad debe estar entre 1 y %d", models.MaxRegistrosPorLote))
	}
	records := mocks.GenerateDirty(*count, *seed)

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fail(enc.Encode(models.SolicitudLote{
			Origen:            "Extractor OCR",
			ReferenciaExterna: fmt.Sprintf("mock-%d", *seed),
			Registros:         records,
		}))
	case "csv":
		data, err := mocks.ToCSV(records)
		fail(err)
		_, err = os.Stdout.Write(data)
		fail(err)
	default:
		fail(fmt.Errorf("-formato debe ser json o csv"))
	}
}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
