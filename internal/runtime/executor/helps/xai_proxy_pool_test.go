package helps

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func writePoolToken(t *testing.T, value string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if errWrite := os.WriteFile(path, []byte(value+"\n"), 0o600); errWrite != nil {
		t.Fatalf("write token: %v", errWrite)
	}
	return path
}

func TestRemoteXAIProxyPoolUsesOpaqueRouteKeyAndProbeLease(t *testing.T) {
	var mu sync.Mutex
	var routeKey string
	var confirmed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer control-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/routes":
			var input struct {
				Key string `json:"key"`
			}
			_ = json.NewDecoder(r.Body).Decode(&input)
			mu.Lock()
			routeKey = input.Key
			mu.Unlock()
			_, _ = w.Write([]byte(`{"enrolled":true,"route":{"lane":"lane-1","proxy_url":"http://mihomo:17891","node":"node-1","provider":"provider-a","egress_ip":"198.51.100.1"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/probes":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"lease_id":"lease-1","current":{"lane":"lane-1","proxy_url":"http://mihomo:17891","node":"node-1","provider":"provider-a","egress_ip":"198.51.100.1"},"route":{"lane":"probe","proxy_url":"http://mihomo:17899","node":"node-2","provider":"provider-b","egress_ip":"198.51.100.2","probe":true}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/probes/lease-1/confirm-ip-block":
			mu.Lock()
			confirmed = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewXAIProxyPool(config.XAIProxyPoolConfig{
		Enabled: true, ServiceURL: server.URL, ServiceTokenFile: writePoolToken(t, "control-secret"),
	})
	defer client.Close()
	route, enrolled, errRoute := client.Route(context.Background(), "raw-auth-id")
	if errRoute != nil || !enrolled || route.ProxyURL != "http://mihomo:17891" {
		t.Fatalf("route/enrolled/error = %#v/%v/%v", route, enrolled, errRoute)
	}
	mu.Lock()
	key := routeKey
	mu.Unlock()
	if key == "" || key == "raw-auth-id" || strings.Contains(key, "raw-auth-id") || len(key) != 64 {
		t.Fatalf("route key was not an opaque SHA-256 digest: %q", key)
	}
	lease, errLease := client.AcquireProbe(context.Background(), route)
	if errLease != nil {
		t.Fatalf("AcquireProbe() error = %v", errLease)
	}
	if lease.AlternateRoute().ProxyURL != "http://mihomo:17899" {
		t.Fatalf("alternate route = %#v", lease.AlternateRoute())
	}
	if errConfirm := lease.ConfirmIPBlock(context.Background()); errConfirm != nil {
		t.Fatalf("ConfirmIPBlock() error = %v", errConfirm)
	}
	mu.Lock()
	wasConfirmed := confirmed
	mu.Unlock()
	if !wasConfirmed {
		t.Fatal("probe confirmation was not sent")
	}
}

func TestRemoteXAIProxyPoolMapsRetryableServiceError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"code":"pool_unavailable","error":"no lane is ready","retry_after_seconds":12}`))
	}))
	defer server.Close()
	client := NewXAIProxyPool(config.XAIProxyPoolConfig{
		Enabled: true, ServiceURL: server.URL, ServiceTokenFile: writePoolToken(t, "control-secret"),
	})
	_, enrolled, errRoute := client.Route(context.Background(), "auth-id")
	if !enrolled {
		t.Fatal("enabled pool did not fail closed")
	}
	poolErr, ok := errRoute.(*XAIProxyPoolError)
	if !ok || poolErr.StatusCode() != http.StatusServiceUnavailable || poolErr.RetryAfter() == nil || poolErr.RetryAfter().Seconds() != 12 {
		t.Fatalf("error = %#v", errRoute)
	}
}

func TestRemoteXAIProxyPoolSubscriptionMutationForwardsRevisionAndRedactsError(t *testing.T) {
	secretURL := "https://subscription.example.invalid/private-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Match") != `"7"` {
			t.Errorf("If-Match = %q", r.Header.Get("If-Match"))
		}
		var input XAIProxySubscriptionCreate
		_ = json.NewDecoder(r.Body).Decode(&input)
		if input.URL != secretURL {
			t.Errorf("URL was not forwarded")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":"provider_activation_failed","error":"subscription provider could not be activated"}`))
	}))
	defer server.Close()
	client := NewXAIProxyPool(config.XAIProxyPoolConfig{
		Enabled: true, ServiceURL: server.URL, ServiceTokenFile: writePoolToken(t, "control-secret"),
	})
	_, errCreate := client.CreateXAIProxySubscription(context.Background(), 7, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: secretURL, Enabled: true,
	})
	if errCreate == nil || strings.Contains(errCreate.Error(), secretURL) {
		t.Fatalf("error leaked subscription URL: %v", errCreate)
	}
	var subscriptionErr *XAIProxySubscriptionError
	if !errors.As(errCreate, &subscriptionErr) || subscriptionErr.Code != "provider_activation_failed" {
		t.Fatalf("error = %#v", errCreate)
	}
}

func TestEnabledRemoteXAIProxyPoolWithMissingTokenFailsClosed(t *testing.T) {
	client := NewXAIProxyPool(config.XAIProxyPoolConfig{
		Enabled: true, ServiceURL: "http://egress-proxy-controller:8080", ServiceTokenFile: filepath.Join(t.TempDir(), "missing"),
	})
	_, enrolled, errRoute := client.Route(context.Background(), "auth-id")
	if !enrolled || errRoute == nil {
		t.Fatalf("enrolled/error = %v/%v", enrolled, errRoute)
	}
}
