package helps

import "testing"

func TestDetectDeepSeekClaudeCompatibilityIssueImage(t *testing.T) {
	body := []byte(`{
		"messages":[{"role":"user","content":[{
			"type":"tool_result",
			"tool_use_id":"call_1",
			"content":[
				{"type":"text","text":"result"},
				{"type":"image","source":{"type":"base64","media_type":"image/png","data":"aW1hZ2U="}}
			]
		}]}]
	}`)

	issue, ok := DetectDeepSeekClaudeCompatibilityIssue(body, "deepseek-v4-pro")
	if !ok {
		t.Fatal("expected image compatibility issue")
	}
	if issue.Code != "model_text_only" {
		t.Fatalf("issue.Code = %q, want model_text_only", issue.Code)
	}
}

func TestDetectDeepSeekClaudeCompatibilityIssueNamedToolChoice(t *testing.T) {
	body := []byte(`{
		"tool_choice":{"type":"tool","name":"read"},
		"messages":[{"role":"user","content":"read"}]
	}`)

	issue, ok := DetectDeepSeekClaudeCompatibilityIssue(body, "deepseek-v4-pro")
	if !ok {
		t.Fatal("expected named tool-choice compatibility issue")
	}
	if issue.Code != "unsupported_tool_choice" {
		t.Fatalf("issue.Code = %q, want unsupported_tool_choice", issue.Code)
	}
}

func TestDetectDeepSeekClaudeCompatibilityIssueAllowsTextAndAutomaticTools(t *testing.T) {
	tests := [][]byte{
		[]byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`),
		[]byte(`{"tool_choice":{"type":"auto"},"messages":[{"role":"user","content":"hello"}]}`),
		[]byte(`{"tool_choice":{"type":"any"},"messages":[{"role":"user","content":"hello"}]}`),
	}
	for _, body := range tests {
		if issue, ok := DetectDeepSeekClaudeCompatibilityIssue(body, "deepseek-v4-pro"); ok {
			t.Fatalf("unexpected compatibility issue %#v for body %s", issue, body)
		}
	}
}
