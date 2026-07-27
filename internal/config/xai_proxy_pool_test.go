package config

import "testing"

func TestParseConfigBytesNormalizesXAIProxyPool(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`
xai-proxy-pool:
  enabled: true
  rollout-percent: 101
  controller-url: " http://mihomo:9090/ "
  controller-secret-file: " /run/secrets/controller "
  state-file: " "
  health-check-timeout: bad
  ip-check-urls:
    - "https://api.ipify.org/"
    - "https://api.ipify.org"
    - "file:///tmp/nope"
  lanes:
    - {name: " lane-1 ", proxy-url: " http://mihomo:17891 ", selector: " xai-lane-1 "}
    - {name: "LANE-1", proxy-url: "http://mihomo:17892", selector: "duplicate"}
    - {name: "broken", proxy-url: "file:///tmp/proxy", selector: "broken"}
  probe: {name: " probe ", proxy-url: "http://mihomo:17899", selector: " xai-probe "}
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	pool := cfg.XAIProxyPool
	if !pool.Enabled || pool.RolloutPercent != 100 {
		t.Fatalf("pool enabled/rollout = %v/%d", pool.Enabled, pool.RolloutPercent)
	}
	if pool.ControllerURL != "http://mihomo:9090" {
		t.Fatalf("controller URL = %q", pool.ControllerURL)
	}
	if pool.ControllerSecretFile != "/run/secrets/controller" {
		t.Fatalf("controller secret file = %q", pool.ControllerSecretFile)
	}
	if pool.StateFile != DefaultXAIProxyPoolStateFile {
		t.Fatalf("state file = %q", pool.StateFile)
	}
	if pool.HealthCheckTimeout != "5s" {
		t.Fatalf("health check timeout = %q", pool.HealthCheckTimeout)
	}
	if len(pool.IPCheckURLs) != 1 || pool.IPCheckURLs[0] != "https://api.ipify.org" {
		t.Fatalf("IP check URLs = %#v", pool.IPCheckURLs)
	}
	if len(pool.Lanes) != 1 {
		t.Fatalf("lanes = %#v", pool.Lanes)
	}
	if got := pool.Lanes[0]; got.Name != "lane-1" || got.ProxyURL != "http://mihomo:17891" || got.Selector != "xai-lane-1" {
		t.Fatalf("lane = %#v", got)
	}
	if pool.Probe.Name != "probe" || pool.Probe.Selector != "xai-probe" {
		t.Fatalf("probe = %#v", pool.Probe)
	}
	if pool.RequestsPerMinutePerLane != 30 || pool.BurstPerLane != 3 || pool.QueueSizePerLane != 30 {
		t.Fatalf("rate defaults = %d/%d/%d", pool.RequestsPerMinutePerLane, pool.BurstPerLane, pool.QueueSizePerLane)
	}
}

func TestParseConfigBytesKeepsXAIProxyPoolDisabledByDefault(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte("port: 8317\n"))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	if cfg.XAIProxyPool.Enabled {
		t.Fatal("xAI proxy pool unexpectedly enabled")
	}
	if cfg.XAIProxyPool.RolloutPercent != 100 {
		t.Fatalf("rollout = %d, want normalized 100", cfg.XAIProxyPool.RolloutPercent)
	}
}

func TestParseConfigBytesNormalizesXAIProxySubscriptionManagement(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`
xai-proxy-pool:
  subscription-management:
    enabled: true
    registry-file: " /private/subscriptions.json "
    generated-config-file: " /shared/mihomo/config.yaml "
    activation-timeout: bad
    max-providers: 0
    max-url-length: -1
    max-download-bytes: 0
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	management := cfg.XAIProxyPool.SubscriptionManagement
	if !management.Enabled || management.RegistryFile != "/private/subscriptions.json" || management.GeneratedConfigFile != "/shared/mihomo/config.yaml" {
		t.Fatalf("subscription management = %#v", management)
	}
	if management.ActivationTimeout != "30s" || management.MaxProviders != 64 || management.MaxURLLength != 4096 || management.MaxDownloadBytes != 8<<20 {
		t.Fatalf("subscription defaults = %#v", management)
	}
}
