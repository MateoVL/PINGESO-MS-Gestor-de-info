-- +goose Up
CREATE TABLE IF NOT EXISTS lotes (
    id UUID PRIMARY KEY,
    origen VARCHAR(254) NOT NULL DEFAULT '',
    referencia_externa VARCHAR(254) NOT NULL DEFAULT '',
    recibidos INT NOT NULL,
    aceptados INT NOT NULL,
    rechazados INT NOT NULL,
    clave_idempotencia VARCHAR(128) UNIQUE,
    huella_contenido VARCHAR(64) NOT NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS registros (
    id UUID PRIMARY KEY,
    lote_id UUID NOT NULL REFERENCES lotes(id) ON DELETE CASCADE,
    posicion INT NOT NULL,
    folio VARCHAR(254) NOT NULL UNIQUE,
    nombre VARCHAR(254) NOT NULL,
    correo VARCHAR(254) NOT NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_registros_lote_id ON registros(lote_id);

CREATE TABLE IF NOT EXISTS rechazos (
    id BIGSERIAL PRIMARY KEY,
    lote_id UUID NOT NULL REFERENCES lotes(id) ON DELETE CASCADE,
    posicion INT NOT NULL,
    folio VARCHAR(254) NOT NULL DEFAULT '',
    motivos TEXT[] NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rechazos_lote_id ON rechazos(lote_id);

-- +goose Down
DROP TABLE IF EXISTS rechazos;
DROP TABLE IF EXISTS registros;
DROP TABLE IF EXISTS lotes;

