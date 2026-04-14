package codex

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielfollent/chatgrep/internal/provider"
)

type Provider struct {
	baseDir string // ~/.codex
}

func NewProvider(baseDir string) *Provider {
	return &Provider{baseDir: baseDir}
}

// DefaultBaseDir returns the standard Codex home directory.
// Respects $CODEX_HOME if set.
func DefaultBaseDir() string {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}

func (p *Provider) Name() string { return "codex" }

func (p *Provider) Discover() ([]provider.SessionInfo, error) {
	sessions, err := DiscoverSessions(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading codex sessions dir: %w", err)
	}

	all := make([]provider.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		all = append(all, provider.SessionInfo{
			ID:       s.ID,
			FilePath: s.RolloutPath,
			CWD:      s.CWD,
			Slug:     s.Slug,
		})
	}
	return all, nil
}

func (p *Provider) Search(query string, numWorkers int) ([]provider.Match, error) {
	sessions, err := DiscoverSessions(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading codex sessions dir: %w", err)
	}

	sessionCache := make(map[string]Session, len(sessions))
	for _, s := range sessions {
		sessionCache[s.ID] = s
	}

	results, err := SearchAll(p.baseDir, query, numWorkers)
	if err != nil {
		return nil, err
	}

	matches := make([]provider.Match, 0, len(results))
	for _, r := range results {
		session := sessionCache[r.SessionID]
		matches = append(matches, provider.Match{
			ProviderName: "codex",
			SessionID:    r.SessionID,
			UUID:         r.ID,
			Role:         r.Role,
			Snippet:      r.Snippet,
			Timestamp:    r.Timestamp,
			CWD:          session.CWD,
			Slug:         session.Slug,
		})
	}
	return matches, nil
}

func (p *Provider) PreviewContext(sessionID, msgUUID string, windowSize int) ([]provider.PreviewMessage, error) {
	path := p.SessionFile(sessionID)
	if path == "" {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}

	msgs, err := ReadMessages(path)
	if err != nil {
		return nil, err
	}

	targetIdx := -1
	for i, m := range msgs {
		if m.ID == msgUUID {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return nil, fmt.Errorf("message %q not found in session %q", msgUUID, sessionID)
	}

	start := targetIdx - windowSize
	if start < 0 {
		start = 0
	}
	end := targetIdx + windowSize + 1
	if end > len(msgs) {
		end = len(msgs)
	}

	result := make([]provider.PreviewMessage, 0, end-start)
	for _, m := range msgs[start:end] {
		result = append(result, provider.PreviewMessage{
			Role:      m.Role,
			Text:      m.Text,
			UUID:      m.ID,
			Timestamp: m.Timestamp,
		})
	}
	return result, nil
}

func (p *Provider) ResumeCommand(sessionID, cwd string) string {
	if cwd != "" {
		return fmt.Sprintf("cd %s && codex resume %s", cwd, sessionID)
	}
	return fmt.Sprintf("codex resume %s", sessionID)
}

func (p *Provider) SessionFile(sessionID string) string {
	sessions, err := DiscoverSessions(p.baseDir)
	if err != nil {
		return ""
	}
	for _, s := range sessions {
		if s.ID == sessionID {
			return s.RolloutPath
		}
	}
	return ""
}
