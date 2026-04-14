package claude

import (
	"encoding/json"
	"strings"
)

type ParsedMessage struct {
	Role        string
	Text        string
	UUID        string
	Timestamp   string
	SessionID   string
	CWD         string
	Slug        string
	IsSidechain bool
}

// ParseLine extracts searchable text from a Claude JSONL line.
// Returns nil for non-message types, tool-only messages, and thinking-only messages.
func ParseLine(line []byte) *ParsedMessage {
	if len(line) == 0 {
		return nil
	}

	// Lazy parse: extract type field first to skip non-message lines (~60% of lines)
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return nil
	}
	if envelope.Type != "user" && envelope.Type != "assistant" {
		return nil
	}

	// Full parse for user/assistant messages
	var raw struct {
		Type    string `json:"type"`
		Message struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
		UUID        string `json:"uuid"`
		Timestamp   string `json:"timestamp"`
		SessionID   string `json:"sessionId"`
		CWD         string `json:"cwd"`
		Slug        string `json:"slug"`
		IsSidechain bool   `json:"isSidechain"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil
	}

	text := extractText(raw.Message.Content)
	if text == "" {
		return nil
	}

	return &ParsedMessage{
		Role:        raw.Type,
		Text:        text,
		UUID:        raw.UUID,
		Timestamp:   raw.Timestamp,
		SessionID:   raw.SessionID,
		CWD:         raw.CWD,
		Slug:        raw.Slug,
		IsSidechain: raw.IsSidechain,
	}
}

// extractText handles both string content and array-of-blocks content.
// For arrays, only "text" type blocks are kept (tool_use, tool_result, thinking skipped).
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try string first (common for user messages)
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}
