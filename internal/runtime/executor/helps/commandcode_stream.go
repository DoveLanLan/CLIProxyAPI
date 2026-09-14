package helps

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const CommandCodeMaxEventBytes = 4 << 20
const CommandCodeMaxResponseBytes = 16 << 20
const commandCodeMaxTools = 1024

// CommandCodeDecoder reads native NDJSON or SSE-framed events independently of
// network chunk boundaries. Command Code generation currently uses NDJSON.
type CommandCodeDecoder struct {
	scanner *bufio.Scanner
	done    bool
}

func NewCommandCodeDecoder(reader io.Reader) *CommandCodeDecoder {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), CommandCodeMaxEventBytes)
	return &CommandCodeDecoder{scanner: scanner}
}

func (d *CommandCodeDecoder) Next() ([]byte, error) {
	if d.done {
		return nil, io.EOF
	}
	var data strings.Builder
	for d.scanner.Scan() {
		line := d.scanner.Text()
		trimmed := strings.TrimSpace(line)
		if data.Len() == 0 && (strings.HasPrefix(trimmed, "{") || trimmed == "[DONE]") {
			return []byte(trimmed), nil
		}
		if line == "" {
			if data.Len() != 0 {
				return []byte(strings.TrimSuffix(data.String(), "\n")), nil
			}
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		part := strings.TrimPrefix(line, "data:")
		part = strings.TrimPrefix(part, " ")
		if data.Len()+len(part)+1 > CommandCodeMaxEventBytes {
			return nil, CommandCodeError(502, "SSE event exceeds size limit")
		}
		data.WriteString(part)
		data.WriteByte('\n')
	}
	d.done = true
	if err := d.scanner.Err(); err != nil {
		return nil, fmt.Errorf("commandcode: read event: %w", err)
	}
	if data.Len() > 0 {
		return []byte(strings.TrimSuffix(data.String(), "\n")), nil
	}
	return nil, io.EOF
}

type commandCodeTool struct {
	index    int
	name     string
	started  bool
	delta    bool
	complete bool
}

// CommandCodeStream normalizes upstream events without retaining the answer text.
type CommandCodeStream struct {
	id           string
	model        string
	created      int64
	tools        map[string]*commandCodeTool
	Finished     bool
	Meaningful   bool
	Usage        map[string]any
	finishReason string
}

func NewCommandCodeStream(model string) *CommandCodeStream {
	return &CommandCodeStream{id: "chatcmpl-" + uuid.NewString(), model: model, created: time.Now().Unix(), tools: make(map[string]*commandCodeTool)}
}

func (s *CommandCodeStream) chunk(delta map[string]any, finish any, usage any) []byte {
	value := map[string]any{"id": s.id, "object": "chat.completion.chunk", "created": s.created, "model": s.model,
		"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}
	if usage != nil {
		value["usage"] = usage
	}
	result, _ := json.Marshal(value)
	return append([]byte("data: "), result...)
}

func (s *CommandCodeStream) tool(id string) (*commandCodeTool, error) {
	if id == "" {
		return nil, CommandCodeError(502, "tool event missing ID")
	}
	if len(id) > 256 {
		return nil, CommandCodeError(502, "tool ID exceeds size limit")
	}
	if tool := s.tools[id]; tool != nil {
		return tool, nil
	}
	if len(s.tools) >= commandCodeMaxTools {
		return nil, CommandCodeError(502, "too many tool calls")
	}
	tool := &commandCodeTool{index: len(s.tools)}
	// gjson strings may reference the entire event buffer. Copy only the small
	// retained identifier so large tool arguments can be collected immediately.
	s.tools[strings.Clone(id)] = tool
	return tool, nil
}

// Convert returns zero or one normalized OpenAI SSE lines per Command Code event.
func (s *CommandCodeStream) Convert(data []byte) ([][]byte, error) {
	if string(data) == "[DONE]" {
		if !s.Finished {
			return nil, CommandCodeError(502, "stream ended without finish event")
		}
		return nil, nil
	}
	if !gjson.ValidBytes(data) {
		return nil, CommandCodeError(502, "invalid upstream SSE JSON")
	}
	event := gjson.ParseBytes(data)
	kind := event.Get("type").String()
	if s.Finished {
		return nil, nil
	}
	delta := map[string]any{}
	switch kind {
	case "text-delta", "reasoning-delta":
		value := event.Get("text").String()
		if value == "" {
			value = event.Get("delta").String()
		}
		if value == "" {
			return nil, nil
		}
		field := "content"
		if kind == "reasoning-delta" {
			field = "reasoning_content"
		}
		delta[field] = value
	case "tool-input-start", "tool-input-delta", "tool-call":
		id := event.Get("toolCallId").String()
		if id == "" {
			id = event.Get("id").String()
		}
		tool, err := s.tool(id)
		if err != nil {
			return nil, err
		}
		if tool.complete {
			return nil, nil
		}
		if name := event.Get("toolName").String(); name != "" {
			if len(name) > 256 {
				return nil, CommandCodeError(502, "tool name exceeds size limit")
			}
			tool.name = strings.Clone(name)
		}
		fn := map[string]any{}
		if !tool.started {
			if tool.name == "" {
				return nil, CommandCodeError(502, "tool event missing name")
			}
			fn["name"] = tool.name
		}
		if kind == "tool-input-delta" {
			value := event.Get("inputTextDelta").String()
			if value == "" {
				value = event.Get("delta").String()
			}
			fn["arguments"] = value
			tool.delta = true
		} else if kind == "tool-call" {
			tool.complete = true
			if tool.delta {
				return nil, nil
			}
			input := event.Get("input")
			args := input.Raw
			if input.Type == gjson.String {
				args = input.String()
			}
			if args == "" {
				args = "{}"
			}
			if !gjson.Valid(args) {
				return nil, CommandCodeError(502, "invalid upstream tool arguments")
			}
			fn["arguments"] = args
		}
		call := map[string]any{"index": tool.index, "function": fn}
		if !tool.started {
			call["id"], call["type"] = id, "function"
		}
		tool.started = true
		delta["tool_calls"] = []any{call}
	case "finish-step", "finish":
		if reason := event.Get("finishReason").String(); reason != "" {
			s.finishReason = reason
		}
		usage := event.Get("totalUsage")
		if !usage.Exists() {
			usage = event.Get("usage")
		}
		if usage.Exists() {
			s.Usage = commandCodeUsage(usage)
		}
		if kind == "finish-step" {
			return nil, nil
		}
		if !s.Meaningful {
			return nil, CommandCodeError(502, "upstream completed without content or tool calls")
		}
		for _, tool := range s.tools {
			if !tool.complete {
				return nil, CommandCodeError(502, "upstream completed with an unfinished tool call")
			}
		}
		s.Finished = true
		reason := s.finishReason
		if reason == "tool-calls" {
			reason = "tool_calls"
		}
		if reason == "" {
			reason = "stop"
		}
		return [][]byte{s.chunk(delta, reason, s.Usage)}, nil
	case "error", "tool-error":
		status := int(event.Get("error.statusCode").Int())
		if status == 0 {
			status = int(event.Get("statusCode").Int())
		}
		message := event.Get("error.message").String()
		if message == "" {
			message = event.Get("message").String()
		}
		if len(message) >= 5 && message[0] == '<' && message[4] == '>' {
			if parsed, err := strconv.Atoi(message[1:4]); err == nil {
				status = parsed
			}
		}
		return nil, CommandCodeError(status, "upstream stream error")
	case "start", "start-step", "text-start", "text-end", "reasoning-start", "reasoning-end", "tool-input-end", "provider-metadata":
		return nil, nil
	default:
		// Ignore forward-compatible metadata, but never treat EOF as a successful finish.
		return nil, nil
	}
	if !s.Meaningful {
		delta["role"] = "assistant"
	}
	s.Meaningful = true
	return [][]byte{s.chunk(delta, nil, nil)}, nil
}

func commandCodeUsage(u gjson.Result) map[string]any {
	input, output := u.Get("inputTokens").Int(), u.Get("outputTokens").Int()
	cached := u.Get("cachedInputTokens")
	if !cached.Exists() {
		cached = u.Get("inputTokenDetails.cacheReadTokens")
	}
	return map[string]any{"prompt_tokens": input, "completion_tokens": output, "total_tokens": input + output,
		"prompt_tokens_details": map[string]any{"cached_tokens": cached.Int(), "cache_creation_tokens": u.Get("inputTokenDetails.cacheWriteTokens").Int()}}
}

// CommandCodeAggregate is used only for non-streaming requests and has a hard byte cap.
type CommandCodeAggregate struct {
	text      strings.Builder
	reasoning strings.Builder
	tools     map[int]map[string]any
	bytes     int
	last      gjson.Result
}

func (a *CommandCodeAggregate) Add(line []byte) error {
	a.bytes += len(line)
	if a.bytes > CommandCodeMaxResponseBytes {
		return CommandCodeError(502, "non-stream response exceeds 16 MiB limit; use streaming")
	}
	root := gjson.ParseBytes(bytesWithoutSSEPrefix(line))
	a.last = root
	delta := root.Get("choices.0.delta")
	a.text.WriteString(delta.Get("content").String())
	a.reasoning.WriteString(delta.Get("reasoning_content").String())
	for _, tc := range delta.Get("tool_calls").Array() {
		if a.tools == nil {
			a.tools = make(map[int]map[string]any)
		}
		idx := int(tc.Get("index").Int())
		tool := a.tools[idx]
		if tool == nil {
			tool = map[string]any{"type": "function", "function": map[string]any{"arguments": ""}}
			a.tools[idx] = tool
		}
		if id := tc.Get("id").String(); id != "" {
			tool["id"] = id
		}
		fn := tool["function"].(map[string]any)
		if name := tc.Get("function.name").String(); name != "" {
			fn["name"] = name
		}
		fn["arguments"] = fn["arguments"].(string) + tc.Get("function.arguments").String()
	}
	return nil
}

func bytesWithoutSSEPrefix(line []byte) []byte {
	return []byte(strings.TrimSpace(strings.TrimPrefix(string(line), "data:")))
}

func (a *CommandCodeAggregate) Response() ([]byte, error) {
	message := map[string]any{"role": "assistant", "content": a.text.String()}
	if a.reasoning.Len() > 0 {
		message["reasoning_content"] = a.reasoning.String()
	}
	if len(a.tools) > 0 {
		tools := make([]any, len(a.tools))
		for i, tool := range a.tools {
			tools[i] = tool
		}
		message["tool_calls"] = tools
	}
	return json.Marshal(map[string]any{"id": a.last.Get("id").String(), "object": "chat.completion", "created": a.last.Get("created").Int(), "model": a.last.Get("model").String(),
		"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": a.last.Get("choices.0.finish_reason").String()}}, "usage": a.last.Get("usage").Value()})
}
