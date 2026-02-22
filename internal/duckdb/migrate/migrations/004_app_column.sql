ALTER TABLE logs ADD COLUMN IF NOT EXISTS app VARCHAR DEFAULT 'default';
CREATE INDEX IF NOT EXISTS idx_logs_app ON logs(app);
CREATE INDEX IF NOT EXISTS idx_logs_app_ts ON logs(app, timestamp);
