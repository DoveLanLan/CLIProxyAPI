package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
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
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type executorResinProxy struct {
	mu              sync.Mutex
	proxyURL        string
	proxyURLForAuth func(string) string
	err             error
	authIDs         []string
	max402Retries   int
	generation      uint64
	rotateErr       error
	rotateAuthIDs   []string
}

func (p *executorResinProxy) ProxyURL(authID string) (string, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.authIDs = append(p.authIDs, authID)
	if p.proxyURLForAuth != nil {
		return p.proxyURLForAuth(authID), true, p.err
	}
	return p.proxyURL, true, p.err
}

func (p *executorResinProxy) calls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.authIDs...)
}

func (p *executorResinProxy) Max402Retries() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.max402Retries
}

func (p *executorResinProxy) LeaseGeneration(string) uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.generation
}

func (p *executorResinProxy) RotateLease(_ context.Context, authID string, observedGeneration uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rotateAuthIDs = append(p.rotateAuthIDs, authID)
	if p.rotateErr != nil {
		return p.rotateErr
	}
	if p.generation == observedGeneration {
		p.generation++
	}
	return nil
}

func (p *executorResinProxy) rotationCalls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.rotateAuthIDs...)
}

func TestXAIRoutedAuthUsesResinBeforeEgressPool(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	resin := &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}
	exec := &XAIAutoExecutor{proxyPool: pool, resinProxy: resin}
	auth := &cliproxyauth.Auth{ID: "auth-1", Provider: "xai"}

	routed, errRoute := exec.routedAuth(context.Background(), auth)
	if errRoute != nil {
		t.Fatalf("routedAuth() error = %v", errRoute)
	}
	if routed.auth == auth || routed.auth.ProxyURL != resin.proxyURL || routed.poolUsed {
		t.Fatalf("routed auth = %#v", routed)
	}
	if auth.ProxyURL != "" {
		t.Fatalf("original auth mutated: %q", auth.ProxyURL)
	}
	if got := resin.calls(); len(got) != 1 || got[0] != auth.ID {
		t.Fatalf("Resin calls = %#v", got)
	}
	if status := controller.Status(context.Background()); status.Counters.Requests != 0 {
		t.Fatalf("Egress pool unexpectedly routed request: %#v", status.Counters)
	}
}

func TestNewXAIAutoExecutorBuildsResinRouterFromConfig(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "proxy-token")
	identityKeyFile := filepath.Join(dir, "identity-key")
	if errWrite := os.WriteFile(tokenFile, []byte("resin-token"), 0o600); errWrite != nil {
		t.Fatalf("write proxy token: %v", errWrite)
	}
	if errWrite := os.WriteFile(identityKeyFile, []byte(strings.Repeat("k", 32)), 0o600); errWrite != nil {
		t.Fatalf("write identity key: %v", errWrite)
	}
	exec := NewXAIAutoExecutor(&config.Config{XAIResinProxy: config.XAIResinProxyConfig{
		Enabled:         true,
		ProxyURL:        "http://resin:2260",
		Platform:        "Default",
		ProxyTokenFile:  tokenFile,
		IdentityKeyFile: identityKeyFile,
	}})
	t.Cleanup(func() { exec.CloseExecutionSession(cliproxyauth.CloseAllExecutionSessionsID) })

	routed, errRoute := exec.routedAuth(context.Background(), &cliproxyauth.Auth{ID: "auth-config", Provider: "xai"})
	if errRoute != nil {
		t.Fatalf("routedAuth() error = %v", errRoute)
	}
	if !routed.resinUsed || routed.poolUsed || routed.auth == nil {
		t.Fatalf("routed auth = %#v", routed)
	}
	parsed, errParse := url.Parse(routed.auth.ProxyURL)
	if errParse != nil || parsed.Host != "resin:2260" || parsed.User == nil {
		t.Fatalf("derived proxy URL = %q, %v", routed.auth.ProxyURL, errParse)
	}
	if password, okPassword := parsed.User.Password(); !okPassword || password != "resin-token" {
		t.Fatalf("derived proxy password = %q, %t", password, okPassword)
	}
}

