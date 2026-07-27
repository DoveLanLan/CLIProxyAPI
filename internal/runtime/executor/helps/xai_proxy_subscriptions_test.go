package helps

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"gopkg.in/yaml.v3"
)

type fakeXAIProxySubscriptionController struct {
	mu          sync.Mutex
	active      []string
	nodeCounts  map[string]int
	reloadCount int
	reloadErrAt map[int]error
	refreshErr  error
	selected    map[string]string
}

func (f *fakeXAIProxySubscriptionController) Snapshot(context.Context) ([]xaiProxyNode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var nodes []xaiProxyNode
	for _, provider := range f.active {
		count := f.nodeCounts[provider]
		for i := 0; i < count; i++ {
			nodes = append(nodes, xaiProxyNode{
				Name: fmt.Sprintf("[%s] node-%d", provider, i+1), Provider: provider, Alive: true, Delay: 10 + i,
			})
		}
	}
	return nodes, nil
}

func (f *fakeXAIProxySubscriptionController) RefreshProviders(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshErr
}

func (f *fakeXAIProxySubscriptionController) RefreshProvider(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshErr
}

func (f *fakeXAIProxySubscriptionController) Select(_ context.Context, selector string, node string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.selected == nil {
		f.selected = make(map[string]string)
	}
	f.selected[selector] = node
	return nil
}

func (f *fakeXAIProxySubscriptionController) CheckNode(context.Context, string) (bool, error) {
	return true, nil
}

func (f *fakeXAIProxySubscriptionController) EgressIP(_ context.Context, _ string, _ string, node string, _ []string) (string, error) {
	sum := sha256Bytes(node)
	return fmt.Sprintf("198.51.100.%d", int(sum[0])%200+1), nil
}

func (f *fakeXAIProxySubscriptionController) ReloadConfig(_ context.Context, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reloadCount++
	if errReload := f.reloadErrAt[f.reloadCount]; errReload != nil {
		return errReload
	}
	var decoded struct {
		Providers map[string]any `yaml:"proxy-providers"`
	}
	if errDecode := yaml.Unmarshal(payload, &decoded); errDecode != nil {
		return errDecode
	}
	f.active = f.active[:0]
	for name := range decoded.Providers {
		f.active = append(f.active, name)
	}
	sortStrings(f.active)
	return nil
}

func sha256Bytes(value string) [32]byte {
	return sha256.Sum256([]byte(value))
}

func sortStrings(values []string) {
	sort.Strings(values)
}

func testXAIProxySubscriptionPool(t *testing.T, controller *fakeXAIProxySubscriptionController) (*xaiProxyPool, config.XAIProxyPoolConfig) {
	return testXAIProxySubscriptionPoolWithRouting(t, controller, false)
}

func testXAIProxySubscriptionPoolWithRouting(t *testing.T, controller *fakeXAIProxySubscriptionController, poolEnabled bool) (*xaiProxyPool, config.XAIProxyPoolConfig) {
	t.Helper()
	dir := t.TempDir()
	secretFile := filepath.Join(dir, "controller-secret")
	if errWrite := os.WriteFile(secretFile, []byte("test-controller-secret\n"), 0o600); errWrite != nil {
		t.Fatalf("write controller secret: %v", errWrite)
	}
	cfg := testXAIProxyPoolConfig(filepath.Join(dir, "state.json"), 1)
	cfg.Enabled = poolEnabled
	cfg.ControllerURL = "http://mihomo:9090"
	cfg.ControllerSecretFile = secretFile
	cfg.SubscriptionManagement = config.XAIProxySubscriptionManagementConfig{
		Enabled:             true,
		RegistryFile:        filepath.Join(dir, "private", "subscriptions.json"),
		GeneratedConfigFile: filepath.Join(dir, "shared", "config.yaml"),
		ActivationTimeout:   "75ms",
		MaxProviders:        8,
		MaxURLLength:        4096,
		MaxDownloadBytes:    1 << 20,
	}
	pool := newXAIProxyPoolWithController(cfg, controller, time.Now, false)
	t.Cleanup(pool.Close)
	if pool.subscriptions == nil || !pool.subscriptions.ready {
		t.Fatalf("subscription manager not ready: %#v", pool.subscriptions)
	}
	return pool, cfg
}

