package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetBudgetsQuery = queries.Get("budgets/list.graphql")

type Budget struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Month        string  `json:"month"`
	Planned      float64 `json:"planned"`
	Actual       float64 `json:"actual"`
}

func (s *Service) ListBudgets(ctx context.Context, startDate, endDate string) ([]Budget, error) {
	var resp struct {
		BudgetData struct {
			MonthlyAmountsByCategory []struct {
				Category struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
				MonthlyAmounts []struct {
					Month                 string  `json:"month"`
					PlannedCashFlowAmount float64 `json:"plannedCashFlowAmount"`
					ActualAmount          float64 `json:"actualAmount"`
				} `json:"monthlyAmounts"`
			} `json:"monthlyAmountsByCategory"`
		} `json:"budgetData"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetJointPlanningData",
		Query:         GetBudgetsQuery,
		Variables: map[string]interface{}{
			"startDate": startDate,
			"endDate":   endDate,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	budgets := make([]Budget, 0)
	for _, cat := range resp.BudgetData.MonthlyAmountsByCategory {
		for _, m := range cat.MonthlyAmounts {
			budgets = append(budgets, Budget{
				CategoryID:   cat.Category.ID,
				CategoryName: cat.Category.Name,
				Month:        m.Month,
				Planned:      m.PlannedCashFlowAmount,
				Actual:       m.ActualAmount,
			})
		}
	}
	return budgets, nil
}