func TestXAIRoutedAuthExplicitProxyBypassesResin(t *testing.T) {
	resin := &executorResinProxy{proxyURL: "http://resin:2260"}
	exec := &XAIAutoExecutor{resinProxy: resin}
	auth := &cliproxyauth.Auth{ID: "auth-explicit", Provider: "xai", ProxyURL: "direct"}

	routed, errRoute := exec.routedAuth(context.Background(), auth)
	if errRoute != nil || routed.auth != auth || routed.auth.ProxyURL != "direct" {
		t.Fatalf("routed auth/error = %#v, %v", routed, errRoute)
	}
	if got := resin.calls(); len(got) != 0 {
		t.Fatalf("Resin calls = %#v", got)
	}
}

func TestXAIExecuteWithResinKeepsSingleAttemptWhenRetryDisabled(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	resin := &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}
	exec := &XAIAutoExecutor{proxyPool: pool, resinProxy: resin}
	attempts := 0

	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-402", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", blockedSpendingLimitError()
	})
	if !isXAIBlockedSpendingLimit(errExecute) || attempts != 1 {
		t.Fatalf("execute error/attempts = %v, %d", errExecute, attempts)
	}
	if status := controller.Status(context.Background()); status.Counters.Requests != 0 || status.Counters.Exact402 != 0 {
		t.Fatalf("Egress pool unexpectedly observed Resin request: %#v", status.Counters)
	}
}

func TestXAIExecuteWithResinRotatesLeaseAndRetriesExact402(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
	}
	exec := &XAIAutoExecutor{proxyPool: pool, resinProxy: resin}
	auth := &cliproxyauth.Auth{ID: "auth-rotate", Provider: "xai"}
	var targets []string
	attempts := 0

	result, errExecute := executeXAIWithProxyPool(context.Background(), exec, auth, func(routed *cliproxyauth.Auth) (string, error) {
		attempts++
		targets = append(targets, xaiProxySessionTarget(routed))
		if attempts < 3 {
			return "", blockedSpendingLimitError()
		}
		return "ok", nil
	})
	if errExecute != nil || result != "ok" || attempts != 3 {
		t.Fatalf("result/error/attempts = %q/%v/%d", result, errExecute, attempts)
	}
	if got := resin.rotationCalls(); len(got) != 2 || got[0] != auth.ID || got[1] != auth.ID {
		t.Fatalf("rotation calls = %#v", got)
	}
	if len(targets) != 3 || targets[0] == targets[1] || targets[1] == targets[2] {
		t.Fatalf("websocket route generations = %#v", targets)
	}
	if status := controller.Status(context.Background()); status.Counters.Requests != 0 || status.Counters.Exact402 != 0 {
		t.Fatalf("Egress pool unexpectedly observed Resin retry: %#v", status.Counters)
	}
	if auth.ProxyURL != "" || auth.Attributes[xaiResinLeaseGenerationAttribute] != "" {
		t.Fatalf("original auth mutated: %#v", auth)
	}
}

func TestXAIExecuteWithResinKeepsNetworkAnd402RetryBudgetsIndependent(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 1,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	auth := &cliproxyauth.Auth{ID: "auth-independent-budgets", Provider: "xai"}
	attempts := make([]*cliproxyauth.Auth, 0, 3)

	result, errExecute := executeXAIWithProxyPool(context.Background(), exec, auth, func(routed *cliproxyauth.Auth) (string, error) {
		attempts = append(attempts, routed)
		switch len(attempts) {
		case 1:
			return "", io.ErrUnexpectedEOF
		case 2:
			return "", blockedSpendingLimitError()
		default:
			return "ok", nil
		}
	})
	if errExecute != nil || result != "ok" || len(attempts) != 3 {
		t.Fatalf("result/error/attempts = %q/%v/%d", result, errExecute, len(attempts))
	}
	if attempts[0] != attempts[1] {
		t.Fatalf("network retry changed routed auth: %#v", attempts)
	}
	if attempts[2] == attempts[1] || xaiProxySessionTarget(attempts[2]) == xaiProxySessionTarget(attempts[1]) {
		t.Fatalf("402 retry did not rebuild the rotated route: %#v", attempts)
	}
	if got := resin.rotationCalls(); len(got) != 1 || got[0] != auth.ID {
		t.Fatalf("rotation calls = %#v", got)
	}
	for _, attempt := range attempts {
		if attempt.ID != auth.ID || attempt.ProxyURL != resin.proxyURL {
			t.Fatalf("retry changed auth identity or Account route: %#v", attempt)
		}
	}
}