func TestXAIProxySubscriptionDisabledDraftPersistsWriteOnlyURL(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	secretURL := "https://subscription.example.com/api?token=super-secret-token"
	result, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: secretURL, Enabled: false,
	})
	if errCreate != nil {
		t.Fatalf("CreateXAIProxySubscription() error = %v", errCreate)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), secretURL) || strings.Contains(string(encoded), "super-secret-token") {
		t.Fatalf("redacted result leaked URL: %s", encoded)
	}
	if result.Revision != 1 || len(result.Subscriptions) != 1 || result.Subscriptions[0].State != "disabled" {
		t.Fatalf("result = %#v", result)
	}
	controller.mu.Lock()
	reloads := controller.reloadCount
	controller.mu.Unlock()
	if reloads != 0 {
		t.Fatalf("disabled draft triggered %d reloads", reloads)
	}
	data, errRead := os.ReadFile(cfg.SubscriptionManagement.RegistryFile)
	if errRead != nil || !strings.Contains(string(data), secretURL) {
		t.Fatalf("registry data/error = %q/%v", data, errRead)
	}
	assertPrivateFileMode(t, cfg.SubscriptionManagement.RegistryFile)
	if _, errStat := os.Stat(cfg.SubscriptionManagement.GeneratedConfigFile); !errors.Is(errStat, os.ErrNotExist) {
		t.Fatalf("disabled draft unexpectedly generated config: %v", errStat)
	}
}

func TestXAIProxySubscriptionEnabledCreateReloadsAndCommits(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 2}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	secretURL := "https://subscription.example.com/api?token=enabled-secret"
	result, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: secretURL, Enabled: true,
	})
	if errCreate != nil {
		t.Fatalf("CreateXAIProxySubscription() error = %v", errCreate)
	}
	if result.Revision != 1 || result.Subscriptions[0].NodeCount != 2 || result.Subscriptions[0].State != "ready" {
		t.Fatalf("result = %#v", result)
	}
	generated, errRead := os.ReadFile(cfg.SubscriptionManagement.GeneratedConfigFile)
	if errRead != nil || !strings.Contains(string(generated), secretURL) || !strings.Contains(string(generated), "test-controller-secret") {
		t.Fatalf("generated config missing private inputs: %v", errRead)
	}
	assertPrivateFileMode(t, cfg.SubscriptionManagement.GeneratedConfigFile)
	controller.mu.Lock()
	reloads := controller.reloadCount
	controller.mu.Unlock()
	if reloads != 1 {
		t.Fatalf("reload count = %d, want 1", reloads)
	}
}

func TestXAIProxySubscriptionActivationFailureRollsBack(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-empty": 0}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	secretURL := "https://subscription.example.com/api?token=never-log-this"
	_, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-empty", URL: secretURL, Enabled: true,
	})
	var subscriptionErr *XAIProxySubscriptionError
	if !errors.As(errCreate, &subscriptionErr) || subscriptionErr.Code != "provider_activation_failed" || strings.Contains(errCreate.Error(), "never-log-this") {
		t.Fatalf("activation error = %#v", errCreate)
	}
	if result := pool.XAIProxySubscriptions(context.Background()); result.Revision != 0 || len(result.Subscriptions) != 0 {
		t.Fatalf("registry changed after failed activation: %#v", result)
	}
	controller.mu.Lock()
	reloads := controller.reloadCount
	active := append([]string(nil), controller.active...)
	controller.mu.Unlock()
	if reloads != 2 || len(active) != 0 {
		t.Fatalf("reloads/active after rollback = %d/%#v", reloads, active)
	}
	if _, errStat := os.Stat(cfg.SubscriptionManagement.RegistryFile); !errors.Is(errStat, os.ErrNotExist) {
		t.Fatalf("failed activation persisted registry: %v", errStat)
	}
}

