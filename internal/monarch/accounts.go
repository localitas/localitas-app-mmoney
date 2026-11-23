package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetAccountsQuery = queries.Get("accounts/list.graphql")
var GetAccountQuery = queries.Get("accounts/show.graphql")
var GetAccountHistoryQuery = queries.Get("accounts/history.graphql")
var GetAggregateSnapshotsQuery = queries.Get("accounts/aggregate_snapshots.graphql")
var GetSnapshotsByAccountTypeQuery = queries.Get("accounts/snapshots_by_type.graphql")

// The variables of the account queries. Fields are in key order, as the
// maps these replaced marshaled them.

type snapshotsByAccountTypeVariables struct {
	StartDate string `json:"startDate"`
	Timeframe string `json:"timeframe"`
}

// snapshotFilters are GetAccountHistory's and GetAggregateSnapshots'
// filters; one not set is left out.
type snapshotFilters struct {
	AccountType string `json:"accountType,omitempty"`
	EndDate     string `json:"endDate,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
}

type snapshotFiltersVariables struct {
	Filters snapshotFilters `json:"filters"`
}

type AccountTypeSnapshot struct {
	AccountType string  `json:"account_type"`
	Month       string  `json:"month"`
	Balance     float64 `json:"balance"`
}

type AccountTypeGroup struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

func (s *Service) GetSnapshotsByAccountType(ctx context.Context, startDate, timeframe string) ([]AccountTypeSnapshot, []AccountTypeGroup, error) {
	var resp struct {
		SnapshotsByAccountType []struct {
			AccountType string  `json:"accountType"`
			Month       string  `json:"month"`
			Balance     float64 `json:"balance"`
		} `json:"snapshotsByAccountType"`
		AccountTypes []struct {
			Name  string `json:"name"`
			Group string `json:"group"`
		} `json:"accountTypes"`
	}

	req, err := monarchgql.NewRequest("GetSnapshotsByAccountType", GetSnapshotsByAccountTypeQuery, snapshotsByAccountTypeVariables{
		StartDate: startDate,
		Timeframe: timeframe,
	})
	if err != nil {
		return nil, nil, err
	}
	if err := s.Client.Do(ctx, req, &resp); err != nil {
		return nil, nil, err
	}

	snapshots := make([]AccountTypeSnapshot, len(resp.SnapshotsByAccountType))
	for i, s := range resp.SnapshotsByAccountType {
		snapshots[i] = AccountTypeSnapshot{
			AccountType: s.AccountType,
			Month:       s.Month,
			Balance:     s.Balance,
		}
	}

	groups := make([]AccountTypeGroup, len(resp.AccountTypes))
	for i, g := range resp.AccountTypes {
		groups[i] = AccountTypeGroup{Name: g.Name, Group: g.Group}
	}

	return snapshots, groups, nil
}

type Account struct {
	ID                              string  `json:"id"`
	DisplayName                     string  `json:"display_name"`
	AccountType                     string  `json:"account_type"`
	AccountSubtype                  string  `json:"account_subtype"`
	DisplayBalance                  float64 `json:"display_balance"`
	CurrentBalance                  float64 `json:"current_balance"`
	Limit                           float64 `json:"limit"`
	UpdatedAt                       string  `json:"updated_at"`
	DisplayLastUpdatedAt            string  `json:"display_last_updated_at"`
	DeactivatedAt                   string  `json:"deactivated_at"`
	IsHidden                        bool    `json:"is_hidden"`
	IsAsset                         bool    `json:"is_asset"`
	Mask                            string  `json:"mask"`
	CreatedAt                       string  `json:"created_at"`
	IncludeInNetWorth               bool    `json:"include_in_net_worth"`
	HideFromList                    bool    `json:"hide_from_list"`
	HideTransactionsFromReports     bool    `json:"hide_transactions_from_reports"`
	IncludeBalanceInNetWorth        bool    `json:"include_balance_in_net_worth"`
	DataProvider                    string  `json:"data_provider"`
	IsManual                        bool    `json:"is_manual"`
	TransactionsCount               int     `json:"transactions_count"`
	HoldingsCount                   int     `json:"holdings_count"`
	ManualInvestmentsTrackingMethod string  `json:"manual_investments_tracking_method"`
	Order                           int     `json:"order"`
	Icon                            string  `json:"icon"`
	LogoURL                         string  `json:"logo_url"`
	IsClosed                        bool    `json:"is_closed"`
}

type HistoryRecord struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	var resp struct {
		Accounts []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			AccountType struct {
				Name string `json:"name"`
			} `json:"type"`
			Subtype struct {
				Name string `json:"name"`
			} `json:"subtype"`
			DisplayBalance                  float64 `json:"displayBalance"`
			CurrentBalance                  float64 `json:"currentBalance"`
			Limit                           float64 `json:"limit"`
			UpdatedAt                       string  `json:"updatedAt"`
			DisplayLastUpdatedAt            string  `json:"displayLastUpdatedAt"`
			DeactivatedAt                   string  `json:"deactivatedAt"`
			IsHidden                        bool    `json:"isHidden"`
			IsAsset                         bool    `json:"isAsset"`
			Mask                            string  `json:"mask"`
			CreatedAt                       string  `json:"createdAt"`
			IncludeInNetWorth               bool    `json:"includeInNetWorth"`
			HideFromList                    bool    `json:"hideFromList"`
			HideTransactionsFromReports     bool    `json:"hideTransactionsFromReports"`
			IncludeBalanceInNetWorth        bool    `json:"includeBalanceInNetWorth"`
			DataProvider                    string  `json:"dataProvider"`
			IsManual                        bool    `json:"isManual"`
			TransactionsCount               int     `json:"transactionsCount"`
			HoldingsCount                   int     `json:"holdingsCount"`
			ManualInvestmentsTrackingMethod string  `json:"manualInvestmentsTrackingMethod"`
			Order                           int     `json:"order"`
			Icon                            string  `json:"icon"`
			LogoURL                         string  `json:"logoUrl"`
			IsClosed                        bool    `json:"isClosed"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetAccounts",
		Query:         GetAccountsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	accounts := make([]Account, len(resp.Accounts))
	for i, a := range resp.Accounts {
		accounts[i] = Account{
			ID:                              a.ID,
			DisplayName:                     a.DisplayName,
			AccountType:                     a.AccountType.Name,
			AccountSubtype:                  a.Subtype.Name,
			DisplayBalance:                  a.DisplayBalance,
			CurrentBalance:                  a.CurrentBalance,
			Limit:                           a.Limit,
			UpdatedAt:                       a.UpdatedAt,
			DisplayLastUpdatedAt:            a.DisplayLastUpdatedAt,
			DeactivatedAt:                   a.DeactivatedAt,
			IsHidden:                        a.IsHidden,
			IsAsset:                         a.IsAsset,
			Mask:                            a.Mask,
			CreatedAt:                       a.CreatedAt,
			IncludeInNetWorth:               a.IncludeInNetWorth,
			HideFromList:                    a.HideFromList,
			HideTransactionsFromReports:     a.HideTransactionsFromReports,
			IncludeBalanceInNetWorth:        a.IncludeBalanceInNetWorth,
			DataProvider:                    a.DataProvider,
			IsManual:                        a.IsManual,
			TransactionsCount:               a.TransactionsCount,
			HoldingsCount:                   a.HoldingsCount,
			ManualInvestmentsTrackingMethod: a.ManualInvestmentsTrackingMethod,
			Order:                           a.Order,
			Icon:                            a.Icon,
			LogoURL:                         a.LogoURL,
			IsClosed:                        a.IsClosed,
		}
	}

	return accounts, nil
}

