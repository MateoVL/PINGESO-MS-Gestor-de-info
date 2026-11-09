package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// limites de un lote y de la paginacion
const (
	MaxRegistrosPorLote    = 5000
	MaxLargoCampo          = 254
	TamanoPaginaPorDefecto = 50
	TamanoPaginaMaximo     = 200
)

// registro tal como llega, sin limpiar
type RegistroEntrada struct {
	Folio  string `json:"folio"`
	Nombre string `json:"nombre"`
	Correo string `json:"correo"`
}

// json de lotes
type SolicitudLote struct {
	Origen            string            `json:"origen"`
	ReferenciaExterna string            `json:"referencia_externa"`
	Registros         []RegistroEntrada `json:"registros"`
}

// resume un lote procesado
type Lote struct {
	ID                string    `json:"id" bson:"_id"`
	Origen            string    `json:"origen" bson:"origen"`
	ReferenciaExterna string    `json:"referencia_externa" bson:"referencia_externa"`
	Recibidos         int       `json:"recibidos" bson:"recibidos"`
	Aceptados         int       `json:"aceptados" bson:"aceptados"`
	Rechazados        int       `json:"rechazados" bson:"rechazados"`
	CreadoEn          time.Time `json:"creado_en" bson:"creado_en"`

	// permiten reconocer un reintento del emisor
	ClaveIdempotencia string `json:"-" bson:"clave_idempotencia,omitempty"`
	HuellaContenido   string `json:"-" bson:"huella_contenido"`
}

// registro limpio y validado
type Registro struct {
	ID       string    `json:"id" bson:"_id"`
	LoteID   string    `json:"lote_id" bson:"lote_id"`
	Posicion int       `json:"posicion" bson:"posicion"`
	Folio    string    `json:"folio" bson:"folio"`
	Nombre   string    `json:"nombre" bson:"nombre"`
	Correo   string    `json:"correo" bson:"correo"`
	CreadoEn time.Time `json:"creado_en" bson:"creado_en"`
}

// explica por que no se acepto un registro del lote
type Rechazo struct {
	LoteID   string   `json:"-" bson:"lote_id"`
	Posicion int      `json:"posicion" bson:"posicion"`
	Folio    string   `json:"folio" bson:"folio"`
	Motivos  []string `json:"motivos" bson:"motivos"`
}

// lote con sus rechazos
type LoteDetalle struct {
	Lote
	Rechazos []Rechazo `json:"rechazos"`
}

// describe la pagina devuelta en un listado
type InfoPagina struct {
	Total        int `json:"total"`
	Pagina       int `json:"pagina"`
	TamanoPagina int `json:"tamano_pagina"`
}

type ListaLotes struct {
	InfoPagina
	Elementos []Lote `json:"elementos"`
}

type ListaRegistros struct {
	InfoPagina
	Elementos []Registro `json:"elementos"`
}

type ListaRechazos struct {
	Elementos []Rechazo `json:"elementos"`
}

// indica que pagina pedir (parte en 1)
type Paginacion struct {
	Numero int
	Tamano int
}

// devuelve cuantos elementos saltar
func (p Paginacion) Desplazamiento() int {
	return (p.Numero - 1) * p.Tamano
}

// acota la consulta de registros
type FiltroRegistros struct {
	Folio      string
	LoteID     string
	Paginacion Paginacion
}

// todo lo que el repositorio guarda en una sola operacion
type NuevoLote struct {
	Lote      Lote
	Registros []Registro
	Rechazos  []Rechazo
}

// errores
var (
	ErrNoEncontrado               = errors.New("recurso no encontrado")
	ErrClaveIdempotenciaEnUso     = errors.New("la Idempotency-Key ya está en uso")
	ErrClaveIdempotenciaConflicto = errors.New("la Idempotency-Key ya se usó con otro contenido")
	ErrDemasiadosConflictos       = errors.New("demasiados conflictos de folio concurrentes")
)

// indica que folios ya estaban registrados en otro lote
type ErrFoliosDuplicados struct {
	Folios []string
}

func (e *ErrFoliosDuplicados) Error() string {
	return fmt.Sprintf("folios ya registrados: %s", strings.Join(e.Folios, ", "))
}