func TestXAIExecuteWithResinStopsAtConfigured402RetryLimit(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	attempts := 0
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-limit", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", blockedSpendingLimitError()
	})
	requestScoped, ok := errExecute.(interface{ IsRequestScoped() bool })
	if !isXAIBlockedSpendingLimit(errExecute) || !ok || !requestScoped.IsRequestScoped() || attempts != 3 || len(resin.rotationCalls()) != 2 {
		t.Fatalf("error/attempts/rotations = %v/%d/%d", errExecute, attempts, len(resin.rotationCalls()))
	}
}

func TestXAIExecuteWithResinDoesNotRetryOther402(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	attempts := 0
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-other-402", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", xaiStatusErr(http.StatusPaymentRequired, []byte(`{"code":"other"}`))
	})
	if errExecute == nil || attempts != 1 || len(resin.rotationCalls()) != 0 {
		t.Fatalf("error/attempts/rotations = %v/%d/%d", errExecute, attempts, len(resin.rotationCalls()))
	}
}

func TestXAIExecuteWithResinLeaseRotationFailureIsRequestScoped(t *testing.T) {
	wantErr := &helps.XAIResinProxyError{Message: "rotation unavailable"}
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
		rotateErr:     wantErr,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	attempts := 0
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-rotate-error", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", blockedSpendingLimitError()
	})
	requestScoped, ok := errExecute.(interface{ IsRequestScoped() bool })
	if !errors.Is(errExecute, wantErr) || !ok || !requestScoped.IsRequestScoped() || attempts != 1 {
		t.Fatalf("error/attempts = %#v/%d", errExecute, attempts)
	}
}

