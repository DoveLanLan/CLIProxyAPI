package helps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

const xaiResinAdminResponseLimit = 1 << 20

type xaiResinAdminClient struct {
	baseURL      *url.URL
	token        string
	platformName string
	httpClient   *http.Client

	mu           sync.RWMutex
	platformID   string
	generations  map[string]uint64
	resolveGroup singleflight.Group
	rotateGroup  singleflight.Group
}

type xaiResinPlatformPage struct {
	Items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"items"`
}

func newXAIResinAdminClient(cfg config.XAIResinProxyConfig, platform string) (*xaiResinAdminClient, error) {
	baseURL, errURL := parseXAIResinAdminURL(cfg.AdminURL)
	if errURL != nil {
		return nil, errURL
	}
	tokenBytes, errToken := os.ReadFile(strings.TrimSpace(cfg.AdminTokenFile))
	token := strings.TrimSpace(string(tokenBytes))
	if errToken != nil || token == "" {
		return nil, resinUnavailable("xai Resin admin token is unavailable")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &xaiResinAdminClient{
		baseURL:      baseURL,
		token:        token,
		platformName: platform,
		httpClient:   &http.Client{Transport: transport},
		generations:  make(map[string]uint64),
	}, nil
}

func parseXAIResinAdminURL(raw string) (*url.URL, error) {
	parsed, errParse := url.Parse(strings.TrimSpace(raw))
	if errParse != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, resinUnavailable("xai Resin admin URL is invalid")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, resinUnavailable("xai Resin admin URL uses an unsupported scheme")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = strings.TrimRight(parsed.RawPath, "/")
	return parsed, nil
}

func (c *xaiResinAdminClient) generation(account string) uint64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.generations[account]
}

func (c *xaiResinAdminClient) rotateLease(ctx context.Context, account string, observedGeneration uint64) error {
	if c == nil {
		return errors.New("Resin admin client is unavailable")
	}
	if c.generation(account) != observedGeneration {
		return nil
	}
	_, errRotate, _ := c.rotateGroup.Do(account, func() (any, error) {
		if c.generation(account) != observedGeneration {
			return nil, nil
		}
		if errDelete := c.deleteLease(ctx, account); errDelete != nil {
			return nil, errDelete
		}
		c.mu.Lock()
		if c.generations[account] == observedGeneration {
			c.generations[account] = observedGeneration + 1
		}
		c.mu.Unlock()
		return nil, nil
	})
	return errRotate
}

func (c *xaiResinAdminClient) deleteLease(ctx context.Context, account string) error {
	platformID, errPlatform := c.resolvePlatformID(ctx)
	if errPlatform != nil {
		return errPlatform
	}
	status, errDelete := c.deleteLeaseFromPlatform(ctx, platformID, account)
	if errDelete != nil || status != http.StatusNotFound {
		return errDelete
	}

	c.clearPlatformID(platformID)
	resolvedID, errResolve := c.resolvePlatformID(ctx)
	if errResolve != nil {
		return errResolve
	}
	if resolvedID == platformID {
		return nil
	}
	_, errRetry := c.deleteLeaseFromPlatform(ctx, resolvedID, account)
	return errRetry
}

func (c *xaiResinAdminClient) deleteLeaseFromPlatform(ctx context.Context, platformID string, account string) (int, error) {
	endpoint := c.endpoint("/api/v1/platforms/" + url.PathEscape(platformID) + "/leases/" + url.PathEscape(account))
	req, errRequest := http.NewRequestWithContext(nonNilContext(ctx), http.MethodDelete, endpoint, nil)
	if errRequest != nil {
		return 0, fmt.Errorf("create Resin lease deletion request: %w", errRequest)
	}
	c.authorize(req)
	resp, errDo := c.httpClient.Do(req)
	if errDo != nil {
		return 0, fmt.Errorf("delete Resin lease: %w", errDo)
	}
	defer closeXAIResinAdminResponse(resp)
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusNotFound:
		return resp.StatusCode, nil
	default:
		return resp.StatusCode, fmt.Errorf("delete Resin lease: unexpected HTTP status %d", resp.StatusCode)
	}
}

func (c *xaiResinAdminClient) resolvePlatformID(ctx context.Context) (string, error) {
	if cached := c.cachedPlatformID(); cached != "" {
		return cached, nil
	}
	resolved, errResolve, _ := c.resolveGroup.Do("platform", func() (any, error) {
		if cached := c.cachedPlatformID(); cached != "" {
			return cached, nil
		}
		endpointURL, errParse := url.Parse(c.endpoint("/api/v1/platforms"))
		if errParse != nil {
			return "", fmt.Errorf("build Resin platform request: %w", errParse)
		}
		query := endpointURL.Query()
		query.Set("keyword", c.platformName)
		query.Set("limit", "100000")
		endpointURL.RawQuery = query.Encode()
		req, errRequest := http.NewRequestWithContext(nonNilContext(ctx), http.MethodGet, endpointURL.String(), nil)
		if errRequest != nil {
			return "", fmt.Errorf("create Resin platform request: %w", errRequest)
		}
		c.authorize(req)
		resp, errDo := c.httpClient.Do(req)
		if errDo != nil {
			return "", fmt.Errorf("list Resin platforms: %w", errDo)
		}
		defer closeXAIResinAdminResponse(resp)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("list Resin platforms: unexpected HTTP status %d", resp.StatusCode)
		}
		var page xaiResinPlatformPage
		decoder := json.NewDecoder(io.LimitReader(resp.Body, xaiResinAdminResponseLimit))
		if errDecode := decoder.Decode(&page); errDecode != nil {
			return "", fmt.Errorf("decode Resin platforms: %w", errDecode)
		}
		for i := range page.Items {
			if strings.TrimSpace(page.Items[i].Name) == strings.TrimSpace(c.platformName) {
				platformID := strings.TrimSpace(page.Items[i].ID)
				if platformID == "" {
					break
				}
				c.mu.Lock()
				c.platformID = platformID
				c.mu.Unlock()
				return platformID, nil
			}
		}
		return "", errors.New("configured Resin platform was not found")
	})
	if errResolve != nil {
		return "", errResolve
	}
	platformID, ok := resolved.(string)
	if !ok || platformID == "" {
		return "", errors.New("configured Resin platform was not found")
	}
	return platformID, nil
}

func (c *xaiResinAdminClient) cachedPlatformID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.platformID
}

func (c *xaiResinAdminClient) clearPlatformID(platformID string) {
	c.mu.Lock()
	if c.platformID == platformID {
		c.platformID = ""
	}
	c.mu.Unlock()
}

func (c *xaiResinAdminClient) endpoint(suffix string) string {
	resolved := *c.baseURL
	resolved.Path = strings.TrimRight(resolved.Path, "/") + suffix
	resolved.RawPath = ""
	resolved.RawQuery = ""
	resolved.Fragment = ""
	return resolved.String()
}

func (c *xaiResinAdminClient) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
}

func closeXAIResinAdminResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	if _, errDrain := io.Copy(io.Discard, io.LimitReader(resp.Body, xaiResinAdminResponseLimit)); errDrain != nil {
		log.Debug("xai Resin admin: drain response body failed")
	}
	if errClose := resp.Body.Close(); errClose != nil {
		log.Debug("xai Resin admin: close response body failed")
	}
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}
