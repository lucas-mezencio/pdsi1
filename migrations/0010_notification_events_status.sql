ALTER TABLE notification_events
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_notification_events_status
    ON notification_events(status)
    WHERE status <> '';