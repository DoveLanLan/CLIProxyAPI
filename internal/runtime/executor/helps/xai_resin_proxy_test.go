package helps

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func newTestXAIResinProxy(t *testing.T, mutate func(*config.XAIResinProxyConfig)) *XAIResinProxy {
	t.Helper()
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "proxy-token")
	identityKeyFile := filepath.Join(dir, "identity-key")
	if errWrite := os.WriteFile(tokenFile, []byte("resin-token-123456789\n"), 0o600); errWrite != nil {
		t.Fatalf("write proxy token: %v", errWrite)
	}
	if errWrite := os.WriteFile(identityKeyFile, []byte(strings.Repeat("k", 32)+"\n"), 0o600); errWrite != nil {
		t.Fatalf("write identity key: %v", errWrite)
	}
	cfg := config.XAIResinProxyConfig{
		Enabled:         true,
		ProxyURL:        "http://resin:2260",
		Platform:        "Default",
		ProxyTokenFile:  tokenFile,
		IdentityKeyFile: identityKeyFile,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	return NewXAIResinProxy(cfg, false)
}

func TestXAIResinProxyDerivesStableAnonymousAccounts(t *testing.T) {
	proxy := newTestXAIResinProxy(t, nil)
	first, routed, errFirst := proxy.ProxyURL("auths/alice.json")
	if errFirst != nil || !routed {
		t.Fatalf("first route = %q, %t, %v", first, routed, errFirst)
	}
	repeated, _, errRepeated := proxy.ProxyURL("auths/alice.json")
	if errRepeated != nil || repeated != first {
		t.Fatalf("repeated route = %q, %v; want %q", repeated, errRepeated, first)
	}
	second, _, errSecond := proxy.ProxyURL("auths/bob.json")
	if errSecond != nil || second == first {
		t.Fatalf("second route = %q, %v; first = %q", second, errSecond, first)
	}
	if strings.Contains(first, "alice") || strings.Contains(second, "bob") {
		t.Fatalf("raw auth ID leaked into routes: %q %q", first, second)
	}

	parsed, errParse := url.Parse(first)
	if errParse != nil {
		t.Fatalf("parse route: %v", errParse)
	}
	if parsed.Scheme != "http" || parsed.Host != "resin:2260" {
		t.Fatalf("route target = %s://%s", parsed.Scheme, parsed.Host)
	}
	username := parsed.User.Username()
	if username != "Default.xai-f8e1567f5c2b26597c4c3e9bd47afa1b" {
		t.Fatalf("username = %q", username)
	}
	password, okPassword := parsed.User.Password()
	if !okPassword || password != "resin-token-123456789" {
		t.Fatalf("password parsed = %q, %t", password, okPassword)
	}
}

func TestXAIResinProxyDisabledPreservesLegacyRouting(t *testing.T) {
	proxy := NewXAIResinProxy(config.XAIResinProxyConfig{}, false)
	proxyURL, routed, errRoute := proxy.ProxyURL("auth-1")
	if errRoute != nil || routed || proxyURL != "" {
		t.Fatalf("disabled route = %q, %t, %v", proxyURL, routed, errRoute)
	}
}

func TestXAIResinProxyFailsClosedForConflictingAutomaticRoutes(t *testing.T) {
	proxy := newTestXAIResinProxy(t, nil)
	proxy = NewXAIResinProxy(config.XAIResinProxyConfig{Enabled: true}, true)
	_, routed, errRoute := proxy.ProxyURL("auth-1")
	if !routed || errRoute == nil {
		t.Fatalf("conflicting route = %t, %v", routed, errRoute)
	}
	var resinErr *XAIResinProxyError
	if !errors.As(errRoute, &resinErr) || resinErr.StatusCode() != http.StatusServiceUnavailable || !resinErr.IsRequestScoped() {
		t.Fatalf("error = %#v", errRoute)
	}
	if resinErr.ManagedHeaders().Get("Retry-After") != "30" {
		t.Fatalf("Retry-After = %q", resinErr.ManagedHeaders().Get("Retry-After"))
	}
}

func TestXAIResinProxyRejectsMissingStableAuthID(t *testing.T) {
	proxy := newTestXAIResinProxy(t, nil)
	_, routed, errRoute := proxy.ProxyURL("  ")
	if !routed || errRoute == nil || strings.Contains(errRoute.Error(), "resin-token") {
		t.Fatalf("missing auth route = %t, %v", routed, errRoute)
	}
}

