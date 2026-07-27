package executor

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type executorPoolController struct {
	mu       sync.Mutex
	nodes    []helps.XAIProxyNode
	egress   map[string]string
	healthy  map[string]bool
	selected map[string]string
}

func (f *executorPoolController) Snapshot(context.Context) ([]helps.XAIProxyNode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]helps.XAIProxyNode(nil), f.nodes...), nil
}

func (f *executorPoolController) RefreshProviders(context.Context) error { return nil }

func (f *executorPoolController) Select(_ context.Context, selector string, node string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.selected == nil {
		f.selected = make(map[string]string)
	}
	f.selected[selector] = node
	return nil
}

func (f *executorPoolController) CheckNode(_ context.Context, node string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.healthy == nil {
		return true, nil
	}
	healthy, exists := f.healthy[node]
	if !exists {
		return true, nil
	}
	return healthy, nil
}

func (f *executorPoolController) EgressIP(ctx context.Context, _ string, selector string, node string, _ []string) (string, error) {
	if errSelect := f.Select(ctx, selector, node); errSelect != nil {
		return "", errSelect
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.egress[node], nil
}

func newExecutorProxyPool(t *testing.T) (*helps.XAIProxyPool, config.XAIProxyPoolConfig, *executorPoolController) {
	t.Helper()
	cfg := config.XAIProxyPoolConfig{
		Enabled:                  true,
		RolloutPercent:           100,
		StateFile:                filepath.Join(t.TempDir(), "state.json"),
		HealthCheckURL:           "https://example.com/generate_204",
		HealthCheckTimeout:       "1s",
		IPCheckURLs:              []string{"https://example.com/ip"},
		ProviderRefreshInterval:  "1h",
		EgressRefreshInterval:    "1h",
		IPQuarantineDuration:     "24h",
		NodeQuarantineDuration:   "10m",
		NetworkFailureWindow:     "2m",
		NetworkFailureThreshold:  3,
		RequestsPerMinutePerLane: 60000,
		BurstPerLane:             100,
		QueueSizePerLane:         10,
		CandidateScanLimit:       8,
		Lanes: []config.XAIProxyPoolLane{
			{Name: "lane-1", ProxyURL: "http://mihomo:17891", Selector: "xai-lane-1"},
		},
		Probe: config.XAIProxyPoolLane{Name: "probe", ProxyURL: "http://mihomo:17899", Selector: "xai-probe"},
	}
	controller := &executorPoolController{
		nodes: []helps.XAIProxyNode{
			{Name: "node-1", Provider: "provider-a", Alive: true, Delay: 10},
			{Name: "node-2", Provider: "provider-b", Alive: true, Delay: 20},
		},
		egress: map[string]string{"node-1": "198.51.100.1", "node-2": "198.51.100.2"},
	}
	pool, errPool := helps.NewXAIProxyPoolWithController(context.Background(), cfg, controller, time.Now)
	if errPool != nil {
		t.Fatalf("NewXAIProxyPoolWithController() error = %v", errPool)
	}
	t.Cleanup(pool.Close)
	return pool, cfg, controller
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
	status := pool.Status()
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
	status := pool.Status()
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
	if status := pool.Status(); status.Counters.Requests != 0 {
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
	controller.healthy = map[string]bool{"node-1": false, "node-2": true}
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
	controller.healthy = map[string]bool{"node-1": false, "node-2": true}
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
	if status := pool.Status(); status.Counters.MidResponseFailures != 1 {
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
