package service

import (
	"context"

	"github.com/PINGESO-CIVIL/gestor-de-info/apps/api/internal/store"
)

// registro no trae un JWT
const actorAnonimo = "anonimo"

const ActorContextKey ctxKey = "actor"

// guarda en el contexto quien hace la operacion
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, ActorContextKey, actor)
}

// devuelve quien hace la operacion o anonimo si no hay JWT
func ActorFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ActorContextKey).(string); ok && v != "" {
		return v
	}
	return actorAnonimo
}

// arma el registro de auditoria de un lote nuevo
func eventoCreacionLote(ctx context.Context, lote store.Lote) store.EventoAuditoria {
	return store.EventoAuditoria{
		OcurridoEn: lote.CreadoEn,
		Actor:      ActorFromContext(ctx),
		Accion:     store.AccionCrearLote,
		Entidad:    store.EntidadLote,
		EntidadID:  lote.ID,
		RequestID:  RequestIDFromContext(ctx),
		Detalle: map[string]any{
			"origen":     lote.Origen,
			"recibidos":  lote.Recibidos,
			"aceptados":  lote.Aceptados,
			"rechazados": lote.Rechazados,
		},
	}
}
