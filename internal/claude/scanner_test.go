package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielfollent/chatgrep/internal/text"
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
	if len(results[0].Snippet) > text.MaxSnippetLen+10 { // small tolerance for rune boundary
		t.Errorf("snippet length %d exceeds max %d", len(results[0].Snippet), text.MaxSnippetLen)
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

func TestScanSession_MultiWordAND(t *testing.T) {
	dir := t.TempDir()
	lines := `{"type":"user","message":{"role":"user","content":"deploy to staging server"},"uuid":"msg-mw1","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-mw","cwd":"/tmp","isSidechain":false}
{"type":"user","message":{"role":"user","content":"deploy to production"},"uuid":"msg-mw2","timestamp":"2026-04-01T10:00:01.000Z","sessionId":"sess-mw","cwd":"/tmp","isSidechain":false}
{"type":"user","message":{"role":"user","content":"staging is down"},"uuid":"msg-mw3","timestamp":"2026-04-01T10:00:02.000Z","sessionId":"sess-mw","cwd":"/tmp","isSidechain":false}
`
	path := filepath.Join(dir, "multiword.jsonl")
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	// "deploy staging" should match only the message with both words
	results, err := ScanSession(path, "deploy staging", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].UUID != "msg-mw1" {
		t.Errorf("uuid = %q, want msg-mw1", results[0].UUID)
	}
}

func TestScanSession_SingleWordUnchanged(t *testing.T) {
	dir := t.TempDir()
	lines := `{"type":"user","message":{"role":"user","content":"deploy to staging"},"uuid":"msg-sw1","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-sw","cwd":"/tmp","isSidechain":false}
`
	path := filepath.Join(dir, "singleword.jsonl")
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "deploy", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestScanSession_SkipsSidechain(t *testing.T) {
	dir := t.TempDir()
	lines := `{"type":"user","message":{"role":"user","content":"real message"},"uuid":"msg-a","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-sc","cwd":"/tmp","isSidechain":false}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"subagent noise"}]},"uuid":"msg-b","timestamp":"2026-04-01T10:00:01.000Z","sessionId":"sess-sc","cwd":"/tmp","isSidechain":true}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"real reply"}]},"uuid":"msg-c","timestamp":"2026-04-01T10:00:02.000Z","sessionId":"sess-sc","cwd":"/tmp","isSidechain":false}
`
	path := filepath.Join(dir, "sidechain.jsonl")
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2 (sidechain excluded)", len(results))
	}
	for _, r := range results {
		if r.UUID == "msg-b" {
			t.Errorf("sidechain message msg-b should be excluded")
		}
	}
}

func TestScanSession_LargeLineDoesNotFail(t *testing.T) {
	dir := t.TempDir()
	// 2MB message content - exceeds the old 1MB buffer
	bigContent := strings.Repeat("x", 2*1024*1024)
	lines := `{"type":"user","message":{"role":"user","content":"` + bigContent + `"},"uuid":"msg-big","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-big","cwd":"/tmp","isSidechain":false}` + "\n"
	lines += `{"type":"user","message":{"role":"user","content":"small findable message"},"uuid":"msg-small","timestamp":"2026-04-01T10:00:01.000Z","sessionId":"sess-big","cwd":"/tmp","isSidechain":false}` + "\n"
	path := filepath.Join(dir, "big.jsonl")
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "findable", 10)
	if err != nil {
		t.Fatalf("ScanSession should not fail on large lines: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestSearchAll_SkipsBadSessions(t *testing.T) {
	dir := t.TempDir()

	// One good session
	good := `{"type":"user","message":{"role":"user","content":"good message"},"uuid":"msg-g","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-good","cwd":"/tmp","isSidechain":false}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "good.jsonl"), []byte(good), 0644); err != nil {
		t.Fatal(err)
	}

	// One session with a line exceeding 10MB (will fail even with bumped buffer)
	hugeLine := `{"type":"user","message":{"role":"user","content":"` + strings.Repeat("x", 11*1024*1024) + `"},"uuid":"msg-h","timestamp":"2026-04-01T10:00:00.000Z","sessionId":"sess-huge","cwd":"/tmp","isSidechain":false}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "huge.jsonl"), []byte(hugeLine), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := SearchAll(dir, "good", 2)
	if err != nil {
		t.Fatalf("SearchAll should not fail when one session has errors: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 from the good session", len(results))
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
