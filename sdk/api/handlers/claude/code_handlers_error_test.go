package claude

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
	basehandlers "github.com/router-for-me/CLIProxyAPI/v7/sdk/api/handlers"
	"github.com/tidwall/gjson"
)

func TestClaudeErrorExtractsOpenAIStyleUpstreamJSON(t *testing.T) {
	handler := NewClaudeCodeAPIHandler(&basehandlers.BaseAPIHandler{})
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"invalid_request_error","code":"context_too_large"}}`),
	}

	got := handler.toClaudeError(msg)

	if got.Type != "error" {
		t.Fatalf("type = %q, want error", got.Type)
	}
	if got.Error.Type != "invalid_request_error" {
		t.Fatalf("error.type = %q, want invalid_request_error", got.Error.Type)
	}
	if got.Error.Message != "Your input exceeds the context window of this model. Please adjust your input and try again." {
		t.Fatalf("error.message = %q", got.Error.Message)
	}
	if got.Error.Code != "context_too_large" {
		t.Fatalf("error.code = %q, want context_too_large", got.Error.Code)
	}
}

func TestClaudeErrorExtractsClaudeStyleUpstreamJSON(t *testing.T) {
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusTooManyRequests,
		Error:      errors.New(`{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your account's rate limit. Please try again later."},"request_id":"req_123"}`),
	}

	got := handler.toClaudeError(msg)

	if got.Error.Type != "rate_limit_error" {
		t.Fatalf("error.type = %q, want rate_limit_error", got.Error.Type)
	}
	if got.Error.Message != "This request would exceed your account's rate limit. Please try again later." {
		t.Fatalf("error.message = %q", got.Error.Message)
	}
	if got.RequestID != "req_123" {
		t.Fatalf("request_id = %q, want req_123", got.RequestID)
	}
}

func TestClaudeErrorExtractsRequestIDFromUpstreamHeader(t *testing.T) {
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadGateway,
		Error:      errors.New(`{"error":{"type":"api_error","message":"upstream failed"}}`),
		Addon:      http.Header{"X-Request-Id": {"req_header_123"}},
	}

	got := handler.toClaudeError(msg)

	if got.RequestID != "req_header_123" {
		t.Fatalf("request_id = %q, want req_header_123", got.RequestID)
	}
}

func TestClaudeErrorPreservesStructuredCompatibilityMetadata(t *testing.T) {
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"type":"invalid_request_error","code":"model_text_only","message":"text only","upstream_model":"deepseek-v4-pro"}}`),
	}

	got := handler.toClaudeError(msg)

	if got.Error.Code != "model_text_only" {
		t.Fatalf("error.code = %q, want model_text_only", got.Error.Code)
	}
	if got.Error.UpstreamModel != "deepseek-v4-pro" {
		t.Fatalf("error.upstream_model = %q, want deepseek-v4-pro", got.Error.UpstreamModel)
	}
}

func TestClaudeErrorRejectsUnsafeRequestID(t *testing.T) {
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadGateway,
		Error:      errors.New(`{"type":"error","error":{"type":"api_error","message":"upstream failed"},"request_id":"unsafe request id"}`),
	}

	got := handler.toClaudeError(msg)

	if got.RequestID != "" {
		t.Fatalf("request_id = %q, want empty", got.RequestID)
	}
}

func TestWriteClaudeErrorResponseUsesClaudeEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"invalid_request_error","code":"context_too_large"}}`),
	}

	handler.WriteErrorResponse(c, msg)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	body := recorder.Body.Bytes()
	if got := gjson.GetBytes(body, "type").String(); got != "error" {
		t.Fatalf("type = %q, want error; body=%s", got, body)
	}
	if got := gjson.GetBytes(body, "error.type").String(); got != "invalid_request_error" {
		t.Fatalf("error.type = %q, want invalid_request_error; body=%s", got, body)
	}
	if got := gjson.GetBytes(body, "error.message").String(); got != "Your input exceeds the context window of this model. Please adjust your input and try again." {
		t.Fatalf("error.message = %q; body=%s", got, body)
	}
}

func TestWriteClaudeErrorResponseIncludesSafeRequestIDWithoutHeaderPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	handler := NewClaudeCodeAPIHandler(&basehandlers.BaseAPIHandler{})
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadGateway,
		Error:      errors.New(`{"error":{"type":"api_error","message":"upstream failed"}}`),
		Addon:      http.Header{"X-Request-Id": {"req_header_123"}},
	}

	handler.WriteErrorResponse(c, msg)

	if got := gjson.GetBytes(recorder.Body.Bytes(), "request_id").String(); got != "req_header_123" {
		t.Fatalf("request_id = %q, want req_header_123; body=%s", got, recorder.Body.Bytes())
	}
	if got := recorder.Header().Get("X-Request-Id"); got != "" {
		t.Fatalf("X-Request-Id header = %q, want empty when passthrough is disabled", got)
	}
}

func TestPendingClaudeStreamErrorUsesBufferedError(t *testing.T) {
	wantErr := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"invalid_request_error","code":"context_too_large"}}`),
	}
	errs := make(chan *interfaces.ErrorMessage, 1)
	errs <- wantErr
	close(errs)

	gotErr, ok := pendingClaudeStreamError(errs)
	if !ok {
		t.Fatal("expected pending stream error")
	}
	if gotErr != wantErr {
		t.Fatalf("pending error = %p, want %p", gotErr, wantErr)
	}
}

func TestClaudeStartupErrorAfterBootstrapHeartbeatStaysSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	handler := &ClaudeCodeAPIHandler{}
	c.Header("Content-Type", "text/event-stream")
	_, _ = c.Writer.Write([]byte(": keep-alive\n\n"))
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadGateway,
		Error:      errors.New("upstream bootstrap failed"),
	}

	handler.writeClaudeStreamStartupError(c, recorder, msg, true)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("body does not contain SSE error event: %q", body)
	}
	if !strings.Contains(body, `"message":"upstream bootstrap failed"`) {
		t.Fatalf("body does not contain Claude error payload: %q", body)
	}
	if !recorder.Flushed {
		t.Fatal("startup SSE error was not flushed")
	}
}

func TestClaudeStartupErrorBeforeBootstrapHeartbeatPreservesHTTPStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadGateway,
		Error:      errors.New("upstream bootstrap failed"),
	}

	handler.writeClaudeStreamStartupError(c, recorder, msg, false)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if strings.Contains(recorder.Body.String(), "event: error") {
		t.Fatalf("body unexpectedly contains SSE error: %q", recorder.Body.String())
	}
}
