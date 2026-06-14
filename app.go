package mmoney

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/localitas/localitas-go"
)

type App struct {
	Store             *Store
	BasePath          string
	client            *client.Client
	coreURL           string
	token             string
	cachedToken       string
	cachedTokenExpiry time.Time
}

func New(c *client.Client, basePath string) *App {
	if basePath == "" {
		basePath = "/"
	}
	return &App{
		BasePath: basePath,
		client:   c,
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
			log.Printf("install: attempt %d failed (retrying): %v", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if err := applyEmbeddedMigrations(ctx, a.client, db.ID); err != nil {
			log.Printf("install: migrations attempt %d failed (retrying): %v", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}
		return db.ID, nil
	}
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(TemplatesFS, "templates/index.html")
	if err != nil {
		log.Printf("mmoney index template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	data := map[string]string{"BasePath": a.BasePath}
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("mmoney index render error: %v", err)
	}
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	h := &handler{app: a}

	mux.HandleFunc("GET /{$}", a.handleIndex)
	mux.HandleFunc("GET /swagger.json", HandleSwagger)
	mux.HandleFunc("GET /help.md", handleHelpMarkdown)
	mux.HandleFunc("GET /api/accounts", h.handleListAccounts)
	mux.HandleFunc("GET /api/transactions", h.handleListTransactions)
	mux.HandleFunc("GET /api/categories", h.handleListCategories)
	mux.HandleFunc("GET /api/budgets", h.handleListBudgets)
	mux.HandleFunc("GET /api/recurring", h.handleListRecurring)
	mux.HandleFunc("GET /api/investments", h.handleListInvestments)
	mux.HandleFunc("GET /api/investment-performance", h.handleInvestmentPerformance)
	mux.HandleFunc("GET /api/credit", h.handleListCreditScores)
	mux.HandleFunc("GET /api/snapshots", h.handleListSnapshots)
	mux.HandleFunc("GET /api/snapshots/breakdown", h.handleAssetLiabilitySnapshots)
	mux.HandleFunc("GET /api/net-worth", h.handleNetWorth)
	mux.HandleFunc("GET /api/cashflow", h.handleCashflow)
	mux.HandleFunc("GET /api/reports/cashflow", h.handleCashflowReport)
	mux.HandleFunc("GET /api/reports/fire", h.handleFIRE)
	mux.HandleFunc("GET /api/search", h.handleSearch)
	mux.HandleFunc("GET /api/sync-status", h.handleSyncStatus)
	mux.HandleFunc("GET /api/sync-history", h.handleSyncHistory)
	mux.HandleFunc("POST /api/sync", h.handleSync)
	mux.HandleFunc("POST /api/config", h.handleSetConfig)
	mux.HandleFunc("GET /api/config/{key}", h.handleGetConfig)
}