func TestXAIHttpRequestSendsDerivedResinProxyAuthorization(t *testing.T) {
	wantCredential := "Default.xai-account:resin-token"
	wantAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(wantCredential))
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Proxy-Authorization"); got != wantAuthorization {
			t.Errorf("Proxy-Authorization = %q, want %q", got, wantAuthorization)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer proxyServer.Close()
	proxyURL, errParse := url.Parse(proxyServer.URL)
	if errParse != nil {
		t.Fatalf("parse proxy URL: %v", errParse)
	}
	proxyURL.User = url.UserPassword("Default.xai-account", "resin-token")

	exec := &XAIAutoExecutor{
		httpExec:   NewXAIExecutor(&config.Config{}),
		resinProxy: &executorResinProxy{proxyURL: proxyURL.String()},
	}
	req, errRequest := http.NewRequest(http.MethodGet, "http://upstream.invalid/v1/models", nil)
	if errRequest != nil {
		t.Fatalf("create request: %v", errRequest)
	}
	resp, errDo := exec.HttpRequest(context.Background(), &cliproxyauth.Auth{ID: "auth-http", Provider: "xai"}, req)
	if errDo != nil {
		t.Fatalf("HttpRequest() error = %v", errDo)
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			t.Errorf("close response: %v", errClose)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestXAIHttpRequestWithResinRetriesReplayableExact402(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	bodies := make([]string, 0, 2)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		attempts++
		attempt := attempts
		bodies = append(bodies, string(body))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if attempt == 1 {
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"code":"personal-team-blocked:spending-limit","error":"blocked"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer proxyServer.Close()
	proxyURL, errParse := url.Parse(proxyServer.URL)
	if errParse != nil {
		t.Fatalf("parse proxy URL: %v", errParse)
	}
	proxyURL.User = url.UserPassword("Default.xai-account", "resin-token")
	resin := &executorResinProxy{proxyURL: proxyURL.String(), max402Retries: 2}
	exec := &XAIAutoExecutor{httpExec: NewXAIExecutor(&config.Config{}), resinProxy: resin}
	req, errRequest := http.NewRequest(http.MethodPost, "http://upstream.invalid/v1/models", bytes.NewReader([]byte("same-body")))
	if errRequest != nil {
		t.Fatalf("create request: %v", errRequest)
	}
	resp, errDo := exec.HttpRequest(context.Background(), &cliproxyauth.Auth{ID: "auth-http-retry", Provider: "xai"}, req)
	if errDo != nil {
		t.Fatalf("HttpRequest() error = %v", errDo)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || attempts != 2 || len(resin.rotationCalls()) != 1 {
		t.Fatalf("status/attempts/rotations = %d/%d/%d", resp.StatusCode, attempts, len(resin.rotationCalls()))
	}
	if len(bodies) != 2 || bodies[0] != "same-body" || bodies[1] != "same-body" {
		t.Fatalf("request bodies = %#v", bodies)
	}
}

func TestXAIHttpRequestWithResinDoesNotRetryNonReplayableBody(t *testing.T) {
	attempts := 0
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"code":"personal-team-blocked:spending-limit","error":"blocked"}`))
	}))
	defer proxyServer.Close()
	resin := &executorResinProxy{proxyURL: proxyServer.URL, max402Retries: 2}
	exec := &XAIAutoExecutor{httpExec: NewXAIExecutor(&config.Config{}), resinProxy: resin}
	req, errRequest := http.NewRequest(http.MethodPost, "http://upstream.invalid/v1/models", io.NopCloser(strings.NewReader("one-shot")))
	if errRequest != nil {
		t.Fatalf("create request: %v", errRequest)
	}
	resp, errDo := exec.HttpRequest(context.Background(), &cliproxyauth.Auth{ID: "auth-http-one-shot", Provider: "xai"}, req)
	if errDo != nil {
		t.Fatalf("HttpRequest() error = %v", errDo)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusPaymentRequired || attempts != 1 || len(resin.rotationCalls()) != 0 {
		t.Fatalf("status/attempts/rotations = %d/%d/%d", resp.StatusCode, attempts, len(resin.rotationCalls()))
	}
}

func TestXAIRoutedAuthReturnsResinErrorWithoutFallback(t *testing.T) {
	wantErr := errors.New("Resin unavailable")
	pool, _, controller := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{
		proxyPool:  pool,
		resinProxy: &executorResinProxy{err: wantErr},
	}
	_, errRoute := exec.routedAuth(context.Background(), &cliproxyauth.Auth{ID: "auth-error", Provider: "xai"})
	if !errors.Is(errRoute, wantErr) {
		t.Fatalf("route error = %v", errRoute)
	}
	if status := controller.Status(context.Background()); status.Counters.Requests != 0 {
		t.Fatalf("Egress fallback occurred: %#v", status.Counters)
	}
}

func TestXAIResinWebsocketTargetDiffersPerAuth(t *testing.T) {
	resin := &executorResinProxy{proxyURLForAuth: func(authID string) string {
		return "http://Default." + authID + ":token@resin:2260"
	}}
	exec := &XAIAutoExecutor{resinProxy: resin}
	first, errFirst := exec.routedAuth(context.Background(), &cliproxyauth.Auth{ID: "xai-a", Provider: "xai"})
	second, errSecond := exec.routedAuth(context.Background(), &cliproxyauth.Auth{ID: "xai-b", Provider: "xai"})
	if errFirst != nil || errSecond != nil {
		t.Fatalf("route errors = %v, %v", errFirst, errSecond)
	}
	if xaiProxySessionTarget(first.auth) == xaiProxySessionTarget(second.auth) {
		t.Fatalf("Resin websocket targets unexpectedly match: %q", xaiProxySessionTarget(first.auth))
	}
}

func TestXAIRefreshRestoresProxyAfterResinRouting(t *testing.T) {
	resin := &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}
	exec := &XAIAutoExecutor{
		httpExec:   NewXAIExecutor(&config.Config{}),
		resinProxy: resin,
	}
	auth := &cliproxyauth.Auth{ID: "auth-refresh", Provider: "xai", Metadata: map[string]any{"type": "xai"}}

	refreshed, errRefresh := exec.Refresh(context.Background(), auth)
	if errRefresh != nil {
		t.Fatalf("Refresh() error = %v", errRefresh)
	}
	if refreshed == auth || refreshed.ProxyURL != "" || auth.ProxyURL != "" {
		t.Fatalf("refresh leaked transient Resin proxy: refreshed=%#v original=%#v", refreshed, auth)
	}
	if got := resin.calls(); len(got) != 1 || got[0] != auth.ID {
		t.Fatalf("Resin calls = %#v", got)
	}
}

func TestXAIRefreshWithResinRotatesLeaseAfterExact402(t *testing.T) {
	var attempts atomic.Int32
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt := attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if attempt == 1 {
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"code":"personal-team-blocked:spending-limit","error":"blocked"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","token_type":"Bearer","expires_in":3600}`))
	}))
	defer proxyServer.Close()
	proxyURL, errParse := url.Parse(proxyServer.URL)
	if errParse != nil {
		t.Fatalf("parse proxy URL: %v", errParse)
	}
	proxyURL.User = url.UserPassword("Default.xai-account", "resin-token")
	resin := &executorResinProxy{proxyURL: proxyURL.String(), max402Retries: 2}
	exec := &XAIAutoExecutor{httpExec: NewXAIExecutor(&config.Config{}), resinProxy: resin}
	auth := &cliproxyauth.Auth{
		ID:       "auth-refresh-402",
		Provider: "xai",
		Metadata: map[string]any{
			"type":           "xai",
			"refresh_token":  "resin-rotation-refresh-token",
			"token_endpoint": "http://upstream.invalid/oauth/token",
		},
	}

	refreshed, errRefresh := exec.Refresh(context.Background(), auth)
	if errRefresh != nil {
		t.Fatalf("Refresh() error = %v", errRefresh)
	}
	if attempts.Load() != 2 || len(resin.rotationCalls()) != 1 {
		t.Fatalf("attempts/rotations = %d/%d", attempts.Load(), len(resin.rotationCalls()))
	}
	if refreshed == nil || refreshed.ProxyURL != "" || refreshed.Metadata["access_token"] != "new-access" {
		t.Fatalf("refreshed auth = %#v", refreshed)
	}
}

