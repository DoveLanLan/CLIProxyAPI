package executor

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type executorPoolConfig struct {
	Lanes []helps.XAIProxyRoute
	Probe helps.XAIProxyRoute
}

type executorProxyPool struct {
	mu              sync.Mutex
	route           helps.XAIProxyRoute
	alternate       helps.XAIProxyRoute
	status          helps.XAIProxyPoolStatus
	preconnectRetry bool
}

func (f *executorProxyPool) Route(context.Context, string) (helps.XAIProxyRoute, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Counters.Requests++
	return f.route, true, nil
}

func (f *executorProxyPool) AcquireProbe(_ context.Context, current helps.XAIProxyRoute) (helps.XAIProxyProbeLease, error) {
	return &executorProbeLease{pool: f, current: current, alternate: f.alternate}, nil
}

func (f *executorProxyPool) HandlePreconnectFailure(context.Context, helps.XAIProxyRoute) (helps.XAIProxyRoute, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Counters.PreconnectFailures++
	if !f.preconnectRetry {
		return helps.XAIProxyRoute{}, false, nil
	}
	f.route.Node = f.alternate.Node
	f.route.Provider = f.alternate.Provider
	f.route.EgressIP = f.alternate.EgressIP
	return f.route, true, nil
}

func (f *executorProxyPool) ObserveMidResponseFailure(context.Context, helps.XAIProxyRoute) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Counters.MidResponseFailures++
}

func (f *executorProxyPool) RecordExact402(context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Counters.Exact402++
}

func (f *executorProxyPool) Status(context.Context) helps.XAIProxyPoolStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *executorProxyPool) RefreshProviders(context.Context) error          { return nil }
func (f *executorProxyPool) RotateLane(context.Context, string) error        { return nil }
func (f *executorProxyPool) CheckLane(context.Context, string) (bool, error) { return true, nil }
func (f *executorProxyPool) QuarantineIP(context.Context, string) error      { return nil }
func (f *executorProxyPool) UnquarantineIP(context.Context, string) error    { return nil }
func (f *executorProxyPool) Close()                                          {}

func (f *executorProxyPool) XAIProxySubscriptions(context.Context) helps.XAIProxySubscriptionList {
	return helps.XAIProxySubscriptionList{Enabled: true, Ready: true, Subscriptions: []helps.XAIProxySubscriptionStatus{}}
}

func (f *executorProxyPool) CreateXAIProxySubscription(context.Context, uint64, helps.XAIProxySubscriptionCreate) (helps.XAIProxySubscriptionList, error) {
	return f.XAIProxySubscriptions(context.Background()), nil
}

func (f *executorProxyPool) UpdateXAIProxySubscription(context.Context, uint64, string, helps.XAIProxySubscriptionUpdate) (helps.XAIProxySubscriptionList, error) {
	return f.XAIProxySubscriptions(context.Background()), nil
}

func (f *executorProxyPool) DeleteXAIProxySubscription(context.Context, uint64, string) (helps.XAIProxySubscriptionList, error) {
	return f.XAIProxySubscriptions(context.Background()), nil
}

func (f *executorProxyPool) CheckXAIProxySubscription(context.Context, string) (helps.XAIProxySubscriptionStatus, error) {
	return helps.XAIProxySubscriptionStatus{}, nil
}

type executorProbeLease struct {
	pool      *executorProxyPool
	current   helps.XAIProxyRoute
	alternate helps.XAIProxyRoute
	once      sync.Once
}

func (l *executorProbeLease) CurrentRoute() helps.XAIProxyRoute   { return l.current }
func (l *executorProbeLease) AlternateRoute() helps.XAIProxyRoute { return l.alternate }

func (l *executorProbeLease) ConfirmIPBlock(context.Context) error {
	l.once.Do(func() {
		l.pool.mu.Lock()
		defer l.pool.mu.Unlock()
		l.pool.status.Counters.ABSuccess++
		l.pool.status.IPQuarantines = append(l.pool.status.IPQuarantines, helps.XAIProxyPoolQuarantineStatus{Value: l.current.EgressIP})
		l.pool.route.Node = l.alternate.Node
		l.pool.route.Provider = l.alternate.Provider
		l.pool.route.EgressIP = l.alternate.EgressIP
	})
	return nil
}

func (l *executorProbeLease) CredentialFailure() {
	l.once.Do(func() {
		l.pool.mu.Lock()
		defer l.pool.mu.Unlock()
		l.pool.status.Counters.ABCredentialFailure++
	})
}

func (l *executorProbeLease) Unavailable() {
	l.once.Do(func() {
		l.pool.mu.Lock()
		defer l.pool.mu.Unlock()
		l.pool.status.Counters.ABUnavailable++
	})
}

func (l *executorProbeLease) Release() { l.once.Do(func() {}) }

