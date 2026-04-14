package fzf

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/danielfollent/chatgrep/internal/provider"
)

type Selection struct {
	ProviderName string
	SessionID    string
	MsgUUID      string
}

// FormatLine builds a tab-delimited fzf input line from a search match.
// Format: provider:sessionID\tmsgUUID\tprovider timeAgo R> snippet
// Fields 1-2 are hidden via --with-nth=3.., but accessible via {1} {2} for preview.
func FormatLine(m provider.Match) string {
	field1 := m.ProviderName + ":" + m.SessionID
	field2 := m.UUID

	roleTag := "U>"
	if m.Role == "assistant" {
		roleTag = "A>"
	}

	age := "?"
	if t, err := time.Parse(time.RFC3339, m.Timestamp); err == nil {
		age = relativeTime(t)
	} else if t, err := time.Parse(time.RFC3339Nano, m.Timestamp); err == nil {
		age = relativeTime(t)
	} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", m.Timestamp); err == nil {
		age = relativeTime(t)
	}

	display := fmt.Sprintf("%-7s %8s %s %s", m.ProviderName, age, roleTag, m.Snippet)
	return field1 + "\t" + field2 + "\t" + display
}

// relativeTime formats a timestamp as a human-readable relative duration.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// ParseSelection extracts provider, session, and message IDs from an fzf output line.
func ParseSelection(line string) (Selection, error) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) < 2 {
		return Selection{}, fmt.Errorf("invalid selection: expected tab-delimited fields")
	}

	provSession := strings.SplitN(parts[0], ":", 2)
	if len(provSession) != 2 {
		return Selection{}, fmt.Errorf("invalid selection: expected provider:sessionID in field 1, got %q", parts[0])
	}

	return Selection{
		ProviderName: provSession[0],
		SessionID:    provSession[1],
		MsgUUID:      parts[1],
	}, nil
}

// Run launches fzf with the given lines and preview command.
// query pre-populates fzf's search box so it ranks by match relevance.
// Returns the selected line or empty string if user cancelled (ESC).
func Run(lines []string, previewCmd string, binaryPath string, query string) (string, error) {
	fzfPath, err := exec.LookPath("fzf")
	if err != nil {
		return "", fmt.Errorf("fzf not found in PATH. Install it: brew install fzf")
	}

	args := []string{
		"--delimiter=\t",
		"--with-nth=3..",
		"--ansi",
		"--tiebreak=index",
	}

	if query != "" {
		args = append(args, "--query="+query)
	}

	if previewCmd != "" {
		args = append(args, "--preview="+previewCmd)
		args = append(args, "--preview-window=right:50%:wrap")
	}

	cmd := exec.Command(fzfPath, args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		for _, line := range lines {
			io.WriteString(stdin, line+"\n")
		}
	}()

	out, err := cmd.Output()
	if err != nil {
		// fzf exits 130 on ESC/Ctrl-C
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", nil
		}
		// fzf exits 1 when no match
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
