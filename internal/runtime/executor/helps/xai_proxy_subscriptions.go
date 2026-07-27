package helps

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	xaiProxySubscriptionRegistryVersion = 1
	xaiProxySubscriptionRegistryMaxSize = 1 << 20
	xaiProxyGeneratedConfigMaxSize      = 2 << 20
)

var xaiProxySubscriptionNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type xaiProxyConfigReloader interface {
	ReloadConfig(context.Context, []byte) error
	RefreshProvider(context.Context, string) error
}

// XAIProxySubscriptionError is a redacted Management API error. It never wraps
// controller or filesystem errors because those may contain a write-only URL.
type XAIProxySubscriptionError struct {
	Code    string
	Message string
	Status  int
}

func (e *XAIProxySubscriptionError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *XAIProxySubscriptionError) StatusCode() int {
	if e == nil || e.Status == 0 {
		return http.StatusBadGateway
	}
	return e.Status
}

func (e *XAIProxySubscriptionError) IsRequestScoped() bool { return true }

// XAIProxySubscriptionList is the redacted subscription-management view.
type XAIProxySubscriptionList struct {
	Enabled       bool                         `json:"enabled"`
	Ready         bool                         `json:"ready"`
	Revision      uint64                       `json:"revision"`
	LastErrorCode string                       `json:"last_error_code,omitempty"`
	Subscriptions []XAIProxySubscriptionStatus `json:"subscriptions"`
}

