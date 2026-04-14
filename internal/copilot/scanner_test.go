package copilot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielfollent/chatgrep/internal/text"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "copilot_sessions")
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
	// sess-c01, sess-c02, sess-c03
	if len(sessions) != 3 {
		t.Errorf("got %d sessions, want 3", len(sessions))
		for _, s := range sessions {
			t.Logf("  found: %s", s.ID)
		}
	}
}

func TestDiscoverSessions_ExtractsID(t *testing.T) {
	dir := testdataDir(t)
	sessions, err := DiscoverSessions(dir)
	if err != nil {
		t.Fatalf("DiscoverSessions: %v", err)
	}
	ids := make(map[string]bool)
	for _, s := range sessions {
		ids[s.ID] = true
	}
	for _, want := range []string{"sess-c01", "sess-c02", "sess-c03"} {
		if !ids[want] {
			t.Errorf("missing session ID %q", want)
		}
	}
}

func TestReadWorkspace(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c01", "workspace.yaml")

	ws, err := ReadWorkspace(path)
	if err != nil {
		t.Fatalf("ReadWorkspace: %v", err)
	}
	if ws.CWD != "/home/user/project" {
		t.Errorf("cwd = %q, want %q", ws.CWD, "/home/user/project")
	}
	if ws.Branch != "main" {
		t.Errorf("branch = %q, want %q", ws.Branch, "main")
	}
	if ws.Summary != "fix the login bug" {
		t.Errorf("summary = %q, want %q", ws.Summary, "fix the login bug")
	}
}

func TestReadWorkspace_NoSummary(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c03", "workspace.yaml")

	ws, err := ReadWorkspace(path)
	if err != nil {
		t.Fatalf("ReadWorkspace: %v", err)
	}
	if ws.Summary != "" {
		t.Errorf("summary = %q, want empty", ws.Summary)
	}
	if ws.CWD != "/home/user/empty" {
		t.Errorf("cwd = %q, want %q", ws.CWD, "/home/user/empty")
	}
}

func TestScanSession_FindsMatch(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c01", "events.jsonl")

	results, err := ScanSession(path, "session token", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Role != "assistant" {
		t.Errorf("role = %q, want %q", results[0].Role, "assistant")
	}
}

func TestScanSession_CaseInsensitive(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c01", "events.jsonl")

	results, err := ScanSession(path, "LOGIN BUG", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestScanSession_SkipsToolEvents(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c02", "events.jsonl")

	// "kubectl" appears in tool events, not in user/assistant messages
	results, err := ScanSession(path, "kubectl", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0 (tool events should be skipped)", len(results))
	}
}

func TestScanSession_EmptySession(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c03", "events.jsonl")

	results, err := ScanSession(path, "anything", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestScanSession_MaxPerSession(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sess-c01", "events.jsonl")

	// "the" appears in multiple messages, but limit to 1
	results, err := ScanSession(path, "the", 1)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (max-per-session)", len(results))
	}
}

func TestScanSession_SnippetFlattensNewlines(t *testing.T) {
	dir := t.TempDir()
	line := `{"type":"user.message","data":{"content":"line one\nline two\nline three"},"id":"evt-nl","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`
	path := filepath.Join(dir, "events.jsonl")
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
	prefix := strings.Repeat("padding ", 50) // 400 chars
	text := prefix + "target word here" + strings.Repeat(" filler", 50)
	line := `{"type":"user.message","data":{"content":"` + text + `"},"id":"evt-center","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`
	path := filepath.Join(dir, "events.jsonl")
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
	dir := t.TempDir()
	longText := strings.Repeat("word ", 200)
	line := `{"type":"user.message","data":{"content":"` + longText + `"},"id":"evt-long","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`
	path := filepath.Join(dir, "events.jsonl")
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
	if len(results[0].Snippet) > text.MaxSnippetLen+10 {
		t.Errorf("snippet length %d exceeds max %d", len(results[0].Snippet), text.MaxSnippetLen)
	}
}

func TestScanSession_SnippetStripsANSI(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, "sess-ansi")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}
	// \u001b[35m is magenta, \u001b[0m is reset
	line := `{"type":"user.message","data":{"content":"before \u001b[35mpurple text\u001b[0m after"},"id":"evt-ansi","timestamp":"2026-04-01T10:00:00.000Z","parentId":null}`
	path := filepath.Join(sessDir, "events.jsonl")
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
	results, err := SearchAll(dir, "staging", 4)
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	// "staging" appears in sess-c02 user msg and assistant msg
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}
