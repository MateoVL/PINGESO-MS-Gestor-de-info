package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
	"github.com/mateo/ms-gestor-de-info/backend/mocks"
)

// lote de datos sucios
func DirtyData(c *gin.Context) {
	errs := map[string]string{}
	count, ok := intQuery(c, "cantidad", 20, 1, models.MaxRegistrosPorLote)
	if !ok {
		errs["cantidad"] = "debe ser un entero entre 1 y 5000"
	}
	format := c.DefaultQuery("formato", "json")
	if format != "json" && format != "csv" {
		errs["formato"] = "debe ser json o csv"
	}
	seed := uint64(time.Now().UnixNano())
	if raw, present := c.GetQuery("semilla"); present {
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			errs["semilla"] = "debe ser un entero mayor o igual a 0"
		}
		seed = n
	}
	if len(errs) > 0 {
		utils.RespondError(c, utils.ErrInvalidData.WithDetails(errs))
		return
	}

	records := mocks.GenerateDirty(count, seed)
	if format == "csv" {
		data, err := mocks.ToCSV(records)
		if err != nil {
			utils.RespondError(c, err)
			return
		}
		c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
		return
	}
	c.JSON(http.StatusOK, models.SolicitudLote{
		Origen:            "Extractor OCR",
		ReferenciaExterna: "mock-" + strconv.FormatUint(seed, 10),
		Registros:         records,
	})
}
