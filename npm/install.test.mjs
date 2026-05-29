import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, existsSync, readFileSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { resolveTargets, installToDir, bundledSkillNames, targetNames, main } from "./install.mjs";

test("resolveTargets: --dir wins outright", () => {
  assert.deepEqual(resolveTargets({ dir: "/custom", targets: ["codex"], scope: "project" }), ["/custom"]);
});

test("resolveTargets: default is generic agents dir", () => {
  assert.deepEqual(resolveTargets({ home: "/home/u" }), [join("/home/u", ".agents", "skills")]);
});

test("resolveTargets: known agents map to their global dirs", () => {
  const h = "/home/u";
  assert.deepEqual(resolveTargets({ targets: ["claude-code"], home: h }), [join(h, ".claude", "skills")]);
  assert.deepEqual(resolveTargets({ targets: ["codex"], home: h }), [join(h, ".codex", "skills")]);
  assert.deepEqual(resolveTargets({ targets: ["opencode"], home: h }), [join(h, ".config", "opencode", "skills")]);
});

test("resolveTargets: project scope is cwd-relative", () => {
  assert.deepEqual(resolveTargets({ targets: ["claude-code"], scope: "project" }), [join(".claude", "skills")]);
});

test("resolveTargets: dedupes shared dir (agents + cline)", () => {
  const h = "/home/u";
  assert.deepEqual(resolveTargets({ targets: ["agents", "cline"], home: h }), [join(h, ".agents", "skills")]);
});

test("resolveTargets: distinct targets produce distinct dirs", () => {
  const h = "/home/u";
  assert.deepEqual(resolveTargets({ targets: ["claude-code", "codex"], home: h }), [
    join(h, ".claude", "skills"),
    join(h, ".codex", "skills"),
  ]);
});

test("resolveTargets: unknown target throws", () => {
  assert.throws(() => resolveTargets({ targets: ["nope"] }), /unknown target/);
});

test("resolveTargets: invalid scope throws", () => {
  assert.throws(() => resolveTargets({ targets: ["agents"], scope: "sideways" }), /invalid --scope/);
});

test("targetNames includes the core agents", () => {
  for (const n of ["agents", "claude-code", "codex", "cursor", "opencode"]) {
    assert.ok(targetNames().includes(n), `missing ${n}`);
  }
});

test("installToDir writes bundled skills and is idempotent", () => {
  const dir = mkdtempSync(join(tmpdir(), "bbxskills-"));
  const names = bundledSkillNames();
  assert.ok(names.length >= 1, "expected bundled skills");
  const first = installToDir(dir, names);
  assert.equal(first.written, names.length);
  for (const n of names) assert.ok(existsSync(join(dir, n, "SKILL.md")), `${n} missing`);
  // Second run: all up to date, nothing written.
  const second = installToDir(dir, names);
  assert.equal(second.written, 0);
  assert.equal(second.skipped, names.length);
});

test("installToDir respects --force on locally-modified skills", () => {
  const dir = mkdtempSync(join(tmpdir(), "bbxskills-"));
  const name = bundledSkillNames()[0];
  const dest = join(dir, name, "SKILL.md");
  mkdirSync(join(dir, name), { recursive: true });
  writeFileSync(dest, "LOCAL EDIT\n");
  // Without force: left alone.
  installToDir(dir, [name]);
  assert.match(readFileSync(dest, "utf8"), /LOCAL EDIT/);
  // With force: restored.
  installToDir(dir, [name], { force: true });
  assert.doesNotMatch(readFileSync(dest, "utf8"), /LOCAL EDIT/);
});

test("installToDir dry-run writes nothing", () => {
  const dir = mkdtempSync(join(tmpdir(), "bbxskills-"));
  const r = installToDir(dir, bundledSkillNames(), { dryRun: true });
  assert.ok(r.written > 0);
  for (const n of bundledSkillNames()) assert.ok(!existsSync(join(dir, n, "SKILL.md")));
});

test("main --list returns 0", () => {
  assert.equal(main(["--list"]), 0);
});
