#!/usr/bin/env node
// bbx agent skills — npm installer.
//
// A dependency-free Node port of `bbx agent skills install`. It copies the
// bundled SKILL.md files (shipped in ./skills, synced from the repo root) into
// the directory an AI agent scans. The agent-target mapping mirrors
// cmd/agent/skills/targets.go so `npx @rahadiangg/bbx-skills --target codex`
// lands in the same place as the Go CLI.
//
//   npx @rahadiangg/bbx-skills --all
//   npx @rahadiangg/bbx-skills bbx-setup --target codex
//   npx @rahadiangg/bbx-skills --all -a claude-code -a cursor
//   npx @rahadiangg/bbx-skills --all --dir ~/.someagent/skills
//   npx @rahadiangg/bbx-skills --list

import { readFileSync, readdirSync, mkdirSync, writeFileSync, existsSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const SKILLS_DIR = fileURLToPath(new URL("./skills", import.meta.url));

// Agent → skills directory. Keep in lockstep with targets.go.
// global(home) and project (cwd-relative) mirror the Go registry.
const TARGETS = {
  agents: { global: (h) => join(h, ".agents", "skills"), project: join(".agents", "skills") },
  "claude-code": { global: (h) => join(h, ".claude", "skills"), project: join(".claude", "skills") },
  codex: { global: (h) => join(h, ".codex", "skills"), project: join(".agents", "skills") },
  cursor: { global: (h) => join(h, ".cursor", "skills"), project: join(".agents", "skills") },
  opencode: { global: (h) => join(h, ".config", "opencode", "skills"), project: join(".agents", "skills") },
  cline: { global: (h) => join(h, ".agents", "skills"), project: join(".agents", "skills") },
  "github-copilot": { global: (h) => join(h, ".copilot", "skills"), project: join(".agents", "skills") },
};
const DEFAULT_TARGET = "agents";

export function targetNames() {
  return Object.keys(TARGETS).sort();
}

// resolveTargets mirrors the Go resolveTargets: --dir wins; else map each
// target through the registry at the given scope; default ⇒ generic agents;
// duplicate dirs collapse.
export function resolveTargets({ dir = "", targets = [], scope = "global", home = homedir() } = {}) {
  if (dir) return [dir];
  if (scope !== "" && scope !== "global" && scope !== "project") {
    throw new Error(`invalid --scope "${scope}" (want global or project)`);
  }
  const project = scope === "project";
  const list = targets.length ? targets : [DEFAULT_TARGET];
  const seen = new Set();
  const dirs = [];
  for (const name of list) {
    const t = TARGETS[name];
    if (!t) {
      throw new Error(`unknown target: ${name} (valid: ${targetNames().join(", ")} — or use --dir)`);
    }
    const d = project ? t.project : t.global(home);
    if (seen.has(d)) continue;
    seen.add(d);
    dirs.push(d);
  }
  return dirs;
}

export function bundledSkillNames() {
  return readdirSync(SKILLS_DIR, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .sort();
}

function readBundledSkill(name) {
  return readFileSync(join(SKILLS_DIR, name, "SKILL.md"));
}

function parseArgs(argv) {
  const opts = { all: false, dir: "", targets: [], scope: "global", force: false, dryRun: false, list: false, help: false, names: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case "--all": opts.all = true; break;
      case "--force": opts.force = true; break;
      case "--dry-run": opts.dryRun = true; break;
      case "--list": opts.list = true; break;
      case "-h": case "--help": opts.help = true; break;
      case "--dir": opts.dir = argv[++i] ?? ""; break;
      case "--scope": opts.scope = argv[++i] ?? "global"; break;
      case "-a": case "--target": opts.targets.push(argv[++i] ?? ""); break;
      default:
        if (a.startsWith("-")) throw new Error(`unknown flag: ${a}`);
        opts.names.push(a);
    }
  }
  return opts;
}

function selectSkills(all, names) {
  const available = bundledSkillNames();
  if (names.length === 0 || all) return available;
  for (const n of names) {
    if (!available.includes(n)) throw new Error(`unknown skill: ${n} (try --list)`);
  }
  return names;
}

// installToDir copies selected skills into one dir; returns {written, skipped}.
export function installToDir(dir, selected, { force = false, dryRun = false } = {}) {
  let written = 0, skipped = 0;
  for (const name of selected) {
    const content = readBundledSkill(name);
    const dest = join(dir, name, "SKILL.md");
    if (existsSync(dest)) {
      const cur = readFileSync(dest);
      if (cur.equals(content)) { skipped++; continue; } // up to date
      if (!force) {
        console.error(`skip ${name}: locally modified (use --force to overwrite)`);
        skipped++;
        continue;
      }
    }
    if (dryRun) {
      console.error(`would write ${content.length} bytes to ${dest}`);
      written++;
      continue;
    }
    mkdirSync(dirname(dest), { recursive: true });
    writeFileSync(dest, content);
    console.error(`installed ${name} -> ${dest}`);
    written++;
  }
  return { written, skipped };
}

const HELP = `bbx agent skills (npm wrapper)

Usage: npx @rahadiangg/bbx-skills [skills...] [flags]

Flags:
  --all              install every bundled skill (default if no names given)
  -a, --target NAME  agent target(s): ${targetNames().join(", ")} (default: agents)
      --scope SCOPE  global (~/) or project (cwd)        [default: global]
      --dir PATH     explicit install directory (overrides --target/--scope)
      --force        overwrite locally-modified skills
      --dry-run      print what would happen; don't write
      --list         list bundled skills and exit
  -h, --help         show this help

Cross-agent alternative: npx skills add rahadiangg/bbx`;

export function main(argv = process.argv.slice(2)) {
  const opts = parseArgs(argv);
  if (opts.help) { console.log(HELP); return 0; }
  if (opts.list) { for (const n of bundledSkillNames()) console.log(n); return 0; }

  const dirs = resolveTargets(opts);
  const selected = selectSkills(opts.all, opts.names);
  let written = 0, skipped = 0;
  for (const d of dirs) {
    if (!opts.dryRun) mkdirSync(d, { recursive: true });
    else if (!existsSync(d)) { /* dry-run: don't create */ } else statSync(d);
    const r = installToDir(d, selected, { force: opts.force, dryRun: opts.dryRun });
    written += r.written;
    skipped += r.skipped;
  }
  console.error(`done: ${written} installed, ${skipped} skipped`);
  return 0;
}

// Run only when invoked as a CLI (not when imported by tests).
if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    process.exit(main());
  } catch (err) {
    console.error(`error: ${err.message}`);
    process.exit(1);
  }
}
