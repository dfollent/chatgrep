package codex

import (
	"encoding/json"
	"fmt"
	"strings"
)

const userMessagePrefix = "## My request for Codex:"

type ParsedMessage struct {
	Role      string
	Text      string
	ID        string
	Timestamp string
	Source    string
}

// ParseLine extracts visible chat text from a Codex rollout JSONL line.
// Visible chat can appear in event_msg or response_item/message lines.
func ParseLine(line []byte, lineNum int) *ParsedMessage {
	if len(line) == 0 {
		return nil
	}

	var envelope struct {
		Timestamp string          `json:"timestamp"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return nil
	}
	if len(envelope.Payload) == 0 {
		return nil
	}

	var (
		role   string
		text   string
		source string
	)

	switch envelope.Type {
	case "event_msg":
		var event struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(envelope.Payload, &event); err != nil {
			return nil
		}
		switch event.Type {
		case "user_message":
			role = "user"
			text = normalizeUserMessage(event.Message)
		case "agent_message":
			role = "assistant"
			text = strings.TrimSpace(event.Message)
		default:
			return nil
		}
		source = "event_msg"
	case "response_item":
		var item struct {
			Type    string          `json:"type"`
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(envelope.Payload, &item); err != nil {
			return nil
		}
		if item.Type != "message" || (item.Role != "user" && item.Role != "assistant") {
			return nil
		}
		role = item.Role
		text = extractResponseItemText(item.Content)
		if role == "user" {
			text = normalizeUserMessage(text)
		}
		source = "response_item"
	default:
		return nil
	}

	if text == "" {
		return nil
	}

	return &ParsedMessage{
		Role:      role,
		Text:      text,
		ID:        previewID(lineNum),
		Timestamp: envelope.Timestamp,
		Source:    source,
	}
}

func previewID(lineNum int) string {
	return fmt.Sprintf("line-%06d", lineNum)
}

func normalizeUserMessage(message string) string {
	message = strings.TrimSpace(message)
	if strings.HasPrefix(message, userMessagePrefix) {
		message = strings.TrimSpace(strings.TrimPrefix(message, userMessagePrefix))
	}
	return message
}

func extractResponseItemText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "output_text", "input_text", "text":
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
