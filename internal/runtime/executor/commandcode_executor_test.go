package executor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	_ "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator/builtin"
	"github.com/tidwall/gjson"
)

func TestCommandCodeExecutorProtocols(t *testing.T) {
	for _, protocol := range []string{"openai", "claude", "openai-response"} {
		for _, streaming := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", protocol, streaming), func(t *testing.T) {
				var initCalls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer user_test" {
						t.Error("wrong credential")
					}
					if r.Header.Get("x-command-code-version") != helps.CommandCodeVersion {
						t.Error("wrong protocol version")
					}
					if r.URL.Path != "/alpha/generate" {
						initCalls.Add(1)
						w.WriteHeader(200)
						return
					}
					body, _ := io.ReadAll(r.Body)
					if !gjson.GetBytes(body, "params.stream").Bool() || gjson.GetBytes(body, "params.model").String() != "cc-test" {
						t.Errorf("wrong envelope %s", body)
					}
					w.Header().Set("Content-Type", "application/x-ndjson")
					fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"hello\"}\n")
					fmt.Fprint(w, "{\"type\":\"finish\",\"finishReason\":\"stop\",\"totalUsage\":{\"inputTokens\":20,\"outputTokens\":5,\"cachedInputTokens\":10}}\n")
				}))
				defer server.Close()
				e := NewCommandCodeExecutor(&config.Config{})
				auth := &cliproxyauth.Auth{ID: "test", Provider: "commandcode", Attributes: map[string]string{"api_key": "user_test", "base_url": server.URL}}
				payload := []byte(`{"model":"cc-test","messages":[{"role":"user","content":"hi"}],"max_tokens":128}`)
				if protocol == "openai-response" {
					payload = []byte(`{"model":"cc-test","input":"hi","max_output_tokens":128}`)
				}
				req := cliproxyexecutor.Request{Model: "cc-test", Payload: payload}
				opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString(protocol), OriginalRequest: payload, Stream: streaming}
				for i := 0; i < 2; i++ {
					if streaming {
						response, err := e.ExecuteStream(context.Background(), auth, req, opts)
						if err != nil {
							t.Fatal(err)
						}
						var out strings.Builder
						for chunk := range response.Chunks {
							if chunk.Err != nil {
								t.Fatal(chunk.Err)
							}
							out.Write(chunk.Payload)
						}
						if !strings.Contains(out.String(), "hello") {
							t.Fatalf("missing response: %s", out.String())
						}
						if protocol == "claude" && !strings.Contains(out.String(), "message_stop") {
							t.Fatalf("missing terminal event: %s", out.String())
						}
						if protocol == "openai-response" && !strings.Contains(out.String(), "response.completed") {
							t.Fatalf("missing terminal event: %s", out.String())
						}
					} else {
						response, err := e.Execute(context.Background(), auth, req, opts)
						if err != nil {
							t.Fatal(err)
						}
						if !strings.Contains(string(response.Payload), "hello") {
							t.Fatalf("missing response %s", response.Payload)
						}
						if protocol == "openai" && gjson.GetBytes(response.Payload, "usage.prompt_tokens_details.cached_tokens").Int() != 10 {
							t.Fatalf("bad usage %s", response.Payload)
						}
					}
				}
				if initCalls.Load() != 2 {
					t.Fatalf("initialization repeated %d", initCalls.Load())
				}
			})
		}
	}
}

