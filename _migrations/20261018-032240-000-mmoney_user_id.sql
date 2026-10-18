-- Migration: mmoney_user_id
-- Created: 2026-10-18T03:22:40Z
--
-- Every synced row and setting belongs to the user whose Monarch login it
-- came from. Rows from before ownership have user '' and are given to the
-- first admin who signs in.

ALTER TABLE accounts ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE categories ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE budgets ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE recurring_transactions ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE investments ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE account_snapshots ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE credit_scores ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sync_log ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE investment_performance ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE asset_liability_snapshots ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_id, date);
CREATE INDEX IF NOT EXISTS idx_categories_user ON categories(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_user ON recurring_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_investments_user ON investments(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_log_user ON sync_log(user_id);

DROP INDEX IF EXISTS idx_budgets_cat_month;
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_cat_month ON budgets(user_id, category_id, month);
DROP INDEX IF EXISTS idx_account_snapshots_date;
CREATE UNIQUE INDEX IF NOT EXISTS idx_account_snapshots_date ON account_snapshots(user_id, date);
DROP INDEX IF EXISTS idx_credit_scores_date;
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_scores_date ON credit_scores(user_id, date);
DROP INDEX IF EXISTS idx_inv_perf_security_date;
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_perf_security_date ON investment_performance(user_id, security_id, date);
DROP INDEX IF EXISTS idx_al_snapshots_month;
CREATE UNIQUE INDEX IF NOT EXISTS idx_al_snapshots_month ON asset_liability_snapshots(user_id, month);

CREATE TABLE IF NOT EXISTS user_config (
    user_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, key)
);

INSERT OR IGNORE INTO user_config (user_id, key, value) SELECT '', key, value FROM config;
DROP TABLE IF EXISTS config;
