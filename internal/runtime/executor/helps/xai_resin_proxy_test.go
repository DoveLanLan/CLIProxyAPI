package helps

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