func TestXAIProxySubscriptionRefreshFailureRollsBack(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{
		nodeCounts: map[string]int{"provider-a": 1},
		refreshErr: errors.New("subscription refresh rejected"),
	}
	pool, _ := testXAIProxySubscriptionPool(t, controller)
	_, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: "https://subscription.example.com/a?token=write-only", Enabled: true,
	})
	if xaiSubscriptionErrorCode(errCreate) != "provider_refresh_failed" || strings.Contains(errCreate.Error(), "write-only") {
		t.Fatalf("refresh error = %#v", errCreate)
	}
	controller.mu.Lock()
	reloads := controller.reloadCount
	active := append([]string(nil), controller.active...)
	controller.mu.Unlock()
	if reloads != 2 || len(active) != 0 {
		t.Fatalf("runtime was not rolled back: reloads=%d active=%#v", reloads, active)
	}
}

func TestXAIProxySubscriptionCandidateReloadFailureRollsBack(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{
		nodeCounts:  map[string]int{"provider-a": 1},
		reloadErrAt: map[int]error{1: errors.New("candidate reload rejected")},
	}
	pool, _ := testXAIProxySubscriptionPool(t, controller)
	_, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: "https://subscription.example.com/a", Enabled: true,
	})
	if xaiSubscriptionErrorCode(errCreate) != "candidate_reload_failed" {
		t.Fatalf("reload error = %#v", errCreate)
	}
	controller.mu.Lock()
	reloads := controller.reloadCount
	active := append([]string(nil), controller.active...)
	controller.mu.Unlock()
	if reloads != 2 || len(active) != 0 {
		t.Fatalf("runtime was not restored after rejected reload: reloads=%d active=%#v", reloads, active)
	}
}

func TestXAIProxySubscriptionFailedUpdatePreservesPreviousURL(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	oldURL := "https://subscription.example.com/old?token=old-secret"
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: oldURL, Enabled: true,
	}); errCreate != nil {
		t.Fatalf("create error = %v", errCreate)
	}
	controller.mu.Lock()
	controller.nodeCounts["provider-a"] = 0
	controller.mu.Unlock()
	newURL := "https://subscription.example.com/new?token=new-secret"
	_, errUpdate := pool.UpdateXAIProxySubscription(context.Background(), 1, "provider-a", XAIProxySubscriptionUpdate{URL: &newURL})
	if xaiSubscriptionErrorCode(errUpdate) != "provider_activation_failed" || strings.Contains(errUpdate.Error(), "new-secret") {
		t.Fatalf("update error = %#v", errUpdate)
	}
	registryData, errRead := os.ReadFile(cfg.SubscriptionManagement.RegistryFile)
	if errRead != nil || !strings.Contains(string(registryData), oldURL) || strings.Contains(string(registryData), newURL) {
		t.Fatalf("registry did not preserve old URL: %v", errRead)
	}
	generatedData, errReadGenerated := os.ReadFile(cfg.SubscriptionManagement.GeneratedConfigFile)
	if errReadGenerated != nil || !strings.Contains(string(generatedData), oldURL) || strings.Contains(string(generatedData), newURL) {
		t.Fatalf("generated config did not preserve old URL: %v", errReadGenerated)
	}
	if result := pool.XAIProxySubscriptions(context.Background()); result.Revision != 1 || result.Subscriptions[0].Fingerprint != xaiSubscriptionFingerprint(oldURL) {
		t.Fatalf("runtime registry changed after failed update: %#v", result)
	}
}

func TestXAIProxySubscriptionRollbackFailureIsExplicit(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{
		nodeCounts:  map[string]int{"provider-empty": 0},
		reloadErrAt: map[int]error{2: errors.New("rollback rejected")},
	}
	pool, _ := testXAIProxySubscriptionPool(t, controller)
	_, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-empty", URL: "https://subscription.example.com/empty", Enabled: true,
	})
	var subscriptionErr *XAIProxySubscriptionError
	if !errors.As(errCreate, &subscriptionErr) || subscriptionErr.Code != "rollback_failed" || subscriptionErr.StatusCode() != 500 {
		t.Fatalf("rollback error = %#v", errCreate)
	}
}

