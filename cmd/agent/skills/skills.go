// Package skills implements `bbx agent skills install|list|uninstall|update|show`.
// Skills are embedded in the bbx binary at compile time (see assets.go in the
// root bbx package) and extracted on demand into a per-user agents directory
// (default `~/.agents/skills/`, overrideable with --dir).
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
agents like Claude Code) embedded in the binary. These commands extract
them into a per-user directory so any agent runtime that scans that
location picks them up.

Default install dir is ~/.agents/skills/. Override with --dir.

Three install paths for end users:
  1. /plugin marketplace add rahadiangg/bbx  (Claude Code native)
  2. bbx agent skills install --all          (after curl|sh install)
  3. git clone + cp -r skills/ ~/.agents/skills/  (manual)`,
	}
	c.AddCommand(newInstallCmd())
	c.AddCommand(newListCmd())
	c.AddCommand(newUninstallCmd())
	c.AddCommand(newUpdateCmd())
	c.AddCommand(newShowCmd())
	return c
}

// defaultInstallDir returns "~/.agents/skills" with $HOME expanded.
func defaultInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agents", "skills"), nil
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

// resolveDir resolves and (mkdir -p)'s the install directory.
func resolveDir(flagDir string) (string, error) {
	dir := flagDir
	if dir == "" {
		d, err := defaultInstallDir()
		if err != nil {
			return "", fail.Wrap(err, "no_home_dir", fail.ExitGeneric)
		}
		dir = d
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fail.New("mkdir_failed", fmt.Sprintf("cannot create %s: %v", dir, err), fail.ExitUsage)
	}
	return dir, nil
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
		all    bool
		dir    string
		force  bool
		dryRun bool
	)
	c := &cobra.Command{
		Use:   "install [skills...]",
		Short: "Extract the bundled skills into the install directory",
		Long: `Without arguments (or with --all), installs every bundled skill.
Pass specific skill names to install only those. Use 'bbx agent skills
list' to see what's available.

Refuses to overwrite a modified on-disk SKILL.md unless --force.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveDir(dir)
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
			for _, name := range selected {
				content, err := readEmbeddedSkill(name)
				if err != nil {
					return fail.Wrap(err, "embed_read_failed", fail.ExitGeneric)
				}
				dest := filepath.Join(dir, name, "SKILL.md")
				st, err := checkInstalled(dir, name, content)
				if err != nil {
					return fail.Wrap(err, "stat_failed", fail.ExitGeneric)
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
					return fail.Wrap(err, "mkdir_failed", fail.ExitGeneric)
				}
				if err := os.WriteFile(dest, content, 0o644); err != nil {
					return fail.New("write_failed", fmt.Sprintf("write %s: %v", dest, err), fail.ExitGeneric)
				}
				cmdctx.G().Stderr("installed %s -> %s", name, dest)
				written++
			}
			cmdctx.G().Stderr("done: %d installed, %d skipped", written, skipped)
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "install every bundled skill (default if no names given)")
	c.Flags().StringVar(&dir, "dir", "", "install directory (default ~/.agents/skills)")
	c.Flags().BoolVar(&force, "force", false, "overwrite locally-modified skills")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen; don't write")
	return c
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
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path,omitempty"`
}

type skillRowList []skillRow

func (l skillRowList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "NAME", "STATUS", "DESCRIPTION")
	for _, r := range l {
		desc := r.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		t.AppendRow([]any{r.Name, r.Status, desc})
	}
	t.Render()
	return nil
}

func newListCmd() *cobra.Command {
	var dir string
	c := &cobra.Command{
		Use:   "list",
		Short: "List bundled skills + their install status",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := resolveDir(dir)
			if err != nil {
				return err
			}
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			rows := make(skillRowList, 0, len(names))
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
				if st != statusNotInstalled {
					row.Path = filepath.Join(d, name, "SKILL.md")
				}
				rows = append(rows, row)
			}
			return cmdctx.G().Emit(rows)
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "install directory (default ~/.agents/skills)")
	return c
}

// uninstall --------------------------------------------------------------

func newUninstallCmd() *cobra.Command {
	var (
		all    bool
		dir    string
		yes    bool
		dryRun bool
	)
	c := &cobra.Command{
		Use:   "uninstall [skills...]",
		Short: "Remove bbx-managed skills from the install directory",
		Long: `Only removes skills that bbx manages (i.e. names match an embedded
skill). Non-bbx files in the install dir are never touched.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := resolveDir(dir)
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

			// Collect candidate dirs that actually exist on disk.
			var toRemove []string
			for _, name := range selected {
				p := filepath.Join(d, name)
				if _, err := os.Stat(p); err == nil {
					toRemove = append(toRemove, p)
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
	c.Flags().StringVar(&dir, "dir", "", "install directory (default ~/.agents/skills)")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen; don't remove")
	return c
}

// update -----------------------------------------------------------------

func newUpdateCmd() *cobra.Command {
	var (
		dir    string
		dryRun bool
	)
	c := &cobra.Command{
		Use:   "update",
		Short: "Refresh already-installed skills with the bundled content",
		Long: `Like 'install --force', but only touches skills that are already
installed in the target dir. Useful after upgrading the bbx binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := resolveDir(dir)
			if err != nil {
				return err
			}
			names, err := embeddedSkillNames()
			if err != nil {
				return err
			}
			var updated, skipped int
			for _, name := range names {
				content, err := readEmbeddedSkill(name)
				if err != nil {
					return fail.Wrap(err, "embed_read_failed", fail.ExitGeneric)
				}
				st, _ := checkInstalled(d, name, content)
				if st == statusNotInstalled {
					skipped++
					continue
				}
				if st == statusInstalled {
					// already up to date
					skipped++
					continue
				}
				dest := filepath.Join(d, name, "SKILL.md")
				if dryRun {
					cmdctx.G().Stderr("would update %s", dest)
					updated++
					continue
				}
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return fail.Wrap(err, "mkdir_failed", fail.ExitGeneric)
				}
				if err := os.WriteFile(dest, content, 0o644); err != nil {
					return fail.New("write_failed", fmt.Sprintf("write %s: %v", dest, err), fail.ExitGeneric)
				}
				cmdctx.G().Stderr("updated %s", dest)
				updated++
			}
			cmdctx.G().Stderr("done: %d updated, %d skipped", updated, skipped)
			return nil
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "install directory (default ~/.agents/skills)")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen; don't write")
	return c
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
