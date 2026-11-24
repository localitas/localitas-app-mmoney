package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetRecurringQuery = queries.Get("recurring/list.graphql")

type RecurringTransaction struct {
	ID           string  `json:"id"`
	Merchant     string  `json:"merchant"`
	Amount       float64 `json:"amount"`
	Frequency    string  `json:"frequency"`
	NextDate     string  `json:"next_date"`
	CategoryName string  `json:"category_name"`
	AccountID    string  `json:"account_id"`
	AccountName  string  `json:"account_name"`
}

// recurringVariables are Web_GetUpcomingRecurringTransactionItems'
// variables, with no filters set, in key order as the map they replaced
// marshaled them.
type recurringVariables struct {
	EndDate   string   `json:"endDate"`
	Filters   struct{} `json:"filters"`
	StartDate string   `json:"startDate"`
}

func (s *Service) ListRecurring(ctx context.Context, startDate, endDate string) ([]RecurringTransaction, error) {
	var resp struct {
		RecurringTransactionItems []struct {
			Stream struct {
				ID        string  `json:"id"`
				Frequency string  `json:"frequency"`
				Amount    float64 `json:"amount"`
				Merchant  struct {
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"stream"`
			Date     string  `json:"date"`
			Amount   float64 `json:"amount"`
			Category struct {
				Name string `json:"name"`
			} `json:"category"`
			Account struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"account"`
		} `json:"recurringTransactionItems"`
	}

	req, err := monarchgql.NewRequest("Web_GetUpcomingRecurringTransactionItems", GetRecurringQuery, recurringVariables{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		return nil, err
	}
	if err := s.Client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	recurring := make([]RecurringTransaction, len(resp.RecurringTransactionItems))
	for i, r := range resp.RecurringTransactionItems {
		recurring[i] = RecurringTransaction{
			ID:           r.Stream.ID,
			Merchant:     r.Stream.Merchant.Name,
			Amount:       r.Amount,
			Frequency:    r.Stream.Frequency,
			NextDate:     r.Date,
			CategoryName: r.Category.Name,
			AccountID:    r.Account.ID,
			AccountName:  r.Account.DisplayName,
		}
	}
	return recurring, nil
}