func TestXAIResinProxyRejectsShortIdentityKey(t *testing.T) {
	proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		if errWrite := os.WriteFile(cfg.IdentityKeyFile, []byte("short"), 0o600); errWrite != nil {
			t.Fatalf("write short key: %v", errWrite)
		}
	})
	_, routed, errRoute := proxy.ProxyURL("auth-1")
	if !routed || errRoute == nil || !strings.Contains(errRoute.Error(), "too short") {
		t.Fatalf("short-key route = %t, %v", routed, errRoute)
	}
}

func TestXAIResinProxyRejectsInvalidForwardProxyURLs(t *testing.T) {
	tests := []string{
		"http://user:token@resin:2260",
		"http://resin:2260/proxy",
		"http://resin:2260?account=xai",
		"ftp://resin:2260",
	}
	for _, proxyURL := range tests {
		t.Run(proxyURL, func(t *testing.T) {
			proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
				cfg.ProxyURL = proxyURL
			})
			_, routed, errRoute := proxy.ProxyURL("auth-1")
			if !routed || errRoute == nil {
				t.Fatalf("invalid URL route = %t, %v", routed, errRoute)
			}
		})
	}
}

func TestXAIResinProxyRejectsMissingProxyTokenFile(t *testing.T) {
	proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		cfg.ProxyTokenFile = filepath.Join(t.TempDir(), "missing-token")
	})
	_, routed, errRoute := proxy.ProxyURL("auth-1")
	if !routed || errRoute == nil || !strings.Contains(errRoute.Error(), "token is unavailable") {
		t.Fatalf("missing-token route = %t, %v", routed, errRoute)
	}
}

func TestXAIResinProxyRejectsReservedV1Values(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		token    string
	}{
		{name: "platform", platform: "api"},
		{name: "token", token: "healthz"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
				if tc.platform != "" {
					cfg.Platform = tc.platform
				}
				if tc.token != "" {
					if errWrite := os.WriteFile(cfg.ProxyTokenFile, []byte(tc.token), 0o600); errWrite != nil {
						t.Fatalf("write reserved token: %v", errWrite)
					}
				}
			})
			_, routed, errRoute := proxy.ProxyURL("auth-1")
			if !routed || errRoute == nil {
				t.Fatalf("reserved route = %t, %v", routed, errRoute)
			}
		})
	}
}

func TestXAIResinProxyRotatesLeaseThroughAuthenticatedAdminAPI(t *testing.T) {
	const platformID = "11111111-1111-1111-1111-111111111111"
	var listCalls atomic.Int32
	var deleteCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer admin-secret" {
			t.Errorf("Authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/control/api/v1/platforms":
			listCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"id":"` + platformID + `","name":"Default"}],"total":1,"limit":100000,"offset":0}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/control/api/v1/platforms/"+platformID+"/leases/xai-f8e1567f5c2b26597c4c3e9bd47afa1b":
			deleteCalls.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected admin request: %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		adminTokenFile := filepath.Join(t.TempDir(), "admin-token")
		if errWrite := os.WriteFile(adminTokenFile, []byte("admin-secret\n"), 0o600); errWrite != nil {
			t.Fatalf("write admin token: %v", errWrite)
		}
		cfg.AdminURL = server.URL + "/control/"
		cfg.AdminTokenFile = adminTokenFile
		cfg.Max402Retries = 2
	})
	if got := proxy.Max402Retries(); got != 2 {
		t.Fatalf("Max402Retries() = %d", got)
	}
	if generation := proxy.LeaseGeneration("auths/alice.json"); generation != 0 {
		t.Fatalf("initial generation = %d", generation)
	}
	if errRotate := proxy.RotateLease(context.Background(), "auths/alice.json", 0); errRotate != nil {
		t.Fatalf("RotateLease() error = %v", errRotate)
	}
	if generation := proxy.LeaseGeneration("auths/alice.json"); generation != 1 {
		t.Fatalf("rotated generation = %d", generation)
	}
	if errRotate := proxy.RotateLease(context.Background(), "auths/alice.json", 0); errRotate != nil {
		t.Fatalf("stale RotateLease() error = %v", errRotate)
	}
	if listCalls.Load() != 1 || deleteCalls.Load() != 1 {
		t.Fatalf("admin calls = list:%d delete:%d", listCalls.Load(), deleteCalls.Load())
	}
}