func newExecutorProxyPool(t *testing.T) (*executorProxyPool, executorPoolConfig, *executorProxyPool) {
	t.Helper()
	cfg := executorPoolConfig{
		Lanes: []helps.XAIProxyRoute{{LaneName: "lane-1", ProxyURL: "http://mihomo:17891", Node: "node-1", Provider: "provider-a", EgressIP: "198.51.100.1"}},
		Probe: helps.XAIProxyRoute{LaneName: "probe", ProxyURL: "http://mihomo:17899", Node: "node-2", Provider: "provider-b", EgressIP: "198.51.100.2", Probe: true},
	}
	pool := &executorProxyPool{
		route: cfg.Lanes[0], alternate: cfg.Probe,
		status: helps.XAIProxyPoolStatus{Enabled: true, Ready: true, IPQuarantines: []helps.XAIProxyPoolQuarantineStatus{}},
	}
	return pool, cfg, pool
}

func blockedSpendingLimitError() error {
	return xaiStatusErr(402, []byte(`{"code":"personal-team-blocked:spending-limit","error":"blocked"}`))
}

func TestExecuteXAIWithProxyPoolConfirmsBlockedIPAndRetriesSameAuth(t *testing.T) {
	pool, cfg, _ := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{proxyPool: pool}
	auth := &cliproxyauth.Auth{ID: "auth-1", Provider: "xai"}
	var proxies []string
	result, errExecute := executeXAIWithProxyPool(context.Background(), exec, auth, func(routed *cliproxyauth.Auth) (string, error) {
		proxies = append(proxies, routed.ProxyURL)
		if routed.ProxyURL == cfg.Lanes[0].ProxyURL {
			return "", blockedSpendingLimitError()
		}
		return "ok", nil
	})
	if errExecute != nil || result != "ok" {
		t.Fatalf("execute result/error = %q, %v", result, errExecute)
	}
	if len(proxies) != 2 || proxies[0] != cfg.Lanes[0].ProxyURL || proxies[1] != cfg.Probe.ProxyURL {
		t.Fatalf("proxy attempts = %#v", proxies)
	}
	if auth.ProxyURL != "" {
		t.Fatalf("original auth proxy mutated to %q", auth.ProxyURL)
	}
	status := pool.Status(context.Background())
	if status.Counters.Exact402 != 1 || status.Counters.ABSuccess != 1 || len(status.IPQuarantines) != 1 {
		t.Fatalf("pool status = %#v", status)
	}
}

func TestExecuteXAIWithProxyPoolRepeated402RemainsCredentialFailure(t *testing.T) {
	pool, _, _ := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{proxyPool: pool}
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-2", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		return "", blockedSpendingLimitError()
	})
	if !isXAIBlockedSpendingLimit(errExecute) {
		t.Fatalf("error = %#v, want exact 402", errExecute)
	}
	status := pool.Status(context.Background())
	if status.Counters.ABCredentialFailure != 1 || len(status.IPQuarantines) != 0 {
		t.Fatalf("pool status = %#v", status)
	}
}

func TestExecuteXAIWithProxyPoolExplicitAuthProxyBypassesPool(t *testing.T) {
	pool, _, _ := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{proxyPool: pool}
	auth := &cliproxyauth.Auth{ID: "auth-manual", Provider: "xai", ProxyURL: "http://manual-proxy:8080"}
	seen := ""
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, auth, func(routed *cliproxyauth.Auth) (string, error) {
		seen = routed.ProxyURL
		return "ok", nil
	})
	if errExecute != nil || seen != auth.ProxyURL {
		t.Fatalf("seen/error = %q, %v", seen, errExecute)
	}
	if status := pool.Status(context.Background()); status.Counters.Requests != 0 {
		t.Fatalf("pool unexpectedly routed explicit proxy: %#v", status.Counters)
	}
}

func TestXAIProxyPoolWebsocketTargetChangesWhenLaneNodeChanges(t *testing.T) {
	auth := &cliproxyauth.Auth{ID: "auth-websocket", Provider: "xai"}
	first := cloneXAIAuthWithRoute(auth, helps.XAIProxyRoute{
		LaneName: "lane-1", ProxyURL: "http://mihomo:17891", Node: "node-1",
	})
	second := cloneXAIAuthWithRoute(auth, helps.XAIProxyRoute{
		LaneName: "lane-1", ProxyURL: "http://mihomo:17891", Node: "node-2",
	})
	if xaiProxySessionTarget(first) == xaiProxySessionTarget(second) {
		t.Fatalf("websocket targets did not change: %q", xaiProxySessionTarget(first))
	}
	restored := restoreXAIAuthProxy(second, auth)
	if restored.ProxyURL != "" || restored.Attributes[xaiProxyRouteAttribute] != "" {
		t.Fatalf("restored auth leaked transient route: %#v", restored)
	}
}

