package mmoney

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const DatabaseName = "mmoney"

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func OpenStore(coreURL, dbID, token string) (*Store, error) {
	dsn := fmt.Sprintf("%s?database_id=%s&token=%s", coreURL, dbID, token)
	db, err := sql.Open("localitas", dsn)
	if err != nil {
		return nil, fmt.Errorf("open localitas db: %w", err)
	}
	return NewStore(db), nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Rows synced from Monarch are keyed by Monarch's own ids. An upsert
// updates only the syncing user's row: when two users sync the same Monarch
// login, the first to sync a row keeps it and the other's copy is not stored.

func (s *Store) UpsertAccount(ctx context.Context, userID string, a LocalAccount) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO accounts (id, user_id, display_name, account_type, account_subtype, display_balance, current_balance, credit_limit, is_hidden, is_asset, is_manual, is_closed, include_in_net_worth, data_provider, icon, logo_url, mask, monarch_created_at, monarch_updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET display_name=excluded.display_name, account_type=excluded.account_type, account_subtype=excluded.account_subtype, display_balance=excluded.display_balance, current_balance=excluded.current_balance, credit_limit=excluded.credit_limit, is_hidden=excluded.is_hidden, is_asset=excluded.is_asset, is_manual=excluded.is_manual, is_closed=excluded.is_closed, include_in_net_worth=excluded.include_in_net_worth, data_provider=excluded.data_provider, icon=excluded.icon, logo_url=excluded.logo_url, mask=excluded.mask, monarch_created_at=excluded.monarch_created_at, monarch_updated_at=excluded.monarch_updated_at, synced_at=excluded.synced_at
		WHERE accounts.user_id = excluded.user_id`,
		a.ID, userID, a.DisplayName, a.AccountType, a.AccountSubtype, a.DisplayBalance, a.CurrentBalance, a.CreditLimit, boolToInt(a.IsHidden), boolToInt(a.IsAsset), boolToInt(a.IsManual), boolToInt(a.IsClosed), boolToInt(a.IncludeInNetWorth), a.DataProvider, a.Icon, a.LogoURL, a.Mask, a.MonarchCreatedAt, a.MonarchUpdatedAt, now)
	return err
}

func (s *Store) ListAccounts(ctx context.Context, userID string) ([]LocalAccount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, display_name, account_type, account_subtype, display_balance, current_balance, credit_limit, is_hidden, is_asset, is_manual, is_closed, include_in_net_worth, data_provider, icon, logo_url, mask, monarch_created_at, monarch_updated_at, synced_at FROM accounts WHERE user_id = ? ORDER BY display_name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalAccount, 0)
	for rows.Next() {
		var a LocalAccount
		var isHidden, isAsset, isManual, isClosed, includeNW int
		if err := rows.Scan(&a.ID, &a.DisplayName, &a.AccountType, &a.AccountSubtype, &a.DisplayBalance, &a.CurrentBalance, &a.CreditLimit, &isHidden, &isAsset, &isManual, &isClosed, &includeNW, &a.DataProvider, &a.Icon, &a.LogoURL, &a.Mask, &a.MonarchCreatedAt, &a.MonarchUpdatedAt, &a.SyncedAt); err != nil {
			return nil, err
		}
		a.IsHidden = isHidden == 1
		a.IsAsset = isAsset == 1
		a.IsManual = isManual == 1
		a.IsClosed = isClosed == 1
		a.IncludeInNetWorth = includeNW == 1
		out = append(out, a)
	}
	return out, nil
}

func (s *Store) UpsertTransaction(ctx context.Context, userID string, t LocalTransaction) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO transactions (id, user_id, date, amount, merchant, category, category_id, category_group_name, category_group_type, notes, tags_json, pending, hide_from_reports, plaid_name, is_recurring, review_status, needs_review, is_split_transaction, account_id, monarch_created_at, monarch_updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET date=excluded.date, amount=excluded.amount, merchant=excluded.merchant, category=excluded.category, category_id=excluded.category_id, category_group_name=excluded.category_group_name, category_group_type=excluded.category_group_type, notes=excluded.notes, tags_json=excluded.tags_json, pending=excluded.pending, hide_from_reports=excluded.hide_from_reports, plaid_name=excluded.plaid_name, is_recurring=excluded.is_recurring, review_status=excluded.review_status, needs_review=excluded.needs_review, is_split_transaction=excluded.is_split_transaction, account_id=excluded.account_id, monarch_created_at=excluded.monarch_created_at, monarch_updated_at=excluded.monarch_updated_at, synced_at=excluded.synced_at
		WHERE transactions.user_id = excluded.user_id`,
		t.ID, userID, t.Date, t.Amount, t.Merchant, t.Category, t.CategoryID, t.CategoryGroupName, t.CategoryGroupType, t.Notes, t.TagsJSON, boolToInt(t.Pending), boolToInt(t.HideFromReports), t.PlaidName, boolToInt(t.IsRecurring), t.ReviewStatus, boolToInt(t.NeedsReview), boolToInt(t.IsSplitTransaction), t.AccountID, t.MonarchCreatedAt, t.MonarchUpdatedAt, now)
	return err
}

