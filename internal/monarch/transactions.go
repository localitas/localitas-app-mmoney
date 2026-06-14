package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetTransactionsQuery = queries.Get("transactions/list.graphql")

type Transaction struct {
	ID                      string  `json:"id"`
	Date                    string  `json:"date"`
	Amount                  float64 `json:"amount"`
	Merchant                string  `json:"merchant"`
	Category                string  `json:"category"`
	CategoryID              string  `json:"category_id"`
	CategoryGroupName       string  `json:"category_group_name"`
	CategoryGroupType       string  `json:"category_group_type"`
	Notes                   string  `json:"notes"`
	Tags                    []Tag   `json:"tags"`
	Pending                 bool    `json:"pending"`
	HideFromReports         bool    `json:"hide_from_reports"`
	PlaidName               string  `json:"plaid_name"`
	DataProviderDescription string  `json:"data_provider_description"`
	IsRecurring             bool    `json:"is_recurring"`
	ReviewStatus            string  `json:"review_status"`
	NeedsReview             bool    `json:"needs_review"`
	IsSplitTransaction      bool    `json:"is_split_transaction"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
	AccountID               string  `json:"account_id"`
}

type ListTransactionsOptions struct {
	Limit     int
	Offset    int
	Search    string
	StartDate string
	EndDate   string
}

func (s *Service) ListTransactions(ctx context.Context, opts ListTransactionsOptions) ([]Transaction, int, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	var resp struct {
		AllTransactions struct {
			Results []struct {
				ID                      string  `json:"id"`
				Date                    string  `json:"date"`
				Amount                  float64 `json:"amount"`
				Pending                 bool    `json:"pending"`
				HideFromReports         bool    `json:"hideFromReports"`
				DataProviderDescription string  `json:"dataProviderDescription"`
				PlaidName               string  `json:"plaidName"`
				Notes                   string  `json:"notes"`
				IsRecurring             bool    `json:"isRecurring"`
				ReviewStatus            string  `json:"reviewStatus"`
				NeedsReview             bool    `json:"needsReview"`
				IsSplitTransaction      bool    `json:"isSplitTransaction"`
				CreatedAt               string  `json:"createdAt"`
				UpdatedAt               string  `json:"updatedAt"`
				Category                struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Group struct {
						Name string `json:"name"`
						Type string `json:"type"`
					} `json:"group"`
				} `json:"category"`
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
				Account struct {
					ID string `json:"id"`
				} `json:"account"`
				Tags []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Color string `json:"color"`
				} `json:"tags"`
			} `json:"results"`
			TotalCount int `json:"totalCount"`
		} `json:"allTransactions"`
	}

	filters := map[string]interface{}{
		"search":     opts.Search,
		"categories": []string{},
		"accounts":   []string{},
		"tags":       []string{},
	}
	if opts.StartDate != "" {
		filters["startDate"] = opts.StartDate
	}
	if opts.EndDate != "" {
		filters["endDate"] = opts.EndDate
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetTransactionsList",
		Query:         GetTransactionsQuery,
		Variables: map[string]interface{}{
			"offset":  opts.Offset,
			"limit":   opts.Limit,
			"filters": filters,
		},
	}, &resp)
	if err != nil {
		return nil, 0, err
	}

	txs := make([]Transaction, len(resp.AllTransactions.Results))
	for i, r := range resp.AllTransactions.Results {
		tags := make([]Tag, len(r.Tags))
		for j, t := range r.Tags {
			tags[j] = Tag{ID: t.ID, Name: t.Name, Color: t.Color}
		}
		txs[i] = Transaction{
			ID:                      r.ID,
			Date:                    r.Date,
			Amount:                  r.Amount,
			Merchant:                r.Merchant.Name,
			Category:                r.Category.Name,
			CategoryID:              r.Category.ID,
			CategoryGroupName:       r.Category.Group.Name,
			CategoryGroupType:       r.Category.Group.Type,
			Notes:                   r.Notes,
			Tags:                    tags,
			Pending:                 r.Pending,
			HideFromReports:         r.HideFromReports,
			PlaidName:               r.PlaidName,
			DataProviderDescription: r.DataProviderDescription,
			IsRecurring:             r.IsRecurring,
			ReviewStatus:            r.ReviewStatus,
			NeedsReview:             r.NeedsReview,
			IsSplitTransaction:      r.IsSplitTransaction,
			CreatedAt:               r.CreatedAt,
			UpdatedAt:               r.UpdatedAt,
			AccountID:               r.Account.ID,
		}
	}

	return txs, resp.AllTransactions.TotalCount, nil
}

func (s *Service) ListAllTransactions(ctx context.Context, opts ListTransactionsOptions) ([]Transaction, error) {
	if opts.Limit <= 0 {
		opts.Limit = 1000
	}
	all := make([]Transaction, 0, opts.Limit)
	for {
		page, total, err := s.ListTransactions(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || opts.Offset+len(page) >= total {
			break
		}
		opts.Offset += len(page)
	}
	return all, nil
}
