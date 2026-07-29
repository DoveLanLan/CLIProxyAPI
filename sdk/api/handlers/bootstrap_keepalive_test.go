package handlers

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type bootstrapKeepAliveRecorder struct {
	*httptest.ResponseRecorder
	wrote chan struct{}
}

func newBootstrapKeepAliveRecorder() *bootstrapKeepAliveRecorder {
	return &bootstrapKeepAliveRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		wrote:            make(chan struct{}, 1),
	}
}

func (r *bootstrapKeepAliveRecorder) Write(data []byte) (int, error) {
	n, errWrite := r.ResponseRecorder.Write(data)
	select {
	case r.wrote <- struct{}{}:
	default:
	}
	return n, errWrite
}

func TestStreamingBootstrapKeepAliveWritesAndFlushesHeartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := newBootstrapKeepAliveRecorder()
	c, _ := gin.CreateTestContext(recorder)

	stop := startStreamingBootstrapKeepAlive(c, context.Background(), 5*time.Millisecond, func() {
		c.Header("Content-Type", "text/event-stream")
	})

	select {
	case <-recorder.wrote:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bootstrap heartbeat")
	}
	wroteHeartbeat := stop()

	if !wroteHeartbeat {
		t.Fatal("stop reported that no heartbeat was written")
	}
	if got := recorder.Body.String(); got != streamingKeepAlivePayload {
		t.Fatalf("body = %q, want %q", got, streamingKeepAlivePayload)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if !recorder.Flushed {
		t.Fatal("heartbeat was not flushed")
	}
}

func TestStreamingBootstrapKeepAliveStopBeforeTickWritesNothing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := newBootstrapKeepAliveRecorder()
	c, _ := gin.CreateTestContext(recorder)
	headerCalls := 0

	stop := startStreamingBootstrapKeepAlive(c, context.Background(), time.Hour, func() {
		headerCalls++
	})
	wroteHeartbeat := stop()

	if wroteHeartbeat {
		t.Fatal("stop reported an unexpected heartbeat")
	}
	if c.Writer.Written() {
		t.Fatal("response was committed before the first heartbeat")
	}
	if got := recorder.Body.String(); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
	if headerCalls != 0 {
		t.Fatalf("setHeaders calls = %d, want 0", headerCalls)
	}
}

func TestStreamingBootstrapKeepAliveStopsOnContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := newBootstrapKeepAliveRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())

	stop := startStreamingBootstrapKeepAlive(c, ctx, time.Hour, nil)
	cancel()
	wroteHeartbeat := stop()

	if wroteHeartbeat || c.Writer.Written() {
		t.Fatal("canceled bootstrap keep-alive wrote a response")
	}
}
