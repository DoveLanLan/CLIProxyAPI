package management

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type managementXAIProxyPoolExecutor struct {
	status              helps.XAIProxyPoolStatus
	subscriptions       helps.XAIProxySubscriptionList
	lastAction          string
	lastSubscriptionURL string
	httpRequests        int
}

func (e *managementXAIProxyPoolExecutor) Identifier() string { return "xai" }

func (e *managementXAIProxyPoolExecutor) Execute(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *managementXAIProxyPoolExecutor) ExecuteStream(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return &cliproxyexecutor.StreamResult{}, nil
}

func (e *managementXAIProxyPoolExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *managementXAIProxyPoolExecutor) CountTokens(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *managementXAIProxyPoolExecutor) HttpRequest(_ context.Context, _ *coreauth.Auth, _ *http.Request) (*http.Response, error) {
	e.httpRequests++
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
	}, nil
}

func (e *managementXAIProxyPoolExecutor) XAIProxyPoolStatus() helps.XAIProxyPoolStatus {
	return e.status
}

func (e *managementXAIProxyPoolExecutor) RefreshXAIProxyProviders(context.Context) error {
	e.lastAction = "refresh"
	return nil
}

func (e *managementXAIProxyPoolExecutor) RotateXAIProxyLane(_ context.Context, lane string) error {
	e.lastAction = "rotate:" + lane
	return nil
}

func (e *managementXAIProxyPoolExecutor) CheckXAIProxyLane(_ context.Context, lane string) (bool, error) {
	e.lastAction = "check:" + lane
	return true, nil
}

func (e *managementXAIProxyPoolExecutor) QuarantineXAIProxyIP(_ context.Context, ip string) error {
	e.lastAction = "quarantine:" + ip
	return nil
}

func (e *managementXAIProxyPoolExecutor) UnquarantineXAIProxyIP(ip string) error {
	e.lastAction = "unquarantine:" + ip
	return nil
}

func (e *managementXAIProxyPoolExecutor) XAIProxySubscriptions(context.Context) helps.XAIProxySubscriptionList {
	return e.subscriptions
}

func (e *managementXAIProxyPoolExecutor) CreateXAIProxySubscription(_ context.Context, revision uint64, input helps.XAIProxySubscriptionCreate) (helps.XAIProxySubscriptionList, error) {
	if revision != e.subscriptions.Revision {
		return e.subscriptions, &helps.XAIProxySubscriptionError{Code: "revision_mismatch", Message: "subscription registry revision does not match", Status: http.StatusPreconditionFailed}
	}
	e.lastAction = "subscription-create:" + input.Name
	e.lastSubscriptionURL = input.URL
	e.subscriptions.Revision++
	e.subscriptions.Subscriptions = append(e.subscriptions.Subscriptions, helps.XAIProxySubscriptionStatus{
		Name: input.Name, Enabled: input.Enabled, Fingerprint: "redacted", State: "ready", NodeCount: 2,
	})
	return e.subscriptions, nil
}

func (e *managementXAIProxyPoolExecutor) UpdateXAIProxySubscription(_ context.Context, revision uint64, name string, input helps.XAIProxySubscriptionUpdate) (helps.XAIProxySubscriptionList, error) {
	if revision != e.subscriptions.Revision {
		return e.subscriptions, &helps.XAIProxySubscriptionError{Code: "revision_mismatch", Message: "subscription registry revision does not match", Status: http.StatusPreconditionFailed}
	}
	e.lastAction = "subscription-update:" + name
	if input.URL != nil {
		e.lastSubscriptionURL = *input.URL
	}
	e.subscriptions.Revision++
	return e.subscriptions, nil
}

func (e *managementXAIProxyPoolExecutor) DeleteXAIProxySubscription(_ context.Context, revision uint64, name string) (helps.XAIProxySubscriptionList, error) {
	if revision != e.subscriptions.Revision {
		return e.subscriptions, &helps.XAIProxySubscriptionError{Code: "revision_mismatch", Message: "subscription registry revision does not match", Status: http.StatusPreconditionFailed}
	}
	e.lastAction = "subscription-delete:" + name
	e.subscriptions.Revision++
	e.subscriptions.Subscriptions = nil
	return e.subscriptions, nil
}

func (e *managementXAIProxyPoolExecutor) CheckXAIProxySubscription(_ context.Context, name string) (helps.XAIProxySubscriptionStatus, error) {
	e.lastAction = "subscription-check:" + name
	return helps.XAIProxySubscriptionStatus{Name: name, Enabled: true, Fingerprint: "redacted", State: "ready", NodeCount: 2}, nil
}

