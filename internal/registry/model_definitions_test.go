package registry

import "testing"

func TestOpenAIStaticModelsIncludeGPT55(t *testing.T) {
	model := findModelInfo(GetOpenAIModels(), "gpt-5.5")
	if model == nil {
		t.Fatal("expected GetOpenAIModels to include gpt-5.5")
	}
	assertGPT55ModelInfo(t, "GetOpenAIModels", model)

	lookup := LookupStaticModelInfo("gpt-5.5")
	if lookup == nil {
		t.Fatal("expected LookupStaticModelInfo to find gpt-5.5")
	}
	assertGPT55ModelInfo(t, "LookupStaticModelInfo", lookup)
}

func findModelInfo(models []*ModelInfo, id string) *ModelInfo {
	for _, model := range models {
		if model != nil && model.ID == id {
			return model
		}
	}
	return nil
}

func assertGPT55ModelInfo(t *testing.T, source string, model *ModelInfo) {
	t.Helper()

	if model.ID != "gpt-5.5" {
		t.Fatalf("%s id mismatch: got %q", source, model.ID)
	}
	if model.Object != "model" {
		t.Fatalf("%s object mismatch: got %q", source, model.Object)
	}
	if model.Created != 1776902400 {
		t.Fatalf("%s created mismatch: got %d", source, model.Created)
	}
	if model.OwnedBy != "openai" {
		t.Fatalf("%s owned_by mismatch: got %q", source, model.OwnedBy)
	}
	if model.Type != "openai" {
		t.Fatalf("%s type mismatch: got %q", source, model.Type)
	}
	if model.Version != "gpt-5.5" {
		t.Fatalf("%s version mismatch: got %q", source, model.Version)
	}
	if model.DisplayName != "GPT 5.5" {
		t.Fatalf("%s display name mismatch: got %q", source, model.DisplayName)
	}
	if model.Description != "Frontier model for complex coding, research, and real-world work." {
		t.Fatalf("%s description mismatch: got %q", source, model.Description)
	}
	if model.ContextLength != 272000 {
		t.Fatalf("%s context length mismatch: got %d", source, model.ContextLength)
	}
	if model.MaxCompletionTokens != 128000 {
		t.Fatalf("%s max completion tokens mismatch: got %d", source, model.MaxCompletionTokens)
	}
	if len(model.SupportedParameters) != 1 || model.SupportedParameters[0] != "tools" {
		t.Fatalf("%s supported parameters mismatch: got %v", source, model.SupportedParameters)
	}
	if model.Thinking == nil {
		t.Fatalf("%s thinking mismatch: expected non-nil", source)
	}

	wantLevels := []string{"low", "medium", "high", "xhigh"}
	if len(model.Thinking.Levels) != len(wantLevels) {
		t.Fatalf("%s thinking levels length mismatch: got %d", source, len(model.Thinking.Levels))
	}
	for idx, level := range wantLevels {
		if model.Thinking.Levels[idx] != level {
			t.Fatalf("%s thinking level %d mismatch: got %q want %q", source, idx, model.Thinking.Levels[idx], level)
		}
	}
}
