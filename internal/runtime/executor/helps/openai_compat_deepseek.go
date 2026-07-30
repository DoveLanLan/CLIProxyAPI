package helps

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// DeepSeekClaudeCompatibilityIssue describes a request shape that the
// configured text-only DeepSeek OpenAI-compatible upstream cannot accept.
type DeepSeekClaudeCompatibilityIssue struct {
	Code    string
	Message string
}

// DetectDeepSeekClaudeCompatibilityIssue validates Claude-origin request
// features that otherwise produce generic upstream 400 responses.
func DetectDeepSeekClaudeCompatibilityIssue(body []byte, model string) (DeepSeekClaudeCompatibilityIssue, bool) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return DeepSeekClaudeCompatibilityIssue{}, false
	}

	model = strings.TrimSpace(model)
	if model == "" {
		model = "DeepSeek"
	}
	if claudeMessagesContainImage(gjson.GetBytes(body, "messages")) {
		return DeepSeekClaudeCompatibilityIssue{
			Code:    "model_text_only",
			Message: fmt.Sprintf("Model %s only supports text input; remove image blocks or use a vision-capable model", model),
		}, true
	}

	toolChoice := gjson.GetBytes(body, "tool_choice")
	if toolChoice.IsObject() && strings.EqualFold(strings.TrimSpace(toolChoice.Get("type").String()), "tool") {
		return DeepSeekClaudeCompatibilityIssue{
			Code:    "unsupported_tool_choice",
			Message: fmt.Sprintf("Model %s does not support forcing a named tool; use automatic tool choice or omit tool_choice", model),
		}, true
	}

	return DeepSeekClaudeCompatibilityIssue{}, false
}

func claudeMessagesContainImage(messages gjson.Result) bool {
	if !messages.IsArray() {
		return false
	}
	for _, message := range messages.Array() {
		if claudeContentContainsImage(message.Get("content")) {
			return true
		}
	}
	return false
}

func claudeContentContainsImage(content gjson.Result) bool {
	if content.IsArray() {
		for _, part := range content.Array() {
			if claudeContentContainsImage(part) {
				return true
			}
		}
		return false
	}
	if !content.IsObject() {
		return false
	}

	partType := strings.ToLower(strings.TrimSpace(content.Get("type").String()))
	if partType == "image" {
		return true
	}
	if partType == "tool_result" {
		return claudeContentContainsImage(content.Get("content"))
	}
	return false
}
