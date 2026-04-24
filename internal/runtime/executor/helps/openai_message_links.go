package helps

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// NormalizeOpenAIToolMessageLinks patches OpenAI-style chat payloads so providers
// that require strict tool/reasoning message linkage can accept translated
// multi-turn conversations.
func NormalizeOpenAIToolMessageLinks(body []byte, component string) ([]byte, error) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return body, nil
	}

	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return body, nil
	}

	component = strings.TrimSpace(component)
	if component == "" {
		component = "openai message normalizer"
	}

	out := body
	pending := make([]string, 0)
	patched := 0
	patchedReasoning := 0
	ambiguous := 0
	latestReasoning := ""
	hasLatestReasoning := false

	removePending := func(id string) {
		for idx := range pending {
			if pending[idx] != id {
				continue
			}
			pending = append(pending[:idx], pending[idx+1:]...)
			return
		}
	}

	msgs := messages.Array()
	for msgIdx := range msgs {
		msg := msgs[msgIdx]
		role := strings.TrimSpace(msg.Get("role").String())
		switch role {
		case "assistant":
			reasoning := msg.Get("reasoning_content")
			if reasoning.Exists() {
				reasoningText := reasoning.String()
				if strings.TrimSpace(reasoningText) != "" {
					latestReasoning = reasoningText
					hasLatestReasoning = true
				}
			}

			toolCalls := msg.Get("tool_calls")
			if !toolCalls.Exists() || !toolCalls.IsArray() || len(toolCalls.Array()) == 0 {
				continue
			}

			if !reasoning.Exists() || strings.TrimSpace(reasoning.String()) == "" {
				reasoningText := fallbackAssistantReasoning(msg, hasLatestReasoning, latestReasoning)
				path := fmt.Sprintf("messages.%d.reasoning_content", msgIdx)
				next, err := sjson.SetBytes(out, path, reasoningText)
				if err != nil {
					return body, fmt.Errorf("%s: failed to set assistant reasoning_content: %w", component, err)
				}
				out = next
				patchedReasoning++
			}

			for _, tc := range toolCalls.Array() {
				id := strings.TrimSpace(tc.Get("id").String())
				if id == "" {
					continue
				}
				pending = append(pending, id)
			}
		case "tool":
			toolCallID := strings.TrimSpace(msg.Get("tool_call_id").String())
			if toolCallID == "" {
				toolCallID = strings.TrimSpace(msg.Get("call_id").String())
				if toolCallID != "" {
					path := fmt.Sprintf("messages.%d.tool_call_id", msgIdx)
					next, err := sjson.SetBytes(out, path, toolCallID)
					if err != nil {
						return body, fmt.Errorf("%s: failed to set tool_call_id from call_id: %w", component, err)
					}
					out = next
					patched++
				}
			}
			if toolCallID == "" {
				if len(pending) == 1 {
					toolCallID = pending[0]
					path := fmt.Sprintf("messages.%d.tool_call_id", msgIdx)
					next, err := sjson.SetBytes(out, path, toolCallID)
					if err != nil {
						return body, fmt.Errorf("%s: failed to infer tool_call_id: %w", component, err)
					}
					out = next
					patched++
				} else if len(pending) > 1 {
					ambiguous++
				}
			}
			if toolCallID != "" {
				removePending(toolCallID)
			}
		}
	}

	if patched > 0 || patchedReasoning > 0 {
		log.WithFields(log.Fields{
			"patched_tool_messages":      patched,
			"patched_reasoning_messages": patchedReasoning,
		}).Debugf("%s: normalized tool message fields", component)
	}
	if ambiguous > 0 {
		log.WithFields(log.Fields{
			"ambiguous_tool_messages": ambiguous,
			"pending_tool_calls":      len(pending),
		}).Warnf("%s: tool messages missing tool_call_id with ambiguous candidates", component)
	}

	return out, nil
}

func fallbackAssistantReasoning(msg gjson.Result, hasLatest bool, latest string) string {
	if hasLatest && strings.TrimSpace(latest) != "" {
		return latest
	}

	content := msg.Get("content")
	if content.Type == gjson.String {
		if text := strings.TrimSpace(content.String()); text != "" {
			return text
		}
	}
	if content.IsArray() {
		parts := make([]string, 0, len(content.Array()))
		for _, item := range content.Array() {
			text := strings.TrimSpace(item.Get("text").String())
			if text == "" {
				continue
			}
			parts = append(parts, text)
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}

	return "[reasoning unavailable]"
}
