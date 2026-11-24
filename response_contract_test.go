package mmoney

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func doRequestAccept(t *testing.T, mux *http.ServeMux, path, accept string) string {
	t.Helper()
	rr := doRequestWith(t, mux, http.MethodGet, path, "", accept)
	if rr.Code != http.StatusOK {
		t.Fatalf("%s: status %d: %s", path, rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}

// The API's answers are byte for byte what the untyped maps they replaced
// wrote, in JSON and in YAML.
func TestResponses_Bytes(t *testing.T) {
	app := setupTestApp(t)
	ctx := context.Background()
	app.Store.UpsertTransaction(ctx, "admin-user", LocalTransaction{ID: "t1", Date: "2026-06-10", Amount: -25, Merchant: "Store", Category: "Food", TagsJSON: "[]"})
	app.Store.UpsertTransaction(ctx, "admin-user", LocalTransaction{ID: "t2", Date: "2026-06-11", Amount: 3000, Merchant: "Employer", Category: "Paycheck", TagsJSON: "[]"})
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	if rr := doRequest(t, mux, "POST", "/api/config", `{"key":"vault_credential_id","value":"test-123"}`); rr.Body.String() != `{"key":"vault_credential_id","value":"test-123"}`+"\n" {
		t.Errorf("POST /api/config: got %s", rr.Body.String())
	}

	cases := []struct {
		path, accept, want string
	}{
		{"/api/accounts", "", `{"accounts":[]}` + "\n"},
		{"/api/transactions?limit=1&offset=5", "", `{"limit":1,"offset":5,"total":2,"transactions":[]}` + "\n"},
		{"/api/transactions?limit=1&offset=5", "application/yaml", "limit: 1\noffset: 5\ntotal: 2\ntransactions: []\n"},
		{"/api/transactions?limit=1&offset=5", "text/markdown", "| field | value |\n|---|---|\n| limit | 1 |\n| offset | 5 |\n| total | 2 |\n| transactions | [] |\n"},
		{"/api/categories", "", `{"categories":[]}` + "\n"},
		{"/api/budgets", "", `{"budgets":[]}` + "\n"},
		{"/api/recurring", "", `{"recurring":[]}` + "\n"},
		{"/api/investments", "", `{"investments":[]}` + "\n"},
		{"/api/credit", "", `{"credit_scores":[]}` + "\n"},
		{"/api/investment-performance?security_id=s1", "", `{"performance":[]}` + "\n"},
		{"/api/snapshots", "", `{"snapshots":[]}` + "\n"},
		{"/api/snapshots/breakdown", "", `{"snapshots":[]}` + "\n"},
		{"/api/net-worth", "", `{"net_worth":[]}` + "\n"},
		{"/api/cashflow?start=2026-06-01&end=2026-06-30", "", `{"end":"2026-06-30","expense":-25,"income":3000,"savings":2975,"savings_rate":99.16666666666667,"start":"2026-06-01"}` + "\n"},
		{"/api/cashflow?start=2026-06-01&end=2026-06-30", "application/yaml", "end: \"2026-06-30\"\nexpense: -25\nincome: 3000\nsavings: 2975\nsavings_rate: 99.16666666666667\nstart: \"2026-06-01\"\n"},
		{"/api/cashflow?start=2026-06-01&end=2026-06-30", "text/markdown", "| field | value |\n|---|---|\n| end | 2026-06-30 |\n| expense | -25 |\n| income | 3000 |\n| savings | 2975 |\n| savings_rate | 99.16666666666667 |\n| start | 2026-06-01 |\n"},
		{"/api/search?q=nomatch", "", `{"transactions":[]}` + "\n"},
		{"/api/sync-history", "", `{"history":[]}` + "\n"},
		{"/api/sync-history", "application/yaml", "history: []\n"},
		{"/api/config/vault_credential_id", "", `{"key":"vault_credential_id","value":"test-123"}` + "\n"},
		{"/api/config/vault_credential_id", "application/yaml", "key: vault_credential_id\nvalue: test-123\n"},
	}
	for _, c := range cases {
		if got := doRequestAccept(t, mux, c.path, c.accept); got != c.want {
			t.Errorf("%s (%s):\n got %q\nwant %q", c.path, c.accept, got, c.want)
		}
	}
}

func TestStatusResponse_Bytes(t *testing.T) {
	b, err := json.Marshal(statusResponse{Status: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"status":"ok"}` {
		t.Errorf("got %s", b)
	}
}
