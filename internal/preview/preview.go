package preview

import (
	"fmt"
	"strings"

	"github.com/danielfollent/chatgrep/internal/provider"
)

// ANSI color codes
const (
	green  = "\033[32m"
	blue   = "\033[34m"
	dim    = "\033[2m"
	bold   = "\033[1m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

// Render formats preview messages with ANSI colors for the fzf preview pane.
// The message matching targetUUID gets a >>> highlight marker.
func Render(msgs []provider.PreviewMessage, targetUUID string) string {
	if len(msgs) == 0 {
		return ""
	}

	var b strings.Builder
	for _, m := range msgs {
		isTarget := m.UUID == targetUUID

		// Timestamp header
		b.WriteString(dim)
		b.WriteString(m.Timestamp)
		b.WriteString(reset)
		b.WriteString("\n")

		// Role label with color
		prefix := ""
		if isTarget {
			prefix = yellow + ">>> " + reset
		}

		var roleColor string
		var roleLabel string
		if m.Role == "user" {
			roleColor = green
			roleLabel = "User"
		} else {
			roleColor = blue
			roleLabel = "Assistant"
		}

		b.WriteString(prefix)
		b.WriteString(bold)
		b.WriteString(roleColor)
		b.WriteString(roleLabel)
		b.WriteString(reset)
		b.WriteString("\n")

		// Message text
		text := m.Text
		if len(text) > 1000 {
			text = text[:1000] + "..."
		}
		if isTarget {
			b.WriteString(bold)
		}
		b.WriteString(text)
		if isTarget {
			b.WriteString(reset)
		}
		b.WriteString("\n")

		b.WriteString(fmt.Sprintf("%s%s%s\n", dim, strings.Repeat("-", 40), reset))
	}

	return b.String()
}
