package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "codex_sessions")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("testdata dir missing: %v", err)
	}
	return abs
}

func TestDefaultBaseDir_UsesCODEX_HOME(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/custom-codex")

	got := DefaultBaseDir()
	if got != "/tmp/custom-codex" {
		t.Errorf("got %q, want %q", got, "/tmp/custom-codex")
	}
}

func TestDefaultBaseDir_Fallback(t *testing.T) {
	t.Setenv("CODEX_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	got := DefaultBaseDir()
	want := filepath.Join(home, ".codex")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDiscoverSessions(t *testing.T) {
	dir := testdataDir(t)

	sessions, err := DiscoverSessions(dir)
	if err != nil {
		t.Fatalf("DiscoverSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	ids := make(map[string]bool)
	for _, s := range sessions {
		ids[s.ID] = true
		if s.CWD == "" {
			t.Errorf("session %q missing cwd", s.ID)
		}
	}

	for _, want := range []string{
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
	} {
		if !ids[want] {
			t.Errorf("missing session ID %q", want)
		}
	}
}

func TestScanSession_CaseInsensitive(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sessions", "2026", "04", "01", "rollout-2026-04-01T10-00-00-11111111-1111-1111-1111-111111111111.jsonl")

	results, err := ScanSession(path, "SESSION TOKEN", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Role != "assistant" {
		t.Errorf("role = %q, want %q", results[0].Role, "assistant")
	}
	if results[0].ID != "line-000007" {
		t.Errorf("id = %q, want %q", results[0].ID, "line-000007")
	}
}

func TestScanSession_MultiWordAND(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sessions", "2026", "04", "02", "rollout-2026-04-02T11-00-00-22222222-2222-2222-2222-222222222222.jsonl")

	results, err := ScanSession(path, "deploy staging", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestScanSession_SkipsInternalRecords(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sessions", "2026", "04", "01", "rollout-2026-04-01T10-00-00-11111111-1111-1111-1111-111111111111.jsonl")

	results, err := ScanSession(path, "checking the auth flow", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestScanSession_MalformedJSONIgnored(t *testing.T) {
	dir := testdataDir(t)
	path := filepath.Join(dir, "sessions", "2026", "04", "01", "rollout-2026-04-01T10-00-00-11111111-1111-1111-1111-111111111111.jsonl")

	results, err := ScanSession(path, "auth middleware", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ID != "line-000004" {
		t.Errorf("id = %q, want %q", results[0].ID, "line-000004")
	}
}

func TestScanSession_IncludesResponseItemOnlyUserMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	lines := `{"timestamp":"2026-04-14T07:19:04.000Z","type":"session_meta","payload":{"id":"44444444-4444-4444-4444-444444444444","timestamp":"2026-04-14T07:19:04.000Z","cwd":"/tmp","originator":"codex-cli","cli_version":"0.1.0","source":"cli","model_provider":"openai"}}
{"timestamp":"2026-04-14T07:19:15.328Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"recent follow-up question"}]}}
{"timestamp":"2026-04-14T07:19:15.329Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"duplicate user prompt"}]}}
{"timestamp":"2026-04-14T07:19:15.329Z","type":"event_msg","payload":{"type":"user_message","message":"duplicate user prompt"}}
{"timestamp":"2026-04-14T07:19:36.629Z","type":"event_msg","payload":{"type":"turn_aborted","reason":"interrupt"}}`
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanSession(path, "recent follow-up", 10)
	if err != nil {
		t.Fatalf("ScanSession: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Role != "user" {
		t.Errorf("role = %q, want %q", results[0].Role, "user")
	}
	if results[0].ID != "line-000002" {
		t.Errorf("id = %q, want %q", results[0].ID, "line-000002")
	}
}

func TestReadMessages_DedupesEventAndResponseEchoes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	lines := `{"timestamp":"2026-04-14T07:19:04.000Z","type":"session_meta","payload":{"id":"55555555-5555-5555-5555-555555555555","timestamp":"2026-04-14T07:19:04.000Z","cwd":"/tmp","originator":"codex-cli","cli_version":"0.1.0","source":"cli","model_provider":"openai"}}
{"timestamp":"2026-04-14T07:19:15.328Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"recent follow-up question"}]}}
{"timestamp":"2026-04-14T07:19:15.329Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"duplicate user prompt"}]}}
{"timestamp":"2026-04-14T07:19:15.329Z","type":"event_msg","payload":{"type":"user_message","message":"duplicate user prompt"}}
{"timestamp":"2026-04-14T07:19:39.873Z","type":"event_msg","payload":{"type":"agent_message","message":"assistant reply"}} 
{"timestamp":"2026-04-14T07:19:39.874Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"assistant reply"}]}}
{"timestamp":"2026-04-14T07:19:57.266Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"assistant only"}]}}`
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	msgs, err := ReadMessages(path)
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 4 {
		t.Fatalf("got %d messages, want 4", len(msgs))
	}
	wantIDs := []string{"line-000002", "line-000003", "line-000005", "line-000007"}
	for i, want := range wantIDs {
		if msgs[i].ID != want {
			t.Errorf("msgs[%d].ID = %q, want %q", i, msgs[i].ID, want)
		}
	}
}

func TestSearchAll(t *testing.T) {
	dir := testdataDir(t)

	results, err := SearchAll(dir, "deploy staging", 4)
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, r := range results {
		if r.SessionID != "22222222-2222-2222-2222-222222222222" {
			t.Errorf("sessionID = %q, want %q", r.SessionID, "22222222-2222-2222-2222-222222222222")
		}
	}
}
