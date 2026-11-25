package api

import (
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/service"
	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/store"
)

type RegistroEntrada struct {
	Folio  string `json:"folio" maxLength:"254" doc:"Folio identificador del documento" example:"DOC-0001"`
	Nombre string `json:"nombre" maxLength:"254" doc:"Nombre completo del titular" example:"Camila Soto"`
	Correo string `json:"correo" maxLength:"254" doc:"Correo electrónico de contacto" example:"camila@example.com"`
}

type SolicitudLote struct {
	Origen            string            `json:"origen" maxLength:"254" doc:"Sistema emisor del lote" example:"Extractor OCR"`
	ReferenciaExterna string            `json:"referencia_externa" maxLength:"254" doc:"Identificador de correlación en el emisor" example:"lote-2026-001"`
	Registros         []RegistroEntrada `json:"registros" nullable:"false" minItems:"1" maxItems:"5000" doc:"Lista de registros a procesar (entre 1 y 5000)"`
}

type LivenessStatus struct {
	Status string `json:"status" doc:"Estado del proceso HTTP" example:"ok"`
}

type LivenessOutput struct {
	Status int            `header:"-"`
	Body   LivenessStatus `json:"body"`
}

type HealthOutput struct {
	Status int                    `header:"-"`
	Body   service.HealthResponse `json:"body"`
}

type CreateBatchInput struct {
	IdempotencyKey string        `header:"Idempotency-Key" doc:"Clave de idempotencia para evitar duplicados en reintentos" maxLength:"128" example:"lote-2026-001"`
	Body           SolicitudLote `json:"body"`
}

type CreateBatchOutput struct {
	Status   int               `header:"-"`
	Location string            `header:"Location" doc:"URI del lote creado"`
	Body     store.LoteDetalle `json:"body"`
}

type ListBatchesInput struct {
	Pagina       int `query:"pagina" default:"1" minimum:"1" doc:"Número de página (1-indexed)" example:"1"`
	TamanoPagina int `query:"tamano_pagina" default:"50" minimum:"1" maximum:"200" doc:"Cantidad de lotes por página" example:"50"`
}

type ListBatchesOutput struct {
	Body store.ListaLotes `json:"body"`
}

type GetByIDInput struct {
	ID string `path:"id" format:"uuid" doc:"UUID del recurso" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type GetBatchOutput struct {
	Body store.LoteDetalle `json:"body"`
}

type ListRejectionsOutput struct {
	Body store.ListaRechazos `json:"body"`
}

type ListRecordsInput struct {
	Pagina       int    `query:"pagina" default:"1" minimum:"1" doc:"Número de página (1-indexed)" example:"1"`
	TamanoPagina int    `query:"tamano_pagina" default:"50" minimum:"1" maximum:"200" doc:"Cantidad de registros por página" example:"50"`
	Folio        string `query:"folio" doc:"Filtro opcional por folio exacto (se normaliza a mayúsculas)" example:"DOC-0001"`
	LoteID       string `query:"lote_id" format:"uuid" doc:"Filtro opcional por UUID de lote" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type ListRecordsOutput struct {
	Body store.ListaRegistros `json:"body"`
}

type GetRecordOutput struct {
	Body store.Registro `json:"body"`
}

type DirtyMocksInput struct {
	Cantidad int    `query:"cantidad" default:"20" minimum:"1" maximum:"5000" doc:"Cantidad de registros simulados a generar" example:"20"`
	Formato  string `query:"formato" default:"json" enum:"json,csv" doc:"Formato de salida (json o csv)" example:"json"`
	Semilla  int    `query:"semilla" doc:"Semilla opcional para generación determinista" example:"7"`
}

type DirtyMocksOutput struct {
	Body SolicitudLote `json:"body"`
}

func toStoreEntries(in []RegistroEntrada) []store.RegistroEntrada {
	out := make([]store.RegistroEntrada, len(in))
	for i, r := range in {
		out[i] = store.RegistroEntrada{
			Folio:  r.Folio,
			Nombre: r.Nombre,
			Correo: r.Correo,
		}
	}
	return out
}

func fromStoreEntries(in []store.RegistroEntrada) []RegistroEntrada {
	out := make([]RegistroEntrada, len(in))
	for i, r := range in {
		out[i] = RegistroEntrada{
			Folio:  r.Folio,
			Nombre: r.Nombre,
			Correo: r.Correo,
		}
	}
	return out
}
