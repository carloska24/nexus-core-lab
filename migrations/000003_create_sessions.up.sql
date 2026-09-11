CREATE TABLE sessions (
    id                 UUID PRIMARY KEY,
    device_id          UUID NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    subscriber_id      UUID NOT NULL REFERENCES subscribers(id) ON DELETE RESTRICT,
    cell_id            VARCHAR(64) NOT NULL,
    ip_address         VARCHAR(45) NOT NULL,
    status             VARCHAR(32) NOT NULL,
    disconnect_reason  VARCHAR(64),
    attached_at        TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL,
    closed_at          TIMESTAMPTZ,
    CONSTRAINT chk_sessions_status CHECK (status IN ('CONNECTED', 'DISCONNECTED')),
    CONSTRAINT chk_sessions_disconnect_reason CHECK (
        disconnect_reason IS NULL OR disconnect_reason IN ('VOLUNTARY_DETACH', 'STALE_DISCONNECT')
    )
);

CREATE INDEX idx_sessions_device_id ON sessions (device_id);
CREATE UNIQUE INDEX idx_sessions_active_device ON sessions (device_id) WHERE status = 'CONNECTED';
CREATE UNIQUE INDEX idx_sessions_active_ip ON sessions (ip_address) WHERE status = 'CONNECTED';
