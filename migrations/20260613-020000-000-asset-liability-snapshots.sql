CREATE TABLE IF NOT EXISTS asset_liability_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    month TEXT NOT NULL DEFAULT '',
    assets REAL NOT NULL DEFAULT 0,
    liabilities REAL NOT NULL DEFAULT 0,
    net_worth REAL NOT NULL DEFAULT 0,
    synced_at INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_al_snapshots_month ON asset_liability_snapshots(month);
