package color

import "fmt"

// Style represents a single element's complete ANSI styling.
// FG/BG: -1 = unset, 0-7 = standard color, 8-255 = extended 256-color.
// Bold/Dim/Underline/Italic: 0 = default (off), 1 = on, -1 = explicit off.
type Style struct {
	FG        int
	BG        int
	Bold      int
	Dim       int
	Underline int
	Italic    int
}

// Theme holds styles for all colorable elements plus a master enable switch.
type Theme struct {
	ProviderClaude  Style
	ProviderCopilot Style
	ProviderCodex   Style
	RoleUser        Style
	RoleAssistant   Style
	Timestamp       Style
	Marker          Style
	Separator       Style
	Match           Style

	Enabled bool
}

// DefaultTheme returns the theme matching chatgrep's original hardcoded colors.
func DefaultTheme() *Theme {
	return &Theme{
		ProviderClaude:  Style{FG: 6, BG: -1},  // cyan
		ProviderCopilot: Style{FG: 5, BG: -1},  // magenta
		ProviderCodex:   Style{FG: 3, BG: -1},  // yellow
		RoleUser:        Style{FG: 2, BG: -1},  // green
		RoleAssistant:   Style{FG: 4, BG: -1},  // blue
		Timestamp:       Style{FG: -1, BG: -1, Dim: 1},
		Marker:          Style{FG: 3, BG: -1, Bold: 1}, // yellow + bold
		Separator:       Style{FG: -1, BG: -1, Dim: 1},
		Match:           Style{FG: -1, BG: -1},
		Enabled:         true,
	}
}

// Sequence renders the Style as an ANSI escape sequence string.
// Returns "" if no attributes are set.
func (s Style) Sequence() string {
	var codes string

	if s.Bold == 1 {
		codes += "\033[1m"
	}
	if s.Dim == 1 {
		codes += "\033[2m"
	}
	if s.Italic == 1 {
		codes += "\033[3m"
	}
	if s.Underline == 1 {
		codes += "\033[4m"
	}

	if s.FG >= 0 && s.FG <= 7 {
		codes += fmt.Sprintf("\033[%dm", 30+s.FG)
	} else if s.FG >= 8 && s.FG <= 255 {
		codes += fmt.Sprintf("\033[38;5;%dm", s.FG)
	}

	if s.BG >= 0 && s.BG <= 7 {
		codes += fmt.Sprintf("\033[%dm", 40+s.BG)
	} else if s.BG >= 8 && s.BG <= 255 {
		codes += fmt.Sprintf("\033[48;5;%dm", s.BG)
	}

	return codes
}

// Apply wraps text in the style's ANSI sequence and reset.
// Returns plain text when the theme is disabled or the style has no attributes.
func (t *Theme) Apply(s Style, text string) string {
	if !t.Enabled {
		return text
	}
	seq := s.Sequence()
	if seq == "" {
		return text
	}
	return seq + text + "\033[0m"
}

// Reset returns the ANSI reset sequence, or "" when disabled.
func (t *Theme) Reset() string {
	if !t.Enabled {
		return ""
	}
	return "\033[0m"
}

// ProviderStyle returns the Style for a provider by name.
func (t *Theme) ProviderStyle(name string) Style {
	switch name {
	case "claude":
		return t.ProviderClaude
	case "copilot":
		return t.ProviderCopilot
	case "codex":
		return t.ProviderCodex
	default:
		return Style{FG: -1, BG: -1}
	}
}

// RoleStyle returns the Style for a message role.
func (t *Theme) RoleStyle(role string) Style {
	switch role {
	case "user":
		return t.RoleUser
	case "assistant":
		return t.RoleAssistant
	default:
		return Style{FG: -1, BG: -1}
	}
}
