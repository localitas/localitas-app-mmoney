package mmoney

// The API's answers. Each struct's fields are in key order and carry json
// and yaml tags, so JSON, YAML and markdown answers are byte for byte what
// the maps they replaced wrote.

type accountsResponse struct {
	Accounts []LocalAccount `json:"accounts" yaml:"accounts"`
}

type transactionsResponse struct {
	Limit        int                `json:"limit" yaml:"limit"`
	Offset       int                `json:"offset" yaml:"offset"`
	Total        int                `json:"total" yaml:"total"`
	Transactions []LocalTransaction `json:"transactions" yaml:"transactions"`
}

type categoriesResponse struct {
	Categories []LocalCategory `json:"categories" yaml:"categories"`
}

type budgetsResponse struct {
	Budgets []LocalBudget `json:"budgets" yaml:"budgets"`
}

type recurringResponse struct {
	Recurring []LocalRecurring `json:"recurring" yaml:"recurring"`
}

type investmentsResponse struct {
	Investments []LocalInvestment `json:"investments" yaml:"investments"`
}

type creditScoresResponse struct {
	CreditScores []LocalCreditScore `json:"credit_scores" yaml:"credit_scores"`
}

type performanceResponse struct {
	Performance []LocalInvestmentPerformance `json:"performance" yaml:"performance"`
}

type assetLiabilitySnapshotsResponse struct {
	Snapshots []AssetLiabilitySnapshot `json:"snapshots" yaml:"snapshots"`
}

type snapshotsResponse struct {
	Snapshots []LocalSnapshot `json:"snapshots" yaml:"snapshots"`
}

type netWorthResponse struct {
	NetWorth []NetWorthPoint `json:"net_worth" yaml:"net_worth"`
}

type cashflowResponse struct {
	End         string  `json:"end" yaml:"end"`
	Expense     float64 `json:"expense" yaml:"expense"`
	Income      float64 `json:"income" yaml:"income"`
	Savings     float64 `json:"savings" yaml:"savings"`
	SavingsRate float64 `json:"savings_rate" yaml:"savings_rate"`
	Start       string  `json:"start" yaml:"start"`
}

type searchResponse struct {
	Transactions []LocalTransaction `json:"transactions" yaml:"transactions"`
}

type statusResponse struct {
	Status string `json:"status" yaml:"status"`
}

type syncHistoryResponse struct {
	History []SyncLogEntry `json:"history" yaml:"history"`
}

type configResponse struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}
