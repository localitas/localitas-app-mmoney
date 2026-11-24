package monarch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
)

type sentRequest struct {
	OperationName string          `json:"operationName"`
	Query         string          `json:"query"`
	Variables     json.RawMessage `json:"variables"`
}

// monarchAt is a Service talking to a stand-in Monarch that hands each
// GraphQL request it gets to the returned channel and answers no data.
func monarchAt(t *testing.T) (*Service, <-chan sentRequest) {
	t.Helper()
	sent := make(chan sentRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		var req sentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("request %s: %v", body, err)
		}
		sent <- req
		w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	return NewService(monarchgql.NewClient(srv.URL, "tok", 5*time.Second)), sent
}

// The variables each query sends Monarch are byte for byte what the
// untyped maps they replaced marshaled: keys sorted, empty filters as {},
// empty lists as [], unset optional filters left out.
func TestQueryVariables_Bytes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		call      func(s *Service) error
		operation string
		query     string
		want      string
	}{
		{"accounts", func(s *Service) error { _, err := s.ListAccounts(ctx); return err },
			"GetAccounts", GetAccountsQuery, `null`},
		{"snapshots by type", func(s *Service) error {
			_, _, err := s.GetSnapshotsByAccountType(ctx, "2026-01-01", "month")
			return err
		}, "GetSnapshotsByAccountType", GetSnapshotsByAccountTypeQuery, `{"startDate":"2026-01-01","timeframe":"month"}`},
		{"history, no dates", func(s *Service) error { _, err := s.GetAccountHistory(ctx, "", ""); return err },
			"GetAccountHistory", GetAccountHistoryQuery, `{"filters":{}}`},
		{"history, dates", func(s *Service) error { _, err := s.GetAccountHistory(ctx, "2026-01-01", "2026-02-01"); return err },
			"GetAccountHistory", GetAccountHistoryQuery, `{"filters":{"endDate":"2026-02-01","startDate":"2026-01-01"}}`},
		{"aggregate, no filters", func(s *Service) error { _, err := s.GetAggregateSnapshots(ctx, "", "", ""); return err },
			"GetAggregateSnapshots", GetAggregateSnapshotsQuery, `{"filters":{}}`},
		{"aggregate, filters", func(s *Service) error {
			_, err := s.GetAggregateSnapshots(ctx, "2026-01-01", "2026-02-01", "brokerage")
			return err
		}, "GetAggregateSnapshots", GetAggregateSnapshotsQuery, `{"filters":{"accountType":"brokerage","endDate":"2026-02-01","startDate":"2026-01-01"}}`},
		{"budgets", func(s *Service) error { _, err := s.ListBudgets(ctx, "2026-01-01", "2026-02-01"); return err },
			"GetJointPlanningData", GetBudgetsQuery, `{"endDate":"2026-02-01","startDate":"2026-01-01"}`},
		{"cashflow", func(s *Service) error { _, err := s.GetCashflowSummary(ctx, "2026-01-01", "2026-02-01"); return err },
			"GetCashflowSummary", GetCashflowSummaryQuery, `{"filters":{"accounts":[],"categories":[],"endDate":"2026-02-01","search":"","startDate":"2026-01-01","tags":[]}}`},
		{"cashflow, no dates", func(s *Service) error { _, err := s.GetCashflowSummary(ctx, "", ""); return err },
			"GetCashflowSummary", GetCashflowSummaryQuery, `{"filters":{"accounts":[],"categories":[],"endDate":"","search":"","startDate":"","tags":[]}}`},
		{"recurring", func(s *Service) error { _, err := s.ListRecurring(ctx, "2026-01-01", "2026-02-01"); return err },
			"Web_GetUpcomingRecurringTransactionItems", GetRecurringQuery, `{"endDate":"2026-02-01","filters":{},"startDate":"2026-01-01"}`},
		{"portfolio", func(s *Service) error { _, err := s.GetInvestmentPortfolio(ctx); return err },
			"Web_GetPortfolio", GetInvestmentPortfolioQuery, `{"portfolioInput":{}}`},
		{"security performance", func(s *Service) error {
			_, err := s.GetSecurityPerformance(ctx, []string{"s1", "s2"}, "2026-01-01", "2026-02-01")
			return err
		}, "Web_GetInvestmentsHoldingDrawerHistoricalPerformance", GetSecurityPerformanceQuery, `{"input":{"endDate":"2026-02-01","securityIds":["s1","s2"],"startDate":"2026-01-01"}}`},
		{"security performance, no ids", func(s *Service) error {
			_, err := s.GetSecurityPerformance(ctx, nil, "2026-01-01", "2026-02-01")
			return err
		}, "Web_GetInvestmentsHoldingDrawerHistoricalPerformance", GetSecurityPerformanceQuery, `{"input":{"endDate":"2026-02-01","securityIds":null,"startDate":"2026-01-01"}}`},
		{"transactions", func(s *Service) error {
			_, _, err := s.ListTransactions(ctx, ListTransactionsOptions{Limit: 100, Offset: 200, Search: "coffee", StartDate: "2026-01-01", EndDate: "2026-02-01"})
			return err
		}, "GetTransactionsList", GetTransactionsQuery, `{"filters":{"accounts":[],"categories":[],"endDate":"2026-02-01","search":"coffee","startDate":"2026-01-01","tags":[]},"limit":100,"offset":200}`},
		{"transactions, no dates", func(s *Service) error {
			_, _, err := s.ListTransactions(ctx, ListTransactionsOptions{Limit: 100})
			return err
		}, "GetTransactionsList", GetTransactionsQuery, `{"filters":{"accounts":[],"categories":[],"search":"","tags":[]},"limit":100,"offset":0}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, sent := monarchAt(t)
			if err := c.call(s); err != nil {
				t.Fatal(err)
			}
			got := <-sent
			if got.OperationName != c.operation || got.Query != c.query {
				t.Errorf("operation %q, query matches: %v", got.OperationName, got.Query == c.query)
			}
			if string(got.Variables) != c.want {
				t.Errorf("variables\n got %s\nwant %s", got.Variables, c.want)
			}
		})
	}
}
