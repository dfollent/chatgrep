package claude

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

const MaxSnippetLen = 200

type Session struct {
	ID       string
	FilePath string
}

type SearchResult struct {
	SessionID string
	UUID      string
	Role      string
	Text      string
	Snippet   string
	Timestamp string
	CWD       string
	Slug      string
}

// DiscoverSessions finds JSONL session files at the top level of baseDir.
// Skips subdirectories (which contain subagent sessions).
func DiscoverSessions(baseDir string) ([]Session, error) {
	pattern := filepath.Join(baseDir, "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(matches))
	for _, path := range matches {
		id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		sessions = append(sessions, Session{ID: id, FilePath: path})
	}
	return sessions, nil
}

// ScanSession searches a single session file for lines matching query.
// Returns up to maxResults matches. Empty query matches all messages.
func ScanSession(path string, query string, maxResults int) ([]SearchResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	queryLower := strings.ToLower(query)
	var results []SearchResult

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		if len(results) >= maxResults {
			break
		}

		msg := ParseLine(scanner.Bytes())
		if msg == nil {
			continue
		}

		if query != "" && !strings.Contains(strings.ToLower(msg.Text), queryLower) {
			continue
		}

		results = append(results, SearchResult{
			SessionID: msg.SessionID,
			UUID:      msg.UUID,
			Role:      msg.Role,
			Text:      msg.Text,
			Snippet:   snippet(msg.Text, query, MaxSnippetLen),
			Timestamp: msg.Timestamp,
			CWD:       msg.CWD,
			Slug:      msg.Slug,
		})
	}

	return results, scanner.Err()
}

// SearchAll searches all sessions in baseDir in parallel using numWorkers goroutines.
func SearchAll(baseDir string, query string, numWorkers int) ([]SearchResult, error) {
	sessions, err := DiscoverSessions(baseDir)
	if err != nil {
		return nil, err
	}

	type result struct {
		results []SearchResult
		err     error
	}

	ch := make(chan Session, len(sessions))
	for _, s := range sessions {
		ch <- s
	}
	close(ch)

	var mu sync.Mutex
	var allResults []SearchResult
	var firstErr error

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range ch {
				results, err := ScanSession(s.FilePath, query, 50)
				mu.Lock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				allResults = append(allResults, results...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return allResults, firstErr
}

// ReadMessages reads all parseable messages from a session file.
// Used by preview to build context around a target message.
func ReadMessages(path string) ([]ParsedMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var msgs []ParsedMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		msg := ParseLine(scanner.Bytes())
		if msg == nil {
			continue
		}
		msgs = append(msgs, *msg)
	}
	return msgs, scanner.Err()
}

// snippet extracts a maxLen-rune window from s, centered on the first
// occurrence of query. Falls back to the start if query is empty or not found.
func snippet(s, query string, maxLen int) string {
	s = collapseWhitespace(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	// Find match position in rune-space
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

func collapseWhitespace(s string) string {
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
