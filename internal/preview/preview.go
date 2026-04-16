package preview

import (
	"fmt"
	"strings"

	"github.com/danielfollent/chatgrep/internal/color"
	"github.com/danielfollent/chatgrep/internal/provider"
)

// Render formats preview messages with ANSI colors for the fzf preview pane.
// The message matching targetUUID gets a >>> highlight marker.
func Render(msgs []provider.PreviewMessage, targetUUID string, theme *color.Theme) string {
	if len(msgs) == 0 {
		return ""
	}

	var b strings.Builder
	for _, m := range msgs {
		isTarget := m.UUID == targetUUID

		// Timestamp header
		b.WriteString(theme.Apply(theme.Timestamp, m.Timestamp))
		b.WriteString("\n")

		// Role label with color
		prefix := ""
		if isTarget {
			prefix = theme.Apply(theme.Marker, ">>> ")
		}

		var roleStyle color.Style
		var roleLabel string
		if m.Role == "user" {
			roleStyle = theme.RoleUser
			roleLabel = "User"
		} else {
			roleStyle = theme.RoleAssistant
			roleLabel = "Assistant"
		}

		// Role label gets bold + role color
		boldRole := roleStyle
		boldRole.Bold = 1
		b.WriteString(prefix)
		b.WriteString(theme.Apply(boldRole, roleLabel))
		b.WriteString("\n")

		// Message text
		text := m.Text
		if len(text) > 1000 {
			text = text[:1000] + "..."
		}
		if isTarget {
			boldStyle := color.Style{FG: -1, BG: -1, Bold: 1}
			b.WriteString(theme.Apply(boldStyle, text))
		} else {
			b.WriteString(text)
		}
		b.WriteString("\n")

		b.WriteString(theme.Apply(theme.Separator, fmt.Sprintf("%s", strings.Repeat("-", 40))))
		b.WriteString("\n")
	}

	return b.String()
}
