package helps

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestRestoreOpenAIReasoningFromClaudeToolCalls(t *testing.T) {
	translated := []byte(`{
		"messages":[
			{"role":"user","content":"read"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"result"}
		]
	}`)
	claudeBody := []byte(`{
		"messages":[
			{"role":"user","content":[{"type":"text","text":"read"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"original reasoning","signature":""},
				{"type":"tool_use","id":"call_1","name":"read","input":{}}
			]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"result"}]}
		]
	}`)

	got, err := RestoreOpenAIReasoningFromClaudeToolCalls(translated, claudeBody, "test")
	if err != nil {
		t.Fatalf("RestoreOpenAIReasoningFromClaudeToolCalls() error = %v", err)
	}
	if reasoning := gjson.GetBytes(got, "messages.1.reasoning_content").String(); reasoning != "original reasoning" {
		t.Fatalf("reasoning_content = %q, want original reasoning; body=%s", reasoning, got)
	}
}

func TestRestoreOpenAIReasoningFromClaudeParallelToolCalls(t *testing.T) {
	translated := []byte(`{
		"messages":[{"role":"assistant","content":"","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}},
			{"id":"call_2","type":"function","function":{"name":"search","arguments":"{}"}}
		]}]
	}`)
	claudeBody := []byte(`{
		"messages":[{"role":"assistant","content":[
			{"type":"thinking","thinking":"first line","signature":""},
			{"type":"thinking","thinking":"second line","signature":""},
			{"type":"tool_use","id":"call_1","name":"read","input":{}},
			{"type":"tool_use","id":"call_2","name":"search","input":{}}
		]}]
	}`)

	got, err := RestoreOpenAIReasoningFromClaudeToolCalls(translated, claudeBody, "test")
	if err != nil {
		t.Fatalf("RestoreOpenAIReasoningFromClaudeToolCalls() error = %v", err)
	}
	if reasoning := gjson.GetBytes(got, "messages.0.reasoning_content").String(); reasoning != "first line\n\nsecond line" {
		t.Fatalf("reasoning_content = %q, want joined original reasoning; body=%s", reasoning, got)
	}
}

func TestRestoreOpenAIReasoningFromClaudeToolCallsKeepsExisting(t *testing.T) {
	translated := []byte(`{
		"messages":[{"role":"assistant","reasoning_content":"signed reasoning","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}
		]}]
	}`)
	claudeBody := []byte(`{
		"messages":[{"role":"assistant","content":[
			{"type":"thinking","thinking":"unsigned reasoning","signature":""},
			{"type":"tool_use","id":"call_1","name":"read","input":{}}
		]}]
	}`)

	got, err := RestoreOpenAIReasoningFromClaudeToolCalls(translated, claudeBody, "test")
	if err != nil {
		t.Fatalf("RestoreOpenAIReasoningFromClaudeToolCalls() error = %v", err)
	}
	if reasoning := gjson.GetBytes(got, "messages.0.reasoning_content").String(); reasoning != "signed reasoning" {
		t.Fatalf("reasoning_content = %q, want existing signed reasoning; body=%s", reasoning, got)
	}
}

func TestRestoreOpenAIReasoningFromClaudeToolCallsSkipsAmbiguousID(t *testing.T) {
	translated := []byte(`{
		"messages":[{"role":"assistant","content":"","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}
		]}]
	}`)
	claudeBody := []byte(`{
		"messages":[
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"first reasoning","signature":""},
				{"type":"tool_use","id":"call_1","name":"read","input":{}}
			]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"second reasoning","signature":""},
				{"type":"tool_use","id":"call_1","name":"read","input":{}}
			]}
		]
	}`)

	got, err := RestoreOpenAIReasoningFromClaudeToolCalls(translated, claudeBody, "test")
	if err != nil {
		t.Fatalf("RestoreOpenAIReasoningFromClaudeToolCalls() error = %v", err)
	}
	if reasoning := gjson.GetBytes(got, "messages.0.reasoning_content"); reasoning.Exists() {
		t.Fatalf("reasoning_content = %q, want no restoration for ambiguous ID; body=%s", reasoning.String(), got)
	}
}
