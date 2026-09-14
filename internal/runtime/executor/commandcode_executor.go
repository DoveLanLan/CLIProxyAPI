package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// CommandCodeExecutor connects directly to Command Code without another process.
type CommandCodeExecutor struct {
	cfg      *config.Config
	sessions helps.CommandCodeSessions
	catalog  helps.CommandCodeCatalog
}

func NewCommandCodeExecutor(cfg *config.Config) *CommandCodeExecutor {
	return &CommandCodeExecutor{cfg: cfg}
}
func (e *CommandCodeExecutor) Identifier() string { return "commandcode" }

func (e *CommandCodeExecutor) PrepareRequest(req *http.Request, auth *cliproxyauth.Auth) error {
	if req == nil {
		return fmt.Errorf("commandcode: request is nil")
	}
	key := ""
	if auth != nil {
		key = strings.TrimSpace(auth.Attributes["api_key"])
	}
	if !helps.ValidCommandCodeKey(key) {
		return helps.CommandCodeError(401, "invalid or missing API key")
	}
	if auth != nil {
		util.ApplyCustomHeadersFromAttrs(req, auth.Attributes)
	}
	helps.ApplyCommandCodeHeaders(req, key, "")
	return nil
}

func (e *CommandCodeExecutor) HttpRequest(ctx context.Context, auth *cliproxyauth.Auth, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("commandcode: request is nil")
	}
	if ctx == nil {
		ctx = req.Context()
	}
	req = req.Clone(ctx)
	if err := e.PrepareRequest(req, auth); err != nil {
		return nil, err
	}
	client := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client.Do(req)
}

func (e *CommandCodeExecutor) Refresh(_ context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	return auth, nil
}

func (e *CommandCodeExecutor) CountTokens(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return NewOpenAICompatExecutor(e.Identifier(), e.cfg).CountTokens(ctx, auth, req, opts)
}

func (e *CommandCodeExecutor) baseURL(auth *cliproxyauth.Auth) string {
	if auth != nil {
		if base := strings.TrimRight(strings.TrimSpace(auth.Attributes["base_url"]), "/"); base != "" {
			return base
		}
	}
	return helps.CommandCodeBaseURL
}

