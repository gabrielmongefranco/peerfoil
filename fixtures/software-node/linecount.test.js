// Tests for linecount.js, run with: node --test

"use strict";

const assert = require("node:assert");
const { execFileSync, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const SCRIPT = path.join(__dirname, "linecount.js");

// Writes content to a fresh temporary file and returns its path.
function writeTempFile(content) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "linecount-"));
  const file = path.join(dir, "input.txt");
  fs.writeFileSync(file, content, "utf8");
  return file;
}

// Runs the command on one file and returns its printed count as a number.
function runCount(file) {
  const output = execFileSync(process.execPath, [SCRIPT, file], { encoding: "utf8" });
  return Number(output.trim());
}

test("counts the lines of a plain file", () => {
  const file = writeTempFile("first\nsecond\nthird\n");
  assert.strictEqual(runCount(file), 3);
});

test("skips empty and whitespace-only lines", () => {
  const file = writeTempFile("first\n\n   \nsecond\n");
  assert.strictEqual(runCount(file), 2);
});

test("reports zero for an empty file", () => {
  const file = writeTempFile("");
  assert.strictEqual(runCount(file), 0);
});

test("fails with a readable error for a missing file", () => {
  const missing = path.join(os.tmpdir(), "linecount-missing", "no-such-file.txt");
  const result = spawnSync(process.execPath, [SCRIPT, missing], { encoding: "utf8" });
  assert.strictEqual(result.status, 1);
  assert.match(result.stderr, /cannot read/);
});

test("fails with usage help when no path is given", () => {
  const result = spawnSync(process.execPath, [SCRIPT], { encoding: "utf8" });
  assert.strictEqual(result.status, 1);
  assert.match(result.stderr, /Usage/);
});
