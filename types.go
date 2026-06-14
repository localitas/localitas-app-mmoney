package mmoney

import "time"

type LocalAccount struct {
	ID                string  `json:"id"`
	DisplayName       string  `json:"display_name"`
	AccountType       string  `json:"account_type"`
	AccountSubtype    string  `json:"account_subtype"`
	DisplayBalance    float64 `json:"display_balance"`
	CurrentBalance    float64 `json:"current_balance"`
	CreditLimit       float64 `json:"credit_limit"`
	IsHidden          bool    `json:"is_hidden"`
	IsAsset           bool    `json:"is_asset"`
	IsManual          bool    `json:"is_manual"`
	IsClosed          bool    `json:"is_closed"`
	IncludeInNetWorth bool    `json:"include_in_net_worth"`
	DataProvider      string  `json:"data_provider"`
	Icon              string  `json:"icon"`
	LogoURL           string  `json:"logo_url"`
	Mask              string  `json:"mask"`
	MonarchCreatedAt  string  `json:"monarch_created_at"`
	MonarchUpdatedAt  string  `json:"monarch_updated_at"`
	SyncedAt          int64   `json:"synced_at"`
}

type LocalTransaction struct {
	ID                 string  `json:"id"`
	Date               string  `json:"date"`
	Amount             float64 `json:"amount"`
	Merchant           string  `json:"merchant"`
	Category           string  `json:"category"`
	CategoryID         string  `json:"category_id"`
	CategoryGroupName  string  `json:"category_group_name"`
	CategoryGroupType  string  `json:"category_group_type"`
	Notes              string  `json:"notes"`
	TagsJSON           string  `json:"tags_json"`
	Pending            bool    `json:"pending"`
	HideFromReports    bool    `json:"hide_from_reports"`
	PlaidName          string  `json:"plaid_name"`
	IsRecurring        bool    `json:"is_recurring"`
	ReviewStatus       string  `json:"review_status"`
	NeedsReview        bool    `json:"needs_review"`
	IsSplitTransaction bool    `json:"is_split_transaction"`
	AccountID          string  `json:"account_id"`
	MonarchCreatedAt   string  `json:"monarch_created_at"`
	MonarchUpdatedAt   string  `json:"monarch_updated_at"`
	SyncedAt           int64   `json:"synced_at"`
}

type LocalCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
	GroupID   string `json:"group_id"`
	GroupType string `json:"group_type"`
	SortOrder int    `json:"sort_order"`
	Icon      string `json:"icon"`
	SyncedAt  int64  `json:"synced_at"`
}

type LocalBudget struct {
	ID           string  `json:"id"`
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Month        string  `json:"month"`
	Planned      float64 `json:"planned"`
	Actual       float64 `json:"actual"`
	SyncedAt     int64   `json:"synced_at"`
}

type LocalRecurring struct {
	ID           string  `json:"id"`
	Merchant     string  `json:"merchant"`
	Amount       float64 `json:"amount"`
	Frequency    string  `json:"frequency"`
	NextDate     string  `json:"next_date"`
	CategoryName string  `json:"category_name"`
	AccountID    string  `json:"account_id"`
	AccountName  string  `json:"account_name"`
	SyncedAt     int64   `json:"synced_at"`
}

type LocalInvestment struct {
	ID         string  `json:"id"`
	SecurityID string  `json:"security_id"`
	Ticker     string  `json:"ticker"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Basis      float64 `json:"basis"`
	TotalValue float64 `json:"total_value"`
	Price      float64 `json:"price"`
	SyncedAt   int64   `json:"synced_at"`
}

type LocalInvestmentPerformance struct {
	SecurityID    string  `json:"security_id"`
	Ticker        string  `json:"ticker"`
	Name          string  `json:"name"`
	Date          string  `json:"date"`
	ReturnPercent float64 `json:"return_percent"`
	Value         float64 `json:"value"`
}

type LocalSnapshot struct {
	Date    string  `json:"date"`
	Balance float64 `json:"balance"`
}

type LocalCreditScore struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

type SyncStatus struct {
	LastSyncAt       *time.Time `json:"last_sync_at"`
	Status           string     `json:"status"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	AccountCount     int        `json:"account_count"`
	TransactionCount int        `json:"transaction_count"`
}

type SyncLogEntry struct {
	ID                 int       `json:"id"`
	StartedAt          time.Time `json:"started_at"`
	CompletedAt        time.Time `json:"completed_at"`
	Status             string    `json:"status"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	AccountsSynced     int       `json:"accounts_synced"`
	TransactionsSynced int       `json:"transactions_synced"`
}

type CashflowMonth struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type CategoryTotal struct {
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	AbsAmount float64 `json:"abs_amount"`
}

type CashflowReport struct {
	Months            []CashflowMonth `json:"months"`
	TotalIncome       float64         `json:"total_income"`
	TotalExpense      float64         `json:"total_expense"`
	Savings           float64         `json:"savings"`
	SavingsRate       float64         `json:"savings_rate"`
	IncomeByCategory  []CategoryTotal `json:"income_by_category"`
	ExpenseByCategory []CategoryTotal `json:"expense_by_category"`
}

type FIREReport struct {
	AnnualExpenses         float64         `json:"annual_expenses"`
	InvestmentValue        float64         `json:"investment_value"`
	InvestmentGrowthDollar float64         `json:"investment_growth_dollar"`
	InvestmentGrowthYoY    float64         `json:"investment_growth_yoy"`
	ExpenseGrowthYoY       float64         `json:"expense_growth_yoy"`
	GrowthExceedsExpenses  bool            `json:"growth_exceeds_expenses"`
	Surplus                float64         `json:"surplus"`
	SavingsRate            float64         `json:"savings_rate"`
	YearsToFIRE            float64         `json:"years_to_fire"`
	MonthlyExpenses        []MonthlyAmount `json:"monthly_expenses"`
	MonthlyInvestments     []MonthlyAmount `json:"monthly_investments"`
}

type MonthlyAmount struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}

type AssetLiabilitySnapshot struct {
	Month       string  `json:"month"`
	Assets      float64 `json:"assets"`
	Liabilities float64 `json:"liabilities"`
	NetWorth    float64 `json:"net_worth"`
	Investments float64 `json:"investments"`
}

type NetWorthPoint struct {
	Date    string  `json:"date"`
	Balance float64 `json:"balance"`
}
