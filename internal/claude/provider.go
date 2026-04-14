package claude

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/danielfollent/chatgrep/internal/provider"
)

type Provider struct {
	baseDir string // ~/.claude/projects
}

func NewProvider(baseDir string) *Provider {
	return &Provider{baseDir: baseDir}
}

// DefaultBaseDir returns ~/.claude/projects, the standard Claude session location.
func DefaultBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

func (p *Provider) Name() string { return "claude" }

// Discover finds all sessions across all project subdirectories.
func (p *Provider) Discover() ([]provider.SessionInfo, error) {
	projectDirs, err := os.ReadDir(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading claude projects dir: %w", err)
	}

	var all []provider.SessionInfo
	for _, d := range projectDirs {
		if !d.IsDir() {
			continue
		}
		projDir := filepath.Join(p.baseDir, d.Name())
		sessions, err := DiscoverSessions(projDir)
		if err != nil {
			continue
		}
		for _, s := range sessions {
			cwd, slug := peekSessionMeta(s.FilePath)
			all = append(all, provider.SessionInfo{
				ID:       s.ID,
				FilePath: s.FilePath,
				CWD:      cwd,
				Slug:     slug,
			})
		}
	}
	return all, nil
}

// Search searches all sessions across all project subdirectories in parallel.
func (p *Provider) Search(query string, numWorkers int) ([]provider.Match, error) {
	projectDirs, err := os.ReadDir(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading claude projects dir: %w", err)
	}

	var allSessions []Session
	for _, d := range projectDirs {
		if !d.IsDir() {
			continue
		}
		projDir := filepath.Join(p.baseDir, d.Name())
		sessions, err := DiscoverSessions(projDir)
		if err != nil {
			continue
		}
		allSessions = append(allSessions, sessions...)
	}

	ch := make(chan Session, len(allSessions))
	for _, s := range allSessions {
		ch <- s
	}
	close(ch)

	var mu sync.Mutex
	var matches []provider.Match
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
				for _, r := range results {
					matches = append(matches, provider.Match{
						ProviderName: "claude",
						SessionID:    r.SessionID,
						UUID:         r.UUID,
						Role:         r.Role,
						Snippet:      r.Snippet,
						Timestamp:    r.Timestamp,
						CWD:          r.CWD,
						Slug:         r.Slug,
					})
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return matches, firstErr
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
		if m.UUID == msgUUID {
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
			UUID:      m.UUID,
			Timestamp: m.Timestamp,
		})
	}
	return result, nil
}

func (p *Provider) ResumeCommand(sessionID, cwd string) string {
	if cwd != "" {
		return fmt.Sprintf("cd %s && claude --resume %s", cwd, sessionID)
	}
	return fmt.Sprintf("claude --resume %s", sessionID)
}

// SessionFile finds the JSONL file for a session ID by searching all project dirs.
func (p *Provider) SessionFile(sessionID string) string {
	filename := sessionID + ".jsonl"
	projectDirs, err := os.ReadDir(p.baseDir)
	if err != nil {
		return ""
	}
	for _, d := range projectDirs {
		if !d.IsDir() {
			continue
		}
		path := filepath.Join(p.baseDir, d.Name(), filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// peekSessionMeta reads the first few lines to extract CWD and slug.
func peekSessionMeta(path string) (cwd, slug string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for i := 0; i < 20 && s.Scan(); i++ {
		msg := ParseLine(s.Bytes())
		if msg == nil {
			continue
		}
		if cwd == "" {
			cwd = msg.CWD
		}
		if slug == "" && msg.Slug != "" {
			slug = msg.Slug
		}
		if cwd != "" && slug != "" {
			break
		}
	}
	return cwd, slug
}