func TestXAIResinNetworkFailureIsRequestScopedWithoutEgressFallback(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{
		proxyPool:  pool,
		resinProxy: &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"},
	}
	attempts := 0
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-network", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", io.ErrUnexpectedEOF
	})
	requestScoped, ok := errExecute.(interface{ IsRequestScoped() bool })
	if !ok || !requestScoped.IsRequestScoped() || !errors.Is(errExecute, io.ErrUnexpectedEOF) {
		t.Fatalf("network error = %#v", errExecute)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if status := controller.Status(context.Background()); status.Counters.Requests != 0 {
		t.Fatalf("Egress fallback occurred: %#v", status.Counters)
	}
}

func TestXAIResinNetworkFailureRetriesSameAuth(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{
		proxyPool:  pool,
		resinProxy: &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"},
	}
	original := &cliproxyauth.Auth{ID: "auth-retry", Provider: "xai"}
	attempts := make([]*cliproxyauth.Auth, 0, 2)
	result, errExecute := executeXAIWithProxyPool(context.Background(), exec, original, func(attemptAuth *cliproxyauth.Auth) (string, error) {
		attempts = append(attempts, attemptAuth)
		if len(attempts) == 1 {
			return "", io.ErrUnexpectedEOF
		}
		return "ok", nil
	})
	if errExecute != nil || result != "ok" {
		t.Fatalf("result/error = %q, %v", result, errExecute)
	}
	if len(attempts) != 2 || attempts[0] != attempts[1] || attempts[0] == original {
		t.Fatalf("attempt auths = %#v, want same routed clone twice", attempts)
	}
	if attempts[0].ProxyURL != "http://Default.xai-account:token@resin:2260" {
		t.Fatalf("retry proxy URL = %q", attempts[0].ProxyURL)
	}
	if status := controller.Status(context.Background()); status.Counters.Requests != 0 {
		t.Fatalf("Egress fallback occurred: %#v", status.Counters)
	}
}

func TestXAIResinCanceledContextDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exec := &XAIAutoExecutor{resinProxy: &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}}
	attempts := 0
	_, errExecute := executeXAIWithProxyPool(ctx, exec, &cliproxyauth.Auth{ID: "auth-canceled", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", context.DeadlineExceeded
	})
	if attempts != 1 || !errors.Is(errExecute, context.DeadlineExceeded) {
		t.Fatalf("attempts/error = %d, %v", attempts, errExecute)
	}
}

func TestXAIResinStreamNetworkFailureIsRequestScoped(t *testing.T) {
	exec := &XAIAutoExecutor{resinProxy: &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}}
	attempts := 0
	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-stream", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 1)
		chunks <- cliproxyexecutor.StreamChunk{Err: io.ErrUnexpectedEOF}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	chunk := <-stream.Chunks
	requestScoped, ok := chunk.Err.(interface{ IsRequestScoped() bool })
	if !ok || !requestScoped.IsRequestScoped() || !errors.Is(chunk.Err, io.ErrUnexpectedEOF) {
		t.Fatalf("stream network error = %#v", chunk.Err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestXAIResinStreamBootstrapFailureRetriesBeforePayload(t *testing.T) {
	exec := &XAIAutoExecutor{resinProxy: &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}}
	attempts := 0
	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-stream-retry", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 1)
		if attempts == 1 {
			chunks <- cliproxyexecutor.StreamChunk{Err: io.ErrUnexpectedEOF}
		} else {
			chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("ok")}
		}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	chunk := <-stream.Chunks
	if attempts != 2 || chunk.Err != nil || string(chunk.Payload) != "ok" {
		t.Fatalf("attempts/chunk = %d, %#v", attempts, chunk)
	}
}

func TestXAIResinStreamMidResponseFailureDoesNotRetry(t *testing.T) {
	exec := &XAIAutoExecutor{resinProxy: &executorResinProxy{proxyURL: "http://Default.xai-account:token@resin:2260"}}
	attempts := 0
	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-stream-mid", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 2)
		chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("started")}
		chunks <- cliproxyexecutor.StreamChunk{Err: io.ErrUnexpectedEOF}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	first := <-stream.Chunks
	second := <-stream.Chunks
	requestScoped, ok := second.Err.(interface{ IsRequestScoped() bool })
	if attempts != 1 || string(first.Payload) != "started" || !ok || !requestScoped.IsRequestScoped() || !errors.Is(second.Err, io.ErrUnexpectedEOF) {
		t.Fatalf("attempts/chunks = %d, %#v, %#v", attempts, first, second)
	}
}

func TestXAIResinHttpRequestRetriesProxyConnectFailureOnce(t *testing.T) {
	var attempts atomic.Int32
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer proxyServer.Close()
	proxyURL, errParse := url.Parse(proxyServer.URL)
	if errParse != nil {
		t.Fatalf("parse proxy URL: %v", errParse)
	}
	proxyURL.User = url.UserPassword("Default.xai-account", "resin-token")
	exec := &XAIAutoExecutor{
		httpExec:   NewXAIExecutor(&config.Config{}),
		resinProxy: &executorResinProxy{proxyURL: proxyURL.String()},
	}
	req, errRequest := http.NewRequest(http.MethodGet, "https://upstream.invalid/v1/models", nil)
	if errRequest != nil {
		t.Fatalf("create request: %v", errRequest)
	}
	_, errDo := exec.HttpRequest(context.Background(), &cliproxyauth.Auth{ID: "auth-http-retry", Provider: "xai"}, req)
	requestScoped, ok := errDo.(interface{ IsRequestScoped() bool })
	if attempts.Load() != 2 || !ok || !requestScoped.IsRequestScoped() {
		t.Fatalf("attempts/error = %d, %#v", attempts.Load(), errDo)
	}
}

