package api

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/store"
)

//go:embed lote-sucio.json
var DirtyBatchJSON []byte

//go:embed lote-sucio.csv
var DirtyBatchCSV []byte

var (
	names   = []string{"Camila Soto", "Diego Pérez", "Elena Díaz", "Matías Rojas", "Valentina Muñoz", "Benjamín Silva", "Javiera Torres", "Tomás Fuentes", "Isidora Vargas", "Joaquín Castro"}
	domains = []string{"example.com", "usach.cl", "correo.cl"}
)

func GenerateDirty(n int, seed uint64) []store.RegistroEntrada {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	records := make([]store.RegistroEntrada, 0, n)

	for i := range n {
		name := names[rng.IntN(len(names))]
		user := strings.ToLower(strings.ReplaceAll(strings.Fields(name)[0], "í", "i"))
		r := store.RegistroEntrada{
			Folio:  fmt.Sprintf("doc-%04d", i+1),
			Nombre: name,
			Correo: fmt.Sprintf("%s%d@%s", user, i+1, domains[rng.IntN(len(domains))]),
		}

		if rng.IntN(3) == 0 {
			r.Folio = "  " + strings.ToUpper(r.Folio) + " "
		}
		if rng.IntN(3) == 0 {
			r.Nombre = " " + strings.ReplaceAll(r.Nombre, " ", "   ") + "  "
		}
		if rng.IntN(3) == 0 {
			r.Correo = " " + strings.ToUpper(r.Correo)
		}

		switch rng.IntN(10) {
		case 0:
			r.Correo = strings.Replace(r.Correo, "@", " at ", 1)
		case 1:
			r.Nombre = "   "
		case 2:
			r.Folio = ""
		case 3:
			if len(records) > 0 {
				r.Folio = records[rng.IntN(len(records))].Folio
			}
		}
		records = append(records, r)
	}
	return records
}

func ToCSV(records []store.RegistroEntrada) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"folio", "nombre", "correo"}); err != nil {
		return nil, err
	}
	for _, r := range records {
		if err := w.Write([]string{r.Folio, r.Nombre, r.Correo}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
