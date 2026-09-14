package cliproxy

import (
	"context"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	log "github.com/sirupsen/logrus"
)

func (s *Service) commandCodeModels(ctx context.Context, auth *coreauth.Auth) []*ModelInfo {
	if s == nil || s.cfg == nil || auth == nil {
		return nil
	}
	for _, entry := range s.cfg.CommandCodeKey {
		if strings.TrimSpace(entry.APIKey) != strings.TrimSpace(auth.Attributes["api_key"]) || strings.TrimSpace(entry.BaseURL) != strings.TrimSpace(auth.Attributes["base_url"]) {
			continue
		}
		if len(entry.Models) > 0 {
			return buildConfigModels(entry.Models, "commandcode", "commandcode")
		}
		break
	}
	if registry.LocalModelsOnly() || s.coreManager == nil {
		return nil
	}
	bound, ok := s.coreManager.Executor("commandcode")
	if !ok {
		return nil
	}
	native, ok := bound.(*executor.CommandCodeExecutor)
	if !ok {
		return nil
	}
	ids, err := native.FetchModels(ctx, auth)
	if err != nil {
		log.WithField("provider", "commandcode").Warn("model discovery failed; configure explicit models for offline startup")
		return nil
	}
	models := make([]config.GeminiModel, 0, len(ids))
	for _, id := range ids {
		models = append(models, config.GeminiModel{Name: id})
	}
	return buildConfigModels(models, "commandcode", "commandcode")
}