func TestXAIResinStreamRetriesExact402OnlyDuringBootstrap(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	attempts := 0
	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-stream-retry", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 1)
		if attempts == 1 {
			chunks <- cliproxyexecutor.StreamChunk{Err: blockedSpendingLimitError()}
		} else {
			chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("data: ok\n\n")}
		}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	chunk := <-stream.Chunks
	if string(chunk.Payload) != "data: ok\n\n" || chunk.Err != nil || attempts != 2 || len(resin.rotationCalls()) != 1 {
		t.Fatalf("chunk/attempts/rotations = %#v/%d/%d", chunk, attempts, len(resin.rotationCalls()))
	}
}

func TestXAIResinStreamKeeps402AndNetworkRetryBudgetsIndependent(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 1,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	auth := &cliproxyauth.Auth{ID: "auth-stream-independent-budgets", Provider: "xai"}
	attempts := make([]*cliproxyauth.Auth, 0, 3)

	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), auth, func(routed *cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts = append(attempts, routed)
		chunks := make(chan cliproxyexecutor.StreamChunk, 1)
		switch len(attempts) {
		case 1:
			chunks <- cliproxyexecutor.StreamChunk{Err: blockedSpendingLimitError()}
		case 2:
			chunks <- cliproxyexecutor.StreamChunk{Err: io.ErrUnexpectedEOF}
		default:
			chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("ok")}
		}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	chunk := <-stream.Chunks
	if len(attempts) != 3 || chunk.Err != nil || string(chunk.Payload) != "ok" {
		t.Fatalf("attempts/chunk = %d/%#v", len(attempts), chunk)
	}
	if attempts[0] == attempts[1] || xaiProxySessionTarget(attempts[0]) == xaiProxySessionTarget(attempts[1]) {
		t.Fatalf("402 retry did not rebuild the rotated route: %#v", attempts)
	}
	if attempts[1] != attempts[2] {
		t.Fatalf("post-rotation network retry changed routed auth: %#v", attempts)
	}
	if got := resin.rotationCalls(); len(got) != 1 || got[0] != auth.ID {
		t.Fatalf("rotation calls = %#v", got)
	}
}

func TestXAIResinStreamRetriesExact402HandshakeError(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	attempts := 0
	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-websocket-handshake", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		if attempts == 1 {
			return nil, blockedSpendingLimitError()
		}
		chunks := make(chan cliproxyexecutor.StreamChunk, 1)
		chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("websocket-ok")}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	chunk := <-stream.Chunks
	if string(chunk.Payload) != "websocket-ok" || attempts != 2 || len(resin.rotationCalls()) != 1 {
		t.Fatalf("chunk/attempts/rotations = %#v/%d/%d", chunk, attempts, len(resin.rotationCalls()))
	}
}

func TestXAIResinStreamDoesNotReplayExact402AfterPayload(t *testing.T) {
	resin := &executorResinProxy{
		proxyURL:      "http://Default.xai-account:token@resin:2260",
		max402Retries: 2,
	}
	exec := &XAIAutoExecutor{resinProxy: resin}
	attempts := 0
	stream, errStream := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-stream-mid-response", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 2)
		chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("data: partial\n\n")}
		chunks <- cliproxyexecutor.StreamChunk{Err: blockedSpendingLimitError()}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errStream != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errStream)
	}
	var chunks []cliproxyexecutor.StreamChunk
	for chunk := range stream.Chunks {
		chunks = append(chunks, chunk)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %#v", chunks)
	}
	requestScoped, ok := chunks[1].Err.(interface{ IsRequestScoped() bool })
	if !isXAIBlockedSpendingLimit(chunks[1].Err) || !ok || !requestScoped.IsRequestScoped() || attempts != 1 || len(resin.rotationCalls()) != 0 {
		t.Fatalf("chunks/attempts/rotations = %#v/%d/%d", chunks, attempts, len(resin.rotationCalls()))
	}
}
