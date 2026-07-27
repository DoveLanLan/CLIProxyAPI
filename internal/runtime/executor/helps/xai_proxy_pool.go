package helps

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/proxyutil"
	log "github.com/sirupsen/logrus"
)

const xaiProxyPoolStateVersion = 1

// XAIProxyPoolError is availability-neutral for credential scheduling. It
// represents a local proxy-pool condition, not an upstream credential failure.
type XAIProxyPoolError struct {
	Message string
	Retry   time.Duration
}

func (e *XAIProxyPoolError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *XAIProxyPoolError) StatusCode() int { return 503 }

func (e *XAIProxyPoolError) IsRequestScoped() bool { return true }

func (e *XAIProxyPoolError) RetryAfter() *time.Duration {
	if e == nil || e.Retry <= 0 {
		return nil
	}
	retry := e.Retry
	return &retry
}

func (e *XAIProxyPoolError) Headers() http.Header {
	if e == nil || e.Retry <= 0 {
		return nil
	}
	seconds := int64((e.Retry-1)/time.Second) + 1
	headers := make(http.Header)
	headers.Set("Retry-After", strconv.FormatInt(seconds, 10))
	return headers
}

// ManagedHeaders marks these headers as CPA-generated rather than upstream
// passthrough so clients receive retry guidance even when passthrough is off.
func (e *XAIProxyPoolError) ManagedHeaders() http.Header {
	return e.Headers()
}

// XAIProxyRoute is the non-sensitive routing decision for one xAI request.
type XAIProxyRoute struct {
	LaneName string `json:"lane"`
	ProxyURL string `json:"-"`
	Selector string `json:"-"`
	Node     string `json:"node"`
	Provider string `json:"provider"`
	EgressIP string `json:"egress_ip"`
	Probe    bool   `json:"probe"`
}

// XAIProxyPoolStatus is the redacted operator view exposed by Management API.
type XAIProxyPoolStatus struct {
	Enabled          bool                           `json:"enabled"`
	Ready            bool                           `json:"ready"`
	RolloutPercent   int                            `json:"rollout_percent"`
	LastError        string                         `json:"last_error,omitempty"`
	LastRefresh      time.Time                      `json:"last_refresh,omitempty"`
	Lanes            []XAIProxyPoolLaneStatus       `json:"lanes"`
	Providers        []XAIProxyPoolProviderStatus   `json:"providers"`
	IPQuarantines    []XAIProxyPoolQuarantineStatus `json:"ip_quarantines"`
	NodeQuarantines  []XAIProxyPoolQuarantineStatus `json:"node_quarantines"`
	Counters         XAIProxyPoolCounters           `json:"counters"`
	ConfiguredLanes  int                            `json:"configured_lanes"`
	AvailableNodes   int                            `json:"available_nodes"`
	StateFileEnabled bool                           `json:"state_file_enabled"`
}

type XAIProxyPoolLaneStatus struct {
	Name          string    `json:"name"`
	ProxyEndpoint string    `json:"proxy_endpoint"`
	Selector      string    `json:"selector"`
	Node          string    `json:"node,omitempty"`
	Provider      string    `json:"provider,omitempty"`
	EgressIP      string    `json:"egress_ip,omitempty"`
	Ready         bool      `json:"ready"`
	LastChanged   time.Time `json:"last_changed,omitempty"`
}

type XAIProxyPoolProviderStatus struct {
	Name           string `json:"name"`
	AvailableNodes int    `json:"available_nodes"`
}

