package provider

// SessionInfo is metadata about a discovered session.
type SessionInfo struct {
	ID       string
	FilePath string // path to the JSONL file (or directory for Copilot)
	CWD      string
	Slug     string // human-readable name (empty if unavailable)
}

// Match is a single search hit within a session.
type Match struct {
	ProviderName string
	SessionID    string
	UUID         string
	Role         string
	Snippet      string
	Timestamp    string
	CWD          string
	Slug         string
}

// PreviewMessage is a message shown in the fzf preview pane.
type PreviewMessage struct {
	Role      string
	Text      string
	UUID      string
	Timestamp string
}

// Provider abstracts session storage for different AI coding tools.
type Provider interface {
	Name() string
	Discover() ([]SessionInfo, error)
	Search(query string, numWorkers int) ([]Match, error)
	PreviewContext(sessionID, msgUUID string, windowSize int) ([]PreviewMessage, error)
	ResumeCommand(sessionID, cwd string) string
	SessionFile(sessionID string) string
}
