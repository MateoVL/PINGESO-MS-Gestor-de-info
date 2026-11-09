package handlers

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/models"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// lee el parametro :id, si no es un UUID responde 400 y devuelve ok=false
func uuidParam(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !isUUID(id) {
		utils.RespondError(c, utils.ErrInvalidData.WithDetails(map[string]string{"id": "debe ser un UUID"}))
		return "", false
	}
	return id, true
}

// lee pagina y tamaño con sus valores por defecto y limites
func bindPagination(c *gin.Context) (models.Paginacion, error) {
	errs := map[string]string{}
	number, ok := intQuery(c, "pagina", 1, 1, 1<<20)
	if !ok {
		errs["pagina"] = "debe ser un entero mayor o igual a 1"
	}
	size, ok := intQuery(c, "tamano_pagina", models.TamanoPaginaPorDefecto, 1, models.TamanoPaginaMaximo)
	if !ok {
		errs["tamano_pagina"] = fmt.Sprintf("debe ser un entero entre 1 y %d", models.TamanoPaginaMaximo)
	}
	if len(errs) > 0 {
		return models.Paginacion{}, utils.ErrInvalidData.WithDetails(errs)
	}
	return models.Paginacion{Numero: number, Tamano: size}, nil
}

// lee un entero opcional
func intQuery(c *gin.Context, name string, def, lo, hi int) (int, bool) {
	raw, present := c.GetQuery(name)
	if !present {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return 0, false
	}
	return n, true
}
