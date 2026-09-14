package cliproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestCommandCodeRegistrationAndLocalModels(t *testing.T) {
	cfg := &config.Config{CommandCodeKey: []config.GeminiKey{{APIKey: "user_one", Models: []config.GeminiModel{{Name: "upstream", Alias: "friendly"}}}}}
	s := &Service{cfg: cfg, coreManager: coreauth.NewManager(nil, nil, nil)}
	auth := &coreauth.Auth{ID: "cc-registration-test", Provider: "commandcode", Attributes: map[string]string{"api_key": "user_one"}}
	s.ensureExecutorsForAuth(auth)
	first, _ := s.coreManager.Executor("commandcode")
	if _, ok := first.(*executor.CommandCodeExecutor); !ok {
		t.Fatalf("wrong executor %T", first)
	}
	s.ensureExecutorsForAuth(auth)
	second, _ := s.coreManager.Executor("commandcode")
	if first != second {
		t.Fatal("binding discarded session cache")
	}
	models := s.commandCodeModels(context.Background(), auth)
	if len(models) != 1 || models[0].ID != "friendly" {
		t.Fatalf("models=%+v", models)
	}
	s.registerModelsForAuth(context.Background(), auth)
	t.Cleanup(func() { registry.GetGlobalRegistry().UnregisterClient(auth.ID) })
	if providers := registry.GetGlobalRegistry().GetModelProviders("friendly"); len(providers) != 1 || providers[0] != "commandcode" {
		t.Fatalf("providers=%v", providers)
	}
	previous := registry.LocalModelsOnly()
	registry.SetLocalModelsOnly(true)
	defer registry.SetLocalModelsOnly(previous)
	cfg.CommandCodeKey[0].Models = nil
	if got := s.commandCodeModels(context.Background(), auth); len(got) != 0 {
		t.Fatalf("local mode invented models: %v", got)
	}
}

func TestCommandCodeNativeSchedulingAndAlias(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/alpha/generate" {
			w.WriteHeader(200)
			return
		}
		key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		body, _ := io.ReadAll(r.Body)
		if gjson.GetBytes(body, "params.model").String() != key+"-model" {
			t.Errorf("wrong model for selected key: %s", body)
		}
		mu.Lock()
		counts[key]++
		mu.Unlock()
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"ok\"}\n{\"type\":\"finish\",\"finishReason\":\"stop\"}\n")
	}))
	defer server.Close()
	cfg := &config.Config{}
	for _, key := range []string{"user_one", "user_two"} {
		cfg.CommandCodeKey = append(cfg.CommandCodeKey, config.GeminiKey{APIKey: key, BaseURL: server.URL, Prefix: "cc", Models: []config.GeminiModel{{Name: key + "-model", Alias: "cc-round-robin-test"}}})
	}
	s := &Service{cfg: cfg, coreManager: coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)}
	s.coreManager.SetConfig(cfg)
	for _, key := range []string{"user_one", "user_two"} {
		auth := &coreauth.Auth{ID: "cc-schedule-" + key, Provider: "commandcode", Prefix: "cc", Status: coreauth.StatusActive, Attributes: map[string]string{"api_key": key, "base_url": server.URL}}
		s.ensureExecutorsForAuth(auth)
		if _, err := s.coreManager.Register(context.Background(), auth); err != nil {
			t.Fatal(err)
		}
		s.registerModelsForAuth(context.Background(), auth)
		t.Cleanup(func() { registry.GetGlobalRegistry().UnregisterClient(auth.ID) })
	}
	for i := 0; i < 4; i++ {
		_, err := s.coreManager.Execute(context.Background(), []string{"commandcode"}, cliproxyexecutor.Request{Model: "cc/cc-round-robin-test", Payload: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai")})
		if err != nil {
			t.Fatal(err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if counts["user_one"] != 2 || counts["user_two"] != 2 {
		t.Fatalf("unexpected scheduling %v", counts)
	}
}