func (s *Service) GetAccountHistory(ctx context.Context, startDate, endDate string) ([]HistoryRecord, error) {
	var resp struct {
		AggregateSnapshots []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"aggregateSnapshots"`
	}

	req, err := monarchgql.NewRequest("GetAccountHistory", GetAccountHistoryQuery, snapshotFiltersVariables{
		Filters: snapshotFilters{StartDate: startDate, EndDate: endDate},
	})
	if err != nil {
		return nil, err
	}
	if err := s.Client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	history := make([]HistoryRecord, len(resp.AggregateSnapshots))
	for i, r := range resp.AggregateSnapshots {
		history[i] = HistoryRecord{
			Date:   r.Date,
			Amount: r.Balance,
		}
	}

	return history, nil
}

func (s *Service) GetAggregateSnapshots(ctx context.Context, startDate, endDate, accountType string) ([]HistoryRecord, error) {
	var resp struct {
		AggregateSnapshots []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"aggregateSnapshots"`
	}

	req, err := monarchgql.NewRequest("GetAggregateSnapshots", GetAggregateSnapshotsQuery, snapshotFiltersVariables{
		Filters: snapshotFilters{StartDate: startDate, EndDate: endDate, AccountType: accountType},
	})
	if err != nil {
		return nil, err
	}
	if err := s.Client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	out := make([]HistoryRecord, len(resp.AggregateSnapshots))
	for i, r := range resp.AggregateSnapshots {
		out[i] = HistoryRecord{Date: r.Date, Amount: r.Balance}
	}
	return out, nil
}
