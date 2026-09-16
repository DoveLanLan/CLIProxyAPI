package helps

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNormalizeOpenAIToolImagesParallelResults(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4.1-flash","stream":true,"messages":[
	{"role":"assistant","reasoning_content":"keep reasoning","tool_calls":[{"id":"a"},{"id":"b"},{"id":"c"}]},
	{"role":"tool","tool_call_id":"a","name":"read","content":[{"type":"text","text":"caption","cache_control":{"type":"ephemeral"}},{"type":"image_url","image_url":{"url":"data:image/png;base64,YQ==","detail":"high"}}]},
	{"role":"tool","tool_call_id":"b","content":[{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}]},
	{"role":"tool","tool_call_id":"c","content":"text result"},
	{"role":"assistant","content":"next"},
	{"role":"tool","tool_call_id":"d","content":[{"type":"image_url","image_url":{"url":"https://example.com/d.png"}}]}
	]}`)
	out, err := NormalizeOpenAIToolImages(body)
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]string{
		"model":                                   "deepseek-v4.1-flash",
		"messages.0.reasoning_content":            "keep reasoning",
		"messages.1.tool_call_id":                 "a",
		"messages.1.name":                         "read",
		"messages.1.content.0.text":               "caption",
		"messages.1.content.0.cache_control.type": "ephemeral",
		"messages.2.tool_call_id":                 "b",
		"messages.3.tool_call_id":                 "c",
		"messages.3.content":                      "text result",
		"messages.4.role":                         "user",
		"messages.4.content.0.text":               "Images returned by tool call a:",
		"messages.4.content.1.image_url.url":      "data:image/png;base64,YQ==",
		"messages.4.content.1.image_url.detail":   "high",
		"messages.4.content.2.text":               "Images returned by tool call b:",
		"messages.4.content.3.image_url.url":      "https://example.com/b.png",
		"messages.5.content":                      "next",
		"messages.6.tool_call_id":                 "d",
		"messages.7.role":                         "user",
		"messages.7.content.1.image_url.url":      "https://example.com/d.png",
	}
	for path, want := range checks {
		if got := gjson.GetBytes(out, path).String(); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if gjson.GetBytes(out, "messages.#").Int() != 8 || gjson.GetBytes(out, "messages.1.content.#").Int() != 1 {
		t.Fatalf("unexpected messages: %s", out)
	}
	again, err := NormalizeOpenAIToolImages(out)
	if err != nil || string(again) != string(out) {
		t.Fatalf("not idempotent: %v", err)
	}
}

func TestNormalizeOpenAIToolImagesUnchanged(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"messages":[]}`,
		`{"messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}]}`,
		`{"messages":[{"role":"tool","tool_call_id":"a","content":"plain"},{"role":"tool","content":[{"type":"text","text":"array"}]}]}`,
	} {
		out, err := NormalizeOpenAIToolImages([]byte(body))
		if err != nil || string(out) != body {
			t.Fatalf("changed no-op request %s: %s, %v", body, out, err)
		}
	}
}
