package preview

import (
	"strings"
	"testing"

	"github.com/danielfollent/chatgrep/internal/provider"
)

func TestRender_BasicContext(t *testing.T) {
	msgs := []provider.PreviewMessage{
		{Role: "user", Text: "how do I fix this bug?", UUID: "msg-001", Timestamp: "2026-04-01T10:00:00.000Z"},
		{Role: "assistant", Text: "Check the null pointer on line 42.", UUID: "msg-002", Timestamp: "2026-04-01T10:00:05.000Z"},
		{Role: "user", Text: "thanks that worked", UUID: "msg-003", Timestamp: "2026-04-01T10:01:00.000Z"},
	}

	output := Render(msgs, "msg-002")

	if !strings.Contains(output, "how do I fix this bug?") {
		t.Error("missing context message before target")
	}
	if !strings.Contains(output, "Check the null pointer") {
		t.Error("missing target message")
	}
	if !strings.Contains(output, "thanks that worked") {
		t.Error("missing context message after target")
	}
}

func TestRender_HighlightsTarget(t *testing.T) {
	msgs := []provider.PreviewMessage{
		{Role: "user", Text: "question", UUID: "msg-001", Timestamp: "2026-04-01T10:00:00.000Z"},
		{Role: "assistant", Text: "answer", UUID: "msg-002", Timestamp: "2026-04-01T10:00:05.000Z"},
	}

	output := Render(msgs, "msg-002")

	// Target message block should have the >>> highlight marker on the role line
	if !strings.Contains(output, ">>>") {
		t.Errorf("target message not highlighted with >>>, output:\n%s", output)
	}
	// The >>> should appear near "Assistant" (the target role)
	idx := strings.Index(output, ">>>")
	after := output[idx:]
	if !strings.Contains(after[:80], "Assistant") {
		t.Errorf(">>> marker not adjacent to target role label, output:\n%s", output)
	}
}

func TestRender_SingleMessage(t *testing.T) {
	msgs := []provider.PreviewMessage{
		{Role: "user", Text: "only message", UUID: "msg-001", Timestamp: "2026-04-01T10:00:00.000Z"},
	}

	output := Render(msgs, "msg-001")
	if !strings.Contains(output, "only message") {
		t.Error("missing the single message")
	}
}

func TestRender_EmptyMessages(t *testing.T) {
	output := Render(nil, "msg-001")
	if output != "" {
		t.Errorf("expected empty output for nil messages, got %q", output)
	}
}

func TestRender_TargetNotFound(t *testing.T) {
	msgs := []provider.PreviewMessage{
		{Role: "user", Text: "some text", UUID: "msg-001", Timestamp: "2026-04-01T10:00:00.000Z"},
	}

	// Should still render all messages, just none highlighted
	output := Render(msgs, "msg-nonexistent")
	if !strings.Contains(output, "some text") {
		t.Error("should still render messages even if target not found")
	}
}

func TestRender_RoleColors(t *testing.T) {
	msgs := []provider.PreviewMessage{
		{Role: "user", Text: "user message", UUID: "msg-001", Timestamp: "2026-04-01T10:00:00.000Z"},
		{Role: "assistant", Text: "assistant message", UUID: "msg-002", Timestamp: "2026-04-01T10:00:05.000Z"},
	}

	output := Render(msgs, "msg-001")

	// Should contain ANSI codes (we check for ESC character)
	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI color codes in output")
	}
}
