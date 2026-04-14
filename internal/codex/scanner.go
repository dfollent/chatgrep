package codex

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danielfollent/chatgrep/internal/text"
)

type Session struct {
	ID          string
	RolloutPath string
	CWD         string
	Slug        string
}

type SearchResult struct {
	SessionID string
	ID        string
	Role      string
	Text      string
	Snippet   string
	Timestamp string
}

// SessionsDir returns the Codex rollout root under a Codex home directory.
func SessionsDir(baseDir string) string {
	return filepath.Join(baseDir, "sessions")
}

// DiscoverSessions finds rollout JSONL files under CODEX_HOME/sessions.
func DiscoverSessions(baseDir string) ([]Session, error) {
	root := SessionsDir(baseDir)
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}

	var sessions []Session
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" || !strings.HasPrefix(filepath.Base(path), "rollout-") {
			return nil
		}

		session, err := readSessionSummary(path)
		if err != nil {
			return nil
		}
		sessions = append(sessions, session)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// ScanSession searches a single rollout file for visible chat messages that match query.
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

	lineNum := 0
	var prev *ParsedMessage
	for scanner.Scan() {
		lineNum++
		if len(results) >= maxResults {
			break
		}

		msg := ParseLine(scanner.Bytes(), lineNum)
		if msg == nil {
			continue
		}
		if isDuplicateVisibleMessage(prev, msg) {
			continue
		}
		prev = msg

		searchText := text.CollapseWhitespace(msg.Text)
		if len(terms) > 0 && !text.MatchesAll(searchText, terms) {
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
			Snippet:   text.Snippet(searchText, snippetQuery, text.MaxSnippetLen),
			Timestamp: msg.Timestamp,
		})
	}

	return results, scanner.Err()
}

// SearchAll searches all discovered Codex rollout files in parallel.
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
				results, err := ScanSession(s.RolloutPath, query, 50)
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

// ReadMessages reads all visible chat messages from a rollout file.
func ReadMessages(path string) ([]ParsedMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var msgs []ParsedMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	lineNum := 0
	var prev *ParsedMessage
	for scanner.Scan() {
		lineNum++
		msg := ParseLine(scanner.Bytes(), lineNum)
		if msg == nil {
			continue
		}
		if isDuplicateVisibleMessage(prev, msg) {
			continue
		}
		prev = msg
		msgs = append(msgs, *msg)
	}

	return msgs, scanner.Err()
}

type sessionMeta struct {
	ID  string
	CWD string
}

func readSessionSummary(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	var session Session
	session.RolloutPath = path

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if session.ID == "" {
			if meta := parseSessionMeta(line); meta != nil {
				session.ID = meta.ID
				session.CWD = meta.CWD
				if session.Slug != "" {
					break
				}
				continue
			}
		}

		if session.Slug == "" {
			msg := ParseLine(line, lineNum)
			if msg != nil && msg.Role == "user" {
				session.Slug = msg.Text
				if session.ID != "" {
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Session{}, err
	}
	if session.ID == "" {
		return Session{}, os.ErrNotExist
	}

	return session, nil
}

func parseSessionMeta(line []byte) *sessionMeta {
	if len(line) == 0 {
		return nil
	}

	var envelope struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return nil
	}
	if envelope.Type != "session_meta" || len(envelope.Payload) == 0 {
		return nil
	}

	var payload struct {
		ID  string `json:"id"`
		CWD string `json:"cwd"`
	}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return nil
	}
	if payload.ID == "" {
		return nil
	}

	return &sessionMeta{
		ID:  payload.ID,
		CWD: payload.CWD,
	}
}

func isDuplicateVisibleMessage(prev, curr *ParsedMessage) bool {
	if prev == nil || curr == nil {
		return false
	}
	if prev.Source == curr.Source || prev.Role != curr.Role {
		return false
	}
	if text.CollapseWhitespace(prev.Text) != text.CollapseWhitespace(curr.Text) {
		return false
	}
	return timestampsClose(prev.Timestamp, curr.Timestamp)
}

func timestampsClose(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	if errA != nil || errB != nil {
		return a == b
	}
	diff := ta.Sub(tb)
	if diff < 0 {
		diff = -diff
	}
	return diff <= time.Second
}
