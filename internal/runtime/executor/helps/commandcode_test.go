package helps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/tidwall/gjson"
)

func TestCommandCodeWire(t *testing.T) {
	body := []byte(`{"model":"test","messages":[{"role":"developer","content":"system"},{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,YQ=="}}]},{"role":"assistant","reasoning_content":"think","content":"call","tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{\"q\":1}"}}]},{"role":"tool","tool_call_id":"call_1","content":"result"}],"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],"tool_choice":"required","prompt_cache_key":"cache-key","reasoning_effort":"high"}`)
	wire, err := CommandCodeRequest(body, "8dca1a5e-761c-43ae-bad3-d38d1610e7fd")
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"params.model": "test", "params.stream": "true", "params.system.0.text": "system", "params.system.0.cache_control.type": "ephemeral",
		"params.messages.0.content.0.type": "image", "params.messages.0.content.0.mimeType": "image/png", "params.messages.1.content.0.type": "reasoning",
		"params.messages.1.content.2.input.q": "1", "params.messages.2.content.0.toolName": "lookup", "params.messages.2.content.0.output.value": "result",
		"params.tools.0.input_schema.type": "object", "params.tool_choice.type": "any", "params.reasoning_effort": "high", "threadId": "8dca1a5e-761c-43ae-bad3-d38d1610e7fd",
	} {
		if got := gjson.GetBytes(wire, path).String(); got != want {
			t.Errorf("%s=%q, want %q", path, got, want)
		}
	}
	for _, bad := range []string{`{`, `{"messages":{}}`, `{"messages":[{"role":"tool","tool_call_id":"missing"}]}`, `{"messages":[{"role":"assistant","tool_calls":[{"function":{"arguments":"{"}}]}]}`, `{"messages":[],"tools":[{"type":"web_search"}]}`, `{"messages":[],"max_tokens":-1}`} {
		if _, err := CommandCodeRequest([]byte(bad), ""); err == nil {
			t.Errorf("accepted invalid input %s", bad)
		}
	}
}

