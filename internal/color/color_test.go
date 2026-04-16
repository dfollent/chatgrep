package color

import (
	"strings"
	"testing"
)

func TestDefaultTheme_Enabled(t *testing.T) {
	th := DefaultTheme()
	if !th.Enabled {
		t.Error("DefaultTheme should have Enabled=true")
	}
}

func TestDefaultTheme_ProviderDefaults(t *testing.T) {
	th := DefaultTheme()
	tests := []struct {
		name string
		want int
	}{
		{"claude", 6},  // cyan
		{"copilot", 5}, // magenta
		{"codex", 3},   // yellow
	}
	for _, tt := range tests {
		s := th.ProviderStyle(tt.name)
		if s.FG != tt.want {
			t.Errorf("ProviderStyle(%q).FG = %d, want %d", tt.name, s.FG, tt.want)
		}
	}
}

func TestDefaultTheme_RoleDefaults(t *testing.T) {
	th := DefaultTheme()
	s := th.RoleStyle("user")
	if s.FG != 2 { // green
		t.Errorf("RoleStyle(user).FG = %d, want 2", s.FG)
	}
	s = th.RoleStyle("assistant")
	if s.FG != 4 { // blue
		t.Errorf("RoleStyle(assistant).FG = %d, want 4", s.FG)
	}
}

func TestDefaultTheme_TimestampDim(t *testing.T) {
	th := DefaultTheme()
	if th.Timestamp.Dim != 1 {
		t.Errorf("Timestamp.Dim = %d, want 1", th.Timestamp.Dim)
	}
}

func TestDefaultTheme_MarkerDefaults(t *testing.T) {
	th := DefaultTheme()
	if th.Marker.FG != 3 { // yellow
		t.Errorf("Marker.FG = %d, want 3", th.Marker.FG)
	}
	if th.Marker.Bold != 1 {
		t.Errorf("Marker.Bold = %d, want 1", th.Marker.Bold)
	}
}

func TestDefaultTheme_SeparatorDim(t *testing.T) {
	th := DefaultTheme()
	if th.Separator.Dim != 1 {
		t.Errorf("Separator.Dim = %d, want 1", th.Separator.Dim)
	}
}

func TestStyle_Sequence_FGOnly(t *testing.T) {
	s := Style{FG: 1, BG: -1} // red fg
	got := s.Sequence()
	if got != "\033[31m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[31m")
	}
}

func TestStyle_Sequence_BGOnly(t *testing.T) {
	s := Style{FG: -1, BG: 4} // blue bg
	got := s.Sequence()
	if got != "\033[44m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[44m")
	}
}

func TestStyle_Sequence_FGAndBG(t *testing.T) {
	s := Style{FG: 1, BG: 4} // red on blue
	got := s.Sequence()
	if !strings.Contains(got, "\033[31m") {
		t.Errorf("expected fg code in %q", got)
	}
	if !strings.Contains(got, "\033[44m") {
		t.Errorf("expected bg code in %q", got)
	}
}

func TestStyle_Sequence_ExtendedColor(t *testing.T) {
	s := Style{FG: 208, BG: -1} // 256-color orange
	got := s.Sequence()
	if got != "\033[38;5;208m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[38;5;208m")
	}
}

func TestStyle_Sequence_ExtendedBG(t *testing.T) {
	s := Style{FG: -1, BG: 208}
	got := s.Sequence()
	if got != "\033[48;5;208m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[48;5;208m")
	}
}

func TestStyle_Sequence_BoldOnly(t *testing.T) {
	s := Style{FG: -1, BG: -1, Bold: 1}
	got := s.Sequence()
	if got != "\033[1m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[1m")
	}
}

func TestStyle_Sequence_DimOnly(t *testing.T) {
	s := Style{FG: -1, BG: -1, Dim: 1}
	got := s.Sequence()
	if got != "\033[2m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[2m")
	}
}

func TestStyle_Sequence_UnderlineOnly(t *testing.T) {
	s := Style{FG: -1, BG: -1, Underline: 1}
	got := s.Sequence()
	if got != "\033[4m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[4m")
	}
}

func TestStyle_Sequence_ItalicOnly(t *testing.T) {
	s := Style{FG: -1, BG: -1, Italic: 1}
	got := s.Sequence()
	if got != "\033[3m" {
		t.Errorf("Sequence() = %q, want %q", got, "\033[3m")
	}
}

func TestStyle_Sequence_Combined(t *testing.T) {
	s := Style{FG: 2, BG: -1, Bold: 1, Dim: 1} // green, bold, dim
	got := s.Sequence()
	if !strings.Contains(got, "\033[1m") {
		t.Errorf("expected bold in %q", got)
	}
	if !strings.Contains(got, "\033[2m") {
		t.Errorf("expected dim in %q", got)
	}
	if !strings.Contains(got, "\033[32m") {
		t.Errorf("expected green fg in %q", got)
	}
}

func TestStyle_Sequence_Empty(t *testing.T) {
	s := Style{FG: -1, BG: -1}
	got := s.Sequence()
	if got != "" {
		t.Errorf("Sequence() = %q, want empty", got)
	}
}

func TestTheme_Apply_Enabled(t *testing.T) {
	th := DefaultTheme()
	s := Style{FG: 1, BG: -1} // red
	got := th.Apply(s, "hello")
	if !strings.Contains(got, "\033[31m") {
		t.Errorf("expected ANSI code in %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("expected text in %q", got)
	}
	if !strings.HasSuffix(got, "\033[0m") {
		t.Errorf("expected reset suffix in %q", got)
	}
}

func TestTheme_Apply_Disabled(t *testing.T) {
	th := DefaultTheme()
	th.Enabled = false
	s := Style{FG: 1, BG: -1}
	got := th.Apply(s, "hello")
	if got != "hello" {
		t.Errorf("Apply with Enabled=false = %q, want %q", got, "hello")
	}
}

func TestTheme_Apply_EmptyStyle(t *testing.T) {
	th := DefaultTheme()
	s := Style{FG: -1, BG: -1}
	got := th.Apply(s, "hello")
	// No sequence to apply, should return plain text
	if got != "hello" {
		t.Errorf("Apply with empty style = %q, want %q", got, "hello")
	}
}

func TestTheme_Reset_Enabled(t *testing.T) {
	th := DefaultTheme()
	if th.Reset() != "\033[0m" {
		t.Errorf("Reset() = %q, want %q", th.Reset(), "\033[0m")
	}
}

func TestTheme_Reset_Disabled(t *testing.T) {
	th := DefaultTheme()
	th.Enabled = false
	if th.Reset() != "" {
		t.Errorf("Reset() = %q, want empty", th.Reset())
	}
}

func TestProviderStyle_Unknown(t *testing.T) {
	th := DefaultTheme()
	s := th.ProviderStyle("unknown")
	// Should return empty style
	if s.FG != -1 {
		t.Errorf("ProviderStyle(unknown).FG = %d, want -1", s.FG)
	}
}

func TestRoleStyle_Unknown(t *testing.T) {
	th := DefaultTheme()
	s := th.RoleStyle("system")
	if s.FG != -1 {
		t.Errorf("RoleStyle(system).FG = %d, want -1", s.FG)
	}
}
