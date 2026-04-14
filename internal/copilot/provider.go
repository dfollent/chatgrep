package copilot

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielfollent/chatgrep/internal/provider"
)

type Provider struct {
	baseDir string // ~/.copilot/session-state
}

func NewProvider(baseDir string) *Provider {
	return &Provider{baseDir: baseDir}
}

// DefaultBaseDir returns the standard Copilot session-state location.
// Respects $COPILOT_CONFIG_DIR if set.
func DefaultBaseDir() string {
	if dir := os.Getenv("COPILOT_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "session-state")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".copilot", "session-state")
}

func (p *Provider) Name() string { return "copilot" }

func (p *Provider) Discover() ([]provider.SessionInfo, error) {
	sessions, err := DiscoverSessions(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading copilot sessions dir: %w", err)
	}

	var all []provider.SessionInfo
	for _, s := range sessions {
		wsPath := filepath.Join(s.Dir, "workspace.yaml")
		ws, _ := ReadWorkspace(wsPath)

		all = append(all, provider.SessionInfo{
			ID:       s.ID,
			FilePath: s.EventsF,
			CWD:      ws.CWD,
			Slug:     truncateSummary(ws.Summary, 60),
		})
	}
	return all, nil
}

func (p *Provider) Search(query string, numWorkers int) ([]provider.Match, error) {
	sessions, err := DiscoverSessions(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading copilot sessions dir: %w", err)
	}

	// Reuse SearchAll but we need to attach workspace metadata
	wsCache := make(map[string]Workspace)
	for _, s := range sessions {
		wsPath := filepath.Join(s.Dir, "workspace.yaml")
		ws, _ := ReadWorkspace(wsPath)
		wsCache[s.ID] = ws
	}

	results, err := SearchAll(p.baseDir, query, numWorkers)
	if err != nil {
		return nil, err
	}

	matches := make([]provider.Match, 0, len(results))
	for _, r := range results {
		ws := wsCache[r.SessionID]
		matches = append(matches, provider.Match{
			ProviderName: "copilot",
			SessionID:    r.SessionID,
			UUID:         r.ID,
			Role:         r.Role,
			Snippet:      r.Snippet,
			Timestamp:    r.Timestamp,
			CWD:          ws.CWD,
			Slug:         truncateSummary(ws.Summary, 60),
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
		return fmt.Sprintf("cd %s && copilot --resume=%s", cwd, sessionID)
	}
	return fmt.Sprintf("copilot --resume=%s", sessionID)
}

func (p *Provider) SessionFile(sessionID string) string {
	path := filepath.Join(p.baseDir, sessionID, "events.jsonl")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// truncateSummary shortens a workspace summary for display.
func truncateSummary(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
