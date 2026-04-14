package copilot

import (
	"encoding/json"
)

type ParsedMessage struct {
	Role      string
	Text      string
	ID        string
	Timestamp string
}

// ParseEvent extracts searchable text from a Copilot events.jsonl line.
// Only user.message and assistant.message events produce results; all others return nil.
func ParseEvent(line []byte) *ParsedMessage {
	if len(line) == 0 {
		return nil
	}

	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return nil
	}

	var role string
	switch envelope.Type {
	case "user.message":
		role = "user"
	case "assistant.message":
		role = "assistant"
	default:
		return nil
	}

	var raw struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil
	}

	if raw.Data.Content == "" {
		return nil
	}

	return &ParsedMessage{
		Role:      role,
		Text:      raw.Data.Content,
		ID:        raw.ID,
		Timestamp: raw.Timestamp,
	}
}
