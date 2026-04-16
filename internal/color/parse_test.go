package color

import (
	"testing"
)

func TestParseSpec_ValidFG(t *testing.T) {
	tests := []struct {
		spec    string
		element string
		attr    string
		value   string
	}{
		{"role.user:fg:red", "role.user", "fg", "red"},
		{"provider.claude:fg:cyan", "provider.claude", "fg", "cyan"},
		{"timestamp:fg:200", "timestamp", "fg", "200"},
		{"marker:fg:0", "marker", "fg", "0"},
	}
	for _, tt := range tests {
		elem, attr, val, err := ParseSpec(tt.spec)
		if err != nil {
			t.Errorf("ParseSpec(%q) error: %v", tt.spec, err)
			continue
		}
		if elem != tt.element || attr != tt.attr || val != tt.value {
			t.Errorf("ParseSpec(%q) = (%q,%q,%q), want (%q,%q,%q)",
				tt.spec, elem, attr, val, tt.element, tt.attr, tt.value)
		}
	}
}

func TestParseSpec_ValidBG(t *testing.T) {
	elem, attr, val, err := ParseSpec("role.assistant:bg:blue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elem != "role.assistant" || attr != "bg" || val != "blue" {
		t.Errorf("got (%q,%q,%q)", elem, attr, val)
	}
}

func TestParseSpec_ValidStyle(t *testing.T) {
	tests := []string{
		"role.user:style:bold",
		"timestamp:style:dim",
		"marker:style:underline",
		"separator:style:italic",
		"role.user:style:nobold",
		"role.user:style:nodim",
		"role.user:style:nounderline",
		"role.user:style:noitalic",
	}
	for _, spec := range tests {
		_, _, _, err := ParseSpec(spec)
		if err != nil {
			t.Errorf("ParseSpec(%q) error: %v", spec, err)
		}
	}
}

func TestParseSpec_InvalidElement(t *testing.T) {
	_, _, _, err := ParseSpec("invalid:fg:red")
	if err == nil {
		t.Error("expected error for invalid element")
	}
}

func TestParseSpec_InvalidAttribute(t *testing.T) {
	_, _, _, err := ParseSpec("role.user:color:red")
	if err == nil {
		t.Error("expected error for invalid attribute")
	}
}

func TestParseSpec_InvalidColorName(t *testing.T) {
	_, _, _, err := ParseSpec("role.user:fg:purple")
	if err == nil {
		t.Error("expected error for invalid color name")
	}
}

func TestParseSpec_ColorNumberOutOfRange(t *testing.T) {
	_, _, _, err := ParseSpec("role.user:fg:256")
	if err == nil {
		t.Error("expected error for color number > 255")
	}
}

func TestParseSpec_NegativeColorNumber(t *testing.T) {
	_, _, _, err := ParseSpec("role.user:fg:-1")
	if err == nil {
		t.Error("expected error for negative color number")
	}
}

func TestParseSpec_InvalidStyleValue(t *testing.T) {
	_, _, _, err := ParseSpec("role.user:style:blink")
	if err == nil {
		t.Error("expected error for invalid style value")
	}
}

func TestParseSpec_WrongFieldCount(t *testing.T) {
	_, _, _, err := ParseSpec("role.user:fg")
	if err == nil {
		t.Error("expected error for missing value")
	}
	_, _, _, err = ParseSpec("role.user")
	if err == nil {
		t.Error("expected error for single field")
	}
}

func TestParseSpec_AllElements(t *testing.T) {
	elements := []string{
		"provider.claude", "provider.copilot", "provider.codex",
		"role.user", "role.assistant",
		"timestamp", "marker", "separator", "match",
	}
	for _, elem := range elements {
		_, _, _, err := ParseSpec(elem + ":fg:red")
		if err != nil {
			t.Errorf("ParseSpec(%q:fg:red) error: %v", elem, err)
		}
	}
}

func TestParseSpec_AllNamedColors(t *testing.T) {
	colors := []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}
	for _, c := range colors {
		_, _, _, err := ParseSpec("role.user:fg:" + c)
		if err != nil {
			t.Errorf("ParseSpec(role.user:fg:%s) error: %v", c, err)
		}
	}
}

