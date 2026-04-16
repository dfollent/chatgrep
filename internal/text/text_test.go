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

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"bold stars", "the **bold** word", "the bold word"},
		{"bold underscores", "the __bold__ word", "the bold word"},
		{"italic stars", "an *italic* word", "an italic word"},
		{"inline code", "use `fmt.Println` here", "use fmt.Println here"},
		{"fenced code block", "before ```go\ncode\n``` after", "before code after"},
		{"heading h3", "### My heading", "My heading"},
		{"heading h1", "# Title", "Title"},
		{"horizontal rule dashes", "above --- below", "above below"},
		{"horizontal rule stars", "above *** below", "above below"},
		{"horizontal rule underscores", "above ___ below", "above below"},
		{"pipe table", "| col1 | col2 |", "col1 col2"},
		{"bullet dash", "- item one", "item one"},
		{"bullet star", "* item two", "item two"},
		{"bullet plus", "+ item three", "item three"},
		{"ordered list", "1. first item", "first item"},
		{"blockquote", "> quoted text", "quoted text"},
		{"plain text", "no markdown here", "no markdown here"},
		{"real world", "**Goal:** describe the system", "Goal: describe the system"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripMarkdown(tt.in)
			// Normalize whitespace for comparison
			got = strings.Join(strings.Fields(got), " ")
			want := strings.Join(strings.Fields(tt.want), " ")
			if got != want {
				t.Errorf("StripMarkdown(%q) = %q, want %q", tt.in, got, want)
			}
		})
	}
}

func TestSnippet_DoesNotCutWords(t *testing.T) {
	// Place query at rune 150 so start = 150-100 = 50.
	// Put a 30-char word at rune 40 so it spans the window edge.
	prefix := strings.Repeat("a ", 20)   // 40 runes
	longWord := "abcdefghijklmnopqrstuvwxyzabcd" // 30 runes, ends at 70
	// pad to push TARGET to ~rune 150
	mid := strings.Repeat("z ", 40) // 80 runes (total so far: 40+30+1+80 = 151)
	text := prefix + longWord + " " + mid + "TARGET" + strings.Repeat(" mm", 80)
	got := Snippet(text, "TARGET", MaxSnippetLen)
	trimmed := strings.TrimPrefix(got, "...")
	if len(trimmed) > 0 {
		firstSpace := strings.IndexByte(trimmed, ' ')
		if firstSpace > 0 {
			firstWord := trimmed[:firstSpace]
			// Should not be a fragment of the long word
			if len(firstWord) > 2 && strings.Contains(longWord, firstWord) && firstWord != longWord {
				t.Errorf("snippet starts with word fragment %q: %q", firstWord, got)
			}
		}
	}
}

func TestSnippet_WordBoundaryPreservesWindowSize(t *testing.T) {
	text := strings.Repeat("abcdefgh ", 50) // ~450 chars
	got := Snippet(text, "abcdefgh", MaxSnippetLen)
	runeLen := len([]rune(strings.TrimPrefix(strings.TrimSuffix(got, "..."), "...")))
	// Should be within 40 runes of MaxSnippetLen
	if runeLen < MaxSnippetLen-40 {
		t.Errorf("snippet too short after word snapping: %d runes (min %d)", runeLen, MaxSnippetLen-40)
	}
}

func TestSnippet_WordBoundary_AllOneWord(t *testing.T) {
	// No spaces to snap to - should still work without panic
	text := strings.Repeat("x", 400)
	got := Snippet(text, "x", MaxSnippetLen)
	if len([]rune(got)) > MaxSnippetLen+10 {
		t.Errorf("snippet too long for all-one-word input: %d runes", len([]rune(got)))
	}
}

func TestSnippet_WordBoundary_QueryAtStart(t *testing.T) {
	text := "target " + strings.Repeat("filler ", 60)
	got := Snippet(text, "target", MaxSnippetLen)
	if !strings.HasPrefix(got, "target") {
		t.Errorf("snippet should start with query when at start: %q", got)
	}
}

func TestSnippet_WordBoundary_QueryAtEnd(t *testing.T) {
	text := strings.Repeat("filler ", 60) + "target"
	got := Snippet(text, "target", MaxSnippetLen)
	if !strings.HasSuffix(got, "target") {
		t.Errorf("snippet should end with query when at end: %q", got)
	}
}

func TestSnippet_StripsMarkdown(t *testing.T) {
	text := "### **Goal:** describe the | system | using `code` and > quoted text"
	got := Snippet(text, "describe", MaxSnippetLen)
	for _, bad := range []string{"**", "###", "|", "`", ">"} {
		if strings.Contains(got, bad) {
			t.Errorf("snippet still contains %q: %q", bad, got)
		}
	}
	if !strings.Contains(got, "describe") {
		t.Errorf("snippet should contain query 'describe': %q", got)
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
