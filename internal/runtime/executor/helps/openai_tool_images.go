package helps

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// NormalizeOpenAIToolImages moves image results to user messages because Chat
// tool messages only support text. Flush after each complete tool-result group
// so a user message never interrupts parallel tool responses.
func NormalizeOpenAIToolImages(body []byte) ([]byte, error) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body, nil
	}
	out := make([]gjson.Result, 0, len(messages.Array()))
	var pending []gjson.Result
	changed := false
	flush := func() {
		if len(pending) == 0 {
			return
		}
		user, _ := sjson.SetRaw(`{"role":"user"}`, "content", rawImageMessageArray(pending))
		out = append(out, gjson.Parse(user))
		pending = nil
	}
	for _, message := range messages.Array() {
		if message.Get("role").String() != "tool" {
			flush()
			out = append(out, message)
			continue
		}
		content := message.Get("content")
		var retained, images []gjson.Result
		if content.IsArray() {
			for _, part := range content.Array() {
				if part.Get("type").String() == "image_url" {
					images = append(images, part)
				} else {
					retained = append(retained, part)
				}
			}
		}
		if len(images) == 0 {
			out = append(out, message)
			continue
		}
		changed = true
		label, _ := sjson.Set(`{"type":"text"}`, "text", fmt.Sprintf("Images returned by tool call %s:", message.Get("tool_call_id").String()))
		pending = append(pending, gjson.Parse(label))
		pending = append(pending, images...)
		var updated string
		var err error
		if len(retained) == 0 {
			updated, err = sjson.Set(message.Raw, "content", "Images are included in the following user message.")
		} else {
			updated, err = sjson.SetRaw(message.Raw, "content", rawImageMessageArray(retained))
		}
		if err != nil {
			return nil, fmt.Errorf("normalize tool images: %w", err)
		}
		out = append(out, gjson.Parse(updated))
	}
	if !changed {
		return body, nil
	}
	flush()
	result, err := sjson.SetRawBytes(body, "messages", []byte(rawImageMessageArray(out)))
	if err != nil {
		return nil, fmt.Errorf("normalize tool image messages: %w", err)
	}
	return result, nil
}

func rawImageMessageArray(items []gjson.Result) string {
	var result strings.Builder
	result.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			result.WriteByte(',')
		}
		result.WriteString(item.Raw)
	}
	result.WriteByte(']')
	return result.String()
}
