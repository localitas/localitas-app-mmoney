package mmoney

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	client "github.com/localitas/localitas-go"
)

const testAPIToken = "lt_test_cluster_token"

func TestMain(m *testing.M) {
	os.Setenv("LOCALITAS_API_TOKEN", testAPIToken)
	client.SetIdentityKeyFromAPIToken(testAPIToken)
	os.Exit(m.Run())
}

// identity signs claims as core does for a request it forwards to the app.
func identity(userID, permission string) string {
	raw, _ := json.Marshal(client.TokenInfo{UserID: userID, Email: userID + "@test.local", Permission: permission})
	payload := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte("localitas-internal-token-v1:" + testAPIToken))
	mac := hmac.New(sha256.New, []byte(base64.StdEncoding.EncodeToString(sum[:])))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func guardedCall(mux *http.ServeMux, method, path, token, body string) int {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Code
}

// The household's finances are not served to callers core has not
// vouched for, and only an admin chooses which credential the app syncs.
func TestMoney_RoutesRequireAVerifiedUser(t *testing.T) {
	app := setupTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	forged := base64.StdEncoding.EncodeToString([]byte(`{"user_id":"x","permission":"admin"}`))
	for _, token := range []string{"", forged} {
		for _, path := range []string{"/api/accounts", "/api/transactions", "/api/net-worth", "/api/config/vault_credential_id"} {
			if got := guardedCall(mux, "GET", path, token, ""); got != http.StatusUnauthorized {
				t.Errorf("GET %s with %q: %d, want 401", path, token, got)
			}
		}
		if got := guardedCall(mux, "POST", "/api/sync", token, "{}"); got != http.StatusUnauthorized {
			t.Errorf("POST /api/sync with %q: %d, want 401", token, got)
		}
	}
	body := `{"key":"vault_credential_id","value":"someone-elses-credential"}`
	if got := guardedCall(mux, "POST", "/api/config", identity("bob", "write"), body); got != http.StatusForbidden {
		t.Fatalf("a non-admin sets the synced credential: %d, want 403", got)
	}
	if got := guardedCall(mux, "POST", "/api/config", identity("root", "admin"), body); got != http.StatusOK {
		t.Fatalf("an admin sets the synced credential: %d, want 200", got)
	}
	if got := guardedCall(mux, "POST", "/api/sync", identity("reader", "read"), "{}"); got != http.StatusForbidden {
		t.Fatalf("a read-only user starts a sync: %d, want 403", got)
	}
}
