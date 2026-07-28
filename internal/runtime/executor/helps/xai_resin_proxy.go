package helps

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

const xaiResinAccountDigestBytes = 16

// XAIResinProxyError is request-scoped so a Resin configuration or availability
// problem does not cool down an otherwise valid xAI credential.
type XAIResinProxyError struct {
	Message string
	Retry   time.Duration
}

func (e *XAIResinProxyError) Error() string {
	if e == nil || strings.TrimSpace(e.Message) == "" {
		return "xai Resin proxy is unavailable"
	}
	return e.Message
}

func (e *XAIResinProxyError) StatusCode() int { return http.StatusServiceUnavailable }

func (e *XAIResinProxyError) IsRequestScoped() bool { return true }

func (e *XAIResinProxyError) ManagedHeaders() http.Header {
	retry := 30 * time.Second
	if e != nil && e.Retry > 0 {
		retry = e.Retry
	}
	return http.Header{"Retry-After": {fmt.Sprintf("%.0f", retry.Seconds())}}
}

// XAIResinProxy derives credentialed Resin forward-proxy URLs without exposing
// raw CPA auth IDs or storing generated proxy credentials on auth records.
type XAIResinProxy struct {
	enabled     bool
	baseURL     *url.URL
	platform    string
	proxyToken  string
	identityKey []byte
	initErr     error
}

// NewXAIResinProxy builds a fail-closed xAI Resin router. automaticRouteConflict
// must be true when another automatic xAI egress backend is enabled.
func NewXAIResinProxy(cfg config.XAIResinProxyConfig, automaticRouteConflict bool) *XAIResinProxy {
	proxy := &XAIResinProxy{enabled: cfg.Enabled}
	if !cfg.Enabled {
		return proxy
	}
	if automaticRouteConflict {
		proxy.initErr = resinUnavailable("xai Resin proxy cannot be enabled with xai proxy pool")
		return proxy
	}

	parsed, errParse := parseXAIResinProxyURL(cfg.ProxyURL)
	if errParse != nil {
		proxy.initErr = errParse
		return proxy
	}
	platform := strings.TrimSpace(cfg.Platform)
	if platform == "" {
		platform = "Default"
	}
	if !validResinV1Platform(platform) {
		proxy.initErr = resinUnavailable("xai Resin proxy platform is invalid for Resin V1")
		return proxy
	}

	proxyTokenBytes, errToken := os.ReadFile(strings.TrimSpace(cfg.ProxyTokenFile))
	proxyToken := strings.TrimSpace(string(proxyTokenBytes))
	if errToken != nil || proxyToken == "" {
		proxy.initErr = resinUnavailable("xai Resin proxy token is unavailable")
		return proxy
	}
	if !validResinV1ProxyToken(proxyToken) {
		proxy.initErr = resinUnavailable("xai Resin proxy token is invalid for Resin V1")
		return proxy
	}

	identityKey, errIdentityKey := os.ReadFile(strings.TrimSpace(cfg.IdentityKeyFile))
	identityKey = bytes.TrimSpace(identityKey)
	if errIdentityKey != nil || len(identityKey) < 32 {
		proxy.initErr = resinUnavailable("xai Resin identity key is unavailable or too short")
		return proxy
	}

	proxy.baseURL = parsed
	proxy.platform = platform
	proxy.proxyToken = proxyToken
	proxy.identityKey = bytes.Clone(identityKey)
	return proxy
}

// ProxyURL returns a per-auth Resin proxy URL. The bool is false when Resin
// routing is disabled and legacy routing should continue.
func (p *XAIResinProxy) ProxyURL(authID string) (string, bool, error) {
	if p == nil || !p.enabled {
		return "", false, nil
	}
	if p.initErr != nil {
		return "", true, p.initErr
	}
	if strings.TrimSpace(authID) == "" {
		return "", true, resinUnavailable("xai Resin proxy requires a stable auth ID")
	}

	mac := hmac.New(sha256.New, p.identityKey)
	_, _ = mac.Write([]byte(authID))
	digest := mac.Sum(nil)
	account := "xai-" + hex.EncodeToString(digest[:xaiResinAccountDigestBytes])

	resolved := *p.baseURL
	resolved.User = url.UserPassword(p.platform+"."+account, p.proxyToken)
	return resolved.String(), true, nil
}

func parseXAIResinProxyURL(raw string) (*url.URL, error) {
	parsed, errParse := url.Parse(strings.TrimSpace(raw))
	if errParse != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, resinUnavailable("xai Resin proxy URL is invalid")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, resinUnavailable("xai Resin proxy URL is invalid")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, resinUnavailable("xai Resin proxy URL uses an unsupported scheme")
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return parsed, nil
}

func validResinV1IdentityPart(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, ".:|/\\@?#%~") {
		return false
	}
	return strings.IndexFunc(value, unicode.IsSpace) < 0
}

func validResinV1Platform(value string) bool {
	return validResinV1IdentityPart(value) && !strings.EqualFold(value, "api")
}

func validResinV1ProxyToken(value string) bool {
	if !validResinV1IdentityPart(value) {
		return false
	}
	switch strings.ToLower(value) {
	case "api", "healthz", "ui":
		return false
	default:
		return true
	}
}

func resinUnavailable(message string) error {
	return &XAIResinProxyError{Message: message, Retry: 30 * time.Second}
}
