package claude

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

type delayedClaudeBootstrapErrorExecutor struct {
	delay time.Duration
}

func (e *delayedClaudeBootstrapErrorExecutor) Identifier() string { return "claude" }

func (e *delayedClaudeBootstrapErrorExecutor) Execute(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, errors.New("not implemented")
}

func (e *delayedClaudeBootstrapErrorExecutor) ExecuteStream(ctx context.Context, _ *coreauth.Auth, _ coreexecutor.Request, _ coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	chunks := make(chan coreexecutor.StreamChunk, 1)
	go func() {
		defer close(chunks)
		select {
		case <-ctx.Done():
			return
		case <-time.After(e.delay):
			chunks <- coreexecutor.StreamChunk{Err: &coreauth.Error{
				Code:       "bootstrap_failed",
				Message:    "upstream bootstrap failed",
				HTTPStatus: http.StatusBadGateway,
			}}
		}
	}()
	return &coreexecutor.StreamResult{Chunks: chunks}, nil
}

func (e *delayedClaudeBootstrapErrorExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *delayedClaudeBootstrapErrorExecutor) CountTokens(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, errors.New("not implemented")
}

func (e *delayedClaudeBootstrapErrorExecutor) HttpRequest(context.Context, *coreauth.Auth, *http.Request) (*http.Response, error) {
	return nil, errors.New("not implemented")
}

func TestClaudeStreamingBootstrapHeartbeatPrecedesDelayedStartupError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const (
		authID = "claude-bootstrap-heartbeat-auth"
		model  = "claude-bootstrap-heartbeat-model"
	)

	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(&delayedClaudeBootstrapErrorExecutor{delay: 1500 * time.Millisecond})
	if _, errRegister := manager.Register(context.Background(), &coreauth.Auth{
		ID:       authID,
		Provider: "claude",
		Status:   coreauth.StatusActive,
	}); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}
	registry.GetGlobalRegistry().RegisterClient(authID, "claude", []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(authID)
	})

	baseHandler := handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{
		Streaming: sdkconfig.StreamingConfig{KeepAliveSeconds: 1},
	}, manager)
	handler := NewClaudeCodeAPIHandler(baseHandler)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"`+model+`","stream":true,"max_tokens":1,"messages":[]}`))

	handler.ClaudeMessages(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	body := recorder.Body.String()
	if !strings.HasPrefix(body, ": keep-alive\n\n") {
		t.Fatalf("body does not start with bootstrap heartbeat: %q", body)
	}
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("body does not contain terminal SSE error: %q", body)
	}
	if !strings.Contains(body, "upstream bootstrap failed") {
		t.Fatalf("body does not contain Claude error payload: %q", body)
	}
}
