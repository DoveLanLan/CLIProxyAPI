// Package xai implements thinking configuration for xAI Grok Responses API models.
//
// xAI models use the OpenAI Responses API compatible reasoning.effort format
// with discrete levels.
package xai

import (
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking/provider/codex"
)

// Applier implements thinking.ProviderApplier for xAI models.
type Applier struct {
	codex.Applier
}

var _ thinking.ProviderApplier = (*Applier)(nil)

// NewApplier creates a new xAI thinking applier.
func NewApplier() *Applier {
	return &Applier{}
}

// Apply applies thinking configuration to xAI requests.
// xAI's Responses API currently accepts none/low/medium/high efforts, so broader
// canonical levels are normalized before using the Codex-compatible applier.
func (a *Applier) Apply(body []byte, config thinking.ThinkingConfig, modelInfo *registry.ModelInfo) ([]byte, error) {
	config.Level = normalizeXAIEffort(config.Level)
	if config.Mode == thinking.ModeLevel && config.Level == "" {
		return body, nil
	}
	return a.Applier.Apply(body, config, modelInfo)
}

func normalizeXAIEffort(level thinking.ThinkingLevel) thinking.ThinkingLevel {
	switch level {
	case thinking.LevelMinimal:
		return thinking.LevelLow
	case thinking.LevelXHigh, thinking.LevelMax:
		return thinking.LevelHigh
	default:
		return level
	}
}

func init() {
	thinking.RegisterProvider("xai", NewApplier())
}
