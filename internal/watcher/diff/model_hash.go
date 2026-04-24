package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

// ComputeOpenAICompatModelsHash returns a stable hash for OpenAI-compat models.
// Used to detect model list changes during hot reload.
func ComputeOpenAICompatModelsHash(models []config.OpenAICompatibilityModel) string {
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			key := openAICompatModelKey(model)
			if key == "" {
				continue
			}
			out(key)
		}
	})
	return hashJoined(keys)
}

// ComputeVertexCompatModelsHash returns a stable hash for Vertex-compatible models.
func ComputeVertexCompatModelsHash(models []config.VertexCompatModel) string {
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return hashJoined(keys)
}

// ComputeClaudeModelsHash returns a stable hash for Claude model aliases.
func ComputeClaudeModelsHash(models []config.ClaudeModel) string {
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return hashJoined(keys)
}

// ComputeCodexModelsHash returns a stable hash for Codex model aliases.
func ComputeCodexModelsHash(models []config.CodexModel) string {
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return hashJoined(keys)
}

// ComputeGeminiModelsHash returns a stable hash for Gemini model aliases.
func ComputeGeminiModelsHash(models []config.GeminiModel) string {
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return hashJoined(keys)
}

// ComputeExcludedModelsHash returns a normalized hash for excluded model lists.
func ComputeExcludedModelsHash(excluded []string) string {
	if len(excluded) == 0 {
		return ""
	}
	normalized := make([]string, 0, len(excluded))
	for _, entry := range excluded {
		if trimmed := strings.TrimSpace(entry); trimmed != "" {
			normalized = append(normalized, strings.ToLower(trimmed))
		}
	}
	if len(normalized) == 0 {
		return ""
	}
	sort.Strings(normalized)
	data, _ := json.Marshal(normalized)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func normalizeModelPairs(collect func(out func(key string))) []string {
	seen := make(map[string]struct{})
	keys := make([]string, 0)
	collect(func(key string) {
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	})
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	return keys
}

func hashJoined(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return hex.EncodeToString(sum[:])
}

type openAICompatThinkingKey struct {
	Min            int      `json:"min,omitempty"`
	Max            int      `json:"max,omitempty"`
	ZeroAllowed    bool     `json:"zero_allowed,omitempty"`
	DynamicAllowed bool     `json:"dynamic_allowed,omitempty"`
	Levels         []string `json:"levels,omitempty"`
}

type openAICompatModelKeyPayload struct {
	Name     string                   `json:"name,omitempty"`
	Alias    string                   `json:"alias,omitempty"`
	Thinking *openAICompatThinkingKey `json:"thinking,omitempty"`
}

func openAICompatModelKey(model config.OpenAICompatibilityModel) string {
	name := strings.ToLower(strings.TrimSpace(model.Name))
	alias := strings.ToLower(strings.TrimSpace(model.Alias))
	if name == "" && alias == "" {
		return ""
	}

	payload := openAICompatModelKeyPayload{
		Name:  name,
		Alias: alias,
	}
	if thinking := model.Thinking; thinking != nil {
		normalized := &openAICompatThinkingKey{
			Min:            thinking.Min,
			Max:            thinking.Max,
			ZeroAllowed:    thinking.ZeroAllowed,
			DynamicAllowed: thinking.DynamicAllowed,
		}
		if len(thinking.Levels) > 0 {
			levels := make([]string, 0, len(thinking.Levels))
			for _, level := range thinking.Levels {
				if trimmed := strings.ToLower(strings.TrimSpace(level)); trimmed != "" {
					levels = append(levels, trimmed)
				}
			}
			if len(levels) > 0 {
				normalized.Levels = levels
			}
		}
		payload.Thinking = normalized
	}

	data, _ := json.Marshal(payload)
	return string(data)
}
