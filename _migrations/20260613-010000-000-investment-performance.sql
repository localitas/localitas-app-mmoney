CREATE TABLE IF NOT EXISTS investment_performance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    security_id TEXT NOT NULL DEFAULT '',
    ticker TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    date TEXT NOT NULL DEFAULT '',
    return_percent REAL NOT NULL DEFAULT 0,
    value REAL NOT NULL DEFAULT 0,
    synced_at INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_perf_security_date ON investment_performance(security_id, date);

ALTER TABLE investments ADD COLUMN security_id TEXT NOT NULL DEFAULT '';
