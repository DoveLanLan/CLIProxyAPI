package helps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/proxyutil"
	log "github.com/sirupsen/logrus"
)

type xaiProxyNode struct {
	Name     string
	Provider string
	Alive    bool
	Delay    int
}

// XAIProxyNode describes one Mihomo provider node for controller adapters.
type XAIProxyNode = xaiProxyNode

type xaiProxyController interface {
	Snapshot(context.Context) ([]xaiProxyNode, error)
	RefreshProviders(context.Context) error
	Select(context.Context, string, string) error
	CheckNode(context.Context, string) (bool, error)
	EgressIP(context.Context, string, string, string, []string) (string, error)
}

// XAIProxyController is the controller contract used by the proxy-pool runtime.
type XAIProxyController = xaiProxyController

type mihomoXAIProxyController struct {
	baseURL            string
	secret             string
	healthCheckURL     string
	healthCheckTimeout time.Duration
	httpClient         *http.Client
}

type mihomoHTTPStatusError struct {
	Status int
}

func (e *mihomoHTTPStatusError) Error() string {
	return fmt.Sprintf("mihomo returned status %d", e.Status)
}

func newMihomoXAIProxyController(cfg config.XAIProxyPoolConfig) (*mihomoXAIProxyController, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.ControllerURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("xai proxy pool: mihomo controller URL is required")
	}
	secretPath := strings.TrimSpace(cfg.ControllerSecretFile)
	if secretPath == "" {
		return nil, fmt.Errorf("xai proxy pool: mihomo controller secret file is required")
	}
	secretBytes, errRead := os.ReadFile(secretPath)
	if errRead != nil {
		return nil, fmt.Errorf("xai proxy pool: read mihomo controller secret: %w", errRead)
	}
	secret := strings.TrimSpace(string(secretBytes))
	if secret == "" {
		return nil, fmt.Errorf("xai proxy pool: mihomo controller secret is empty")
	}
	timeout := parsePositiveDuration(cfg.HealthCheckTimeout, 5*time.Second)
	return &mihomoXAIProxyController{
		baseURL:            baseURL,
		secret:             secret,
		healthCheckURL:     strings.TrimSpace(cfg.HealthCheckURL),
		healthCheckTimeout: timeout,
		httpClient:         &http.Client{Timeout: timeout + 5*time.Second},
	}, nil
}

func (c *mihomoXAIProxyController) Snapshot(ctx context.Context) ([]xaiProxyNode, error) {
	var payload struct {
		Providers map[string]struct {
			Proxies []struct {
				Name    string `json:"name"`
				Alive   *bool  `json:"alive"`
				History []struct {
					Delay int `json:"delay"`
				} `json:"history"`
			} `json:"proxies"`
		} `json:"providers"`
	}
	if errGet := c.doJSON(ctx, http.MethodGet, "/providers/proxies", nil, &payload); errGet != nil {
		return nil, errGet
	}

	counts := make(map[string]int)
	for _, provider := range payload.Providers {
		for _, proxy := range provider.Proxies {
			name := strings.TrimSpace(proxy.Name)
			if name != "" {
				counts[name]++
			}
		}
	}
	nodes := make([]xaiProxyNode, 0)
	for providerName, provider := range payload.Providers {
		providerName = strings.TrimSpace(providerName)
		for _, proxy := range provider.Proxies {
			name := strings.TrimSpace(proxy.Name)
			if name == "" || counts[name] != 1 {
				continue
			}
			alive := true
			if proxy.Alive != nil {
				alive = *proxy.Alive
			}
			delay := 0
			for i := len(proxy.History) - 1; i >= 0; i-- {
				if proxy.History[i].Delay > 0 {
					delay = proxy.History[i].Delay
					break
				}
			}
			nodes = append(nodes, xaiProxyNode{Name: name, Provider: providerName, Alive: alive, Delay: delay})
		}
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Alive != nodes[j].Alive {
			return nodes[i].Alive
		}
		leftDelay := nodes[i].Delay
		rightDelay := nodes[j].Delay
		if leftDelay == 0 {
			leftDelay = int(^uint(0) >> 1)
		}
		if rightDelay == 0 {
			rightDelay = int(^uint(0) >> 1)
		}
		if leftDelay != rightDelay {
			return leftDelay < rightDelay
		}
		if nodes[i].Provider != nodes[j].Provider {
			return nodes[i].Provider < nodes[j].Provider
		}
		return nodes[i].Name < nodes[j].Name
	})
	return nodes, nil
}

func (c *mihomoXAIProxyController) RefreshProviders(ctx context.Context) error {
	var payload struct {
		Providers map[string]json.RawMessage `json:"providers"`
	}
	if errGet := c.doJSON(ctx, http.MethodGet, "/providers/proxies", nil, &payload); errGet != nil {
		return errGet
	}
	names := make([]string, 0, len(payload.Providers))
	for name := range payload.Providers {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var errs []error
	for _, name := range names {
		path := "/providers/proxies/" + url.PathEscape(name)
		if errRefresh := c.doJSON(ctx, http.MethodPut, path, nil, nil); errRefresh != nil {
			errs = append(errs, fmt.Errorf("refresh provider %q: %w", name, errRefresh))
		}
	}
	return errors.Join(errs...)
}

func (c *mihomoXAIProxyController) RefreshProvider(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("xai proxy pool: provider name is required")
	}
	return c.doJSON(ctx, http.MethodPut, "/providers/proxies/"+url.PathEscape(name), nil, nil)
}

