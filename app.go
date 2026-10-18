package mmoney

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/localitas/localitas-go"
)

var logger = slog.Default().With("component", "mmoney")

type App struct {
	Store    *Store
	BasePath string
	client   *client.Client
	// token is the API token core gave the app; a sync signs each user's
	// identity with the key core derives from it.
	token string
	// sessions holds each user's signed-in Monarch session.
	sessionsMu sync.Mutex
	sessions   map[string]monarchSession
	// legacyClaimed is set once an admin has been given the data synced
	// before per-user ownership (see Store.ClaimLegacyData).
	legacyClaimed atomic.Bool
	// httpClient is a single pooled client for calls to core, created once
	// here and reused instead of per-sync.
	httpClient *http.Client
}

func New(c *client.Client, basePath string) *App {
	if basePath == "" {
		basePath = "/"
	}
	return &App{
		BasePath:   basePath,
		client:     c,
		sessions:   make(map[string]monarchSession),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *App) SetAPIToken(token string) {
	a.token = token
}

func (a *App) InitStore(coreURL, dbID, token string) error {
	store, err := OpenStore(coreURL, dbID, token)
	if err != nil {
		return err
	}
	a.Store = store
	return nil
}

func (a *App) Install(ctx context.Context) (string, error) {
	for attempt := 1; ; attempt++ {
		db, err := a.client.CreateSystemDatabase(ctx, DatabaseName)
		if err != nil {
			logger.Warn("install attempt failed", "attempt", attempt, "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if err := applyEmbeddedMigrations(ctx, a.client, db.ID); err != nil {
			logger.Warn("install migrations attempt failed", "attempt", attempt, "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return db.ID, nil
	}
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(TemplatesFS, "templates/index.html")
	if err != nil {
		logger.Error("index template error", "error", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	data := map[string]string{"BasePath": a.BasePath}
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		logger.Error("index render error", "error", err)
	}
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	h := &handler{app: a}

	mux.HandleFunc("GET /{$}", a.handleIndex)
	mux.HandleFunc("GET /swagger.json", HandleSwagger)
	mux.HandleFunc("GET /help.md", handleHelpMarkdown)
	// Each user syncs their own Monarch login and sees only what it synced:
	// every route needs the identity core vouches for, and sync serves
	// core's cron (every configured user) and a user's refresh (theirs).
	user := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, client.AuthMiddleware(a.claimLegacyData(handler)))
	}
	user("GET /api/accounts", h.handleListAccounts)
	user("GET /api/transactions", h.handleListTransactions)
	user("GET /api/categories", h.handleListCategories)
	user("GET /api/budgets", h.handleListBudgets)
	user("GET /api/recurring", h.handleListRecurring)
	user("GET /api/investments", h.handleListInvestments)
	user("GET /api/investment-performance", h.handleInvestmentPerformance)
	user("GET /api/credit", h.handleListCreditScores)
	user("GET /api/snapshots", h.handleListSnapshots)
	user("GET /api/snapshots/breakdown", h.handleAssetLiabilitySnapshots)
	user("GET /api/net-worth", h.handleNetWorth)
	user("GET /api/cashflow", h.handleCashflow)
	user("GET /api/reports/cashflow", h.handleCashflowReport)
	user("GET /api/reports/fire", h.handleFIRE)
	user("GET /api/search", h.handleSearch)
	user("GET /api/sync-status", h.handleSyncStatus)
	user("GET /api/sync-history", h.handleSyncHistory)
	syncAsUser := client.AuthMiddleware(a.claimLegacyData(client.RequireScopeFunc(client.ScopeWrite, h.handleSync)))
	mux.HandleFunc("POST /api/sync", func(w http.ResponseWriter, r *http.Request) {
		if client.IsSystemRequest(r) {
			h.handleSyncAll(w, r)
			return
		}
		syncAsUser.ServeHTTP(w, r)
	})
	user("POST /api/config", client.RequireScopeFunc(client.ScopeWrite, h.handleSetConfig))
	user("GET /api/config/{key}", h.handleGetConfig)
}

// claimLegacyData gives the data synced before per-user ownership to the
// first admin core vouches for. ListUsers does not say who is an admin, so
// the first verified admin request is where the app learns it. A failed
// claim is retried on that admin's next request.
func (a *App) claimLegacyData(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.legacyClaimed.Load() && client.GetScope(r.Context()) == client.ScopeAdmin {
			if err := a.Store.ClaimLegacyData(r.Context(), client.GetUserID(r.Context())); err != nil {
				logger.Error("legacy data not claimed", "user_id", client.GetUserID(r.Context()), "error", err)
			} else {
				a.legacyClaimed.Store(true)
			}
		}
		next(w, r)
	}
}
