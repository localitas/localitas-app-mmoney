package mmoney

import (
	"context"
	"testing"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()
	return NewStore(openMigratedDB(t, ""))
}

func TestUpsertAndListAccounts(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	err := store.UpsertAccount(ctx, "u1", LocalAccount{
		ID:                "acc1",
		DisplayName:       "Chase Checking",
		AccountType:       "depository",
		AccountSubtype:    "checking",
		DisplayBalance:    5000.50,
		IsAsset:           true,
		IncludeInNetWorth: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.UpsertAccount(ctx, "u1", LocalAccount{
		ID:             "acc2",
		DisplayName:    "Visa Card",
		AccountType:    "credit",
		DisplayBalance: -1200.00,
	})
	if err != nil {
		t.Fatal(err)
	}

	accounts, err := store.ListAccounts(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	err = store.UpsertAccount(ctx, "u1", LocalAccount{
		ID:             "acc1",
		DisplayName:    "Chase Checking Updated",
		AccountType:    "depository",
		DisplayBalance: 5500.00,
		IsAsset:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	accounts, err = store.ListAccounts(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts after upsert, got %d", len(accounts))
	}
	for _, a := range accounts {
		if a.ID == "acc1" && a.DisplayBalance != 5500.00 {
			t.Fatalf("expected updated balance 5500, got %f", a.DisplayBalance)
		}
	}
}

func TestUpsertAndListTransactions(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		store.UpsertTransaction(ctx, "u1", LocalTransaction{
			ID:       "tx" + string(rune('0'+i)),
			Date:     "2026-06-10",
			Amount:   -42.50,
			Merchant: "Grocery Store",
			Category: "Food",
			TagsJSON: "[]",
		})
	}
	for i := 0; i < 3; i++ {
		store.UpsertTransaction(ctx, "u1", LocalTransaction{
			ID:       "tx" + string(rune('5'+i)),
			Date:     "2026-06-11",
			Amount:   3000.00,
			Merchant: "Employer",
			Category: "Paycheck",
			TagsJSON: "[]",
		})
	}

	txs, total, err := store.ListTransactions(ctx, "u1", "2026-06-10", "2026-06-11", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 8 {
		t.Fatalf("expected 8 transactions, got %d", total)
	}
	if len(txs) != 8 {
		t.Fatalf("expected 8 returned, got %d", len(txs))
	}

	_, total, err = store.ListTransactions(ctx, "u1", "2026-06-10", "2026-06-10", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("expected 5 transactions for June 10, got %d", total)
	}
}

func TestSearchTransactions(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertTransaction(ctx, "u1", LocalTransaction{
		ID: "tx1", Date: "2026-06-10", Amount: -25.00,
		Merchant: "Starbucks", Category: "Coffee", TagsJSON: "[]",
	})
	store.UpsertTransaction(ctx, "u1", LocalTransaction{
		ID: "tx2", Date: "2026-06-10", Amount: -100.00,
		Merchant: "Amazon", Category: "Shopping", TagsJSON: "[]",
	})

	results, err := store.SearchTransactions(ctx, "u1", "Starbucks", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}
	if results[0].Merchant != "Starbucks" {
		t.Fatalf("expected Starbucks, got %s", results[0].Merchant)
	}
}

func TestUpsertAndListCategories(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertCategory(ctx, "u1", LocalCategory{ID: "cat1", Name: "Food", GroupName: "Expenses", GroupType: "expense"})
	store.UpsertCategory(ctx, "u1", LocalCategory{ID: "cat2", Name: "Paycheck", GroupName: "Income", GroupType: "income"})

	cats, err := store.ListCategories(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
}

func TestUpsertAndListBudgets(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertBudget(ctx, "u1", LocalBudget{ID: "b1", CategoryID: "cat1", CategoryName: "Food", Month: "2026-06", Planned: 500, Actual: -320})
	store.UpsertBudget(ctx, "u1", LocalBudget{ID: "b2", CategoryID: "cat2", CategoryName: "Transport", Month: "2026-06", Planned: 200, Actual: -150})

	budgets, err := store.ListBudgets(ctx, "u1", "2026-06")
	if err != nil {
		t.Fatal(err)
	}
	if len(budgets) != 2 {
		t.Fatalf("expected 2 budgets, got %d", len(budgets))
	}

	all, err := store.ListBudgets(ctx, "u1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 budgets with no filter, got %d", len(all))
	}
}

func TestUpsertAndListRecurring(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertRecurring(ctx, "u1", LocalRecurring{ID: "r1", Merchant: "Netflix", Amount: -15.49, Frequency: "monthly", NextDate: "2026-07-01"})
	store.UpsertRecurring(ctx, "u1", LocalRecurring{ID: "r2", Merchant: "Spotify", Amount: -9.99, Frequency: "monthly", NextDate: "2026-07-05"})

	recurring, err := store.ListRecurring(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(recurring) != 2 {
		t.Fatalf("expected 2 recurring, got %d", len(recurring))
	}
}

func TestUpsertAndListInvestments(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertInvestment(ctx, "u1", LocalInvestment{ID: "inv1", SecurityID: "sec1", Ticker: "VTI", Name: "Vanguard Total", Quantity: 50, Basis: 10000, TotalValue: 12500, Price: 250})

	investments, err := store.ListInvestments(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(investments) != 1 {
		t.Fatalf("expected 1 investment, got %d", len(investments))
	}
	if investments[0].SecurityID != "sec1" {
		t.Fatalf("expected security_id sec1, got %s", investments[0].SecurityID)
	}
}

func TestSnapshotsAndNetWorth(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertSnapshot(ctx, "u1", LocalSnapshot{Date: "2026-06-01", Balance: 100000})
	store.UpsertSnapshot(ctx, "u1", LocalSnapshot{Date: "2026-06-02", Balance: 101000})
	store.UpsertSnapshot(ctx, "u1", LocalSnapshot{Date: "2026-06-03", Balance: 99000})

	snaps, err := store.ListSnapshots(ctx, "u1", "2026-06-01", "2026-06-03")
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snaps))
	}

	store.UpsertSnapshot(ctx, "u1", LocalSnapshot{Date: "2026-06-01", Balance: 100500})
	snaps, err = store.ListSnapshots(ctx, "u1", "2026-06-01", "2026-06-01")
	if err != nil {
		t.Fatal(err)
	}
	if snaps[0].Balance != 100500 {
		t.Fatalf("expected upserted balance 100500, got %f", snaps[0].Balance)
	}
}

func TestAssetLiabilitySnapshots(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertAssetLiabilitySnapshot(ctx, "u1", AssetLiabilitySnapshot{Month: "2026-05", Assets: 500000, Liabilities: -100000, NetWorth: 400000})
	store.UpsertAssetLiabilitySnapshot(ctx, "u1", AssetLiabilitySnapshot{Month: "2026-06", Assets: 520000, Liabilities: -95000, NetWorth: 425000})

	snaps, err := store.ListAssetLiabilitySnapshots(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}
	if snaps[0].Month != "2026-05" {
		t.Fatalf("expected first month 2026-05, got %s", snaps[0].Month)
	}
}

func TestSyncLogAndHistory(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.LogSync(ctx, "u1", 1000, "ok", "", 10, 500)
	store.LogSync(ctx, "u1", 2000, "error", "auth failed", 0, 0)

	status, err := store.GetLastSync(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "error" {
		t.Fatalf("expected last status error, got %s", status.Status)
	}

	history, err := store.ListSyncHistory(ctx, "u1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	if history[0].Status != "error" {
		t.Fatalf("expected first entry (newest) to be error, got %s", history[0].Status)
	}
}

func TestConfig(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	val, err := store.GetConfig(ctx, "u1", "vault_credential_id")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Fatalf("expected empty config, got %s", val)
	}

	store.SetConfig(ctx, "u1", "vault_credential_id", "abc-123")
	val, err = store.GetConfig(ctx, "u1", "vault_credential_id")
	if err != nil {
		t.Fatal(err)
	}
	if val != "abc-123" {
		t.Fatalf("expected abc-123, got %s", val)
	}

	store.SetConfig(ctx, "u1", "vault_credential_id", "def-456")
	val, _ = store.GetConfig(ctx, "u1", "vault_credential_id")
	if val != "def-456" {
		t.Fatalf("expected updated value def-456, got %s", val)
	}
}

func TestCashflowReport(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertTransaction(ctx, "u1", LocalTransaction{ID: "t1", Date: "2026-06-01", Amount: 5000, Merchant: "Employer", Category: "Paycheck", CategoryGroupType: "income", TagsJSON: "[]"})
	store.UpsertTransaction(ctx, "u1", LocalTransaction{ID: "t2", Date: "2026-06-05", Amount: -42.50, Merchant: "Grocery", Category: "Food", CategoryGroupType: "expense", TagsJSON: "[]"})
	store.UpsertTransaction(ctx, "u1", LocalTransaction{ID: "t3", Date: "2026-06-05", Amount: -15.00, Merchant: "Netflix", Category: "Entertainment", CategoryGroupType: "expense", TagsJSON: "[]"})
	store.UpsertTransaction(ctx, "u1", LocalTransaction{ID: "t4", Date: "2026-06-10", Amount: 200, Merchant: "Side Job", Category: "Freelance", CategoryGroupType: "income", TagsJSON: "[]"})

	report, err := store.GetCashflowReport(ctx, "u1", "2026-06-01", "2026-06-30", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Months) != 1 {
		t.Fatalf("expected 1 month, got %d", len(report.Months))
	}
	if report.TotalIncome != 5200 {
		t.Fatalf("expected total income 5200, got %f", report.TotalIncome)
	}
	if report.TotalExpense != -57.50 {
		t.Fatalf("expected total expense -57.50, got %f", report.TotalExpense)
	}
	if len(report.IncomeByCategory) != 2 {
		t.Fatalf("expected 2 income categories, got %d", len(report.IncomeByCategory))
	}
	if report.IncomeByCategory[0].Category != "Paycheck" {
		t.Fatalf("expected largest income category Paycheck, got %s", report.IncomeByCategory[0].Category)
	}
	if len(report.ExpenseByCategory) != 2 {
		t.Fatalf("expected 2 expense categories, got %d", len(report.ExpenseByCategory))
	}
}

func TestPruneOldSyncLogs(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.LogSync(ctx, "u1", 1000, "ok", "", 10, 500)
	store.LogSync(ctx, "u1", 2000, "ok", "", 10, 500)

	err := store.PruneOldSyncLogs(ctx, -1)
	if err != nil {
		t.Fatal(err)
	}

	history, _ := store.ListSyncHistory(ctx, "u1", 10)
	if len(history) != 0 {
		t.Fatalf("expected 0 after prune, got %d", len(history))
	}
}

func TestInvestmentPerformance(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	store.UpsertInvestmentPerformance(ctx, "u1", LocalInvestmentPerformance{SecurityID: "sec1", Ticker: "VTI", Name: "Vanguard", Date: "2026-06-01", ReturnPercent: 5.2, Value: 12500})
	store.UpsertInvestmentPerformance(ctx, "u1", LocalInvestmentPerformance{SecurityID: "sec1", Ticker: "VTI", Name: "Vanguard", Date: "2026-06-02", ReturnPercent: 5.5, Value: 12600})

	perf, err := store.ListInvestmentPerformance(ctx, "u1", "sec1", "2026-06-01", "2026-06-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(perf) != 2 {
		t.Fatalf("expected 2 points, got %d", len(perf))
	}

	perf, err = store.ListInvestmentPerformance(ctx, "u1", "sec1", "2026-06-02", "2026-06-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(perf) != 1 {
		t.Fatalf("expected 1 point with date filter, got %d", len(perf))
	}
}
