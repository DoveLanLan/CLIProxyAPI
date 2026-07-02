package cliproxy

import (
	"context"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking"
	_ "github.com/router-for-me/CLIProxyAPI/v7/internal/thinking/provider/openai"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
	"github.com/tidwall/gjson"
)

func TestOpenAICompatDefaultThinkingIncludesXHigh(t *testing.T) {
	support := resolveOpenAICompatThinking(config.OpenAICompatibilityModel{
		Name: "deepseek-ai/deepseek-v3.2",
	})

	if support == nil {
		t.Fatal("expected default thinking support")
	}
	if !support.ZeroAllowed {
		t.Fatal("expected default openai-compat thinking to allow zero/none")
	}
	for _, level := range []string{"none", "low", "medium", "high", "xhigh"} {
		if !thinking.HasLevel(support.Levels, level) {
			t.Fatalf("expected default openai-compat thinking level %q in %v", level, support.Levels)
		}
	}
}

func TestOpenAICompatExplicitThinkingRemainsAuthoritative(t *testing.T) {
	explicit := &registry.ThinkingSupport{
		Levels: []string{"low", "high"},
	}
	support := resolveOpenAICompatThinking(config.OpenAICompatibilityModel{
		Name:     "deepseek-ai/deepseek-v3.2",
		Thinking: explicit,
	})

	if support == nil {
		t.Fatal("expected explicit thinking support")
	}
	if thinking.HasLevel(support.Levels, "xhigh") {
		t.Fatalf("explicit thinking levels should not be widened, got %v", support.Levels)
	}

	explicit.Levels[0] = "mutated"
	if support.Levels[0] != "low" {
		t.Fatalf("explicit thinking support should be cloned, got %v", support.Levels)
	}
}

func TestOpenAICompatNamespacedDeepSeekPreservesXHighFromBudgetSuffix(t *testing.T) {
	const (
		clientID = "test-openai-compat-xhigh-client"
		provider = "bytevirt-test-openai-compat"
		model    = "deepseek-ai/deepseek-v3.2"
	)

	registry.GetGlobalRegistry().RegisterClient(clientID, provider, []*registry.ModelInfo{
		{
			ID:          model,
			Object:      "model",
			OwnedBy:     provider,
			Type:        "openai-compatibility",
			DisplayName: model,
			Thinking: resolveOpenAICompatThinking(config.OpenAICompatibilityModel{
				Name: model,
			}),
		},
	})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(clientID)
	})

	body := []byte(`{"model":"deepseek-ai/deepseek-v3.2(64000)","messages":[{"role":"user","content":"hi"}]}`)
	out, err := thinking.ApplyThinking(body, model+"(64000)", "openai", "openai", provider)
	if err != nil {
		t.Fatalf("ApplyThinking returned error: %v", err)
	}
	if got := gjson.GetBytes(out, "reasoning_effort").String(); got != "xhigh" {
		t.Fatalf("reasoning_effort = %q, want %q; body=%s", got, "xhigh", string(out))
	}
}

func TestOpenAICompatRegisterModelsForAuthPrefixedDeepSeekPreservesXHigh(t *testing.T) {
	const (
		clientID = "test-openai-compat-prefixed-xhigh-client"
		provider = "bytevirt"
		prefix   = "nih"
		baseID   = "deepseek-ai/deepseek-v3.2"
		modelID  = prefix + "/" + baseID
	)

	service := &Service{
		cfg: &config.Config{
			SDKConfig: config.SDKConfig{
				ForceModelPrefix: true,
			},
			OpenAICompatibility: []config.OpenAICompatibility{
				{
					Name:   provider,
					Prefix: prefix,
					Models: []config.OpenAICompatibilityModel{
						{Name: baseID},
					},
				},
			},
		},
	}
	auth := &coreauth.Auth{
		ID:       clientID,
		Provider: "openai-compatibility",
		Label:    provider,
		Prefix:   prefix,
	}

	service.registerModelsForAuth(context.Background(), auth)
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(clientID)
	})

	modelInfo := registry.LookupModelInfo(modelID, provider)
	if modelInfo == nil || modelInfo.Thinking == nil {
		t.Fatalf("expected prefixed model %q to be registered with thinking support", modelID)
	}
	if !thinking.HasLevel(modelInfo.Thinking.Levels, "xhigh") {
		t.Fatalf("expected prefixed model thinking levels to include xhigh, got %v", modelInfo.Thinking.Levels)
	}

	body := []byte(`{"model":"nih/deepseek-ai/deepseek-v3.2(64000)","messages":[{"role":"user","content":"hi"}]}`)
	out, err := thinking.ApplyThinking(body, modelID+"(64000)", "openai", "openai", provider)
	if err != nil {
		t.Fatalf("ApplyThinking returned error: %v", err)
	}
	if got := gjson.GetBytes(out, "reasoning_effort").String(); got != "xhigh" {
		t.Fatalf("reasoning_effort = %q, want %q; body=%s", got, "xhigh", string(out))
	}
}