func TestApplySpec_ChangesFG(t *testing.T) {
	th := DefaultTheme()
	if err := th.ApplySpec("role.user:fg:red"); err != nil {
		t.Fatal(err)
	}
	if th.RoleUser.FG != 1 { // red
		t.Errorf("RoleUser.FG = %d, want 1", th.RoleUser.FG)
	}
}

func TestApplySpec_ChangesBG(t *testing.T) {
	th := DefaultTheme()
	if err := th.ApplySpec("role.user:bg:blue"); err != nil {
		t.Fatal(err)
	}
	if th.RoleUser.BG != 4 { // blue
		t.Errorf("RoleUser.BG = %d, want 4", th.RoleUser.BG)
	}
}

func TestApplySpec_ChangesStyleBold(t *testing.T) {
	th := DefaultTheme()
	if err := th.ApplySpec("role.user:style:bold"); err != nil {
		t.Fatal(err)
	}
	if th.RoleUser.Bold != 1 {
		t.Errorf("RoleUser.Bold = %d, want 1", th.RoleUser.Bold)
	}
	// FG should be unchanged
	if th.RoleUser.FG != 2 { // still green
		t.Errorf("RoleUser.FG = %d, want 2 (unchanged)", th.RoleUser.FG)
	}
}

func TestApplySpec_ChangesStyleNobold(t *testing.T) {
	th := DefaultTheme()
	th.Marker.Bold = 1
	if err := th.ApplySpec("marker:style:nobold"); err != nil {
		t.Fatal(err)
	}
	if th.Marker.Bold != -1 {
		t.Errorf("Marker.Bold = %d, want -1", th.Marker.Bold)
	}
}

func TestApplySpec_ExtendedColor(t *testing.T) {
	th := DefaultTheme()
	if err := th.ApplySpec("provider.claude:fg:208"); err != nil {
		t.Fatal(err)
	}
	if th.ProviderClaude.FG != 208 {
		t.Errorf("ProviderClaude.FG = %d, want 208", th.ProviderClaude.FG)
	}
}

func TestApplySpec_AllProviders(t *testing.T) {
	th := DefaultTheme()
	if err := th.ApplySpec("provider.claude:fg:red"); err != nil {
		t.Fatal(err)
	}
	if th.ProviderClaude.FG != 1 {
		t.Errorf("ProviderClaude.FG = %d, want 1", th.ProviderClaude.FG)
	}
	if err := th.ApplySpec("provider.copilot:fg:red"); err != nil {
		t.Fatal(err)
	}
	if th.ProviderCopilot.FG != 1 {
		t.Errorf("ProviderCopilot.FG = %d, want 1", th.ProviderCopilot.FG)
	}
	if err := th.ApplySpec("provider.codex:fg:red"); err != nil {
		t.Fatal(err)
	}
	if th.ProviderCodex.FG != 1 {
		t.Errorf("ProviderCodex.FG = %d, want 1", th.ProviderCodex.FG)
	}
}

func TestApplySpecs_Multiple(t *testing.T) {
	th := DefaultTheme()
	specs := []string{
		"role.user:fg:red",
		"role.assistant:fg:yellow",
		"timestamp:style:bold",
	}
	if err := th.ApplySpecs(specs); err != nil {
		t.Fatal(err)
	}
	if th.RoleUser.FG != 1 {
		t.Errorf("RoleUser.FG = %d, want 1", th.RoleUser.FG)
	}
	if th.RoleAssistant.FG != 3 {
		t.Errorf("RoleAssistant.FG = %d, want 3", th.RoleAssistant.FG)
	}
	if th.Timestamp.Bold != 1 {
		t.Errorf("Timestamp.Bold = %d, want 1", th.Timestamp.Bold)
	}
}

func TestApplySpecs_Empty(t *testing.T) {
	th := DefaultTheme()
	if err := th.ApplySpecs(nil); err != nil {
		t.Fatal(err)
	}
	if err := th.ApplySpecs([]string{}); err != nil {
		t.Fatal(err)
	}
}

func TestApplySpecs_ErrorStopsProcessing(t *testing.T) {
	th := DefaultTheme()
	specs := []string{
		"role.user:fg:red",
		"invalid:fg:red",
	}
	err := th.ApplySpecs(specs)
	if err == nil {
		t.Error("expected error from invalid spec")
	}
}
