package color

import "os"

// ResolveEnabled determines whether color output should be enabled.
// Precedence: "never"->false, "always"->true, "auto"->NO_COLOR env then TTY.
func ResolveEnabled(flagValue string, isTTY bool) bool {
	switch flagValue {
	case "never":
		return false
	case "always":
		return true
	default: // "auto"
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		return isTTY
	}
}