func TestXAIProxySubscriptionRegistryPersistFailureRollsBackRuntime(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	registryDirectory := filepath.Join(t.TempDir(), "registry-directory")
	if errMkdir := os.Mkdir(registryDirectory, 0o700); errMkdir != nil {
		t.Fatalf("mkdir registry directory: %v", errMkdir)
	}
	pool.subscriptions.management.RegistryFile = registryDirectory
	_, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: "https://subscription.example.com/a", Enabled: true,
	})
	if xaiSubscriptionErrorCode(errCreate) != "registry_persist_failed" {
		t.Fatalf("persist error = %#v", errCreate)
	}
	controller.mu.Lock()
	reloads := controller.reloadCount
	active := append([]string(nil), controller.active...)
	controller.mu.Unlock()
	if reloads != 2 || len(active) != 0 {
		t.Fatalf("runtime was not rolled back: reloads=%d active=%#v", reloads, active)
	}
	if _, errStat := os.Stat(cfg.SubscriptionManagement.GeneratedConfigFile); !errors.Is(errStat, os.ErrNotExist) {
		t.Fatalf("generated config was not restored: %v", errStat)
	}
	if result := pool.XAIProxySubscriptions(context.Background()); result.Revision != 0 || len(result.Subscriptions) != 0 {
		t.Fatalf("in-memory registry changed: %#v", result)
	}
}

func TestXAIProxySubscriptionDisableThenDelete(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: "https://subscription.example.com/a", Enabled: true,
	}); errCreate != nil {
		t.Fatalf("create error = %v", errCreate)
	}
	if _, errDelete := pool.DeleteXAIProxySubscription(context.Background(), 1, "provider-a"); xaiSubscriptionErrorCode(errDelete) != "subscription_enabled" {
		t.Fatalf("enabled delete error = %#v", errDelete)
	}
	cachePath := filepath.Join(filepath.Dir(cfg.SubscriptionManagement.GeneratedConfigFile), "providers", "provider-a.yaml")
	if errMkdir := os.MkdirAll(filepath.Dir(cachePath), 0o700); errMkdir != nil {
		t.Fatalf("mkdir provider cache: %v", errMkdir)
	}
	if errWrite := os.WriteFile(cachePath, []byte("proxies: []\n"), 0o600); errWrite != nil {
		t.Fatalf("write provider cache: %v", errWrite)
	}
	disabled := false
	result, errDisable := pool.UpdateXAIProxySubscription(context.Background(), 1, "provider-a", XAIProxySubscriptionUpdate{Enabled: &disabled})
	if errDisable != nil || result.Revision != 2 || result.Subscriptions[0].Enabled {
		t.Fatalf("disable result/error = %#v/%v", result, errDisable)
	}
	if _, errStat := os.Stat(cachePath); errStat != nil {
		t.Fatalf("provider cache should remain until hard delete: %v", errStat)
	}
	result, errDelete := pool.DeleteXAIProxySubscription(context.Background(), 2, "provider-a")
	if errDelete != nil || result.Revision != 3 || len(result.Subscriptions) != 0 {
		t.Fatalf("delete result/error = %#v/%v", result, errDelete)
	}
	if _, errStat := os.Stat(cachePath); !errors.Is(errStat, os.ErrNotExist) {
		t.Fatalf("provider cache was not removed by hard delete: %v", errStat)
	}
}

func TestXAIProxySubscriptionDisableDrainsActiveLane(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1, "provider-b": 1}}
	pool, _ := testXAIProxySubscriptionPoolWithRouting(t, controller, true)
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: "https://subscription.example.com/a", Enabled: true,
	}); errCreate != nil {
		t.Fatalf("create provider-a error = %v", errCreate)
	}
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 1, XAIProxySubscriptionCreate{
		Name: "provider-b", URL: "https://subscription.example.com/b", Enabled: true,
	}); errCreate != nil {
		t.Fatalf("create provider-b error = %v", errCreate)
	}
	pool.mu.RLock()
	before := pool.lanes[pool.laneOrder[0]].Provider
	pool.mu.RUnlock()
	if before != "provider-a" {
		t.Fatalf("initial lane provider = %q, want provider-a", before)
	}
	disabled := false
	result, errDisable := pool.UpdateXAIProxySubscription(context.Background(), 2, "provider-a", XAIProxySubscriptionUpdate{Enabled: &disabled})
	if errDisable != nil || result.Revision != 3 {
		t.Fatalf("disable result/error = %#v/%v", result, errDisable)
	}
	pool.mu.RLock()
	after := pool.lanes[pool.laneOrder[0]].Provider
	pool.mu.RUnlock()
	if after != "provider-b" || pool.providerInUse("provider-a") {
		t.Fatalf("lane did not drain: before=%q after=%q", before, after)
	}
	if _, errDelete := pool.DeleteXAIProxySubscription(context.Background(), 3, "provider-a"); errDelete != nil {
		t.Fatalf("delete drained provider error = %v", errDelete)
	}
}

