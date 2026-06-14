package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetCreditHistoryQuery = queries.Get("credit/history.graphql")

type CreditRecord struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

func (s *Service) GetCreditHistory(ctx context.Context) ([]CreditRecord, error) {
	var resp struct {
		CreditScoreSnapshots []struct {
			ReportedDate string `json:"reportedDate"`
			Score        int    `json:"score"`
		} `json:"creditScoreSnapshots"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetCreditScoreSnapshots",
		Query:         GetCreditHistoryQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	history := make([]CreditRecord, len(resp.CreditScoreSnapshots))
	for i, r := range resp.CreditScoreSnapshots {
		history[i] = CreditRecord{
			Date:  r.ReportedDate,
			Score: r.Score,
		}
	}
	return history, nil
}
