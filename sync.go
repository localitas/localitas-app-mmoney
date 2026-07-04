package mmoney

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/localitas/localitas-app-mmoney/internal/monarch"
	"github.com/localitas/localitas-app-mmoney/internal/monarchauth"
	"github.com/localitas/localitas-app-mmoney/internal/monarcherr"
	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
)

const monarchGraphQLEndpoint = "https://api.monarch.com/graphql"
const sp500SecurityID = "119563102644090322"
const bondsSecurityID = "78665972690706707"

type VaultCredential struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	MFASecret string `json:"mfa_secret"`
}

func (a *App) resolveVaultCredential(ctx context.Context) (*VaultCredential, error) {
	credID, err := a.Store.GetConfig(ctx, "vault_credential_id")
	if err != nil || credID == "" {
		return nil, fmt.Errorf("no vault credential configured: set vault_credential_id in config")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", a.coreURL+"/apps/vault/api/credentials/"+credID+"/secrets", nil)
	if err != nil {
		return nil, err
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("vault returned %d", resp.StatusCode)
	}

	var secrets map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		return nil, fmt.Errorf("vault decode failed: %w", err)
	}

	return &VaultCredential{
		Email:     secrets["email"],
		Password:  secrets["password"],
		MFASecret: secrets["mfa_secret"],
	}, nil
}

func (a *App) authenticate(ctx context.Context) (*monarchgql.Client, error) {
	if a.cachedToken != "" && time.Now().UTC().Before(a.cachedTokenExpiry) {
		return monarchgql.NewClient(monarchGraphQLEndpoint, a.cachedToken, 30*time.Second), nil
	}

	cred, err := a.resolveVaultCredential(ctx)
	if err != nil {
		return nil, err
	}

	session, err := monarchauth.Authenticate(cred.Email, cred.Password, "", cred.MFASecret)
	if err != nil {
		return nil, fmt.Errorf("monarch auth failed: %w", err)
	}

	a.cachedToken = session.Token
	a.cachedTokenExpiry = time.Now().UTC().Add(9 * time.Minute)

	return monarchgql.NewClient(monarchGraphQLEndpoint, session.Token, 30*time.Second), nil
}

func (a *App) RunSync(ctx context.Context) error {
	startedAt := time.Now().UTC().Unix()
	accountCount := 0
	txCount := 0

	a.Store.PruneOldSyncLogs(ctx, 3)

	gqlClient, err := a.authenticate(ctx)
	if err != nil {
		a.Store.LogSync(ctx, startedAt, "error", err.Error(), 0, 0)
		return err
	}

	svc := monarch.NewService(gqlClient)

	accounts, err := svc.ListAccounts(ctx)
	if err != nil {
		if merr, ok := err.(*monarcherr.Error); ok && merr.Code == monarcherr.AuthSessionExpired {
			a.cachedToken = ""
			gqlClient, err = a.authenticate(ctx)
			if err != nil {
				a.Store.LogSync(ctx, startedAt, "error", err.Error(), 0, 0)
				return err
			}
			svc = monarch.NewService(gqlClient)
			accounts, err = svc.ListAccounts(ctx)
		}
		if err != nil {
			a.Store.LogSync(ctx, startedAt, "error", err.Error(), 0, 0)
			return err
		}
	}

	for _, acc := range accounts {
		if err := a.Store.UpsertAccount(ctx, LocalAccount{
			ID:                acc.ID,
			DisplayName:       acc.DisplayName,
			AccountType:       acc.AccountType,
			AccountSubtype:    acc.AccountSubtype,
			DisplayBalance:    acc.DisplayBalance,
			CurrentBalance:    acc.CurrentBalance,
			CreditLimit:       acc.Limit,
			IsHidden:          acc.IsHidden,
			IsAsset:           acc.IsAsset,
			IsManual:          acc.IsManual,
			IsClosed:          acc.IsClosed,
			IncludeInNetWorth: acc.IncludeInNetWorth,
			DataProvider:      acc.DataProvider,
			Icon:              acc.Icon,
			LogoURL:           acc.LogoURL,
			Mask:              acc.Mask,
			MonarchCreatedAt:  acc.CreatedAt,
			MonarchUpdatedAt:  acc.UpdatedAt,
		}); err != nil {
			logger.Error("upsert account failed", "account_id", acc.ID, "error", err)
		} else {
			accountCount++
		}
	}

	txCount, err = a.syncTransactions(ctx, svc)
	if err != nil {
		logger.Error("transaction sync error", "error", err)
	}

	a.syncCategories(ctx, svc)
	a.syncBudgets(ctx, svc)
	a.syncRecurring(ctx, svc)
	a.syncInvestments(ctx, svc)
	a.syncSnapshots(ctx, svc)
	a.syncAssetLiabilitySnapshots(ctx, svc)
	a.syncCreditScores(ctx, svc)

	a.Store.LogSync(ctx, startedAt, "ok", "", accountCount, txCount)
	logger.Info("sync complete", "accounts", accountCount, "transactions", txCount)
	return nil
}

