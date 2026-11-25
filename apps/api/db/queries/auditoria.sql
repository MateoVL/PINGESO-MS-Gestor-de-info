-- name: CreateEventoAuditoria :exec
INSERT INTO auditoria (
    ocurrido_en, actor, accion, entidad, entidad_id, request_id, detalle
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);
