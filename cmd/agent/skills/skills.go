// Package skills implements `bbx agent skills install|list|uninstall|update|show`.
// Skills are embedded in the bbx binary at compile time (see assets.go in the
// root bbx package) and extracted on demand into the directory an AI-agent
// runtime scans for SKILL.md bundles.
//
// The generic default is `~/.agents/skills/`. Use --target to install where a
// specific agent reads (e.g. claude-code → ~/.claude/skills, codex →
// ~/.codex/skills); see targets.go for the registry. --dir is the explicit
// escape hatch for niche agents not in the registry.
//
// Pattern mirrors Grafana's `gcx agent skills` for ecosystem consistency.
package skills

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	bbx "github.com/rahadiangg/bbx"
	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

// New returns the `bbx agent skills` parent command.
func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "skills",
		Short: "Install, list, and update the bundled bbx agent skills",
		Long: `bbx ships a curated set of agent skills (markdown contracts for AI
agents — SKILL.md is an open standard shared by Claude Code, Codex, Cursor,
OpenCode and more) embedded in the binary. These commands extract them into
the directory an agent scans.

Default install dir is ~/.agents/skills/ (generic). Use --target to install
where a specific agent reads (claude-code, codex, cursor, opencode, cline,
github-copilot); --dir for anything else.

Install paths for end users:
  1. /plugin marketplace add rahadiangg/bbx       (Claude Code native)
  2. npx skills add rahadiangg/bbx [-a <agent>]    (any agent, no bbx needed)
  3. bbx agent skills install --all [--target X]   (after curl|sh install)
  4. git clone + cp -r skills/ <agent skills dir>  (manual)`,
	}
	c.AddCommand(newInstallCmd())
	c.AddCommand(newListCmd())
	c.AddCommand(newUninstallCmd())
	c.AddCommand(newUpdateCmd())
	c.AddCommand(newShowCmd())
	return c
}

// addTargetFlags registers the install-location flags shared by every
// subcommand: --dir (explicit override), --target/-a (agent name, repeatable),
// and --scope (global|project). See targets.go for how they resolve.
func addTargetFlags(c *cobra.Command, dir *string, targets *[]string, scope *string) {
	c.Flags().StringVar(dir, "dir", "", "explicit install directory (overrides --target/--scope)")
	c.Flags().StringSliceVarP(targets, "target", "a", nil,
		"agent target(s): "+strings.Join(targetNames(), ", ")+" (default: agents → ~/.agents/skills)")
	c.Flags().StringVar(scope, "scope", "global", "install scope: global (~/) or project (cwd)")
}