func newManagementXAIProxyPoolHandler(t *testing.T, enabled bool) (*Handler, *managementXAIProxyPoolExecutor, *coreauth.Auth) {
	t.Helper()
	manager := coreauth.NewManager(nil, nil, nil)
	executor := &managementXAIProxyPoolExecutor{
		status:        helps.XAIProxyPoolStatus{Enabled: enabled, Ready: enabled},
		subscriptions: helps.XAIProxySubscriptionList{Enabled: enabled, Ready: enabled, Subscriptions: []helps.XAIProxySubscriptionStatus{}},
	}
	manager.RegisterExecutor(executor)
	auth := &coreauth.Auth{ID: "xai-auth", Provider: "xai", Metadata: map[string]any{"access_token": "token"}}
	if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("Register() error = %v", errRegister)
	}
	return &Handler{authManager: manager}, executor, auth
}

func TestGetXAIProxyPoolStatusDisabledReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newManagementXAIProxyPoolHandler(t, false)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/xai-proxy-pool/status", nil)
	h.GetXAIProxyPoolStatus(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRotateAndQuarantineXAIProxyPool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, executor, _ := newManagementXAIProxyPoolHandler(t, true)

	rotateRecorder := httptest.NewRecorder()
	rotateCtx, _ := gin.CreateTestContext(rotateRecorder)
	rotateCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/xai-proxy-pool/lanes/lane-1/rotate", nil)
	rotateCtx.Params = gin.Params{{Key: "lane", Value: "lane-1"}}
	h.RotateXAIProxyLane(rotateCtx)
	if rotateRecorder.Code != http.StatusOK || executor.lastAction != "rotate:lane-1" {
		t.Fatalf("rotate status/action = %d/%q", rotateRecorder.Code, executor.lastAction)
	}

	quarantineRecorder := httptest.NewRecorder()
	quarantineCtx, _ := gin.CreateTestContext(quarantineRecorder)
	quarantineCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/xai-proxy-pool/quarantine", bytes.NewBufferString(`{"ip":"203.0.113.8"}`))
	quarantineCtx.Request.Header.Set("Content-Type", "application/json")
	h.QuarantineXAIProxyIP(quarantineCtx)
	if quarantineRecorder.Code != http.StatusOK || executor.lastAction != "quarantine:203.0.113.8" {
		t.Fatalf("quarantine status/action = %d/%q", quarantineRecorder.Code, executor.lastAction)
	}
}

func TestExecuteAPIRequestUsesRegisteredXAIExecutor(t *testing.T) {
	h, executor, auth := newManagementXAIProxyPoolHandler(t, true)
	req := httptest.NewRequest(http.MethodPost, "https://cli-chat-proxy.grok.com/v1/responses", bytes.NewBufferString(`{}`))
	resp, errExecute := h.executeAPIRequest(context.Background(), auth, req)
	if errExecute != nil {
		t.Fatalf("executeAPIRequest() error = %v", errExecute)
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			t.Errorf("close response body: %v", errClose)
		}
	}()
	if executor.httpRequests != 1 {
		t.Fatalf("executor HTTP requests = %d", executor.httpRequests)
	}
	var payload map[string]bool
	if errDecode := json.NewDecoder(resp.Body).Decode(&payload); errDecode != nil || !payload["ok"] {
		t.Fatalf("response payload/error = %#v/%v", payload, errDecode)
	}
}

