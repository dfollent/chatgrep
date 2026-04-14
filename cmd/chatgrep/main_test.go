package main

import (
	"os"
	"testing"

	"github.com/danielfollent/chatgrep/internal/provider"
)

func TestResolveProviderNames_Claude(t *testing.T) {
	names, err := resolveProviderNames("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("got %v, want [claude]", names)
	}
}

func TestResolveProviderNames_Copilot(t *testing.T) {
	names, err := resolveProviderNames("copilot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "copilot" {
		t.Errorf("got %v, want [copilot]", names)
	}
}

func TestResolveProviderNames_Codex(t *testing.T) {
	names, err := resolveProviderNames("codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "codex" {
		t.Errorf("got %v, want [codex]", names)
	}
}

func TestResolveProviderNames_All(t *testing.T) {
	names, err := resolveProviderNames("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("got %d names, want 3", len(names))
	}
	has := map[string]bool{}
	for _, n := range names {
		has[n] = true
	}
	if !has["claude"] || !has["copilot"] || !has["codex"] {
		t.Errorf("got %v, want [claude copilot codex]", names)
	}
}

func TestResolveProviderNames_Invalid(t *testing.T) {
	_, err := resolveProviderNames("cursor")
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestParsePreviewTarget_Valid(t *testing.T) {
	prov, sess, err := parsePreviewTarget("claude:sess-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov != "claude" {
		t.Errorf("provider = %q, want %q", prov, "claude")
	}
	if sess != "sess-001" {
		t.Errorf("sessionID = %q, want %q", sess, "sess-001")
	}
}

func TestParsePreviewTarget_Copilot(t *testing.T) {
	prov, sess, err := parsePreviewTarget("copilot:sess-c01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov != "copilot" {
		t.Errorf("provider = %q, want %q", prov, "copilot")
	}
	if sess != "sess-c01" {
		t.Errorf("sessionID = %q, want %q", sess, "sess-c01")
	}
}

func TestParsePreviewTarget_Codex(t *testing.T) {
	prov, sess, err := parsePreviewTarget("codex:11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov != "codex" {
		t.Errorf("provider = %q, want %q", prov, "codex")
	}
	if sess != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("sessionID = %q, want %q", sess, "11111111-1111-1111-1111-111111111111")
	}
}

func TestParsePreviewTarget_NoColon(t *testing.T) {
	_, _, err := parsePreviewTarget("sess-001")
	if err == nil {
		t.Error("expected error for missing colon")
	}
}

func TestMakeProvider_CodexMissingSessionsDir(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())

	_, err := makeProvider("codex")
	if err == nil {
		t.Fatal("expected error for missing codex sessions dir")
	}
	if !contains(err.Error(), "codex sessions dir not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Empty query must reach the provider, not get rejected at dispatch.
// We verify by passing an invalid agent: if the error is about the agent,
// the empty query was accepted. If it says "usage", it was wrongly rejected.
func TestRunSearch_EmptyQueryNotRejected(t *testing.T) {
	err := runSearch("nonexistent", "", "", false)
	if err == nil {
		t.Fatal("expected error for invalid agent")
	}
	if contains(err.Error(), "usage") {
		t.Error("empty query should not be rejected; browse-all mode is the default UX")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestFilterByProject_PrefixMatch(t *testing.T) {
	matches := []provider.Match{
		{SessionID: "s1", CWD: "/home/user/project"},
		{SessionID: "s2", CWD: "/home/user/project/sub"},
		{SessionID: "s3", CWD: "/home/other/thing"},
	}
	got := filterByProject(matches, "/home/user/project")
	if len(got) != 2 {
		t.Errorf("got %d results, want 2", len(got))
	}
}

func TestFilterByProject_Empty(t *testing.T) {
	matches := []provider.Match{
		{SessionID: "s1", CWD: "/foo"},
	}
	got := filterByProject(matches, "")
	if len(got) != 1 {
		t.Errorf("empty filter should return all, got %d", len(got))
	}
}

func TestFilterByProject_NoMatch(t *testing.T) {
	matches := []provider.Match{
		{SessionID: "s1", CWD: "/home/user/project"},
	}
	got := filterByProject(matches, "/other")
	if len(got) != 0 {
		t.Errorf("got %d results, want 0", len(got))
	}
}

func TestResolveProjectFlag_Dot(t *testing.T) {
	resolved, err := resolveProjectFlag(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	if resolved != cwd {
		t.Errorf("got %q, want %q", resolved, cwd)
	}
}

func TestResolveProjectFlag_Passthrough(t *testing.T) {
	resolved, err := resolveProjectFlag("/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "/some/path" {
		t.Errorf("got %q, want %q", resolved, "/some/path")
	}
}

func TestResolveProjectFlag_Empty(t *testing.T) {
	resolved, err := resolveProjectFlag("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "" {
		t.Errorf("got %q, want empty", resolved)
	}
}

func TestBuildPreviewCmd(t *testing.T) {
	cmd := buildPreviewCmd("/usr/local/bin/chatgrep")
	want := "/usr/local/bin/chatgrep --preview {1} {2}"
	if cmd != want {
		t.Errorf("got %q, want %q", cmd, want)
	}
}