func TestCommandCodeExecutorClaudeToolReasoning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/alpha/generate" {
			w.WriteHeader(200)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if gjson.GetBytes(body, "params.messages.0.content.0.type").String() != "reasoning" || gjson.GetBytes(body, "params.messages.0.content.0.text").String() != "exact thought" {
			t.Errorf("lost source reasoning %s", body)
		}
		if gjson.GetBytes(body, "params.messages.1.content.0.toolName").String() != "lookup" {
			t.Errorf("lost tool link %s", body)
		}
		fmt.Fprint(w, "data: {\"type\":\"text-delta\",\"text\":\"done\"}\n\ndata: {\"type\":\"finish\",\"finishReason\":\"stop\"}\n\n")
	}))
	defer server.Close()
	e := NewCommandCodeExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"api_key": "user_test", "base_url": server.URL}}
	payload := []byte(`{"model":"cc-test","messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"exact thought"},{"type":"tool_use","id":"call_1","name":"lookup","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"result"}]}],"max_tokens":128}`)
	_, err := e.Execute(context.Background(), auth, cliproxyexecutor.Request{Model: "cc-test", Payload: payload}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("claude"), OriginalRequest: payload})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommandCodeExecutorErrorsAndCancel(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		for _, mode := range []string{"truncated", "error", "cancel", "status"} {
			t.Run(fmt.Sprintf("%s/%v", mode, streaming), func(t *testing.T) {
				entered := make(chan struct{})
				canceled := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/alpha/generate" {
						w.WriteHeader(200)
						return
					}
					if mode == "status" {
						w.WriteHeader(429)
						return
					}
					w.Header().Set("Content-Type", "text/event-stream")
					w.WriteHeader(200)
					w.(http.Flusher).Flush()
					if mode == "cancel" {
						close(entered)
						<-r.Context().Done()
						close(canceled)
						return
					}
					if mode == "error" {
						fmt.Fprint(w, "data: {\"type\":\"error\",\"message\":\"<429> key=user_test\"}\n\n")
						return
					}
					fmt.Fprint(w, "data: {\"type\":\"text-delta\",\"text\":\"partial\"}\n\n")
				}))
				defer server.Close()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				e := NewCommandCodeExecutor(&config.Config{})
				auth := &cliproxyauth.Auth{Attributes: map[string]string{"api_key": "user_test", "base_url": server.URL}}
				req := cliproxyexecutor.Request{Model: "test", Payload: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)}
				opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai")}
				done := make(chan error, 1)
				go func() {
					if !streaming {
						_, err := e.Execute(ctx, auth, req, opts)
						done <- err
						return
					}
					response, err := e.ExecuteStream(ctx, auth, req, opts)
					if err != nil {
						done <- err
						return
					}
					for chunk := range response.Chunks {
						if chunk.Err != nil {
							done <- chunk.Err
							return
						}
					}
					done <- ctx.Err()
				}()
				if mode == "cancel" {
					select {
					case <-entered:
						cancel()
					case <-time.After(3 * time.Second):
						t.Fatal("upstream never entered")
					}
				}
				select {
				case err := <-done:
					if err == nil {
						t.Fatal("expected failure")
					}
					if strings.Contains(err.Error(), "user_test") {
						t.Fatal("key leaked")
					}
				case <-time.After(3 * time.Second):
					t.Fatal("execution hung")
				}
				if mode == "cancel" {
					select {
					case <-canceled:
					case <-time.After(3 * time.Second):
						t.Fatal("upstream not canceled")
					}
				}
			})
		}
	}
}