func (c *mihomoXAIProxyController) ReloadConfig(ctx context.Context, payload []byte) error {
	if len(bytes.TrimSpace(payload)) == 0 {
		return fmt.Errorf("xai proxy pool: mihomo configuration payload is empty")
	}
	body, errMarshal := json.Marshal(map[string]string{"payload": string(payload)})
	if errMarshal != nil {
		return fmt.Errorf("xai proxy pool: encode mihomo reload request: %w", errMarshal)
	}
	return c.doJSON(ctx, http.MethodPut, "/configs?force=true", body, nil)
}

func (c *mihomoXAIProxyController) Select(ctx context.Context, selector string, node string) error {
	selector = strings.TrimSpace(selector)
	node = strings.TrimSpace(node)
	if selector == "" || node == "" {
		return fmt.Errorf("xai proxy pool: selector and node are required")
	}
	body, errMarshal := json.Marshal(map[string]string{"name": node})
	if errMarshal != nil {
		return fmt.Errorf("xai proxy pool: encode selector request: %w", errMarshal)
	}
	return c.doJSON(ctx, http.MethodPut, "/proxies/"+url.PathEscape(selector), body, nil)
}

func (c *mihomoXAIProxyController) CheckNode(ctx context.Context, node string) (bool, error) {
	node = strings.TrimSpace(node)
	if node == "" {
		return false, fmt.Errorf("xai proxy pool: node is required")
	}
	query := url.Values{}
	query.Set("url", c.healthCheckURL)
	query.Set("timeout", strconv.FormatInt(c.healthCheckTimeout.Milliseconds(), 10))
	query.Set("expected", "204")
	path := "/proxies/" + url.PathEscape(node) + "/delay?" + query.Encode()
	var payload struct {
		Delay int `json:"delay"`
	}
	if errCheck := c.doJSON(ctx, http.MethodGet, path, nil, &payload); errCheck != nil {
		var statusErr *mihomoHTTPStatusError
		if errors.As(errCheck, &statusErr) && statusErr.Status >= http.StatusBadRequest {
			return false, nil
		}
		return false, errCheck
	}
	return payload.Delay > 0, nil
}

func (c *mihomoXAIProxyController) EgressIP(ctx context.Context, proxyURL string, selector string, node string, checkURLs []string) (string, error) {
	if errSelect := c.Select(ctx, selector, node); errSelect != nil {
		return "", errSelect
	}
	transport, _, errTransport := proxyutil.BuildHTTPTransport(proxyURL)
	if errTransport != nil {
		return "", fmt.Errorf("xai proxy pool: build probe transport: %w", errTransport)
	}
	client := &http.Client{Transport: transport, Timeout: c.healthCheckTimeout}
	var errs []error
	for _, endpoint := range checkURLs {
		req, errRequest := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if errRequest != nil {
			errs = append(errs, errRequest)
			continue
		}
		resp, errDo := client.Do(req)
		if errDo != nil {
			errs = append(errs, errDo)
			continue
		}
		body, errRead := io.ReadAll(io.LimitReader(resp.Body, 4096))
		errClose := resp.Body.Close()
		if errRead != nil {
			errs = append(errs, errRead)
			continue
		}
		if errClose != nil {
			errs = append(errs, errClose)
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			errs = append(errs, fmt.Errorf("IP check returned status %d", resp.StatusCode))
			continue
		}
		if ip := parsePublicIP(body); ip != "" {
			return ip, nil
		}
		errs = append(errs, fmt.Errorf("IP check response did not contain a public IP"))
	}
	return "", fmt.Errorf("xai proxy pool: resolve egress IP: %w", errors.Join(errs...))
}

func (c *mihomoXAIProxyController) doJSON(ctx context.Context, method string, path string, body []byte, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, errRequest := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if errRequest != nil {
		return fmt.Errorf("xai proxy pool: create mihomo request: %w", errRequest)
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, errDo := c.httpClient.Do(req)
	if errDo != nil {
		return fmt.Errorf("xai proxy pool: mihomo request failed: %w", errDo)
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.WithError(errClose).Debug("xai proxy pool: close mihomo response body")
		}
	}()
	responseBody, errRead := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if errRead != nil {
		return fmt.Errorf("xai proxy pool: read mihomo response: %w", errRead)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("xai proxy pool: %w", &mihomoHTTPStatusError{Status: resp.StatusCode})
	}
	if out == nil || len(bytes.TrimSpace(responseBody)) == 0 {
		return nil
	}
	if errDecode := json.Unmarshal(responseBody, out); errDecode != nil {
		return fmt.Errorf("xai proxy pool: decode mihomo response: %w", errDecode)
	}
	return nil
}

func parsePublicIP(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var payload struct {
		IP string `json:"ip"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if ip := normalizePublicIP(payload.IP); ip != "" {
			return ip
		}
	}
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ip=") {
			if ip := normalizePublicIP(strings.TrimSpace(strings.TrimPrefix(line, "ip="))); ip != "" {
				return ip
			}
		}
	}
	return normalizePublicIP(trimmed)
}

func normalizePublicIP(raw string) string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || !ip.IsGlobalUnicast() {
		return ""
	}
	return ip.String()
}

func parsePositiveDuration(raw string, fallback time.Duration) time.Duration {
	parsed, errParse := time.ParseDuration(strings.TrimSpace(raw))
	if errParse != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
