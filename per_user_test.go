package mmoney

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	client "github.com/localitas/localitas-go"
	_ "github.com/mattn/go-sqlite3"
)

const userIDMigration = "20261018-032240-000-mmoney_user_id.sql"

// openMigratedDB is a fresh mmoney database with every migration applied
// in order, except the one named skip.
func openMigratedDB(t *testing.T, skip string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "mmoney.sqlitedb")+"?_synchronous=OFF&_journal_mode=MEMORY")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	files, _ := filepath.Glob("_migrations/*.sql")
	for _, f := range files {
		if filepath.Base(f) != skip {
			execFile(t, db, f)
		}
	}
	return db
}

func execFile(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	sqlText, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(sqlText)); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// standInCore answers the calls mmoney makes to core: the user directory,
// read with the app's own token, and vault credentials, which it serves
// only to the user who owns them, as core does.
type standInCore struct {
	*httptest.Server
	users  []client.UserSummary
	owners map[string]string

	mu sync.Mutex
	// secretReads lists "user:credential" for every secrets read, the user
	// being whoever the request's identity token names.
	secretReads []string
	// forUser lists "user:permission" for every token core minted for the
	// app's background work.
	forUser []string
}

func newStandInCore(t *testing.T, owners map[string]string, users ...string) *standInCore {
	t.Helper()
	c := &standInCore{owners: owners}
	for _, u := range users {
		c.users = append(c.users, client.UserSummary{ID: u, Email: u + "@test.local", Name: u})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
		if client.TokenFromRequest(r) != testAPIToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(struct {
			Users []client.UserSummary `json:"users"`
		}{c.users})
	})
	// Core mints a user's token only for its own system token, capped at
	// write; the app cannot sign one itself.
	mux.HandleFunc("POST /api/auth/identity/for-user", func(w http.ResponseWriter, r *http.Request) {
		if client.TokenFromRequest(r) != testAPIToken {
			http.Error(w, "only core's system token may mint a token for another user", http.StatusForbidden)
			return
		}
		var req struct {
			UserID     string `json:"user_id"`
			Permission string `json:"permission"`
			TTLSeconds int    `json:"ttl_seconds"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Permission == "admin" || req.TTLSeconds > 900 {
			http.Error(w, "permission must be read or write, ttl at most 15 minutes", http.StatusBadRequest)
			return
		}
		c.mu.Lock()
		c.forUser = append(c.forUser, req.UserID+":"+req.Permission)
		c.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"token": signer.Token(client.TokenInfo{UserID: req.UserID, Email: req.UserID + "@test.local", Permission: req.Permission})})
	})
	mux.HandleFunc("GET /apps/vault/api/credentials/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !c.readable(r, id) {
			http.Error(w, "credential not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(client.VaultCredentialSummary{PublicID: id, Name: "monarch-money"})
	})
	mux.HandleFunc("GET /apps/vault/api/credentials/{id}/secrets", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		c.mu.Lock()
		c.secretReads = append(c.secretReads, c.caller(r)+":"+id)
		c.mu.Unlock()
		if !c.readable(r, id) {
			http.Error(w, "credential not found", http.StatusNotFound)
			return
		}
		// A sync that got this far would sign in to Monarch; stop it here.
		http.Error(w, "vault sealed", http.StatusServiceUnavailable)
	})
	c.Server = httptest.NewServer(mux)
	t.Cleanup(c.Close)
	return c
}

// caller is the user an identity token names, as core resolves it: a
// token core cannot verify, or without an email, names nobody.
func (c *standInCore) caller(r *http.Request) string {
	claims, err := client.ParseBearerToken(client.TokenFromRequest(r))
	if err != nil || claims.Email == "" {
		return ""
	}
	for _, u := range c.users {
		if u.Email == claims.Email {
			return u.ID
		}
	}
	return ""
}

func (c *standInCore) readable(r *http.Request, id string) bool {
	owner := c.owners[id]
	return owner != "" && owner == c.caller(r)
}

func (c *standInCore) reads() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := slices.Clone(c.secretReads)
	slices.Sort(out)
	return out
}

func (c *standInCore) mintedForUsers() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := slices.Clone(c.forUser)
	slices.Sort(out)
	return out
}

func newMoneyApp(core *standInCore, db *sql.DB) *http.ServeMux {
	app := New(client.New(core.URL).WithToken(testAPIToken), "/")
	app.SetAPIToken(testAPIToken)
	app.Store = NewStore(db)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	return mux
}

func call(mux *http.ServeMux, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func want(t *testing.T, rec *httptest.ResponseRecorder, status int, what string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("%s: %d %s, want %d", what, rec.Code, rec.Body, status)
	}
}

func configValue(t *testing.T, mux *http.ServeMux, token string) string {
	t.Helper()
	rec := call(mux, "GET", "/api/config/vault_credential_id", token, "")
	want(t, rec, http.StatusOK, "read config")
	var resp struct {
		Value string `json:"value"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.Value
}

func setCredential(mux *http.ServeMux, token, credential string) *httptest.ResponseRecorder {
	return call(mux, "POST", "/api/config", token, `{"key":"vault_credential_id","value":"`+credential+`"}`)
}

func TestMoney_ConfigIsPerUser(t *testing.T) {
	core := newStandInCore(t, map[string]string{"cred-a": "alice", "cred-a2": "alice", "cred-b": "bob"}, "alice", "bob")
	db := openMigratedDB(t, "")
	mux := newMoneyApp(core, db)
	alice, bob := identity("alice", "write"), identity("bob", "write")

	want(t, setCredential(mux, alice, "cred-a"), http.StatusOK, "alice picks her credential")
	if got := configValue(t, mux, bob); got != "" {
		t.Fatalf("bob reads alice's credential choice: %q", got)
	}
	want(t, setCredential(mux, bob, "cred-b"), http.StatusOK, "bob picks his credential")
	if got := configValue(t, mux, alice); got != "cred-a" {
		t.Fatalf("bob's choice changed alice's: %q", got)
	}
	if got := configValue(t, mux, bob); got != "cred-b" {
		t.Fatalf("bob's credential = %q, want cred-b", got)
	}

	syncedUpTo := func(user string) string {
		var v string
		db.QueryRow(`SELECT value FROM user_config WHERE user_id = ? AND key = 'last_tx_sync_date'`, user).Scan(&v)
		return v
	}
	for _, user := range []string{"alice", "bob"} {
		if _, err := db.Exec(`INSERT OR REPLACE INTO user_config (user_id, key, value) VALUES (?, 'last_tx_sync_date', '2026-06-01')`, user); err != nil {
			t.Fatal(err)
		}
	}
	want(t, setCredential(mux, alice, "cred-a"), http.StatusOK, "alice saves the same credential")
	if got := syncedUpTo("alice"); got != "2026-06-01" {
		t.Fatalf("saving the same credential reset alice's sync progress to %q", got)
	}
	want(t, setCredential(mux, alice, "cred-a2"), http.StatusOK, "alice switches to another login")
	if got := syncedUpTo("alice"); got != "" {
		t.Fatalf("another login would sync from alice's old progress %q, want from the start", got)
	}
	if got := syncedUpTo("bob"); got != "2026-06-01" {
		t.Fatalf("alice's switch reset bob's sync progress to %q", got)
	}
}

func TestMoney_ConfigRefusesACredentialTheUserCannotRead(t *testing.T) {
	core := newStandInCore(t, map[string]string{"cred-a": "alice"}, "alice", "bob", "reader")
	mux := newMoneyApp(core, openMigratedDB(t, ""))
	bob := identity("bob", "admin")

	want(t, setCredential(mux, bob, "cred-a"), http.StatusBadRequest, "bob points his sync at alice's credential")
	want(t, setCredential(mux, bob, "no-such-credential"), http.StatusBadRequest, "bob points his sync at a missing credential")
	if got := configValue(t, mux, bob); got != "" {
		t.Fatalf("a refused credential was saved: %q", got)
	}
	want(t, setCredential(mux, identity("reader", "read"), "cred-a"), http.StatusForbidden, "a read-only user sets a credential")
}

func TestMoney_ScheduledSyncRunsForEachConfiguredUser(t *testing.T) {
	core := newStandInCore(t, map[string]string{"cred-a": "alice", "cred-b": "bob"}, "alice", "bob", "carol")
	mux := newMoneyApp(core, openMigratedDB(t, ""))
	alice, bob, carol := identity("alice", "admin"), identity("bob", "admin"), identity("carol", "write")
	want(t, setCredential(mux, alice, "cred-a"), http.StatusOK, "alice configures")
	want(t, setCredential(mux, bob, "cred-b"), http.StatusOK, "bob configures")

	want(t, call(mux, "POST", "/api/sync", testAPIToken, "{}"), http.StatusOK, "core's scheduled sync")

	if got, wantReads := core.reads(), []string{"alice:cred-a", "bob:cred-b"}; !slices.Equal(got, wantReads) {
		t.Fatalf("the scheduled sync read %v from the vault, want each user's own credential as that user: %v", got, wantReads)
	}
	if got, wantMinted := core.mintedForUsers(), []string{"alice:read", "bob:read"}; !slices.Equal(got, wantMinted) {
		t.Fatalf("core minted %v for the sync, want a read token per configured user: %v", got, wantMinted)
	}
	history := func(token string) []SyncLogEntry {
		var resp struct {
			History []SyncLogEntry `json:"history"`
		}
		json.Unmarshal(call(mux, "GET", "/api/sync-history", token, "").Body.Bytes(), &resp)
		return resp.History
	}
	if h := history(alice); len(h) != 1 || h[0].Status != "error" {
		t.Fatalf("alice's sync history = %+v, want the one failed run", h)
	}
	if h := history(bob); len(h) != 1 || h[0].Status != "error" {
		t.Fatalf("bob's sync history = %+v, want the one failed run", h)
	}
	if h := history(carol); len(h) != 0 {
		t.Fatalf("carol never configured a credential but has sync history %+v", h)
	}
}

func TestMoney_EachUsersFinancesAreTheirOwn(t *testing.T) {
	core := newStandInCore(t, nil, "alice", "bob")
	db := openMigratedDB(t, "")
	for _, stmt := range []string{
		`INSERT INTO accounts (id, user_id, display_name, account_type, display_balance) VALUES ('acc-a', 'alice', 'Alice Checking', 'depository', 5000)`,
		`INSERT INTO transactions (id, user_id, date, amount, merchant, category, tags_json) VALUES ('tx-a', 'alice', '2026-06-10', 3000, 'Starbucks', 'Paycheck', '[]')`,
		`INSERT INTO budgets (id, user_id, category_id, category_name, month, planned, actual) VALUES ('cat1_2026-06', 'alice', 'cat1', 'Food', '2026-06', 500, -320)`,
		`INSERT INTO sync_log (user_id, started_at, completed_at, status) VALUES ('alice', 1, 2, 'ok')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	mux := newMoneyApp(core, db)
	alice, bob := identity("alice", "read"), identity("bob", "read")

	count := func(token, path, field string) int {
		t.Helper()
		rec := call(mux, "GET", path, token, "")
		want(t, rec, http.StatusOK, "GET "+path)
		var resp struct {
			Accounts     []json.RawMessage `json:"accounts"`
			Transactions []json.RawMessage `json:"transactions"`
			Budgets      []json.RawMessage `json:"budgets"`
			History      []json.RawMessage `json:"history"`
		}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		switch field {
		case "accounts":
			return len(resp.Accounts)
		case "transactions":
			return len(resp.Transactions)
		case "budgets":
			return len(resp.Budgets)
		default:
			return len(resp.History)
		}
	}
	for _, c := range []struct{ path, field string }{
		{"/api/accounts", "accounts"},
		{"/api/transactions?start=2026-06-01&end=2026-06-30", "transactions"},
		{"/api/budgets?month=2026-06", "budgets"},
		{"/api/search?q=Starbucks", "transactions"},
		{"/api/sync-history", "history"},
	} {
		if n := count(bob, c.path, c.field); n != 0 {
			t.Errorf("bob sees %d of alice's %s", n, c.field)
		}
		if n := count(alice, c.path, c.field); n != 1 {
			t.Errorf("alice sees %d of her %s, want 1", n, c.field)
		}
	}
	var report CashflowReport
	json.Unmarshal(call(mux, "GET", "/api/reports/cashflow?start=2026-06-01&end=2026-06-30", bob, "").Body.Bytes(), &report)
	if report.TotalIncome != 0 {
		t.Errorf("bob's cashflow counts alice's income: %v", report.TotalIncome)
	}
	var status SyncStatus
	json.Unmarshal(call(mux, "GET", "/api/sync-status", bob, "").Body.Bytes(), &status)
	if status.Status != "never" {
		t.Errorf("bob's sync status shows alice's sync: %q", status.Status)
	}
}

// What the household synced before ownership goes to the first admin who
// signs in, and to nobody else, even across restarts.
func TestMoney_LegacyDataGoesToTheFirstAdminOnce(t *testing.T) {
	core := newStandInCore(t, nil, "root", "root2", "bob")
	db := openMigratedDB(t, userIDMigration)
	for _, stmt := range []string{
		`INSERT INTO accounts (id, display_name, account_type, display_balance) VALUES ('acc-legacy', 'Joint Checking', 'depository', 900)`,
		`INSERT INTO transactions (id, date, amount, merchant, category, tags_json) VALUES ('tx-legacy', '2026-06-10', -12, 'Bakery', 'Food', '[]')`,
		`INSERT INTO account_snapshots (date, balance) VALUES ('2026-06-01', 900)`,
		`INSERT INTO config (key, value) VALUES ('vault_credential_id', 'cred-legacy')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	execFile(t, db, filepath.Join("_migrations", userIDMigration))
	accounts := func(mux *http.ServeMux, token string) int {
		var resp struct {
			Accounts []LocalAccount `json:"accounts"`
		}
		json.Unmarshal(call(mux, "GET", "/api/accounts", token, "").Body.Bytes(), &resp)
		return len(resp.Accounts)
	}
	bob, root, root2 := identity("bob", "write"), identity("root", "admin"), identity("root2", "admin")

	mux := newMoneyApp(core, db)
	if n := accounts(mux, bob); n != 0 {
		t.Fatalf("a non-admin was given %d legacy accounts", n)
	}
	if got := configValue(t, mux, bob); got != "" {
		t.Fatalf("a non-admin was given the legacy credential %q", got)
	}
	if n := accounts(mux, root); n != 1 {
		t.Fatalf("the first admin has %d legacy accounts, want 1", n)
	}
	if got := configValue(t, mux, root); got != "cred-legacy" {
		t.Fatalf("the first admin's credential = %q, want the legacy one", got)
	}

	restarted := newMoneyApp(core, db)
	if n := accounts(restarted, root2); n != 0 {
		t.Fatalf("a second admin was given %d legacy accounts after a restart", n)
	}
	if n := accounts(restarted, root); n != 1 {
		t.Fatalf("the first admin has %d accounts after a restart, want 1", n)
	}
	for _, table := range []string{"accounts", "transactions", "account_snapshots", "user_config"} {
		var unowned int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE user_id = ''`).Scan(&unowned); err != nil {
			t.Fatal(err)
		}
		if unowned != 0 {
			t.Errorf("%d legacy %s rows are still unowned", unowned, table)
		}
	}
}

// Rows keyed by Monarch's ids stay with the user who synced them first: a
// second user syncing the same Monarch login cannot overwrite them (and,
// the known limit, does not get their own copy). Rows keyed by date are
// kept per user.
func TestMoney_ASyncCannotOverwriteAnotherUsersRows(t *testing.T) {
	store := NewStore(openMigratedDB(t, ""))
	ctx := t.Context()

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(store.UpsertAccount(ctx, "alice", LocalAccount{ID: "acc1", DisplayName: "Alice Checking", DisplayBalance: 100}))
	must(store.UpsertAccount(ctx, "bob", LocalAccount{ID: "acc1", DisplayName: "Bob's copy", DisplayBalance: 1}))
	must(store.UpsertBudget(ctx, "alice", LocalBudget{ID: "cat1_2026-06", CategoryID: "cat1", CategoryName: "Food", Month: "2026-06", Planned: 500}))
	must(store.UpsertBudget(ctx, "bob", LocalBudget{ID: "cat1_2026-06", CategoryID: "cat1", CategoryName: "Bob food", Month: "2026-06", Planned: 1}))
	must(store.UpsertBudget(ctx, "alice", LocalBudget{ID: "cat1_2026-06", CategoryID: "cat1", CategoryName: "Food", Month: "2026-06", Planned: 600}))
	must(store.UpsertSnapshot(ctx, "alice", LocalSnapshot{Date: "2026-06-01", Balance: 100}))
	must(store.UpsertSnapshot(ctx, "bob", LocalSnapshot{Date: "2026-06-01", Balance: 7}))

	accounts, err := store.ListAccounts(ctx, "alice")
	must(err)
	if len(accounts) != 1 || accounts[0].DisplayName != "Alice Checking" || accounts[0].DisplayBalance != 100 {
		t.Fatalf("bob's sync changed alice's account: %+v", accounts)
	}
	if bobs, _ := store.ListAccounts(ctx, "bob"); len(bobs) != 0 {
		t.Fatalf("bob reads alice's account through the shared Monarch id: %+v", bobs)
	}
	budgets, err := store.ListBudgets(ctx, "alice", "2026-06")
	must(err)
	if len(budgets) != 1 || budgets[0].Planned != 600 || budgets[0].CategoryName != "Food" {
		t.Fatalf("alice's budget = %+v, want her own update to 600", budgets)
	}
	aliceSnaps, _ := store.ListSnapshots(ctx, "alice", "", "")
	bobSnaps, _ := store.ListSnapshots(ctx, "bob", "", "")
	if len(aliceSnaps) != 1 || aliceSnaps[0].Balance != 100 || len(bobSnaps) != 1 || bobSnaps[0].Balance != 7 {
		t.Fatalf("snapshots on one date must be per user: alice %+v bob %+v", aliceSnaps, bobSnaps)
	}
}
