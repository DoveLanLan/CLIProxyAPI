package auth

import (
	"context"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func TestCommandCodeModelAliasReload(t *testing.T) {
	cfg := &config.Config{CommandCodeKey: []config.GeminiKey{{APIKey: "user_one", Models: []config.GeminiModel{{Name: "upstream-one", Alias: "friendly"}}}, {APIKey: "user_two", Models: []config.GeminiModel{{Name: "upstream-two", Alias: "friendly"}}}}}
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(cfg)
	for _, tc := range []struct{ key, want string }{{"user_one", "upstream-one"}, {"user_two", "upstream-two"}} {
		auth := &Auth{ID: tc.key, Provider: "commandcode", Attributes: map[string]string{"api_key": tc.key}}
		if _, err := manager.Register(context.Background(), auth); err != nil {
			t.Fatal(err)
		}
		if got := manager.applyAPIKeyModelAlias(auth, "friendly"); got != tc.want {
			t.Fatalf("alias=%s want %s", got, tc.want)
		}
		if got := manager.resolveAPIKeyModelAliasWithResult(auth, "friendly").UpstreamModel; got != tc.want {
			t.Fatalf("pool alias=%s want %s", got, tc.want)
		}
	}
	cfg.CommandCodeKey[0].Models[0].Name = "updated"
	manager.SetConfig(cfg)
	auth := &Auth{ID: "user_one", Provider: "commandcode", Attributes: map[string]string{"api_key": "user_one"}}
	if got := manager.applyAPIKeyModelAlias(auth, "friendly"); got != "updated" {
		t.Fatalf("reload alias=%s", got)
	}
}

func TestCommandCodeCredentialCaseIsolation(t *testing.T) {
	cfg := &config.Config{CommandCodeKey: []config.GeminiKey{{APIKey: "user_ABC", Models: []config.GeminiModel{{Name: "upper", Alias: "model"}}}, {APIKey: "user_abc", Models: []config.GeminiModel{{Name: "lower", Alias: "model"}}}}}
	entry := resolveCommandCodeAPIKeyConfig(cfg, &Auth{Attributes: map[string]string{"api_key": "user_abc"}})
	if entry == nil || entry.Models[0].Name != "lower" {
		t.Fatal("case-sensitive credentials were conflated")
	}
}