func (s *Store) ListTransactions(ctx context.Context, userID, startDate, endDate string, limit, offset int) ([]LocalTransaction, int, error) {
	if limit <= 0 {
		limit = 50
	}

	countQuery := `SELECT COUNT(*) FROM transactions WHERE user_id = ?`
	query := `SELECT id, date, amount, merchant, category, category_id, category_group_name, category_group_type, notes, tags_json, pending, hide_from_reports, plaid_name, is_recurring, review_status, needs_review, is_split_transaction, account_id, monarch_created_at, monarch_updated_at, synced_at FROM transactions WHERE user_id = ?`

	args := []interface{}{userID}
	if startDate != "" {
		countQuery += ` AND date >= ?`
		query += ` AND date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		countQuery += ` AND date <= ?`
		query += ` AND date <= ?`
		args = append(args, endDate)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query += ` ORDER BY date DESC, amount DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanTransactions(rows, total)
}

func (s *Store) SearchTransactions(ctx context.Context, userID, q string, limit int) ([]LocalTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.date, t.amount, t.merchant, t.category, t.category_id, t.category_group_name, t.category_group_type, t.notes, t.tags_json, t.pending, t.hide_from_reports, t.plaid_name, t.is_recurring, t.review_status, t.needs_review, t.is_split_transaction, t.account_id, t.monarch_created_at, t.monarch_updated_at, t.synced_at
		FROM transactions t
		JOIN transactions_fts ON t.rowid = transactions_fts.rowid
		WHERE transactions_fts MATCH ? AND t.user_id = ?
		ORDER BY rank LIMIT ?`, q, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	txs, _, err := scanTransactions(rows, 0)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

func (s *Store) UpsertCategory(ctx context.Context, userID string, c LocalCategory) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO categories (id, user_id, name, group_name, group_id, group_type, sort_order, icon, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, group_name=excluded.group_name, group_id=excluded.group_id, group_type=excluded.group_type, sort_order=excluded.sort_order, icon=excluded.icon, synced_at=excluded.synced_at
		WHERE categories.user_id = excluded.user_id`,
		c.ID, userID, c.Name, c.GroupName, c.GroupID, c.GroupType, c.SortOrder, c.Icon, now)
	return err
}

