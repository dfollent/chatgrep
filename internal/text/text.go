package text

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Compiled regexes for markdown stripping, ordered to handle ambiguity
// (e.g. *** is HR before bold-italic).
var (
	mdFencedCode  = regexp.MustCompile("```[a-zA-Z]*")
	mdHRule       = regexp.MustCompile(`(?:^|(?:\s))(?:---+|\*\*\*+|___+)(?:(?:\s)|$)`)
	mdBoldStars   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdBoldUnder   = regexp.MustCompile(`__(.+?)__`)
	mdItalicStars = regexp.MustCompile(`\*(\S[^*]*?\S|\S)\*`)
	mdInlineCode  = regexp.MustCompile("`([^`]+)`")
	mdHeading     = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	mdPipe        = regexp.MustCompile(`\|`)
	mdBlockquote  = regexp.MustCompile(`(?m)(?:^|(?:\s))>\s`)
	mdBullet      = regexp.MustCompile(`(?m)^[\t ]*[-*+]\s+`)
	mdOrderedList = regexp.MustCompile(`(?m)^[\t ]*\d+\.\s+`)
)

const MaxSnippetLen = 200

// StripMarkdown removes common markdown formatting so snippets read cleanly.
// Display-only - the full text is preserved elsewhere for matching.
func StripMarkdown(s string) string {
	s = mdFencedCode.ReplaceAllString(s, " ")
	s = mdHRule.ReplaceAllString(s, " ")
	s = mdBoldStars.ReplaceAllString(s, "$1")
	s = mdBoldUnder.ReplaceAllString(s, "$1")
	s = mdItalicStars.ReplaceAllString(s, "$1")
	s = mdInlineCode.ReplaceAllString(s, "$1")
	s = mdHeading.ReplaceAllString(s, " ")
	s = mdPipe.ReplaceAllString(s, " ")
	s = mdBlockquote.ReplaceAllString(s, " ")
	s = mdBullet.ReplaceAllString(s, " ")
	s = mdOrderedList.ReplaceAllString(s, " ")
	return s
}

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
	s = StripMarkdown(s)
	s = strings.Join(strings.Fields(s), " ")
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

	// Snap window edges to word boundaries within 20-rune tolerance
	const snapTolerance = 20
	if start > 0 {
		best := start
		for i := start; i < start+snapTolerance && i < len(runes); i++ {
			if runes[i] == ' ' {
				best = i + 1 // start after the space
				break
			}
		}
		start = best
	}
	if end < len(runes) {
		best := end
		for i := end - 1; i >= end-snapTolerance && i >= start; i-- {
			if runes[i] == ' ' {
				best = i // end before the space
				break
			}
		}
		end = best
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
