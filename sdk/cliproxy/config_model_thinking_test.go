package cliproxy

import (
	"reflect"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
)

func TestBuildConfigModelsThinkingOverride(t *testing.T) {
	// glm levels including "max", which is absent from the static registry
	// (the model is too new for the bundled definitions).
	glmLevels := []string{"none", "low", "medium", "high", "xhigh", "max"}

	models := buildClaudeConfigModels(&config.ClaudeKey{Models: []config.ClaudeModel{
		{Name: "Pro/zai-org/GLM-5", Thinking: &registry.ThinkingSupport{Levels: glmLevels}},
		// No explicit thinking: keeps falling back to static registry lookup.
		{Name: "claude-opus-4-6"},
	}})

	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}

	glm := models[0]
	if glm.Thinking == nil {
		t.Fatal("GLM model Thinking is nil, want explicit override")
	}
	if !reflect.DeepEqual(glm.Thinking.Levels, glmLevels) {
		t.Errorf("GLM Levels = %v, want %v", glm.Thinking.Levels, glmLevels)
	}
	hasMax := false
	for _, l := range glm.Thinking.Levels {
		if l == "max" {
			hasMax = true
			break
		}
	}
	if !hasMax {
		t.Errorf("GLM Levels %v missing \"max\"", glm.Thinking.Levels)
	}

	opus := models[1]
	if opus.Thinking == nil {
		t.Fatal("known Claude model Thinking is nil, want static registry fallback")
	}
	static := registry.LookupStaticModelInfo("claude-opus-4-6")
	if static == nil || static.Thinking == nil {
		t.Skip("claude-opus-4-6 not in static registry")
	}
	if !reflect.DeepEqual(opus.Thinking.Levels, static.Thinking.Levels) {
		t.Errorf("opus Levels = %v, want static %v", opus.Thinking.Levels, static.Thinking.Levels)
	}
}
