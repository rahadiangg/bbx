package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bbx "github.com/rahadiangg/bbx"
	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/output"
)

// helper: dest path for one skill
func skillPath(dir, name string) string {
	return filepath.Join(dir, name, "SKILL.md")
}

// run executes the skills parent's child command (install/list/etc.) with
// the given args. Returns the cobra command after running so callers can
// inspect output via the writers they set.
func run(t *testing.T, args ...string) error {
	t.Helper()
	// Prime cmdctx: tests invoke the `skills` subtree directly without
	// going through cmd.New() / PersistentPreRunE, so Format would be
	// empty. Force JSON so output.Print works deterministically.
	cmdctx.Set(cmdctx.Globals{Format: output.FormatJSON})
	root := New()
	root.SetArgs(args)
	root.SetOut(os.Stderr)
	root.SetErr(os.Stderr)
	return root.Execute()
}

// embeddedNames returns the canonical embedded skill names (real values
// from assets.go, not hard-coded — so the test stays in sync as we add
// skills).
func embeddedNames(t *testing.T) []string {
	t.Helper()
	entries, err := fs.ReadDir(bbx.SkillsFS(), ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestSkillsInstallToTempDir(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "--all"); err != nil {
		t.Fatalf("install: %v", err)
	}
	for _, name := range embeddedNames(t) {
		if _, err := os.Stat(skillPath(dir, name)); err != nil {
			t.Errorf("%s missing on disk: %v", name, err)
		}
	}
}

func TestSkillsInstallDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "--all", "--dry-run"); err != nil {
		t.Fatalf("install --dry-run: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote files: %v", entries)
	}
}

func TestSkillsInstallSelective(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "bbx-setup"); err != nil {
		t.Fatalf("install bbx-setup: %v", err)
	}
	if _, err := os.Stat(skillPath(dir, "bbx-setup")); err != nil {
		t.Errorf("bbx-setup should exist: %v", err)
	}
	if _, err := os.Stat(skillPath(dir, "bbx")); !os.IsNotExist(err) {
		t.Errorf("bbx should NOT exist (selective install); got %v", err)
	}
}

func TestSkillsInstallUnknownSkill(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "no-such-skill"); err == nil {
		t.Fatalf("expected error for unknown skill")
	}
}

func TestSkillsInstallRefusesNonBBXModifiedOverwrite(t *testing.T) {
	dir := t.TempDir()
	// install once
	if err := run(t, "install", "--dir", dir, "bbx-setup"); err != nil {
		t.Fatal(err)
	}
	// modify the on-disk copy
	if err := os.WriteFile(skillPath(dir, "bbx-setup"), []byte("LOCAL EDIT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// install again WITHOUT --force: should leave the modified content alone
	if err := run(t, "install", "--dir", dir, "bbx-setup"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(skillPath(dir, "bbx-setup"))
	if !strings.Contains(string(data), "LOCAL EDIT") {
		t.Fatalf("install without --force overwrote local edit; got %q", data)
	}
}

func TestSkillsInstallForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "bbx-setup"); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(skillPath(dir, "bbx-setup"), []byte("LOCAL EDIT\n"), 0o644)
	// install --force should restore bundled content
	if err := run(t, "install", "--dir", dir, "bbx-setup", "--force"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(skillPath(dir, "bbx-setup"))
	if strings.Contains(string(data), "LOCAL EDIT") {
		t.Fatalf("--force did not overwrite the modified content")
	}
	if !strings.Contains(string(data), "---") {
		t.Fatalf("restored content missing frontmatter")
	}
}

func TestSkillsListNotInstalled(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "list", "--dir", dir); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestSkillsUninstall(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "--all"); err != nil {
		t.Fatal(err)
	}
	if err := run(t, "uninstall", "--dir", dir, "--all", "--yes"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	for _, name := range embeddedNames(t) {
		if _, err := os.Stat(skillPath(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should be removed; got %v", name, err)
		}
	}
}

func TestSkillsUninstallNeedsYesInNonInteractive(t *testing.T) {
	dir := t.TempDir()
	if err := run(t, "install", "--dir", dir, "--all"); err != nil {
		t.Fatal(err)
	}
	// Without --yes, in test (non-TTY stdin), should refuse cleanly.
	err := run(t, "uninstall", "--dir", dir, "--all")
	if err == nil {
		t.Fatalf("expected error without --yes in non-TTY context")
	}
}

func TestSkillsUninstallDoesNotTouchUnknown(t *testing.T) {
	dir := t.TempDir()
	// Plant a non-bbx skill the user wrote themselves.
	custom := filepath.Join(dir, "my-custom-skill")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, "SKILL.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Install + uninstall bbx skills.
	_ = run(t, "install", "--dir", dir, "--all")
	_ = run(t, "uninstall", "--dir", dir, "--all", "--yes")
	// The user's custom skill must still be there.
	if _, err := os.Stat(filepath.Join(custom, "SKILL.md")); err != nil {
		t.Fatalf("user's custom skill was touched: %v", err)
	}
}

func TestSkillsUpdateOnlyRefreshesInstalled(t *testing.T) {
	dir := t.TempDir()
	// Install just one skill.
	if err := run(t, "install", "--dir", dir, "bbx-setup"); err != nil {
		t.Fatal(err)
	}
	// Modify it.
	_ = os.WriteFile(skillPath(dir, "bbx-setup"), []byte("LOCAL\n"), 0o644)
	// Run update — should restore bbx-setup (it's installed-but-modified)
	// but NOT install the other skills.
	if err := run(t, "update", "--dir", dir); err != nil {
		t.Fatalf("update: %v", err)
	}
	data, _ := os.ReadFile(skillPath(dir, "bbx-setup"))
	if strings.Contains(string(data), "LOCAL") {
		t.Errorf("update did not refresh modified skill")
	}
	// Other skills should NOT have been installed.
	if _, err := os.Stat(skillPath(dir, "bbx")); !os.IsNotExist(err) {
		t.Errorf("update should not install new skills; bbx exists: %v", err)
	}
}

func TestSkillsShow(t *testing.T) {
	// Capture stdout via a pipe.
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	err := run(t, "show", "bbx-setup")
	_ = w.Close()

	if err != nil {
		t.Fatalf("show: %v", err)
	}
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	if n == 0 || !strings.Contains(string(buf[:n]), "---") {
		t.Fatalf("show output missing frontmatter; got %q", buf[:n])
	}
}

func TestSkillsShowUnknown(t *testing.T) {
	if err := run(t, "show", "no-such-skill"); err == nil {
		t.Fatalf("expected error for unknown skill")
	}
}