func (a *App) syncTransactions(ctx context.Context, svc *monarch.Service) (int, error) {
	startDate := ""
	lastSync, _ := a.Store.GetConfig(ctx, "last_tx_sync_date")
	if lastSync != "" {
		t, err := time.Parse("2006-01-02", lastSync)
		if err == nil {
			startDate = t.AddDate(0, 0, -7).Format("2006-01-02")
		}
	}

	today := time.Now().UTC().Format("2006-01-02")
	txs, err := svc.ListAllTransactions(ctx, monarch.ListTransactionsOptions{
		Limit:     1000,
		StartDate: startDate,
		EndDate:   today,
	})
	if err != nil {
		return 0, err
	}

	count := 0
	for _, tx := range txs {
		tagsJSON, _ := json.Marshal(tx.Tags)
		if err := a.Store.UpsertTransaction(ctx, LocalTransaction{
			ID:                 tx.ID,
			Date:               tx.Date,
			Amount:             tx.Amount,
			Merchant:           tx.Merchant,
			Category:           tx.Category,
			CategoryID:         tx.CategoryID,
			CategoryGroupName:  tx.CategoryGroupName,
			CategoryGroupType:  tx.CategoryGroupType,
			Notes:              tx.Notes,
			TagsJSON:           string(tagsJSON),
			Pending:            tx.Pending,
			HideFromReports:    tx.HideFromReports,
			PlaidName:          tx.PlaidName,
			IsRecurring:        tx.IsRecurring,
			ReviewStatus:       tx.ReviewStatus,
			NeedsReview:        tx.NeedsReview,
			IsSplitTransaction: tx.IsSplitTransaction,
			AccountID:          tx.AccountID,
			MonarchCreatedAt:   tx.CreatedAt,
			MonarchUpdatedAt:   tx.UpdatedAt,
		}); err != nil {
			logger.Error("upsert transaction failed", "tx_id", tx.ID, "error", err)
		} else {
			count++
		}
	}

	a.Store.SetConfig(ctx, "last_tx_sync_date", today)
	return count, nil
}

func (a *App) syncCategories(ctx context.Context, svc *monarch.Service) {
	cats, err := svc.ListCategories(ctx)
	if err != nil {
		logger.Error("category sync error", "error", err)
		return
	}
	for _, c := range cats {
		a.Store.UpsertCategory(ctx, LocalCategory{
			ID:        c.ID,
			Name:      c.Name,
			GroupName: c.GroupName,
			GroupID:   c.GroupID,
			GroupType: c.GroupType,
			SortOrder: c.Order,
			Icon:      c.Icon,
		})
	}
}

func (a *App) syncBudgets(ctx context.Context, svc *monarch.Service) {
	now := time.Now().UTC()
	startDate := now.AddDate(0, -1, 0).Format("2006-01-02")
	endDate := now.AddDate(0, 1, 0).Format("2006-01-02")

	budgets, err := svc.ListBudgets(ctx, startDate, endDate)
	if err != nil {
		logger.Error("budget sync error", "error", err)
		return
	}
	for _, b := range budgets {
		a.Store.UpsertBudget(ctx, LocalBudget{
			ID:           b.CategoryID + "_" + b.Month,
			CategoryID:   b.CategoryID,
			CategoryName: b.CategoryName,
			Month:        b.Month,
			Planned:      b.Planned,
			Actual:       b.Actual,
		})
	}
}

func (a *App) syncRecurring(ctx context.Context, svc *monarch.Service) {
	now := time.Now().UTC()
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 3, 0).Format("2006-01-02")

	recurring, err := svc.ListRecurring(ctx, startDate, endDate)
	if err != nil {
		logger.Error("recurring sync error", "error", err)
		return
	}
	for _, r := range recurring {
		a.Store.UpsertRecurring(ctx, LocalRecurring{
			ID:           r.ID,
			Merchant:     r.Merchant,
			Amount:       r.Amount,
			Frequency:    r.Frequency,
			NextDate:     r.NextDate,
			CategoryName: r.CategoryName,
			AccountID:    r.AccountID,
			AccountName:  r.AccountName,
		})
	}
}