func TestXAIProxySubscriptionDeleteRejectsActiveLane(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{}}
	pool, _ := testXAIProxySubscriptionPool(t, controller)
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: "https://subscription.example.com/a", Enabled: false,
	}); errCreate != nil {
		t.Fatalf("create error = %v", errCreate)
	}
	pool.mu.Lock()
	pool.lanes[pool.laneOrder[0]].Provider = "provider-a"
	pool.lanes[pool.laneOrder[0]].Ready = true
	pool.mu.Unlock()
	_, errDelete := pool.DeleteXAIProxySubscription(context.Background(), 1, "provider-a")
	if xaiSubscriptionErrorCode(errDelete) != "subscription_in_use" {
		t.Fatalf("in-use delete error = %#v", errDelete)
	}
}

func TestXAIProxySubscriptionConcurrentRevisionPreventsLostUpdate(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{}}
	pool, _ := testXAIProxySubscriptionPool(t, controller)
	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, name := range []string{"provider-a", "provider-b"} {
		name := name
		go func() {
			<-start
			_, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
				Name: name, URL: "https://subscription.example.com/" + name, Enabled: false,
			})
			errs <- errCreate
		}()
	}
	close(start)
	firstErr, secondErr := <-errs, <-errs
	successes := 0
	mismatches := 0
	for _, errCreate := range []error{firstErr, secondErr} {
		if errCreate == nil {
			successes++
		} else if xaiSubscriptionErrorCode(errCreate) == "revision_mismatch" {
			mismatches++
		}
	}
	if successes != 1 || mismatches != 1 {
		t.Fatalf("successes/mismatches/errors = %d/%d/%v/%v", successes, mismatches, firstErr, secondErr)
	}
	if result := pool.XAIProxySubscriptions(context.Background()); result.Revision != 1 || len(result.Subscriptions) != 1 {
		t.Fatalf("final registry = %#v", result)
	}
}

func TestXAIProxySubscriptionCheckStatusIsRedacted(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1}}
	pool, _ := testXAIProxySubscriptionPool(t, controller)
	secretURL := "https://subscription.example.com/a?token=never-return"
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: secretURL, Enabled: true,
	}); errCreate != nil {
		t.Fatalf("create error = %v", errCreate)
	}
	controller.mu.Lock()
	controller.refreshErr = errors.New("refresh failed for " + secretURL)
	controller.mu.Unlock()
	if _, errCheck := pool.CheckXAIProxySubscription(context.Background(), "provider-a"); xaiSubscriptionErrorCode(errCheck) != "provider_refresh_failed" {
		t.Fatalf("check error = %#v", errCheck)
	}
	status := pool.XAIProxySubscriptions(context.Background()).Subscriptions[0]
	encoded, errMarshal := json.Marshal(status)
	if errMarshal != nil {
		t.Fatalf("marshal status: %v", errMarshal)
	}
	if status.LastCheckAt == nil || status.LastErrorCode != "provider_refresh_failed" || strings.Contains(string(encoded), secretURL) || strings.Contains(string(encoded), "never-return") {
		t.Fatalf("redacted check status = %s", encoded)
	}
}

