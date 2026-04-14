package fzf

import (
	"testing"
	"time"

	"github.com/danielfollent/chatgrep/internal/provider"
)

func TestFormatLine_ClaudeWithTimestamp(t *testing.T) {
	ts := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	m := provider.Match{
		ProviderName: "claude",
		SessionID:    "sess-001",
		UUID:         "msg-002",
		Role:         "assistant",
		Snippet:      "You need to check the null pointer.",
		Timestamp:    ts,
		Slug:         "friendly-red-fox",
	}

	line := FormatLine(m)
	parts := splitTabs(line)
	if len(parts) < 3 {
		t.Fatalf("expected at least 3 tab-delimited fields, got %d: %q", len(parts), line)
	}
	if parts[0] != "claude:sess-001" {
		t.Errorf("field[0] = %q, want %q", parts[0], "claude:sess-001")
	}
	if parts[1] != "msg-002" {
		t.Errorf("field[1] = %q, want %q", parts[1], "msg-002")
	}
	display := parts[2]
	// Should contain provider name, relative time, role tag, and snippet
	if !containsAll(display, "claude", "ago", "A>", "null pointer") {
		t.Errorf("display missing expected content: %q", display)
	}
	// Should NOT contain the slug
	if contains(display, "friendly-red-fox") {
		t.Errorf("display should not contain slug: %q", display)
	}
}

func TestFormatLine_CopilotProvider(t *testing.T) {
	ts := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	m := provider.Match{
		ProviderName: "copilot",
		SessionID:    "sess-c01",
		UUID:         "evt-002",
		Role:         "user",
		Snippet:      "fix the login bug",
		Timestamp:    ts,
	}

	line := FormatLine(m)
	parts := splitTabs(line)
	if parts[0] != "copilot:sess-c01" {
		t.Errorf("field[0] = %q, want %q", parts[0], "copilot:sess-c01")
	}
	display := parts[2]
	if !containsAll(display, "copilot", "ago", "U>", "fix the login") {
		t.Errorf("display missing expected content: %q", display)
	}
}

func TestFormatLine_OldTimestamp(t *testing.T) {
	// 3 days ago should show "3d ago"
	ts := time.Now().Add(-3 * 24 * time.Hour).UTC().Format(time.RFC3339)
	m := provider.Match{
		ProviderName: "claude",
		SessionID:    "sess-001",
		UUID:         "msg-010",
		Role:         "user",
		Snippet:      "deploy the service",
		Timestamp:    ts,
	}

	line := FormatLine(m)
	parts := splitTabs(line)
	display := parts[2]
	if !containsAll(display, "3d ago") {
		t.Errorf("expected '3d ago' in display: %q", display)
	}
}

func TestFormatLine_FallbackTimestamp(t *testing.T) {
	// Malformed timestamp should not crash
	m := provider.Match{
		ProviderName: "claude",
		SessionID:    "sess-001",
		UUID:         "msg-010",
		Role:         "user",
		Snippet:      "deploy the service",
		Timestamp:    "not-a-timestamp",
	}

	line := FormatLine(m)
	parts := splitTabs(line)
	if len(parts) < 3 {
		t.Fatalf("expected at least 3 fields, got %d", len(parts))
	}
}

func TestParseSelection(t *testing.T) {
	line := "claude:sess-001\tmsg-002\tclaude  2h ago A> You need to check..."

	sel, err := ParseSelection(line)
	if err != nil {
		t.Fatalf("ParseSelection: %v", err)
	}
	if sel.ProviderName != "claude" {
		t.Errorf("provider = %q, want %q", sel.ProviderName, "claude")
	}
	if sel.SessionID != "sess-001" {
		t.Errorf("sessionID = %q, want %q", sel.SessionID, "sess-001")
	}
	if sel.MsgUUID != "msg-002" {
		t.Errorf("msgUUID = %q, want %q", sel.MsgUUID, "msg-002")
	}
}

func TestParseSelection_Copilot(t *testing.T) {
	line := "copilot:sess-c01\tevt-002\tcopilot 30m ago U> fix the login bug"

	sel, err := ParseSelection(line)
	if err != nil {
		t.Fatalf("ParseSelection: %v", err)
	}
	if sel.ProviderName != "copilot" {
		t.Errorf("provider = %q, want %q", sel.ProviderName, "copilot")
	}
	if sel.SessionID != "sess-c01" {
		t.Errorf("sessionID = %q, want %q", sel.SessionID, "sess-c01")
	}
}

func TestParseSelection_InvalidLine(t *testing.T) {
	_, err := ParseSelection("no tabs here")
	if err == nil {
		t.Error("expected error for invalid line")
	}
}

