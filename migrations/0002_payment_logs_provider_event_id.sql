BEGIN;

ALTER TABLE payment_logs
ADD COLUMN IF NOT EXISTS provider_event_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS ux_payment_logs_provider_event_id
ON payment_logs (provider_event_id)
WHERE provider_event_id IS NOT NULL;

COMMIT;

