package executor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestOpenAICompatExecutorDeepSeekClaudeForwardsImages(t *testing.T) {
	const imagePart = `{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aS1cAAAAASUVORK5CYII="}}`
	const imageURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aS1cAAAAASUVORK5CYII="
	for _, model := range []string{"deepseek-v4.1-flash", "deepseek/deepseek-v4.1-flash"} {
		for _, stream := range []bool{false, true} {
			for _, nested := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%t/tool-result=%t", model, stream, nested), func(t *testing.T) {
					var gotBody []byte
					var gotPath string
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						gotPath = r.URL.Path
						gotBody, _ = io.ReadAll(r.Body)
						for _, message := range gjson.GetBytes(gotBody, "messages").Array() {
							if message.Get("role").String() == "tool" {
								for _, part := range message.Get("content").Array() {
									if part.Get("type").String() == "image_url" {
										w.WriteHeader(http.StatusBadRequest)
										_, _ = io.WriteString(w, `{"error":{"message":"Invalid input"}}`)
										return
									}
								}
							}
						}
						if stream {
							w.Header().Set("Content-Type", "text/event-stream")
							_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"id":"chatcmpl_1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
					}))
					defer server.Close()
					messages := `[{"role":"user","content":[` + imagePart + `]}]`
					if nested {
						messages = `[{"role":"user","content":"read image"},{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":[` + imagePart + `]}]}]`
					}
					payload := []byte(fmt.Sprintf(`{"model":%q,"max_tokens":32,"stream":%t,"messages":%s}`, model, stream, messages))
					executor := NewOpenAICompatExecutor("commandcode", &config.Config{})
					auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL + "/v1", "api_key": "test"}}
					req := cliproxyexecutor.Request{Model: model, Payload: payload}
					opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("claude"), Stream: stream, OriginalRequest: payload}
					if stream {
						result, err := executor.ExecuteStream(context.Background(), auth, req, opts)
						if err != nil {
							t.Fatalf("ExecuteStream: %v", err)
						}
						for chunk := range result.Chunks {
							if chunk.Err != nil {
								t.Fatalf("stream: %v", chunk.Err)
							}
						}
					} else if _, err := executor.Execute(context.Background(), auth, req, opts); err != nil {
						t.Fatalf("Execute: %v", err)
					}
					if gotPath != "/v1/chat/completions" {
						t.Fatalf("upstream path = %q", gotPath)
					}
					if got := gjson.GetBytes(gotBody, "model").String(); got != model {
						t.Fatalf("model = %q, want %q", got, model)
					}
					images := 0
					for _, message := range gjson.GetBytes(gotBody, "messages").Array() {
						for _, part := range message.Get("content").Array() {
							if part.Get("type").String() == "image_url" {
								if message.Get("role").String() != "user" {
									t.Fatalf("image must be carried by user, got %s", message.Raw)
								}
								images++
								if got := part.Get("image_url.url").String(); got != imageURL {
									t.Fatalf("image URL = %q, want %q", got, imageURL)
								}
							}
						}
					}
					if images != 1 {
						t.Fatalf("translated images = %d, want 1; body=%s", images, gotBody)
					}
				})
			}
		}
	}
}
