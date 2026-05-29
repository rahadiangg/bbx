package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rahadiangg/bbx/internal/fail"
)

// agentTarget maps a known AI-agent runtime to the directory it scans for
// SKILL.md bundles. SKILL.md is an open standard shared across agents — only
// the install *location* differs per agent, so a single embedded bundle serves
// all of them. `globalDir` is the per-user location (default); `projectDir` is
// the repo-relative location used with --scope project.
//
// Paths follow the conventions established by the vercel-labs/skills CLI so
// `bbx agent skills install --target X` lands where `npx skills add` would.
type agentTarget struct {
	name       string
	globalDir  func(home string) string
	projectDir string
}

// defaultTargetName is the generic location bbx has always used. Keeping it the
// default preserves backward compatibility: `bbx agent skills install` with no
// --target/--dir still writes to ~/.agents/skills.
const defaultTargetName = "agents"

var agentTargets = []agentTarget{
	{"agents", func(h string) string { return filepath.Join(h, ".agents", "skills") }, filepath.Join(".agents", "skills")},
	{"claude-code", func(h string) string { return filepath.Join(h, ".claude", "skills") }, filepath.Join(".claude", "skills")},
	{"codex", func(h string) string { return filepath.Join(h, ".codex", "skills") }, filepath.Join(".agents", "skills")},
	{"cursor", func(h string) string { return filepath.Join(h, ".cursor", "skills") }, filepath.Join(".agents", "skills")},
	{"opencode", func(h string) string { return filepath.Join(h, ".config", "opencode", "skills") }, filepath.Join(".agents", "skills")},
	{"cline", func(h string) string { return filepath.Join(h, ".agents", "skills") }, filepath.Join(".agents", "skills")},
	{"github-copilot", func(h string) string { return filepath.Join(h, ".copilot", "skills") }, filepath.Join(".agents", "skills")},
}

// lookupTarget finds a target by name.
func lookupTarget(name string) (agentTarget, bool) {
	for _, t := range agentTargets {
		if t.name == name {
			return t, true
		}
	}
	return agentTarget{}, false
}

// targetNames returns the sorted list of known target names (for help/errors).
func targetNames() []string {
	names := make([]string, 0, len(agentTargets))
	for _, t := range agentTargets {
		names = append(names, t.name)
	}
	sort.Strings(names)
	return names
}

// resolveTargets turns the user's flags into the concrete install directories.
//
//   - --dir wins outright (returns exactly that one dir) — the explicit escape
//     hatch for niche agents not in the registry.
//   - otherwise each --target is mapped through the registry at the requested
//     scope ("global" default, or "project"). No --target ⇒ the generic default.
//
// Duplicate directories (e.g. agents+cline both → ~/.agents/skills) collapse to
// one. Directories are NOT created here; callers ensureDir each as needed.
func resolveTargets(flagDir string, targets []string, scope string) ([]string, error) {
	if flagDir != "" {
		return []string{flagDir}, nil
	}

	switch scope {
	case "", "global", "project":
	default:
		return nil, fail.New("usage", fmt.Sprintf("invalid --scope %q (want global or project)", scope), fail.ExitUsage)
	}
	project := scope == "project"

	if len(targets) == 0 {
		targets = []string{defaultTargetName}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fail.Wrap(err, "no_home_dir", fail.ExitGeneric)
	}

	seen := map[string]struct{}{}
	dirs := make([]string, 0, len(targets))
	for _, name := range targets {
		t, ok := lookupTarget(name)
		if !ok {
			return nil, fail.New("unknown_target",
				fmt.Sprintf("unknown target: %s (valid: %s — or use --dir for a custom path)", name, strings.Join(targetNames(), ", ")),
				fail.ExitUsage)
		}
		dir := t.globalDir(home)
		if project {
			dir = t.projectDir
		}
		if _, dup := seen[dir]; dup {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

// ensureDir mkdir -p's a resolved install directory.
func ensureDir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fail.New("mkdir_failed", fmt.Sprintf("cannot create %s: %v", dir, err), fail.ExitUsage)
	}
	return dir, nil
}
