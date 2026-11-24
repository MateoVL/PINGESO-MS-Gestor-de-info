-- name: CreateLote :one
INSERT INTO lotes (
    id, origen, referencia_externa, recibidos, aceptados, rechazados, clave_idempotencia, huella_contenido, creado_en
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetLoteByID :one
SELECT * FROM lotes
WHERE id = $1 LIMIT 1;

-- name: GetLoteByIdempotencyKey :one
SELECT * FROM lotes
WHERE clave_idempotencia = $1 LIMIT 1;

-- name: ListLotes :many
SELECT * FROM lotes
ORDER BY creado_en DESC
LIMIT $1 OFFSET $2;

-- name: CountLotes :one
SELECT COUNT(*) FROM lotes;

-- name: FindExistingFolios :many
SELECT folio FROM registros
WHERE folio = ANY($1::varchar[]);

-- name: CreateRegistro :one
INSERT INTO registros (
    id, lote_id, posicion, folio, nombre, correo, creado_en
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetRegistroByID :one
SELECT * FROM registros
WHERE id = $1 LIMIT 1;

-- name: CountRegistros :one
SELECT COUNT(*) FROM registros
WHERE (sqlc.narg('folio')::varchar IS NULL OR folio = sqlc.narg('folio'))
  AND (sqlc.narg('lote_id')::uuid IS NULL OR lote_id = sqlc.narg('lote_id'));

-- name: ListRegistros :many
SELECT * FROM registros
WHERE (sqlc.narg('folio')::varchar IS NULL OR folio = sqlc.narg('folio'))
  AND (sqlc.narg('lote_id')::uuid IS NULL OR lote_id = sqlc.narg('lote_id'))
ORDER BY creado_en ASC, posicion ASC
LIMIT $1 OFFSET $2;

-- name: CreateRechazo :one
INSERT INTO rechazos (
    lote_id, posicion, folio, motivos
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: ListRechazosByLoteID :many
SELECT * FROM rechazos
WHERE lote_id = $1
ORDER BY posicion ASC;