func (e *CommandCodeExecutor) start(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, reporter *helps.UsageReporter) (*http.Response, []byte, error) {
	if len(req.Payload) > helps.CommandCodeMaxRequestBytes || len(opts.OriginalRequest) > helps.CommandCodeMaxRequestBytes {
		return nil, nil, helps.CommandCodeError(413, "request exceeds 8 MiB limit")
	}
	if !gjson.ValidBytes(req.Payload) {
		return nil, nil, helps.CommandCodeError(400, "invalid request JSON")
	}
	if opts.Alt != "" {
		return nil, nil, helps.CommandCodeError(400, "unsupported alternate endpoint")
	}
	if path := helps.PayloadRequestPath(opts); strings.Contains(path, "/images/") || strings.Contains(path, "/videos") {
		return nil, nil, helps.CommandCodeError(400, "media generation is not supported")
	}
	baseModel := thinking.ParseSuffix(req.Model).ModelName
	from, to := opts.SourceFormat, sdktranslator.FromString("openai")
	original := opts.OriginalRequest
	if len(original) == 0 {
		original = req.Payload
	}
	for _, tool := range gjson.GetBytes(original, "tools").Array() {
		kind := tool.Get("type").String()
		if kind != "" && kind != "function" && kind != "custom" {
			return nil, nil, helps.CommandCodeError(400, "built-in server-side tools are not supported")
		}
	}
	if (from.String() == "openai-response" || from.String() == "codex") &&
		(gjson.GetBytes(original, "previous_response_id").String() != "" || gjson.GetBytes(original, "background").Bool()) {
		return nil, nil, helps.CommandCodeError(400, "stateful or background Responses requests are not supported; send full history")
	}
	translated := sdktranslator.TranslateRequest(from, to, baseModel, req.Payload, opts.Stream)
	originalTranslated := sdktranslator.TranslateRequest(from, to, baseModel, original, opts.Stream)
	var err error
	translated, err = thinking.ApplyThinking(translated, req.Model, from.String(), "openai", e.Identifier())
	if err != nil {
		return nil, nil, err
	}
	translated = helps.ApplyPayloadConfigWithRequest(e.cfg, baseModel, "openai", from.String(), "", translated, originalTranslated, helps.PayloadRequestedModel(opts, req.Model), helps.PayloadRequestPath(opts), opts.Headers)
	if len(translated) > helps.CommandCodeMaxRequestBytes {
		return nil, nil, helps.CommandCodeError(413, "translated request exceeds 8 MiB limit")
	}
	// Keep recovery provider-local; never relax the shared Claude signature policy.
	if from.String() == "claude" {
		translated, err = helps.RestoreOpenAIReasoningFromClaudeToolCalls(translated, req.Payload, "commandcode executor")
		if err != nil {
			return nil, nil, err
		}
	}
	translated, err = sjson.SetBytes(translated, "model", baseModel)
	if err != nil {
		return nil, nil, err
	}
	if len(translated) > helps.CommandCodeMaxRequestBytes {
		return nil, nil, helps.CommandCodeError(413, "translated request exceeds 8 MiB limit")
	}
	if gjson.GetBytes(translated, "n").Int() > 1 {
		return nil, nil, helps.CommandCodeError(400, "only n=1 is supported")
	}
	reporter.SetTranslatedReasoningEffort(translated, "openai")
	base := e.baseURL(auth)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/alpha/generate", nil)
	if err != nil {
		return nil, nil, err
	}
	if err = e.PrepareRequest(httpReq, auth); err != nil {
		return nil, nil, err
	}
	client := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	key := strings.TrimSpace(auth.Attributes["api_key"])
	session, err := e.sessions.Ensure(ctx, client, base, key, httpReq.Header)
	if err != nil {
		return nil, nil, err
	}
	// Caller-provided session IDs are scoped by selected credential on the upstream.
	for _, candidate := range []string{opts.Headers.Get("x-session-id"), opts.Headers.Get("x-claude-code-session-id"), opts.Headers.Get("session_id"), gjson.GetBytes(translated, "prompt_cache_key").String()} {
		if len(candidate) >= 8 && len(candidate) <= 256 && !strings.ContainsAny(candidate, "\r\n") {
			session = candidate
			break
		}
	}
	wire, err := helps.CommandCodeRequest(translated, session)
	if err != nil {
		return nil, nil, err
	}
	httpReq.Body = io.NopCloser(bytes.NewReader(wire))
	httpReq.ContentLength = int64(len(wire))
	helps.ApplyCommandCodeHeaders(httpReq, key, session)
	helps.RecordAPIRequest(ctx, e.cfg, helps.UpstreamRequestLog{URL: httpReq.URL.String(), Method: http.MethodPost, Headers: httpReq.Header.Clone(), Body: wire, Provider: e.Identifier(), AuthID: auth.ID, AuthLabel: auth.Label})
	client = reporter.TrackHTTPClient(client)
	response, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	helps.RecordAPIResponseMetadata(ctx, e.cfg, response.StatusCode, response.Header.Clone())
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		helps.CloseCommandCodeBody(response.Body)
		return nil, nil, helps.CommandCodeHTTPError(response.StatusCode, "generation request rejected", response.Header)
	}
	return response, translated, nil
}

