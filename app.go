package mmoney

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/localitas/localitas-go"
)

var logger = slog.Default().With("component", "mmoney")

type App struct {
	Store             *Store
	BasePath          string
	client            *client.Client
	coreURL           string
	token             string
	cachedToken       string
	cachedTokenExpiry time.Time
	// httpClient is a single pooled client for raw calls to core (vault) and to
	// Monarch, created once here and reused instead of per-sync.
	httpClient *http.Client
}

func New(c *client.Client, basePath string) *App {
	if basePath == "" {
		basePath = "/"
	}
	return &App{
		BasePath:   basePath,
		client:     c,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *App) SetCoreAccess(coreURL, token string) {
	a.coreURL = coreURL
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
	// One Monarch account serves the household: every route needs the
	// identity core vouches for, choosing the credential it syncs from is
	// an admin's, and sync serves core's cron and a user's refresh.
	user := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, client.AuthMiddleware(handler))
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
	syncAsUser := client.AuthMiddleware(client.RequireScopeFunc(client.ScopeWrite, h.handleSync))
	mux.HandleFunc("POST /api/sync", func(w http.ResponseWriter, r *http.Request) {
		if client.IsSystemRequest(r) {
			h.handleSync(w, r)
			return
		}
		syncAsUser.ServeHTTP(w, r)
	})
	user("POST /api/config", client.RequireScopeFunc(client.ScopeAdmin, h.handleSetConfig))
	user("GET /api/config/{key}", h.handleGetConfig)
}
