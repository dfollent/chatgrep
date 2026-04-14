package text

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

const MaxSnippetLen = 200

// CollapseWhitespace strips ANSI escapes, normalizes all whitespace runs
// to a single space, and trims leading/trailing space.
func CollapseWhitespace(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// Snippet extracts a maxLen-rune window from s, centered on the first
// occurrence of query. Falls back to the start if query is empty or not found.
func Snippet(s, query string, maxLen int) string {
	s = CollapseWhitespace(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	center := 0
	if query != "" {
		idx := strings.Index(strings.ToLower(s), strings.ToLower(query))
		if idx >= 0 {
			center = len([]rune(s[:idx]))
		}
	}

	start := center - maxLen/2
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(runes) {
		end = len(runes)
		start = end - maxLen
		if start < 0 {
			start = 0
		}
	}

	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < len(runes) {
		suffix = "..."
	}
	return prefix + string(runes[start:end]) + suffix
}

// MatchesAll returns true if text contains every term (case-insensitive).
func MatchesAll(text string, terms []string) bool {
	lower := strings.ToLower(text)
	for _, t := range terms {
		if !strings.Contains(lower, strings.ToLower(t)) {
			return false
		}
	}
	return true
}
