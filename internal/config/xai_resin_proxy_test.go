package config

import "testing"

func TestParseConfigBytesNormalizesXAIResinProxy(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`
xai-resin-proxy:
  enabled: true
  proxy-url: " socks5h://resin:2260/ "
  platform: " XAI "
  proxy-token-file: " /run/secrets/resin-proxy-token "
  identity-key-file: " /run/secrets/resin-identity-key "
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	resin := cfg.XAIResinProxy
	if !resin.Enabled {
		t.Fatal("Resin proxy was not enabled")
	}
	if resin.ProxyURL != "socks5h://resin:2260" {
		t.Fatalf("proxy URL = %q", resin.ProxyURL)
	}
	if resin.Platform != "XAI" {
		t.Fatalf("platform = %q", resin.Platform)
	}
	if resin.ProxyTokenFile != "/run/secrets/resin-proxy-token" {
		t.Fatalf("proxy token file = %q", resin.ProxyTokenFile)
	}
	if resin.IdentityKeyFile != "/run/secrets/resin-identity-key" {
		t.Fatalf("identity key file = %q", resin.IdentityKeyFile)
	}
}

func TestParseConfigBytesDefaultsXAIResinPlatform(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte("port: 8317\n"))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	if cfg.XAIResinProxy.Enabled {
		t.Fatal("xAI Resin proxy unexpectedly enabled")
	}
	if cfg.XAIResinProxy.Platform != "Default" {
		t.Fatalf("default platform = %q", cfg.XAIResinProxy.Platform)
	}
}

func TestParseConfigBytesRejectsCredentialedXAIResinProxyURL(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`
xai-resin-proxy:
  enabled: true
  proxy-url: "http://user:secret@resin:2260"
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	if cfg.XAIResinProxy.ProxyURL != "" {
		t.Fatalf("proxy URL = %q, want invalid URL removed", cfg.XAIResinProxy.ProxyURL)
	}
}
