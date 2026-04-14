package claude

import (
	"testing"
)

func TestParseLine_UserStringContent(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"how do I fix this bug?"},"uuid":"msg-001","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-001","cwd":"/home/user/project","slug":"friendly-red-fox","isSidechain":false}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}
	if msg.Text != "how do I fix this bug?" {
		t.Errorf("text = %q, want %q", msg.Text, "how do I fix this bug?")
	}
	if msg.UUID != "msg-001" {
		t.Errorf("uuid = %q, want %q", msg.UUID, "msg-001")
	}
	if msg.Timestamp != "2026-04-01T10:00:00.000Z" {
		t.Errorf("timestamp = %q, want %q", msg.Timestamp, "2026-04-01T10:00:00.000Z")
	}
	if msg.SessionID != "sess-001" {
		t.Errorf("sessionId = %q, want %q", msg.SessionID, "sess-001")
	}
	if msg.CWD != "/home/user/project" {
		t.Errorf("cwd = %q, want %q", msg.CWD, "/home/user/project")
	}
	if msg.Slug != "friendly-red-fox" {
		t.Errorf("slug = %q, want %q", msg.Slug, "friendly-red-fox")
	}
	if msg.IsSidechain {
		t.Error("expected isSidechain=false")
	}
}

func TestParseLine_AssistantArrayContent(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"You need to check the null pointer on line 42."}]},"uuid":"msg-002","timestamp":"2026-04-01T10:00:05.000Z","sessionId":"sess-001","cwd":"/home/user/project","isSidechain":false}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Text != "You need to check the null pointer on line 42." {
		t.Errorf("text = %q, want %q", msg.Text, "You need to check the null pointer on line 42.")
	}
}

func TestParseLine_UserArrayContent(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"here is my code"},{"type":"text","text":"please review it"}]},"uuid":"msg-050","timestamp":"2026-04-06T14:00:00.000Z","sessionId":"sess-006","cwd":"/home/user/review","isSidechain":false}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Text != "here is my code please review it" {
		t.Errorf("text = %q, want %q", msg.Text, "here is my code please review it")
	}
}

func TestParseLine_ToolResultOnly(t *testing.T) {
	// User message containing only tool_result blocks should be skipped
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":[{"type":"text","text":"file contents here"}]}]},"uuid":"msg-020","timestamp":"2026-04-03T08:00:00.000Z","sessionId":"sess-003","cwd":"/home/user/project","isSidechain":false}`)

	msg := ParseLine(line)
	if msg != nil {
		t.Errorf("expected nil for tool_result-only message, got text=%q", msg.Text)
	}
}

func TestParseLine_ThinkingOnly(t *testing.T) {
	// Assistant message with only thinking blocks should be skipped
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"Let me think about this carefully..."}]},"uuid":"msg-040","timestamp":"2026-04-05T11:00:00.000Z","sessionId":"sess-005","cwd":"/home/user/project","isSidechain":false}`)

	msg := ParseLine(line)
	if msg != nil {
		t.Errorf("expected nil for thinking-only message, got text=%q", msg.Text)
	}
}

func TestParseLine_AssistantMixedContent(t *testing.T) {
	// Assistant with tool_use + text should extract only text
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_456","name":"Read","input":{"path":"/tmp/foo"}},{"type":"text","text":"I found the config file."}]},"uuid":"msg-021","timestamp":"2026-04-03T08:00:02.000Z","sessionId":"sess-003","cwd":"/home/user/project","isSidechain":false}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Text != "I found the config file." {
		t.Errorf("text = %q, want %q", msg.Text, "I found the config file.")
	}
}

func TestParseLine_SlugExtraction(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"test"},"uuid":"msg-001","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-001","slug":"friendly-red-fox","isSidechain":false}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Slug != "friendly-red-fox" {
		t.Errorf("slug = %q, want %q", msg.Slug, "friendly-red-fox")
	}
}

func TestParseLine_NoSlug(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"test"},"uuid":"msg-001","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-001","isSidechain":false}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Slug != "" {
		t.Errorf("slug = %q, want empty", msg.Slug)
	}
}

func TestParseLine_MalformedJSON(t *testing.T) {
	line := []byte(`{not valid json`)
	msg := ParseLine(line)
	if msg != nil {
		t.Error("expected nil for malformed JSON")
	}
}

func TestParseLine_EmptyLine(t *testing.T) {
	msg := ParseLine([]byte(""))
	if msg != nil {
		t.Error("expected nil for empty line")
	}
}

func TestParseLine_SkipsNonMessageTypes(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"file-history-snapshot", `{"type":"file-history-snapshot","messageId":"snap-001"}`},
		{"progress", `{"type":"progress","message":"Processing...","uuid":"msg-030"}`},
		{"system", `{"type":"system","message":{"role":"system","content":"system prompt"},"uuid":"msg-099"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := ParseLine([]byte(tc.line))
			if msg != nil {
				t.Errorf("expected nil for type %s", tc.name)
			}
		})
	}
}

func TestParseLine_Sidechain(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"sidechain response"}]},"uuid":"msg-061","timestamp":"2026-04-07T10:00:05.000Z","sessionId":"sess-007","isSidechain":true}`)

	msg := ParseLine(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if !msg.IsSidechain {
		t.Error("expected isSidechain=true")
	}
	if msg.Text != "sidechain response" {
		t.Errorf("text = %q, want %q", msg.Text, "sidechain response")
	}
}
