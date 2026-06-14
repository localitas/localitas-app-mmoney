package mmoney

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestApp(t *testing.T) *App {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "mmoney-handler-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := sql.Open("sqlite3", tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	schema, err := os.ReadFile("migrations/20260613-000000-000-init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}

	perf, err := os.ReadFile("migrations/20260613-010000-000-investment-performance.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range splitSQL(string(perf)) {
		db.Exec(stmt)
	}

	al, _ := os.ReadFile("migrations/20260613-020000-000-asset-liability-snapshots.sql")
	db.Exec(string(al))

	store := NewStore(db)
	app := &App{Store: store, BasePath: "/"}
	return app
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func TestHandlerListAccounts(t *testing.T) {
	app := setupTestApp(t)
	ctx := context.Background()
	app.Store.UpsertAccount(ctx, LocalAccount{ID: "a1", DisplayName: "Checking", AccountType: "depository", DisplayBalance: 5000})
	app.Store.UpsertAccount(ctx, LocalAccount{ID: "a2", DisplayName: "Credit", AccountType: "credit", DisplayBalance: -1000})

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/accounts", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Accounts []LocalAccount `json:"accounts"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(resp.Accounts))
	}
}

func TestHandlerListTransactions(t *testing.T) {
	app := setupTestApp(t)
	ctx := context.Background()
	app.Store.UpsertTransaction(ctx, LocalTransaction{ID: "t1", Date: "2026-06-10", Amount: -25, Merchant: "Store", Category: "Food", TagsJSON: "[]"})
	app.Store.UpsertTransaction(ctx, LocalTransaction{ID: "t2", Date: "2026-06-11", Amount: 3000, Merchant: "Employer", Category: "Paycheck", TagsJSON: "[]"})

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/transactions?limit=10", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Transactions []LocalTransaction `json:"transactions"`
		Total        int                `json:"total"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
}

func TestHandlerSearch(t *testing.T) {
	app := setupTestApp(t)
	ctx := context.Background()
	app.Store.UpsertTransaction(ctx, LocalTransaction{ID: "t1", Date: "2026-06-10", Amount: -25, Merchant: "Starbucks", Category: "Coffee", TagsJSON: "[]"})
	app.Store.UpsertTransaction(ctx, LocalTransaction{ID: "t2", Date: "2026-06-10", Amount: -100, Merchant: "Amazon", Category: "Shopping", TagsJSON: "[]"})

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/search?q=Starbucks", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Transactions []LocalTransaction `json:"transactions"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Transactions) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Transactions))
	}
}

func TestHandlerSearchRequiresQ(t *testing.T) {
	app := setupTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/search", "")
	if rr.Code != 400 {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandlerConfig(t *testing.T) {
	app := setupTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "POST", "/api/config", `{"key":"vault_credential_id","value":"test-123"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	rr = doRequest(t, mux, "GET", "/api/config/vault_credential_id", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Value string `json:"value"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Value != "test-123" {
		t.Fatalf("expected test-123, got %s", resp.Value)
	}
}

func TestHandlerSyncStatus(t *testing.T) {
	app := setupTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/sync-status", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp SyncStatus
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Status != "never" {
		t.Fatalf("expected status never, got %s", resp.Status)
	}
}

func TestHandlerSyncHistory(t *testing.T) {
	app := setupTestApp(t)
	ctx := context.Background()
	app.Store.LogSync(ctx, 1000, "ok", "", 10, 500)

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/sync-history", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		History []SyncLogEntry `json:"history"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.History) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(resp.History))
	}
}

func TestHandlerCashflowReport(t *testing.T) {
	app := setupTestApp(t)
	ctx := context.Background()
	app.Store.UpsertTransaction(ctx, LocalTransaction{ID: "t1", Date: "2026-06-01", Amount: 5000, Merchant: "Employer", Category: "Paycheck", TagsJSON: "[]"})
	app.Store.UpsertTransaction(ctx, LocalTransaction{ID: "t2", Date: "2026-06-05", Amount: -42.50, Merchant: "Grocery", Category: "Food", TagsJSON: "[]"})

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/api/reports/cashflow?start=2026-06-01&end=2026-06-30", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp CashflowReport
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.TotalIncome != 5000 {
		t.Fatalf("expected income 5000, got %f", resp.TotalIncome)
	}
}

func TestHandlerSwagger(t *testing.T) {
	app := setupTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := doRequest(t, mux, "GET", "/swagger.json", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var doc APIDoc
	json.NewDecoder(rr.Body).Decode(&doc)
	if doc.AppName != "MMoney" {
		t.Fatalf("expected app name MMoney, got %s", doc.AppName)
	}
	if len(doc.Endpoints) < 15 {
		t.Fatalf("expected at least 15 endpoints, got %d", len(doc.Endpoints))
	}
}

func TestHandlerHealth(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health.json", nil)
	HandleHealth(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var health AppHealth
	json.NewDecoder(rr.Body).Decode(&health)
	if health.Name != "mmoney" {
		t.Fatalf("expected name mmoney, got %s", health.Name)
	}
	if health.Status != "healthy" {
		t.Fatalf("expected status healthy, got %s", health.Status)
	}
}
