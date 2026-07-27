package helps

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

type fakeXAIProxyController struct {
	mu          sync.Mutex
	nodes       []xaiProxyNode
	egress      map[string]string
	healthy     map[string]bool
	selected    map[string]string
	refreshErr  error
	snapshotErr error
}

func (f *fakeXAIProxyController) Snapshot(context.Context) ([]xaiProxyNode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]xaiProxyNode(nil), f.nodes...), f.snapshotErr
}

func (f *fakeXAIProxyController) RefreshProviders(context.Context) error {
	return f.refreshErr
}

func (f *fakeXAIProxyController) Select(_ context.Context, selector string, node string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.selected == nil {
		f.selected = make(map[string]string)
	}
	f.selected[selector] = node
	return nil
}

func (f *fakeXAIProxyController) CheckNode(_ context.Context, node string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.healthy[node], nil
}

func (f *fakeXAIProxyController) EgressIP(ctx context.Context, _ string, selector string, node string, _ []string) (string, error) {
	if errSelect := f.Select(ctx, selector, node); errSelect != nil {
		return "", errSelect
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	ip := f.egress[node]
	if ip == "" {
		return "", errors.New("missing fake egress")
	}
	return ip, nil
}

func testXAIProxyPoolConfig(stateFile string, lanes int) config.XAIProxyPoolConfig {
	cfg := config.XAIProxyPoolConfig{
		Enabled:                  true,
		RolloutPercent:           100,
		StateFile:                stateFile,
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
		CandidateScanLimit:       32,
		Probe: config.XAIProxyPoolLane{
			Name: "probe", ProxyURL: "http://mihomo:17899", Selector: "xai-probe",
		},
	}
	for i := 1; i <= lanes; i++ {
		cfg.Lanes = append(cfg.Lanes, config.XAIProxyPoolLane{
			Name:     "lane-" + string(rune('0'+i)),
			ProxyURL: "http://mihomo:1789" + string(rune('0'+i)),
			Selector: "xai-lane-" + string(rune('0'+i)),
		})
	}
	return cfg
}

func initializedTestXAIProxyPool(t *testing.T, cfg config.XAIProxyPoolConfig, controller xaiProxyController, now func() time.Time) *xaiProxyPool {
	t.Helper()
	pool := newXAIProxyPoolWithController(cfg, controller, now, false)
	errInit := pool.initialize(context.Background())
	pool.finishInitialization(errInit)
	if errInit != nil {
		pool.Close()
		t.Fatalf("initialize() error = %v", errInit)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestXAIProxyPoolRendezvousHashMinimizesLaneMovement(t *testing.T) {
	six := newXAIProxyPoolWithController(testXAIProxyPoolConfig("", 6), nil, time.Now, false)
	eight := newXAIProxyPoolWithController(testXAIProxyPoolConfig("", 8), nil, time.Now, false)
	defer six.Close()
	defer eight.Close()
	moved := 0
	for i := 0; i < 1000; i++ {
		authID := "auth-" + time.Unix(int64(i), 0).UTC().Format(time.RFC3339)
		if six.rendezvousLane(authID) != eight.rendezvousLane(authID) {
			moved++
		}
	}
	if moved == 0 || moved >= 350 {
		t.Fatalf("moved auths = %d, want 1..349", moved)
	}
}

func TestXAIProxyPoolConfirmIPBlockPromotesAlternateAndPersists(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	stateFile := filepath.Join(t.TempDir(), "state.json")
	controller := &fakeXAIProxyController{
		nodes: []xaiProxyNode{
			{Name: "provider-a node-1", Provider: "provider-a", Alive: true, Delay: 10},
			{Name: "provider-b node-2", Provider: "provider-b", Alive: true, Delay: 20},
			{Name: "provider-c node-3", Provider: "provider-c", Alive: true, Delay: 30},
		},
		egress: map[string]string{
			"provider-a node-1": "198.51.100.1",
			"provider-b node-2": "198.51.100.2",
			"provider-c node-3": "198.51.100.3",
		},
		healthy: make(map[string]bool),
	}
	pool := initializedTestXAIProxyPool(t, testXAIProxyPoolConfig(stateFile, 2), controller, func() time.Time { return now })
	current, enrolled, errRoute := pool.Route(context.Background(), "auth-stable")
	if errRoute != nil || !enrolled {
		t.Fatalf("Route() = %#v, %v, %v", current, enrolled, errRoute)
	}
	lease, errProbe := pool.AcquireProbe(context.Background(), current)
	if errProbe != nil {
		t.Fatalf("AcquireProbe() error = %v", errProbe)
	}
	if lease.Route.EgressIP == current.EgressIP || lease.Route.Node == current.Node {
		lease.Release()
		t.Fatalf("alternate route = %#v, current = %#v", lease.Route, current)
	}
	observedRoute := current
	pool.ObserveMidResponseFailure(observedRoute)
	if errConfirm := lease.ConfirmIPBlock(context.Background()); errConfirm != nil {
		t.Fatalf("ConfirmIPBlock() error = %v", errConfirm)
	}
	status := pool.Status()
	if status.Counters.ABSuccess != 1 || status.Counters.Rotations != 1 {
		t.Fatalf("counters = %#v", status.Counters)
	}
	if len(status.IPQuarantines) != 1 || status.IPQuarantines[0].Value != current.EgressIP {
		t.Fatalf("IP quarantines = %#v", status.IPQuarantines)
	}
	if got := status.IPQuarantines[0].ExpiresAt; !got.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("quarantine expiry = %v", got)
	}

	data, errRead := os.ReadFile(stateFile)
	if errRead != nil {
		t.Fatalf("ReadFile(state) error = %v", errRead)
	}
	if !json.Valid(data) {
		t.Fatalf("state is not JSON: %s", data)
	}
	if string(data) == "" || containsAny(string(data), "auth-stable", "controller-secret", "subscription") {
		t.Fatalf("state leaked forbidden data: %s", data)
	}
	info, errStat := os.Stat(stateFile)
	if errStat != nil {
		t.Fatalf("Stat(state) error = %v", errStat)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %o", info.Mode().Perm())
	}
	reloaded := newXAIProxyPoolWithController(testXAIProxyPoolConfig(stateFile, 2), controller, func() time.Time { return now }, false)
	defer reloaded.Close()
	reloaded.mu.RLock()
	reloadedFailures := append([]time.Time(nil), reloaded.lanes[current.LaneName].MidFailures...)
	reloaded.mu.RUnlock()
	if len(reloadedFailures) != 1 || !reloadedFailures[0].Equal(now) {
		t.Fatalf("reloaded mid-response failures = %#v", reloadedFailures)
	}
}

func TestXAIProxyPoolRefreshRecoversFailedInitialization(t *testing.T) {
	controller := &fakeXAIProxyController{
		nodes:       []xaiProxyNode{{Name: "node-1", Provider: "provider-a", Alive: true, Delay: 10}},
		egress:      map[string]string{"node-1": "198.51.100.10"},
		snapshotErr: errors.New("provider cache unavailable"),
	}
	pool := newXAIProxyPoolWithController(testXAIProxyPoolConfig(filepath.Join(t.TempDir(), "state.json"), 1), controller, time.Now, false)
	defer pool.Close()
	errInit := pool.initialize(context.Background())
	pool.finishInitialization(errInit)
	if errInit == nil || pool.Status().Ready {
		t.Fatalf("initialization error/status = %v/%#v", errInit, pool.Status())
	}
	controller.mu.Lock()
	controller.snapshotErr = nil
	controller.mu.Unlock()
	if errRefresh := pool.RefreshProviders(context.Background()); errRefresh != nil {
		t.Fatalf("RefreshProviders() recovery error = %v", errRefresh)
	}
	if status := pool.Status(); !status.Ready || status.AvailableNodes != 1 {
		t.Fatalf("recovered status = %#v", status)
	}
}

func TestXAIProxyPoolRefreshReportsPartialProviderFailure(t *testing.T) {
	controller := &fakeXAIProxyController{
		nodes:      []xaiProxyNode{{Name: "node-1", Provider: "provider-a", Alive: true, Delay: 10}},
		egress:     map[string]string{"node-1": "198.51.100.10"},
		refreshErr: errors.New("provider-b refresh failed"),
	}
	pool := initializedTestXAIProxyPool(t, testXAIProxyPoolConfig(filepath.Join(t.TempDir(), "state.json"), 1), controller, time.Now)
	errRefresh := pool.RefreshProviders(context.Background())
	if errRefresh == nil || !strings.Contains(errRefresh.Error(), "provider-b refresh failed") {
		t.Fatalf("RefreshProviders() error = %v", errRefresh)
	}
	status := pool.Status()
	if !status.Ready || status.AvailableNodes != 1 || status.Counters.ProviderRefreshErrors != 1 {
		t.Fatalf("status after partial refresh = %#v", status)
	}
}

func TestXAIProxyPoolExplicitNodeFailureRotatesSeparately(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	controller := &fakeXAIProxyController{
		nodes: []xaiProxyNode{
			{Name: "node-1", Provider: "a", Alive: true, Delay: 10},
			{Name: "node-2", Provider: "b", Alive: true, Delay: 20},
		},
		egress:  map[string]string{"node-1": "198.51.100.10", "node-2": "198.51.100.20"},
		healthy: map[string]bool{"node-1": false, "node-2": true},
	}
	pool := initializedTestXAIProxyPool(t, testXAIProxyPoolConfig(filepath.Join(t.TempDir(), "state.json"), 1), controller, func() time.Time { return now })
	current, _, errRoute := pool.Route(context.Background(), "auth-1")
	if errRoute != nil {
		t.Fatalf("Route() error = %v", errRoute)
	}
	next, retry, errFailure := pool.HandlePreconnectFailure(context.Background(), current)
	if errFailure != nil || !retry {
		t.Fatalf("HandlePreconnectFailure() = %#v, %v, %v", next, retry, errFailure)
	}
	if next.Node == current.Node || next.EgressIP == current.EgressIP {
		t.Fatalf("node did not rotate: current=%#v next=%#v", current, next)
	}
	status := pool.Status()
	if len(status.NodeQuarantines) != 1 || len(status.IPQuarantines) != 0 {
		t.Fatalf("network quarantine state = nodes %#v IPs %#v", status.NodeQuarantines, status.IPQuarantines)
	}
}

func TestXAIProxyPoolMidResponseThresholdRotatesFutureRoute(t *testing.T) {
	controller := &fakeXAIProxyController{
		nodes: []xaiProxyNode{
			{Name: "node-1", Provider: "a", Alive: true, Delay: 10},
			{Name: "node-2", Provider: "b", Alive: true, Delay: 20},
		},
		egress: map[string]string{"node-1": "198.51.100.10", "node-2": "198.51.100.20"},
	}
	pool := initializedTestXAIProxyPool(t, testXAIProxyPoolConfig(filepath.Join(t.TempDir(), "state.json"), 1), controller, time.Now)
	current, _, errRoute := pool.Route(context.Background(), "auth-mid-response")
	if errRoute != nil {
		t.Fatalf("Route() error = %v", errRoute)
	}
	for range 3 {
		pool.ObserveMidResponseFailure(current)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		status := pool.Status()
		if status.Counters.Rotations == 1 {
			if status.Counters.MidResponseFailures != 3 || len(status.NodeQuarantines) != 1 || len(status.IPQuarantines) != 0 {
				t.Fatalf("status after mid-response rotation = %#v", status)
			}
			if status.Lanes[0].Node == current.Node {
				t.Fatalf("future route kept failed node: %#v", status.Lanes[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("mid-response rotation did not complete: %#v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestXAIProxyStartLimiterRejectsFullQueue(t *testing.T) {
	limiter := newXAIProxyStartLimiter(testXAIProxyPoolConfig("", 1), time.Now, nil)
	for i := 0; i < cap(limiter.queue); i++ {
		limiter.queue <- struct{}{}
	}
	errWait := limiter.Wait(context.Background())
	for i := 0; i < cap(limiter.queue); i++ {
		<-limiter.queue
	}
	var poolErr *XAIProxyPoolError
	if !errors.As(errWait, &poolErr) || poolErr.StatusCode() != 503 || !poolErr.IsRequestScoped() {
		t.Fatalf("Wait() error = %#v", errWait)
	}
	if got := poolErr.Headers().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want 1", got)
	}
}

func TestParsePublicIP(t *testing.T) {
	for _, input := range []string{
		"203.0.113.7",
		"ip=203.0.113.7\nloc=US\n",
		`{"ip":"203.0.113.7"}`,
	} {
		if got := parsePublicIP([]byte(input)); got != "203.0.113.7" {
			t.Fatalf("parsePublicIP(%q) = %q", input, got)
		}
	}
	if got := parsePublicIP([]byte("127.0.0.1")); got != "" {
		t.Fatalf("private IP accepted: %q", got)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
