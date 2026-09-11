CREATE TABLE devices (
    id             UUID PRIMARY KEY,
    subscriber_id  UUID NOT NULL REFERENCES subscribers(id) ON DELETE RESTRICT,
    imei           VARCHAR(15) NOT NULL,
    technology     VARCHAR(10) NOT NULL,
    status         VARCHAR(32) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_devices_technology CHECK (technology IN ('LTE', '5G')),
    CONSTRAINT chk_devices_status CHECK (status IN ('REGISTERED', 'INACTIVE'))
);

CREATE UNIQUE INDEX idx_devices_imei ON devices (imei);
CREATE INDEX idx_devices_subscriber_id ON devices (subscriber_id);
