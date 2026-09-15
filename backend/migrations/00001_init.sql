-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE endpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id   TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    target_url  TEXT NOT NULL,
    secret      TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    rate_limit  INTEGER NOT NULL DEFAULT 100 CHECK (rate_limit > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_endpoints_active ON endpoints (active);
CREATE INDEX idx_endpoints_created_at ON endpoints (created_at DESC);

CREATE TABLE events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id         UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    external_event_id   TEXT NOT NULL,
    event_type          TEXT NOT NULL DEFAULT 'webhook',
    payload             JSONB NOT NULL,
    status              TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'DELIVERED', 'RETRYING', 'FAILED')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (endpoint_id, external_event_id)
);

CREATE INDEX idx_events_endpoint_id ON events (endpoint_id);
CREATE INDEX idx_events_status ON events (status);
CREATE INDEX idx_events_created_at ON events (created_at DESC);
CREATE INDEX idx_events_endpoint_status ON events (endpoint_id, status);
CREATE INDEX idx_events_type ON events (event_type);

CREATE TABLE deliveries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
    endpoint_id      UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'DELIVERED', 'RETRYING', 'FAILED')),
    attempt_count    INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    response_status  INTEGER,
    response_body    TEXT,
    next_retry_at    TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deliveries_endpoint_id ON deliveries (endpoint_id);
CREATE INDEX idx_deliveries_status ON deliveries (status);
CREATE INDEX idx_deliveries_next_retry_at ON deliveries (next_retry_at)
    WHERE status = 'RETRYING' AND next_retry_at IS NOT NULL;
CREATE INDEX idx_deliveries_created_at ON deliveries (created_at DESC);

CREATE TABLE delivery_attempts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id      UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL CHECK (attempt_number > 0),
    status           TEXT NOT NULL
        CHECK (status IN ('SUCCESS', 'FAILED', 'TIMEOUT')),
    response_status  INTEGER,
    response_body    TEXT,
    duration_ms      BIGINT NOT NULL DEFAULT 0,
    error            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (delivery_id, attempt_number)
);

CREATE INDEX idx_delivery_attempts_delivery_id ON delivery_attempts (delivery_id);
CREATE INDEX idx_delivery_attempts_created_at ON delivery_attempts (created_at DESC);

CREATE TABLE api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    key_prefix    TEXT NOT NULL,
    key_hash      TEXT NOT NULL UNIQUE,
    revoked_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_prefix ON api_keys (key_prefix);
CREATE INDEX idx_api_keys_active ON api_keys (id) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS delivery_attempts;
DROP TABLE IF EXISTS deliveries;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS endpoints;
