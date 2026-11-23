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

// dateRangeVariables are GetJointPlanningData's variables, in key order as
// the map they replaced marshaled them.
type dateRangeVariables struct {
	EndDate   string `json:"endDate"`
	StartDate string `json:"startDate"`
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

	req, err := monarchgql.NewRequest("GetJointPlanningData", GetBudgetsQuery, dateRangeVariables{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		return nil, err
	}
	if err := s.Client.Do(ctx, req, &resp); err != nil {
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
