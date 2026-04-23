-- Migration: Auth tables
-- Description: Creates users, passcodes, device_fingerprints, and updates sessions table

-- users
CREATE TABLE IF NOT EXISTS users (
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

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone) WHERE is_active = true AND phone IS NOT NULL;

-- passcodes (OTP for login)
CREATE TABLE IF NOT EXISTS passcodes (
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

CREATE INDEX IF NOT EXISTS idx_passcodes_user ON passcodes(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_passcodes_code ON passcodes(code, is_used);
CREATE INDEX IF NOT EXISTS idx_passcodes_expires ON passcodes(expires_at);

-- device fingerprints (detect account sharing)
CREATE TABLE IF NOT EXISTS device_fingerprints (
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

CREATE INDEX IF NOT EXISTS idx_fingerprints_user ON device_fingerprints(user_id, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_fingerprints_hash ON device_fingerprints(fingerprint_hash);

-- Update sessions table to link with users and devices
DO $$ 
BEGIN
    -- Check if user_id column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'user_id'
    ) THEN
        ALTER TABLE sessions ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;
    END IF;

    -- Check if device_fingerprint_id column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'device_fingerprint_id'
    ) THEN
        ALTER TABLE sessions ADD COLUMN device_fingerprint_id UUID REFERENCES device_fingerprints(id) ON DELETE CASCADE;
    END IF;

    -- Check if token_hash column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'token_hash'
    ) THEN
        ALTER TABLE sessions ADD COLUMN token_hash VARCHAR(64) UNIQUE NOT NULL;
    END IF;

    -- Check if ip_address column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'ip_address'
    ) THEN
        ALTER TABLE sessions ADD COLUMN ip_address INET;
    END IF;

    -- Check if user_agent column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'user_agent'
    ) THEN
        ALTER TABLE sessions ADD COLUMN user_agent TEXT;
    END IF;

    -- Check if last_activity_at column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'last_activity_at'
    ) THEN
        ALTER TABLE sessions ADD COLUMN last_activity_at TIMESTAMPTZ DEFAULT NOW();
    END IF;

    -- Check if created_at column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'sessions' AND column_name = 'created_at'
    ) THEN
        ALTER TABLE sessions ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_device ON sessions(device_fingerprint_id);

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