func TestParseSelection_MissingProvider(t *testing.T) {
	_, err := ParseSelection("sess-001\tmsg-002\tsome text")
	if err == nil {
		t.Error("expected error when provider:session format missing colon")
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{30 * time.Second, "now"},
		{5 * time.Minute, "5m ago"},
		{90 * time.Minute, "1h ago"},
		{3 * time.Hour, "3h ago"},
		{36 * time.Hour, "1d ago"},
		{7 * 24 * time.Hour, "7d ago"},
		{45 * 24 * time.Hour, "45d ago"},
	}

	for _, tt := range tests {
		ts := time.Now().Add(-tt.dur)
		got := relativeTime(ts)
		if got != tt.want {
			t.Errorf("relativeTime(-%v) = %q, want %q", tt.dur, got, tt.want)
		}
	}
}

func TestFormatPlainLine(t *testing.T) {
	m := provider.Match{
		ProviderName: "claude",
		SessionID:    "sess-001",
		UUID:         "msg-002",
		Role:         "assistant",
		Snippet:      "Check the null pointer",
		Timestamp:    "2026-04-01T10:00:05.000Z",
	}

	line := FormatPlainLine(m)
	// Format: provider:sessionID\ttimestamp\trole\tsnippet
	parts := splitTabs(line)
	if len(parts) != 4 {
		t.Fatalf("got %d fields, want 4: %q", len(parts), line)
	}
	if parts[0] != "claude:sess-001" {
		t.Errorf("field[0] = %q, want %q", parts[0], "claude:sess-001")
	}
	if parts[1] != "2026-04-01T10:00:05.000Z" {
		t.Errorf("field[1] = %q, want timestamp", parts[1])
	}
	if parts[2] != "assistant" {
		t.Errorf("field[2] = %q, want %q", parts[2], "assistant")
	}
	if parts[3] != "Check the null pointer" {
		t.Errorf("field[3] = %q, want snippet", parts[3])
	}
}

func TestFormatLine_ColorClaude(t *testing.T) {
	ts := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	m := provider.Match{
		ProviderName: "claude",
		SessionID:    "sess-001",
		UUID:         "msg-002",
		Role:         "assistant",
		Snippet:      "check the pointer",
		Timestamp:    ts,
	}

	line := FormatLine(m)
	display := splitTabs(line)[2]

	// Provider "claude" wrapped in cyan
	if !contains(display, "\033[36m") {
		t.Errorf("expected cyan ANSI for claude provider in: %q", display)
	}
	// Timestamp wrapped in dim
	if !contains(display, "\033[2m") {
		t.Errorf("expected dim ANSI for timestamp in: %q", display)
	}
	// Assistant role wrapped in blue
	if !contains(display, "\033[34m") {
		t.Errorf("expected blue ANSI for assistant role in: %q", display)
	}
	// Snippet should NOT contain ANSI codes
	snippetIdx := indexStr(display, "check the pointer")
	if snippetIdx < 0 {
		t.Fatalf("snippet not found in display: %q", display)
	}
	snippetPart := display[snippetIdx:]
	if contains(snippetPart, "\033[") {
		t.Errorf("snippet should not contain ANSI codes: %q", snippetPart)
	}
}

func TestFormatLine_ColorCopilotUser(t *testing.T) {
	ts := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	m := provider.Match{
		ProviderName: "copilot",
		SessionID:    "sess-c01",
		UUID:         "evt-002",
		Role:         "user",
		Snippet:      "fix the bug",
		Timestamp:    ts,
	}

	line := FormatLine(m)
	display := splitTabs(line)[2]

	// Provider "copilot" wrapped in magenta
	if !contains(display, "\033[35m") {
		t.Errorf("expected magenta ANSI for copilot provider in: %q", display)
	}
	// User role wrapped in green
	if !contains(display, "\033[32m") {
		t.Errorf("expected green ANSI for user role in: %q", display)
	}
}

func TestFormatLine_ColorCodex(t *testing.T) {
	ts := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	m := provider.Match{
		ProviderName: "codex",
		SessionID:    "sess-x01",
		UUID:         "line-000123",
		Role:         "assistant",
		Snippet:      "recent codex reply",
		Timestamp:    ts,
	}

	line := FormatLine(m)
	display := splitTabs(line)[2]

	// Provider "codex" wrapped in yellow
	if !contains(display, "\033[33m") {
		t.Errorf("expected yellow ANSI for codex provider in: %q", display)
	}
	// Assistant role stays blue
	if !contains(display, "\033[34m") {
		t.Errorf("expected blue ANSI for assistant role in: %q", display)
	}
}

func TestFormatPlainLine_NoColor(t *testing.T) {
	m := provider.Match{
		ProviderName: "claude",
		SessionID:    "sess-001",
		UUID:         "msg-002",
		Role:         "assistant",
		Snippet:      "no color here",
		Timestamp:    "2026-04-01T10:00:05.000Z",
	}

	line := FormatPlainLine(m)
	if contains(line, "\033[") {
		t.Errorf("plain line should not contain ANSI codes: %q", line)
	}
}

// helpers

func indexStr(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// helpers (original)

func splitTabs(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
