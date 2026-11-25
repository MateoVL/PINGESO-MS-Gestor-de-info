-- +goose Up
CREATE TABLE IF NOT EXISTS auditoria (
    id BIGSERIAL PRIMARY KEY,
    ocurrido_en TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor VARCHAR(254) NOT NULL,
    accion VARCHAR(64) NOT NULL,
    entidad VARCHAR(64) NOT NULL,
    entidad_id UUID NOT NULL,
    request_id VARCHAR(64) NOT NULL DEFAULT '',
    detalle JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_auditoria_entidad ON auditoria(entidad, entidad_id);

-- +goose Down
DROP TABLE IF EXISTS auditoria;
