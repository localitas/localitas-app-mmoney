CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    account_type TEXT NOT NULL DEFAULT '',
    account_subtype TEXT NOT NULL DEFAULT '',
    display_balance REAL NOT NULL DEFAULT 0,
    current_balance REAL NOT NULL DEFAULT 0,
    credit_limit REAL NOT NULL DEFAULT 0,
    is_hidden INTEGER NOT NULL DEFAULT 0,
    is_asset INTEGER NOT NULL DEFAULT 0,
    is_manual INTEGER NOT NULL DEFAULT 0,
    is_closed INTEGER NOT NULL DEFAULT 0,
    include_in_net_worth INTEGER NOT NULL DEFAULT 1,
    data_provider TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    mask TEXT NOT NULL DEFAULT '',
    monarch_created_at TEXT NOT NULL DEFAULT '',
    monarch_updated_at TEXT NOT NULL DEFAULT '',
    synced_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    date TEXT NOT NULL DEFAULT '',
    amount REAL NOT NULL DEFAULT 0,
    merchant TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    category_id TEXT NOT NULL DEFAULT '',
    category_group_name TEXT NOT NULL DEFAULT '',
    category_group_type TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    tags_json TEXT NOT NULL DEFAULT '[]',
    pending INTEGER NOT NULL DEFAULT 0,
    hide_from_reports INTEGER NOT NULL DEFAULT 0,
    plaid_name TEXT NOT NULL DEFAULT '',
    is_recurring INTEGER NOT NULL DEFAULT 0,
    review_status TEXT NOT NULL DEFAULT '',
    needs_review INTEGER NOT NULL DEFAULT 0,
    is_split_transaction INTEGER NOT NULL DEFAULT 0,
    account_id TEXT NOT NULL DEFAULT '',
    monarch_created_at TEXT NOT NULL DEFAULT '',
    monarch_updated_at TEXT NOT NULL DEFAULT '',
    synced_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(category);
CREATE INDEX IF NOT EXISTS idx_transactions_merchant ON transactions(merchant);

CREATE VIRTUAL TABLE IF NOT EXISTS transactions_fts USING fts5(
    id, merchant, category, notes,
    content='transactions', content_rowid='rowid',
    tokenize='porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS transactions_ai AFTER INSERT ON transactions BEGIN
    INSERT INTO transactions_fts(rowid, id, merchant, category, notes)
    VALUES (new.rowid, new.id, new.merchant, new.category, new.notes);
END;

CREATE TRIGGER IF NOT EXISTS transactions_ad AFTER DELETE ON transactions BEGIN
    INSERT INTO transactions_fts(transactions_fts, rowid, id, merchant, category, notes)
    VALUES ('delete', old.rowid, old.id, old.merchant, old.category, old.notes);
END;

CREATE TRIGGER IF NOT EXISTS transactions_au AFTER UPDATE ON transactions BEGIN
    INSERT INTO transactions_fts(transactions_fts, rowid, id, merchant, category, notes)
    VALUES ('delete', old.rowid, old.id, old.merchant, old.category, old.notes);
    INSERT INTO transactions_fts(rowid, id, merchant, category, notes)
    VALUES (new.rowid, new.id, new.merchant, new.category, new.notes);
END;

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    group_name TEXT NOT NULL DEFAULT '',
    group_id TEXT NOT NULL DEFAULT '',
    group_type TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    icon TEXT NOT NULL DEFAULT '',
    synced_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS budgets (
    id TEXT PRIMARY KEY,
    category_id TEXT NOT NULL DEFAULT '',
    category_name TEXT NOT NULL DEFAULT '',
    month TEXT NOT NULL DEFAULT '',
    planned REAL NOT NULL DEFAULT 0,
    actual REAL NOT NULL DEFAULT 0,
    synced_at INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_cat_month ON budgets(category_id, month);

CREATE TABLE IF NOT EXISTS recurring_transactions (
    id TEXT PRIMARY KEY,
    merchant TEXT NOT NULL DEFAULT '',
    amount REAL NOT NULL DEFAULT 0,
    frequency TEXT NOT NULL DEFAULT '',
    next_date TEXT NOT NULL DEFAULT '',
    category_name TEXT NOT NULL DEFAULT '',
    account_id TEXT NOT NULL DEFAULT '',
    account_name TEXT NOT NULL DEFAULT '',
    synced_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS investments (
    id TEXT PRIMARY KEY,
    ticker TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    quantity REAL NOT NULL DEFAULT 0,
    basis REAL NOT NULL DEFAULT 0,
    total_value REAL NOT NULL DEFAULT 0,
    price REAL NOT NULL DEFAULT 0,
    synced_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS account_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL DEFAULT '',
    balance REAL NOT NULL DEFAULT 0,
    synced_at INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_account_snapshots_date ON account_snapshots(date);

CREATE TABLE IF NOT EXISTS credit_scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL DEFAULT '',
    score INTEGER NOT NULL DEFAULT 0,
    synced_at INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_scores_date ON credit_scores(date);

CREATE TABLE IF NOT EXISTS sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at INTEGER NOT NULL DEFAULT 0,
    completed_at INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    accounts_synced INTEGER NOT NULL DEFAULT 0,
    transactions_synced INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
