package color

import (
	"testing"
)

func TestResolveEnabled_AutoTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if !ResolveEnabled("auto", true) {
		t.Error("auto + TTY should be true")
	}
}

func TestResolveEnabled_AutoNoTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if ResolveEnabled("auto", false) {
		t.Error("auto + no TTY should be false")
	}
}

func TestResolveEnabled_Always(t *testing.T) {
	if !ResolveEnabled("always", false) {
		t.Error("always should be true regardless of TTY")
	}
	if !ResolveEnabled("always", true) {
		t.Error("always should be true regardless of TTY")
	}
}

func TestResolveEnabled_Never(t *testing.T) {
	if ResolveEnabled("never", true) {
		t.Error("never should be false regardless of TTY")
	}
	if ResolveEnabled("never", false) {
		t.Error("never should be false regardless of TTY")
	}
}

func TestResolveEnabled_AutoNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ResolveEnabled("auto", true) {
		t.Error("auto + NO_COLOR=1 + TTY should be false")
	}
}

func TestResolveEnabled_AlwaysOverridesNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !ResolveEnabled("always", true) {
		t.Error("always should override NO_COLOR")
	}
}

func TestResolveEnabled_NoColorEmptyIgnored(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if !ResolveEnabled("auto", true) {
		t.Error("empty NO_COLOR should be ignored")
	}
}
