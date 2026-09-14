package auth

import (
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

// resolveCommandCodeAPIKeyConfig compares opaque credentials case-sensitively.
func resolveCommandCodeAPIKeyConfig(cfg *config.Config, auth *Auth) *config.GeminiKey {
	if cfg == nil || auth == nil {
		return nil
	}
	key := strings.TrimSpace(auth.Attributes["api_key"])
	base := strings.TrimRight(strings.TrimSpace(auth.Attributes["base_url"]), "/")
	if key == "" {
		return nil
	}
	for i := range cfg.CommandCodeKey {
		entry := &cfg.CommandCodeKey[i]
		if strings.TrimSpace(entry.APIKey) == key && strings.TrimRight(strings.TrimSpace(entry.BaseURL), "/") == base {
			return entry
		}
	}
	return nil
}