type XAIProxyPoolQuarantineStatus struct {
	Value     string    `json:"value"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
}

type XAIProxyPoolCounters struct {
	Requests              uint64 `json:"requests"`
	QueueRejected         uint64 `json:"queue_rejected"`
	Exact402              uint64 `json:"exact_402"`
	ABSuccess             uint64 `json:"ab_success"`
	ABCredentialFailure   uint64 `json:"ab_credential_failure"`
	ABUnavailable         uint64 `json:"ab_unavailable"`
	Rotations             uint64 `json:"rotations"`
	PreconnectFailures    uint64 `json:"preconnect_failures"`
	MidResponseFailures   uint64 `json:"mid_response_failures"`
	ProviderRefreshErrors uint64 `json:"provider_refresh_errors"`
}

type xaiProxyPool struct {
	cfg        config.XAIProxyPoolConfig
	controller xaiProxyController
	now        func() time.Time

	mu              sync.RWMutex
	saveMu          sync.Mutex
	lifecycleMu     sync.Mutex
	lanes           map[string]*xaiProxyLaneRuntime
	laneOrder       []string
	providers       map[string]int
	candidates      []xaiProxyNode
	ipQuarantines   map[string]xaiProxyQuarantine
	nodeQuarantines map[string]xaiProxyQuarantine
	counters        XAIProxyPoolCounters
	ready           bool
	lastError       string
	lastRefresh     time.Time

	probeGate     chan struct{}
	probeLimiter  *xaiProxyStartLimiter
	initDone      chan struct{}
	initOnce      sync.Once
	initClose     sync.Once
	closed        atomic.Bool
	done          chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	subscriptions *xaiProxySubscriptionManager
}

// XAIProxyPool is the exported alias used by xAI executors and management code.
type XAIProxyPool = xaiProxyPool

type xaiProxyLaneRuntime struct {
	config.XAIProxyPoolLane
	Node            string
	Provider        string
	EgressIP        string
	Ready           bool
	LastChanged     time.Time
	Limiter         *xaiProxyStartLimiter
	MidFailures     []time.Time
	RotationPending bool
}

type xaiProxyQuarantine struct {
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
}

type xaiProxyPersistedState struct {
	Version         int                            `json:"version"`
	UpdatedAt       time.Time                      `json:"updated_at"`
	Lanes           map[string]xaiProxyPersistLane `json:"lanes"`
	IPQuarantines   map[string]xaiProxyQuarantine  `json:"ip_quarantines"`
	NodeQuarantines map[string]xaiProxyQuarantine  `json:"node_quarantines"`
	Counters        XAIProxyPoolCounters           `json:"counters"`
}

type xaiProxyPersistLane struct {
	Node        string      `json:"node"`
	Provider    string      `json:"provider"`
	EgressIP    string      `json:"egress_ip"`
	LastChanged time.Time   `json:"last_changed"`
	MidFailures []time.Time `json:"mid_failures,omitempty"`
}

type xaiProxyStartLimiter struct {
	mu       sync.Mutex
	rate     float64
	burst    float64
	tokens   float64
	last     time.Time
	queue    chan struct{}
	now      func() time.Time
	rejected func()
}

// XAIProxyProbeLease holds exclusive access to the shared probe selector until
// the alternate request reaches a conclusive pre-response result.
type XAIProxyProbeLease struct {
	pool        *xaiProxyPool
	Current     XAIProxyRoute
	Route       XAIProxyRoute
	promote     bool
	releaseOnce sync.Once
}

// NewXAIProxyPool creates an xAI proxy-pool runtime. Initialization runs in the
// background so server startup does not wait for subscription nodes.
func NewXAIProxyPool(cfg config.XAIProxyPoolConfig) *XAIProxyPool {
	if !cfg.Enabled && !cfg.SubscriptionManagement.Enabled {
		return newXAIProxyPoolWithController(cfg, nil, time.Now, false)
	}
	controller, errController := newMihomoXAIProxyController(cfg)
	pool := newXAIProxyPoolWithController(cfg, controller, time.Now, false)
	if errController != nil {
		pool.finishInitialization(errController)
		return pool
	}
	if cfg.Enabled {
		pool.startInitialization()
	} else if pool.subscriptions != nil {
		pool.startWorker(func() {
			if errReload := pool.subscriptions.reloadSourceOfTruth(pool.ctx); errReload != nil {
				pool.setLastError(errReload)
			}
		})
	}
	return pool
}

// NewXAIProxyPoolWithController creates and synchronously initializes a pool
// with a caller-supplied controller adapter. It is useful for embedded runtimes
// and deterministic integration tests.
func NewXAIProxyPoolWithController(ctx context.Context, cfg config.XAIProxyPoolConfig, controller XAIProxyController, now func() time.Time) (*XAIProxyPool, error) {
	pool := newXAIProxyPoolWithController(cfg, controller, now, false)
	errInit := pool.initialize(ctx)
	pool.finishInitialization(errInit)
	if errInit != nil {
		return pool, errInit
	}
	if cfg.Enabled {
		pool.startWorker(pool.runMaintenance)
	}
	return pool, nil
}

func newXAIProxyPoolWithController(cfg config.XAIProxyPoolConfig, controller xaiProxyController, now func() time.Time, initialize bool) *xaiProxyPool {
	if now == nil {
		now = time.Now
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool := &xaiProxyPool{
		cfg:             cfg,
		controller:      controller,
		now:             now,
		lanes:           make(map[string]*xaiProxyLaneRuntime, len(cfg.Lanes)),
		providers:       make(map[string]int),
		ipQuarantines:   make(map[string]xaiProxyQuarantine),
		nodeQuarantines: make(map[string]xaiProxyQuarantine),
		probeGate:       make(chan struct{}, 1),
		initDone:        make(chan struct{}),
		done:            make(chan struct{}),
		ctx:             ctx,
		cancel:          cancel,
	}
	pool.probeLimiter = newXAIProxyStartLimiter(cfg, now, func() {
		pool.mu.Lock()
		pool.counters.QueueRejected++
		pool.mu.Unlock()
	})
	for i := range cfg.Lanes {
		laneCfg := cfg.Lanes[i]
		lane := &xaiProxyLaneRuntime{XAIProxyPoolLane: laneCfg}
		lane.Limiter = newXAIProxyStartLimiter(cfg, now, func() {
			pool.mu.Lock()
			pool.counters.QueueRejected++
			pool.mu.Unlock()
		})
		pool.lanes[laneCfg.Name] = lane
		pool.laneOrder = append(pool.laneOrder, laneCfg.Name)
	}
	sort.Strings(pool.laneOrder)
	if errLoad := pool.loadState(); errLoad != nil {
		pool.lastError = errLoad.Error()
	}
	if !cfg.Enabled {
		pool.finishInitialization(nil)
	} else if initialize {
		pool.startInitialization()
	}
	pool.subscriptions = newXAIProxySubscriptionManager(cfg, controller, pool, now)
	return pool
}

func (p *xaiProxyPool) startInitialization() {
	if p == nil || !p.cfg.Enabled || p.closed.Load() {
		return
	}
	p.initOnce.Do(func() {
		if !p.startWorker(func() {
			ctx, cancel := context.WithTimeout(p.ctx, 2*time.Minute)
			errInit := p.initialize(ctx)
			cancel()
			p.finishInitialization(errInit)
			p.runMaintenance()
		}) {
			p.finishInitialization(poolUnavailable("xai proxy pool was closed during initialization"))
		}
	})
}

func (p *xaiProxyPool) finishInitialization(errInit error) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if errInit != nil {
		p.ready = false
		p.lastError = errInit.Error()
	} else if !p.cfg.Enabled {
		p.ready = false
	}
	p.mu.Unlock()
	p.initClose.Do(func() { close(p.initDone) })
}

func (p *xaiProxyPool) initialize(ctx context.Context) error {
	if p == nil {
		return nil
	}
	ctx = nonNilContext(ctx)
	if p.subscriptions != nil {
		if errReload := p.subscriptions.reloadSourceOfTruth(ctx); errReload != nil {
			return fmt.Errorf("xai proxy pool: restore subscription registry: %w", errReload)
		}
	}
	if !p.cfg.Enabled {
		return nil
	}
	if p.controller == nil {
		return fmt.Errorf("xai proxy pool: controller is unavailable")
	}
	if len(p.laneOrder) == 0 || p.cfg.Probe.ProxyURL == "" || p.cfg.Probe.Selector == "" {
		return fmt.Errorf("xai proxy pool: at least one lane and a probe route are required")
	}
	release, errAcquire := p.acquireProbeGate(ctx)
	if errAcquire != nil {
		return errAcquire
	}
	defer release()
	return p.initializeLocked(ctx)
}

func (p *xaiProxyPool) initializeLocked(ctx context.Context) error {
	nodes, errSnapshot := p.controller.Snapshot(ctx)
	if errSnapshot != nil {
		return errSnapshot
	}
	if errReconcile := p.reconcileLanes(ctx, nodes); errReconcile != nil {
		return errReconcile
	}
	p.mu.Lock()
	p.candidates = append([]xaiProxyNode(nil), nodes...)
	p.updateProviderCountsLocked(nodes)
	p.ready = true
	p.lastError = ""
	p.lastRefresh = p.now()
	p.mu.Unlock()
	return p.saveState()
}

func (p *xaiProxyPool) acquireProbeGate(ctx context.Context) (func(), error) {
	if p == nil {
		return nil, fmt.Errorf("xai proxy pool is unavailable")
	}
	ctx = nonNilContext(ctx)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p.probeGate <- struct{}{}:
		return func() { <-p.probeGate }, nil
	}
}

func (p *xaiProxyPool) reconcileLanes(ctx context.Context, nodes []xaiProxyNode) error {
	usedNodes := make(map[string]struct{}, len(p.laneOrder))
	usedIPs := make(map[string]struct{}, len(p.laneOrder))
	usedProviders := make(map[string]int)
	for _, laneName := range p.laneOrder {
		p.mu.RLock()
		lane := p.lanes[laneName]
		preferred := ""
		if lane != nil {
			preferred = lane.Node
		}
		p.mu.RUnlock()
		node, egressIP, errCandidate := p.resolveCandidate(ctx, nodes, preferred, usedNodes, usedIPs, usedProviders)
		if errCandidate != nil {
			return fmt.Errorf("xai proxy pool: initialize lane %q: %w", laneName, errCandidate)
		}
		if errSelect := p.controller.Select(ctx, lane.Selector, node.Name); errSelect != nil {
			return fmt.Errorf("xai proxy pool: select lane %q: %w", laneName, errSelect)
		}
		p.mu.Lock()
		lane.Node = node.Name
		lane.Provider = node.Provider
		lane.EgressIP = egressIP
		lane.Ready = true
		lane.LastChanged = p.now()
		p.mu.Unlock()
		usedNodes[node.Name] = struct{}{}
		usedIPs[egressIP] = struct{}{}
		usedProviders[node.Provider]++
	}
	return nil
}

func (p *xaiProxyPool) resolveCandidate(ctx context.Context, nodes []xaiProxyNode, preferred string, usedNodes map[string]struct{}, usedIPs map[string]struct{}, usedProviders map[string]int) (xaiProxyNode, string, error) {
	ordered := append([]xaiProxyNode(nil), nodes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Name == preferred {
			return true
		}
		if ordered[j].Name == preferred {
			return false
		}
		leftUsed := usedProviders[ordered[i].Provider]
		rightUsed := usedProviders[ordered[j].Provider]
		if leftUsed != rightUsed {
			return leftUsed < rightUsed
		}
		leftDelay, rightDelay := ordered[i].Delay, ordered[j].Delay
		if leftDelay == 0 {
			leftDelay = int(^uint(0) >> 1)
		}
		if rightDelay == 0 {
			rightDelay = int(^uint(0) >> 1)
		}
		if leftDelay != rightDelay {
			return leftDelay < rightDelay
		}
		return ordered[i].Name < ordered[j].Name
	})
	limit := p.cfg.CandidateScanLimit
	if limit <= 0 || limit > len(ordered) {
		limit = len(ordered)
	}
	now := p.now()
	var errs []error
	for i := 0; i < limit; i++ {
		node := ordered[i]
		if !node.Alive || node.Name == "" {
			continue
		}
		if _, exists := usedNodes[node.Name]; exists {
			continue
		}
		p.mu.Lock()
		p.pruneQuarantinesLocked(now)
		_, nodeBlocked := p.nodeQuarantines[node.Name]
		p.mu.Unlock()
		if nodeBlocked {
			continue
		}
		egressIP, errIP := p.controller.EgressIP(ctx, p.cfg.Probe.ProxyURL, p.cfg.Probe.Selector, node.Name, p.cfg.IPCheckURLs)
		if errIP != nil {
			errs = append(errs, fmt.Errorf("node %q: %w", node.Name, errIP))
			continue
		}
		p.mu.RLock()
		_, ipBlocked := p.ipQuarantines[egressIP]
		p.mu.RUnlock()
		if ipBlocked {
			continue
		}
		if _, exists := usedIPs[egressIP]; exists {
			continue
		}
		return node, egressIP, nil
	}
	if len(errs) > 0 {
		return xaiProxyNode{}, "", fmt.Errorf("no unique healthy candidate: %w", errors.Join(errs...))
	}
	return xaiProxyNode{}, "", fmt.Errorf("no unique healthy candidate")
}

// Route returns the stable lane for an auth ID. The bool is false when staged
// rollout leaves the auth on legacy routing.
func (p *xaiProxyPool) Route(ctx context.Context, authID string) (XAIProxyRoute, bool, error) {
	if p == nil || !p.cfg.Enabled {
		return XAIProxyRoute{}, false, nil
	}
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return XAIProxyRoute{}, true, poolUnavailable("xai proxy pool requires a stable auth ID")
	}
	if !p.enrolled(authID) {
		return XAIProxyRoute{}, false, nil
	}
	if errReady := p.waitReady(ctx); errReady != nil {
		return XAIProxyRoute{}, true, errReady
	}
	laneName := p.rendezvousLane(authID)
	p.mu.RLock()
	lane := p.lanes[laneName]
	if lane == nil || !lane.Ready || lane.Node == "" || lane.EgressIP == "" {
		p.mu.RUnlock()
		return XAIProxyRoute{}, true, poolUnavailable("xai proxy pool lane is unavailable")
	}
	route := lane.route()
	limiter := lane.Limiter
	p.mu.RUnlock()
	if limiter != nil {
		if errWait := limiter.Wait(ctx); errWait != nil {
			return XAIProxyRoute{}, true, errWait
		}
	}
	p.mu.Lock()
	p.counters.Requests++
	p.mu.Unlock()
	return route, true, nil
}

func (p *xaiProxyPool) waitReady(ctx context.Context) error {
	if p == nil {
		return poolUnavailable("xai proxy pool is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.initDone:
	}
	p.mu.RLock()
	ready := p.ready
	lastError := p.lastError
	p.mu.RUnlock()
	if ready {
		return nil
	}
	if lastError == "" {
		lastError = "xai proxy pool is not ready"
	}
	return poolUnavailable(lastError)
}

func (p *xaiProxyPool) enrolled(authID string) bool {
	percent := p.cfg.RolloutPercent
	if percent >= 100 {
		return true
	}
	if percent <= 0 {
		return false
	}
	sum := sha256.Sum256([]byte("xai-proxy-rollout\x00" + authID))
	bucket := binary.BigEndian.Uint64(sum[:8]) % 100
	return int(bucket) < percent
}

func (p *xaiProxyPool) rendezvousLane(authID string) string {
	bestName := ""
	var bestScore uint64
	for _, laneName := range p.laneOrder {
		sum := sha256.Sum256([]byte(authID + "\x00" + laneName))
		score := binary.BigEndian.Uint64(sum[:8])
		if bestName == "" || score > bestScore {
			bestName = laneName
			bestScore = score
		}
	}
	return bestName
}

func (lane *xaiProxyLaneRuntime) route() XAIProxyRoute {
	if lane == nil {
		return XAIProxyRoute{}
	}
	return XAIProxyRoute{
		LaneName: lane.Name,
		ProxyURL: lane.ProxyURL,
		Selector: lane.Selector,
		Node:     lane.Node,
		Provider: lane.Provider,
		EgressIP: lane.EgressIP,
	}
}

// AcquireProbe chooses a verified alternate egress and holds the shared probe
// selector until the caller confirms or aborts the A/B result.
func (p *xaiProxyPool) AcquireProbe(ctx context.Context, current XAIProxyRoute) (*XAIProxyProbeLease, error) {
	if errReady := p.waitReady(ctx); errReady != nil {
		return nil, errReady
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p.probeGate <- struct{}{}:
	}
	releaseOnError := true
	defer func() {
		if releaseOnError {
			<-p.probeGate
		}
	}()

	p.mu.RLock()
	if lane := p.lanes[current.LaneName]; lane != nil && lane.Ready && lane.EgressIP != "" && lane.EgressIP != current.EgressIP {
		route := lane.route()
		limiter := lane.Limiter
		p.mu.RUnlock()
		if limiter != nil {
			if errWait := limiter.Wait(ctx); errWait != nil {
				return nil, errWait
			}
		}
		releaseOnError = false
		return &XAIProxyProbeLease{pool: p, Current: current, Route: route}, nil
	}
	nodes := append([]xaiProxyNode(nil), p.candidates...)
	p.mu.RUnlock()
	if len(nodes) == 0 {
		var errSnapshot error
		nodes, errSnapshot = p.controller.Snapshot(ctx)
		if errSnapshot != nil {
			return nil, poolUnavailable("xai proxy pool could not list alternate nodes")
		}
	}
	usedNodes, usedIPs, usedProviders := p.activeSets(current)
	node, egressIP, errCandidate := p.resolveCandidate(ctx, nodes, "", usedNodes, usedIPs, usedProviders)
	if errCandidate != nil {
		return nil, poolUnavailable("xai proxy pool has no verified alternate egress")
	}
	limiter := p.probeLimiter
	if errWait := limiter.Wait(ctx); errWait != nil {
		return nil, errWait
	}
	releaseOnError = false
	return &XAIProxyProbeLease{
		pool:    p,
		Current: current,
		Route: XAIProxyRoute{
			LaneName: p.cfg.Probe.Name,
			ProxyURL: p.cfg.Probe.ProxyURL,
			Selector: p.cfg.Probe.Selector,
			Node:     node.Name,
			Provider: node.Provider,
			EgressIP: egressIP,
			Probe:    true,
		},
		promote: true,
	}, nil
}

func (p *xaiProxyPool) activeSets(exclude XAIProxyRoute) (map[string]struct{}, map[string]struct{}, map[string]int) {
	usedNodes := make(map[string]struct{})
	usedIPs := make(map[string]struct{})
	usedProviders := make(map[string]int)
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, lane := range p.lanes {
		if lane == nil || !lane.Ready {
			continue
		}
		if lane.Name == exclude.LaneName && lane.Node == exclude.Node {
			continue
		}
		usedNodes[lane.Node] = struct{}{}
		usedIPs[lane.EgressIP] = struct{}{}
		usedProviders[lane.Provider]++
	}
	usedNodes[exclude.Node] = struct{}{}
	usedIPs[exclude.EgressIP] = struct{}{}
	return usedNodes, usedIPs, usedProviders
}

// ConfirmIPBlock promotes the alternate node, quarantines the original public
// IP, and releases the probe selector.
func (l *XAIProxyProbeLease) ConfirmIPBlock(ctx context.Context) error {
	if l == nil || l.pool == nil {
		return fmt.Errorf("xai proxy pool: probe lease is nil")
	}
	defer l.Release()
	p := l.pool
	if l.promote {
		p.mu.RLock()
		lane := p.lanes[l.Current.LaneName]
		p.mu.RUnlock()
		if lane == nil {
			return fmt.Errorf("xai proxy pool: lane %q no longer exists", l.Current.LaneName)
		}
		if errSelect := p.controller.Select(ctx, lane.Selector, l.Route.Node); errSelect != nil {
			return fmt.Errorf("xai proxy pool: promote alternate node: %w", errSelect)
		}
		p.mu.Lock()
		lane.Node = l.Route.Node
		lane.Provider = l.Route.Provider
		lane.EgressIP = l.Route.EgressIP
		lane.Ready = true
		lane.LastChanged = p.now()
		p.counters.Rotations++
		p.mu.Unlock()
	}
	p.mu.Lock()
	if l.Current.EgressIP != "" {
		p.ipQuarantines[l.Current.EgressIP] = xaiProxyQuarantine{
			Reason:    "xai_402_confirmed",
			ExpiresAt: p.now().Add(parsePositiveDuration(p.cfg.IPQuarantineDuration, 24*time.Hour)),
		}
	}
	p.counters.ABSuccess++
	p.lastError = ""
	p.mu.Unlock()
	if errSave := p.saveState(); errSave != nil {
		p.setLastError(errSave)
		log.WithError(errSave).Warn("xai proxy pool state persistence failed after IP rotation")
	}
	return nil
}

func (l *XAIProxyProbeLease) CredentialFailure() {
	if l == nil || l.pool == nil {
		return
	}
	l.pool.mu.Lock()
	l.pool.counters.ABCredentialFailure++
	l.pool.mu.Unlock()
	l.Release()
}

func (l *XAIProxyProbeLease) Unavailable() {
	if l == nil || l.pool == nil {
		return
	}
	l.pool.mu.Lock()
	l.pool.counters.ABUnavailable++
	l.pool.mu.Unlock()
	l.Release()
}

func (l *XAIProxyProbeLease) Release() {
	if l == nil || l.pool == nil {
		return
	}
	l.releaseOnce.Do(func() { <-l.pool.probeGate })
}

func (p *xaiProxyPool) RecordExact402() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.counters.Exact402++
	p.mu.Unlock()
}

// HandlePreconnectFailure rotates and returns a retry route only after Mihomo
// confirms the selected node is unhealthy.
func (p *xaiProxyPool) HandlePreconnectFailure(ctx context.Context, route XAIProxyRoute) (XAIProxyRoute, bool, error) {
	if p == nil || route.Node == "" {
		return XAIProxyRoute{}, false, nil
	}
	p.mu.Lock()
	p.counters.PreconnectFailures++
	p.mu.Unlock()
	healthy, errCheck := p.controller.CheckNode(ctx, route.Node)
	if errCheck != nil {
		return XAIProxyRoute{}, false, poolUnavailable("xai proxy node health could not be verified")
	}
	if healthy {
		return XAIProxyRoute{}, false, nil
	}
	if errRotate := p.rotateLane(ctx, route.LaneName, "network_unhealthy", route.Node); errRotate != nil {
		return XAIProxyRoute{}, false, poolUnavailable("xai proxy lane could not replace an unhealthy node")
	}
	p.mu.RLock()
	lane := p.lanes[route.LaneName]
	if lane == nil || !lane.Ready {
		p.mu.RUnlock()
		return XAIProxyRoute{}, false, poolUnavailable("xai proxy lane is unavailable after rotation")
	}
	next := lane.route()
	limiter := lane.Limiter
	p.mu.RUnlock()
	if limiter != nil {
		if errWait := limiter.Wait(ctx); errWait != nil {
			return XAIProxyRoute{}, false, errWait
		}
	}
	return next, true, nil
}

func (p *xaiProxyPool) ObserveMidResponseFailure(route XAIProxyRoute) {
	if p == nil || route.LaneName == "" || p.closed.Load() {
		return
	}
	now := p.now()
	window := parsePositiveDuration(p.cfg.NetworkFailureWindow, 2*time.Minute)
	threshold := p.cfg.NetworkFailureThreshold
	if threshold <= 0 {
		threshold = 3
	}
	shouldRotate := false
	p.mu.Lock()
	p.counters.MidResponseFailures++
	lane := p.lanes[route.LaneName]
	if lane != nil {
		kept := lane.MidFailures[:0]
		for _, occurred := range lane.MidFailures {
			if now.Sub(occurred) <= window {
				kept = append(kept, occurred)
			}
		}
		lane.MidFailures = append(kept, now)
		if len(lane.MidFailures) >= threshold && !lane.RotationPending {
			lane.RotationPending = true
			shouldRotate = true
		}
	}
	p.mu.Unlock()
	if !shouldRotate {
		if errSave := p.saveState(); errSave != nil {
			p.setLastError(errSave)
		}
		return
	}
	if !p.startWorker(func() {
		ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
		errRotate := p.rotateLane(ctx, route.LaneName, "mid_response_failures", route.Node)
		cancel()
		p.mu.Lock()
		if lane := p.lanes[route.LaneName]; lane != nil {
			lane.RotationPending = false
			lane.MidFailures = nil
		}
		p.mu.Unlock()
		if errRotate != nil {
			p.setLastError(errRotate)
		}
		if errSave := p.saveState(); errSave != nil {
			p.setLastError(errSave)
		}
	}) {
		p.mu.Lock()
		if lane := p.lanes[route.LaneName]; lane != nil {
			lane.RotationPending = false
		}
		p.mu.Unlock()
	}
}

func (p *xaiProxyPool) rotateLane(ctx context.Context, laneName string, reason string, failedNode string) error {
	if p == nil {
		return fmt.Errorf("xai proxy pool is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.probeGate <- struct{}{}:
	}
	defer func() { <-p.probeGate }()
	p.mu.RLock()
	lane := p.lanes[laneName]
	nodes := append([]xaiProxyNode(nil), p.candidates...)
	current := XAIProxyRoute{}
	if lane != nil {
		current = lane.route()
	}
	p.mu.RUnlock()
	if lane == nil {
		return fmt.Errorf("xai proxy pool: unknown lane %q", laneName)
	}
	if failedNode != "" {
		p.mu.Lock()
		p.nodeQuarantines[failedNode] = xaiProxyQuarantine{
			Reason:    reason,
			ExpiresAt: p.now().Add(parsePositiveDuration(p.cfg.NodeQuarantineDuration, 10*time.Minute)),
		}
		p.mu.Unlock()
	}
	if len(nodes) == 0 {
		var errSnapshot error
		nodes, errSnapshot = p.controller.Snapshot(ctx)
		if errSnapshot != nil {
			return errSnapshot
		}
	}
	usedNodes, usedIPs, usedProviders := p.activeSets(current)
	node, egressIP, errCandidate := p.resolveCandidate(ctx, nodes, "", usedNodes, usedIPs, usedProviders)
	if errCandidate != nil {
		p.mu.Lock()
		lane.Ready = false
		p.mu.Unlock()
		return errCandidate
	}
	if errSelect := p.controller.Select(ctx, lane.Selector, node.Name); errSelect != nil {
		return errSelect
	}
	p.mu.Lock()
	lane.Node = node.Name
	lane.Provider = node.Provider
	lane.EgressIP = egressIP
	lane.Ready = true
	lane.LastChanged = p.now()
	p.counters.Rotations++
	p.mu.Unlock()
	if errSave := p.saveState(); errSave != nil {
		p.setLastError(errSave)
	}
	return nil
}

func (p *xaiProxyPool) RefreshProviders(ctx context.Context) error {
	if p == nil || !p.cfg.Enabled || p.controller == nil {
		return poolUnavailable("xai proxy pool controller is unavailable")
	}
	ctx = nonNilContext(ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.initDone:
	}
	release, errAcquire := p.acquireProbeGate(ctx)
	if errAcquire != nil {
		return errAcquire
	}
	defer release()
	errRefresh := p.controller.RefreshProviders(ctx)
	if errRefresh != nil {
		p.mu.Lock()
		p.counters.ProviderRefreshErrors++
		p.lastError = errRefresh.Error()
		p.mu.Unlock()
		// Mihomo retains its last good provider cache; continue with a snapshot.
	}
	p.mu.RLock()
	ready := p.ready
	p.mu.RUnlock()
	if !ready {
		errInit := p.initializeLocked(ctx)
		p.finishInitialization(errInit)
		if errCombined := errors.Join(errRefresh, errInit); errCombined != nil {
			p.setLastError(errCombined)
			return errCombined
		}
		return nil
	}
	nodes, errSnapshot := p.controller.Snapshot(ctx)
	if errSnapshot != nil {
		errCombined := errors.Join(errRefresh, errSnapshot)
		p.setLastError(errCombined)
		return errCombined
	}
	p.mu.Lock()
	p.candidates = append([]xaiProxyNode(nil), nodes...)
	p.updateProviderCountsLocked(nodes)
	p.lastRefresh = p.now()
	p.mu.Unlock()
	errEgress := p.refreshActiveEgressLocked(ctx)
	errCombined := errors.Join(errRefresh, errEgress)
	if errCombined != nil {
		p.setLastError(errCombined)
	}
	return errCombined
}

func (p *xaiProxyPool) RotateLane(ctx context.Context, laneName string) error {
	return p.rotateLane(ctx, strings.TrimSpace(laneName), "manual_rotation", "")
}

func (p *xaiProxyPool) CheckLane(ctx context.Context, laneName string) (bool, error) {
	p.mu.RLock()
	lane := p.lanes[strings.TrimSpace(laneName)]
	if lane == nil {
		p.mu.RUnlock()
		return false, fmt.Errorf("xai proxy pool: unknown lane %q", laneName)
	}
	node := lane.Node
	p.mu.RUnlock()
	healthy, errCheck := p.controller.CheckNode(ctx, node)
	if errCheck != nil || healthy {
		return healthy, errCheck
	}
	return false, p.rotateLane(ctx, laneName, "manual_health_check", node)
}

func (p *xaiProxyPool) QuarantineIP(ctx context.Context, rawIP string) error {
	ip := normalizePublicIP(rawIP)
	if ip == "" {
		return fmt.Errorf("xai proxy pool: invalid public IP")
	}
	p.mu.Lock()
	p.ipQuarantines[ip] = xaiProxyQuarantine{
		Reason:    "manual",
		ExpiresAt: p.now().Add(parsePositiveDuration(p.cfg.IPQuarantineDuration, 24*time.Hour)),
	}
	var affected []string
	for name, lane := range p.lanes {
		if lane != nil && lane.EgressIP == ip {
			affected = append(affected, name)
		}
	}
	p.mu.Unlock()
	var errs []error
	for _, laneName := range affected {
		if errRotate := p.rotateLane(ctx, laneName, "manual_ip_quarantine", ""); errRotate != nil {
			errs = append(errs, errRotate)
		}
	}
	if errSave := p.saveState(); errSave != nil {
		errs = append(errs, errSave)
	}
	return errors.Join(errs...)
}

func (p *xaiProxyPool) UnquarantineIP(rawIP string) error {
	ip := net.ParseIP(strings.TrimSpace(rawIP))
	if ip == nil {
		return fmt.Errorf("xai proxy pool: invalid IP")
	}
	p.mu.Lock()
	delete(p.ipQuarantines, ip.String())
	p.mu.Unlock()
	return p.saveState()
}

func (p *xaiProxyPool) XAIProxySubscriptions(ctx context.Context) XAIProxySubscriptionList {
	if p == nil || p.subscriptions == nil {
		return XAIProxySubscriptionList{}
	}
	return p.subscriptions.List(ctx)
}

func (p *xaiProxyPool) CreateXAIProxySubscription(ctx context.Context, expectedRevision uint64, input XAIProxySubscriptionCreate) (XAIProxySubscriptionList, error) {
	if p == nil || p.subscriptions == nil {
		return XAIProxySubscriptionList{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	return p.subscriptions.Create(ctx, expectedRevision, input)
}

func (p *xaiProxyPool) UpdateXAIProxySubscription(ctx context.Context, expectedRevision uint64, name string, input XAIProxySubscriptionUpdate) (XAIProxySubscriptionList, error) {
	if p == nil || p.subscriptions == nil {
		return XAIProxySubscriptionList{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	return p.subscriptions.Update(ctx, expectedRevision, name, input)
}

func (p *xaiProxyPool) DeleteXAIProxySubscription(ctx context.Context, expectedRevision uint64, name string) (XAIProxySubscriptionList, error) {
	if p == nil || p.subscriptions == nil {
		return XAIProxySubscriptionList{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	return p.subscriptions.Delete(ctx, expectedRevision, name)
}

func (p *xaiProxyPool) CheckXAIProxySubscription(ctx context.Context, name string) (XAIProxySubscriptionStatus, error) {
	if p == nil || p.subscriptions == nil {
		return XAIProxySubscriptionStatus{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	return p.subscriptions.Check(ctx, name)
}

func (p *xaiProxyPool) providerInUse(provider string) bool {
	if p == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, lane := range p.lanes {
		if lane != nil && lane.Ready && strings.EqualFold(lane.Provider, provider) {
			return true
		}
	}
	return false
}

func (p *xaiProxyPool) reconcileSubscriptionSnapshot(ctx context.Context, nodes []xaiProxyNode) error {
	if p == nil || !p.cfg.Enabled {
		return nil
	}
	release, errAcquire := p.acquireProbeGate(ctx)
	if errAcquire != nil {
		return errAcquire
	}
	defer release()
	return p.reconcileSubscriptionSnapshotLocked(ctx, nodes)
}

func (p *xaiProxyPool) reconcileSubscriptionSnapshotLocked(ctx context.Context, nodes []xaiProxyNode) error {
	ctx = nonNilContext(ctx)
	if errReconcile := p.reconcileLanes(ctx, nodes); errReconcile != nil {
		return errReconcile
	}
	p.mu.Lock()
	p.candidates = append([]xaiProxyNode(nil), nodes...)
	p.updateProviderCountsLocked(nodes)
	p.ready = true
	p.lastError = ""
	p.lastRefresh = p.now()
	p.mu.Unlock()
	return p.saveState()
}

func (p *xaiProxyPool) Status() XAIProxyPoolStatus {
	if p == nil {
		return XAIProxyPoolStatus{}
	}
	now := p.now()
	p.mu.Lock()
	p.pruneQuarantinesLocked(now)
	status := XAIProxyPoolStatus{
		Enabled:          p.cfg.Enabled,
		Ready:            p.ready,
		RolloutPercent:   p.cfg.RolloutPercent,
		LastError:        p.lastError,
		LastRefresh:      p.lastRefresh,
		Counters:         p.counters,
		ConfiguredLanes:  len(p.laneOrder),
		AvailableNodes:   len(p.candidates),
		StateFileEnabled: strings.TrimSpace(p.cfg.StateFile) != "",
	}
	for _, laneName := range p.laneOrder {
		lane := p.lanes[laneName]
		if lane == nil {
			continue
		}
		status.Lanes = append(status.Lanes, XAIProxyPoolLaneStatus{
			Name:          lane.Name,
			ProxyEndpoint: proxyutil.Redact(lane.ProxyURL),
			Selector:      lane.Selector,
			Node:          lane.Node,
			Provider:      lane.Provider,
			EgressIP:      lane.EgressIP,
			Ready:         lane.Ready,
			LastChanged:   lane.LastChanged,
		})
	}
	providerNames := make([]string, 0, len(p.providers))
	for name := range p.providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)
	for _, name := range providerNames {
		status.Providers = append(status.Providers, XAIProxyPoolProviderStatus{Name: name, AvailableNodes: p.providers[name]})
	}
	status.IPQuarantines = quarantineStatuses(p.ipQuarantines)
	status.NodeQuarantines = quarantineStatuses(p.nodeQuarantines)
	p.mu.Unlock()
	return status
}

func quarantineStatuses(values map[string]xaiProxyQuarantine) []XAIProxyPoolQuarantineStatus {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	out := make([]XAIProxyPoolQuarantineStatus, 0, len(keys))
	for _, value := range keys {
		entry := values[value]
		out = append(out, XAIProxyPoolQuarantineStatus{Value: value, Reason: entry.Reason, ExpiresAt: entry.ExpiresAt})
	}
	return out
}

func (p *xaiProxyPool) updateProviderCountsLocked(nodes []xaiProxyNode) {
	p.providers = make(map[string]int)
	for _, node := range nodes {
		if node.Alive {
			p.providers[node.Provider]++
		}
	}
}

func (p *xaiProxyPool) pruneQuarantinesLocked(now time.Time) {
	for value, entry := range p.ipQuarantines {
		if !entry.ExpiresAt.After(now) {
			delete(p.ipQuarantines, value)
		}
	}
	for value, entry := range p.nodeQuarantines {
		if !entry.ExpiresAt.After(now) {
			delete(p.nodeQuarantines, value)
		}
	}
}

func (p *xaiProxyPool) runMaintenance() {
	providerInterval := parsePositiveDuration(p.cfg.ProviderRefreshInterval, time.Hour)
	egressInterval := parsePositiveDuration(p.cfg.EgressRefreshInterval, time.Hour)
	providerTicker := time.NewTicker(providerInterval)
	egressTicker := time.NewTicker(egressInterval)
	defer providerTicker.Stop()
	defer egressTicker.Stop()
	for {
		select {
		case <-providerTicker.C:
			refreshCtx, refreshCancel := context.WithTimeout(p.ctx, 30*time.Second)
			if errRefresh := p.RefreshProviders(refreshCtx); errRefresh != nil {
				p.setLastError(errRefresh)
			}
			refreshCancel()
		case <-egressTicker.C:
			refreshCtx, refreshCancel := context.WithTimeout(p.ctx, time.Minute)
			if errRefresh := p.refreshActiveEgress(refreshCtx); errRefresh != nil {
				p.setLastError(errRefresh)
			}
			refreshCancel()
		case <-p.done:
			return
		}
	}
}

func (p *xaiProxyPool) refreshActiveEgress(ctx context.Context) error {
	release, errAcquire := p.acquireProbeGate(ctx)
	if errAcquire != nil {
		return errAcquire
	}
	defer release()
	return p.refreshActiveEgressLocked(ctx)
}

func (p *xaiProxyPool) refreshActiveEgressLocked(ctx context.Context) error {
	ctx = nonNilContext(ctx)
	p.mu.RLock()
	routes := make([]XAIProxyRoute, 0, len(p.laneOrder))
	for _, laneName := range p.laneOrder {
		if lane := p.lanes[laneName]; lane != nil && lane.Ready {
			routes = append(routes, lane.route())
		}
	}
	p.mu.RUnlock()
	seen := make(map[string]string)
	var errs []error
	for _, route := range routes {
		ip, errIP := p.controller.EgressIP(ctx, p.cfg.Probe.ProxyURL, p.cfg.Probe.Selector, route.Node, p.cfg.IPCheckURLs)
		if errIP != nil {
			errs = append(errs, errIP)
			continue
		}
		if other, duplicate := seen[ip]; duplicate && other != route.LaneName {
			errs = append(errs, fmt.Errorf("lanes %q and %q share egress IP %s", other, route.LaneName, ip))
			p.mu.Lock()
			if lane := p.lanes[route.LaneName]; lane != nil && lane.Node == route.Node {
				lane.Ready = false
			}
			p.mu.Unlock()
			continue
		}
		p.mu.RLock()
		_, quarantined := p.ipQuarantines[ip]
		p.mu.RUnlock()
		if quarantined {
			p.mu.Lock()
			if lane := p.lanes[route.LaneName]; lane != nil && lane.Node == route.Node {
				lane.Ready = false
			}
			p.mu.Unlock()
			errs = append(errs, fmt.Errorf("lane %q resolved to quarantined egress IP %s", route.LaneName, ip))
			continue
		}
		seen[ip] = route.LaneName
		p.mu.Lock()
		if lane := p.lanes[route.LaneName]; lane != nil && lane.Node == route.Node {
			lane.EgressIP = ip
		}
		p.mu.Unlock()
	}
	if errSave := p.saveState(); errSave != nil {
		errs = append(errs, errSave)
	}
	return errors.Join(errs...)
}

func (p *xaiProxyPool) loadState() error {
	path := strings.TrimSpace(p.cfg.StateFile)
	if path == "" {
		return nil
	}
	data, errRead := os.ReadFile(path)
	if errors.Is(errRead, os.ErrNotExist) {
		return nil
	}
	if errRead != nil {
		return fmt.Errorf("xai proxy pool: read state: %w", errRead)
	}
	var state xaiProxyPersistedState
	if errDecode := json.Unmarshal(data, &state); errDecode != nil {
		return fmt.Errorf("xai proxy pool: decode state: %w", errDecode)
	}
	if state.Version != xaiProxyPoolStateVersion {
		return fmt.Errorf("xai proxy pool: unsupported state version %d", state.Version)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for name, saved := range state.Lanes {
		if lane := p.lanes[name]; lane != nil {
			lane.Node = saved.Node
			lane.Provider = saved.Provider
			lane.EgressIP = saved.EgressIP
			lane.LastChanged = saved.LastChanged
			lane.MidFailures = append([]time.Time(nil), saved.MidFailures...)
		}
	}
	if state.IPQuarantines != nil {
		p.ipQuarantines = state.IPQuarantines
	}
	if state.NodeQuarantines != nil {
		p.nodeQuarantines = state.NodeQuarantines
	}
	p.counters = state.Counters
	p.pruneQuarantinesLocked(p.now())
	return nil
}

func (p *xaiProxyPool) saveState() error {
	p.saveMu.Lock()
	defer p.saveMu.Unlock()
	path := strings.TrimSpace(p.cfg.StateFile)
	if path == "" {
		return nil
	}
	p.mu.RLock()
	state := xaiProxyPersistedState{
		Version:         xaiProxyPoolStateVersion,
		UpdatedAt:       p.now(),
		Lanes:           make(map[string]xaiProxyPersistLane, len(p.lanes)),
		IPQuarantines:   cloneQuarantines(p.ipQuarantines),
		NodeQuarantines: cloneQuarantines(p.nodeQuarantines),
		Counters:        p.counters,
	}
	for name, lane := range p.lanes {
		if lane == nil || lane.Node == "" {
			continue
		}
		state.Lanes[name] = xaiProxyPersistLane{
			Node: lane.Node, Provider: lane.Provider, EgressIP: lane.EgressIP, LastChanged: lane.LastChanged,
			MidFailures: append([]time.Time(nil), lane.MidFailures...),
		}
	}
	p.mu.RUnlock()
	data, errMarshal := json.MarshalIndent(state, "", "  ")
	if errMarshal != nil {
		return fmt.Errorf("xai proxy pool: encode state: %w", errMarshal)
	}
	dir := filepath.Dir(path)
	if errMkdir := os.MkdirAll(dir, 0o700); errMkdir != nil {
		return fmt.Errorf("xai proxy pool: create state directory: %w", errMkdir)
	}
	tmp, errCreate := os.CreateTemp(dir, ".xai-proxy-pool-*.tmp")
	if errCreate != nil {
		return fmt.Errorf("xai proxy pool: create state temp file: %w", errCreate)
	}
	tmpName := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			if errRemove := os.Remove(tmpName); errRemove != nil && !errors.Is(errRemove, os.ErrNotExist) {
				log.WithError(errRemove).Debug("xai proxy pool: remove state temp file")
			}
		}
	}()
	if errChmod := tmp.Chmod(0o600); errChmod != nil {
		return closeXAIProxyStateTemp(tmp, "chmod state temp file", errChmod)
	}
	if _, errWrite := tmp.Write(append(data, '\n')); errWrite != nil {
		return closeXAIProxyStateTemp(tmp, "write state", errWrite)
	}
	if errSync := tmp.Sync(); errSync != nil {
		return closeXAIProxyStateTemp(tmp, "sync state", errSync)
	}
	if errClose := tmp.Close(); errClose != nil {
		return fmt.Errorf("xai proxy pool: close state: %w", errClose)
	}
	if errRename := os.Rename(tmpName, path); errRename != nil {
		return fmt.Errorf("xai proxy pool: replace state: %w", errRename)
	}
	removeTemp = false
	return nil
}

func closeXAIProxyStateTemp(tmp *os.File, operation string, cause error) error {
	errClose := tmp.Close()
	if errClose != nil {
		cause = errors.Join(cause, fmt.Errorf("close state temp file: %w", errClose))
	}
	return fmt.Errorf("xai proxy pool: %s: %w", operation, cause)
}

func cloneQuarantines(src map[string]xaiProxyQuarantine) map[string]xaiProxyQuarantine {
	out := make(map[string]xaiProxyQuarantine, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func (p *xaiProxyPool) setLastError(err error) {
	if p == nil || err == nil {
		return
	}
	p.mu.Lock()
	p.lastError = err.Error()
	p.mu.Unlock()
}

func (p *xaiProxyPool) Close() {
	if p == nil {
		return
	}
	p.lifecycleMu.Lock()
	if !p.closed.CompareAndSwap(false, true) {
		p.lifecycleMu.Unlock()
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	close(p.done)
	p.lifecycleMu.Unlock()
	p.wg.Wait()
}

func (p *xaiProxyPool) startWorker(run func()) bool {
	if p == nil || run == nil {
		return false
	}
	p.lifecycleMu.Lock()
	defer p.lifecycleMu.Unlock()
	if p.closed.Load() {
		return false
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		run()
	}()
	return true
}

func newXAIProxyStartLimiter(cfg config.XAIProxyPoolConfig, now func() time.Time, rejected func()) *xaiProxyStartLimiter {
	if now == nil {
		now = time.Now
	}
	requests := cfg.RequestsPerMinutePerLane
	if requests <= 0 {
		requests = 30
	}
	burst := cfg.BurstPerLane
	if burst <= 0 {
		burst = 3
	}
	queueSize := cfg.QueueSizePerLane
	if queueSize <= 0 {
		queueSize = 30
	}
	return &xaiProxyStartLimiter{
		rate:     float64(requests) / 60,
		burst:    float64(burst),
		tokens:   float64(burst),
		last:     now(),
		queue:    make(chan struct{}, queueSize),
		now:      now,
		rejected: rejected,
	}
}

func (l *xaiProxyStartLimiter) Wait(ctx context.Context) error {
	if l == nil || l.rate <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case l.queue <- struct{}{}:
		defer func() { <-l.queue }()
	default:
		if l.rejected != nil {
			l.rejected()
		}
		return &XAIProxyPoolError{Message: "xai proxy lane queue is full", Retry: time.Second}
	}

	now := l.now()
	l.mu.Lock()
	elapsed := now.Sub(l.last).Seconds()
	if elapsed > 0 {
		l.tokens += elapsed * l.rate
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
	}
	l.last = now
	l.tokens--
	wait := time.Duration(0)
	if l.tokens < 0 {
		wait = time.Duration((-l.tokens / l.rate) * float64(time.Second))
	}
	l.mu.Unlock()
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func poolUnavailable(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "xai proxy pool is unavailable"
	}
	return &XAIProxyPoolError{Message: message, Retry: 30 * time.Second}
}
