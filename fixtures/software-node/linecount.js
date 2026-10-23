// linecount prints the number of non-empty lines in the named text file.
// Usage: node linecount.js <file>

"use strict";

const fs = require("node:fs");

function countNonEmptyLines(content) {
  return content
    .split("\n")
    .slice(0, -1)
    .filter((line) => line.trim() !== "").length;
}

function main() {
  const path = process.argv[2];
  if (!path) {
    console.error("Usage: node linecount.js <file>");
    process.exit(1);
  }

  let content;
  try {
    content = fs.readFileSync(path, "utf8");
  } catch (error) {
    console.error(`linecount: cannot read ${path}: ${error.message}`);
    process.exit(1);
  }

  console.log(countNonEmptyLines(content));
}

main();