func TestXAIProxyPoolHTTPStatusUsesWrappedStatus(t *testing.T) {
	errWrapped := fmt.Errorf("wrapped: %w", &helps.XAIProxyPoolError{Message: "unavailable"})
	if got := xaiProxyPoolHTTPStatus(errWrapped, http.StatusBadGateway); got != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestCreateXAIProxySubscriptionRequiresRevisionAndRedactsURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, executor, _ := newManagementXAIProxyPoolHandler(t, true)
	secretURL := "https://subscription.example.com/source?token=write-only-secret"

	missingRevisionRecorder := httptest.NewRecorder()
	missingRevisionCtx, _ := gin.CreateTestContext(missingRevisionRecorder)
	missingRevisionCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/xai-proxy-pool/subscriptions", bytes.NewBufferString(`{"name":"provider-a","url":"`+secretURL+`"}`))
	missingRevisionCtx.Request.Header.Set("Content-Type", "application/json")
	h.CreateXAIProxySubscription(missingRevisionCtx)
	if missingRevisionRecorder.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing revision status = %d", missingRevisionRecorder.Code)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/xai-proxy-pool/subscriptions", bytes.NewBufferString(`{"name":"provider-a","url":"`+secretURL+`"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("If-Match", `"0"`)
	h.CreateXAIProxySubscription(ctx)
	if recorder.Code != http.StatusCreated || recorder.Header().Get("ETag") != `"1"` {
		t.Fatalf("create status/etag = %d/%q body=%s", recorder.Code, recorder.Header().Get("ETag"), recorder.Body.String())
	}
	if executor.lastSubscriptionURL != secretURL || executor.lastAction != "subscription-create:provider-a" {
		t.Fatalf("executor URL/action = %q/%q", executor.lastSubscriptionURL, executor.lastAction)
	}
	if strings.Contains(recorder.Body.String(), secretURL) || strings.Contains(recorder.Body.String(), "write-only-secret") {
		t.Fatalf("response leaked write-only URL: %s", recorder.Body.String())
	}
}

func TestXAIProxySubscriptionMutationRevisionMismatchIsRedacted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newManagementXAIProxyPoolHandler(t, true)
	secretURL := "https://subscription.example.com/source?token=never-return"
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/v0/management/xai-proxy-pool/subscriptions/provider-a", bytes.NewBufferString(`{"url":"`+secretURL+`"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("If-Match", `"9"`)
	ctx.Params = gin.Params{{Key: "name", Value: "provider-a"}}
	h.UpdateXAIProxySubscription(ctx)
	if recorder.Code != http.StatusPreconditionFailed || recorder.Header().Get("ETag") != `"0"` || !strings.Contains(recorder.Body.String(), "revision_mismatch") {
		t.Fatalf("mismatch status/etag/body = %d/%q/%s", recorder.Code, recorder.Header().Get("ETag"), recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "never-return") {
		t.Fatalf("error leaked write-only URL: %s", recorder.Body.String())
	}
}

func TestXAIProxySubscriptionMutationRejectsMalformedRevision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, executor, _ := newManagementXAIProxyPoolHandler(t, true)
	for _, revision := range []string{"0", `W/"0"`, `"0"junk`} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodDelete, "/v0/management/xai-proxy-pool/subscriptions/provider-a", nil)
		ctx.Request.Header.Set("If-Match", revision)
		ctx.Params = gin.Params{{Key: "name", Value: "provider-a"}}
		h.DeleteXAIProxySubscription(ctx)
		if recorder.Code != http.StatusBadRequest || executor.lastAction != "" {
			t.Fatalf("revision %q status/action = %d/%q", revision, recorder.Code, executor.lastAction)
		}
	}
}

func TestCheckAndDeleteXAIProxySubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, executor, _ := newManagementXAIProxyPoolHandler(t, true)
	executor.subscriptions.Revision = 3
	executor.subscriptions.Subscriptions = []helps.XAIProxySubscriptionStatus{{Name: "provider-a", Enabled: false, Fingerprint: "redacted", State: "disabled"}}

	checkRecorder := httptest.NewRecorder()
	checkCtx, _ := gin.CreateTestContext(checkRecorder)
	checkCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/xai-proxy-pool/subscriptions/provider-a/check", nil)
	checkCtx.Params = gin.Params{{Key: "name", Value: "provider-a"}}
	h.CheckXAIProxySubscription(checkCtx)
	if checkRecorder.Code != http.StatusOK || executor.lastAction != "subscription-check:provider-a" {
		t.Fatalf("check status/action = %d/%q", checkRecorder.Code, executor.lastAction)
	}

	deleteRecorder := httptest.NewRecorder()
	deleteCtx, _ := gin.CreateTestContext(deleteRecorder)
	deleteCtx.Request = httptest.NewRequest(http.MethodDelete, "/v0/management/xai-proxy-pool/subscriptions/provider-a", nil)
	deleteCtx.Request.Header.Set("If-Match", `"3"`)
	deleteCtx.Params = gin.Params{{Key: "name", Value: "provider-a"}}
	h.DeleteXAIProxySubscription(deleteCtx)
	if deleteRecorder.Code != http.StatusOK || deleteRecorder.Header().Get("ETag") != `"4"` || executor.lastAction != "subscription-delete:provider-a" {
		t.Fatalf("delete status/etag/action = %d/%q/%q", deleteRecorder.Code, deleteRecorder.Header().Get("ETag"), executor.lastAction)
	}
}

func TestGetXAIProxySubscriptionsDisabledReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newManagementXAIProxyPoolHandler(t, false)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/xai-proxy-pool/subscriptions", nil)
	h.GetXAIProxySubscriptions(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status/body = %d/%s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateXAIProxySubscriptionRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, executor, _ := newManagementXAIProxyPoolHandler(t, true)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body := `{"name":"provider-a","url":"https://subscription.example.com/` + strings.Repeat("x", xaiProxySubscriptionRequestMaxBytes) + `"}`
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/xai-proxy-pool/subscriptions", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("If-Match", `"0"`)
	h.CreateXAIProxySubscription(ctx)
	if recorder.Code != http.StatusBadRequest || executor.lastSubscriptionURL != "" {
		t.Fatalf("oversized status/url = %d/%q", recorder.Code, executor.lastSubscriptionURL)
	}
}