func (s *Store) ListCategories(ctx context.Context, userID string) ([]LocalCategory, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, group_name, group_id, group_type, sort_order, icon, synced_at FROM categories WHERE user_id = ? ORDER BY group_name, sort_order, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalCategory, 0)
	for rows.Next() {
		var c LocalCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.GroupName, &c.GroupID, &c.GroupType, &c.SortOrder, &c.Icon, &c.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) UpsertBudget(ctx context.Context, userID string, b LocalBudget) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO budgets (id, user_id, category_id, category_name, month, planned, actual, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET category_name=excluded.category_name, planned=excluded.planned, actual=excluded.actual, synced_at=excluded.synced_at
		WHERE budgets.user_id = excluded.user_id`,
		b.ID, userID, b.CategoryID, b.CategoryName, b.Month, b.Planned, b.Actual, now)
	return err
}

func (s *Store) ListBudgets(ctx context.Context, userID, month string) ([]LocalBudget, error) {
	query := `SELECT id, category_id, category_name, month, planned, actual, synced_at FROM budgets WHERE user_id = ?`
	args := []interface{}{userID}
	if month != "" {
		query += ` AND month = ?`
		args = append(args, month)
	}
	query += ` ORDER BY category_name`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalBudget, 0)
	for rows.Next() {
		var b LocalBudget
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.CategoryName, &b.Month, &b.Planned, &b.Actual, &b.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (s *Store) UpsertRecurring(ctx context.Context, userID string, r LocalRecurring) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO recurring_transactions (id, user_id, merchant, amount, frequency, next_date, category_name, account_id, account_name, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET merchant=excluded.merchant, amount=excluded.amount, frequency=excluded.frequency, next_date=excluded.next_date, category_name=excluded.category_name, account_id=excluded.account_id, account_name=excluded.account_name, synced_at=excluded.synced_at
		WHERE recurring_transactions.user_id = excluded.user_id`,
		r.ID, userID, r.Merchant, r.Amount, r.Frequency, r.NextDate, r.CategoryName, r.AccountID, r.AccountName, now)
	return err
}

func (s *Store) ListRecurring(ctx context.Context, userID string) ([]LocalRecurring, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, merchant, amount, frequency, next_date, category_name, account_id, account_name, synced_at FROM recurring_transactions WHERE user_id = ? ORDER BY next_date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalRecurring, 0)
	for rows.Next() {
		var r LocalRecurring
		if err := rows.Scan(&r.ID, &r.Merchant, &r.Amount, &r.Frequency, &r.NextDate, &r.CategoryName, &r.AccountID, &r.AccountName, &r.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) UpsertInvestment(ctx context.Context, userID string, inv LocalInvestment) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO investments (id, user_id, security_id, ticker, name, quantity, basis, total_value, price, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET security_id=excluded.security_id, ticker=excluded.ticker, name=excluded.name, quantity=excluded.quantity, basis=excluded.basis, total_value=excluded.total_value, price=excluded.price, synced_at=excluded.synced_at
		WHERE investments.user_id = excluded.user_id`,
		inv.ID, userID, inv.SecurityID, inv.Ticker, inv.Name, inv.Quantity, inv.Basis, inv.TotalValue, inv.Price, now)
	return err
}

func (s *Store) ListInvestments(ctx context.Context, userID string) ([]LocalInvestment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, security_id, ticker, name, quantity, basis, total_value, price, synced_at FROM investments WHERE user_id = ? ORDER BY total_value DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalInvestment, 0)
	for rows.Next() {
		var inv LocalInvestment
		if err := rows.Scan(&inv.ID, &inv.SecurityID, &inv.Ticker, &inv.Name, &inv.Quantity, &inv.Basis, &inv.TotalValue, &inv.Price, &inv.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, nil
}

func (s *Store) UpsertInvestmentPerformance(ctx context.Context, userID string, p LocalInvestmentPerformance) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO investment_performance (user_id, security_id, ticker, name, date, return_percent, value, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, security_id, date) DO UPDATE SET ticker=excluded.ticker, name=excluded.name, return_percent=excluded.return_percent, value=excluded.value, synced_at=excluded.synced_at`,
		userID, p.SecurityID, p.Ticker, p.Name, p.Date, p.ReturnPercent, p.Value, now)
	return err
}