func TestCommandCodeSSEFragmentedToolsUsage(t *testing.T) {
	events := []string{
		`{"type":"reasoning-delta","text":"thinking"}`,
		`{"type":"text-delta","text":"hello"}`,
		`{"type":"tool-input-start","id":"a","toolName":"lookup"}`,
		`{"type":"tool-input-delta","id":"a","inputTextDelta":"{\"q\":"}`,
		`{"type":"tool-input-delta","id":"a","inputTextDelta":"1}"}`,
		`{"type":"tool-call","toolCallId":"a","toolName":"lookup","input":{"q":1}}`,
		`{"type":"finish","finishReason":"tool-calls","totalUsage":{"inputTokens":20,"outputTokens":5,"inputTokenDetails":{"cacheReadTokens":10,"cacheWriteTokens":2}}}`,
	}
	data := ": keepalive\r\n\r\n"
	for _, event := range events {
		data += "event: message\r\ndata: " + event + "\r\n\r\n"
	}
	decoder := NewCommandCodeDecoder(iotest.OneByteReader(strings.NewReader(data)))
	stream := NewCommandCodeStream("test")
	var aggregate CommandCodeAggregate
	for !stream.Finished {
		event, err := decoder.Next()
		if err != nil {
			t.Fatal(err)
		}
		lines, err := stream.Convert(event)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range lines {
			if err := aggregate.Add(line); err != nil {
				t.Fatal(err)
			}
		}
	}
	response, err := aggregate.Response()
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{"choices.0.message.content": "hello", "choices.0.message.reasoning_content": "thinking", "choices.0.message.tool_calls.0.function.arguments": `{"q":1}`, "choices.0.finish_reason": "tool_calls", "usage.prompt_tokens": "20", "usage.prompt_tokens_details.cached_tokens": "10", "usage.prompt_tokens_details.cache_creation_tokens": "2"} {
		if got := gjson.GetBytes(response, path).String(); got != want {
			t.Errorf("%s=%q, want %q", path, got, want)
		}
	}
	if _, err := decoder.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestCommandCodeStreamErrorsAndLimits(t *testing.T) {
	for _, event := range []string{`{`, `[DONE]`, `{"type":"finish"}`, `{"type":"error","message":"<429> limit"}`, `{"type":"tool-call","toolName":"missing-id"}`} {
		if _, err := NewCommandCodeStream("test").Convert([]byte(event)); err == nil {
			t.Errorf("accepted %s", event)
		}
	}
	decoder := NewCommandCodeDecoder(strings.NewReader("data: " + strings.Repeat("x", CommandCodeMaxEventBytes) + "\n\n"))
	if _, err := decoder.Next(); err == nil {
		t.Fatal("accepted oversize SSE event")
	}
	aggregate := CommandCodeAggregate{bytes: CommandCodeMaxResponseBytes}
	if err := aggregate.Add([]byte("data: {}")); err == nil {
		t.Fatal("accepted oversize aggregation")
	}
	stream := NewCommandCodeStream("test")
	_, _ = stream.Convert([]byte(`{"type":"text-delta","text":"hello"}`))
	_, err := stream.Convert([]byte(`{"type":"finish","usage":{"inputTokens":100,"outputTokens":0}}`))
	if err != nil || stream.Usage["prompt_tokens"] != int64(100) {
		t.Fatalf("usage was discarded: %+v %v", stream.Usage, err)
	}
}

func TestCommandCodeNativeNDJSON(t *testing.T) {
	input := ": heartbeat\n{\"type\":\"text-delta\",\"text\":\"你好\"}\n{\"type\":\"finish\",\"finishReason\":\"stop\"}"
	decoder := NewCommandCodeDecoder(iotest.OneByteReader(strings.NewReader(input)))
	stream := NewCommandCodeStream("test")
	for !stream.Finished {
		event, err := decoder.Next()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := stream.Convert(event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := decoder.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
	decoder = NewCommandCodeDecoder(strings.NewReader("data: {\"type\":\"text-delta\",\n" + "data: \"text\":\"multiline\"}\n\n"))
	if event, err := decoder.Next(); err != nil || !gjson.ValidBytes(event) {
		t.Fatalf("multiline SSE: %s %v", event, err)
	}
}

func TestCommandCodeSessionsCoalesceAndIsolate(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	var sessions CommandCodeSessions
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h := http.Header{"Authorization": []string{"Bearer user_one"}, "Traceparent": []string{fmt.Sprint(i)}}
			id, err := sessions.Ensure(context.Background(), server.Client(), server.URL, "user_one", h)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- id
		}(i)
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Error("same credential produced different sessions")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("initialization requests=%d want 2", calls.Load())
	}
	other, err := sessions.Ensure(context.Background(), server.Client(), server.URL, "user_two", http.Header{"Authorization": []string{"Bearer user_two"}})
	if err != nil || other == first {
		t.Fatalf("credential isolation failed %v", err)
	}
	if reflect.DeepEqual(CommandCodeFingerprint("user_one"), CommandCodeFingerprint("user_two")) {
		t.Fatal("fingerprints collide")
	}
	if !reflect.DeepEqual(CommandCodeFingerprint("user_one"), CommandCodeFingerprint("user_one")) {
		t.Fatal("fingerprint unstable")
	}
}

func TestCommandCodeSessionCapacityCancellationAndFailure(t *testing.T) {
	sessions := CommandCodeSessions{entries: make(map[[32]byte]*commandCodeSession)}
	for i := 0; i < CommandCodeMaxSessions; i++ {
		id := commandCodeIdentity(fmt.Sprint(i), "", nil)
		sessions.entries[id] = &commandCodeSession{ready: make(chan struct{}), expires: time.Now().Add(time.Hour)}
	}
	if _, err := sessions.Ensure(context.Background(), http.DefaultClient, "http://unused", "user_one", nil); err == nil {
		t.Fatal("unbounded initialization")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	key := commandCodeIdentity("http://unused", "user_wait", nil)
	sessions.entries[key] = &commandCodeSession{ready: make(chan struct{}), expires: time.Now().Add(time.Hour)}
	if _, err := sessions.Ensure(ctx, http.DefaultClient, "http://unused", "user_wait", nil); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer server.Close()
	var failed CommandCodeSessions
	if _, err := failed.Ensure(context.Background(), server.Client(), server.URL, "user_fail", nil); err == nil {
		t.Fatal("init failure ignored")
	}
	if len(failed.entries) != 0 {
		t.Fatal("failed init cached")
	}
}

func TestCommandCodeCatalog(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]string{"id": r.Header.Get("Authorization")}, map[string]string{"id": "shared"}}})
	}))
	defer server.Close()
	var catalog CommandCodeCatalog
	for _, key := range []string{"user_one", "user_one", "user_two"} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
		req.Header.Set("Authorization", key)
		models, err := catalog.Models(context.Background(), server.Client(), req)
		if err != nil || len(models) != 2 || models[0] != key {
			t.Fatalf("catalog %v %v", models, err)
		}
	}
	if count.Load() != 2 {
		t.Fatalf("catalog fetches=%d", count.Load())
	}
}

func TestCommandCodeRetryAfter(t *testing.T) {
	err := CommandCodeHTTPError(402, "generation request rejected", http.Header{"Retry-After": []string{"45"}, "Set-Cookie": []string{"private"}}).(*commandCodeError)
	if err.StatusCode() != 429 || err.RetryAfter() == nil || *err.RetryAfter() != 45*time.Second {
		t.Fatalf("bad retry metadata: %+v", err)
	}
	if err.Headers().Get("Set-Cookie") != "" {
		t.Fatal("unrelated header leaked")
	}
	if !strings.Contains(err.Error(), "rate_limit_error") {
		t.Fatal("wrong error category")
	}
}

func BenchmarkCommandCodeStream(b *testing.B) {
	data := []byte(`{"type":"text-delta","text":"abcdefghijklmnopqrstuvwxyz"}`)
	b.ReportAllocs()
	stream := NewCommandCodeStream("test")
	for i := 0; i < b.N; i++ {
		if _, err := stream.Convert(data); err != nil {
			b.Fatal(err)
		}
	}
}