// XAIProxySubscriptionStatus intentionally omits the write-only URL.
type XAIProxySubscriptionStatus struct {
	Name          string     `json:"name"`
	Enabled       bool       `json:"enabled"`
	HostLabel     string     `json:"host_label,omitempty"`
	Fingerprint   string     `json:"fingerprint"`
	NodeCount     int        `json:"node_count"`
	State         string     `json:"state"`
	LastCheckAt   *time.Time `json:"last_check_at,omitempty"`
	LastErrorCode string     `json:"last_error_code,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// XAIProxySubscriptionCreate is the write-only create input.
type XAIProxySubscriptionCreate struct {
	Name    string
	URL     string
	Enabled bool
}

// XAIProxySubscriptionUpdate is the write-only partial update input.
type XAIProxySubscriptionUpdate struct {
	URL     *string
	Enabled *bool
}

type xaiProxySubscriptionManager struct {
	cfg        config.XAIProxyPoolConfig
	management config.XAIProxySubscriptionManagementConfig
	controller xaiProxyController
	reloader   xaiProxyConfigReloader
	pool       *xaiProxyPool
	now        func() time.Time

	mu            sync.Mutex
	registry      xaiProxySubscriptionRegistry
	checks        map[string]xaiProxySubscriptionCheck
	ready         bool
	lastErrorCode string
}

type xaiProxySubscriptionCheck struct {
	At        time.Time
	ErrorCode string
}

type xaiProxySubscriptionRegistry struct {
	Version       int                         `json:"version"`
	Revision      uint64                      `json:"revision"`
	UpdatedAt     time.Time                   `json:"updated_at"`
	Subscriptions []xaiProxySubscriptionEntry `json:"subscriptions"`
}

type xaiProxySubscriptionEntry struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newXAIProxySubscriptionManager(cfg config.XAIProxyPoolConfig, controller xaiProxyController, pool *xaiProxyPool, now func() time.Time) *xaiProxySubscriptionManager {
	management := cfg.SubscriptionManagement
	if !management.Enabled {
		return nil
	}
	if now == nil {
		now = time.Now
	}
	if management.MaxProviders <= 0 {
		management.MaxProviders = 64
	}
	if management.MaxURLLength <= 0 {
		management.MaxURLLength = 4096
	}
	if management.MaxDownloadBytes <= 0 {
		management.MaxDownloadBytes = 8 << 20
	}
	if strings.TrimSpace(management.ActivationTimeout) == "" {
		management.ActivationTimeout = "30s"
	}
	manager := &xaiProxySubscriptionManager{
		cfg:        cfg,
		management: management,
		controller: controller,
		pool:       pool,
		now:        now,
		registry: xaiProxySubscriptionRegistry{
			Version:       xaiProxySubscriptionRegistryVersion,
			Subscriptions: []xaiProxySubscriptionEntry{},
		},
		checks: make(map[string]xaiProxySubscriptionCheck),
	}
	reloader, okReloader := controller.(xaiProxyConfigReloader)
	if !okReloader || reloader == nil {
		manager.lastErrorCode = "controller_reload_unsupported"
		return manager
	}
	manager.reloader = reloader
	if strings.TrimSpace(management.RegistryFile) == "" || strings.TrimSpace(management.GeneratedConfigFile) == "" {
		manager.lastErrorCode = "storage_paths_missing"
		return manager
	}
	if filepath.Clean(management.RegistryFile) == filepath.Clean(management.GeneratedConfigFile) {
		manager.lastErrorCode = "storage_paths_conflict"
		return manager
	}
	registry, errLoad := manager.loadRegistry()
	if errLoad != nil {
		manager.lastErrorCode = "registry_load_failed"
		return manager
	}
	manager.registry = registry
	if errSync := manager.syncGeneratedConfig(); errSync != nil {
		manager.lastErrorCode = "generated_config_sync_failed"
		return manager
	}
	manager.ready = true
	return manager
}

func (m *xaiProxySubscriptionManager) List(ctx context.Context) XAIProxySubscriptionList {
	if m == nil {
		return XAIProxySubscriptionList{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listLocked(ctx)
}

func (m *xaiProxySubscriptionManager) listLocked(ctx context.Context) XAIProxySubscriptionList {
	result := XAIProxySubscriptionList{
		Enabled:       m.management.Enabled,
		Ready:         m.ready,
		Revision:      m.registry.Revision,
		LastErrorCode: m.lastErrorCode,
		Subscriptions: make([]XAIProxySubscriptionStatus, 0, len(m.registry.Subscriptions)),
	}
	counts := make(map[string]int)
	if m.ready && m.controller != nil {
		if nodes, errSnapshot := m.controller.Snapshot(nonNilContext(ctx)); errSnapshot == nil {
			for _, node := range nodes {
				if node.Alive {
					counts[node.Provider]++
				}
			}
		}
	}
	for _, entry := range m.registry.Subscriptions {
		state := "disabled"
		if entry.Enabled {
			state = "empty"
			if counts[entry.Name] > 0 {
				state = "ready"
			}
		}
		status := XAIProxySubscriptionStatus{
			Name:        entry.Name,
			Enabled:     entry.Enabled,
			HostLabel:   xaiSubscriptionHostLabel(entry.URL),
			Fingerprint: xaiSubscriptionFingerprint(entry.URL),
			NodeCount:   counts[entry.Name],
			State:       state,
			CreatedAt:   entry.CreatedAt,
			UpdatedAt:   entry.UpdatedAt,
		}
		if check, exists := m.checks[strings.ToLower(entry.Name)]; exists {
			checkedAt := check.At
			status.LastCheckAt = &checkedAt
			status.LastErrorCode = check.ErrorCode
		}
		result.Subscriptions = append(result.Subscriptions, status)
	}
	sort.Slice(result.Subscriptions, func(i, j int) bool {
		return result.Subscriptions[i].Name < result.Subscriptions[j].Name
	})
	return result
}

func (m *xaiProxySubscriptionManager) Create(ctx context.Context, expectedRevision uint64, input XAIProxySubscriptionCreate) (XAIProxySubscriptionList, error) {
	if m == nil {
		return XAIProxySubscriptionList{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if errReady := m.requireReadyLocked(expectedRevision); errReady != nil {
		return m.listLocked(ctx), errReady
	}
	entry, errEntry := m.validatedEntry(input.Name, input.URL, input.Enabled, m.now(), m.now())
	if errEntry != nil {
		return m.listLocked(ctx), errEntry
	}
	if _, exists := findXAIProxySubscription(m.registry.Subscriptions, entry.Name); exists {
		return m.listLocked(ctx), subscriptionError(http.StatusConflict, "subscription_exists", "subscription provider already exists")
	}
	if len(m.registry.Subscriptions) >= m.management.MaxProviders {
		return m.listLocked(ctx), subscriptionError(http.StatusConflict, "provider_limit_reached", "subscription provider limit reached")
	}
	candidate := cloneXAIProxySubscriptionRegistry(m.registry)
	candidate.Revision++
	candidate.UpdatedAt = m.now()
	candidate.Subscriptions = append(candidate.Subscriptions, entry)
	if errCommit := m.commitCandidateLocked(ctx, candidate, entry.Name, input.Enabled, ""); errCommit != nil {
		return m.listLocked(ctx), errCommit
	}
	return m.listLocked(ctx), nil
}

func (m *xaiProxySubscriptionManager) Update(ctx context.Context, expectedRevision uint64, name string, input XAIProxySubscriptionUpdate) (XAIProxySubscriptionList, error) {
	if m == nil {
		return XAIProxySubscriptionList{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if errReady := m.requireReadyLocked(expectedRevision); errReady != nil {
		return m.listLocked(ctx), errReady
	}
	index, exists := findXAIProxySubscription(m.registry.Subscriptions, name)
	if !exists {
		return m.listLocked(ctx), subscriptionError(http.StatusNotFound, "subscription_not_found", "subscription provider was not found")
	}
	if input.URL == nil && input.Enabled == nil {
		return m.listLocked(ctx), subscriptionError(http.StatusBadRequest, "empty_update", "subscription update is empty")
	}
	candidate := cloneXAIProxySubscriptionRegistry(m.registry)
	current := candidate.Subscriptions[index]
	updatedURL := current.URL
	if input.URL != nil {
		updatedURL = *input.URL
	}
	updatedEnabled := current.Enabled
	if input.Enabled != nil {
		updatedEnabled = *input.Enabled
	}
	updated, errEntry := m.validatedEntry(current.Name, updatedURL, updatedEnabled, current.CreatedAt, m.now())
	if errEntry != nil {
		return m.listLocked(ctx), errEntry
	}
	candidate.Revision++
	candidate.UpdatedAt = m.now()
	candidate.Subscriptions[index] = updated
	reloadRequired := current.Enabled || updated.Enabled
	disabledProvider := ""
	if current.Enabled && !updated.Enabled {
		disabledProvider = current.Name
	}
	if errCommit := m.commitCandidateLocked(ctx, candidate, updated.Name, reloadRequired, disabledProvider); errCommit != nil {
		return m.listLocked(ctx), errCommit
	}
	return m.listLocked(ctx), nil
}

func (m *xaiProxySubscriptionManager) Delete(ctx context.Context, expectedRevision uint64, name string) (XAIProxySubscriptionList, error) {
	if m == nil {
		return XAIProxySubscriptionList{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if errReady := m.requireReadyLocked(expectedRevision); errReady != nil {
		return m.listLocked(ctx), errReady
	}
	index, exists := findXAIProxySubscription(m.registry.Subscriptions, name)
	if !exists {
		return m.listLocked(ctx), subscriptionError(http.StatusNotFound, "subscription_not_found", "subscription provider was not found")
	}
	entry := m.registry.Subscriptions[index]
	if entry.Enabled {
		return m.listLocked(ctx), subscriptionError(http.StatusConflict, "subscription_enabled", "disable the subscription provider before deleting it")
	}
	if m.pool != nil && m.pool.providerInUse(entry.Name) {
		return m.listLocked(ctx), subscriptionError(http.StatusConflict, "subscription_in_use", "subscription provider is still used by an active lane")
	}
	candidate := cloneXAIProxySubscriptionRegistry(m.registry)
	candidate.Revision++
	candidate.UpdatedAt = m.now()
	candidate.Subscriptions = append(candidate.Subscriptions[:index], candidate.Subscriptions[index+1:]...)
	if errRemove := m.removeProviderCache(entry.Name); errRemove != nil {
		m.lastErrorCode = "provider_cache_remove_failed"
		return m.listLocked(ctx), subscriptionError(http.StatusInternalServerError, "provider_cache_remove_failed", "subscription provider cache could not be removed")
	}
	if errSave := m.saveRegistry(candidate); errSave != nil {
		m.lastErrorCode = "registry_persist_failed"
		return m.listLocked(ctx), subscriptionError(http.StatusInternalServerError, "registry_persist_failed", "subscription registry could not be persisted")
	}
	m.registry = candidate
	delete(m.checks, strings.ToLower(entry.Name))
	m.lastErrorCode = ""
	return m.listLocked(ctx), nil
}

func (m *xaiProxySubscriptionManager) Check(ctx context.Context, name string) (XAIProxySubscriptionStatus, error) {
	if m == nil {
		return XAIProxySubscriptionStatus{}, subscriptionError(http.StatusServiceUnavailable, "subscription_management_unavailable", "xAI subscription management is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ready {
		return XAIProxySubscriptionStatus{}, subscriptionError(http.StatusServiceUnavailable, m.lastErrorCode, "xAI subscription management is not ready")
	}
	index, exists := findXAIProxySubscription(m.registry.Subscriptions, name)
	if !exists {
		return XAIProxySubscriptionStatus{}, subscriptionError(http.StatusNotFound, "subscription_not_found", "subscription provider was not found")
	}
	entry := m.registry.Subscriptions[index]
	if !entry.Enabled {
		return XAIProxySubscriptionStatus{}, subscriptionError(http.StatusConflict, "subscription_disabled", "subscription provider is disabled")
	}
	activationCtx, cancel := context.WithTimeout(nonNilContext(ctx), parsePositiveDuration(m.management.ActivationTimeout, 30*time.Second))
	defer cancel()
	nodes, errWait := m.waitForFreshProvider(activationCtx, entry.Name)
	if errWait != nil {
		m.recordCheckLocked(entry.Name, subscriptionErrorCode(errWait, "provider_refresh_failed"))
		return XAIProxySubscriptionStatus{}, errWait
	}
	m.recordCheckLocked(entry.Name, "")
	count := countXAIProxyProviderNodes(nodes, entry.Name)
	checkedAt := m.checks[strings.ToLower(entry.Name)].At
	return XAIProxySubscriptionStatus{
		Name: entry.Name, Enabled: true, HostLabel: xaiSubscriptionHostLabel(entry.URL),
		Fingerprint: xaiSubscriptionFingerprint(entry.URL), NodeCount: count, State: "ready",
		LastCheckAt: &checkedAt, CreatedAt: entry.CreatedAt, UpdatedAt: entry.UpdatedAt,
	}, nil
}

func (m *xaiProxySubscriptionManager) requireReadyLocked(expectedRevision uint64) error {
	if !m.ready {
		return subscriptionError(http.StatusServiceUnavailable, m.lastErrorCode, "xAI subscription management is not ready")
	}
	if expectedRevision != m.registry.Revision {
		return subscriptionError(http.StatusPreconditionFailed, "revision_mismatch", "subscription registry revision does not match")
	}
	return nil
}

func (m *xaiProxySubscriptionManager) validatedEntry(name string, rawURL string, enabled bool, createdAt time.Time, updatedAt time.Time) (xaiProxySubscriptionEntry, error) {
	name = strings.TrimSpace(name)
	if !xaiProxySubscriptionNamePattern.MatchString(name) || m.reservedName(name) {
		return xaiProxySubscriptionEntry{}, subscriptionError(http.StatusBadRequest, "invalid_provider_name", "subscription provider name is invalid or reserved")
	}
	normalizedURL, errURL := validateXAIProxySubscriptionURL(rawURL, m.management.MaxURLLength)
	if errURL != nil {
		return xaiProxySubscriptionEntry{}, errURL
	}
	return xaiProxySubscriptionEntry{
		Name: name, URL: normalizedURL, Enabled: enabled, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func (m *xaiProxySubscriptionManager) reservedName(name string) bool {
	reserved := map[string]struct{}{
		"direct": {}, "reject": {}, "global": {}, "compatible": {}, "pass": {},
	}
	for _, lane := range m.cfg.Lanes {
		reserved[strings.ToLower(lane.Name)] = struct{}{}
		reserved[strings.ToLower(lane.Selector)] = struct{}{}
	}
	reserved[strings.ToLower(m.cfg.Probe.Name)] = struct{}{}
	reserved[strings.ToLower(m.cfg.Probe.Selector)] = struct{}{}
	_, exists := reserved[strings.ToLower(strings.TrimSpace(name))]
	return exists
}

func (m *xaiProxySubscriptionManager) commitCandidateLocked(ctx context.Context, candidate xaiProxySubscriptionRegistry, targetProvider string, reloadRequired bool, disabledProvider string) error {
	if !reloadRequired {
		if errSave := m.saveRegistry(candidate); errSave != nil {
			m.lastErrorCode = "registry_persist_failed"
			return subscriptionError(http.StatusInternalServerError, "registry_persist_failed", "subscription registry could not be persisted")
		}
		m.registry = candidate
		m.lastErrorCode = ""
		return nil
	}
	oldPayload, oldPayloadExists, errOld := m.currentGeneratedPayload()
	if errOld != nil {
		m.lastErrorCode = "generated_config_read_failed"
		return subscriptionError(http.StatusInternalServerError, "generated_config_read_failed", "current generated Mihomo configuration could not be read")
	}
	candidatePayload, errRender := m.renderConfig(candidate)
	if errRender != nil {
		m.lastErrorCode = "config_render_failed"
		return subscriptionError(http.StatusInternalServerError, "config_render_failed", "candidate Mihomo configuration could not be rendered")
	}
	activationCtx, cancel := context.WithTimeout(nonNilContext(ctx), parsePositiveDuration(m.management.ActivationTimeout, 30*time.Second))
	defer cancel()
	releaseTopology := func() {}
	if m.pool != nil && m.cfg.Enabled {
		release, errAcquire := m.pool.acquireProbeGate(activationCtx)
		if errAcquire != nil {
			m.lastErrorCode = "topology_lock_failed"
			return subscriptionError(http.StatusServiceUnavailable, "topology_lock_failed", "xAI proxy topology is busy")
		}
		releaseTopology = release
	}
	defer releaseTopology()
	if errReload := m.reloader.ReloadConfig(activationCtx, candidatePayload); errReload != nil {
		m.lastErrorCode = "candidate_reload_failed"
		m.recordCheckLocked(targetProvider, "candidate_reload_failed")
		if errRollback := m.rollbackRuntime(oldPayload, targetProvider); errRollback != nil {
			return subscriptionError(http.StatusInternalServerError, "rollback_failed", "previous Mihomo configuration could not be restored")
		}
		return subscriptionError(http.StatusBadGateway, "candidate_reload_failed", "candidate Mihomo configuration was rejected")
	}
	nodes, errVerify := m.verifyCandidate(activationCtx, targetProvider, candidate, disabledProvider)
	if errVerify != nil {
		m.recordCheckLocked(targetProvider, subscriptionErrorCode(errVerify, "provider_activation_failed"))
		if errRollback := m.rollbackRuntime(oldPayload, targetProvider); errRollback != nil {
			return subscriptionError(http.StatusInternalServerError, "rollback_failed", "previous Mihomo configuration could not be restored")
		}
		return errVerify
	}
	if m.pool != nil && m.cfg.Enabled {
		if errReconcile := m.pool.reconcileSubscriptionSnapshotLocked(activationCtx, nodes); errReconcile != nil {
			m.recordCheckLocked(targetProvider, "lane_reconcile_failed")
			if errRollback := m.rollbackRuntime(oldPayload, targetProvider); errRollback != nil {
				return subscriptionError(http.StatusInternalServerError, "rollback_failed", "previous Mihomo configuration could not be restored")
			}
			m.lastErrorCode = "lane_reconcile_failed"
			return subscriptionError(http.StatusServiceUnavailable, "lane_reconcile_failed", "candidate subscription set could not provide all proxy lanes")
		}
	}
	if errWriteConfig := writeXAIProxyPrivateFile(m.management.GeneratedConfigFile, candidatePayload); errWriteConfig != nil {
		m.recordCheckLocked(targetProvider, "generated_config_persist_failed")
		if errRollback := m.rollbackRuntime(oldPayload, targetProvider); errRollback != nil {
			return subscriptionError(http.StatusInternalServerError, "rollback_failed", "previous Mihomo configuration could not be restored")
		}
		m.lastErrorCode = "generated_config_persist_failed"
		return subscriptionError(http.StatusInternalServerError, "generated_config_persist_failed", "generated Mihomo configuration could not be persisted")
	}
	if errSave := m.saveRegistry(candidate); errSave != nil {
		errRestore := m.restoreGeneratedFile(oldPayload, oldPayloadExists)
		errRollback := m.rollbackRuntime(oldPayload, targetProvider)
		if errRestore != nil || errRollback != nil {
			m.lastErrorCode = "rollback_failed"
			return subscriptionError(http.StatusInternalServerError, "rollback_failed", "previous subscription configuration could not be restored")
		}
		m.lastErrorCode = "registry_persist_failed"
		m.recordCheckLocked(targetProvider, "registry_persist_failed")
		return subscriptionError(http.StatusInternalServerError, "registry_persist_failed", "subscription registry could not be persisted")
	}
	m.registry = candidate
	if index, exists := findXAIProxySubscription(candidate.Subscriptions, targetProvider); exists && candidate.Subscriptions[index].Enabled {
		m.recordCheckLocked(targetProvider, "")
	}
	m.lastErrorCode = ""
	return nil
}

func (m *xaiProxySubscriptionManager) verifyCandidate(ctx context.Context, targetProvider string, candidate xaiProxySubscriptionRegistry, disabledProvider string) ([]xaiProxyNode, error) {
	targetEnabled := false
	if index, exists := findXAIProxySubscription(candidate.Subscriptions, targetProvider); exists {
		targetEnabled = candidate.Subscriptions[index].Enabled
	}
	if targetEnabled {
		return m.waitForFreshProvider(ctx, targetProvider)
	}
	nodes, errWait := m.waitForProviderAbsent(ctx, targetProvider)
	if errWait != nil {
		return nil, errWait
	}
	if disabledProvider != "" && countXAIProxyProviderNodes(nodes, disabledProvider) != 0 {
		m.lastErrorCode = "provider_disable_failed"
		return nil, subscriptionError(http.StatusBadGateway, "provider_disable_failed", "disabled subscription provider remained active")
	}
	return nodes, nil
}

func (m *xaiProxySubscriptionManager) waitForFreshProvider(ctx context.Context, targetProvider string) ([]xaiProxyNode, error) {
	if errRefresh := m.waitForProviderRefresh(ctx, targetProvider); errRefresh != nil {
		m.lastErrorCode = "provider_refresh_failed"
		return nil, subscriptionError(http.StatusBadGateway, "provider_refresh_failed", "subscription provider refresh failed")
	}
	return m.waitForProviderPresent(ctx, targetProvider)
}

func (m *xaiProxySubscriptionManager) waitForProviderRefresh(ctx context.Context, targetProvider string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if errRefresh := m.reloader.RefreshProvider(ctx, targetProvider); errRefresh == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *xaiProxySubscriptionManager) waitForProviderPresent(ctx context.Context, targetProvider string) ([]xaiProxyNode, error) {
	return m.waitForProviderCondition(ctx, func(nodes []xaiProxyNode) bool {
		return countXAIProxyProviderNodes(nodes, targetProvider) > 0
	}, "provider_activation_failed", "subscription provider did not expose a valid node before timeout")
}

func (m *xaiProxySubscriptionManager) waitForProviderAbsent(ctx context.Context, targetProvider string) ([]xaiProxyNode, error) {
	return m.waitForProviderCondition(ctx, func(nodes []xaiProxyNode) bool {
		return countXAIProxyProviderNodes(nodes, targetProvider) == 0
	}, "provider_disable_failed", "disabled subscription provider remained active")
}

func (m *xaiProxySubscriptionManager) waitForProviderCondition(ctx context.Context, ready func([]xaiProxyNode) bool, code string, message string) ([]xaiProxyNode, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		nodes, errSnapshot := m.controller.Snapshot(ctx)
		if errSnapshot == nil && ready(nodes) {
			return nodes, nil
		}
		select {
		case <-ctx.Done():
			m.lastErrorCode = code
			return nil, subscriptionError(http.StatusBadGateway, code, message)
		case <-ticker.C:
		}
	}
}

func (m *xaiProxySubscriptionManager) rollbackRuntime(oldPayload []byte, targetProvider string) error {
	if len(oldPayload) == 0 {
		m.lastErrorCode = "rollback_unavailable"
		return fmt.Errorf("rollback payload unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsePositiveDuration(m.management.ActivationTimeout, 30*time.Second))
	defer cancel()
	if errReload := m.reloader.ReloadConfig(ctx, oldPayload); errReload != nil {
		m.lastErrorCode = "rollback_failed"
		return errReload
	}
	var nodes []xaiProxyNode
	if index, exists := findXAIProxySubscription(m.registry.Subscriptions, targetProvider); exists && m.registry.Subscriptions[index].Enabled {
		if errRefresh := m.waitForProviderRefresh(ctx, targetProvider); errRefresh != nil {
			m.lastErrorCode = "rollback_failed"
			return errRefresh
		}
		var errSnapshot error
		nodes, errSnapshot = m.waitForProviderCondition(ctx, func([]xaiProxyNode) bool { return true }, "rollback_failed", "previous Mihomo provider snapshot could not be restored")
		if errSnapshot != nil {
			m.lastErrorCode = "rollback_failed"
			return errSnapshot
		}
	} else {
		var errAbsent error
		nodes, errAbsent = m.waitForProviderAbsent(ctx, targetProvider)
		if errAbsent != nil {
			m.lastErrorCode = "rollback_failed"
			return errAbsent
		}
		if errRemove := m.removeProviderCache(targetProvider); errRemove != nil {
			m.lastErrorCode = "rollback_failed"
			return errRemove
		}
	}
	if m.pool != nil && m.cfg.Enabled {
		if errReconcile := m.pool.reconcileSubscriptionSnapshotLocked(ctx, nodes); errReconcile != nil {
			m.lastErrorCode = "rollback_failed"
			return errReconcile
		}
	}
	return nil
}

func (m *xaiProxySubscriptionManager) syncGeneratedConfig() error {
	current, exists, errRead := readXAIProxyPrivateFile(strings.TrimSpace(m.management.GeneratedConfigFile), xaiProxyGeneratedConfigMaxSize)
	if errRead != nil {
		return errRead
	}
	if !exists && len(m.registry.Subscriptions) == 0 {
		return nil
	}
	expected, errRender := m.renderConfig(m.registry)
	if errRender != nil {
		return errRender
	}
	if exists && bytes.Equal(current, expected) {
		return nil
	}
	return writeXAIProxyPrivateFile(m.management.GeneratedConfigFile, expected)
}

func (m *xaiProxySubscriptionManager) reloadSourceOfTruth(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ready {
		return subscriptionError(http.StatusServiceUnavailable, m.lastErrorCode, "xAI subscription management is not ready")
	}
	payload, errRender := m.renderConfig(m.registry)
	if errRender != nil {
		m.lastErrorCode = "config_render_failed"
		return subscriptionError(http.StatusInternalServerError, "config_render_failed", "registry Mihomo configuration could not be rendered")
	}
	reloadCtx, cancel := context.WithTimeout(nonNilContext(ctx), parsePositiveDuration(m.management.ActivationTimeout, 30*time.Second))
	defer cancel()
	releaseTopology := func() {}
	if m.pool != nil && m.cfg.Enabled {
		release, errAcquire := m.pool.acquireProbeGate(reloadCtx)
		if errAcquire != nil {
			m.lastErrorCode = "topology_lock_failed"
			return subscriptionError(http.StatusServiceUnavailable, "topology_lock_failed", "xAI proxy topology is busy")
		}
		releaseTopology = release
	}
	defer releaseTopology()
	if errReload := m.reloader.ReloadConfig(reloadCtx, payload); errReload != nil {
		m.lastErrorCode = "startup_reload_failed"
		return subscriptionError(http.StatusBadGateway, "startup_reload_failed", "registry Mihomo configuration could not be activated")
	}
	if errWrite := writeXAIProxyPrivateFile(m.management.GeneratedConfigFile, payload); errWrite != nil {
		m.lastErrorCode = "generated_config_persist_failed"
		return subscriptionError(http.StatusInternalServerError, "generated_config_persist_failed", "generated Mihomo configuration could not be persisted")
	}
	m.lastErrorCode = ""
	return nil
}

func (m *xaiProxySubscriptionManager) currentGeneratedPayload() ([]byte, bool, error) {
	path := strings.TrimSpace(m.management.GeneratedConfigFile)
	data, exists, errRead := readXAIProxyPrivateFile(path, xaiProxyGeneratedConfigMaxSize)
	if !exists && errRead == nil {
		payload, errRender := m.renderConfig(m.registry)
		return payload, false, errRender
	}
	if errRead != nil {
		return nil, false, errRead
	}
	return data, exists, nil
}

func (m *xaiProxySubscriptionManager) restoreGeneratedFile(oldPayload []byte, existed bool) error {
	path := strings.TrimSpace(m.management.GeneratedConfigFile)
	if existed {
		return writeXAIProxyPrivateFile(path, oldPayload)
	}
	if errRemove := os.Remove(path); errRemove != nil && !errors.Is(errRemove, os.ErrNotExist) {
		m.lastErrorCode = "generated_config_restore_failed"
		return errRemove
	}
	return nil
}

func (m *xaiProxySubscriptionManager) removeProviderCache(name string) error {
	if !xaiProxySubscriptionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid provider name")
	}
	path := filepath.Join(filepath.Dir(m.management.GeneratedConfigFile), "providers", name+".yaml")
	if errRemove := os.Remove(path); errRemove != nil && !errors.Is(errRemove, os.ErrNotExist) {
		return errRemove
	}
	return nil
}

func (m *xaiProxySubscriptionManager) loadRegistry() (xaiProxySubscriptionRegistry, error) {
	path := strings.TrimSpace(m.management.RegistryFile)
	data, exists, errRead := readXAIProxyPrivateFile(path, xaiProxySubscriptionRegistryMaxSize)
	if !exists && errRead == nil {
		return xaiProxySubscriptionRegistry{Version: xaiProxySubscriptionRegistryVersion, Subscriptions: []xaiProxySubscriptionEntry{}}, nil
	}
	if errRead != nil {
		return xaiProxySubscriptionRegistry{}, errRead
	}
	var registry xaiProxySubscriptionRegistry
	if errDecode := json.Unmarshal(data, &registry); errDecode != nil || registry.Version != xaiProxySubscriptionRegistryVersion {
		return xaiProxySubscriptionRegistry{}, fmt.Errorf("invalid registry format")
	}
	if len(registry.Subscriptions) > m.management.MaxProviders {
		return xaiProxySubscriptionRegistry{}, fmt.Errorf("provider limit exceeded")
	}
	seen := make(map[string]struct{}, len(registry.Subscriptions))
	for i := range registry.Subscriptions {
		entry := registry.Subscriptions[i]
		validated, errEntry := m.validatedEntry(entry.Name, entry.URL, entry.Enabled, entry.CreatedAt, entry.UpdatedAt)
		if errEntry != nil {
			return xaiProxySubscriptionRegistry{}, fmt.Errorf("invalid registry entry")
		}
		key := strings.ToLower(validated.Name)
		if _, exists := seen[key]; exists {
			return xaiProxySubscriptionRegistry{}, fmt.Errorf("duplicate registry entry")
		}
		seen[key] = struct{}{}
		registry.Subscriptions[i] = validated
	}
	sort.Slice(registry.Subscriptions, func(i, j int) bool { return registry.Subscriptions[i].Name < registry.Subscriptions[j].Name })
	return registry, nil
}

func (m *xaiProxySubscriptionManager) saveRegistry(registry xaiProxySubscriptionRegistry) error {
	registry.Version = xaiProxySubscriptionRegistryVersion
	sort.Slice(registry.Subscriptions, func(i, j int) bool { return registry.Subscriptions[i].Name < registry.Subscriptions[j].Name })
	data, errMarshal := json.MarshalIndent(registry, "", "  ")
	if errMarshal != nil {
		return errMarshal
	}
	return writeXAIProxyPrivateFile(m.management.RegistryFile, append(data, '\n'))
}

func (m *xaiProxySubscriptionManager) renderConfig(registry xaiProxySubscriptionRegistry) ([]byte, error) {
	secretBytes, errSecret := os.ReadFile(strings.TrimSpace(m.cfg.ControllerSecretFile))
	if errSecret != nil || strings.TrimSpace(string(secretBytes)) == "" {
		return nil, fmt.Errorf("controller secret unavailable")
	}
	controllerPort, errPort := xaiProxyControllerPort(m.cfg.ControllerURL)
	if errPort != nil {
		return nil, errPort
	}
	providers := make(map[string]xaiMihomoProvider)
	providerNames := make([]string, 0, len(registry.Subscriptions))
	for _, entry := range registry.Subscriptions {
		if !entry.Enabled {
			continue
		}
		providerNames = append(providerNames, entry.Name)
		providers[entry.Name] = xaiMihomoProvider{
			Type: "http", URL: entry.URL, Path: "./providers/" + entry.Name + ".yaml", Interval: 3600,
			SizeLimit: m.management.MaxDownloadBytes,
			HealthCheck: xaiMihomoHealthCheck{
				Enable: true, URL: m.cfg.HealthCheckURL, Interval: 300,
				Timeout: int(parsePositiveDuration(m.cfg.HealthCheckTimeout, 5*time.Second).Milliseconds()),
				Lazy:    true, ExpectedStatus: "204",
			},
			Override: xaiMihomoOverride{AdditionalPrefix: "[" + entry.Name + "] "},
		}
	}
	sort.Strings(providerNames)
	groups := make([]xaiMihomoProxyGroup, 0, len(m.cfg.Lanes)+1)
	listeners := make([]xaiMihomoListener, 0, len(m.cfg.Lanes)+1)
	seenSelectors := make(map[string]struct{})
	appendRoute := func(lane config.XAIProxyPoolLane) error {
		if _, exists := seenSelectors[lane.Selector]; exists {
			return fmt.Errorf("duplicate selector")
		}
		seenSelectors[lane.Selector] = struct{}{}
		port, errListener := xaiProxyListenerPort(lane.ProxyURL)
		if errListener != nil {
			return errListener
		}
		groups = append(groups, xaiMihomoProxyGroup{
			Name: lane.Selector, Type: "select", Use: append([]string(nil), providerNames...), Proxies: []string{"REJECT"},
		})
		listeners = append(listeners, xaiMihomoListener{
			Name: lane.Name + "-in", Type: "http", Listen: "0.0.0.0", Port: port, Proxy: lane.Selector, Users: []string{},
		})
		return nil
	}
	for _, lane := range m.cfg.Lanes {
		if errRoute := appendRoute(lane); errRoute != nil {
			return nil, errRoute
		}
	}
	if errProbe := appendRoute(m.cfg.Probe); errProbe != nil {
		return nil, errProbe
	}
	output := xaiMihomoGeneratedConfig{
		AllowLAN: true, BindAddress: "*", Mode: "rule", LogLevel: "warning", IPv6: true,
		ExternalController: "0.0.0.0:" + strconv.Itoa(controllerPort), Secret: strings.TrimSpace(string(secretBytes)),
		Profile: xaiMihomoProfile{StoreSelected: true}, ProxyProviders: providers,
		ProxyGroups: groups, Listeners: listeners, Rules: []string{"MATCH,REJECT"},
	}
	data, errMarshal := yaml.Marshal(output)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return data, nil
}

type xaiMihomoGeneratedConfig struct {
	AllowLAN           bool                         `yaml:"allow-lan"`
	BindAddress        string                       `yaml:"bind-address"`
	Mode               string                       `yaml:"mode"`
	LogLevel           string                       `yaml:"log-level"`
	IPv6               bool                         `yaml:"ipv6"`
	ExternalController string                       `yaml:"external-controller"`
	Secret             string                       `yaml:"secret"`
	Profile            xaiMihomoProfile             `yaml:"profile"`
	ProxyProviders     map[string]xaiMihomoProvider `yaml:"proxy-providers"`
	ProxyGroups        []xaiMihomoProxyGroup        `yaml:"proxy-groups"`
	Listeners          []xaiMihomoListener          `yaml:"listeners"`
	Rules              []string                     `yaml:"rules"`
}

type xaiMihomoProfile struct {
	StoreSelected bool `yaml:"store-selected"`
}

type xaiMihomoProvider struct {
	Type        string               `yaml:"type"`
	URL         string               `yaml:"url"`
	Path        string               `yaml:"path"`
	Interval    int                  `yaml:"interval"`
	SizeLimit   int64                `yaml:"size-limit"`
	HealthCheck xaiMihomoHealthCheck `yaml:"health-check"`
	Override    xaiMihomoOverride    `yaml:"override"`
}

type xaiMihomoHealthCheck struct {
	Enable         bool   `yaml:"enable"`
	URL            string `yaml:"url"`
	Interval       int    `yaml:"interval"`
	Timeout        int    `yaml:"timeout"`
	Lazy           bool   `yaml:"lazy"`
	ExpectedStatus string `yaml:"expected-status"`
}

type xaiMihomoOverride struct {
	AdditionalPrefix string `yaml:"additional-prefix"`
}

type xaiMihomoProxyGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Use     []string `yaml:"use,omitempty"`
	Proxies []string `yaml:"proxies"`
}

type xaiMihomoListener struct {
	Name   string   `yaml:"name"`
	Type   string   `yaml:"type"`
	Listen string   `yaml:"listen"`
	Port   int      `yaml:"port"`
	Proxy  string   `yaml:"proxy"`
	Users  []string `yaml:"users"`
}

func validateXAIProxySubscriptionURL(raw string, maxLength int) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxLength {
		return "", subscriptionError(http.StatusBadRequest, "invalid_subscription_url", "subscription URL is empty or too long")
	}
	parsed, errParse := url.Parse(raw)
	if errParse != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return "", subscriptionError(http.StatusBadRequest, "invalid_subscription_url", "subscription URL must be HTTPS without userinfo or fragment")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" || strings.Contains(host, "%") || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return "", subscriptionError(http.StatusBadRequest, "invalid_subscription_host", "subscription URL host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if normalizePublicIP(host) == "" {
			return "", subscriptionError(http.StatusBadRequest, "invalid_subscription_host", "subscription URL IP must be public")
		}
	} else if !strings.Contains(host, ".") {
		return "", subscriptionError(http.StatusBadRequest, "invalid_subscription_host", "subscription URL host must be a public fully qualified name")
	}
	if port := parsed.Port(); port != "" {
		value, errPort := strconv.Atoi(port)
		if errPort != nil || value <= 0 || value > 65535 {
			return "", subscriptionError(http.StatusBadRequest, "invalid_subscription_url", "subscription URL port is invalid")
		}
	}
	parsed.Scheme = "https"
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}

func xaiSubscriptionFingerprint(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

func xaiSubscriptionHostLabel(raw string) string {
	parsed, errParse := url.Parse(raw)
	if errParse != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if net.ParseIP(host) != nil {
		return "public-ip"
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}
	return "…." + strings.Join(parts[len(parts)-2:], ".")
}

func xaiProxyControllerPort(raw string) (int, error) {
	parsed, errParse := url.Parse(strings.TrimSpace(raw))
	if errParse != nil || parsed.Host == "" {
		return 0, fmt.Errorf("invalid controller URL")
	}
	port := parsed.Port()
	if port == "" {
		if strings.EqualFold(parsed.Scheme, "https") {
			return 443, nil
		}
		return 80, nil
	}
	value, errAtoi := strconv.Atoi(port)
	if errAtoi != nil || value <= 0 || value > 65535 {
		return 0, fmt.Errorf("invalid controller port")
	}
	return value, nil
}

func xaiProxyListenerPort(raw string) (int, error) {
	parsed, errParse := url.Parse(strings.TrimSpace(raw))
	if errParse != nil || !strings.EqualFold(parsed.Scheme, "http") || parsed.Port() == "" {
		return 0, fmt.Errorf("subscription-managed lanes require HTTP proxy URLs with explicit ports")
	}
	value, errAtoi := strconv.Atoi(parsed.Port())
	if errAtoi != nil || value <= 0 || value > 65535 {
		return 0, fmt.Errorf("invalid listener port")
	}
	return value, nil
}

func countXAIProxyProviderNodes(nodes []xaiProxyNode, provider string) int {
	count := 0
	for _, node := range nodes {
		if node.Alive && node.Provider == provider {
			count++
		}
	}
	return count
}

func findXAIProxySubscription(entries []xaiProxySubscriptionEntry, name string) (int, bool) {
	name = strings.TrimSpace(name)
	for i := range entries {
		if strings.EqualFold(entries[i].Name, name) {
			return i, true
		}
	}
	return -1, false
}

func cloneXAIProxySubscriptionRegistry(src xaiProxySubscriptionRegistry) xaiProxySubscriptionRegistry {
	out := src
	out.Subscriptions = append([]xaiProxySubscriptionEntry(nil), src.Subscriptions...)
	return out
}

func (m *xaiProxySubscriptionManager) recordCheckLocked(name string, code string) {
	if m == nil {
		return
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return
	}
	m.checks[name] = xaiProxySubscriptionCheck{At: m.now(), ErrorCode: strings.TrimSpace(code)}
}

func writeXAIProxyPrivateFile(path string, data []byte) (resultErr error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("private file path is empty")
	}
	dir := filepath.Dir(path)
	if errMkdir := os.MkdirAll(dir, 0o700); errMkdir != nil {
		return errMkdir
	}
	tmp, errCreate := os.CreateTemp(dir, ".xai-proxy-private-*.tmp")
	if errCreate != nil {
		return errCreate
	}
	tmpName := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			if errRemove := os.Remove(tmpName); errRemove != nil && !errors.Is(errRemove, os.ErrNotExist) {
				resultErr = errors.Join(resultErr, errRemove)
			}
		}
	}()
	if errChmod := tmp.Chmod(0o600); errChmod != nil {
		return errors.Join(errChmod, tmp.Close())
	}
	if _, errWrite := tmp.Write(data); errWrite != nil {
		return errors.Join(errWrite, tmp.Close())
	}
	if errSync := tmp.Sync(); errSync != nil {
		return errors.Join(errSync, tmp.Close())
	}
	if errClose := tmp.Close(); errClose != nil {
		return errClose
	}
	if errRename := os.Rename(tmpName, path); errRename != nil {
		return errRename
	}
	removeTemp = false
	return nil
}

func readXAIProxyPrivateFile(path string, maxSize int64) ([]byte, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false, fmt.Errorf("private file path is empty")
	}
	info, errInfo := os.Lstat(path)
	if errors.Is(errInfo, os.ErrNotExist) {
		return nil, false, nil
	}
	if errInfo != nil {
		return nil, false, errInfo
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() > maxSize {
		return nil, true, fmt.Errorf("private file has unsafe type, permissions, or size")
	}
	data, errRead := os.ReadFile(path)
	if errRead != nil {
		return nil, true, errRead
	}
	return data, true, nil
}

func subscriptionError(status int, code string, message string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "subscription_operation_failed"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "xAI subscription operation failed"
	}
	return &XAIProxySubscriptionError{Code: code, Message: message, Status: status}
}

func subscriptionErrorCode(err error, fallback string) string {
	var subscriptionErr *XAIProxySubscriptionError
	if errors.As(err, &subscriptionErr) && subscriptionErr != nil && strings.TrimSpace(subscriptionErr.Code) != "" {
		return subscriptionErr.Code
	}
	return strings.TrimSpace(fallback)
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
