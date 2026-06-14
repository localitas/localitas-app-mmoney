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

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetCashflowSummary",
		Query:         GetCashflowSummaryQuery,
		Variables: map[string]interface{}{
			"filters": map[string]interface{}{
				"startDate":  startDate,
				"endDate":    endDate,
				"search":     "",
				"categories": []string{},
				"accounts":   []string{},
				"tags":       []string{},
			},
		},
	}, &resp)
	if err != nil {
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
