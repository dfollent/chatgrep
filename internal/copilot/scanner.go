package copilot

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/danielfollent/chatgrep/internal/text"
)

type Session struct {
	ID      string
	Dir     string // path to the session directory
	EventsF string // path to events.jsonl
}

type Workspace struct {
	CWD     string
	Branch  string
	Summary string
}

type SearchResult struct {
	SessionID string
	ID        string
	Role      string
	Text      string
	Snippet   string
	Timestamp string
}

// DiscoverSessions finds session directories under baseDir.
// Each valid session has an events.jsonl file.
func DiscoverSessions(baseDir string) ([]Session, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		eventsPath := filepath.Join(baseDir, e.Name(), "events.jsonl")
		if _, err := os.Stat(eventsPath); err != nil {
			continue
		}
		sessions = append(sessions, Session{
			ID:      e.Name(),
			Dir:     filepath.Join(baseDir, e.Name()),
			EventsF: eventsPath,
		})
	}
	return sessions, nil
}

// ReadWorkspace parses workspace.yaml for session metadata.
// Uses line-based parsing to avoid a YAML dependency.
func ReadWorkspace(path string) (Workspace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Workspace{}, err
	}

	var ws Workspace
	lines := strings.Split(string(data), "\n")
	inSummary := false

	for _, line := range lines {
		if inSummary {
			// Summary block scalar: indented lines belong to summary
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				if ws.Summary != "" {
					ws.Summary += "\n"
				}
				ws.Summary += strings.TrimSpace(line)
				continue
			}
			inSummary = false
		}

		if strings.HasPrefix(line, "cwd: ") {
			ws.CWD = strings.TrimPrefix(line, "cwd: ")
		} else if strings.HasPrefix(line, "branch: ") {
			ws.Branch = strings.TrimPrefix(line, "branch: ")
		} else if strings.HasPrefix(line, "summary: |-") {
			inSummary = true
		} else if strings.HasPrefix(line, "summary: ") {
			ws.Summary = strings.TrimPrefix(line, "summary: ")
		}
	}
	return ws, nil
}

// ScanSession searches a single events.jsonl for lines matching query.
func ScanSession(path string, query string, maxResults int) ([]SearchResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	terms := strings.Fields(query)
	var results []SearchResult

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		if len(results) >= maxResults {
			break
		}

		msg := ParseEvent(scanner.Bytes())
		if msg == nil {
			continue
		}

		if len(terms) > 0 && !text.MatchesAll(msg.Text, terms) {
			continue
		}

		snippetQuery := query
		if len(terms) > 0 {
			snippetQuery = terms[0]
		}

		results = append(results, SearchResult{
			ID:        msg.ID,
			Role:      msg.Role,
			Text:      msg.Text,
			Snippet:   text.Snippet(msg.Text, snippetQuery, text.MaxSnippetLen),
			Timestamp: msg.Timestamp,
		})
	}

	return results, scanner.Err()
}

// SearchAll searches all sessions in baseDir in parallel.
func SearchAll(baseDir string, query string, numWorkers int) ([]SearchResult, error) {
	sessions, err := DiscoverSessions(baseDir)
	if err != nil {
		return nil, err
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
				results, err := ScanSession(s.EventsF, query, 50)
				mu.Lock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				for i := range results {
					results[i].SessionID = s.ID
				}
				allResults = append(allResults, results...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return allResults, firstErr
}

// ReadMessages reads all parseable messages from an events.jsonl file.
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
		msg := ParseEvent(scanner.Bytes())
		if msg == nil {
			continue
		}
		msgs = append(msgs, *msg)
	}
	return msgs, scanner.Err()
}

