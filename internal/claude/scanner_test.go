package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	// testdata/claude_sessions/ mimics ~/.claude/projects/<encoded-path>/
	dir := filepath.Join("..", "..", "testdata", "claude_sessions")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("testdata dir missing: %v", err)
	}
	return abs
}

func TestDiscoverSessions(t *testing.T) {
	dir := testdataDir(t)
	sessions, err := DiscoverSessions(dir)
	if err != nil {
		t.Fatalf("DiscoverSessions: %v", err)
	}

	// Should find sess-001.jsonl, sess-002.jsonl, sess-003.jsonl
	// Should NOT find sess-002/subagent-abc.jsonl (subdirectory)
	if len(sessions) != 3 {
		t.Errorf("got %d sessions, want 3", len(sessions))
		for _, s := range sessions {
			t.Logf("  found: %s", s.FilePath)
		}
	}
}

func TestDiscoverSessions_SkipsSubagentDirs(t *testing.T) {
	dir := testdataDir(t)
	sessions, err := DiscoverSessions(dir)
	if err != nil {
		t.Fatalf("DiscoverSessions: %v", err)
	}

	for _, s := range sessions {
		if strings.Contains(s.FilePath, "subagent") {
			t.Errorf("should skip subagent file, found: %s", s.FilePath)
		}
	}
}

func TestDiscoverSessions_ExtractsSessionID(t *testing.T) {
	dir := testdataDir(t)
	sessions, err := DiscoverSessions(dir)
	if err != nil {
		t.Fatalf("DiscoverSessions: %v", err)
	}

	ids := make(map[string]bool)
	for _, s := range sessions {
		ids[s.ID] = true
	}
	for _, want := range []string{"sess-001", "sess-002", "sess-003"} {
		if !ids[want] {
			t.Errorf("missing session ID %q", want)
		}
	}
}

func TestScanSession_FindsMatch(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-001.jsonl")

	results, err := ScanSession(path, "null pointer", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Role != "assistant" {
		t.Errorf("role = %q, want %q", results[0].Role, "assistant")
	}
	if results[0].UUID != "msg-002" {
		t.Errorf("uuid = %q, want %q", results[0].UUID, "msg-002")
	}
}

func TestScanSession_CaseInsensitive(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-001.jsonl")

	results, err := ScanSession(path, "NULL POINTER", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (case-insensitive)", len(results))
	}
}

func TestScanSession_MultipleMatches(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-001.jsonl")

	// "fix" appears in user msg "how do I fix this bug?"
	// No other messages contain "fix"
	results, err := ScanSession(path, "fix", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestScanSession_MaxPerSession(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-001.jsonl")

	// All 3 messages match a broad query - but limit to 1
	results, err := ScanSession(path, "a]e]i]o]u", 1) // won't match anything
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	// This shouldn't match, just testing the max cap doesn't crash
	if len(results) > 1 {
		t.Errorf("got %d results, want at most 1", len(results))
	}

	// Use a broad query that matches multiple lines
	results, err = ScanSession(path, "th", 1) // matches "this", "the", "that", "thanks"
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (max-per-session)", len(results))
	}
}

func TestScanSession_NoMatches(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-001.jsonl")

	results, err := ScanSession(path, "xyznonexistent", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestScanSession_EmptySession(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-003.jsonl")

	results, err := ScanSession(path, "anything", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0 for empty session", len(results))
	}
}

func TestScanSession_SnippetFlattensNewlines(t *testing.T) {
	dir := t.TempDir()
	line := `{"type":"user","message":{"role":"user","content":"line one\nline two\nline three"},"uuid":"msg-nl","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-nl","cwd":"/tmp","isSidechain":false}`
	path := filepath.Join(dir, "newlines.jsonl")
	if err := os.WriteFile(path, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "line", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if strings.Contains(results[0].Snippet, "\n") {
		t.Errorf("snippet contains newline: %q", results[0].Snippet)
	}
}

func TestScanSession_SnippetCentersOnMatch(t *testing.T) {
	dir := t.TempDir()
	// "target" appears far into the message, past the 200-char snippet window
	prefix := strings.Repeat("padding ", 50) // 400 chars
	text := prefix + "target word here" + strings.Repeat(" filler", 50)
	line := `{"type":"user","message":{"role":"user","content":"` + text + `"},"uuid":"msg-center","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-center","cwd":"/tmp","isSidechain":false}`
	path := filepath.Join(dir, "center.jsonl")
	if err := os.WriteFile(path, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "target", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !strings.Contains(results[0].Snippet, "target") {
		t.Errorf("snippet should contain the matched term 'target': %q", results[0].Snippet)
	}
}

func TestScanSession_SnippetTruncation(t *testing.T) {
	// Create a temp file with a very long message
	dir := t.TempDir()
	longText := strings.Repeat("word ", 200) // 1000 chars
	line := `{"type":"user","message":{"role":"user","content":"` + longText + `"},"uuid":"msg-long","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-long","cwd":"/tmp","isSidechain":false}`
	path := filepath.Join(dir, "long.jsonl")
	if err := os.WriteFile(path, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "word", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if len(results[0].Snippet) > MaxSnippetLen+10 { // small tolerance for rune boundary
		t.Errorf("snippet length %d exceeds max %d", len(results[0].Snippet), MaxSnippetLen)
	}
}

func TestScanSession_ExtractsSlug(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-001.jsonl")

	results, err := ScanSession(path, "bug", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Slug != "friendly-red-fox" {
		t.Errorf("slug = %q, want %q", results[0].Slug, "friendly-red-fox")
	}
}

func TestScanSession_SnippetStripsANSI(t *testing.T) {
	dir := t.TempDir()
	// \u001b[35m is magenta, \u001b[0m is reset - these are JSON-encoded ANSI escapes
	line := `{"type":"user","message":{"role":"user","content":"before \u001b[35mpurple text\u001b[0m after"},"uuid":"msg-ansi","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-ansi","cwd":"/tmp","isSidechain":false}`
	path := filepath.Join(dir, "ansi.jsonl")
	if err := os.WriteFile(path, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "purple", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if strings.Contains(results[0].Snippet, "\x1b") {
		t.Errorf("snippet contains ANSI escape: %q", results[0].Snippet)
	}
}

func TestSearchAll(t *testing.T) {
	dir := testdataDir(t)
	results, err := SearchAll(dir, "deploy", 4)
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	// Both user ("deploy the service") and assistant ("Running deploy script") match
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, r := range results {
		if r.SessionID != "sess-002" {
			t.Errorf("sessionId = %q, want %q", r.SessionID, "sess-002")
		}
	}
}

func TestSearchAll_EmptyQuery(t *testing.T) {
	dir := testdataDir(t)
	// Empty query should return all messages
	results, err := SearchAll(dir, "", 4)
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	// sess-001 has 3 messages, sess-002 has 2, sess-003 has 0
	if len(results) != 5 {
		t.Errorf("got %d results, want 5 for empty query", len(results))
	}
}
