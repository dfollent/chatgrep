package color

import (
	"fmt"
	"strconv"
	"strings"
)

var validElements = map[string]bool{
	"provider.claude":  true,
	"provider.copilot": true,
	"provider.codex":   true,
	"role.user":        true,
	"role.assistant":   true,
	"timestamp":        true,
	"marker":           true,
	"separator":        true,
	"match":            true,
}

var namedColors = map[string]int{
	"black":   0,
	"red":     1,
	"green":   2,
	"yellow":  3,
	"blue":    4,
	"magenta": 5,
	"cyan":    6,
	"white":   7,
}

var validStyles = map[string]bool{
	"bold":        true,
	"dim":         true,
	"underline":   true,
	"italic":      true,
	"nobold":      true,
	"nodim":       true,
	"nounderline": true,
	"noitalic":    true,
}

// ParseSpec parses a color spec string in the format "element:attribute:value".
func ParseSpec(spec string) (element, attr, value string, err error) {
	parts := strings.SplitN(spec, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid spec %q: expected element:attribute:value", spec)
	}
	element, attr, value = parts[0], parts[1], parts[2]

	if !validElements[element] {
		return "", "", "", fmt.Errorf("unknown element %q", element)
	}

	switch attr {
	case "fg", "bg":
		if err := validateColor(value); err != nil {
			return "", "", "", fmt.Errorf("invalid color in %q: %w", spec, err)
		}
	case "style":
		if !validStyles[value] {
			return "", "", "", fmt.Errorf("invalid style %q in %q", value, spec)
		}
	default:
		return "", "", "", fmt.Errorf("unknown attribute %q in %q (use fg, bg, or style)", attr, spec)
	}

	return element, attr, value, nil
}

func validateColor(value string) error {
	if _, ok := namedColors[value]; ok {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("unknown color %q", value)
	}
	if n < 0 || n > 255 {
		return fmt.Errorf("color number %d out of range (0-255)", n)
	}
	return nil
}

// resolveColor converts a color name or number string to an int.
func resolveColor(value string) int {
	if n, ok := namedColors[value]; ok {
		return n
	}
	n, _ := strconv.Atoi(value)
	return n
}

// ApplySpec parses and applies a single spec to the theme.
func (t *Theme) ApplySpec(spec string) error {
	element, attr, value, err := ParseSpec(spec)
	if err != nil {
		return err
	}

	s := t.elementPtr(element)
	if s == nil {
		return fmt.Errorf("unknown element %q", element)
	}

	switch attr {
	case "fg":
		s.FG = resolveColor(value)
	case "bg":
		s.BG = resolveColor(value)
	case "style":
		applyStyle(s, value)
	}

	return nil
}

// ApplySpecs applies multiple specs to the theme, stopping on first error.
func (t *Theme) ApplySpecs(specs []string) error {
	for _, spec := range specs {
		if err := t.ApplySpec(spec); err != nil {
			return err
		}
	}
	return nil
}

// elementPtr returns a pointer to the Style field for the given element name.
func (t *Theme) elementPtr(name string) *Style {
	switch name {
	case "provider.claude":
		return &t.ProviderClaude
	case "provider.copilot":
		return &t.ProviderCopilot
	case "provider.codex":
		return &t.ProviderCodex
	case "role.user":
		return &t.RoleUser
	case "role.assistant":
		return &t.RoleAssistant
	case "timestamp":
		return &t.Timestamp
	case "marker":
		return &t.Marker
	case "separator":
		return &t.Separator
	case "match":
		return &t.Match
	default:
		return nil
	}
}

func applyStyle(s *Style, value string) {
	switch value {
	case "bold":
		s.Bold = 1
	case "nobold":
		s.Bold = -1
	case "dim":
		s.Dim = 1
	case "nodim":
		s.Dim = -1
	case "underline":
		s.Underline = 1
	case "nounderline":
		s.Underline = -1
	case "italic":
		s.Italic = 1
	case "noitalic":
		s.Italic = -1
	}
}
