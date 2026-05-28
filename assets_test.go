package bbx

import (
	"io/fs"
	"strings"
	"testing"
)

// TestSkillsFSContainsAllExpectedSkills is the safety net for the //go:embed
// directive: if a skill directory is added but the embed pattern misses it,
// this test fails loudly. If a skill is intentionally removed, update the
// expected list here.
func TestSkillsFSContainsAllExpectedSkills(t *testing.T) {
	expected := []string{
		"bbx",
		"bbx-extract-config",
		"bbx-investigate-build",
		"bbx-setup",
		"bbx-trigger-build",
	}
	got := map[string]bool{}
	entries, err := fs.ReadDir(SkillsFS(), ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			got[e.Name()] = true
		}
	}
	for _, name := range expected {
		if !got[name] {
			t.Errorf("missing skill %q from embedded FS", name)
		}
		// Every skill must have a non-empty SKILL.md.
		data, err := fs.ReadFile(SkillsFS(), name+"/SKILL.md")
		if err != nil {
			t.Errorf("%s/SKILL.md: %v", name, err)
			continue
		}
		if len(data) < 200 {
			t.Errorf("%s/SKILL.md suspiciously short: %d bytes", name, len(data))
		}
		// Frontmatter sanity: starts with `---`.
		if !strings.HasPrefix(strings.TrimSpace(string(data)), "---") {
			t.Errorf("%s/SKILL.md missing YAML frontmatter", name)
		}
	}
}