func (s *Store) UpsertAssetLiabilitySnapshot(ctx context.Context, userID string, snap AssetLiabilitySnapshot) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO asset_liability_snapshots (user_id, month, assets, liabilities, net_worth, investments, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, month) DO UPDATE SET assets=excluded.assets, liabilities=excluded.liabilities, net_worth=excluded.net_worth, investments=excluded.investments, synced_at=excluded.synced_at`,
		userID, snap.Month, snap.Assets, snap.Liabilities, snap.NetWorth, snap.Investments, now)
	return err
}

func (s *Store) ListAssetLiabilitySnapshots(ctx context.Context, userID string) ([]AssetLiabilitySnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT month, assets, liabilities, net_worth, investments FROM asset_liability_snapshots WHERE user_id = ? ORDER BY month`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AssetLiabilitySnapshot, 0)
	for rows.Next() {
		var s AssetLiabilitySnapshot
		if err := rows.Scan(&s.Month, &s.Assets, &s.Liabilities, &s.NetWorth, &s.Investments); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (s *Store) ListInvestmentPerformance(ctx context.Context, userID, securityID, startDate, endDate string) ([]LocalInvestmentPerformance, error) {
	query := `SELECT security_id, ticker, name, date, return_percent, value FROM investment_performance WHERE user_id = ? AND security_id = ?`
	args := []interface{}{userID, securityID}
	if startDate != "" {
		query += ` AND date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY date`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalInvestmentPerformance, 0)
	for rows.Next() {
		var p LocalInvestmentPerformance
		if err := rows.Scan(&p.SecurityID, &p.Ticker, &p.Name, &p.Date, &p.ReturnPercent, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Store) GetSecurityIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT security_id FROM investments WHERE user_id = ? AND security_id != '' ORDER BY total_value DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *Store) UpsertSnapshot(ctx context.Context, userID string, snap LocalSnapshot) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO account_snapshots (user_id, date, balance, synced_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET balance=excluded.balance, synced_at=excluded.synced_at`,
		userID, snap.Date, snap.Balance, now)
	return err
}

func (s *Store) ListSnapshots(ctx context.Context, userID, startDate, endDate string) ([]LocalSnapshot, error) {
	query := `SELECT date, balance FROM account_snapshots WHERE user_id = ?`
	args := []interface{}{userID}
	if startDate != "" {
		query += ` AND date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY date`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalSnapshot, 0)
	for rows.Next() {
		var snap LocalSnapshot
		if err := rows.Scan(&snap.Date, &snap.Balance); err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

func (s *Store) UpsertCreditScore(ctx context.Context, userID string, cs LocalCreditScore) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO credit_scores (user_id, date, score, synced_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET score=excluded.score, synced_at=excluded.synced_at`,
		userID, cs.Date, cs.Score, now)
	return err
}

func (s *Store) ListCreditScores(ctx context.Context, userID string) ([]LocalCreditScore, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT date, score FROM credit_scores WHERE user_id = ? ORDER BY date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LocalCreditScore, 0)
	for rows.Next() {
		var cs LocalCreditScore
		if err := rows.Scan(&cs.Date, &cs.Score); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}

func (s *Store) GetFIREReport(ctx context.Context, userID string) (*FIREReport, error) {
	now := time.Now().UTC()
	oneYearAgo := now.AddDate(-1, 0, 0).Format("2006-01-02")
	twoYearsAgo := now.AddDate(-2, 0, 0).Format("2006-01-02")

	expenseFilter := `user_id = ? AND amount < 0 AND hide_from_reports = 0 AND category_group_type != 'transfer'`

	var annualExpenses float64
	s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(ABS(amount)), 0) FROM transactions WHERE `+expenseFilter+` AND date >= ?`, userID, oneYearAgo).Scan(&annualExpenses)

	var prevYearExpenses float64
	s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(ABS(amount)), 0) FROM transactions WHERE `+expenseFilter+` AND date >= ? AND date < ?`, userID, twoYearsAgo, oneYearAgo).Scan(&prevYearExpenses)

	var currentMonths, prevMonths int
	s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT substr(date, 1, 7)) FROM transactions WHERE `+expenseFilter+` AND date >= ?`, userID, oneYearAgo).Scan(&currentMonths)
	s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT substr(date, 1, 7)) FROM transactions WHERE `+expenseFilter+` AND date >= ? AND date < ?`, userID, twoYearsAgo, oneYearAgo).Scan(&prevMonths)

	var investmentValue float64
	s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(display_balance), 0) FROM accounts WHERE user_id = ? AND account_type = 'brokerage'`, userID).Scan(&investmentValue)

	var annualIncome float64
	s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = ? AND amount >= 0 AND hide_from_reports = 0 AND category_group_type != 'transfer' AND date >= ?`, userID, oneYearAgo).Scan(&annualIncome)

	var investOneYearAgo float64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(balance, 0) FROM account_snapshots WHERE user_id = ? AND date <= ? ORDER BY date DESC LIMIT 1`, userID, oneYearAgo).Scan(&investOneYearAgo)
	if err != nil || investOneYearAgo == 0 {
		s.db.QueryRowContext(ctx, `SELECT COALESCE(balance, 0) FROM account_snapshots WHERE user_id = ? ORDER BY date ASC LIMIT 1`, userID).Scan(&investOneYearAgo)
	}

	investmentGrowthYoY := 0.0
	if investOneYearAgo > 0 {
		investmentGrowthYoY = (investmentValue - investOneYearAgo) / investOneYearAgo * 100
	}

	investmentGrowthDollar := investmentValue - investOneYearAgo
	growthExceedsExpenses := investmentGrowthDollar > annualExpenses
	surplus := investmentGrowthDollar - annualExpenses

	expenseGrowthYoY := 0.0
	if prevYearExpenses > 0 && annualExpenses > 0 && currentMonths >= 6 && prevMonths >= 6 {
		currentNormalized := annualExpenses / float64(currentMonths) * 12
		prevNormalized := prevYearExpenses / float64(prevMonths) * 12
		expenseGrowthYoY = (currentNormalized - prevNormalized) / prevNormalized * 100
	}

	totalIncome := annualIncome + investmentGrowthDollar
	savingsRate := 0.0
	if totalIncome > 0 {
		savingsRate = (totalIncome - annualExpenses) / totalIncome * 100
	}

	yearsToFIRE := -1.0
	if investmentGrowthYoY > 0 && !growthExceedsExpenses {
		growthRate := investmentGrowthYoY / 100
		expenseRate := 0.0
		if expenseGrowthYoY > 0 {
			expenseRate = expenseGrowthYoY / 100
		}
		projected := investmentValue
		projectedExpenses := annualExpenses
		for y := 1.0; y <= 100; y++ {
			projected *= (1 + growthRate)
			projectedExpenses *= (1 + expenseRate)
			if projected*growthRate > projectedExpenses {
				yearsToFIRE = y
				break
			}
		}
	} else if growthExceedsExpenses {
		yearsToFIRE = 0
	}

	twoYearsAgoDate := now.AddDate(-2, 0, 0).Format("2006-01-02")
	expRows, err := s.db.QueryContext(ctx, `SELECT substr(date, 1, 7) as month, SUM(ABS(amount)) as total
		FROM transactions WHERE `+expenseFilter+` AND date >= ?
		GROUP BY month ORDER BY month`, userID, twoYearsAgoDate)
	if err != nil {
		return nil, err
	}
	defer expRows.Close()
	var monthlyExpenses []MonthlyAmount
	for expRows.Next() {
		var m MonthlyAmount
		if err := expRows.Scan(&m.Month, &m.Amount); err != nil {
			return nil, err
		}
		monthlyExpenses = append(monthlyExpenses, m)
	}

	invRows, err := s.db.QueryContext(ctx, `SELECT month, investments FROM asset_liability_snapshots
		WHERE user_id = ? AND month >= ? AND investments > 0 ORDER BY month`, userID, twoYearsAgoDate[:7])
	if err != nil {
		return nil, err
	}
	defer invRows.Close()
	var monthlyInvestments []MonthlyAmount
	for invRows.Next() {
		var m MonthlyAmount
		if err := invRows.Scan(&m.Month, &m.Amount); err != nil {
			return nil, err
		}
		monthlyInvestments = append(monthlyInvestments, m)
	}

	return &FIREReport{
		AnnualExpenses:         annualExpenses,
		InvestmentValue:        investmentValue,
		InvestmentGrowthDollar: investmentGrowthDollar,
		InvestmentGrowthYoY:    investmentGrowthYoY,
		ExpenseGrowthYoY:       expenseGrowthYoY,
		GrowthExceedsExpenses:  growthExceedsExpenses,
		Surplus:                surplus,
		SavingsRate:            savingsRate,
		YearsToFIRE:            yearsToFIRE,
		MonthlyExpenses:        monthlyExpenses,
		MonthlyInvestments:     monthlyInvestments,
	}, nil
}

func (s *Store) GetCashflowReport(ctx context.Context, userID, startDate, endDate string, recurringOnly bool) (*CashflowReport, error) {
	recurringFilter := ""
	if recurringOnly {
		recurringFilter = " AND is_recurring = 1"
	}

	monthRows, err := s.db.QueryContext(ctx, `SELECT substr(date, 1, 7) as month,
		SUM(CASE WHEN amount >= 0 THEN amount ELSE 0 END) as income,
		SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END) as expense
		FROM transactions
		WHERE user_id = ? AND hide_from_reports = 0 AND date >= ? AND date <= ?`+recurringFilter+`
		GROUP BY month ORDER BY month`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer monthRows.Close()

	var months []CashflowMonth
	var totalIncome, totalExpense float64
	for monthRows.Next() {
		var m CashflowMonth
		if err := monthRows.Scan(&m.Month, &m.Income, &m.Expense); err != nil {
			return nil, err
		}
		totalIncome += m.Income
		totalExpense += m.Expense
		months = append(months, m)
	}

	incomeRows, err := s.db.QueryContext(ctx, `SELECT category, SUM(amount) as total
		FROM transactions
		WHERE user_id = ? AND hide_from_reports = 0 AND amount >= 0 AND date >= ? AND date <= ?`+recurringFilter+`
		GROUP BY category ORDER BY total DESC`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer incomeRows.Close()

	var incomeByCategory []CategoryTotal
	for incomeRows.Next() {
		var c CategoryTotal
		if err := incomeRows.Scan(&c.Category, &c.Amount); err != nil {
			return nil, err
		}
		c.AbsAmount = c.Amount
		incomeByCategory = append(incomeByCategory, c)
	}

	expenseRows, err := s.db.QueryContext(ctx, `SELECT category, SUM(amount) as total
		FROM transactions
		WHERE user_id = ? AND hide_from_reports = 0 AND amount < 0 AND date >= ? AND date <= ?`+recurringFilter+`
		GROUP BY category ORDER BY total ASC`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer expenseRows.Close()

	var expenseByCategory []CategoryTotal
	for expenseRows.Next() {
		var c CategoryTotal
		if err := expenseRows.Scan(&c.Category, &c.Amount); err != nil {
			return nil, err
		}
		c.AbsAmount = -c.Amount
		expenseByCategory = append(expenseByCategory, c)
	}

	savings := totalIncome + totalExpense
	savingsRate := 0.0
	if totalIncome > 0 {
		savingsRate = savings / totalIncome * 100
	}

	return &CashflowReport{
		Months:            months,
		TotalIncome:       totalIncome,
		TotalExpense:      totalExpense,
		Savings:           savings,
		SavingsRate:       savingsRate,
		IncomeByCategory:  incomeByCategory,
		ExpenseByCategory: expenseByCategory,
	}, nil
}

func (s *Store) LogSync(ctx context.Context, userID string, startedAt int64, status, errMsg string, accounts, transactions int) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_log (user_id, started_at, completed_at, status, error_message, accounts_synced, transactions_synced) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, startedAt, now, status, errMsg, accounts, transactions)
	return err
}

func (s *Store) GetLastSync(ctx context.Context, userID string) (*SyncStatus, error) {
	var startedAt, completedAt int64
	var status, errMsg string
	var accounts, transactions int
	err := s.db.QueryRowContext(ctx, `SELECT started_at, completed_at, status, error_message, accounts_synced, transactions_synced FROM sync_log WHERE user_id = ? ORDER BY id DESC LIMIT 1`, userID).Scan(&startedAt, &completedAt, &status, &errMsg, &accounts, &transactions)
	if err == sql.ErrNoRows {
		return &SyncStatus{Status: "never"}, nil
	}
	if err != nil {
		return nil, err
	}
	t := time.Unix(completedAt, 0).UTC()
	return &SyncStatus{
		LastSyncAt:       &t,
		Status:           status,
		ErrorMessage:     errMsg,
		AccountCount:     accounts,
		TransactionCount: transactions,
	}, nil
}

func (s *Store) ListSyncHistory(ctx context.Context, userID string, limit int) ([]SyncLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, started_at, completed_at, status, error_message, accounts_synced, transactions_synced FROM sync_log WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SyncLogEntry, 0)
	for rows.Next() {
		var e SyncLogEntry
		var startedAt, completedAt int64
		if err := rows.Scan(&e.ID, &startedAt, &completedAt, &e.Status, &e.ErrorMessage, &e.AccountsSynced, &e.TransactionsSynced); err != nil {
			return nil, err
		}
		e.StartedAt = time.Unix(startedAt, 0).UTC()
		e.CompletedAt = time.Unix(completedAt, 0).UTC()
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) PruneOldSyncLogs(ctx context.Context, olderThanDays int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -olderThanDays).Unix()
	_, err := s.db.ExecContext(ctx, `DELETE FROM sync_log WHERE completed_at < ?`, cutoff)
	return err
}

func (s *Store) GetConfig(ctx context.Context, userID, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM user_config WHERE user_id = ? AND key = ?`, userID, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *Store) SetConfig(ctx context.Context, userID, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO user_config (user_id, key, value) VALUES (?, ?, ?) ON CONFLICT(user_id, key) DO UPDATE SET value=excluded.value`, userID, key, value)
	return err
}

// ConfiguredUsers lists the users who picked a credential to sync from.
func (s *Store) ConfiguredUsers(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id FROM user_config WHERE key = 'vault_credential_id' AND value != '' AND user_id != '' ORDER BY user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ownedTables are the tables whose rows belong to a user.
var ownedTables = []string{"accounts", "transactions", "categories", "budgets", "recurring_transactions", "investments", "account_snapshots", "credit_scores", "sync_log", "investment_performance", "asset_liability_snapshots", "user_config"}

// ClaimLegacyData gives userID the rows and settings synced before
// per-user ownership, which the migration left under user ”. Where userID
// already has the same row, theirs is kept. Running it again finds nothing
// to claim.
func (s *Store) ClaimLegacyData(ctx context.Context, userID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range ownedTables {
		if _, err := tx.ExecContext(ctx, `UPDATE OR IGNORE `+table+` SET user_id = ? WHERE user_id = ''`, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id = ''`); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanTransactions(rows *sql.Rows, total int) ([]LocalTransaction, int, error) {
	out := make([]LocalTransaction, 0)
	for rows.Next() {
		var t LocalTransaction
		var pending, hideFromReports, isRecurring, needsReview, isSplit int
		if err := rows.Scan(&t.ID, &t.Date, &t.Amount, &t.Merchant, &t.Category, &t.CategoryID, &t.CategoryGroupName, &t.CategoryGroupType, &t.Notes, &t.TagsJSON, &pending, &hideFromReports, &t.PlaidName, &isRecurring, &t.ReviewStatus, &needsReview, &isSplit, &t.AccountID, &t.MonarchCreatedAt, &t.MonarchUpdatedAt, &t.SyncedAt); err != nil {
			return nil, 0, err
		}
		t.Pending = pending == 1
		t.HideFromReports = hideFromReports == 1
		t.IsRecurring = isRecurring == 1
		t.NeedsReview = needsReview == 1
		t.IsSplitTransaction = isSplit == 1
		out = append(out, t)
	}
	return out, total, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
