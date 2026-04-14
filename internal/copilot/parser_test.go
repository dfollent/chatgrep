package copilot

import (
	"testing"
)

func TestParseEvent_UserMessage(t *testing.T) {
	line := []byte(`{"type":"user.message","data":{"content":"fix the login bug"},"id":"evt-002","timestamp":"2026-04-01T10:00:05.000Z","parentId":"evt-001"}`)

	msg := ParseEvent(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}
	if msg.Text != "fix the login bug" {
		t.Errorf("text = %q, want %q", msg.Text, "fix the login bug")
	}
	if msg.ID != "evt-002" {
		t.Errorf("id = %q, want %q", msg.ID, "evt-002")
	}
	if msg.Timestamp != "2026-04-01T10:00:05.000Z" {
		t.Errorf("timestamp = %q, want %q", msg.Timestamp, "2026-04-01T10:00:05.000Z")
	}
}

func TestParseEvent_AssistantMessage(t *testing.T) {
	line := []byte(`{"type":"assistant.message","data":{"content":"I found the issue in the auth middleware."},"id":"evt-004","timestamp":"2026-04-01T10:00:10.000Z","parentId":"evt-003"}`)

	msg := ParseEvent(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Text != "I found the issue in the auth middleware." {
		t.Errorf("text = %q, want %q", msg.Text, "I found the issue in the auth middleware.")
	}
}

func TestParseEvent_ToolEventsSkipped(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"tool_start", `{"type":"tool.execution_start","data":{"toolName":"bash","input":"ls"},"id":"evt-013","timestamp":"2026-04-02T09:00:07.000Z","parentId":"evt-012"}`},
		{"tool_complete", `{"type":"tool.execution_complete","data":{"toolName":"bash","output":"done"},"id":"evt-014","timestamp":"2026-04-02T09:00:10.000Z","parentId":"evt-013"}`},
		{"turn_start", `{"type":"assistant.turn_start","data":{},"id":"evt-003","timestamp":"2026-04-01T10:00:06.000Z","parentId":"evt-002"}`},
		{"turn_end", `{"type":"assistant.turn_end","data":{},"id":"evt-005","timestamp":"2026-04-01T10:00:11.000Z","parentId":"evt-004"}`},
		{"model_change", `{"type":"session.model_change","data":{"previousModel":"gpt-5.3","newModel":"gpt-5.3"},"id":"evt-100","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`},
		{"session_start", `{"type":"session.start","data":{"sessionId":"sess-c01"},"id":"evt-001","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`},
		{"session_shutdown", `{"type":"session.shutdown","data":{},"id":"evt-021","timestamp":"2026-04-03T08:00:01.000Z","parentId":"evt-020"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := ParseEvent([]byte(tc.line))
			if msg != nil {
				t.Errorf("expected nil for type %s, got text=%q", tc.name, msg.Text)
			}
		})
	}
}

func TestParseEvent_MalformedJSON(t *testing.T) {
	msg := ParseEvent([]byte(`{not valid`))
	if msg != nil {
		t.Error("expected nil for malformed JSON")
	}
}

func TestParseEvent_EmptyLine(t *testing.T) {
	msg := ParseEvent([]byte(""))
	if msg != nil {
		t.Error("expected nil for empty line")
	}
}

func TestParseEvent_EmptyContent(t *testing.T) {
	line := []byte(`{"type":"user.message","data":{"content":""},"id":"evt-099","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`)
	msg := ParseEvent(line)
	if msg != nil {
		t.Error("expected nil for empty content")
	}
}
