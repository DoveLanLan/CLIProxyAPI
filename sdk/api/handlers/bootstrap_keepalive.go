package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const streamingKeepAlivePayload = ": keep-alive\n\n"

// StartStreamingBootstrapKeepAlive emits SSE comments while synchronous stream setup is in progress.
// The returned stop function waits for the writer goroutine and reports whether a heartbeat was written.
func (h *BaseAPIHandler) StartStreamingBootstrapKeepAlive(c *gin.Context, ctx context.Context, setHeaders func()) func() bool {
	if h == nil {
		return func() bool { return false }
	}
	return startStreamingBootstrapKeepAlive(c, ctx, StreamingKeepAliveInterval(h.Cfg), setHeaders)
}

func startStreamingBootstrapKeepAlive(c *gin.Context, ctx context.Context, interval time.Duration, setHeaders func()) func() bool {
	if c == nil || interval <= 0 {
		return func() bool { return false }
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return func() bool { return false }
	}
	if ctx == nil {
		ctx = context.Background()
	}

	stopChan := make(chan struct{})
	var stopOnce sync.Once
	var wg sync.WaitGroup
	wroteHeartbeat := false
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if setHeaders != nil {
					setHeaders()
				}
				n, errWrite := c.Writer.Write([]byte(streamingKeepAlivePayload))
				if n > 0 {
					wroteHeartbeat = true
				}
				flusher.Flush()
				if errWrite != nil {
					return
				}
			}
		}
	}()

	return func() bool {
		stopOnce.Do(func() {
			close(stopChan)
		})
		wg.Wait()
		return wroteHeartbeat
	}
}