func TestCommandCodeRejectsMissingKeyAndCompact(t *testing.T) {
	e := NewCommandCodeExecutor(&config.Config{})
	req, _ := http.NewRequest(http.MethodGet, "http://unused", nil)
	if err := e.PrepareRequest(req, nil); err == nil {
		t.Fatal("accepted missing key")
	}
	_, err := e.Execute(context.Background(), nil, cliproxyexecutor.Request{}, cliproxyexecutor.Options{Alt: "responses/compact"})
	if err == nil {
		t.Fatal("accepted compact")
	}
	for _, payload := range []string{`{`, `{"input":"hi","previous_response_id":"resp_old"}`, `{"input":"hi","tools":[{"type":"web_search"}]}`} {
		_, err := e.Execute(context.Background(), nil, cliproxyexecutor.Request{Payload: []byte(payload)}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response")})
		if err == nil {
			t.Fatalf("accepted unsupported input %s", payload)
		}
		if status, ok := err.(interface{ StatusCode() int }); !ok || status.StatusCode() != 400 {
			t.Fatalf("expected validation before credential/network access: %v", err)
		}
	}
}

type commandCodeUsageCapture struct{ records chan usage.Record }

func (c *commandCodeUsageCapture) HandleUsage(_ context.Context, record usage.Record) {
	if record.AuthID != "cc-usage-test" {
		return
	}
	select {
	case c.records <- record:
	default:
	}
}

func TestCommandCodeToolProtocolsAndUsage(t *testing.T) {
	plugin := &commandCodeUsageCapture{records: make(chan usage.Record, 16)}
	usage.RegisterPlugin(plugin)
	for _, protocol := range []string{"openai", "claude", "openai-response"} {
		for _, streaming := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", protocol, streaming), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/alpha/generate" {
						w.WriteHeader(200)
						return
					}
					fmt.Fprint(w, "{\"type\":\"reasoning-delta\",\"text\":\"thinking\"}\n")
					fmt.Fprint(w, "{\"type\":\"tool-input-start\",\"id\":\"call_tool\",\"toolName\":\"lookup\"}\n")
					fmt.Fprint(w, "{\"type\":\"tool-input-delta\",\"id\":\"call_tool\",\"inputTextDelta\":\"{}\"}\n")
					fmt.Fprint(w, "{\"type\":\"tool-call\",\"toolCallId\":\"call_tool\",\"toolName\":\"lookup\",\"input\":{}}\n")
					fmt.Fprint(w, "{\"type\":\"finish\",\"finishReason\":\"tool-calls\",\"totalUsage\":{\"inputTokens\":100,\"outputTokens\":10,\"cachedInputTokens\":50,\"inputTokenDetails\":{\"cacheWriteTokens\":5}}}\n")
				}))
				defer server.Close()
				e := NewCommandCodeExecutor(&config.Config{})
				auth := &cliproxyauth.Auth{ID: "cc-usage-test", Provider: "commandcode", Attributes: map[string]string{"api_key": "user_test", "base_url": server.URL}}
				payload := []byte(`{"messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}]}`)
				if protocol == "claude" {
					payload = []byte(`{"messages":[{"role":"user","content":"hi"}],"max_tokens":128,"tools":[{"name":"lookup","input_schema":{"type":"object"}}]}`)
				}
				if protocol == "openai-response" {
					payload = []byte(`{"input":"hi","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}`)
				}
				req := cliproxyexecutor.Request{Model: "test", Payload: payload}
				opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString(protocol), OriginalRequest: payload}
				var output strings.Builder
				if streaming {
					response, err := e.ExecuteStream(context.Background(), auth, req, opts)
					if err != nil {
						t.Fatal(err)
					}
					for chunk := range response.Chunks {
						if chunk.Err != nil {
							t.Fatal(chunk.Err)
						}
						output.Write(chunk.Payload)
					}
				} else {
					response, err := e.Execute(context.Background(), auth, req, opts)
					if err != nil {
						t.Fatal(err)
					}
					output.Write(response.Payload)
				}
				for _, want := range []string{"lookup", "call_tool", "thinking"} {
					if !strings.Contains(output.String(), want) {
						t.Fatalf("missing %s: %s", want, output.String())
					}
				}
				if strings.Contains(output.String(), `"signature"`) {
					t.Fatalf("fabricated signature: %s", output.String())
				}
				select {
				case record := <-plugin.records:
					if record.Provider != "commandcode" || record.ExecutorType != "CommandCodeExecutor" || record.Failed || record.Detail.InputTokens != 100 || record.Detail.OutputTokens != 10 || record.Detail.CacheReadTokens != 50 || record.Detail.CacheCreationTokens != 5 {
						t.Fatalf("bad usage %+v", record)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("no usage published")
				}
			})
		}
	}
}
