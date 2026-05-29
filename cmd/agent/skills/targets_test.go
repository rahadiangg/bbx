package skills

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveTargetsDirOverrideWins(t *testing.T) {
	// --dir trumps everything, including a (would-be invalid) target/scope.
	got, err := resolveTargets("/custom/path", []string{"claude-code"}, "project")
	if err != nil {
		t.Fatalf("resolveTargets: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"/custom/path"}) {
		t.Fatalf("got %v, want [/custom/path]", got)
	}
}

func TestResolveTargetsDefaultIsGenericAgents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No --target, no --dir, default scope ⇒ ~/.agents/skills (backward compat).
	got, err := resolveTargets("", nil, "")
	if err != nil {
		t.Fatalf("resolveTargets: %v", err)
	}
	want := []string{filepath.Join(home, ".agents", "skills")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveTargetsKnownAgentsGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cases := map[string]string{
		"agents":         filepath.Join(home, ".agents", "skills"),
		"claude-code":    filepath.Join(home, ".claude", "skills"),
		"codex":          filepath.Join(home, ".codex", "skills"),
		"cursor":         filepath.Join(home, ".cursor", "skills"),
		"opencode":       filepath.Join(home, ".config", "opencode", "skills"),
		"cline":          filepath.Join(home, ".agents", "skills"),
		"github-copilot": filepath.Join(home, ".copilot", "skills"),
	}
	for name, want := range cases {
		got, err := resolveTargets("", []string{name}, "global")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(got, []string{want}) {
			t.Errorf("%s global: got %v, want [%s]", name, got, want)
		}
	}
}

func TestResolveTargetsProjectScope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := resolveTargets("", []string{"claude-code", "codex"}, "project")
	if err != nil {
		t.Fatalf("resolveTargets: %v", err)
	}
	want := []string{
		filepath.Join(".claude", "skills"),
		filepath.Join(".agents", "skills"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveTargetsDedupesSharedDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// agents and cline both resolve to ~/.agents/skills ⇒ collapse to one.
	got, err := resolveTargets("", []string{"agents", "cline"}, "global")
	if err != nil {
		t.Fatalf("resolveTargets: %v", err)
	}
	want := []string{filepath.Join(home, ".agents", "skills")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v (should dedupe)", got, want)
	}
}

func TestResolveTargetsMultipleDistinct(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := resolveTargets("", []string{"claude-code", "codex"}, "global")
	if err != nil {
		t.Fatalf("resolveTargets: %v", err)
	}
	want := []string{
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(home, ".codex", "skills"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveTargetsUnknownTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := resolveTargets("", []string{"no-such-agent"}, "global"); err == nil {
		t.Fatalf("expected error for unknown target")
	}
}

func TestResolveTargetsInvalidScope(t *testing.T) {
	if _, err := resolveTargets("", []string{"agents"}, "sideways"); err == nil {
		t.Fatalf("expected error for invalid scope")
	}
}
