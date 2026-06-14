package mmoney

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type handler struct {
	app *App
}

func (h *handler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.app.Store.ListAccounts(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list accounts: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"accounts": accounts})
}

func (h *handler) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			offset = v
		}
	}

	txs, total, err := h.app.Store.ListTransactions(r.Context(), startDate, endDate, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list transactions: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

func (h *handler) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.app.Store.ListCategories(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list categories: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"categories": cats})
}

func (h *handler) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	budgets, err := h.app.Store.ListBudgets(r.Context(), month)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list budgets: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"budgets": budgets})
}

func (h *handler) handleListRecurring(w http.ResponseWriter, r *http.Request) {
	recurring, err := h.app.Store.ListRecurring(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list recurring: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"recurring": recurring})
}

func (h *handler) handleListInvestments(w http.ResponseWriter, r *http.Request) {
	investments, err := h.app.Store.ListInvestments(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list investments: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"investments": investments})
}

func (h *handler) handleListCreditScores(w http.ResponseWriter, r *http.Request) {
	scores, err := h.app.Store.ListCreditScores(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list credit scores: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"credit_scores": scores})
}

func (h *handler) handleInvestmentPerformance(w http.ResponseWriter, r *http.Request) {
	securityID := r.URL.Query().Get("security_id")
	if securityID == "" {
		writeErr(w, http.StatusBadRequest, "security_id parameter required")
		return
	}
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")

	perf, err := h.app.Store.ListInvestmentPerformance(r.Context(), securityID, startDate, endDate)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get investment performance: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"performance": perf})
}

func (h *handler) handleAssetLiabilitySnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := h.app.Store.ListAssetLiabilitySnapshots(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get snapshots: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"snapshots": snapshots})
}

func (h *handler) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	snapshots, err := h.app.Store.ListSnapshots(r.Context(), startDate, endDate)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list snapshots: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"snapshots": snapshots})
}

func (h *handler) handleNetWorth(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	snapshots, err := h.app.Store.ListSnapshots(r.Context(), startDate, endDate)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get net worth: %v", err)
		return
	}
	points := make([]NetWorthPoint, len(snapshots))
	for i, s := range snapshots {
		points[i] = NetWorthPoint{Date: s.Date, Balance: s.Balance}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"net_worth": points})
}

func (h *handler) handleCashflow(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	if startDate == "" {
		startDate = time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().UTC().Format("2006-01-02")
	}

	txs, _, err := h.app.Store.ListTransactions(r.Context(), startDate, endDate, 10000, 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to compute cashflow: %v", err)
		return
	}

	var income, expense float64
	for _, tx := range txs {
		if tx.Amount >= 0 {
			income += tx.Amount
		} else {
			expense += tx.Amount
		}
	}
	savings := income + expense
	savingsRate := 0.0
	if income > 0 {
		savingsRate = savings / income * 100
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"income":       income,
		"expense":      expense,
		"savings":      savings,
		"savings_rate": savingsRate,
		"start":        startDate,
		"end":          endDate,
	})
}

func (h *handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeErr(w, http.StatusBadRequest, "q parameter required")
		return
	}
	txs, err := h.app.Store.SearchTransactions(r.Context(), q, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "search failed: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"transactions": txs})
}

func (h *handler) handleFIRE(w http.ResponseWriter, r *http.Request) {
	report, err := h.app.Store.GetFIREReport(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get FIRE report: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *handler) handleCashflowReport(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	if startDate == "" {
		startDate = time.Now().UTC().AddDate(0, -6, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().UTC().Format("2006-01-02")
	}
	recurringOnly := r.URL.Query().Get("recurring") == "1"
	report, err := h.app.Store.GetCashflowReport(r.Context(), startDate, endDate, recurringOnly)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get cashflow report: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *handler) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.app.Store.GetLastSync(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get sync status: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *handler) handleSync(w http.ResponseWriter, r *http.Request) {
	if err := h.app.RunSync(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "sync failed: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (h *handler) handleSyncHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.app.Store.ListSyncHistory(r.Context(), 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get sync history: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"history": history})
}

func (h *handler) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Key == "" {
		writeErr(w, http.StatusBadRequest, "key required")
		return
	}
	if err := h.app.Store.SetConfig(r.Context(), body.Key, body.Value); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to set config: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"key": body.Key, "value": body.Value})
}

func (h *handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, err := h.app.Store.GetConfig(r.Context(), key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get config: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"key": key, "value": value})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, format string, args ...interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf(format, args...)})
}
