package mmoney

import (
	"testing"
	"time"

	"github.com/localitas/localitas-go"
)

// resolveVaultCredential used to build a fresh http.Client on every sync. The App
// now holds one pooled client, created once in New. Assert it exists and is
// bounded.
func TestApp_SharedHTTPClient(t *testing.T) {
	app := New(&client.Client{}, "/")
	if app.httpClient == nil {
		t.Fatal("App must hold a shared http client created in New")
	}
	if app.httpClient.Timeout <= 0 || app.httpClient.Timeout > time.Minute {
		t.Fatalf("shared client timeout = %v, want a sane positive bound", app.httpClient.Timeout)
	}
}