// embeddedSkillNames returns the sorted list of skill names embedded in the
// binary. It's the source of truth for "which skills does bbx manage."
func embeddedSkillNames() ([]string, error) {
	entries, err := fs.ReadDir(bbx.SkillsFS(), ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded skills: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// readEmbeddedSkill returns the SKILL.md bytes for a given skill name.
func readEmbeddedSkill(name string) ([]byte, error) {
	return fs.ReadFile(bbx.SkillsFS(), filepath.ToSlash(filepath.Join(name, "SKILL.md")))
}

// skillDescription parses the YAML frontmatter `description:` line from a
// SKILL.md body. Best-effort; returns "" if the line isn't found.
func skillDescription(content []byte) string {
	// Look for a `description: ...` line within the first ~30 lines.
	scanner := bytes.NewReader(content)
	br := bufio{r: scanner}
	for i := 0; i < 40; i++ {
		line, ok := br.next()
		if !ok {
			break
		}
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}

// bufio is a tiny line reader so we don't pull in bufio everywhere.
type bufio struct {
	r   io.Reader
	buf []byte
	eof bool
}

func (b *bufio) next() (string, bool) {
	for !b.eof {
		idx := bytes.IndexByte(b.buf, '\n')
		if idx >= 0 {
			line := string(b.buf[:idx])
			b.buf = b.buf[idx+1:]
			return line, true
		}
		// refill
		chunk := make([]byte, 1024)
		n, err := b.r.Read(chunk)
		if n > 0 {
			b.buf = append(b.buf, chunk[:n]...)
		}
		if err != nil {
			b.eof = true
			if len(b.buf) > 0 {
				line := string(b.buf)
				b.buf = nil
				return line, true
			}
		}
	}
	return "", false
}

// fileStatus describes the install state of a skill on disk.
type fileStatus int

const (
	statusNotInstalled fileStatus = iota
	statusInstalled
	statusInstalledModified
)

func (s fileStatus) String() string {
	switch s {
	case statusInstalled:
		return "installed"
	case statusInstalledModified:
		return "installed (modified)"
	default:
		return "not installed"
	}
}

// checkInstalled returns the status of one skill at <dir>/<name>/SKILL.md.
func checkInstalled(dir, name string, embedded []byte) (fileStatus, error) {
	p := filepath.Join(dir, name, "SKILL.md")
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return statusNotInstalled, nil
		}
		return statusNotInstalled, err
	}
	if sha256.Sum256(data) == sha256.Sum256(embedded) {
		return statusInstalled, nil
	}
	return statusInstalledModified, nil
}

// install ----------------------------------------------------------------

func newInstallCmd() *cobra.Command {
	var (
		all     bool
		dir     string
		targets []string
		scope   string
		force   bool
		dryRun  bool
	)
	c := &cobra.Command{
		Use:   "install [skills...]",
		Short: "Extract the bundled skills into an agent's skills directory",
		Long: `Without arguments (or with --all), installs every bundled skill.
Pass specific skill names to install only those. Use 'bbx agent skills
list' to see what's available.

By default skills install to ~/.agents/skills (the generic location). Use
--target to install where a specific agent reads, e.g.:

  bbx agent skills install --all --target claude-code   # ~/.claude/skills
  bbx agent skills install --all --target codex          # ~/.codex/skills
  bbx agent skills install --all -a cursor -a opencode   # both, in one run
  bbx agent skills install --all --target codex --scope project  # ./.agents/skills

For an agent not in the registry, point --dir at its skills directory:

  bbx agent skills install --all --dir ~/.someagent/skills

The cross-agent 'npx skills add rahadiangg/bbx' is an alternative that fans
out to many agents at once.

Refuses to overwrite a modified on-disk SKILL.md unless --force.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := resolveTargets(dir, targets, scope)
			if err != nil {
				return err
			}
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			selected, err := selectSkills(names, args, all)
			if err != nil {
				return err
			}

			var written, skipped int
			for _, d := range dirs {
				if _, err := ensureDir(d); err != nil {
					return err
				}
				w, s, err := installToDir(d, selected, force, dryRun)
				if err != nil {
					return err
				}
				written += w
				skipped += s
			}
			cmdctx.G().Stderr("done: %d installed, %d skipped", written, skipped)
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "install every bundled skill (default if no names given)")
	c.Flags().BoolVar(&force, "force", false, "overwrite locally-modified skills")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen; don't write")
	addTargetFlags(c, &dir, &targets, &scope)
	return c
}

// installToDir writes the selected skills into one resolved directory,
// honouring --force (overwrite locally-modified) and --dry-run. It returns the
// number written and skipped. The directory must already exist (ensureDir).
func installToDir(dir string, selected []string, force, dryRun bool) (written, skipped int, err error) {
	for _, name := range selected {
		content, err := readEmbeddedSkill(name)
		if err != nil {
			return written, skipped, fail.Wrap(err, "embed_read_failed", fail.ExitGeneric)
		}
		dest := filepath.Join(dir, name, "SKILL.md")
		st, err := checkInstalled(dir, name, content)
		if err != nil {
			return written, skipped, fail.Wrap(err, "stat_failed", fail.ExitGeneric)
		}
		if st == statusInstalledModified && !force {
			cmdctx.G().Stderr("skip %s: locally modified (use --force to overwrite, or 'bbx agent skills show %s' to inspect bundled)", name, name)
			skipped++
			continue
		}
		if st == statusInstalled {
			// Already up to date; no work needed.
			skipped++
			continue
		}
		if dryRun {
			cmdctx.G().Stderr("would write %d bytes to %s", len(content), dest)
			written++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return written, skipped, fail.Wrap(err, "mkdir_failed", fail.ExitGeneric)
		}
		if err := os.WriteFile(dest, content, 0o644); err != nil {
			return written, skipped, fail.New("write_failed", fmt.Sprintf("write %s: %v", dest, err), fail.ExitGeneric)
		}
		cmdctx.G().Stderr("installed %s -> %s", name, dest)
		written++
	}
	return written, skipped, nil
}

// selectSkills resolves the list of skills to act on: explicit args, or all
// embedded names if args is empty / --all.
func selectSkills(allNames, args []string, allFlag bool) ([]string, error) {
	if len(args) == 0 || allFlag {
		return allNames, nil
	}
	allowed := map[string]struct{}{}
	for _, n := range allNames {
		allowed[n] = struct{}{}
	}
	var sel []string
	for _, a := range args {
		if _, ok := allowed[a]; !ok {
			return nil, fail.New("unknown_skill",
				fmt.Sprintf("unknown skill: %s (try 'bbx agent skills list')", a),
				fail.ExitUsage)
		}
		sel = append(sel, a)
	}
	return sel, nil
}

// list -------------------------------------------------------------------

type skillRow struct {
	Name        string `json:"name"`
	Dir         string `json:"dir,omitempty"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path,omitempty"`
}

type skillRowList []skillRow

func (l skillRowList) RenderTable(w io.Writer) error {
	// Only show the DIR column when more than one install dir is in play —
	// keeps single-target (the common case) output identical to before.
	multiDir := false
	for _, r := range l {
		if r.Dir != "" {
			multiDir = true
			break
		}
	}
	t := output.NewTable(w, "NAME", "STATUS", "DESCRIPTION")
	if multiDir {
		t = output.NewTable(w, "NAME", "DIR", "STATUS", "DESCRIPTION")
	}
	for _, r := range l {
		desc := r.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		if multiDir {
			t.AppendRow([]any{r.Name, r.Dir, r.Status, desc})
		} else {
			t.AppendRow([]any{r.Name, r.Status, desc})
		}
	}
	t.Render()
	return nil
}

func newListCmd() *cobra.Command {
	var (
		dir     string
		targets []string
		scope   string
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List bundled skills + their install status",
		Long: `Lists the bundled skills and whether each is installed in the
target directory. With multiple --target values, a DIR column disambiguates
which agent location each row describes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := resolveTargets(dir, targets, scope)
			if err != nil {
				return err
			}
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			multiDir := len(dirs) > 1
			rows := make(skillRowList, 0, len(names)*len(dirs))
			for _, d := range dirs {
				for _, name := range names {
					content, err := readEmbeddedSkill(name)
					if err != nil {
						return fail.Wrap(err, "embed_read_failed", fail.ExitGeneric)
					}
					st, _ := checkInstalled(d, name, content)
					row := skillRow{
						Name:        name,
						Status:      st.String(),
						Description: skillDescription(content),
					}
					if multiDir {
						row.Dir = d
					}
					if st != statusNotInstalled {
						row.Path = filepath.Join(d, name, "SKILL.md")
					}
					rows = append(rows, row)
				}
			}
			return cmdctx.G().Emit(rows)
		},
	}
	addTargetFlags(c, &dir, &targets, &scope)
	return c
}

// uninstall --------------------------------------------------------------

func newUninstallCmd() *cobra.Command {
	var (
		all     bool
		dir     string
		targets []string
		scope   string
		yes     bool
		dryRun  bool
	)
	c := &cobra.Command{
		Use:   "uninstall [skills...]",
		Short: "Remove bbx-managed skills from an agent's skills directory",
		Long: `Only removes skills that bbx manages (i.e. names match an embedded
skill). Non-bbx files in the install dir are never touched. Use --target /
--dir / --scope to select the directory, exactly like 'install'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := resolveTargets(dir, targets, scope)
			if err != nil {
				return err
			}
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			selected, err := selectSkills(names, args, all)
			if err != nil {
				return err
			}

			// Collect candidate dirs that actually exist on disk, across every
			// resolved target directory.
			var toRemove []string
			for _, d := range dirs {
				for _, name := range selected {
					p := filepath.Join(d, name)
					if _, err := os.Stat(p); err == nil {
						toRemove = append(toRemove, p)
					}
				}
			}
			if len(toRemove) == 0 {
				cmdctx.G().Stderr("nothing to uninstall")
				return nil
			}

			if !yes && !dryRun {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fail.New("cancelled",
						"refusing to uninstall in non-interactive mode without --yes",
						fail.ExitCancelled)
				}
				cmdctx.G().Stderr("about to remove:")
				for _, p := range toRemove {
					cmdctx.G().Stderr("  %s", p)
				}
				cmdctx.G().Stderr("proceed? (y/N): ")
				var resp string
				_, _ = fmt.Fscanln(os.Stdin, &resp)
				if !strings.EqualFold(strings.TrimSpace(resp), "y") &&
					!strings.EqualFold(strings.TrimSpace(resp), "yes") {
					return fail.New("cancelled", "user declined", fail.ExitCancelled)
				}
			}

			for _, p := range toRemove {
				if dryRun {
					cmdctx.G().Stderr("would remove %s", p)
					continue
				}
				if err := os.RemoveAll(p); err != nil {
					return fail.New("remove_failed", fmt.Sprintf("remove %s: %v", p, err), fail.ExitGeneric)
				}
				cmdctx.G().Stderr("removed %s", p)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "uninstall every bundled skill (default if no names given)")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen; don't remove")
	addTargetFlags(c, &dir, &targets, &scope)
	return c
}

// update -----------------------------------------------------------------

func newUpdateCmd() *cobra.Command {
	var (
		dir     string
		targets []string
		scope   string
		dryRun  bool
	)
	c := &cobra.Command{
		Use:   "update",
		Short: "Refresh already-installed skills with the bundled content",
		Long: `Like 'install --force', but only touches skills that are already
installed in the target dir(s). Useful after upgrading the bbx binary. Use
--target / --dir / --scope to select the directory, exactly like 'install'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := resolveTargets(dir, targets, scope)
			if err != nil {
				return err
			}
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			var updated, skipped int
			for _, d := range dirs {
				u, s, err := updateDir(d, names, dryRun)
				if err != nil {
					return err
				}
				updated += u
				skipped += s
			}
			cmdctx.G().Stderr("done: %d updated, %d skipped", updated, skipped)
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen; don't write")
	addTargetFlags(c, &dir, &targets, &scope)
	return c
}

// updateDir refreshes only the already-installed skills in one directory.
func updateDir(dir string, names []string, dryRun bool) (updated, skipped int, err error) {
	for _, name := range names {
		content, err := readEmbeddedSkill(name)
		if err != nil {
			return updated, skipped, fail.Wrap(err, "embed_read_failed", fail.ExitGeneric)
		}
		st, _ := checkInstalled(dir, name, content)
		if st == statusNotInstalled || st == statusInstalled {
			// not present, or already up to date
			skipped++
			continue
		}
		dest := filepath.Join(dir, name, "SKILL.md")
		if dryRun {
			cmdctx.G().Stderr("would update %s", dest)
			updated++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return updated, skipped, fail.Wrap(err, "mkdir_failed", fail.ExitGeneric)
		}
		if err := os.WriteFile(dest, content, 0o644); err != nil {
			return updated, skipped, fail.New("write_failed", fmt.Sprintf("write %s: %v", dest, err), fail.ExitGeneric)
		}
		cmdctx.G().Stderr("updated %s", dest)
		updated++
	}
	return updated, skipped, nil
}

// show -------------------------------------------------------------------

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <skill>",
		Short: "Print one bundled SKILL.md to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			if _, err := selectSkills(names, args, false); err != nil {
				return err
			}
			content, err := readEmbeddedSkill(args[0])
			if err != nil {
				return fail.Wrap(err, "embed_read_failed", fail.ExitGeneric)
			}
			_, _ = os.Stdout.Write(content)
			return nil
		},
	}
}