func TestExecuteXAIWithProxyPoolSecondNetworkFailureIsRequestScoped(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	controller.mu.Lock()
	controller.preconnectRetry = true
	controller.mu.Unlock()
	exec := &XAIAutoExecutor{proxyPool: pool}
	attempts := 0
	_, errExecute := executeXAIWithProxyPool(context.Background(), exec, &cliproxyauth.Auth{ID: "auth-network", Provider: "xai"}, func(*cliproxyauth.Auth) (string, error) {
		attempts++
		return "", io.ErrUnexpectedEOF
	})
	requestScoped, ok := errExecute.(interface{ IsRequestScoped() bool })
	if attempts != 2 || !ok || !requestScoped.IsRequestScoped() || !errors.Is(errExecute, io.ErrUnexpectedEOF) {
		t.Fatalf("attempts/error = %d/%#v", attempts, errExecute)
	}
}

func TestXAIProxyPoolNetworkRetryStillObservesMidResponseFailure(t *testing.T) {
	pool, _, controller := newExecutorProxyPool(t)
	controller.mu.Lock()
	controller.preconnectRetry = true
	controller.mu.Unlock()
	exec := &XAIAutoExecutor{proxyPool: pool}
	attempts := 0
	stream, errExecute := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-stream-network", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		if attempts == 1 {
			return nil, io.ErrUnexpectedEOF
		}
		chunks := make(chan cliproxyexecutor.StreamChunk, 2)
		chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("data: partial\n\n")}
		chunks <- cliproxyexecutor.StreamChunk{Err: io.ErrUnexpectedEOF}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errExecute != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errExecute)
	}
	var terminal error
	for chunk := range stream.Chunks {
		if chunk.Err != nil {
			terminal = chunk.Err
		}
	}
	requestScoped, ok := terminal.(interface{ IsRequestScoped() bool })
	if attempts != 2 || !ok || !requestScoped.IsRequestScoped() || !errors.Is(terminal, io.ErrUnexpectedEOF) {
		t.Fatalf("attempts/terminal = %d/%#v", attempts, terminal)
	}
	if status := pool.Status(context.Background()); status.Counters.MidResponseFailures != 1 {
		t.Fatalf("mid-response counter = %#v", status.Counters)
	}
}

func TestXAIProxyPoolStreamRetriesOnlyBeforeFirstPayload(t *testing.T) {
	pool, cfg, _ := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{proxyPool: pool}
	auth := &cliproxyauth.Auth{ID: "auth-stream", Provider: "xai"}
	attempts := 0
	stream, errExecute := exec.executeStreamWithProxyPool(context.Background(), auth, func(routed *cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 1)
		if routed.ProxyURL == cfg.Lanes[0].ProxyURL {
			chunks <- cliproxyexecutor.StreamChunk{Err: blockedSpendingLimitError()}
		} else {
			chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("data: ok\n\n")}
		}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errExecute != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errExecute)
	}
	var payload []byte
	for chunk := range stream.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream error = %v", chunk.Err)
		}
		payload = append(payload, chunk.Payload...)
	}
	if attempts != 2 || string(payload) != "data: ok\n\n" {
		t.Fatalf("attempts/payload = %d/%q", attempts, payload)
	}
}

func TestXAIProxyPoolDoesNotReplayMidResponseNetworkFailure(t *testing.T) {
	pool, _, _ := newExecutorProxyPool(t)
	exec := &XAIAutoExecutor{proxyPool: pool}
	attempts := 0
	stream, errExecute := exec.executeStreamWithProxyPool(context.Background(), &cliproxyauth.Auth{ID: "auth-midstream", Provider: "xai"}, func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error) {
		attempts++
		chunks := make(chan cliproxyexecutor.StreamChunk, 2)
		chunks <- cliproxyexecutor.StreamChunk{Payload: []byte("data: partial\n\n")}
		chunks <- cliproxyexecutor.StreamChunk{Err: io.ErrUnexpectedEOF}
		close(chunks)
		return &cliproxyexecutor.StreamResult{Chunks: chunks}, nil
	})
	if errExecute != nil {
		t.Fatalf("executeStreamWithProxyPool() error = %v", errExecute)
	}
	var terminal error
	for chunk := range stream.Chunks {
		if chunk.Err != nil {
			terminal = chunk.Err
		}
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	requestScoped, ok := terminal.(interface{ IsRequestScoped() bool })
	if !ok || !requestScoped.IsRequestScoped() || !errors.Is(terminal, io.ErrUnexpectedEOF) {
		t.Fatalf("terminal error = %#v", terminal)
	}
}

func TestIsXAIBlockedSpendingLimitRequiresExactStatusAndCode(t *testing.T) {
	if !isXAIBlockedSpendingLimit(blockedSpendingLimitError()) {
		t.Fatal("exact spending-limit 402 not detected")
	}
	if isXAIBlockedSpendingLimit(xaiStatusErr(402, []byte(`{"code":"other"}`))) {
		t.Fatal("other 402 detected as IP block")
	}
	if isXAIBlockedSpendingLimit(xaiStatusErr(429, []byte(`{"code":"personal-team-blocked:spending-limit"}`))) {
		t.Fatal("wrong status detected as IP block")
	}
}
