package helps

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

type commandCodeCatalogEntry struct {
	models  []string
	expires time.Time
	ready   chan struct{}
	err     error
}

// CommandCodeCatalog bounds discovery state and coalesces same-credential fetches.
type CommandCodeCatalog struct {
	mu      sync.Mutex
	entries map[[32]byte]*commandCodeCatalogEntry
}

func (c *CommandCodeCatalog) Models(ctx context.Context, client *http.Client, req *http.Request) ([]string, error) {
	id := commandCodeIdentity(req.URL.String(), req.Header.Get("Authorization"), req.Header)
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[[32]byte]*commandCodeCatalogEntry)
	}
	if entry := c.entries[id]; entry != nil && time.Now().Before(entry.expires) {
		c.mu.Unlock()
		select {
		case <-entry.ready:
			return append([]string(nil), entry.models...), entry.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if len(c.entries) >= CommandCodeMaxSessions {
		for key, entry := range c.entries {
			select {
			case <-entry.ready:
				delete(c.entries, key)
			default:
			}
			if len(c.entries) < CommandCodeMaxSessions {
				break
			}
		}
	}
	if len(c.entries) >= CommandCodeMaxSessions {
		c.mu.Unlock()
		return nil, CommandCodeError(503, "model discovery capacity reached")
	}
	entry := &commandCodeCatalogEntry{ready: make(chan struct{}), expires: time.Now().Add(5 * time.Minute)}
	c.entries[id] = entry
	c.mu.Unlock()
	models, err := commandCodeFetchModels(client, req)
	c.mu.Lock()
	entry.models, entry.err = models, err
	if err != nil {
		entry.expires = time.Now().Add(30 * time.Second)
	}
	close(entry.ready)
	c.mu.Unlock()
	return append([]string(nil), models...), err
}

func commandCodeFetchModels(client *http.Client, req *http.Request) ([]string, error) {
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer CloseCommandCodeBody(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, CommandCodeHTTPError(response.StatusCode, "model discovery rejected", response.Header)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 1<<20 {
		return nil, CommandCodeError(502, "model catalog exceeds size limit")
	}
	var catalog struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, CommandCodeError(502, "invalid model catalog")
	}
	models := make([]string, 0, len(catalog.Data))
	seen := make(map[string]bool)
	for _, model := range catalog.Data {
		if model.ID != "" && len(model.ID) <= 256 && !seen[model.ID] {
			models = append(models, model.ID)
			seen[model.ID] = true
		}
		if len(models) >= 2048 {
			break
		}
	}
	return models, nil
}
