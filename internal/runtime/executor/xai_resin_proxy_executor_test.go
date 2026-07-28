package executor

import (
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
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type executorResinProxy struct {
	mu              sync.Mutex
	proxyURL        string
	proxyURLForAuth func(string) string
	err             error
	authIDs         []string
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

func TestXAIExecuteWithResinDoesNotInvokeEgress402Probe(t *testing.T) {
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