func TestXAIResinProxyCoalescesConcurrentLeaseRotation(t *testing.T) {
	const platformID = "22222222-2222-2222-2222-222222222222"
	var deleteCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"items":[{"id":"` + platformID + `","name":"Default"}]}`))
			return
		}
		deleteCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		adminTokenFile := filepath.Join(t.TempDir(), "admin-token")
		if errWrite := os.WriteFile(adminTokenFile, []byte("admin-secret"), 0o600); errWrite != nil {
			t.Fatalf("write admin token: %v", errWrite)
		}
		cfg.AdminURL = server.URL
		cfg.AdminTokenFile = adminTokenFile
		cfg.Max402Retries = 2
	})

	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- proxy.RotateLease(context.Background(), "auth-concurrent", 0)
		}()
	}
	wg.Wait()
	close(errs)
	for errRotate := range errs {
		if errRotate != nil {
			t.Fatalf("RotateLease() error = %v", errRotate)
		}
	}
	if deleteCalls.Load() != 1 || proxy.LeaseGeneration("auth-concurrent") != 1 {
		t.Fatalf("delete calls/generation = %d/%d", deleteCalls.Load(), proxy.LeaseGeneration("auth-concurrent"))
	}
}

func TestXAIResinProxyRefreshesCachedPlatformAfterNotFound(t *testing.T) {
	const oldPlatformID = "33333333-3333-3333-3333-333333333333"
	const newPlatformID = "44444444-4444-4444-4444-444444444444"
	var listCalls atomic.Int32
	var deleteCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			id := oldPlatformID
			if listCalls.Add(1) > 1 {
				id = newPlatformID
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"` + id + `","name":"Default"}]}`))
		case http.MethodDelete:
			deleteCalls.Add(1)
			if strings.Contains(r.URL.Path, oldPlatformID) {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		adminTokenFile := filepath.Join(t.TempDir(), "admin-token")
		if errWrite := os.WriteFile(adminTokenFile, []byte("admin-secret"), 0o600); errWrite != nil {
			t.Fatalf("write admin token: %v", errWrite)
		}
		cfg.AdminURL = server.URL
		cfg.AdminTokenFile = adminTokenFile
		cfg.Max402Retries = 1
	})
	if errRotate := proxy.RotateLease(context.Background(), "auth-platform-refresh", 0); errRotate != nil {
		t.Fatalf("RotateLease() error = %v", errRotate)
	}
	if listCalls.Load() != 2 || deleteCalls.Load() != 2 {
		t.Fatalf("admin calls = list:%d delete:%d", listCalls.Load(), deleteCalls.Load())
	}
}

func TestXAIResinProxyRequiresAdminSecretsOnlyWhenRetriesEnabled(t *testing.T) {
	proxy := newTestXAIResinProxy(t, nil)
	if _, routed, errRoute := proxy.ProxyURL("auth-no-retry"); errRoute != nil || !routed {
		t.Fatalf("proxy-only route = %t, %v", routed, errRoute)
	}

	proxy = newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		cfg.AdminURL = "http://admin:secret@resin:2260"
		cfg.AdminTokenFile = "/missing/admin-token"
		cfg.Max402Retries = 2
	})
	_, routed, errRoute := proxy.ProxyURL("auth-retry")
	if !routed || errRoute == nil || strings.Contains(errRoute.Error(), "admin:secret") {
		t.Fatalf("invalid admin route = %t, %v", routed, errRoute)
	}
}

func TestXAIResinProxyAdminFailureIsRequestScopedAndRedacted(t *testing.T) {
	const leakedAccount = "xai-f8e1567f5c2b26597c4c3e9bd47afa1b"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"items":[{"id":"55555555-5555-5555-5555-555555555555","name":"Default"}]}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"admin-secret ` + leakedAccount + `"}`))
	}))
	defer server.Close()

	proxy := newTestXAIResinProxy(t, func(cfg *config.XAIResinProxyConfig) {
		adminTokenFile := filepath.Join(t.TempDir(), "admin-token")
		if errWrite := os.WriteFile(adminTokenFile, []byte("admin-secret"), 0o600); errWrite != nil {
			t.Fatalf("write admin token: %v", errWrite)
		}
		cfg.AdminURL = server.URL
		cfg.AdminTokenFile = adminTokenFile
		cfg.Max402Retries = 1
	})
	errRotate := proxy.RotateLease(context.Background(), "auths/alice.json", 0)
	var resinErr *XAIResinProxyError
	if !errors.As(errRotate, &resinErr) || !resinErr.IsRequestScoped() || resinErr.StatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("rotation error = %#v", errRotate)
	}
	if strings.Contains(errRotate.Error(), "admin-secret") || strings.Contains(errRotate.Error(), leakedAccount) {
		t.Fatalf("rotation error leaked secret data: %v", errRotate)
	}
	if generation := proxy.LeaseGeneration("auths/alice.json"); generation != 0 {
		t.Fatalf("generation advanced after failed deletion: %d", generation)
	}
}
