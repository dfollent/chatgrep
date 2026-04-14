package main

import (
	"testing"
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

func TestResolveProviderNames_All(t *testing.T) {
	names, err := resolveProviderNames("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d names, want 2", len(names))
	}
	has := map[string]bool{}
	for _, n := range names {
		has[n] = true
	}
	if !has["claude"] || !has["copilot"] {
		t.Errorf("got %v, want [claude copilot]", names)
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

func TestParsePreviewTarget_NoColon(t *testing.T) {
	_, _, err := parsePreviewTarget("sess-001")
	if err == nil {
		t.Error("expected error for missing colon")
	}
}

// Empty query must reach the provider, not get rejected at dispatch.
// We verify by passing an invalid agent: if the error is about the agent,
// the empty query was accepted. If it says "usage", it was wrongly rejected.
func TestRunSearch_EmptyQueryNotRejected(t *testing.T) {
	err := runSearch("nonexistent", "")
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

func TestBuildPreviewCmd(t *testing.T) {
	cmd := buildPreviewCmd("/usr/local/bin/chatgrep")
	want := "/usr/local/bin/chatgrep --preview {1} {2}"
	if cmd != want {
		t.Errorf("got %q, want %q", cmd, want)
	}
}
