package mmoney

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	client "github.com/localitas/localitas-go"
	"github.com/localitas/localitas-go/identitytest"
)

const testAPIToken = "lt_test_cluster_token"

// signer signs identity tokens as core does; the app verifies them with its
// public key.
var signer *identitytest.Signer

// untrustedKey is an identity key core does not hold: the tokens it signs
// are well formed, but the app does not trust it.
var untrustedKey = ed25519.NewKeyFromSeed([]byte("mmoney-test-untrusted-key-seed-0"))

func TestMain(m *testing.M) {
	os.Setenv("LOCALITAS_API_TOKEN", testAPIToken)
	var err error
	if signer, err = identitytest.Install(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// identity signs claims as core does for a request it forwards to the app.
func identity(userID, permission string) string {
	return signer.Token(client.TokenInfo{UserID: userID, Email: userID + "@test.local", Permission: permission})
}

// untrustedIdentity signs claims as core does but with a key core does not
// hold — a forgery, which must not be accepted.
func untrustedIdentity(userID, permission string) string {
	now := time.Now().UTC()
	info := client.TokenInfo{
		UserID:     userID,
		Email:      userID + "@test.local",
		Permission: permission,
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(5 * time.Minute).Unix(),
		MaxExpiry:  now.Add(24 * time.Hour).Unix(),
		KeyID:      client.Ed25519KeyID(untrustedKey.Public().(ed25519.PublicKey)),
	}
	raw, _ := json.Marshal(info)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(untrustedKey, []byte(payload)))
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

// Finances are not served to callers core has not vouched for, and a user
// syncs only from a credential the vault shares with them.
func TestMoney_RoutesRequireAVerifiedUser(t *testing.T) {
	app := setupTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	forged := base64.StdEncoding.EncodeToString([]byte(`{"user_id":"x","permission":"admin"}`))
	for _, token := range []string{"", forged, untrustedIdentity("x", "admin")} {
		for _, path := range []string{"/api/accounts", "/api/transactions", "/api/net-worth", "/api/config/vault_credential_id"} {
			if got := guardedCall(mux, "GET", path, token, ""); got != http.StatusUnauthorized {
				t.Errorf("GET %s with %q: %d, want 401", path, token, got)
			}
		}
		if got := guardedCall(mux, "POST", "/api/sync", token, "{}"); got != http.StatusUnauthorized {
			t.Errorf("POST /api/sync with %q: %d, want 401", token, got)
		}
	}
	body := `{"key":"vault_credential_id","value":"test-123"}`
	if got := guardedCall(mux, "POST", "/api/config", identity("bob", "write"), body); got != http.StatusBadRequest {
		t.Fatalf("a user points their sync at someone else's credential: %d, want 400", got)
	}
	if got := guardedCall(mux, "POST", "/api/config", identity("reader", "read"), body); got != http.StatusForbidden {
		t.Fatalf("a read-only user sets a credential: %d, want 403", got)
	}
	if got := guardedCall(mux, "POST", "/api/config", identity("admin-user", "write"), body); got != http.StatusOK {
		t.Fatalf("a user picks their own credential: %d, want 200", got)
	}
	if got := guardedCall(mux, "POST", "/api/sync", identity("reader", "read"), "{}"); got != http.StatusForbidden {
		t.Fatalf("a read-only user starts a sync: %d, want 403", got)
	}
}
