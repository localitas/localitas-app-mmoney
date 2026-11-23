package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetCashflowSummaryQuery = queries.Get("cashflow/summary.graphql")

type CashflowSummary struct {
	Income      float64 `json:"income"`
	Expense     float64 `json:"expense"`
	Savings     float64 `json:"savings"`
	SavingsRate float64 `json:"savings_rate"`
}

// cashflowFilters are GetCashflowSummary's filters, every one always sent,
// in key order as the map they replaced marshaled them.
type cashflowFilters struct {
	Accounts   []string `json:"accounts"`
	Categories []string `json:"categories"`
	EndDate    string   `json:"endDate"`
	Search     string   `json:"search"`
	StartDate  string   `json:"startDate"`
	Tags       []string `json:"tags"`
}

type cashflowSummaryVariables struct {
	Filters cashflowFilters `json:"filters"`
}

func (s *Service) GetCashflowSummary(ctx context.Context, startDate, endDate string) (*CashflowSummary, error) {
	var resp struct {
		Aggregates []struct {
			Summary struct {
				SumIncome   float64 `json:"sumIncome"`
				SumExpense  float64 `json:"sumExpense"`
				Savings     float64 `json:"savings"`
				SavingsRate float64 `json:"savingsRate"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	req, err := monarchgql.NewRequest("GetCashflowSummary", GetCashflowSummaryQuery, cashflowSummaryVariables{
		Filters: cashflowFilters{
			Accounts:   []string{},
			Categories: []string{},
			EndDate:    endDate,
			StartDate:  startDate,
			Tags:       []string{},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := s.Client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	if len(resp.Aggregates) == 0 {
		return &CashflowSummary{}, nil
	}

	return &CashflowSummary{
		Income:      resp.Aggregates[0].Summary.SumIncome,
		Expense:     resp.Aggregates[0].Summary.SumExpense,
		Savings:     resp.Aggregates[0].Summary.Savings,
		SavingsRate: resp.Aggregates[0].Summary.SavingsRate,
	}, nil
}
