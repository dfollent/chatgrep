package codex

import (
	"path/filepath"
	"testing"
)

func TestProviderDiscover(t *testing.T) {
	p := NewProvider(testdataDir(t))

	sessions, err := p.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	found := false
	for _, s := range sessions {
		if s.ID != "11111111-1111-1111-1111-111111111111" {
			continue
		}
		found = true
		if s.CWD != "/home/user/project" {
			t.Errorf("cwd = %q, want %q", s.CWD, "/home/user/project")
		}
		if s.Slug != "fix the login bug" {
			t.Errorf("slug = %q, want %q", s.Slug, "fix the login bug")
		}
	}
	if !found {
		t.Fatal("expected to discover session 11111111-1111-1111-1111-111111111111")
	}
}

func TestProviderPreviewContext(t *testing.T) {
	p := NewProvider(testdataDir(t))

	msgs, err := p.PreviewContext("11111111-1111-1111-1111-111111111111", "line-000004", 1)
	if err != nil {
		t.Fatalf("PreviewContext: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("got %d preview messages, want 3", len(msgs))
	}

	if msgs[0].UUID != "line-000002" || msgs[0].Text != "fix the login bug" {
		t.Errorf("first preview message = %#v", msgs[0])
	}
	if msgs[1].UUID != "line-000004" || msgs[1].Role != "assistant" {
		t.Errorf("second preview message = %#v", msgs[1])
	}
	if msgs[2].UUID != "line-000007" {
		t.Errorf("third preview message = %#v", msgs[2])
	}
}

func TestProviderResumeCommand(t *testing.T) {
	p := NewProvider(testdataDir(t))

	if got := p.ResumeCommand("11111111-1111-1111-1111-111111111111", "/home/user/project"); got != "cd /home/user/project && codex resume 11111111-1111-1111-1111-111111111111" {
		t.Errorf("got %q", got)
	}
	if got := p.ResumeCommand("11111111-1111-1111-1111-111111111111", ""); got != "codex resume 11111111-1111-1111-1111-111111111111" {
		t.Errorf("got %q", got)
	}
}

func TestProviderSessionFile(t *testing.T) {
	p := NewProvider(testdataDir(t))

	got := p.SessionFile("22222222-2222-2222-2222-222222222222")
	want := filepath.Join(testdataDir(t), "sessions", "2026", "04", "02", "rollout-2026-04-02T11-00-00-22222222-2222-2222-2222-222222222222.jsonl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
