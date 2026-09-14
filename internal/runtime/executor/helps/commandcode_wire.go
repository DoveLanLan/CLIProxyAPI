package helps

// Protocol mapping adapted from MAXeaglet/commandcode-proxy (MIT).
// See docs/licenses/commandcode-proxy.txt for attribution.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const CommandCodeBaseURL = "https://api.commandcode.ai"
const CommandCodeVersion = "1.53.1"
const CommandCodeMaxRequestBytes = 8 << 20
const commandCodeWorkingDir = `C:\Users\dev\projects\app`

// CommandCodeRequest converts normalized OpenAI chat into the upstream CLI envelope.
func CommandCodeRequest(body []byte, session string) ([]byte, error) {
	if !gjson.ValidBytes(body) {
		return nil, CommandCodeError(400, "invalid request JSON")
	}
	root := gjson.ParseBytes(body)
	if !root.Get("messages").IsArray() {
		return nil, CommandCodeError(400, "messages must be an array")
	}
	system := make([]any, 0)
	messages := make([]any, 0)
	toolNames := make(map[string]string)
	for _, msg := range root.Get("messages").Array() {
		for _, tc := range msg.Get("tool_calls").Array() {
			toolNames[tc.Get("id").String()] = tc.Get("function.name").String()
		}
	}
	for _, msg := range root.Get("messages").Array() {
		role := msg.Get("role").String()
		content := msg.Get("content")
		parts := make([]any, 0)
		if role == "assistant" && msg.Get("reasoning_content").String() != "" {
			parts = append(parts, map[string]any{"type": "reasoning", "text": msg.Get("reasoning_content").String()})
		}
		if content.Type == gjson.String {
			if content.String() != "" {
				parts = append(parts, map[string]any{"type": "text", "text": content.String()})
			}
		} else if content.IsArray() {
			for _, part := range content.Array() {
				switch part.Get("type").String() {
				case "text":
					parts = append(parts, part.Value())
				case "reasoning":
					if msg.Get("reasoning_content").String() == "" {
						parts = append(parts, part.Value())
					}
				case "image_url":
					url := part.Get("image_url.url").String()
					image := map[string]any{"type": "image", "image": url}
					if strings.HasPrefix(url, "data:") {
						if end := strings.IndexAny(url[5:], ";,"); end >= 0 {
							image["mimeType"] = url[5 : 5+end]
						}
					}
					parts = append(parts, image)
				default:
					return nil, CommandCodeError(400, "unsupported message content type")
				}
			}
		}
		switch role {
		case "system", "developer":
			system = append(system, parts...)
			continue
		case "assistant":
			for _, tc := range msg.Get("tool_calls").Array() {
				args := tc.Get("function.arguments")
				var input any = map[string]any{}
				if args.Type == gjson.String {
					if err := json.Unmarshal([]byte(args.String()), &input); err != nil {
						return nil, CommandCodeError(400, "invalid tool call arguments")
					}
				} else if args.Exists() {
					input = args.Value()
				}
				parts = append(parts, map[string]any{"type": "tool-call", "toolCallId": tc.Get("id").String(), "toolName": tc.Get("function.name").String(), "input": input})
			}
		case "tool":
			id := msg.Get("tool_call_id").String()
			name := toolNames[id]
			if name == "" {
				name = msg.Get("name").String()
			}
			if id == "" || name == "" {
				return nil, CommandCodeError(400, "tool result has no matching tool call")
			}
			value := content.String()
			if content.IsArray() {
				texts := make([]string, 0)
				for _, part := range content.Array() {
					if part.Get("type").String() != "text" {
						return nil, CommandCodeError(400, "only text tool results are supported")
					}
					texts = append(texts, part.Get("text").String())
				}
				value = strings.Join(texts, "\n")
			} else if content.IsObject() {
				value = content.Raw
			}
			parts = []any{map[string]any{"type": "tool-result", "toolCallId": id, "toolName": name, "output": map[string]any{"type": "text", "value": value}}}
		case "user":
		default:
			return nil, CommandCodeError(400, "unsupported message role")
		}
		messages = append(messages, map[string]any{"role": role, "content": parts})
	}
	for i := 0; i+1 < len(system); i++ {
		if block, ok := system[i].(map[string]any); ok {
			if text, okText := block["text"].(string); okText {
				block["text"] = text + "\n"
			}
		}
	}
	if len(system) == 0 {
		// Avoid the upstream's implicit coding-agent system prompt.
		system = append(system, map[string]any{"type": "text", "text": " "})
	}
	if root.Get("prompt_cache_key").String() != "" && !strings.Contains(string(body), `"cache_control"`) {
		if block, ok := system[len(system)-1].(map[string]any); ok {
			block["cache_control"] = map[string]any{"type": "ephemeral"}
		}
	}
	tools := make([]any, 0)
	for _, tool := range root.Get("tools").Array() {
		if tool.Get("type").String() != "function" {
			return nil, CommandCodeError(400, "only function tools are supported")
		}
		fn := tool.Get("function")
		schema := fn.Get("parameters").Value()
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, map[string]any{"name": fn.Get("name").String(), "description": fn.Get("description").String(), "input_schema": schema})
	}
	maxTokens := root.Get("max_completion_tokens").Int()
	if maxTokens == 0 {
		maxTokens = root.Get("max_tokens").Int()
	}
	if maxTokens == 0 {
		maxTokens = 64000
	}
	if maxTokens < 1 || maxTokens > 200000 {
		return nil, CommandCodeError(400, "max_tokens must be between 1 and 200000")
	}
	params := map[string]any{"model": root.Get("model").String(), "messages": messages, "system": system, "tools": tools, "max_tokens": maxTokens, "stream": true}
	for _, key := range []string{"temperature", "reasoning_effort", "parallel_tool_calls"} {
		if val := root.Get(key); val.Exists() {
			params[key] = val.Value()
		}
	}
	if choice := root.Get("tool_choice"); choice.Exists() {
		switch choice.Type {
		case gjson.String:
			value := choice.String()
			if value == "required" {
				value = "any"
			}
			params["tool_choice"] = map[string]any{"type": value}
		default:
			params["tool_choice"] = map[string]any{"type": "tool", "name": choice.Get("function.name").String()}
		}
	}
	envelope := map[string]any{
		"config": map[string]any{"workingDir": commandCodeWorkingDir, "date": time.Now().UTC().Format("2006-01-02"), "environment": "win32", "structure": []any{}, "isGitRepo": false, "currentBranch": "", "mainBranch": "", "gitStatus": "", "recentCommits": []any{}},
		"memory": nil, "taste": nil, "skills": nil, "permissionMode": "standard", "mode": "agent", "params": params,
	}
	if _, err := uuid.Parse(session); err == nil {
		envelope["threadId"] = session
	}
	result, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("commandcode: encode request: %w", err)
	}
	return result, nil
}
