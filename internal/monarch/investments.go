package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetInvestmentPortfolioQuery = queries.Get("investments/portfolio.graphql")
var GetSecurityPerformanceQuery = queries.Get("investments/performance.graphql")

type InvestmentPortfolio struct {
	TotalValue         float64                 `json:"total_value"`
	TotalChangePercent float64                 `json:"total_change_percent"`
	TotalChangeDollars float64                 `json:"total_change_dollars"`
	Holdings           []InvestmentHoldingNode `json:"holdings"`
}

type InvestmentHoldingNode struct {
	ID         string  `json:"id"`
	SecurityID string  `json:"security_id"`
	Quantity   float64 `json:"quantity"`
	Basis      float64 `json:"basis"`
	TotalValue float64 `json:"total_value"`
	Ticker     string  `json:"ticker"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
}

type SecurityPerformancePoint struct {
	Date          string  `json:"date"`
	ReturnPercent float64 `json:"return_percent"`
	Value         float64 `json:"value"`
}

type SecurityPerformance struct {
	SecurityID string                     `json:"security_id"`
	Ticker     string                     `json:"ticker"`
	Name       string                     `json:"name"`
	Points     []SecurityPerformancePoint `json:"points"`
}

func (s *Service) GetInvestmentPortfolio(ctx context.Context) (*InvestmentPortfolio, error) {
	var resp struct {
		Portfolio struct {
			Performance struct {
				TotalValue         float64 `json:"totalValue"`
				TotalChangePercent float64 `json:"totalChangePercent"`
				TotalChangeDollars float64 `json:"totalChangeDollars"`
			} `json:"performance"`
			AggregateHoldings struct {
				Edges []struct {
					Node struct {
						ID         string  `json:"id"`
						Quantity   float64 `json:"quantity"`
						Basis      float64 `json:"basis"`
						TotalValue float64 `json:"totalValue"`
						Security   struct {
							ID           string  `json:"id"`
							Ticker       string  `json:"ticker"`
							Name         string  `json:"name"`
							CurrentPrice float64 `json:"currentPrice"`
						} `json:"security"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"aggregateHoldings"`
		} `json:"portfolio"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "Web_GetPortfolio",
		Query:         GetInvestmentPortfolioQuery,
		Variables:     map[string]interface{}{"portfolioInput": map[string]interface{}{}},
	}, &resp)
	if err != nil {
		return nil, err
	}

	portfolio := &InvestmentPortfolio{
		TotalValue:         resp.Portfolio.Performance.TotalValue,
		TotalChangePercent: resp.Portfolio.Performance.TotalChangePercent,
		TotalChangeDollars: resp.Portfolio.Performance.TotalChangeDollars,
		Holdings:           make([]InvestmentHoldingNode, 0, len(resp.Portfolio.AggregateHoldings.Edges)),
	}

	for _, edge := range resp.Portfolio.AggregateHoldings.Edges {
		node := edge.Node
		portfolio.Holdings = append(portfolio.Holdings, InvestmentHoldingNode{
			ID:         node.ID,
			SecurityID: node.Security.ID,
			Quantity:   node.Quantity,
			Basis:      node.Basis,
			TotalValue: node.TotalValue,
			Ticker:     node.Security.Ticker,
			Name:       node.Security.Name,
			Price:      node.Security.CurrentPrice,
		})
	}

	return portfolio, nil
}

func (s *Service) GetSecurityPerformance(ctx context.Context, securityIDs []string, startDate, endDate string) ([]SecurityPerformance, error) {
	var resp struct {
		SecurityHistoricalPerformance []struct {
			Security struct {
				ID     string `json:"id"`
				Ticker string `json:"ticker"`
				Name   string `json:"name"`
			} `json:"security"`
			HistoricalChart []struct {
				Date          string   `json:"date"`
				ReturnPercent float64  `json:"returnPercent"`
				Value         *float64 `json:"value"`
			} `json:"historicalChart"`
		} `json:"securityHistoricalPerformance"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "Web_GetInvestmentsHoldingDrawerHistoricalPerformance",
		Query:         GetSecurityPerformanceQuery,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"securityIds": securityIDs,
				"startDate":   startDate,
				"endDate":     endDate,
			},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]SecurityPerformance, 0, len(resp.SecurityHistoricalPerformance))
	for _, item := range resp.SecurityHistoricalPerformance {
		perf := SecurityPerformance{
			SecurityID: item.Security.ID,
			Ticker:     item.Security.Ticker,
			Name:       item.Security.Name,
			Points:     make([]SecurityPerformancePoint, 0, len(item.HistoricalChart)),
		}
		for _, pt := range item.HistoricalChart {
			val := 0.0
			if pt.Value != nil {
				val = *pt.Value
			}
			perf.Points = append(perf.Points, SecurityPerformancePoint{
				Date:          pt.Date,
				ReturnPercent: pt.ReturnPercent,
				Value:         val,
			})
		}
		out = append(out, perf)
	}
	return out, nil
}
