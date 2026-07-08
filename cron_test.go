package mmoney

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCron(t *testing.T) {
	req := httptest.NewRequest("GET", "/cron.json", nil)
	w := httptest.NewRecorder()
	HandleCron(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var spec struct {
		Jobs []struct {
			ID       string `json:"id"`
			Path     string `json:"path"`
			Method   string `json:"method"`
			Schedule string `json:"schedule"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(spec.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(spec.Jobs))
	}

	if spec.Jobs[0].ID != "cron:mmoney:sync" {
		t.Errorf("expected id cron:mmoney:sync, got %s", spec.Jobs[0].ID)
	}
	if spec.Jobs[0].Schedule != "*/10 * * * *" {
		t.Errorf("expected schedule */10 * * * *, got %s", spec.Jobs[0].Schedule)
	}
}

func TestHandleCron_ContentType(t *testing.T) {
	req := httptest.NewRequest("GET", "/cron.json", nil)
	w := httptest.NewRecorder()
	HandleCron(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestHandleCron_RetryConfig(t *testing.T) {
	req := httptest.NewRequest("GET", "/cron.json", nil)
	w := httptest.NewRecorder()
	HandleCron(w, req)

	var spec struct {
		Jobs []struct {
			Retry struct {
				BackoffMultiplier float64 `json:"backoff_multiplier"`
			} `json:"retry"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if spec.Jobs[0].Retry.BackoffMultiplier != 2.0 {
		t.Errorf("expected backoff_multiplier 2.0, got %f", spec.Jobs[0].Retry.BackoffMultiplier)
	}
}

func TestHandleCron_SyncPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/cron.json", nil)
	w := httptest.NewRecorder()
	HandleCron(w, req)

	var spec struct {
		Jobs []struct {
			Path string `json:"path"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if spec.Jobs[0].Path != "/api/sync" {
		t.Errorf("expected path /api/sync, got %s", spec.Jobs[0].Path)
	}
}
