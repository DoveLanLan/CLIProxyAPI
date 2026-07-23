package executor

import (
	"net/http"
	"testing"
	"time"
)

func TestXAIStatusErr_FreeUsageExhaustedSets24hRetryAfter(t *testing.T) {
	body := []byte(`{"code":"subscription:free-usage-exhausted","error":"You've used all the included free usage for model grok-4.5-build-free for now. Usage resets over a rolling 24-hour window — tokens (actual/limit): 1065387/1000000."}`)
	err := xaiStatusErr(http.StatusTooManyRequests, body)
	if err.StatusCode() != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", err.StatusCode())
	}
	if err.RetryAfter() == nil {
		t.Fatal("expected RetryAfter for free-usage-exhausted")
	}
	if *err.RetryAfter() != 24*time.Hour {
		t.Fatalf("RetryAfter = %v, want 24h", *err.RetryAfter())
	}
}

func TestXAIStatusErr_Generic429HasNoRetryAfter(t *testing.T) {
	body := []byte(`{"code":"rate_limit","error":"too many requests"}`)
	err := xaiStatusErr(http.StatusTooManyRequests, body)
	if err.RetryAfter() != nil {
		t.Fatalf("expected nil RetryAfter for generic 429, got %v", *err.RetryAfter())
	}
}

func TestXAIStatusErr_Non429Unchanged(t *testing.T) {
	body := []byte(`{"error":"nope"}`)
	err := xaiStatusErr(http.StatusBadRequest, body)
	if err.RetryAfter() != nil {
		t.Fatalf("expected nil RetryAfter for 400, got %v", *err.RetryAfter())
	}
}

func TestXAIStreamEventError_FreeUsageExhausted(t *testing.T) {
	payload := []byte(`{"code":"subscription:free-usage-exhausted","error":"You've used all the included free usage for now."}`)
	err, ok := xaiStreamEventError(payload)
	if !ok || err == nil {
		t.Fatal("xaiStreamEventError() did not classify free-usage exhaustion")
	}
	statusErr, okStatus := err.(interface{ StatusCode() int })
	if !okStatus || statusErr.StatusCode() != http.StatusTooManyRequests {
		t.Fatalf("status = %v, want 429", statusErr)
	}
	retryErr, okRetry := err.(interface{ RetryAfter() *time.Duration })
	if !okRetry || retryErr.RetryAfter() == nil || *retryErr.RetryAfter() != 24*time.Hour {
		t.Fatalf("RetryAfter = %#v, want 24h", retryErr)
	}
}

func TestXAIStreamEventError_NormalResponse(t *testing.T) {
	payload := []byte(`{"type":"response.completed","response":{"status":"completed","error":null}}`)
	if err, ok := xaiStreamEventError(payload); ok || err != nil {
		t.Fatalf("xaiStreamEventError() = (%v, %v), want no error", err, ok)
	}
}