func (e *CommandCodeExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (result cliproxyexecutor.Response, err error) {
	reporter := helps.NewExecutorUsageReporter(ctx, e, thinking.ParseSuffix(req.Model).ModelName, auth)
	defer reporter.TrackFailure(ctx, &err)
	opts.Stream = false
	response, translated, err := e.start(ctx, auth, req, opts, reporter)
	if err != nil {
		return result, err
	}
	defer helps.CloseCommandCodeBody(response.Body)
	decoder := helps.NewCommandCodeDecoder(response.Body)
	stream := helps.NewCommandCodeStream(req.Model)
	var aggregate helps.CommandCodeAggregate
	for !stream.Finished {
		data, errRead := decoder.Next()
		if errRead != nil {
			if errRead == io.EOF {
				errRead = helps.CommandCodeError(502, "upstream stream truncated")
			}
			return result, errRead
		}
		lines, errConvert := stream.Convert(data)
		if errConvert != nil {
			return result, errConvert
		}
		for _, line := range lines {
			if errAdd := aggregate.Add(line); errAdd != nil {
				return result, errAdd
			}
		}
	}
	body, err := aggregate.Response()
	if err != nil {
		return result, err
	}
	reporter.Publish(ctx, helps.ParseOpenAIUsage(body))
	reporter.EnsurePublished(ctx)
	var param any
	result.Payload = sdktranslator.TranslateNonStream(ctx, sdktranslator.FromString("openai"), cliproxyexecutor.ResponseFormatOrSource(opts), req.Model, opts.OriginalRequest, translated, body, &param)
	result.Headers = helps.CommandCodeResponseHeaders(false)
	return result, nil
}

func (e *CommandCodeExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (_ *cliproxyexecutor.StreamResult, err error) {
	reporter := helps.NewExecutorUsageReporter(ctx, e, thinking.ParseSuffix(req.Model).ModelName, auth)
	defer reporter.TrackFailure(ctx, &err)
	opts.Stream = true
	original := opts.OriginalRequest
	if len(original) == 0 {
		original = req.Payload
	}
	// Some shared translators inspect the original body's stream flag, including
	// SDK calls that select ExecuteStream without specifying a body flag.
	opts.OriginalRequest, err = sjson.SetBytes(original, "stream", true)
	if err != nil {
		return nil, err
	}
	response, translated, err := e.start(ctx, auth, req, opts, reporter)
	if err != nil {
		return nil, err
	}
	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		defer helps.CloseCommandCodeBody(response.Body)
		stream := helps.NewCommandCodeStream(req.Model)
		decoder := helps.NewCommandCodeDecoder(response.Body)
		var param any
		translatedBytes := 0
		send := func(chunk cliproxyexecutor.StreamChunk) bool {
			select {
			case out <- chunk:
				return true
			case <-ctx.Done():
				return false
			}
		}
		fail := func(err error) { reporter.PublishFailure(ctx, err); send(cliproxyexecutor.StreamChunk{Err: err}) }
		for !stream.Finished {
			data, errRead := decoder.Next()
			if errRead != nil {
				if errRead == io.EOF {
					errRead = helps.CommandCodeError(502, "upstream stream truncated")
				}
				fail(errRead)
				return
			}
			lines, errConvert := stream.Convert(data)
			if errConvert != nil {
				fail(errConvert)
				return
			}
			for _, line := range lines {
				// Existing non-OpenAI translators may accumulate output internally.
				// Bound their input without buffering another copy in this executor.
				if cliproxyexecutor.ResponseFormatOrSource(opts).String() != "openai" {
					translatedBytes += len(line)
					if translatedBytes > helps.CommandCodeMaxResponseBytes {
						fail(helps.CommandCodeError(502, "translated stream exceeds 16 MiB limit"))
						return
					}
				}
				for _, chunk := range sdktranslator.TranslateStream(ctx, sdktranslator.FromString("openai"), cliproxyexecutor.ResponseFormatOrSource(opts), req.Model, opts.OriginalRequest, translated, line, &param) {
					if !send(cliproxyexecutor.StreamChunk{Payload: chunk}) {
						reporter.PublishFailure(ctx, ctx.Err())
						return
					}
				}
			}
		}
		if stream.Usage != nil {
			usage, _ := json.Marshal(map[string]any{"usage": stream.Usage})
			reporter.Publish(ctx, helps.ParseOpenAIUsage(usage))
		}
		reporter.EnsurePublished(ctx)
		for _, chunk := range sdktranslator.TranslateStream(ctx, sdktranslator.FromString("openai"), cliproxyexecutor.ResponseFormatOrSource(opts), req.Model, opts.OriginalRequest, translated, []byte("data: [DONE]"), &param) {
			if !send(cliproxyexecutor.StreamChunk{Payload: chunk}) {
				return
			}
		}
	}()
	return &cliproxyexecutor.StreamResult{Headers: helps.CommandCodeResponseHeaders(true), Chunks: out}, nil
}

// FetchModels caches discovery per credential; it does not initialize generation sessions.
func (e *CommandCodeExecutor) FetchModels(ctx context.Context, auth *cliproxyauth.Auth) ([]string, error) {
	base := e.baseURL(auth)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/provider/v1/models", nil)
	if err != nil {
		return nil, err
	}
	if err = e.PrepareRequest(req, auth); err != nil {
		return nil, err
	}
	req.Header.Del("x-cmd-zdr")
	client := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return e.catalog.Models(ctx, client, req)
}
