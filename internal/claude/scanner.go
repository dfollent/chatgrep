package claude

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/danielfollent/chatgrep/internal/text"
)

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

	terms := strings.Fields(query)
	var results []SearchResult

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		if len(results) >= maxResults {
			break
		}

		msg := ParseLine(scanner.Bytes())
		if msg == nil || msg.IsSidechain {
			continue
		}

		if len(terms) > 0 && !text.MatchesAll(msg.Text, terms) {
			continue
		}

		// Center snippet on first matching term
		snippetQuery := query
		if len(terms) > 0 {
			snippetQuery = terms[0]
		}

		results = append(results, SearchResult{
			SessionID: msg.SessionID,
			UUID:      msg.UUID,
			Role:      msg.Role,
			Text:      msg.Text,
			Snippet:   text.Snippet(msg.Text, snippetQuery, text.MaxSnippetLen),
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

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range ch {
				results, err := ScanSession(s.FilePath, query, 50)
				if err != nil {
					continue
				}
				mu.Lock()
				allResults = append(allResults, results...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return allResults, nil
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
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		msg := ParseLine(scanner.Bytes())
		if msg == nil {
			continue
		}
		msgs = append(msgs, *msg)
	}
	return msgs, scanner.Err()
}

