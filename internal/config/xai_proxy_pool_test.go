package config

import "testing"

func TestParseConfigBytesNormalizesStandaloneXAIProxyPool(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`
xai-proxy-pool:
  enabled: true
  service-url: " http://egress-proxy-controller:8080/ "
  service-token-file: " /run/secrets/egress-proxy-api-token "
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	pool := cfg.XAIProxyPool
	if !pool.Enabled {
		t.Fatal("pool was not enabled")
	}
	if pool.ServiceURL != "http://egress-proxy-controller:8080" {
		t.Fatalf("service URL = %q", pool.ServiceURL)
	}
	if pool.ServiceTokenFile != "/run/secrets/egress-proxy-api-token" {
		t.Fatalf("service token file = %q", pool.ServiceTokenFile)
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
}

func TestParseConfigBytesRejectsNonHTTPProxyPoolServiceURL(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`
xai-proxy-pool:
  enabled: true
  service-url: "file:///tmp/control.sock"
  service-token-file: "/run/secrets/token"
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	if cfg.XAIProxyPool.ServiceURL != "" {
		t.Fatalf("service URL = %q", cfg.XAIProxyPool.ServiceURL)
	}
}
