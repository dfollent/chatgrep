package text

import (
	"strings"
	"testing"
)

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello  world", "hello world"},
		{"line\none\ttwo", "line one two"},
		{"  leading trailing  ", "leading trailing"},
		{"no change", "no change"},
	}
	for _, tt := range tests {
		got := CollapseWhitespace(tt.in)
		if got != tt.want {
			t.Errorf("CollapseWhitespace(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStripANSI(t *testing.T) {
	got := CollapseWhitespace("before \x1b[35mpurple\x1b[0m after")
	if strings.Contains(got, "\x1b") {
		t.Errorf("CollapseWhitespace should strip ANSI: %q", got)
	}
	if got != "before purple after" {
		t.Errorf("got %q, want %q", got, "before purple after")
	}
}

func TestSnippet_Short(t *testing.T) {
	got := Snippet("short text", "short", MaxSnippetLen)
	if got != "short text" {
		t.Errorf("got %q, want %q", got, "short text")
	}
}

func TestSnippet_CentersOnQuery(t *testing.T) {
	prefix := strings.Repeat("padding ", 50) // 400 chars
	text := prefix + "target word" + strings.Repeat(" filler", 50)
	got := Snippet(text, "target", MaxSnippetLen)
	if !strings.Contains(got, "target") {
		t.Errorf("snippet should contain 'target': %q", got)
	}
}

func TestSnippet_Truncation(t *testing.T) {
	longText := strings.Repeat("word ", 200)
	got := Snippet(longText, "word", MaxSnippetLen)
	if len([]rune(got)) > MaxSnippetLen+10 {
		t.Errorf("snippet rune length %d exceeds max %d", len([]rune(got)), MaxSnippetLen)
	}
}

func TestSnippet_FlattensNewlines(t *testing.T) {
	got := Snippet("line\none\ntwo", "one", MaxSnippetLen)
	if strings.Contains(got, "\n") {
		t.Errorf("snippet contains newline: %q", got)
	}
}

func TestMatchesAll_SingleTerm(t *testing.T) {
	if !MatchesAll("hello world", []string{"hello"}) {
		t.Error("should match single term")
	}
}

func TestMatchesAll_MultiTerm(t *testing.T) {
	if !MatchesAll("deploy to staging server", []string{"deploy", "staging"}) {
		t.Error("should match both terms")
	}
}

func TestMatchesAll_OrderIndependent(t *testing.T) {
	if !MatchesAll("staging deploy", []string{"deploy", "staging"}) {
		t.Error("should match regardless of order")
	}
}

func TestMatchesAll_CaseInsensitive(t *testing.T) {
	if !MatchesAll("Deploy Staging", []string{"deploy", "staging"}) {
		t.Error("should match case-insensitively")
	}
}

func TestMatchesAll_MissingTerm(t *testing.T) {
	if MatchesAll("deploy to production", []string{"deploy", "staging"}) {
		t.Error("should not match when a term is missing")
	}
}

func TestMatchesAll_EmptyTerms(t *testing.T) {
	if !MatchesAll("anything", []string{}) {
		t.Error("empty terms should match everything")
	}
}