func (a *App) syncInvestments(ctx context.Context, svc *monarch.Service) {
	portfolio, err := svc.GetInvestmentPortfolio(ctx)
	if err != nil {
		logger.Error("investment sync error", "error", err)
		return
	}
	securityIDs := make([]string, 0)
	for _, h := range portfolio.Holdings {
		a.Store.UpsertInvestment(ctx, LocalInvestment{
			ID:         h.ID,
			SecurityID: h.SecurityID,
			Ticker:     h.Ticker,
			Name:       h.Name,
			Quantity:   h.Quantity,
			Basis:      h.Basis,
			TotalValue: h.TotalValue,
			Price:      h.Price,
		})
		if h.SecurityID != "" {
			securityIDs = append(securityIDs, h.SecurityID)
		}
	}

	benchmarkIDs := []string{sp500SecurityID, bondsSecurityID}
	existing := make(map[string]bool)
	for _, id := range securityIDs {
		existing[id] = true
	}
	for _, bid := range benchmarkIDs {
		if !existing[bid] {
			securityIDs = append(securityIDs, bid)
		}
	}

	if len(securityIDs) > 0 {
		a.syncInvestmentPerformance(ctx, svc, securityIDs)
	}
}

func (a *App) syncInvestmentPerformance(ctx context.Context, svc *monarch.Service, securityIDs []string) {
	endDate := time.Now().UTC().Format("2006-01-02")
	startDate := time.Now().UTC().AddDate(-1, 0, 0).Format("2006-01-02")

	lastPerf, _ := a.Store.GetConfig(ctx, "last_perf_sync_date")
	if lastPerf != "" {
		t, err := time.Parse("2006-01-02", lastPerf)
		if err == nil {
			startDate = t.AddDate(0, 0, -7).Format("2006-01-02")
		}
	}

	perfs, err := svc.GetSecurityPerformance(ctx, securityIDs, startDate, endDate)
	if err != nil {
		logger.Error("investment performance sync error", "error", err)
		return
	}

	count := 0
	for _, perf := range perfs {
		for _, pt := range perf.Points {
			a.Store.UpsertInvestmentPerformance(ctx, LocalInvestmentPerformance{
				SecurityID:    perf.SecurityID,
				Ticker:        perf.Ticker,
				Name:          perf.Name,
				Date:          pt.Date,
				ReturnPercent: pt.ReturnPercent,
				Value:         pt.Value,
			})
			count++
		}
	}

	a.Store.SetConfig(ctx, "last_perf_sync_date", endDate)
	logger.Info("synced investment performance", "count", count)
}

func (a *App) syncSnapshots(ctx context.Context, svc *monarch.Service) {
	snapshots, err := svc.GetAggregateSnapshots(ctx, "", "", "")
	if err != nil {
		logger.Error("snapshot sync error", "error", err)
		return
	}
	for _, s := range snapshots {
		a.Store.UpsertSnapshot(ctx, LocalSnapshot{
			Date:    s.Date,
			Balance: s.Amount,
		})
	}
}

func (a *App) syncAssetLiabilitySnapshots(ctx context.Context, svc *monarch.Service) {
	startDate := "2020-01-01"
	snapshots, groups, err := svc.GetSnapshotsByAccountType(ctx, startDate, "month")
	if err != nil {
		logger.Error("asset/liability snapshot sync error", "error", err)
		return
	}

	assetTypes := make(map[string]bool)
	for _, g := range groups {
		if g.Group == "asset" {
			assetTypes[g.Name] = true
		}
	}

	monthData := make(map[string]*AssetLiabilitySnapshot)
	for _, s := range snapshots {
		if monthData[s.Month] == nil {
			monthData[s.Month] = &AssetLiabilitySnapshot{Month: s.Month}
		}
		if assetTypes[s.AccountType] {
			monthData[s.Month].Assets += s.Balance
			if s.AccountType == "brokerage" {
				monthData[s.Month].Investments += s.Balance
			}
		} else {
			monthData[s.Month].Liabilities += s.Balance
		}
	}

	for _, snap := range monthData {
		snap.NetWorth = snap.Assets + snap.Liabilities
		a.Store.UpsertAssetLiabilitySnapshot(ctx, *snap)
	}
	logger.Info("synced asset/liability monthly snapshots", "count", len(monthData))
}

func (a *App) syncCreditScores(ctx context.Context, svc *monarch.Service) {
	scores, err := svc.GetCreditHistory(ctx)
	if err != nil {
		logger.Error("credit score sync error", "error", err)
		return
	}
	for _, s := range scores {
		a.Store.UpsertCreditScore(ctx, LocalCreditScore{
			Date:  s.Date,
			Score: s.Score,
		})
	}
}
