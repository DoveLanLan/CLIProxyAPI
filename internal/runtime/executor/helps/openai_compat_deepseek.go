package helps

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// DeepSeekClaudeCompatibilityIssue describes a request shape that the
// configured DeepSeek OpenAI-compatible upstream cannot accept.
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
	// Vision support varies by model and upstream. Let the upstream validate
	// translated images instead of treating the whole DeepSeek family as text-only.

	toolChoice := gjson.GetBytes(body, "tool_choice")
	if toolChoice.IsObject() && strings.EqualFold(strings.TrimSpace(toolChoice.Get("type").String()), "tool") {
		return DeepSeekClaudeCompatibilityIssue{
			Code:    "unsupported_tool_choice",
			Message: fmt.Sprintf("Model %s does not support forcing a named tool; use automatic tool choice or omit tool_choice", model),
		}, true
	}

	return DeepSeekClaudeCompatibilityIssue{}, false
}
