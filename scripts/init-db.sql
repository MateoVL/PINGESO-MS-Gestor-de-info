-- ==============================================================================
-- MS Gestor de Información - Initial Database Schema
-- ==============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabla de verificación y metadatos del servicio
CREATE TABLE IF NOT EXISTS service_health (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    initialized_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'INITIALIZED'
);

-- Inserción inicial de verificación
INSERT INTO service_health (service_name, status)
VALUES ('MS Gestor de Información', 'READY')
ON CONFLICT DO NOTHING;

-- Tabla base para la futura ingesta de información (desde Digitalización)
CREATE TABLE IF NOT EXISTS raw_data_ingest (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_service VARCHAR(100) NOT NULL DEFAULT 'MS Digitalización',
    batch_reference VARCHAR(100),
    payload JSONB,
    status VARCHAR(50) DEFAULT 'PENDING', -- PENDING, PROCESSING, ORDERED, ERROR
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla base para el ordenamiento y entrega a MS Reportes
CREATE TABLE IF NOT EXISTS processed_info (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ingest_id UUID REFERENCES raw_data_ingest(id) ON DELETE SET NULL,
    category VARCHAR(100),
    data_order INT DEFAULT 0,
    normalized_content JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE raw_data_ingest IS 'Almacén preliminar para datos recibidos del microservicio de Digitalización';
COMMENT ON TABLE processed_info IS 'Información estructurada, procesada y ordenada lista para ser consumida por MS Reportes';
