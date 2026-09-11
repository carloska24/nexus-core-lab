CREATE TABLE subscribers (
    id                  UUID PRIMARY KEY,
    imsi                VARCHAR(15) NOT NULL,
    msisdn              VARCHAR(15) NOT NULL,
    status              VARCHAR(32) NOT NULL,
    suspension_reason   TEXT,
    deactivation_reason TEXT,
    created_at          TIMESTAMPTZ NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_subscribers_status CHECK (
        status IN ('PENDING_ACTIVATION', 'ACTIVE', 'SUSPENDED', 'DEACTIVATED')
    )
);

CREATE UNIQUE INDEX idx_subscribers_imsi ON subscribers (imsi);
CREATE UNIQUE INDEX idx_subscribers_msisdn ON subscribers (msisdn);
