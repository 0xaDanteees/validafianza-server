--Extensions
CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;


--providers
CREATE TABLE providers (
    id UUID PRIMARY KEY,
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('poliza', 'proveedor', 'seguros', 'otro')),
    provider_key VARCHAR(255) NOT NULL,
    config JSONB DEFAULT '{}'::jsonb,
    metadata hstore,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_providers_type_active ON providers(provider_type, is_active);
CREATE INDEX idx_providers_key ON providers(provider_key) WHERE is_active = true;


-- unlogged cache 
CREATE UNLOGGED TABLE validation_cache (
    lookup_hash VARCHAR(64) PRIMARY KEY,
    provider_id UUID REFERENCES providers(id),
    lookup_key VARCHAR(255) NOT NULL,
    result JSONB NOT NULL,
    access_count INTEGER DEFAULT 1,
    scraped_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    metadata hstore
) WITH (fillfactor = 70);

CREATE INDEX idx_cache_expires ON validation_cache(expires_at) WHERE expires_at > NOW();
CREATE INDEX idx_cache_provider ON validation_cache(provider_id, scraped_at DESC);

--Scraping jobs queue
CREATE TABLE scraping_jobs (
    id UUID PRIMARY KEY,
    provider_id UUID REFERENCES providers(id),
    lookup_key VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    priority INTEGER DEFAULT 0,
    result JSONB,
    error_message TEXT,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 5,
    locked_until TIMESTAMPTZ,
    locked_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_jobs_status_priority ON scraping_jobs(status, priority DESC, created_at ASC) WHERE status = 'pending';
CREATE INDEX idx_jobs_locked ON scraping_jobs(locked_until) WHERE locked_until > NOW() AND status = 'processing';
CREATE INDEX idx_jobs_provider ON scraping_jobs(provider_id);

-- rate limiting 
CREATE UNLOGGED TABLE rate_limits (
    key VARCHAR(255) PRIMARY KEY,
    requests INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT NOW(),
    metadata hstore
);

CREATE INDEX idx_rate_limits_window ON rate_limits(window_start);

-- sessions
CREATE UNLOGGED TABLE sessions (
    id UUID PRIMARY KEY,
    -- user_id UUID NOT NULL,
    data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
);

CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE expires_at > NOW();

--dequeue func with SKIP LOCKED approach
CREATE OR REPLACE FUNCTION 
dequeue_scraping_job(worker_id_param VARCHAR(255))
RETURNS TABLE (
    job_id UUID,
    provider_id UUID,
    provider_key VARCHAR(255),
    lookup_key VARCHAR(255),
    config JSONB
) AS $$
DECLARE
    job_record RECORD;
BEGIN
    SELECT sj.id, sj.provider_id, p.provider_key, sj.lookup_key, p.config
    INTO job_record
    FROM scraping_jobs sj
    JOIN providers p ON sj.provider_id = p.id
    WHERE sj.status = 'pending'
    ORDER BY sj.priority DESC, sj.created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1;

    IF job_record IS NOT NULL THEN
        UPDATE scraping_jobs
        SET status = 'processing',
            started_at = NOW(),
            locked_until = NOW() + INTERVAL '3 minutes',
            locked_by = worker_id_param
        WHERE id = job_record.id;
        
        RETURN QUERY SELECT 
            job_record.id, 
            job_record.provider_id, 
            job_record.provider_key, 
            job_record.lookup_key, 
            job_record.config;
    END IF;
    
END;
$$ LANGUAGE plpgsql;

-- views for analytics as we might need this later for internal ETLs
CREATE MATERIALIZED VIEW provider_stats AS
SELECT
    p.provider_key,
    p.provider_type,
    COUNT(vc.lookup_hash) as total_cached,
    AVG(vc.access_count) as avg_access_count,
    MAX(vc.scraped_at) as last_scraped_at
FROM providers p
LEFT JOIN validation_cache vc ON p.id = vc.provider_id
GROUP BY p.id, p.provider_key, p.provider_type;

CREATE UNIQUE INDEX idx_provider_stats_key ON provider_stats(provider_key);



