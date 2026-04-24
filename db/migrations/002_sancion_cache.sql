-- Migration: Sancion Cache Table
-- Description: Tabla para cachear resultados de validación de sanciones

CREATE TABLE IF NOT EXISTS sancion_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rfc VARCHAR(30) NOT NULL UNIQUE,
    sancionado BOOLEAN NOT NULL DEFAULT FALSE,
    detalles JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sancion_cache_rfc ON sancion_cache(rfc);
CREATE INDEX IF NOT EXISTS idx_sancion_cache_expires ON sancion_cache(expires_at);

COMMENT ON TABLE sancion_cache IS 'Cache de validaciones de sanciones ante SCG y SFP';
COMMENT ON COLUMN sancion_cache.rfc IS 'RFC del proveedor validado';
COMMENT ON COLUMN sancion_cache.sancionado IS 'Indica si el proveedor está sancionado';
COMMENT ON COLUMN sancion_cache.detalles IS 'Detalles de las sanciones encontradas en formato JSON';
COMMENT ON COLUMN sancion_cache.expires_at IS 'Fecha de expiración del cache';
