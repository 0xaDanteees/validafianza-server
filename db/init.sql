--Extensions
CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;


--providers
CREATE TABLE providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('poliza', 'proveedor', 'seguros', 'otro')),
    provider_key VARCHAR(255) NOT NULL,
    config JSONB DEFAULT '{}'::jsonb,
    -- native: direct scraping, playwright: browser automation
    scraper_type VARCHAR(50) NOT NULL CHECK (scraper_type IN ('native', 'playwright')),
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

CREATE INDEX idx_cache_expires ON validation_cache(expires_at);
CREATE INDEX idx_cache_provider ON validation_cache(provider_id, scraped_at DESC);

--Scraping jobs queue
CREATE TABLE scraping_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
CREATE INDEX idx_jobs_locked ON scraping_jobs(locked_until) WHERE status = 'processing';
CREATE INDEX idx_jobs_provider ON scraping_jobs(provider_id);

-- rate limiting (sliding window: stores per-request timestamps as JSONB array)
CREATE UNLOGGED TABLE rate_limits (
    key VARCHAR(255) PRIMARY KEY,
    requests JSONB DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    metadata hstore
) WITH (fillfactor = 80);

CREATE INDEX idx_rate_limits_updated ON rate_limits(updated_at);

-- Returns TRUE if the request is within the limit, FALSE if rate-limited.
-- Atomically prunes stale timestamps and appends the current one.
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_key            VARCHAR(255),
    p_limit          INTEGER,
    p_window_seconds INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    cutoff       TIMESTAMPTZ := NOW() - (p_window_seconds || ' seconds')::INTERVAL;
    new_requests JSONB;
BEGIN
    INSERT INTO rate_limits (key, requests, updated_at)
    VALUES (p_key, jsonb_build_array(NOW()::text), NOW())
    ON CONFLICT (key) DO UPDATE SET
        requests   = COALESCE(
                         (SELECT jsonb_agg(ts ORDER BY ts)
                          FROM   jsonb_array_elements_text(rate_limits.requests) AS ts
                          WHERE  ts::TIMESTAMPTZ > cutoff),
                         '[]'::jsonb
                     ) || jsonb_build_array(NOW()::text),
        updated_at = NOW()
    RETURNING requests INTO new_requests;

    RETURN jsonb_array_length(new_requests) <= p_limit;
END;
$$ LANGUAGE plpgsql;

-- users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    full_name VARCHAR(255),
    company_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    max_devices INTEGER DEFAULT 3,
    metadata hstore,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email) WHERE is_active = true;
CREATE INDEX idx_users_phone ON users(phone) WHERE is_active = true AND phone IS NOT NULL;

-- passcodes (OTP for login)
CREATE TABLE passcodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(8) NOT NULL,
    delivery_method VARCHAR(10) NOT NULL CHECK (delivery_method IN ('email', 'sms')),
    delivery_target VARCHAR(255) NOT NULL,
    is_used BOOLEAN DEFAULT false,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_passcodes_user ON passcodes(user_id, created_at DESC);
CREATE INDEX idx_passcodes_code ON passcodes(code, is_used);
CREATE INDEX idx_passcodes_expires ON passcodes(expires_at);

-- device fingerprints (detect account sharing)
CREATE TABLE device_fingerprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    fingerprint_hash VARCHAR(64) UNIQUE NOT NULL,
    device_name VARCHAR(255),
    user_agent TEXT,
    ip_address INET,
    metadata JSONB DEFAULT '{}'::jsonb,
    is_trusted BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_fingerprints_user ON device_fingerprints(user_id, last_seen_at DESC);
CREATE INDEX idx_fingerprints_hash ON device_fingerprints(fingerprint_hash);

-- sessions (linked to user and device)
CREATE UNLOGGED TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_fingerprint_id UUID REFERENCES device_fingerprints(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    data JSONB DEFAULT '{}'::jsonb,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id, last_activity_at DESC);
CREATE INDEX idx_sessions_token ON sessions(token_hash);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_sessions_device ON sessions(device_fingerprint_id);

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

-- Function: validate and extend session
CREATE OR REPLACE FUNCTION validate_session(p_token_hash VARCHAR(64))
RETURNS TABLE (
    session_id UUID,
    user_id UUID,
    user_email VARCHAR(255),
    device_fingerprint_id UUID,
    is_valid BOOLEAN
) AS $$
DECLARE
    session_record RECORD;
BEGIN
    SELECT s.id, s.user_id, u.email, s.device_fingerprint_id, s.expires_at
    INTO session_record
    FROM sessions s
    JOIN users u ON s.user_id = u.id
    WHERE s.token_hash = p_token_hash
      AND s.expires_at > NOW()
      AND u.is_active = true
    FOR UPDATE;
    
    IF session_record.id IS NOT NULL THEN
        UPDATE sessions
        SET last_activity_at = NOW(),
            expires_at = NOW() + INTERVAL '7 days'
        WHERE id = session_record.id;
        
        UPDATE device_fingerprints
        SET last_seen_at = NOW()
        WHERE id = session_record.device_fingerprint_id;
        
        RETURN QUERY SELECT 
            session_record.id,
            session_record.user_id,
            session_record.email,
            session_record.device_fingerprint_id,
            true;
    ELSE
        RETURN QUERY SELECT NULL::UUID, NULL::UUID, NULL::VARCHAR, NULL::UUID, false;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function: cleanup expired data
CREATE OR REPLACE FUNCTION cleanup_expired_data()
RETURNS TABLE (
    deleted_sessions INTEGER,
    deleted_passcodes INTEGER
) AS $$
DECLARE
    sessions_count INTEGER;
    passcodes_count INTEGER;
BEGIN
    DELETE FROM sessions WHERE expires_at < NOW() - INTERVAL '1 day';
    GET DIAGNOSTICS sessions_count = ROW_COUNT;
    
    DELETE FROM passcodes WHERE expires_at < NOW() - INTERVAL '1 day';
    GET DIAGNOSTICS passcodes_count = ROW_COUNT;
    
    RETURN QUERY SELECT sessions_count, passcodes_count;
END;
$$ LANGUAGE plpgsql;

-- Function: check device limit for user
CREATE OR REPLACE FUNCTION check_device_limit(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    device_count INTEGER;
    max_allowed INTEGER;
BEGIN
    SELECT max_devices INTO max_allowed FROM users WHERE id = p_user_id;
    
    SELECT COUNT(DISTINCT device_fingerprint_id)
    INTO device_count
    FROM sessions
    WHERE user_id = p_user_id
      AND expires_at > NOW();
    
    RETURN device_count < max_allowed;
END;
$$ LANGUAGE plpgsql;