func TestXAIProxySubscriptionStartupRestoresRegistrySourceOfTruth(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	registryURL := "https://subscription.example.com/registry?token=registry-secret"
	if _, errCreate := pool.CreateXAIProxySubscription(context.Background(), 0, XAIProxySubscriptionCreate{
		Name: "provider-a", URL: registryURL, Enabled: true,
	}); errCreate != nil {
		t.Fatalf("create error = %v", errCreate)
	}
	pool.Close()
	stalePayload := []byte("secret: stale\nproxy-providers:\n  provider-stale:\n    type: http\n    url: https://stale.example.com/source\n")
	if errWrite := os.WriteFile(cfg.SubscriptionManagement.GeneratedConfigFile, stalePayload, 0o600); errWrite != nil {
		t.Fatalf("write stale generated config: %v", errWrite)
	}
	restartedController := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{"provider-a": 1}}
	restarted, errRestart := NewXAIProxyPoolWithController(context.Background(), cfg, restartedController, time.Now)
	if errRestart != nil {
		t.Fatalf("restart error = %v", errRestart)
	}
	defer restarted.Close()
	generated, errRead := os.ReadFile(cfg.SubscriptionManagement.GeneratedConfigFile)
	if errRead != nil || !strings.Contains(string(generated), registryURL) || strings.Contains(string(generated), "provider-stale") {
		t.Fatalf("generated config was not restored from registry: %v", errRead)
	}
	restartedController.mu.Lock()
	reloads := restartedController.reloadCount
	active := append([]string(nil), restartedController.active...)
	restartedController.mu.Unlock()
	if reloads != 1 || len(active) != 1 || active[0] != "provider-a" {
		t.Fatalf("startup reloads/active = %d/%#v", reloads, active)
	}
}

func TestXAIProxySubscriptionRejectsConflictingStoragePaths(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	pool.Close()
	cfg.SubscriptionManagement.GeneratedConfigFile = cfg.SubscriptionManagement.RegistryFile
	conflicted := newXAIProxyPoolWithController(cfg, controller, time.Now, false)
	defer conflicted.Close()
	status := conflicted.XAIProxySubscriptions(context.Background())
	if status.Ready || status.LastErrorCode != "storage_paths_conflict" {
		t.Fatalf("conflicting path status = %#v", status)
	}
}

