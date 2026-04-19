-- Seed providers

-- Eventually I'll need to handle polizas, proveedores, seguros, and other types of validations separately
-- For now is tech debt and is fine
INSERT INTO providers (provider_type, provider_key, scraper_type, config, metadata) VALUES
-- Polizas
('poliza', 'MAPFRE', 'native', 
 '{"url": "https://validacionfianzas.mapfre.com.mx/ValidaMapfre/Sitios/Default.aspx"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'ASERTA_FIANZAS_SEGUROS', 'native',
 '{"url": "https://www.aserta.mx/validacion-polizas"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'INSURGENTES_AFIANZADORA', 'native',
 '{"url": "https://www.insurgentes.com.mx/validacion"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'BERKLEY_MEXICO_FIANZAS', 'native',
 '{"url": "https://validafianza.berkleymex.com"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'DORAMA', 'native',
 '{"url": "https://www.dorama.com.mx/validacion"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'FIANZAS_Y_CAUCIONES_ATLAS', 'native',
 '{"url": "https://www.atlas.com.mx/validacion"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'SOFIMEX', 'native',
 '{"url": "https://www.sofimex.com.mx/validacion"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('poliza', 'TOKIO_MARINE', 'native',
 '{"url": "https://www.tokiomarine.com.mx/validacion"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

-- Proveedores
('proveedor', 'SFP', 'native',
 '{"url": "https://consultapublicamx.inai.org.mx/vut-web"}'::jsonb,
 'timeout => 30, retry => 3'::hstore),

('proveedor', 'SCG', 'native',
 '{"url": "https://www.gob.mx/sfp"}'::jsonb,
 'timeout => 30, retry => 3'::hstore);