// Package bbx is the library root of the bbx repository. It exposes the
// embedded agent-skill bundle so the `bbx agent skills` CLI subtree can
// install/list/uninstall skills offline (the binary carries them at
// compile time). The same `skills/` directory is also consumed by Claude
// Code when this repo is added as a plugin marketplace
// (see .claude-plugin/plugin.json) — no duplication.
//
// The bbx binary entrypoint lives at cmd/bbx/main.go; this file is library
// code at the module root.
package bbx

import (
	"embed"
	"io/fs"
)

// skillsFS holds the canonical bbx Agent Skill bundle. Files live at
// skills/<name>/SKILL.md in the source tree and are embedded at compile
// time. The `skills` pattern (without `/**`) recurses by default.
//
//go:embed skills
var skillsFS embed.FS

// SkillsFS returns the embedded skill bundle rooted at "skills" — callers
// see paths like "<name>/SKILL.md" without the leading "skills/".
func SkillsFS() fs.FS {
	sub, err := fs.Sub(skillsFS, "skills")
	if err != nil {
		// Unreachable at runtime: `skills` is always present at build time
		// thanks to //go:embed. A panic here means a build-time misconfig.
		panic(err)
	}
	return sub
}