func TestMihomoXAIProxyControllerReloadAndProviderRefresh(t *testing.T) {
	const controllerSecret = "controller-test-secret"
	var reloadPayload string
	var refreshCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+controllerSecret {
			t.Errorf("authorization header is missing")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/configs" && r.URL.Query().Get("force") == "true":
			var body struct {
				Payload string `json:"payload"`
			}
			if errDecode := json.NewDecoder(r.Body).Decode(&body); errDecode != nil {
				t.Errorf("decode reload payload: %v", errDecode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reloadPayload = body.Payload
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/providers/proxies/provider-a":
			refreshCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected controller request: %s %s", r.Method, r.URL.RequestURI())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	controller := &mihomoXAIProxyController{
		baseURL: server.URL, secret: controllerSecret, httpClient: server.Client(),
	}
	wantPayload := "proxy-providers: {}\n"
	if errReload := controller.ReloadConfig(context.Background(), []byte(wantPayload)); errReload != nil {
		t.Fatalf("ReloadConfig() error = %v", errReload)
	}
	if errRefresh := controller.RefreshProvider(context.Background(), "provider-a"); errRefresh != nil {
		t.Fatalf("RefreshProvider() error = %v", errRefresh)
	}
	if reloadPayload != wantPayload || !refreshCalled {
		t.Fatalf("reload payload/refresh = %q/%v", reloadPayload, refreshCalled)
	}
}

func TestValidateXAIProxySubscriptionURL(t *testing.T) {
	valid, errValid := validateXAIProxySubscriptionURL(" https://Sub.Example.com/path?token=secret ", 4096)
	if errValid != nil || valid != "https://sub.example.com/path?token=secret" {
		t.Fatalf("valid URL/error = %q/%v", valid, errValid)
	}
	for _, raw := range []string{
		"http://sub.example.com/a",
		"https://user:pass@sub.example.com/a",
		"https://sub.example.com/a#fragment",
		"https://127.0.0.1/a",
		"https://10.0.0.1/a",
		"https://[fe80::1%25eth0]/a",
		"https://intranet/a",
		"https://sub.example.com:70000/a",
		"https://service.local/a",
	} {
		if _, errURL := validateXAIProxySubscriptionURL(raw, 4096); errURL == nil {
			t.Fatalf("unsafe URL accepted: %s", raw)
		}
	}
}

func TestXAIProxySubscriptionRegistryRejectsSymlink(t *testing.T) {
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{}}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if errWrite := os.WriteFile(target, []byte(`{"version":1,"revision":0,"subscriptions":[]}`), 0o600); errWrite != nil {
		t.Fatalf("write target: %v", errWrite)
	}
	link := filepath.Join(dir, "registry.json")
	if errLink := os.Symlink(target, link); errLink != nil {
		t.Fatalf("symlink: %v", errLink)
	}
	secretFile := filepath.Join(dir, "secret")
	if errWrite := os.WriteFile(secretFile, []byte("secret"), 0o600); errWrite != nil {
		t.Fatalf("write secret: %v", errWrite)
	}
	cfg := testXAIProxyPoolConfig(filepath.Join(dir, "state.json"), 1)
	cfg.Enabled = false
	cfg.ControllerURL = "http://mihomo:9090"
	cfg.ControllerSecretFile = secretFile
	cfg.SubscriptionManagement = config.XAIProxySubscriptionManagementConfig{
		Enabled: true, RegistryFile: link, GeneratedConfigFile: filepath.Join(dir, "config.yaml"),
		ActivationTimeout: "100ms", MaxProviders: 8, MaxURLLength: 4096, MaxDownloadBytes: 1 << 20,
	}
	pool := newXAIProxyPoolWithController(cfg, controller, time.Now, false)
	defer pool.Close()
	if status := pool.XAIProxySubscriptions(context.Background()); status.Ready || status.LastErrorCode != "registry_load_failed" {
		t.Fatalf("symlink registry status = %#v", status)
	}
}

func TestXAIProxyGeneratedMihomoConfigPinnedImage(t *testing.T) {
	if os.Getenv("CLIPROXY_MIHOMO_INTEGRATION") != "1" {
		t.Skip("set CLIPROXY_MIHOMO_INTEGRATION=1 to run pinned Mihomo validation")
	}
	controller := &fakeXAIProxySubscriptionController{nodeCounts: map[string]int{}}
	pool, cfg := testXAIProxySubscriptionPool(t, controller)
	registry := xaiProxySubscriptionRegistry{
		Version: 1,
		Subscriptions: []xaiProxySubscriptionEntry{{
			Name: "provider-a", URL: "https://subscription.example.invalid/source", Enabled: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}},
	}
	payload, errRender := pool.subscriptions.renderConfig(registry)
	if errRender != nil {
		t.Fatalf("renderConfig() error = %v", errRender)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if errWrite := os.WriteFile(configPath, payload, 0o600); errWrite != nil {
		t.Fatalf("write generated config: %v", errWrite)
	}
	image := os.Getenv("MIHOMO_IMAGE")
	if image == "" {
		image = "docker.io/metacubex/mihomo:v1.19.28@sha256:e6acd921addecfd59a8e2d38203f88356d635b54de6c0673db0e015139989312"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-v", configPath+":/root/.config/mihomo/config.yaml:ro", image, "-t", "-d", "/root/.config/mihomo")
	output, errRun := cmd.CombinedOutput()
	if errRun != nil {
		t.Fatalf("pinned Mihomo validation failed: %v: %s\nconfig=%s", errRun, output, cfg.SubscriptionManagement.GeneratedConfigFile)
	}
}

func assertPrivateFileMode(t *testing.T, path string) {
	t.Helper()
	info, errStat := os.Stat(path)
	if errStat != nil {
		t.Fatalf("stat %s: %v", path, errStat)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %s = %o", path, info.Mode().Perm())
	}
}

func xaiSubscriptionErrorCode(err error) string {
	var subscriptionErr *XAIProxySubscriptionError
	if errors.As(err, &subscriptionErr) && subscriptionErr != nil {
		return subscriptionErr.Code
	}
	return ""
}
