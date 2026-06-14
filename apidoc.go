package mmoney

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type APIEndpoint struct {
	Method      string     `json:"method"`
	Path        string     `json:"path"`
	Summary     string     `json:"summary"`
	QueryParams []APIParam `json:"query_params,omitempty"`
	RequestBody *APIBody   `json:"request_body,omitempty"`
	Response    *APIBody   `json:"response,omitempty"`
}

type APIParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type APIBody struct {
	ContentType string `json:"content_type"`
	Example     string `json:"example"`
}

type APIFieldDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

type APIDoc struct {
	AppName     string        `json:"app_name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Keywords    []string      `json:"keywords,omitempty"`
	Fields      []APIFieldDoc `json:"fields,omitempty"`
	Endpoints   []APIEndpoint `json:"endpoints"`
}

var MMoneyAPIDoc = APIDoc{
	AppName:     "MMoney",
	Version:     "1.0.0",
	Description: "Personal finance dashboard synced from MMoney. Tracks accounts, transactions, budgets, investments, net worth, and credit scores locally.",
	Keywords:    []string{"finance", "money", "budget", "transactions", "accounts", "investments", "net worth", "credit score", "monarch", "banking"},
	Fields: []APIFieldDoc{
		{Name: "Vault Setup", Description: "Store Monarch Money credentials in the vault app", Example: "POST /apps/vault/api/credentials with:\n{\"name\": \"monarch-money\", \"data\": {\"email\": \"you@example.com\", \"password\": \"...\", \"mfa_secret\": \"BASE32SECRET\"}}\n\nThen configure: POST /api/config {\"key\": \"vault_credential_id\", \"value\": \"<public_id>\"}"},
		{Name: "Sync", Description: "Data syncs every 10 minutes via automation, or manually", Example: "POST /api/sync triggers a full sync\nGET /api/sync-status shows last sync result"},
		{Name: "Date Filters", Description: "Most list endpoints accept start/end query params", Example: "?start=2026-01-01&end=2026-06-30"},
	},
	Endpoints: []APIEndpoint{
		{
			Method:   "GET",
			Path:     "/api/accounts",
			Summary:  "List all synced financial accounts",
			Response: &APIBody{ContentType: "application/json", Example: `{"accounts":[{"id":"123","display_name":"Chase Checking","account_type":"depository","display_balance":5432.10}]}`},
		},
		{
			Method:  "GET",
			Path:    "/api/transactions",
			Summary: "List transactions with pagination and date filtering",
			QueryParams: []APIParam{
				{Name: "start", Type: "string", Description: "Start date YYYY-MM-DD"},
				{Name: "end", Type: "string", Description: "End date YYYY-MM-DD"},
				{Name: "limit", Type: "int", Description: "Page size (default 50)"},
				{Name: "offset", Type: "int", Description: "Offset for pagination"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"transactions":[{"id":"456","date":"2026-06-10","amount":-42.50,"merchant":"Grocery Store","category":"Food"}],"total":1234,"limit":50,"offset":0}`},
		},
		{
			Method:  "GET",
			Path:    "/api/search",
			Summary: "Full-text search across transactions",
			QueryParams: []APIParam{
				{Name: "q", Type: "string", Required: true, Description: "Search query"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"transactions":[{"id":"456","merchant":"Amazon","amount":-29.99}]}`},
		},
		{
			Method:   "GET",
			Path:     "/api/categories",
			Summary:  "List all transaction categories",
			Response: &APIBody{ContentType: "application/json", Example: `{"categories":[{"id":"789","name":"Food & Drink","group_name":"Expenses"}]}`},
		},
		{
			Method:  "GET",
			Path:    "/api/budgets",
			Summary: "List budget entries by month",
			QueryParams: []APIParam{
				{Name: "month", Type: "string", Description: "Month filter YYYY-MM"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"budgets":[{"category_name":"Food","month":"2026-06","planned":500,"actual":320}]}`},
		},
		{
			Method:   "GET",
			Path:     "/api/recurring",
			Summary:  "List recurring transactions",
			Response: &APIBody{ContentType: "application/json", Example: `{"recurring":[{"merchant":"Netflix","amount":-15.49,"frequency":"monthly"}]}`},
		},
		{
			Method:   "GET",
			Path:     "/api/investments",
			Summary:  "List investment holdings",
			Response: &APIBody{ContentType: "application/json", Example: `{"investments":[{"ticker":"VTI","name":"Vanguard Total Stock Market","quantity":50,"total_value":12500}]}`},
		},
		{
			Method:   "GET",
			Path:     "/api/credit",
			Summary:  "Credit score history",
			Response: &APIBody{ContentType: "application/json", Example: `{"credit_scores":[{"date":"2026-06-01","score":780}]}`},
		},
		{
			Method:  "GET",
			Path:    "/api/net-worth",
			Summary: "Net worth history from account snapshots",
			QueryParams: []APIParam{
				{Name: "start", Type: "string", Description: "Start date YYYY-MM-DD"},
				{Name: "end", Type: "string", Description: "End date YYYY-MM-DD"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"net_worth":[{"date":"2026-06-01","balance":125000}]}`},
		},
		{
			Method:  "GET",
			Path:    "/api/cashflow",
			Summary: "Cashflow summary computed from local transactions",
			QueryParams: []APIParam{
				{Name: "start", Type: "string", Description: "Start date (default: 1 month ago)"},
				{Name: "end", Type: "string", Description: "End date (default: today)"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"income":5000,"expense":-3200,"savings":1800,"savings_rate":36}`},
		},
		{
			Method:  "GET",
			Path:    "/api/reports/cashflow",
			Summary: "Cashflow report with monthly bars, totals, and category breakdowns",
			QueryParams: []APIParam{
				{Name: "start", Type: "string", Description: "Start date YYYY-MM-DD (default: 6 months ago)"},
				{Name: "end", Type: "string", Description: "End date YYYY-MM-DD (default: today)"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"months":[{"month":"2026-06","income":5000,"expense":-3200}],"total_income":30000,"total_expense":-19200,"savings":10800,"savings_rate":36,"income_by_category":[{"category":"Paycheck","amount":28000}],"expense_by_category":[{"category":"Food","amount":-4500,"abs_amount":4500}]}`},
		},
		{
			Method:   "GET",
			Path:     "/api/reports/fire",
			Summary:  "FIRE calculator — investment growth vs expenses, progress to financial independence",
			Response: &APIBody{ContentType: "application/json", Example: `{"annual_expenses":80000,"investment_value":1500000,"fire_number":2000000,"progress_percent":75,"investment_growth_yoy":12.5,"expense_growth_yoy":3.2,"savings_rate":45,"years_to_fire":4,"monthly_expenses":[{"month":"2026-06","amount":6500}],"monthly_investments":[{"month":"2026-06","amount":1500000}]}`},
		},
		{
			Method:  "GET",
			Path:    "/api/snapshots",
			Summary: "Raw account balance snapshots",
			QueryParams: []APIParam{
				{Name: "start", Type: "string", Description: "Start date YYYY-MM-DD"},
				{Name: "end", Type: "string", Description: "End date YYYY-MM-DD"},
			},
		},
		{
			Method:   "GET",
			Path:     "/api/snapshots/breakdown",
			Summary:  "Monthly assets vs liabilities breakdown over time",
			Response: &APIBody{ContentType: "application/json", Example: `{"snapshots":[{"month":"2026-06","assets":6500000,"liabilities":-300000,"net_worth":6200000}]}`},
		},
		{
			Method:  "GET",
			Path:    "/api/investment-performance",
			Summary: "Day-by-day investment performance with return percentage",
			QueryParams: []APIParam{
				{Name: "security_id", Type: "string", Required: true, Description: "Monarch security ID"},
				{Name: "start", Type: "string", Description: "Start date YYYY-MM-DD"},
				{Name: "end", Type: "string", Description: "End date YYYY-MM-DD"},
			},
			Response: &APIBody{ContentType: "application/json", Example: `{"performance":[{"security_id":"123","ticker":"VTI","date":"2026-06-01","return_percent":12.5,"value":15000}]}`},
		},
		{
			Method:   "GET",
			Path:     "/api/sync-status",
			Summary:  "Get last sync status and timestamp",
			Response: &APIBody{ContentType: "application/json", Example: `{"last_sync_at":"2026-06-13T10:00:00Z","status":"ok","account_count":5,"transaction_count":150}`},
		},
		{
			Method:   "GET",
			Path:     "/api/sync-history",
			Summary:  "List past sync runs with status and error details",
			Response: &APIBody{ContentType: "application/json", Example: `{"history":[{"id":1,"started_at":"2026-06-13T10:00:00Z","completed_at":"2026-06-13T10:00:12Z","status":"ok","accounts_synced":30,"transactions_synced":4745}]}`},
		},
		{
			Method:   "POST",
			Path:     "/api/sync",
			Summary:  "Trigger a manual sync from Monarch Money",
			Response: &APIBody{ContentType: "application/json", Example: `{"status":"ok"}`},
		},
		{
			Method:      "POST",
			Path:        "/api/config",
			Summary:     "Set a configuration value (e.g. vault_credential_id)",
			RequestBody: &APIBody{ContentType: "application/json", Example: `{"key":"vault_credential_id","value":"abc-123"}`},
		},
		{
			Method:  "GET",
			Path:    "/api/config/{key}",
			Summary: "Get a configuration value",
		},
	},
}

func HandleSwagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MMoneyAPIDoc)
}

func RenderDocsHTML(doc APIDoc) template.HTML {
	var sb strings.Builder
	if len(doc.Fields) > 0 {
		sb.WriteString(`<h3 style="font-size: 0.875rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-secondary); margin-bottom: 1rem;">Reference</h3><div class="accordion-list">`)
		for _, f := range doc.Fields {
			sb.WriteString(fmt.Sprintf(`<details class="glass-panel" style="border-radius: 0.5rem; margin-bottom: 0.5rem;"><summary style="padding: 0.75rem 1rem; cursor: pointer; font-weight: 500; color: var(--color-text-primary);">%s</summary><div style="padding: 0 1rem 0.75rem; font-size: 0.875rem; color: var(--color-text-secondary);"><p style="margin-bottom: 0.5rem;">%s</p><pre style="background: var(--color-bg-base); padding: 0.75rem; border-radius: 0.375rem; overflow-x: auto; font-size: 0.8125rem;">%s</pre></div></details>`, template.HTMLEscapeString(f.Name), template.HTMLEscapeString(f.Description), template.HTMLEscapeString(f.Example)))
		}
		sb.WriteString(`</div><hr style="border-color: var(--color-glass-border); margin: 1.5rem 0;">`)
	}
	sb.WriteString(`<h3 style="font-size: 0.875rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-secondary); margin-bottom: 1rem;">API Endpoints</h3><div class="accordion-list">`)
	for _, ep := range doc.Endpoints {
		title := fmt.Sprintf("%s %s — %s", ep.Method, ep.Path, ep.Summary)
		sb.WriteString(fmt.Sprintf(`<details class="glass-panel" style="border-radius: 0.5rem; margin-bottom: 0.5rem;"><summary style="padding: 0.75rem 1rem; cursor: pointer; font-weight: 500; color: var(--color-text-primary);">%s</summary><div style="padding: 0 1rem 0.75rem; font-size: 0.875rem; color: var(--color-text-secondary);">`, template.HTMLEscapeString(title)))
		var ex strings.Builder
		if ep.RequestBody != nil {
			ex.WriteString("# Request\n")
			ex.WriteString(prettyJSON(ep.RequestBody.Example))
			ex.WriteString("\n\n")
		}
		if ep.Response != nil {
			ex.WriteString("# Response\n")
			ex.WriteString(prettyJSON(ep.Response.Example))
		}
		if ex.Len() > 0 {
			sb.WriteString(fmt.Sprintf(`<pre style="background: var(--color-bg-base); padding: 0.75rem; border-radius: 0.375rem; overflow-x: auto; font-size: 0.8125rem;">%s</pre>`, template.HTMLEscapeString(ex.String())))
		}
		sb.WriteString(`</div></details>`)
	}
	sb.WriteString(`</div>`)
	return template.HTML(sb.String())
}

func prettyJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(b)
}
